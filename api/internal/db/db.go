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
		BEGIN IMMEDIATE;
		CREATE TABLE IF NOT EXISTS links (
			id TEXT PRIMARY KEY,
			presenter_token_hash TEXT NOT NULL,
			created_at TEXT NOT NULL,
			state TEXT NOT NULL
		);
		CREATE TABLE links_media_v2 (
			id TEXT PRIMARY KEY,
			presenter_token_hash TEXT NOT NULL,
			created_at TEXT NOT NULL,
			state TEXT NOT NULL CHECK (state IN ('waiting', 'connecting', 'sharing', 'reconnecting', 'failed'))
		);
		INSERT INTO links_media_v2 (id, presenter_token_hash, created_at, state)
			SELECT id, presenter_token_hash, created_at, 'waiting' FROM links;
		DROP TABLE links;
		ALTER TABLE links_media_v2 RENAME TO links;
		COMMIT;
	`); err != nil {
		_, _ = database.Exec(`ROLLBACK`)
		_ = database.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return database, nil
}
