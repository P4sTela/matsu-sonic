package store

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Conversion represents a single conversion execution record.
type Conversion struct {
	ID               string `json:"id"`
	FileID           string `json:"file_id"`
	Converter        string `json:"converter"`
	InputPath        string `json:"input_path"`
	OutputPath       string `json:"output_path"`
	Status           string `json:"status"` // pending | running | completed | failed
	ErrorMessage     string `json:"error_message,omitempty"`
	StartedAt        string `json:"started_at,omitempty"`
	FinishedAt       string `json:"finished_at,omitempty"`
	OriginalSize     int64  `json:"original_size"`
	OriginalModified string `json:"original_modified"`
}

// InsertConversion creates a new pending conversion record.
func (db *DB) InsertConversion(fileID, converter, inputPath string) (*Conversion, error) {
	id := newID()
	now := time.Now().UTC().Format(time.RFC3339)
	c := &Conversion{
		ID:        id,
		FileID:    fileID,
		Converter: converter,
		InputPath: inputPath,
		Status:    "pending",
		StartedAt: now,
	}
	_, err := db.conn.Exec(`
		INSERT INTO conversions (id, file_id, converter, input_path, status, started_at)
		VALUES (?, ?, ?, ?, 'running', ?)
		ON CONFLICT(file_id, converter) DO UPDATE SET
			status = 'running', started_at = ?, error_message = NULL, finished_at = NULL
	`, id, fileID, converter, inputPath, now, now)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// FinishConversion marks a conversion as completed and records output metadata.
func (db *DB) FinishConversion(id, outputPath string, originalSize int64, originalModified string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.conn.Exec(`
		UPDATE conversions SET status = 'completed', finished_at = ?, output_path = ?,
			original_size = ?, original_modified = ?
		WHERE id = ?
	`, now, outputPath, originalSize, originalModified, id)
	return err
}

// FailConversion marks a conversion as failed with an error message.
func (db *DB) FailConversion(id, errMsg string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.conn.Exec(`UPDATE conversions SET status = 'failed', finished_at = ?, error_message = ? WHERE id = ?`,
		now, errMsg, id)
	return err
}

// GetConversion returns a conversion by file ID and converter name.
func (db *DB) GetConversion(fileID, converter string) (*Conversion, error) {
	row := db.conn.QueryRow(`
		SELECT id, file_id, converter, input_path, COALESCE(output_path,''), status,
			COALESCE(error_message,''), COALESCE(started_at,''), COALESCE(finished_at,''),
			original_size, original_modified
		FROM conversions WHERE file_id = ? AND converter = ?
	`, fileID, converter)
	return scanConversion(row)
}

// GetConversionByID returns a conversion by its ID.
func (db *DB) GetConversionByID(id string) (*Conversion, error) {
	row := db.conn.QueryRow(`
		SELECT id, file_id, converter, input_path, COALESCE(output_path,''), status,
			COALESCE(error_message,''), COALESCE(started_at,''), COALESCE(finished_at,''),
			original_size, original_modified
		FROM conversions WHERE id = ?
	`, id)
	return scanConversion(row)
}

// ListConversionsByFile returns all conversions for a given file.
func (db *DB) ListConversionsByFile(fileID string) ([]Conversion, error) {
	rows, err := db.conn.Query(`
		SELECT id, file_id, converter, input_path, COALESCE(output_path,''), status,
			COALESCE(error_message,''), COALESCE(started_at,''), COALESCE(finished_at,''),
			original_size, original_modified
		FROM conversions WHERE file_id = ? ORDER BY started_at DESC
	`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConversions(rows)
}

// ListConversions returns recent conversions with an optional limit.
func (db *DB) ListConversions(limit int) ([]Conversion, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.conn.Query(`
		SELECT id, file_id, converter, input_path, COALESCE(output_path,''), status,
			COALESCE(error_message,''), COALESCE(started_at,''), COALESCE(finished_at,''),
			original_size, original_modified
		FROM conversions ORDER BY started_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConversions(rows)
}

// ListStaleConversions returns completed conversions whose original file
// no longer matches the recorded size/modified time (stale), or whose
// input or output files are missing.
func (db *DB) ListStaleConversions() ([]Conversion, error) {
	rows, err := db.conn.Query(`
		SELECT id, file_id, converter, input_path, COALESCE(output_path,''), status,
			COALESCE(error_message,''), COALESCE(started_at,''), COALESCE(finished_at,''),
			original_size, original_modified
		FROM conversions WHERE status = 'completed' ORDER BY started_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConversions(rows)
}

// DeleteConversion removes a conversion record.
func (db *DB) DeleteConversion(id string) error {
	_, err := db.conn.Exec(`DELETE FROM conversions WHERE id = ?`, id)
	return err
}

// ---- helpers ----

func newID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func scanConversion(s scanner) (*Conversion, error) {
	var c Conversion
	err := s.Scan(&c.ID, &c.FileID, &c.Converter, &c.InputPath, &c.OutputPath,
		&c.Status, &c.ErrorMessage, &c.StartedAt, &c.FinishedAt,
		&c.OriginalSize, &c.OriginalModified)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func scanConversions(rows interface{ Next() bool; Scan(...any) error; Close() error }) ([]Conversion, error) {
	var cs []Conversion
	for rows.Next() {
		var c Conversion
		if err := rows.Scan(&c.ID, &c.FileID, &c.Converter, &c.InputPath, &c.OutputPath,
			&c.Status, &c.ErrorMessage, &c.StartedAt, &c.FinishedAt,
			&c.OriginalSize, &c.OriginalModified); err != nil {
			return nil, err
		}
		cs = append(cs, c)
	}
	return cs, rows.Close()
}
