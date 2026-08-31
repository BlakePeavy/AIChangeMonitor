package store

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	agent TEXT NOT NULL,
	started_at TEXT,
	ended_at TEXT,
	cwd TEXT,
	repo TEXT,
	branch TEXT,
	intent TEXT,
	why TEXT,
	files TEXT,
	status TEXT NOT NULL DEFAULT 'unseen',
	risks TEXT,
	source_path TEXT,
	source_mtime INTEGER,
	added_lines INTEGER,
	deleted_lines INTEGER,
	source TEXT
);
CREATE TABLE IF NOT EXISTS sources (
	path TEXT PRIMARY KEY,
	mtime INTEGER
);
CREATE INDEX IF NOT EXISTS sessions_repo ON sessions(repo);
`)
	if err != nil {
		return err
	}
	// Old DBs created before `source` existed. Ignore "duplicate column".
	_, _ = s.db.Exec(`ALTER TABLE sessions ADD COLUMN source TEXT`)
	return nil
}
