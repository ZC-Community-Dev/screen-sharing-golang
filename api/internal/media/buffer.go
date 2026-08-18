package media

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrBufferClosed = errors.New("media buffer closed")
	ErrSlowConsumer = errors.New("media slow consumer")
	ErrNoBootstrap  = errors.New("media bootstrap unavailable")
)

type Cluster struct {
	Data         []byte
	Timestamp    int64
	RandomAccess bool
}

type Bootstrap struct {
	Init       []byte
	Generation uint64
	Clusters   []Cluster
}

type ViewerQueue struct {
	C <-chan Cluster

	mu      sync.Mutex
	ch      chan Cluster
	closed  bool
	err     error
	pending []Cluster
}

func (q *ViewerQueue) Closed() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.closed
}

func (q *ViewerQueue) Err() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.err
}

func (q *ViewerQueue) close(err error) {
	q.mu.Lock()
	if !q.closed {
		q.closed = true
		q.err = err
		q.pending = nil
		close(q.ch)
	}
	q.mu.Unlock()
}

func (q *ViewerQueue) enqueue(cluster Cluster, maxBytes int, maxDuration time.Duration) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return false
	}
	queuedBytes := len(cluster.Data)
	for _, item := range q.pending {
		queuedBytes += len(item.Data)
	}
	overDuration := len(q.pending) > 0 &&
		time.Duration(cluster.Timestamp-q.pending[0].Timestamp)*time.Millisecond > maxDuration
	if queuedBytes > maxBytes || overDuration {
		return false
	}
	select {
	case q.ch <- cluster:
		q.pending = append(q.pending, cluster)
		return true
	default:
		return false
	}
}

func (q *ViewerQueue) Consumed(cluster Cluster) {
	q.mu.Lock()
	if len(q.pending) > 0 {
		q.pending[0] = Cluster{}
		q.pending = q.pending[1:]
	}
	q.mu.Unlock()
}

type ClusterRing struct {
	mu          sync.Mutex
	maxDuration time.Duration
	maxBytes    int
	queueSize   int
	init        []byte
	generation  uint64
	clusters    []Cluster
	bytes       int
	viewers     map[string]*ViewerQueue
	closed      bool
}

func NewClusterRing(maxDuration time.Duration, maxBytes, queueSize int) *ClusterRing {
	if maxDuration <= 0 || maxDuration > 2*time.Second {
		maxDuration = 2 * time.Second
	}
	if maxBytes <= 0 {
		maxBytes = 8 << 20
	}
	if queueSize <= 0 {
		queueSize = 16
	}
	return &ClusterRing{
		maxDuration: maxDuration,
		maxBytes:    maxBytes,
		queueSize:   queueSize,
		viewers:     make(map[string]*ViewerQueue),
	}
}

func (r *ClusterRing) SetInit(init []byte, generation uint64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || len(init) == 0 || len(init) > r.maxBytes {
		return false
	}
	r.releaseLocked()
	r.init = cloneBytes(init)
	r.generation = generation
	return true
}

func (r *ClusterRing) Reset(init []byte, generation uint64) {
	r.mu.Lock()
	old := make([]*ViewerQueue, 0, len(r.viewers))
	for _, viewer := range r.viewers {
		old = append(old, viewer)
	}
	r.viewers = make(map[string]*ViewerQueue)
	r.releaseLocked()
	r.init = cloneBytes(init)
	r.generation = generation
	r.mu.Unlock()
	for _, viewer := range old {
		viewer.close(ErrBufferClosed)
	}
}

func (r *ClusterRing) Append(cluster Cluster) bool {
	cluster.Data = cloneBytes(cluster.Data)
	r.mu.Lock()
	if r.closed || len(cluster.Data)+len(r.init) > r.maxBytes {
		r.mu.Unlock()
		zero(cluster.Data)
		return false
	}
	r.clusters = append(r.clusters, cluster)
	r.bytes += len(cluster.Data)
	r.trimLocked()
	var slow []*ViewerQueue
	for id, viewer := range r.viewers {
		if !viewer.enqueue(cluster, r.maxBytes, r.maxDuration) {
			delete(r.viewers, id)
			slow = append(slow, viewer)
		}
	}
	r.mu.Unlock()
	for _, viewer := range slow {
		viewer.close(ErrSlowConsumer)
	}
	return true
}

// Subscribe atomically snapshots bootstrap data and registers the live queue
// under the same lock, preventing a duplicate or gap at handoff.
func (r *ClusterRing) Subscribe(id string) (Bootstrap, *ViewerQueue, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return Bootstrap{}, nil, ErrBufferClosed
	}
	if len(r.init) == 0 || len(r.clusters) == 0 {
		return Bootstrap{}, nil, ErrNoBootstrap
	}
	start := -1
	for i := len(r.clusters) - 1; i >= 0; i-- {
		if r.clusters[i].RandomAccess {
			start = i
			break
		}
	}
	if start < 0 {
		return Bootstrap{}, nil, ErrNoBootstrap
	}
	ch := make(chan Cluster, r.queueSize)
	viewer := &ViewerQueue{ch: ch, C: ch}
	if previous := r.viewers[id]; previous != nil {
		previous.close(ErrBufferClosed)
	}
	r.viewers[id] = viewer
	snapshot := Bootstrap{Init: cloneBytes(r.init), Generation: r.generation}
	for _, cluster := range r.clusters[start:] {
		snapshot.Clusters = append(snapshot.Clusters, cloneCluster(cluster))
	}
	return snapshot, viewer, nil
}

func (r *ClusterRing) Unsubscribe(id string) {
	r.mu.Lock()
	viewer := r.viewers[id]
	delete(r.viewers, id)
	r.mu.Unlock()
	if viewer != nil {
		viewer.close(nil)
	}
}

func (r *ClusterRing) Close() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	viewers := make([]*ViewerQueue, 0, len(r.viewers))
	for _, viewer := range r.viewers {
		viewers = append(viewers, viewer)
	}
	clear(r.viewers)
	r.releaseLocked()
	r.mu.Unlock()
	for _, viewer := range viewers {
		viewer.close(ErrBufferClosed)
	}
}

func (r *ClusterRing) Bytes() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.bytes + len(r.init)
}

func (r *ClusterRing) trimLocked() {
	for len(r.clusters) > 0 {
		overBytes := r.bytes+len(r.init) > r.maxBytes
		overTime := len(r.clusters) > 1 &&
			time.Duration(r.clusters[len(r.clusters)-1].Timestamp-r.clusters[0].Timestamp)*time.Millisecond > r.maxDuration
		if !overBytes && !overTime {
			break
		}
		r.bytes -= len(r.clusters[0].Data)
		r.clusters = r.clusters[1:]
	}
}

// releaseLocked drops all server-owned references. Mutating shared cluster
// bytes here would race with an in-flight viewer write.
func (r *ClusterRing) releaseLocked() {
	r.init = nil
	r.clusters = nil
	r.bytes = 0
}

func cloneCluster(cluster Cluster) Cluster {
	cluster.Data = cloneBytes(cluster.Data)
	return cluster
}

func cloneBytes(data []byte) []byte {
	return append([]byte(nil), data...)
}

func zero(data []byte) {
	for i := range data {
		data[i] = 0
	}
}
