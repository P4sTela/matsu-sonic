package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/P4sTela/matsu-sonic/internal/config"
	"github.com/P4sTela/matsu-sonic/internal/drive"
	"github.com/P4sTela/matsu-sonic/internal/store"
	driveapi "google.golang.org/api/drive/v3"
)

// --- fakes ---

type fakeBroadcaster struct {
	messages [][]byte
}

func (f *fakeBroadcaster) Broadcast(data []byte) {
	f.messages = append(f.messages, data)
}

// fakeDriveClient embeds drive.DriveClient but we won't use its Service.
// We override behavior via the SyncEngine's methods.
// For engine tests we need to provide a DriveClient with a nil Service
// and test at the integration boundary.

// Since engine.go calls methods on *drive.DriveClient directly,
// we test via an integration approach with a real DB and mock the Drive calls.

// --- helpers ---

func setupTestEngine(t *testing.T) (*SyncEngine, *store.DB, *fakeBroadcaster, string) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	syncDir := filepath.Join(tmpDir, "sync")
	os.MkdirAll(syncDir, 0o755)

	cfg := &config.Config{
		SyncFolderID: "root-folder",
		LocalSyncDir: syncDir,
		MaxWorkers:   2,
		ChunkSizeMB:  1,
		ExportFormats: map[string]config.Export{
			"application/vnd.google-apps.document": {MimeType: "application/pdf", Extension: ".pdf"},
		},
	}

	hub := &fakeBroadcaster{}
	// DriveClient with nil Service — we can't call real Drive API in tests.
	drv := &drive.DriveClient{Config: cfg}

	engine := NewSyncEngine(cfg, drv, db, hub)
	return engine, db, hub, syncDir
}

// --- tests ---

func TestNewSyncEngine(t *testing.T) {
	engine, _, _, _ := setupTestEngine(t)

	if engine.IsRunning() {
		t.Error("new engine should not be running")
	}

	snap := engine.Status()
	if snap.TotalFiles != 0 {
		t.Errorf("initial TotalFiles = %d, want 0", snap.TotalFiles)
	}
}

func TestAcquireReleaseLock(t *testing.T) {
	engine, _, _, _ := setupTestEngine(t)

	if err := engine.acquireLock(); err != nil {
		t.Fatalf("first acquire should succeed: %v", err)
	}
	if !engine.IsRunning() {
		t.Error("should be running after acquire")
	}

	// Second acquire should fail
	if err := engine.acquireLock(); err == nil {
		t.Error("second acquire should fail")
	}

	engine.releaseLock()
	if engine.IsRunning() {
		t.Error("should not be running after release")
	}

	// Should be able to acquire again
	if err := engine.acquireLock(); err != nil {
		t.Fatalf("acquire after release should succeed: %v", err)
	}
	engine.releaseLock()
}

func TestCancel(t *testing.T) {
	engine, _, _, _ := setupTestEngine(t)

	// Cancel without running should not panic
	engine.Cancel()

	// Set up a cancel func
	ctx, cancel := context.WithCancel(context.Background())
	engine.mu.Lock()
	engine.cancel = cancel
	engine.mu.Unlock()

	engine.Cancel()
	if ctx.Err() == nil {
		t.Error("context should be cancelled after Cancel()")
	}
}

func TestLocalPath(t *testing.T) {
	engine, db, _, syncDir := setupTestEngine(t)

	// Register a parent folder in DB
	db.UpsertFile(store.SyncedFile{
		FileID:   "parent-1",
		Name:     "ProjectFolder",
		IsFolder: true,
		ParentID: "root-folder",
	})

	tests := []struct {
		name     string
		file     *driveapi.File
		wantPath string
	}{
		{
			name: "simple file in root",
			file: &driveapi.File{
				Id:       "f1",
				Name:     "photo.jpg",
				MimeType: "image/jpeg",
				Parents:  []string{"root-folder"},
			},
			wantPath: filepath.Join(syncDir, "photo.jpg"),
		},
		{
			name: "file in subfolder",
			file: &driveapi.File{
				Id:       "f2",
				Name:     "report.txt",
				MimeType: "text/plain",
				Parents:  []string{"parent-1"},
			},
			wantPath: filepath.Join(syncDir, "ProjectFolder", "report.txt"),
		},
		{
			name: "google doc gets extension",
			file: &driveapi.File{
				Id:       "f3",
				Name:     "MyDoc",
				MimeType: "application/vnd.google-apps.document",
				Parents:  []string{"root-folder"},
			},
			wantPath: filepath.Join(syncDir, "MyDoc.pdf"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.localPath(tt.file)
			if got != tt.wantPath {
				t.Errorf("localPath = %q, want %q", got, tt.wantPath)
			}
		})
	}
}

func TestCreateFolders(t *testing.T) {
	engine, db, _, syncDir := setupTestEngine(t)

	folders := []*driveapi.File{
		{Id: "d1", Name: "Docs", MimeType: "application/vnd.google-apps.folder", Parents: []string{"root-folder"}},
		{Id: "d2", Name: "Images", MimeType: "application/vnd.google-apps.folder", Parents: []string{"root-folder"}},
	}

	err := engine.createFolders(context.Background(), folders)
	if err != nil {
		t.Fatalf("createFolders: %v", err)
	}

	// Verify directories exist
	for _, name := range []string{"Docs", "Images"} {
		dirPath := filepath.Join(syncDir, name)
		info, err := os.Stat(dirPath)
		if err != nil {
			t.Errorf("directory %s should exist: %v", name, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s should be a directory", name)
		}
	}

	// Verify DB records
	f, err := db.GetFile("d1")
	if err != nil {
		t.Fatalf("GetFile d1: %v", err)
	}
	if !f.IsFolder {
		t.Error("d1 should be a folder in DB")
	}
	if f.Name != "Docs" {
		t.Errorf("d1 name = %q, want %q", f.Name, "Docs")
	}
}

func TestCreateFoldersCancellation(t *testing.T) {
	engine, _, _, _ := setupTestEngine(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	folders := []*driveapi.File{
		{Id: "d1", Name: "Docs", MimeType: "application/vnd.google-apps.folder", Parents: []string{"root-folder"}},
	}

	err := engine.createFolders(ctx, folders)
	if err == nil {
		t.Error("createFolders should return error when context is cancelled")
	}
}

func TestHandleRemoval(t *testing.T) {
	engine, db, _, syncDir := setupTestEngine(t)

	// Create a file to remove
	filePath := filepath.Join(syncDir, "test.txt")
	os.WriteFile(filePath, []byte("content"), 0o644)

	db.UpsertFile(store.SyncedFile{
		FileID:    "rm1",
		Name:      "test.txt",
		LocalPath: filePath,
	})

	engine.handleRemoval("rm1")

	// File should be in trash
	trashPath := filepath.Join(syncDir, ".gdrive-trash", "test.txt")
	if _, err := os.Stat(trashPath); os.IsNotExist(err) {
		t.Error("file should be moved to .gdrive-trash")
	}

	// Original should not exist
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("original file should be removed")
	}

	// DB record should be deleted
	_, err := db.GetFile("rm1")
	if err == nil {
		t.Error("DB record should be deleted")
	}
}

func TestIsFatalError(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		fatal bool
	}{
		{"nil", nil, false},
		{"generic", fmt.Errorf("network error"), false},
		{"cancelled", context.Canceled, true},
		{"deadline", context.DeadlineExceeded, true},
		{"permission", os.ErrPermission, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFatalError(tt.err); got != tt.fatal {
				t.Errorf("isFatalError(%v) = %v, want %v", tt.err, got, tt.fatal)
			}
		})
	}
}

func TestProgressLoop(t *testing.T) {
	engine, _, hub, _ := setupTestEngine(t)

	tracker := NewProgressTracker(3)
	progressChan := make(chan ProgressEvent, 10)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		engine.progressLoop(ctx, progressChan, tracker)
		close(done)
	}()

	// Send some events
	progressChan <- ProgressEvent{Type: "file_start", FileID: "f1", FileName: "a.txt"}
	progressChan <- ProgressEvent{Type: "file_done", FileID: "f1", FileName: "a.txt", BytesDownloaded: 100}
	progressChan <- ProgressEvent{Type: "file_skip", FileID: "f2", FileName: "b.txt"}

	// Wait for ticker to fire at least once
	time.Sleep(200 * time.Millisecond)

	// Close channel to end the loop
	close(progressChan)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("progressLoop didn't finish in time")
	}
	cancel()

	// Verify tracker state
	snap := tracker.Snapshot()
	if snap.CompletedFiles != 1 {
		t.Errorf("CompletedFiles = %d, want 1", snap.CompletedFiles)
	}
	if snap.SkippedFiles != 1 {
		t.Errorf("SkippedFiles = %d, want 1", snap.SkippedFiles)
	}

	// Should have broadcast at least once
	if len(hub.messages) == 0 {
		t.Error("expected at least one broadcast message")
	}
}

func TestDownloadFilesSkipsUpToDate(t *testing.T) {
	engine, db, _, syncDir := setupTestEngine(t)

	// Pre-register a file that's already synced (matching MD5)
	localPath := filepath.Join(syncDir, "existing.txt")
	os.WriteFile(localPath, []byte("data"), 0o644)
	db.UpsertFile(store.SyncedFile{
		FileID:      "f1",
		Name:        "existing.txt",
		MD5Checksum: "abc123",
		LocalPath:   localPath,
	})

	files := []*driveapi.File{
		{
			Id:          "f1",
			Name:        "existing.txt",
			MimeType:    "text/plain",
			Md5Checksum: "abc123",
			Parents:     []string{"root-folder"},
		},
	}

	progressChan := make(chan ProgressEvent, 10)
	done := make(chan struct{})
	var events []ProgressEvent
	go func() {
		for e := range progressChan {
			events = append(events, e)
		}
		close(done)
	}()

	synced, failed, bytes := engine.downloadFiles(context.Background(), files, progressChan)
	close(progressChan)
	<-done

	if synced != 0 {
		t.Errorf("synced = %d, want 0 (should skip)", synced)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0", failed)
	}
	if bytes != 0 {
		t.Errorf("bytes = %d, want 0", bytes)
	}

	// Should have a skip event
	found := false
	for _, e := range events {
		if e.Type == "file_skip" && e.FileID == "f1" {
			found = true
		}
	}
	if !found {
		t.Error("expected file_skip event for f1")
	}
}
