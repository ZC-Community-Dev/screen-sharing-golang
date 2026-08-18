package media

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"

	"github.com/pion/webrtc/v4"
)

var (
	ErrPublisherConflict = errors.New("publisher already exists")
	ErrCapacity          = errors.New("media capacity reached")
	ErrNotReady          = errors.New("publisher media is not ready")
	ErrSessionNotFound   = errors.New("media session not found")
	ErrUnauthorized      = errors.New("media session is not owned by room session")
	ErrInvalidSDP        = errors.New("invalid SDP offer")
)

type State string

const (
	StateWaiting      State = "waiting"
	StateConnecting   State = "connecting"
	StateSharing      State = "sharing"
	StateReconnecting State = "reconnecting"
	StateFailed       State = "failed"
)

type Limits struct {
	MaxRooms          int
	MaxViewersPerRoom int
}

type Stats struct {
	ActiveRooms       int
	ActiveSubscribers int
}

type StateCallback func(linkID string, state State)
type MediaStateCallback func(linkID, roomSessionID, mediaSessionID, role, state string)
type PublicationStateCallback func(linkID string, publication Publication)

type Manager struct {
	mu            sync.Mutex
	engine        *Engine
	factory       func() (*Engine, error)
	limits        Limits
	rooms         map[string]*mediaRoom
	generations   map[string]uint64
	onState       StateCallback
	onMedia       MediaStateCallback
	onPublication PublicationStateCallback
}

type mediaRoom struct {
	state       State
	publisher   *PublisherSession
	relay       *Relay
	publication Publication
	wsBuffer    *ClusterRing
	subscribers map[string]*SubscriberSession
}

func NewManager(engine *Engine, limits Limits) *Manager {
	if limits.MaxRooms < 1 {
		limits.MaxRooms = 20
	}
	if limits.MaxViewersPerRoom < 10 {
		limits.MaxViewersPerRoom = 10
	}
	return &Manager{
		engine: engine, limits: limits, rooms: make(map[string]*mediaRoom),
		generations: make(map[string]uint64),
	}
}

func NewManagerWithFactory(factory func() (*Engine, error), limits Limits) *Manager {
	m := NewManager(nil, limits)
	m.factory = factory
	return m
}

func (m *Manager) SetCallbacks(state StateCallback, media MediaStateCallback) {
	m.mu.Lock()
	m.onState, m.onMedia = state, media
	m.mu.Unlock()
}

func (m *Manager) SetPublicationCallback(callback PublicationStateCallback) {
	m.mu.Lock()
	m.onPublication = callback
	m.mu.Unlock()
}

func (m *Manager) engineAPI() (*webrtc.API, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.engine == nil && m.factory != nil {
		engine, err := m.factory()
		if err != nil {
			return nil, err
		}
		m.engine = engine
	}
	if m.engine == nil {
		return nil, errors.New("media engine unavailable")
	}
	return m.engine.API, nil
}

func (m *Manager) reservePublisher(linkID, owner string) (string, error) {
	id, _, err := m.ReservePublication(linkID, owner, TransportWebRTC)
	return id, err
}

func (m *Manager) ReservePublication(linkID, owner string, transport Transport) (string, uint64, error) {
	if !transport.Valid() {
		return "", 0, ErrTransportInvalid
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	room := m.rooms[linkID]
	if room != nil && room.publisher != nil {
		return "", 0, ErrPublisherConflict
	}
	if room == nil {
		if m.activeRoomsLocked() >= m.limits.MaxRooms {
			return "", 0, ErrCapacity
		}
		room = &mediaRoom{state: StateWaiting, subscribers: make(map[string]*SubscriberSession)}
		m.rooms[linkID] = room
	}
	id := opaqueID()
	generation := m.generations[linkID] + 1
	if generation == 0 {
		generation = 1
	}
	m.generations[linkID] = generation
	room.publisher = &PublisherSession{ID: id, LinkID: linkID, OwnerSessionID: owner, Transport: transport}
	room.publication = Publication{
		ID: id, LinkID: linkID, OwnerSessionID: owner, Transport: transport,
		State: PublicationConnecting, Generation: generation,
	}
	room.state = StateConnecting
	return id, generation, nil
}

func (m *Manager) ReserveWebSocketSubscriber(linkID, owner string, generation uint64) (string, Publication, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := m.rooms[linkID]
	if room == nil || room.publisher == nil || room.wsBuffer == nil {
		return "", Publication{}, ErrNotReady
	}
	if room.publication.Transport != TransportWebSocket || room.publication.Generation != generation ||
		room.publication.State != PublicationLive {
		return "", Publication{}, ErrTransportMismatch
	}
	if len(room.subscribers) >= m.limits.MaxViewersPerRoom {
		return "", Publication{}, ErrCapacity
	}
	id := opaqueID()
	room.subscribers[id] = &SubscriberSession{
		ID: id, LinkID: linkID, OwnerSessionID: owner, Transport: TransportWebSocket,
	}
	return id, room.publication, nil
}

func (m *Manager) AttachWebSocketSubscriber(linkID, id, owner string, closeSocket func()) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := m.rooms[linkID]
	if room == nil || room.subscribers[id] == nil {
		return ErrSessionNotFound
	}
	sub := room.subscribers[id]
	if sub.OwnerSessionID != owner || sub.Transport != TransportWebSocket {
		return ErrUnauthorized
	}
	sub.closeSocket = closeSocket
	return nil
}

func (m *Manager) reserveSubscriber(linkID, owner string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := m.rooms[linkID]
	if room == nil {
		room = &mediaRoom{state: StateWaiting, subscribers: make(map[string]*SubscriberSession)}
		m.rooms[linkID] = room
	}
	if len(room.subscribers) >= m.limits.MaxViewersPerRoom {
		return "", ErrCapacity
	}
	id := opaqueID()
	room.subscribers[id] = &SubscriberSession{
		ID: id, LinkID: linkID, OwnerSessionID: owner, Transport: TransportWebRTC,
	}
	return id, nil
}

func (m *Manager) Publisher(linkID string) *PublisherSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	if room := m.rooms[linkID]; room != nil {
		return room.publisher
	}
	return nil
}

func (m *Manager) Publication(linkID string) (Publication, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := m.rooms[linkID]
	if room == nil || room.publisher == nil {
		return Publication{}, false
	}
	return room.publication, true
}

func (m *Manager) AttachWebSocketPublisher(linkID, id string, ring *ClusterRing, closeSocket func()) error {
	m.mu.Lock()
	room := m.rooms[linkID]
	if room == nil || room.publisher == nil || room.publisher.ID != id {
		m.mu.Unlock()
		return ErrSessionNotFound
	}
	if room.publication.Transport != TransportWebSocket {
		m.mu.Unlock()
		return ErrTransportMismatch
	}
	room.publisher.closeSocket = closeSocket
	room.wsBuffer = ring
	publication := room.publication
	stateCB, publicationCB, mediaCB := m.onState, m.onPublication, m.onMedia
	m.mu.Unlock()
	if publicationCB != nil {
		publicationCB(linkID, publication)
	}
	if stateCB != nil {
		stateCB(linkID, StateConnecting)
	}
	if mediaCB != nil {
		mediaCB(linkID, publication.OwnerSessionID, publication.ID, "presenter", "connecting")
	}
	return nil
}

func (m *Manager) WebSocketBuffer(linkID string, generation uint64) (*ClusterRing, Publication, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := m.rooms[linkID]
	if room == nil || room.publisher == nil || room.publication.Transport != TransportWebSocket {
		return nil, Publication{}, ErrNotReady
	}
	if generation != 0 && room.publication.Generation != generation {
		return nil, Publication{}, ErrTransportMismatch
	}
	if room.wsBuffer == nil {
		return nil, Publication{}, ErrNotReady
	}
	return room.wsBuffer, room.publication, nil
}

func (m *Manager) MarkWebSocketLive(linkID, id string) bool {
	m.mu.Lock()
	room := m.rooms[linkID]
	if room == nil || room.publisher == nil || room.publisher.ID != id ||
		room.publication.Transport != TransportWebSocket || room.wsBuffer == nil {
		m.mu.Unlock()
		return false
	}
	room.state = StateSharing
	room.publication.State = PublicationLive
	publication := room.publication
	stateCB, publicationCB, mediaCB := m.onState, m.onPublication, m.onMedia
	m.mu.Unlock()
	if publicationCB != nil {
		publicationCB(linkID, publication)
	}
	if stateCB != nil {
		stateCB(linkID, StateSharing)
	}
	if mediaCB != nil {
		mediaCB(linkID, publication.OwnerSessionID, publication.ID, "presenter", "connected")
	}
	return true
}

func (m *Manager) ResetWebSocketPublication(linkID, id string) (uint64, bool) {
	m.mu.Lock()
	room := m.rooms[linkID]
	if room == nil || room.publisher == nil || room.publisher.ID != id ||
		room.publication.Transport != TransportWebSocket || room.wsBuffer == nil {
		m.mu.Unlock()
		return 0, false
	}
	generation := m.generations[linkID] + 1
	m.generations[linkID] = generation
	room.publication.Generation = generation
	room.publication.State = PublicationConnecting
	room.state = StateConnecting
	buffer := room.wsBuffer
	publication := room.publication
	stateCB, publicationCB := m.onState, m.onPublication
	m.mu.Unlock()
	buffer.Reset(nil, generation)
	if publicationCB != nil {
		publicationCB(linkID, publication)
	}
	if stateCB != nil {
		stateCB(linkID, StateConnecting)
	}
	return generation, true
}

func (m *Manager) relayFor(linkID string) (*Relay, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := m.rooms[linkID]
	if room == nil || room.publisher == nil {
		return nil, ErrNotReady
	}
	if room.publication.Transport != TransportWebRTC {
		return nil, ErrTransportMismatch
	}
	if room.relay == nil {
		return nil, ErrNotReady
	}
	return room.relay, nil
}

func (m *Manager) installPublisher(linkID, id string, session *PublisherSession) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := m.rooms[linkID]
	if room == nil || room.publisher == nil || room.publisher.ID != id {
		return false
	}
	room.publisher = session
	return true
}

func (m *Manager) installSubscriber(linkID, id string, session *SubscriberSession) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := m.rooms[linkID]
	if room == nil || room.subscribers[id] == nil {
		return false
	}
	room.subscribers[id] = session
	return true
}

func (m *Manager) setRelay(linkID, publisherID string, relay *Relay) bool {
	m.mu.Lock()
	room := m.rooms[linkID]
	if room == nil || room.publisher == nil || room.publisher.ID != publisherID || room.relay != nil {
		m.mu.Unlock()
		return false
	}
	room.relay = relay
	room.state = StateSharing
	room.publication.State = PublicationLive
	publication := room.publication
	cb, publicationCB := m.onState, m.onPublication
	m.mu.Unlock()
	if publicationCB != nil {
		publicationCB(linkID, publication)
	}
	if cb != nil {
		cb(linkID, StateSharing)
	}
	return true
}

func (m *Manager) ClosePublisher(linkID, id string) {
	m.mu.Lock()
	room := m.rooms[linkID]
	if room == nil || room.publisher == nil || room.publisher.ID != id {
		m.mu.Unlock()
		return
	}
	publisher := room.publisher
	relay := room.relay
	buffer := room.wsBuffer
	publication := room.publication
	subs := make([]*SubscriberSession, 0, len(room.subscribers))
	for _, sub := range room.subscribers {
		subs = append(subs, sub)
	}
	delete(m.rooms, linkID)
	stateCB, mediaCB, publicationCB := m.onState, m.onMedia, m.onPublication
	m.mu.Unlock()

	publisher.closePeer()
	if buffer != nil {
		buffer.Close()
	}
	if relay != nil {
		relay.Close()
	}
	for _, sub := range subs {
		sub.closePeer()
		if mediaCB != nil {
			mediaCB(linkID, sub.OwnerSessionID, sub.ID, "viewer", "closed")
		}
	}
	if mediaCB != nil {
		mediaCB(linkID, publisher.OwnerSessionID, publisher.ID, "presenter", "closed")
	}
	if publicationCB != nil {
		publication.State = PublicationEnded
		publicationCB(linkID, publication)
	}
	if stateCB != nil {
		stateCB(linkID, StateWaiting)
	}
}

func (m *Manager) CloseSubscriber(linkID, id, owner string) error {
	m.mu.Lock()
	room := m.rooms[linkID]
	if room == nil {
		m.mu.Unlock()
		return nil
	}
	sub := room.subscribers[id]
	if sub == nil {
		m.mu.Unlock()
		return nil
	}
	if owner != "" && sub.OwnerSessionID != owner {
		m.mu.Unlock()
		return ErrUnauthorized
	}
	delete(room.subscribers, id)
	cb := m.onMedia
	m.mu.Unlock()
	sub.closePeer()
	if cb != nil {
		cb(linkID, sub.OwnerSessionID, sub.ID, "viewer", "closed")
	}
	return nil
}

func (m *Manager) CloseRoomSession(linkID, owner string) {
	m.mu.Lock()
	room := m.rooms[linkID]
	if room == nil {
		m.mu.Unlock()
		return
	}
	if room.publisher != nil && room.publisher.OwnerSessionID == owner {
		id := room.publisher.ID
		m.mu.Unlock()
		m.ClosePublisher(linkID, id)
		return
	}
	var ids []string
	for id, sub := range room.subscribers {
		if sub.OwnerSessionID == owner {
			ids = append(ids, id)
		}
	}
	m.mu.Unlock()
	for _, id := range ids {
		_ = m.CloseSubscriber(linkID, id, owner)
	}
}

func (m *Manager) Stats() Stats {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out Stats
	out.ActiveRooms = m.activeRoomsLocked()
	for _, room := range m.rooms {
		out.ActiveSubscribers += len(room.subscribers)
	}
	return out
}

func (m *Manager) activeRoomsLocked() int {
	n := 0
	for _, room := range m.rooms {
		if room.publisher != nil {
			n++
		}
	}
	return n
}

func (m *Manager) Close() error {
	m.mu.Lock()
	var rooms []struct{ link, id string }
	for link, room := range m.rooms {
		if room.publisher != nil {
			rooms = append(rooms, struct{ link, id string }{link, room.publisher.ID})
		}
	}
	engine := m.engine
	m.mu.Unlock()
	for _, room := range rooms {
		m.ClosePublisher(room.link, room.id)
	}
	if engine != nil {
		return engine.Close()
	}
	return nil
}

func (m *Manager) SubscriberCount(linkID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if room := m.rooms[linkID]; room != nil {
		return len(room.subscribers)
	}
	return 0
}

func (m *Manager) EmitConnected(linkID, owner, id, role string) {
	m.emitMedia(linkID, owner, id, role, "connected")
}

func (m *Manager) FailPublication(linkID, id string) {
	m.mu.Lock()
	room := m.rooms[linkID]
	if room == nil || room.publisher == nil || room.publisher.ID != id {
		m.mu.Unlock()
		return
	}
	owner := room.publisher.OwnerSessionID
	m.mu.Unlock()
	m.emitState(linkID, StateFailed)
	m.emitMedia(linkID, owner, id, "presenter", "failed")
}

func opaqueID() string {
	var raw [16]byte
	_, _ = rand.Read(raw[:])
	return hex.EncodeToString(raw[:])
}
