package store

import (
	"database/sql"
	"fmt"
	"time"
)

// SyncedFile represents a file tracked in the database.
type SyncedFile struct {
	FileID        string `json:"file_id"`
	Name          string `json:"name"`
	MimeType      string `json:"mime_type"`
	MD5Checksum   string `json:"md5_checksum"`
	Size          int64  `json:"size"`
	DriveModified string `json:"drive_modified"`
	LocalPath     string `json:"local_path"`
	LastSynced    string `json:"last_synced"`
	ParentID      string `json:"parent_id"`
	IsFolder      bool   `json:"is_folder"`
}

// UpsertFile inserts or updates a synced file record.
func (db *DB) UpsertFile(f SyncedFile) error {
	if f.LastSynced == "" {
		f.LastSynced = time.Now().UTC().Format(time.RFC3339)
	}
	isFolder := 0
	if f.IsFolder {
		isFolder = 1
	}
	_, err := db.conn.Exec(`
		INSERT INTO synced_files (file_id, name, mime_type, md5_checksum, size, drive_modified, local_path, last_synced, parent_id, is_folder)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(file_id) DO UPDATE SET
			name = excluded.name, mime_type = excluded.mime_type, md5_checksum = excluded.md5_checksum,
			size = excluded.size, drive_modified = excluded.drive_modified, local_path = excluded.local_path,
			last_synced = excluded.last_synced, parent_id = excluded.parent_id, is_folder = excluded.is_folder
	`, f.FileID, f.Name, f.MimeType, f.MD5Checksum, f.Size, f.DriveModified, f.LocalPath, f.LastSynced, f.ParentID, isFolder)
	return err
}

// GetFile returns a single synced file by its Drive file ID.
func (db *DB) GetFile(fileID string) (*SyncedFile, error) {
	row := db.conn.QueryRow(`SELECT file_id, name, mime_type, md5_checksum, size, drive_modified, local_path, last_synced, parent_id, is_folder FROM synced_files WHERE file_id = ?`, fileID)
	return scanFile(row)
}

// ListFiles returns all synced files, optionally filtered by a search keyword.
func (db *DB) ListFiles(search string) ([]SyncedFile, error) {
	var rows *sql.Rows
	var err error
	if search != "" {
		rows, err = db.conn.Query(`SELECT file_id, name, mime_type, md5_checksum, size, drive_modified, local_path, last_synced, parent_id, is_folder FROM synced_files WHERE name LIKE ? ORDER BY name`, "%"+search+"%")
	} else {
		rows, err = db.conn.Query(`SELECT file_id, name, mime_type, md5_checksum, size, drive_modified, local_path, last_synced, parent_id, is_folder FROM synced_files ORDER BY name`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []SyncedFile
	for rows.Next() {
		var f SyncedFile
		var isFolder int
		if err := rows.Scan(&f.FileID, &f.Name, &f.MimeType, &f.MD5Checksum, &f.Size, &f.DriveModified, &f.LocalPath, &f.LastSynced, &f.ParentID, &isFolder); err != nil {
			return nil, err
		}
		f.IsFolder = isFolder != 0
		files = append(files, f)
	}
	return files, rows.Err()
}

// ClearFileChecksums clears the MD5 checksum for the given file IDs,
// so the next sync will re-download them.
func (db *DB) ClearFileChecksums(fileIDs []string) (int64, error) {
	if len(fileIDs) == 0 {
		return 0, nil
	}
	placeholders := ""
	args := make([]any, len(fileIDs))
	for i, id := range fileIDs {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
		args[i] = id
	}
	res, err := db.conn.Exec(`UPDATE synced_files SET md5_checksum = '' WHERE file_id IN (`+placeholders+`)`, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DeleteFile removes a synced file record.
func (db *DB) DeleteFile(fileID string) error {
	_, err := db.conn.Exec(`DELETE FROM synced_files WHERE file_id = ?`, fileID)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanFile(s scanner) (*SyncedFile, error) {
	var f SyncedFile
	var isFolder int
	err := s.Scan(&f.FileID, &f.Name, &f.MimeType, &f.MD5Checksum, &f.Size, &f.DriveModified, &f.LocalPath, &f.LastSynced, &f.ParentID, &isFolder)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("file not found")
		}
		return nil, err
	}
	f.IsFolder = isFolder != 0
	return &f, nil
}
