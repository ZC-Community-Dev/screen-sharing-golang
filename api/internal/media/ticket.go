package media

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

type TicketRole string

const (
	RolePublisher TicketRole = "publisher"
	RoleViewer    TicketRole = "viewer"
)

var (
	ErrTicketInvalid = errors.New("media ticket invalid")
	ErrTicketExpired = errors.New("media ticket expired")
	ErrTicketBinding = errors.New("media ticket binding mismatch")
)

type TicketClaims struct {
	LinkID         string
	SessionID      string
	Role           TicketRole
	Transport      Transport
	Generation     uint64
	PublicationID  string
	MediaSessionID string
}

func (s *TicketStore) ConsumeForLink(raw, linkID string) (Ticket, error) {
	if raw == "" {
		return Ticket{}, ErrTicketInvalid
	}
	hash := sha256.Sum256([]byte(raw))
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	ticket, ok := s.records[hash]
	if !ok {
		s.purgeLocked(now)
		return Ticket{}, ErrTicketInvalid
	}
	delete(s.records, hash)
	s.purgeLocked(now)
	if !now.Before(ticket.ExpiresAt) {
		return Ticket{}, ErrTicketExpired
	}
	if !sameString(ticket.LinkID, linkID) {
		return Ticket{}, ErrTicketBinding
	}
	return ticket, nil
}

type TicketBinding struct {
	LinkID     string
	SessionID  string
	Role       TicketRole
	Generation uint64
}

type Ticket struct {
	TicketClaims
	ExpiresAt time.Time
}

type TicketStore struct {
	mu      sync.Mutex
	ttl     time.Duration
	now     func() time.Time
	records map[[sha256.Size]byte]Ticket
}

func NewTicketStore(ttl time.Duration, now func() time.Time) *TicketStore {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	if now == nil {
		now = time.Now
	}
	return &TicketStore{ttl: ttl, now: now, records: make(map[[sha256.Size]byte]Ticket)}
}

func (s *TicketStore) Issue(claims TicketClaims) (string, Ticket, error) {
	if claims.LinkID == "" || claims.SessionID == "" ||
		(claims.Role != RolePublisher && claims.Role != RoleViewer) ||
		claims.Transport != TransportWebSocket {
		return "", Ticket{}, ErrTicketBinding
	}
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", Ticket{}, err
	}
	value := base64.RawURLEncoding.EncodeToString(raw[:])
	hash := sha256.Sum256([]byte(value))
	ticket := Ticket{TicketClaims: claims, ExpiresAt: s.now().Add(s.ttl)}
	s.mu.Lock()
	s.purgeLocked(s.now())
	s.records[hash] = ticket
	s.mu.Unlock()
	return value, ticket, nil
}

func (s *TicketStore) Consume(raw string, binding TicketBinding) (Ticket, error) {
	if raw == "" {
		return Ticket{}, ErrTicketInvalid
	}
	hash := sha256.Sum256([]byte(raw))
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	ticket, ok := s.records[hash]
	if !ok {
		s.purgeLocked(now)
		return Ticket{}, ErrTicketInvalid
	}
	delete(s.records, hash)
	s.purgeLocked(now)
	if !now.Before(ticket.ExpiresAt) {
		return Ticket{}, ErrTicketExpired
	}
	if !sameString(ticket.LinkID, binding.LinkID) ||
		!sameString(ticket.SessionID, binding.SessionID) ||
		ticket.Role != binding.Role ||
		ticket.Generation != binding.Generation {
		return Ticket{}, ErrTicketBinding
	}
	return ticket, nil
}

func (s *TicketStore) DeleteForSession(linkID, sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for hash, ticket := range s.records {
		if ticket.LinkID == linkID && ticket.SessionID == sessionID {
			delete(s.records, hash)
		}
	}
}

func (s *TicketStore) DeleteForLink(linkID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for hash, ticket := range s.records {
		if ticket.LinkID == linkID {
			delete(s.records, hash)
		}
	}
}

func (s *TicketStore) Close() {
	s.mu.Lock()
	clear(s.records)
	s.mu.Unlock()
}

func (s *TicketStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeLocked(s.now())
	return len(s.records)
}

// DebugHashes is intentionally useful only for tests: it exposes hashes, never
// bearer values.
func (s *TicketStore) DebugHashes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]byte, 0, len(s.records)*sha256.Size)
	for hash := range s.records {
		out = append(out, hash[:]...)
	}
	return out
}

func (s *TicketStore) purgeLocked(now time.Time) {
	for hash, ticket := range s.records {
		if !now.Before(ticket.ExpiresAt) {
			delete(s.records, hash)
		}
	}
}

func sameString(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
