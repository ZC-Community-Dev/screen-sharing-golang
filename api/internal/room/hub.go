package room

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

const unattachedTTL = 30 * time.Second

var (
	ErrPresenterConflict = errors.New("presenter already active")
	ErrShareConflict     = errors.New("share already active")
	ErrUnauthorized      = errors.New("session is not presenter")
	ErrUnknownSession    = errors.New("unknown session")
)

type Role string

const (
	RolePresenter Role = "presenter"
	RoleViewer    Role = "viewer"
)

type Session struct {
	ID        string
	LinkID    string
	Role      Role
	send      chan []byte
	createdAt time.Time
}

type roomState struct {
	sessions    map[string]*Session
	presenter   string
	state       string
	publication *Publication
}

type Publication struct {
	ID         string `json:"id"`
	Transport  string `json:"transport"`
	State      string `json:"state"`
	Generation uint64 `json:"generation,omitempty"`
}

type Hub struct {
	mu    sync.Mutex
	rooms map[string]*roomState
}

func NewHub() *Hub {
	return &Hub{rooms: map[string]*roomState{}}
}

func (h *Hub) ClaimPresenter(linkID string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.expireLocked(unattachedTTL)
	r := h.ensure(linkID)
	if r.presenter != "" {
		return "", ErrPresenterConflict
	}
	id := newID()
	r.sessions[id] = &Session{ID: id, LinkID: linkID, Role: RolePresenter, createdAt: time.Now()}
	r.presenter = id
	return id, nil
}

func (h *Hub) JoinViewer(linkID string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.expireLocked(unattachedTTL)
	r := h.ensure(linkID)
	id := newID()
	r.sessions[id] = &Session{ID: id, LinkID: linkID, Role: RoleViewer, createdAt: time.Now()}
	return id
}

func (h *Hub) StartShare(linkID, sessionID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	r := h.room(linkID)
	if r == nil {
		return ErrUnknownSession
	}
	if r.presenter != sessionID {
		return ErrUnauthorized
	}
	if r.state == "sharing" {
		return ErrShareConflict
	}
	r.state = "sharing"
	h.broadcastStateLocked(linkID, "sharing")
	return nil
}

func (h *Hub) StopShare(linkID, sessionID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	r := h.room(linkID)
	if r == nil {
		return ErrUnknownSession
	}
	if r.presenter != sessionID {
		return ErrUnauthorized
	}
	r.state = "waiting"
	h.broadcastStateLocked(linkID, "waiting")
	return nil
}

func (h *Hub) Disconnect(linkID, sessionID string) (presenterLeft bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.expireLocked(unattachedTTL)
	return h.disconnectLocked(linkID, sessionID)
}

func (h *Hub) ExpireUnattached(maxAge time.Duration) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.expireLocked(maxAge)
}

func (h *Hub) PresenterID(linkID string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	r := h.room(linkID)
	if r == nil {
		return ""
	}
	return r.presenter
}

func (h *Hub) ViewerIDs(linkID string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	r := h.room(linkID)
	if r == nil {
		return nil
	}
	var ids []string
	for id, s := range r.sessions {
		if s.Role == RoleViewer && s.send != nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func (h *Hub) Count(linkID string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.expireLocked(unattachedTTL)
	r := h.room(linkID)
	if r == nil {
		return 0
	}
	return attachedCount(r)
}

func (h *Hub) Sharing(linkID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	r := h.room(linkID)
	return r != nil && r.state == "sharing"
}

func (h *Hub) HasSession(linkID, sessionID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	r := h.room(linkID)
	if r == nil {
		return false
	}
	_, ok := r.sessions[sessionID]
	return ok
}

func (h *Hub) SessionRole(linkID, sessionID string) (Role, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r := h.room(linkID)
	if r == nil {
		return "", false
	}
	s, ok := r.sessions[sessionID]
	if !ok {
		return "", false
	}
	return s.Role, true
}

func (h *Hub) Attach(linkID, sessionID string, send chan []byte) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.expireLocked(unattachedTTL)
	r := h.room(linkID)
	if r == nil {
		return false
	}
	s, ok := r.sessions[sessionID]
	if !ok {
		return false
	}
	s.send = send
	state := "waiting"
	if r.state != "" {
		state = r.state
	}
	h.sendLocked(s, mustJSON(map[string]any{
		"type":    "room.state",
		"payload": h.statePayloadLocked(r, state),
	}))
	h.broadcastLocked(linkID, mustJSON(map[string]any{
		"type":    "presence",
		"payload": map[string]any{"participantCount": attachedCount(r)},
	}))
	return true
}

func (h *Hub) BroadcastState(linkID, state string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r := h.room(linkID)
	if r == nil {
		return
	}
	r.state = state
	if state == "waiting" {
		r.publication = nil
	}
	h.broadcastStateLocked(linkID, state)
}

func (h *Hub) BroadcastPublication(linkID string, publication Publication) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r := h.room(linkID)
	if r == nil {
		return
	}
	copy := publication
	r.publication = &copy
	h.broadcastLocked(linkID, mustJSON(map[string]any{
		"type": "publication.state",
		"payload": map[string]any{
			"publicationId": publication.ID,
			"transport":     publication.Transport,
			"state":         publication.State,
			"generation":    publication.Generation,
		},
	}))
}

func (h *Hub) Publication(linkID string) *Publication {
	h.mu.Lock()
	defer h.mu.Unlock()
	r := h.room(linkID)
	if r == nil || r.publication == nil {
		return nil
	}
	copy := *r.publication
	return &copy
}

func (h *Hub) SendMediaState(linkID, sessionID, mediaSessionID, role, state string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r := h.room(linkID)
	if r == nil {
		return
	}
	target := r.sessions[sessionID]
	if target == nil {
		return
	}
	payload := map[string]any{
		"state": state, "role": role, "mediaSessionId": mediaSessionID,
	}
	if r.publication != nil {
		payload["transport"] = r.publication.Transport
		payload["publicationId"] = r.publication.ID
		payload["generation"] = r.publication.Generation
	}
	h.sendLocked(target, mustJSON(map[string]any{
		"type":    "media.state",
		"payload": payload,
	}))
}

func (h *Hub) disconnectLocked(linkID, sessionID string) (presenterLeft bool) {
	r := h.room(linkID)
	if r == nil {
		return false
	}
	sess, ok := r.sessions[sessionID]
	if !ok {
		return false
	}
	delete(r.sessions, sessionID)
	if sess.send != nil {
		close(sess.send)
		sess.send = nil
	}
	if r.presenter == sessionID {
		r.presenter = ""
		r.state = "waiting"
		presenterLeft = true
		h.broadcastStateLocked(linkID, "waiting")
	}
	h.broadcastLocked(linkID, mustJSON(map[string]any{
		"type":    "presence",
		"payload": map[string]any{"participantCount": attachedCount(r)},
	}))
	return presenterLeft
}

func (h *Hub) expireLocked(maxAge time.Duration) int {
	removed := 0
	now := time.Now()
	for linkID, r := range h.rooms {
		var stale []string
		for id, s := range r.sessions {
			if s.send == nil && now.Sub(s.createdAt) >= maxAge {
				stale = append(stale, id)
			}
		}
		for _, id := range stale {
			h.disconnectLocked(linkID, id)
			removed++
		}
	}
	return removed
}

func attachedCount(r *roomState) int {
	if r == nil {
		return 0
	}
	n := 0
	for _, s := range r.sessions {
		if s.send != nil {
			n++
		}
	}
	return n
}

func (h *Hub) ensure(linkID string) *roomState {
	r := h.rooms[linkID]
	if r == nil {
		r = &roomState{sessions: map[string]*Session{}, state: "waiting"}
		h.rooms[linkID] = r
	}
	return r
}

func (h *Hub) room(linkID string) *roomState {
	return h.rooms[linkID]
}

func (h *Hub) broadcastLocked(linkID string, msg []byte) {
	r := h.room(linkID)
	if r == nil {
		return
	}
	for _, s := range r.sessions {
		h.sendLocked(s, msg)
	}
}

func (h *Hub) broadcastStateLocked(linkID, state string) {
	r := h.room(linkID)
	h.broadcastLocked(linkID, mustJSON(map[string]any{
		"type": "room.state", "payload": h.statePayloadLocked(r, state),
	}))
}

func (h *Hub) statePayloadLocked(r *roomState, state string) map[string]any {
	payload := map[string]any{"state": state}
	if r != nil && r.publication != nil && state != "waiting" {
		payload["publication"] = *r.publication
	} else {
		payload["publication"] = nil
	}
	return payload
}

func (h *Hub) sendLocked(s *Session, msg []byte) {
	if s == nil || s.send == nil {
		return
	}
	select {
	case s.send <- append([]byte(nil), msg...):
	default:
	}
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"type":"internal_error"}`)
	}
	return b
}
