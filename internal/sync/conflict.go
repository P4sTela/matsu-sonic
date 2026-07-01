package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/P4sTela/matsu-sonic/internal/store"
)

// ConflictType describes the kind of local conflict detected.
type ConflictType string

const (
	// ConflictModifiedLocally means the local file has changed since last sync.
	ConflictModifiedLocally ConflictType = "modified_locally"
	// ConflictMissingLocally means the local file no longer exists.
	ConflictMissingLocally ConflictType = "missing_locally"
)

// Conflict represents a discrepancy between the local filesystem state and
// the state recorded at the time of the last successful sync.
type Conflict struct {
	FileID       string       `json:"file_id"`
	Name         string       `json:"name"`
	LocalPath    string       `json:"local_path"`
	Type         ConflictType `json:"type"`
	ExpectedSize int64        `json:"expected_size"`
	ActualSize   int64        `json:"actual_size"`
	ExpectedMod  string       `json:"expected_modified"`
	ActualMod    string       `json:"actual_modified"`
}

// IsConflictStrategy reports whether s is a known conflict strategy.
func IsConflictStrategy(s string) bool {
	switch s {
	case "skip", "overwrite", "":
		return true
	}
	return false
}

// CheckConflict compares the current local filesystem state of f against the
// state captured at the last sync. It returns nil when no conflict is detected
// or when no baseline is available (e.g. files synced before this feature was
// introduced).
func CheckConflict(f *store.SyncedFile) (*Conflict, error) {
	if f == nil || f.IsFolder || f.LocalPath == "" {
		return nil, nil
	}

	// No baseline: avoid false positives on legacy records.
	if f.LocalModified == "" {
		return nil, nil
	}

	info, err := os.Stat(f.LocalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Conflict{
				FileID:    f.FileID,
				Name:      f.Name,
				LocalPath: f.LocalPath,
				Type:      ConflictMissingLocally,
			}, nil
		}
		return nil, fmt.Errorf("stat %s: %w", f.LocalPath, err)
	}

	actualMod := info.ModTime().UTC().Format(time.RFC3339Nano)
	if info.Size() != f.LocalSize || actualMod != f.LocalModified {
		return &Conflict{
			FileID:       f.FileID,
			Name:         f.Name,
			LocalPath:    f.LocalPath,
			Type:         ConflictModifiedLocally,
			ExpectedSize: f.LocalSize,
			ActualSize:   info.Size(),
			ExpectedMod:  f.LocalModified,
			ActualMod:    actualMod,
		}, nil
	}

	return nil, nil
}

// DetectConflicts scans all non-folder synced files and returns those whose
// local state differs from the recorded baseline.
func DetectConflicts(st *store.DB) ([]Conflict, error) {
	files, err := st.ListFiles("")
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	var conflicts []Conflict
	for _, f := range files {
		if f.IsFolder {
			continue
		}
		c, err := CheckConflict(&f)
		if err != nil {
			return nil, err
		}
		if c != nil {
			conflicts = append(conflicts, *c)
		}
	}
	return conflicts, nil
}

// RecordLocalState captures the current size and modification time of path.
// It returns zero values when the file does not exist.
func RecordLocalState(path string) (int64, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, "", nil
		}
		return 0, "", fmt.Errorf("stat %s: %w", path, err)
	}
	return info.Size(), info.ModTime().UTC().Format(time.RFC3339Nano), nil
}

// normalizeLocalPath returns a clean absolute path for consistent comparison.
func normalizeLocalPath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}
