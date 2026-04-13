package store

import "time"

// DistJob represents a distribution job record.
type DistJob struct {
	ID           int64  `json:"id"`
	FileID       string `json:"file_id"`
	SourcePath   string `json:"source_path"`
	TargetType   string `json:"target_type"`
	TargetPath   string `json:"target_path"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	CompletedAt  string `json:"completed_at"`
	ErrorMessage string `json:"error_message"`
}

// InsertDistJob creates a new distribution job.
func (db *DB) InsertDistJob(j DistJob) (int64, error) {
	if j.CreatedAt == "" {
		j.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if j.Status == "" {
		j.Status = "pending"
	}
	res, err := db.conn.Exec(`
		INSERT INTO distribution_jobs (file_id, source_path, target_type, target_path, status, created_at, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		j.FileID, j.SourcePath, j.TargetType, j.TargetPath, j.Status, j.CreatedAt, j.ErrorMessage)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateDistJob updates a distribution job's status.
func (db *DB) UpdateDistJob(id int64, status string, errMsg string) error {
	completedAt := ""
	if status == "completed" || status == "failed" {
		completedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := db.conn.Exec(`
		UPDATE distribution_jobs SET status = ?, completed_at = ?, error_message = ? WHERE id = ?`,
		status, completedAt, errMsg, id)
	return err
}

// ListDistJobs returns the most recent distribution jobs.
func (db *DB) ListDistJobs(limit int) ([]DistJob, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.conn.Query(`
		SELECT id, file_id, source_path, target_type, target_path, status, created_at, COALESCE(completed_at,''), COALESCE(error_message,'')
		FROM distribution_jobs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []DistJob
	for rows.Next() {
		var j DistJob
		if err := rows.Scan(&j.ID, &j.FileID, &j.SourcePath, &j.TargetType, &j.TargetPath, &j.Status, &j.CreatedAt, &j.CompletedAt, &j.ErrorMessage); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}
