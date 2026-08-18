package media

import (
	"bytes"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestPublicationReservationIsCrossTransportAndImmutable(t *testing.T) {
	m := NewManager(nil, Limits{MaxRooms: 2, MaxViewersPerRoom: 10})
	id, generation, err := m.ReservePublication("link", "owner", TransportWebSocket)
	if err != nil {
		t.Fatal(err)
	}
	if id == "" || generation != 1 {
		t.Fatalf("reservation = %q generation %d", id, generation)
	}
	if _, _, err := m.ReservePublication("link", "owner", TransportWebRTC); !errors.Is(err, ErrPublisherConflict) {
		t.Fatalf("cross-transport reservation: %v", err)
	}
	pub, ok := m.Publication("link")
	if !ok || pub.Transport != TransportWebSocket || pub.OwnerSessionID != "owner" {
		t.Fatalf("publication = %+v, %v", pub, ok)
	}
	m.ClosePublisher("link", id)
	m.ClosePublisher("link", id)
}

func TestTicketStoreHashesBindsExpiresAndConsumesOnce(t *testing.T) {
	now := time.Unix(1000, 0)
	store := NewTicketStore(30*time.Second, func() time.Time { return now })
	raw, ticket, err := store.Issue(TicketClaims{
		LinkID: "link", SessionID: "session", Role: RolePublisher,
		Transport: TransportWebSocket, Generation: 7,
	})
	if err != nil || raw == "" || ticket.ExpiresAt != now.Add(30*time.Second) {
		t.Fatalf("issue: %+v %v", ticket, err)
	}
	if bytes.Contains(store.DebugHashes(), []byte(raw)) {
		t.Fatal("raw ticket retained")
	}

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.Consume(raw, TicketBinding{
				LinkID: "link", SessionID: "session", Role: RolePublisher, Generation: 7,
			})
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful consumes = %d", successes)
	}

	expired, _, err := store.Issue(TicketClaims{LinkID: "link", SessionID: "viewer", Role: RoleViewer, Transport: TransportWebSocket})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(31 * time.Second)
	if _, err := store.Consume(expired, TicketBinding{LinkID: "link", SessionID: "viewer", Role: RoleViewer}); !errors.Is(err, ErrTicketExpired) {
		t.Fatalf("expired consume: %v", err)
	}
	if store.Len() != 0 {
		t.Fatalf("tickets retained: %d", store.Len())
	}
}

func TestClusterRingBoundsBootstrapAndSlowViewer(t *testing.T) {
	ring := NewClusterRing(2*time.Second, 12, 2)
	ring.SetInit([]byte("init"), 1)
	ring.Append(Cluster{Data: []byte("aaaa"), Timestamp: 0, RandomAccess: true})
	ring.Append(Cluster{Data: []byte("bbbb"), Timestamp: 1000, RandomAccess: false})
	ring.Append(Cluster{Data: []byte("cccc"), Timestamp: 2100, RandomAccess: true})

	snapshot, viewer, err := ring.Subscribe("viewer")
	if err != nil {
		t.Fatal(err)
	}
	if string(snapshot.Init) != "init" || snapshot.Generation != 1 || len(snapshot.Clusters) != 1 || string(snapshot.Clusters[0].Data) != "cccc" {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	ring.Append(Cluster{Data: []byte("dddd"), Timestamp: 2200})
	ring.Append(Cluster{Data: []byte("eeee"), Timestamp: 2300})
	ring.Append(Cluster{Data: []byte("ffff"), Timestamp: 2400})
	if !viewer.Closed() || !errors.Is(viewer.Err(), ErrSlowConsumer) {
		t.Fatal("slow viewer was not isolated")
	}
	ring.Close()
	if ring.Bytes() != 0 {
		t.Fatalf("bytes retained: %d", ring.Bytes())
	}
}

func TestIncrementalWebMParserEveryBoundary(t *testing.T) {
	fixture := deterministicWebMFixture()
	for split := 0; split <= len(fixture); split++ {
		parser := NewWebMParser(1 << 20)
		var records []WebMRecord
		for _, part := range [][]byte{fixture[:split], fixture[split:]} {
			out, err := parser.Push(part)
			if err != nil {
				t.Fatalf("split %d: %v", split, err)
			}
			records = append(records, out...)
		}
		if len(parser.Init()) == 0 || len(records) != 1 || records[0].Timestamp != 0 || !records[0].RandomAccess {
			t.Fatalf("split %d init=%d records=%+v", split, len(parser.Init()), records)
		}
	}
}

func TestClusterRingFansOutToTenAndIsolatesOne(t *testing.T) {
	ring := NewClusterRing(2*time.Second, 1<<20, 4)
	if !ring.SetInit([]byte("init"), 1) {
		t.Fatal("set init")
	}
	ring.Append(Cluster{Data: []byte("key"), Timestamp: 0, RandomAccess: true})
	viewers := make([]*ViewerQueue, 10)
	for i := range viewers {
		_, viewer, err := ring.Subscribe(string(rune('a' + i)))
		if err != nil {
			t.Fatal(err)
		}
		viewers[i] = viewer
	}
	ring.Unsubscribe("a")
	want := []byte("shared-cluster")
	if !ring.Append(Cluster{Data: want, Timestamp: 100}) {
		t.Fatal("append")
	}
	for i, viewer := range viewers[1:] {
		select {
		case cluster := <-viewer.C:
			if !bytes.Equal(cluster.Data, want) {
				t.Fatalf("viewer %d data=%q", i+1, cluster.Data)
			}
			viewer.Consumed(cluster)
		case <-time.After(time.Second):
			t.Fatalf("viewer %d blocked", i+1)
		}
	}
	if viewers[0].Err() != nil || !viewers[0].Closed() {
		t.Fatal("explicitly closed viewer lifecycle is not isolated")
	}
}

func TestClusterRingCloseDoesNotMutateInFlightPayload(t *testing.T) {
	ring := NewClusterRing(2*time.Second, 1<<20, 2)
	if !ring.SetInit([]byte("init"), 1) {
		t.Fatal("set init")
	}
	if !ring.Append(Cluster{Data: []byte("key"), Timestamp: 0, RandomAccess: true}) {
		t.Fatal("append key")
	}
	_, viewer, err := ring.Subscribe("viewer")
	if err != nil {
		t.Fatal(err)
	}
	if !ring.Append(Cluster{Data: []byte("in-flight"), Timestamp: 100}) {
		t.Fatal("append live")
	}
	cluster := <-viewer.C
	ring.Close()
	if got := string(cluster.Data); got != "in-flight" {
		t.Fatalf("in-flight payload mutated during close: %q", got)
	}
	if !viewer.Closed() {
		t.Fatal("viewer queue remained open")
	}
}

func deterministicWebMFixture() []byte {
	// EBML header; Segment; Info(TimecodeScale); Tracks(VP8 video);
	// Cluster(Timecode=0, SimpleBlock track=1 keyframe with a VP8 keyframe).
	return []byte{
		0x1a, 0x45, 0xdf, 0xa3, 0x80,
		0x18, 0x53, 0x80, 0x67, 0xaf,
		0x15, 0x49, 0xa9, 0x66, 0x87, 0x2a, 0xd7, 0xb1, 0x83, 0x0f, 0x42, 0x40,
		0x16, 0x54, 0xae, 0x6b, 0x8f,
		0xae, 0x8d, 0xd7, 0x81, 0x01, 0x83, 0x81, 0x01, 0x86, 0x85, 'V', '_', 'V', 'P', '8',
		0x1f, 0x43, 0xb6, 0x75, 0x8a,
		0xe7, 0x81, 0x00,
		0xa3, 0x85, 0x81, 0x00, 0x00, 0x80, 0x00,
	}
}
