package store

import "time"

// DownloadedRevision represents a downloaded file revision.
type DownloadedRevision struct {
	ID           int64  `json:"id"`
	FileID       string `json:"file_id"`
	RevisionID   string `json:"revision_id"`
	LocalPath    string `json:"local_path"`
	DownloadedAt string `json:"downloaded_at"`
	Size         int64  `json:"size"`
}

// InsertRevision records a downloaded revision.
func (db *DB) InsertRevision(r DownloadedRevision) (int64, error) {
	if r.DownloadedAt == "" {
		r.DownloadedAt = time.Now().UTC().Format(time.RFC3339)
	}
	res, err := db.conn.Exec(`
		INSERT INTO downloaded_revisions (file_id, revision_id, local_path, downloaded_at, size)
		VALUES (?, ?, ?, ?, ?)`,
		r.FileID, r.RevisionID, r.LocalPath, r.DownloadedAt, r.Size)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListDownloadedRevisions returns all downloaded revisions for a file.
func (db *DB) ListDownloadedRevisions(fileID string) ([]DownloadedRevision, error) {
	rows, err := db.conn.Query(`
		SELECT id, file_id, revision_id, local_path, downloaded_at, COALESCE(size, 0)
		FROM downloaded_revisions WHERE file_id = ? ORDER BY downloaded_at DESC`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revs []DownloadedRevision
	for rows.Next() {
		var r DownloadedRevision
		if err := rows.Scan(&r.ID, &r.FileID, &r.RevisionID, &r.LocalPath, &r.DownloadedAt, &r.Size); err != nil {
			return nil, err
		}
		revs = append(revs, r)
	}
	return revs, rows.Err()
}
