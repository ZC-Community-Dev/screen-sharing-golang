package links

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"api/internal/ids"
)

var (
	ErrNotFound = errors.New("link not found")
	ErrConflict = errors.New("id conflict")
)

type Service struct {
	db   *sql.DB
	salt string
}

func NewService(database *sql.DB, salt string) *Service {
	return &Service{db: database, salt: salt}
}

type Created struct {
	ID             string
	PublicURL      string
	PresenterToken string
}

func (s *Service) Create() (Created, error) {
	token, err := ids.NewPresenterToken()
	if err != nil {
		return Created{}, err
	}
	hash := ids.HashToken(token)
	var id string
	for range 8 {
		id, err = ids.Generate(s.salt)
		if err != nil {
			return Created{}, err
		}
		exists, err := s.exists(id)
		if err != nil {
			return Created{}, err
		}
		if !exists {
			break
		}
		id = ""
	}
	if id == "" {
		return Created{}, fmt.Errorf("could not allocate unique id")
	}
	link := Link{
		ID:                 id,
		PresenterTokenHash: hash,
		CreatedAt:          time.Now().UTC(),
		State:              StateWaiting,
	}
	if err := s.insert(link); err != nil {
		return Created{}, err
	}
	return Created{ID: id, PublicURL: "/r/" + id, PresenterToken: token}, nil
}

func (s *Service) GetByID(id string) (Link, error) {
	var link Link
	var created string
	err := s.db.QueryRow(
		`SELECT id, presenter_token_hash, created_at, state FROM links WHERE id = ?`,
		id,
	).Scan(&link.ID, &link.PresenterTokenHash, &created, &link.State)
	if errors.Is(err, sql.ErrNoRows) {
		return Link{}, ErrNotFound
	}
	if err != nil {
		return Link{}, err
	}
	link.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return link, nil
}

func (s *Service) VerifyToken(id, token string) error {
	link, err := s.GetByID(id)
	if err != nil {
		return err
	}
	want := ids.HashToken(token)
	if subtle.ConstantTimeCompare([]byte(want), []byte(link.PresenterTokenHash)) != 1 {
		return ErrUnauthorized
	}
	return nil
}

var ErrUnauthorized = errors.New("presenter unauthorized")

func (s *Service) SetState(id, state string) error {
	res, err := s.db.Exec(`UPDATE links SET state = ? WHERE id = ?`, state, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) insert(link Link) error {
	_, err := s.db.Exec(
		`INSERT INTO links (id, presenter_token_hash, created_at, state) VALUES (?, ?, ?, ?)`,
		link.ID, link.PresenterTokenHash, link.CreatedAt.Format(time.RFC3339), link.State,
	)
	return err
}

func (s *Service) exists(id string) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM links WHERE id = ?`, id).Scan(&n)
	return n > 0, err
}
