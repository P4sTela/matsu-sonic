package store

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps a SQLite connection with migration support.
type DB struct {
	conn *sql.DB
}

// New opens (or creates) a SQLite database at dbPath and runs migrations.
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// Close closes the underlying database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying *sql.DB for advanced usage.
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// ClearAll deletes all synced files and sync run records.
func (db *DB) ClearAll() error {
	_, err := db.conn.Exec(`DELETE FROM synced_files; DELETE FROM sync_runs; DELETE FROM downloaded_revisions; DELETE FROM distribution_jobs;`)
	return err
}

var migrations = []string{
	// v1: initial schema
	`CREATE TABLE IF NOT EXISTS synced_files (
		file_id         TEXT PRIMARY KEY,
		name            TEXT NOT NULL,
		mime_type       TEXT,
		md5_checksum    TEXT,
		size            INTEGER,
		drive_modified  TEXT,
		local_path      TEXT,
		last_synced     TEXT,
		parent_id       TEXT,
		is_folder       INTEGER DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_synced_parent ON synced_files(parent_id);

	CREATE TABLE IF NOT EXISTS sync_runs (
		id               INTEGER PRIMARY KEY AUTOINCREMENT,
		started_at       TEXT NOT NULL,
		finished_at      TEXT,
		status           TEXT DEFAULT 'running',
		files_synced     INTEGER DEFAULT 0,
		files_failed     INTEGER DEFAULT 0,
		bytes_downloaded INTEGER DEFAULT 0,
		change_token     TEXT
	);

	CREATE TABLE IF NOT EXISTS downloaded_revisions (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		file_id       TEXT NOT NULL,
		revision_id   TEXT NOT NULL,
		local_path    TEXT NOT NULL,
		downloaded_at TEXT NOT NULL,
		size          INTEGER,
		UNIQUE(file_id, revision_id)
	);
	CREATE INDEX IF NOT EXISTS idx_revisions_file ON downloaded_revisions(file_id);

	CREATE TABLE IF NOT EXISTS distribution_jobs (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		file_id       TEXT NOT NULL,
		source_path   TEXT NOT NULL,
		target_type   TEXT NOT NULL,
		target_path   TEXT NOT NULL,
		status        TEXT DEFAULT 'pending',
		created_at    TEXT NOT NULL,
		completed_at  TEXT,
		error_message TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_dist_status ON distribution_jobs(status);`,
}

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY)`)
	if err != nil {
		return err
	}

	var current int
	row := db.conn.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`)
	if err := row.Scan(&current); err != nil {
		return err
	}

	for i := current; i < len(migrations); i++ {
		tx, err := db.conn.Begin()
		if err != nil {
			return err
		}

		if _, err := tx.Exec(migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}

		if _, err := tx.Exec(`INSERT INTO schema_version (version) VALUES (?)`, i+1); err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}
