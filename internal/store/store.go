// Package store is the SQLite persistence layer.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY,
	username TEXT NOT NULL UNIQUE COLLATE NOCASE,
	password_hash TEXT NOT NULL,
	api_key TEXT NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS sessions (
	token TEXT PRIMARY KEY,
	user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	expires_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS authors (
	id INTEGER PRIMARY KEY,
	asin TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	image_url TEXT NOT NULL DEFAULT '',
	monitored INTEGER NOT NULL DEFAULT 0,
	monitor_new_books INTEGER NOT NULL DEFAULT 0,
	added_at INTEGER NOT NULL,
	last_sync_at INTEGER
);
CREATE TABLE IF NOT EXISTS series (
	id INTEGER PRIMARY KEY,
	asin TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS books (
	id INTEGER PRIMARY KEY,
	asin TEXT NOT NULL UNIQUE,
	title TEXT NOT NULL,
	subtitle TEXT NOT NULL DEFAULT '',
	author_id INTEGER NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
	authors TEXT NOT NULL DEFAULT '[]',
	narrators TEXT NOT NULL DEFAULT '[]',
	series_id INTEGER REFERENCES series(id) ON DELETE SET NULL,
	series_position TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	cover_url TEXT NOT NULL DEFAULT '',
	release_date TEXT NOT NULL DEFAULT '',
	runtime_minutes INTEGER NOT NULL DEFAULT 0,
	language TEXT NOT NULL DEFAULT '',
	publisher TEXT NOT NULL DEFAULT '',
	monitored INTEGER NOT NULL DEFAULT 0,
	path TEXT NOT NULL DEFAULT '',
	size_bytes INTEGER NOT NULL DEFAULT 0,
	format TEXT NOT NULL DEFAULT '',
	added_at INTEGER NOT NULL,
	last_search_at INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS books_author ON books(author_id);
CREATE INDEX IF NOT EXISTS books_series ON books(series_id);
CREATE TABLE IF NOT EXISTS book_files (
	id INTEGER PRIMARY KEY,
	book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
	path TEXT NOT NULL UNIQUE,
	size_bytes INTEGER NOT NULL,
	format TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS book_files_book ON book_files(book_id);
CREATE TABLE IF NOT EXISTS root_folders (
	id INTEGER PRIMARY KEY,
	path TEXT NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS indexers (
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	url TEXT NOT NULL,
	api_key TEXT NOT NULL DEFAULT '',
	categories TEXT NOT NULL DEFAULT '3030',
	enabled INTEGER NOT NULL DEFAULT 1,
	priority INTEGER NOT NULL DEFAULT 25
);
CREATE TABLE IF NOT EXISTS download_clients (
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	url TEXT NOT NULL,
	username TEXT NOT NULL DEFAULT '',
	password TEXT NOT NULL DEFAULT '',
	api_key TEXT NOT NULL DEFAULT '',
	category TEXT NOT NULL DEFAULT 'audiobooks',
	enabled INTEGER NOT NULL DEFAULT 1
);
CREATE TABLE IF NOT EXISTS downloads (
	id INTEGER PRIMARY KEY,
	book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
	release_title TEXT NOT NULL,
	guid TEXT NOT NULL DEFAULT '',
	indexer TEXT NOT NULL DEFAULT '',
	protocol TEXT NOT NULL,
	client_id INTEGER NOT NULL,
	client_name TEXT NOT NULL DEFAULT '',
	client_item_id TEXT NOT NULL,
	status TEXT NOT NULL,
	progress REAL NOT NULL DEFAULT 0,
	size_bytes INTEGER NOT NULL DEFAULT 0,
	eta_seconds INTEGER,
	error TEXT NOT NULL DEFAULT '',
	added_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS downloads_book ON downloads(book_id);
CREATE TABLE IF NOT EXISTS history (
	id INTEGER PRIMARY KEY,
	book_id INTEGER NOT NULL,
	book_title TEXT NOT NULL,
	author_name TEXT NOT NULL,
	type TEXT NOT NULL,
	source_title TEXT NOT NULL DEFAULT '',
	guid TEXT NOT NULL DEFAULT '',
	message TEXT NOT NULL DEFAULT '',
	date INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS history_book ON history(book_id);
CREATE TABLE IF NOT EXISTS settings (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	data TEXT NOT NULL
);
`

// Open opens (creating if needed) the database in dataDir.
func Open(dataDir string) (*Store, error) {
	dsn := "file:" + filepath.Join(dataDir, "chaptarr.db") +
		"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=foreign_keys(1)&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func unix(t time.Time) int64 { return t.Unix() }

func fromUnix(v int64) time.Time { return time.Unix(v, 0).UTC() }

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func today() string { return time.Now().Format("2006-01-02") }
