package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS links (
			id TEXT PRIMARY KEY,
			presenter_token_hash TEXT NOT NULL,
			created_at TEXT NOT NULL,
			state TEXT NOT NULL CHECK (state IN ('waiting', 'sharing'))
		);
		UPDATE links SET state = 'waiting' WHERE state = 'sharing';
	`); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return database, nil
}
