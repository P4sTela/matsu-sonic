package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/P4sTela/matsu-sonic/internal/store"
)

func TestCheckConflict_NoBaseline(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "file.txt")
	os.WriteFile(path, []byte("hello"), 0o644)

	f := &store.SyncedFile{
		FileID:    "f1",
		Name:      "file.txt",
		LocalPath: path,
		// LocalModified intentionally empty
	}

	c, err := CheckConflict(f)
	if err != nil {
		t.Fatalf("CheckConflict: %v", err)
	}
	if c != nil {
		t.Errorf("expected no conflict for file without baseline, got %+v", c)
	}
}

func TestCheckConflict_NoConflict(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "file.txt")
	os.WriteFile(path, []byte("hello"), 0o644)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	f := &store.SyncedFile{
		FileID:        "f1",
		Name:          "file.txt",
		LocalPath:     path,
		LocalSize:     info.Size(),
		LocalModified: info.ModTime().UTC().Format(time.RFC3339Nano),
	}

	c, err := CheckConflict(f)
	if err != nil {
		t.Fatalf("CheckConflict: %v", err)
	}
	if c != nil {
		t.Errorf("expected no conflict, got %+v", c)
	}
}

func TestCheckConflict_ModifiedSize(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "file.txt")
	os.WriteFile(path, []byte("hello"), 0o644)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	f := &store.SyncedFile{
		FileID:        "f1",
		Name:          "file.txt",
		LocalPath:     path,
		LocalSize:     info.Size() - 1,
		LocalModified: info.ModTime().UTC().Format(time.RFC3339Nano),
	}

	c, err := CheckConflict(f)
	if err != nil {
		t.Fatalf("CheckConflict: %v", err)
	}
	if c == nil {
		t.Fatal("expected conflict, got nil")
	}
	if c.Type != ConflictModifiedLocally {
		t.Errorf("conflict type = %q, want %q", c.Type, ConflictModifiedLocally)
	}
	if c.ActualSize != info.Size() {
		t.Errorf("actual size = %d, want %d", c.ActualSize, info.Size())
	}
}

func TestCheckConflict_ModifiedTime(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "file.txt")
	os.WriteFile(path, []byte("hello"), 0o644)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	f := &store.SyncedFile{
		FileID:        "f1",
		Name:          "file.txt",
		LocalPath:     path,
		LocalSize:     info.Size(),
		LocalModified: info.ModTime().UTC().Add(-time.Hour).Format(time.RFC3339Nano),
	}

	c, err := CheckConflict(f)
	if err != nil {
		t.Fatalf("CheckConflict: %v", err)
	}
	if c == nil {
		t.Fatal("expected conflict, got nil")
	}
	if c.Type != ConflictModifiedLocally {
		t.Errorf("conflict type = %q, want %q", c.Type, ConflictModifiedLocally)
	}
}

func TestCheckConflict_MissingLocally(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "missing.txt")

	f := &store.SyncedFile{
		FileID:        "f1",
		Name:          "missing.txt",
		LocalPath:     path,
		LocalSize:     5,
		LocalModified: time.Now().UTC().Format(time.RFC3339Nano),
	}

	c, err := CheckConflict(f)
	if err != nil {
		t.Fatalf("CheckConflict: %v", err)
	}
	if c == nil {
		t.Fatal("expected conflict, got nil")
	}
	if c.Type != ConflictMissingLocally {
		t.Errorf("conflict type = %q, want %q", c.Type, ConflictMissingLocally)
	}
}

func TestCheckConflict_SkipsFolder(t *testing.T) {
	f := &store.SyncedFile{
		FileID:        "d1",
		Name:          "folder",
		LocalPath:     "/tmp/folder",
		IsFolder:      true,
		LocalModified: time.Now().UTC().Format(time.RFC3339Nano),
	}
	c, err := CheckConflict(f)
	if err != nil {
		t.Fatalf("CheckConflict: %v", err)
	}
	if c != nil {
		t.Errorf("expected no conflict for folder, got %+v", c)
	}
}

func TestDetectConflicts(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create two files: one unchanged, one modified.
	unchanged := filepath.Join(tmpDir, "unchanged.txt")
	os.WriteFile(unchanged, []byte("same"), 0o644)
	unchangedInfo, _ := os.Stat(unchanged)

	modified := filepath.Join(tmpDir, "modified.txt")
	os.WriteFile(modified, []byte("new content"), 0o644)

	db.UpsertFile(store.SyncedFile{
		FileID:        "u1",
		Name:          "unchanged.txt",
		LocalPath:     unchanged,
		LocalSize:     unchangedInfo.Size(),
		LocalModified: unchangedInfo.ModTime().UTC().Format(time.RFC3339Nano),
	})
	db.UpsertFile(store.SyncedFile{
		FileID:        "m1",
		Name:          "modified.txt",
		LocalPath:     modified,
		LocalSize:     0,
		LocalModified: time.Now().UTC().Add(-time.Hour).Format(time.RFC3339Nano),
	})

	conflicts, err := DetectConflicts(db)
	if err != nil {
		t.Fatalf("DetectConflicts: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].FileID != "m1" {
		t.Errorf("conflict file_id = %q, want m1", conflicts[0].FileID)
	}
}

func TestRecordLocalState(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "file.txt")
	os.WriteFile(path, []byte("hello"), 0o644)

	size, mod, err := RecordLocalState(path)
	if err != nil {
		t.Fatalf("RecordLocalState: %v", err)
	}
	if size != 5 {
		t.Errorf("size = %d, want 5", size)
	}
	if mod == "" {
		t.Error("expected non-empty modification time")
	}

	// Non-existent path returns zero values.
	size, mod, err = RecordLocalState(filepath.Join(tmp, "missing.txt"))
	if err != nil {
		t.Fatalf("RecordLocalState missing: %v", err)
	}
	if size != 0 || mod != "" {
		t.Errorf("expected zero values for missing file, got size=%d mod=%q", size, mod)
	}
}

func TestIsConflictStrategy(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"skip", true},
		{"overwrite", true},
		{"", true},
		{"warn", false},
		{"other", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsConflictStrategy(tt.input); got != tt.want {
				t.Errorf("IsConflictStrategy(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
