package store

import (
	"database/sql"
	"time"
)

// SyncRun represents a sync execution record.
type SyncRun struct {
	ID              int64  `json:"id"`
	StartedAt       string `json:"started_at"`
	FinishedAt      string `json:"finished_at"`
	Status          string `json:"status"`
	FilesSynced     int    `json:"files_synced"`
	FilesFailed     int    `json:"files_failed"`
	BytesDownloaded int64  `json:"bytes_downloaded"`
	ChangeToken     string `json:"change_token"`
}

// StartRun creates a new sync run with status "running".
func (db *DB) StartRun() (int64, error) {
	res, err := db.conn.Exec(`INSERT INTO sync_runs (started_at, status) VALUES (?, 'running')`,
		time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// FinishRun updates a sync run with final results.
func (db *DB) FinishRun(id int64, status string, filesSynced, filesFailed int, bytesDownloaded int64, changeToken string) error {
	_, err := db.conn.Exec(`
		UPDATE sync_runs SET finished_at = ?, status = ?, files_synced = ?, files_failed = ?, bytes_downloaded = ?, change_token = ?
		WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), status, filesSynced, filesFailed, bytesDownloaded, changeToken, id)
	return err
}

// GetLastChangeToken returns the change token from the most recent completed run.
func (db *DB) GetLastChangeToken() (string, error) {
	var token sql.NullString
	err := db.conn.QueryRow(`SELECT change_token FROM sync_runs WHERE status = 'completed' AND change_token IS NOT NULL ORDER BY id DESC LIMIT 1`).Scan(&token)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return token.String, nil
}

// ListRuns returns the most recent sync runs.
func (db *DB) ListRuns(limit int) ([]SyncRun, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := db.conn.Query(`SELECT id, started_at, COALESCE(finished_at,''), status, files_synced, files_failed, bytes_downloaded, COALESCE(change_token,'') FROM sync_runs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []SyncRun
	for rows.Next() {
		var r SyncRun
		if err := rows.Scan(&r.ID, &r.StartedAt, &r.FinishedAt, &r.Status, &r.FilesSynced, &r.FilesFailed, &r.BytesDownloaded, &r.ChangeToken); err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}
