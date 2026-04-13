package store

import (
	"path/filepath"
	"testing"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestNewAndMigrate(t *testing.T) {
	db := newTestDB(t)

	var version int
	err := db.Conn().QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&version)
	if err != nil {
		t.Fatalf("query schema_version: %v", err)
	}
	if version != 1 {
		t.Errorf("schema version = %d, want 1", version)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db1, err := New(dbPath)
	if err != nil {
		t.Fatalf("first New() error = %v", err)
	}
	db1.Close()

	db2, err := New(dbPath)
	if err != nil {
		t.Fatalf("second New() error = %v", err)
	}
	db2.Close()
}

// --- synced_files ---

func TestUpsertAndGetFile(t *testing.T) {
	db := newTestDB(t)

	f := SyncedFile{
		FileID:        "file1",
		Name:          "doc.pdf",
		MimeType:      "application/pdf",
		MD5Checksum:   "abc123",
		Size:          1024,
		DriveModified: "2026-01-01T00:00:00Z",
		LocalPath:     "/sync/doc.pdf",
		ParentID:      "root",
	}

	if err := db.UpsertFile(f); err != nil {
		t.Fatalf("UpsertFile() error = %v", err)
	}

	got, err := db.GetFile("file1")
	if err != nil {
		t.Fatalf("GetFile() error = %v", err)
	}
	if got.Name != "doc.pdf" {
		t.Errorf("Name = %q, want %q", got.Name, "doc.pdf")
	}
	if got.MD5Checksum != "abc123" {
		t.Errorf("MD5 = %q, want %q", got.MD5Checksum, "abc123")
	}
	if got.LastSynced == "" {
		t.Error("LastSynced should be auto-filled")
	}
}

func TestUpsertFileUpdate(t *testing.T) {
	db := newTestDB(t)

	f := SyncedFile{FileID: "file1", Name: "v1.pdf", MimeType: "application/pdf"}
	if err := db.UpsertFile(f); err != nil {
		t.Fatalf("UpsertFile() error = %v", err)
	}

	f.Name = "v2.pdf"
	f.Size = 2048
	if err := db.UpsertFile(f); err != nil {
		t.Fatalf("UpsertFile() update error = %v", err)
	}

	got, err := db.GetFile("file1")
	if err != nil {
		t.Fatalf("GetFile() error = %v", err)
	}
	if got.Name != "v2.pdf" {
		t.Errorf("Name = %q, want %q", got.Name, "v2.pdf")
	}
	if got.Size != 2048 {
		t.Errorf("Size = %d, want 2048", got.Size)
	}
}

func TestListFiles(t *testing.T) {
	db := newTestDB(t)

	for _, name := range []string{"alpha.pdf", "beta.doc", "gamma.pdf"} {
		if err := db.UpsertFile(SyncedFile{FileID: name, Name: name}); err != nil {
			t.Fatalf("UpsertFile(%s) error = %v", name, err)
		}
	}

	all, err := db.ListFiles("")
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}
	if len(all) != 3 {
		t.Errorf("ListFiles() returned %d, want 3", len(all))
	}

	filtered, err := db.ListFiles("pdf")
	if err != nil {
		t.Fatalf("ListFiles(pdf) error = %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("ListFiles(pdf) returned %d, want 2", len(filtered))
	}
}

func TestDeleteFile(t *testing.T) {
	db := newTestDB(t)

	if err := db.UpsertFile(SyncedFile{FileID: "f1", Name: "test"}); err != nil {
		t.Fatalf("UpsertFile() error = %v", err)
	}

	if err := db.DeleteFile("f1"); err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}

	_, err := db.GetFile("f1")
	if err == nil {
		t.Error("GetFile() after delete should error")
	}
}

func TestGetFileNotFound(t *testing.T) {
	db := newTestDB(t)

	_, err := db.GetFile("nonexistent")
	if err == nil {
		t.Error("GetFile(nonexistent) should error")
	}
}

// --- sync_runs ---

func TestStartAndFinishRun(t *testing.T) {
	db := newTestDB(t)

	id, err := db.StartRun()
	if err != nil {
		t.Fatalf("StartRun() error = %v", err)
	}
	if id <= 0 {
		t.Errorf("StartRun() id = %d, want > 0", id)
	}

	if err := db.FinishRun(id, "completed", 10, 2, 5000, "token123"); err != nil {
		t.Fatalf("FinishRun() error = %v", err)
	}

	runs, err := db.ListRuns(10)
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("ListRuns() returned %d, want 1", len(runs))
	}
	if runs[0].Status != "completed" {
		t.Errorf("Status = %q, want %q", runs[0].Status, "completed")
	}
	if runs[0].FilesSynced != 10 {
		t.Errorf("FilesSynced = %d, want 10", runs[0].FilesSynced)
	}
}

func TestGetLastChangeToken(t *testing.T) {
	db := newTestDB(t)

	token, err := db.GetLastChangeToken()
	if err != nil {
		t.Fatalf("GetLastChangeToken() error = %v", err)
	}
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}

	id, _ := db.StartRun()
	db.FinishRun(id, "completed", 1, 0, 100, "tok1")

	id2, _ := db.StartRun()
	db.FinishRun(id2, "completed", 2, 0, 200, "tok2")

	token, err = db.GetLastChangeToken()
	if err != nil {
		t.Fatalf("GetLastChangeToken() error = %v", err)
	}
	if token != "tok2" {
		t.Errorf("token = %q, want %q", token, "tok2")
	}
}

// --- downloaded_revisions ---

func TestInsertAndListRevisions(t *testing.T) {
	db := newTestDB(t)

	id, err := db.InsertRevision(DownloadedRevision{
		FileID:     "file1",
		RevisionID: "rev1",
		LocalPath:  "/revisions/file1.rev1.pdf",
		Size:       512,
	})
	if err != nil {
		t.Fatalf("InsertRevision() error = %v", err)
	}
	if id <= 0 {
		t.Errorf("InsertRevision() id = %d, want > 0", id)
	}

	revs, err := db.ListDownloadedRevisions("file1")
	if err != nil {
		t.Fatalf("ListDownloadedRevisions() error = %v", err)
	}
	if len(revs) != 1 {
		t.Fatalf("got %d revisions, want 1", len(revs))
	}
	if revs[0].RevisionID != "rev1" {
		t.Errorf("RevisionID = %q, want %q", revs[0].RevisionID, "rev1")
	}
}

// --- distribution_jobs ---

func TestInsertAndListDistJobs(t *testing.T) {
	db := newTestDB(t)

	id, err := db.InsertDistJob(DistJob{
		FileID:     "file1",
		SourcePath: "/sync/file1.pdf",
		TargetType: "local",
		TargetPath: "/backup/file1.pdf",
	})
	if err != nil {
		t.Fatalf("InsertDistJob() error = %v", err)
	}

	if err := db.UpdateDistJob(id, "completed", ""); err != nil {
		t.Fatalf("UpdateDistJob() error = %v", err)
	}

	jobs, err := db.ListDistJobs(10)
	if err != nil {
		t.Fatalf("ListDistJobs() error = %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("got %d jobs, want 1", len(jobs))
	}
	if jobs[0].Status != "completed" {
		t.Errorf("Status = %q, want %q", jobs[0].Status, "completed")
	}
	if jobs[0].CompletedAt == "" {
		t.Error("CompletedAt should be set for completed job")
	}
}

func TestUpdateDistJobFailed(t *testing.T) {
	db := newTestDB(t)

	id, _ := db.InsertDistJob(DistJob{
		FileID:     "file2",
		SourcePath: "/sync/file2.pdf",
		TargetType: "smb",
		TargetPath: "//server/share/file2.pdf",
	})

	if err := db.UpdateDistJob(id, "failed", "connection refused"); err != nil {
		t.Fatalf("UpdateDistJob() error = %v", err)
	}

	jobs, _ := db.ListDistJobs(10)
	if jobs[0].ErrorMessage != "connection refused" {
		t.Errorf("ErrorMessage = %q, want %q", jobs[0].ErrorMessage, "connection refused")
	}
}
