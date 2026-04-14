package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	gosync "sync"
	"syscall"
	"time"

	"github.com/P4sTela/matsu-sonic/internal/config"
	"github.com/P4sTela/matsu-sonic/internal/drive"
	"github.com/P4sTela/matsu-sonic/internal/store"
	"golang.org/x/sync/errgroup"
	driveapi "google.golang.org/api/drive/v3"
)

// Broadcaster is the interface for sending progress snapshots to clients.
type Broadcaster interface {
	Broadcast(data []byte)
}

// SyncEngine orchestrates full and incremental sync operations.
type SyncEngine struct {
	cfg      *config.Config
	drive    *drive.DriveClient
	store    *store.DB
	hub      Broadcaster
	progress *ProgressTracker
	cancel   context.CancelFunc
	mu       gosync.Mutex
	running  bool
}

// NewSyncEngine creates a new SyncEngine.
func NewSyncEngine(cfg *config.Config, drv *drive.DriveClient, st *store.DB, hub Broadcaster) *SyncEngine {
	return &SyncEngine{
		cfg:   cfg,
		drive: drv,
		store: st,
		hub:   hub,
	}
}

// SetDriveClient replaces the Drive client (e.g. after config change).
func (e *SyncEngine) SetDriveClient(drv *drive.DriveClient) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.drive = drv
}

// IsRunning returns whether a sync is currently in progress.
func (e *SyncEngine) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

// Status returns the current progress snapshot.
func (e *SyncEngine) Status() ProgressSnapshot {
	e.mu.Lock()
	p := e.progress
	e.mu.Unlock()

	if p == nil {
		return ProgressSnapshot{}
	}
	return p.Snapshot()
}

// Cancel requests cancellation of the running sync.
func (e *SyncEngine) Cancel() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cancel != nil {
		e.cancel()
	}
}

// StartFull runs a full sync: list all files from Drive and download as needed.
func (e *SyncEngine) StartFull(ctx context.Context) error {
	if err := e.acquireLock(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	e.cancel = cancel
	e.mu.Unlock()

	defer func() {
		cancel()
		e.releaseLock()
	}()

	runID, err := e.store.StartRun()
	if err != nil {
		return fmt.Errorf("start run: %w", err)
	}

	// List all files from Drive
	log.Printf("[sync] full: listing files from folder %s", e.cfg.SyncFolderID)
	files, err := e.drive.ListAllRecursive(ctx, e.cfg.SyncFolderID)
	if err != nil {
		e.finishRun(runID, "failed", 0, 0, 0, "")
		return fmt.Errorf("list files: %w", err)
	}

	// Separate folders and files
	var folders, regularFiles []*driveapi.File
	for _, f := range files {
		if f.MimeType == "application/vnd.google-apps.folder" {
			folders = append(folders, f)
		} else {
			regularFiles = append(regularFiles, f)
		}
	}

	// Set up progress tracking
	tracker := NewProgressTracker(len(regularFiles))
	e.mu.Lock()
	e.progress = tracker
	e.mu.Unlock()

	progressChan := make(chan ProgressEvent, 100)
	go e.progressLoop(ctx, progressChan, tracker)

	// Create folder structure locally and register in DB
	if err := e.createFolders(ctx, folders); err != nil {
		e.finishRun(runID, "failed", 0, 0, 0, "")
		close(progressChan)
		return fmt.Errorf("create folders: %w", err)
	}

	// Download files in parallel
	synced, failed, bytes := e.downloadFiles(ctx, regularFiles, progressChan)

	close(progressChan)
	tracker.SetRunning(false)

	// Record change token for future incremental syncs
	changeToken, _ := e.drive.GetStartPageToken(ctx)
	log.Printf("[sync] full: folders=%d files=%d synced=%d skipped=%d failed=%d changeToken=%s",
		len(folders), len(regularFiles), synced, len(regularFiles)-synced-failed, failed, changeToken)

	status := "completed"
	if ctx.Err() != nil {
		status = "cancelled"
	} else if failed > 0 && synced == 0 {
		status = "failed"
	}

	e.finishRun(runID, status, synced, failed, bytes, changeToken)
	log.Printf("[sync] full sync done: synced=%d failed=%d bytes=%d", synced, failed, bytes)
	return nil
}

// StartIncremental runs an incremental sync using the Changes API.
func (e *SyncEngine) StartIncremental(ctx context.Context) error {
	token, err := e.store.GetLastChangeToken()
	if err != nil {
		return fmt.Errorf("get change token: %w", err)
	}
	if token == "" {
		return fmt.Errorf("no change token found, run full sync first")
	}

	if err := e.acquireLock(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	e.cancel = cancel
	e.mu.Unlock()

	defer func() {
		cancel()
		e.releaseLock()
	}()

	runID, err := e.store.StartRun()
	if err != nil {
		return fmt.Errorf("start run: %w", err)
	}

	// Fetch changes since last sync
	log.Printf("[sync] incremental: using change token %s", token)
	changes, newToken, err := e.drive.GetChanges(ctx, token)
	if err != nil {
		e.finishRun(runID, "failed", 0, 0, 0, "")
		return fmt.Errorf("get changes: %w", err)
	}
	log.Printf("[sync] incremental: %d changes found, newToken=%s", len(changes), newToken)

	// Separate changed files and removed files
	var changedFiles []*driveapi.File
	var removedIDs []string
	for _, c := range changes {
		if c.Removed || c.File == nil || c.File.Trashed {
			removedIDs = append(removedIDs, c.FileId)
			log.Printf("[sync] incremental: removed/trashed fileId=%s", c.FileId)
		} else {
			changedFiles = append(changedFiles, c.File)
			log.Printf("[sync] incremental: changed file=%s id=%s mime=%s", c.File.Name, c.File.Id, c.File.MimeType)
		}
	}

	// Handle removals
	for _, id := range removedIDs {
		e.handleRemoval(id)
	}

	// Separate folders and files
	var folders, regularFiles []*driveapi.File
	for _, f := range changedFiles {
		if f.MimeType == "application/vnd.google-apps.folder" {
			folders = append(folders, f)
		} else {
			regularFiles = append(regularFiles, f)
		}
	}

	tracker := NewProgressTracker(len(regularFiles))
	e.mu.Lock()
	e.progress = tracker
	e.mu.Unlock()

	progressChan := make(chan ProgressEvent, 100)
	go e.progressLoop(ctx, progressChan, tracker)

	if err := e.createFolders(ctx, folders); err != nil {
		e.finishRun(runID, "failed", 0, 0, 0, "")
		close(progressChan)
		return fmt.Errorf("create folders: %w", err)
	}

	synced, failed, bytes := e.downloadFiles(ctx, regularFiles, progressChan)

	close(progressChan)
	tracker.SetRunning(false)

	status := "completed"
	if ctx.Err() != nil {
		status = "cancelled"
	} else if failed > 0 && synced == 0 {
		status = "failed"
	}

	e.finishRun(runID, status, synced, failed, bytes, newToken)
	log.Printf("[sync] incremental sync done: synced=%d failed=%d bytes=%d", synced, failed, bytes)
	return nil
}

// downloadFiles downloads files in parallel using errgroup.
// Returns (synced, failed, totalBytes).
func (e *SyncEngine) downloadFiles(ctx context.Context, files []*driveapi.File, progressChan chan<- ProgressEvent) (int, int, int64) {
	var (
		synced     int
		failed     int
		totalBytes int64
		counterMu  gosync.Mutex
	)

	workers := e.cfg.MaxWorkers
	if workers <= 0 {
		workers = 3
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(workers)

	for _, file := range files {
		f := file
		g.Go(func() error {
			n, err := e.syncOneFile(gctx, f, progressChan)
			counterMu.Lock()
			if err != nil {
				failed++
			} else if n > 0 {
				synced++
				totalBytes += n
			} else {
				// n == 0 means skipped, already counted in tracker
			}
			counterMu.Unlock()
			return nil // always nil — individual errors don't stop the group
		})
	}

	_ = g.Wait()

	return synced, failed, totalBytes
}

// syncOneFile handles downloading a single file.
// Returns bytes written (0 if skipped) and error for counting purposes.
func (e *SyncEngine) syncOneFile(ctx context.Context, file *driveapi.File, progressChan chan<- ProgressEvent) (int64, error) {
	// Check cancellation
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	// Check if sync is needed
	local, _ := e.store.GetFile(file.Id)
	if !NeedsSync(file, local) {
		progressChan <- ProgressEvent{
			Type:     "file_skip",
			FileID:   file.Id,
			FileName: file.Name,
		}
		return 0, nil
	}

	progressChan <- ProgressEvent{
		Type:     "file_start",
		FileID:   file.Id,
		FileName: file.Name,
	}

	// Determine local path
	destPath := e.localPath(file)
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		progressChan <- ProgressEvent{
			Type:     "file_error",
			FileID:   file.Id,
			FileName: file.Name,
			Error:    err.Error(),
		}
		if isFatalError(err) {
			return 0, err
		}
		return 0, err
	}

	// Download
	written, err := e.drive.DownloadFile(ctx, file.Id, destPath, file.MimeType, file.Size, func(pct float64) {
		progressChan <- ProgressEvent{
			Type:         "file_progress",
			FileID:       file.Id,
			FileName:     file.Name,
			FileProgress: pct,
		}
	})

	if err != nil {
		progressChan <- ProgressEvent{
			Type:     "file_error",
			FileID:   file.Id,
			FileName: file.Name,
			Error:    err.Error(),
		}
		if isFatalError(err) {
			return 0, err
		}
		return 0, err
	}

	// Update DB
	parentID := ""
	if len(file.Parents) > 0 {
		parentID = file.Parents[0]
	}
	_ = e.store.UpsertFile(store.SyncedFile{
		FileID:        file.Id,
		Name:          file.Name,
		MimeType:      file.MimeType,
		MD5Checksum:   file.Md5Checksum,
		Size:          file.Size,
		DriveModified: file.ModifiedTime,
		LocalPath:     destPath,
		LastSynced:    time.Now().UTC().Format(time.RFC3339),
		ParentID:      parentID,
	})

	progressChan <- ProgressEvent{
		Type:            "file_done",
		FileID:          file.Id,
		FileName:        file.Name,
		FileProgress:    1.0,
		BytesDownloaded: written,
	}

	return written, nil
}

// progressLoop reads events from progressChan, applies them to the tracker,
// and broadcasts snapshots via WebSocket at a throttled rate.
func (e *SyncEngine) progressLoop(ctx context.Context, progressChan <-chan ProgressEvent, tracker *ProgressTracker) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	dirty := false

	for {
		select {
		case event, ok := <-progressChan:
			if !ok {
				// Channel closed — final broadcast
				if e.hub != nil {
					e.broadcastSnapshot(tracker)
				}
				return
			}
			tracker.Apply(event)
			dirty = true
		case <-ticker.C:
			if dirty && e.hub != nil {
				e.broadcastSnapshot(tracker)
				dirty = false
			}
		case <-ctx.Done():
			// Drain remaining events
			for event := range progressChan {
				tracker.Apply(event)
			}
			if e.hub != nil {
				e.broadcastSnapshot(tracker)
			}
			return
		}
	}
}

func (e *SyncEngine) broadcastSnapshot(tracker *ProgressTracker) {
	snap := tracker.Snapshot()
	msg := struct {
		Type string           `json:"type"`
		Data ProgressSnapshot `json:"data"`
	}{Type: "sync_progress", Data: snap}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	e.hub.Broadcast(data)
}

// createFolders creates local directories for Drive folders and registers them in the DB.
func (e *SyncEngine) createFolders(ctx context.Context, folders []*driveapi.File) error {
	for _, f := range folders {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		dirPath := e.localPath(f)
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dirPath, err)
		}
		parentID := ""
		if len(f.Parents) > 0 {
			parentID = f.Parents[0]
		}
		_ = e.store.UpsertFile(store.SyncedFile{
			FileID:        f.Id,
			Name:          f.Name,
			MimeType:      f.MimeType,
			DriveModified: f.ModifiedTime,
			LocalPath:     dirPath,
			LastSynced:    time.Now().UTC().Format(time.RFC3339),
			ParentID:      parentID,
			IsFolder:      true,
		})
	}
	return nil
}

// localPath computes the local filesystem path for a Drive file.
// For Google Docs, it appends the export extension.
func (e *SyncEngine) localPath(file *driveapi.File) string {
	name := file.Name
	if exp, ok := e.cfg.ExportFormats[file.MimeType]; ok {
		if !strings.HasSuffix(name, exp.Extension) {
			name += exp.Extension
		}
	}
	// Build path from parent chain using DB lookups
	parts := []string{name}
	parentID := ""
	if len(file.Parents) > 0 {
		parentID = file.Parents[0]
	}
	for parentID != "" && parentID != e.cfg.SyncFolderID {
		parent, err := e.store.GetFile(parentID)
		if err != nil || parent == nil {
			break
		}
		parts = append([]string{parent.Name}, parts...)
		parentID = parent.ParentID
	}
	return filepath.Join(e.cfg.LocalSyncDir, filepath.Join(parts...))
}

// handleRemoval moves a removed file to .gdrive-trash.
func (e *SyncEngine) handleRemoval(fileID string) {
	f, err := e.store.GetFile(fileID)
	if err != nil || f == nil {
		return
	}

	if f.LocalPath != "" {
		trashDir := filepath.Join(e.cfg.LocalSyncDir, ".gdrive-trash")
		_ = os.MkdirAll(trashDir, 0o755)
		trashPath := filepath.Join(trashDir, filepath.Base(f.LocalPath))
		_ = os.Rename(f.LocalPath, trashPath)
	}

	_ = e.store.DeleteFile(fileID)
}

func (e *SyncEngine) acquireLock() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		return errors.New("sync already running")
	}
	e.running = true
	return nil
}

func (e *SyncEngine) releaseLock() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.running = false
	e.cancel = nil
}

func (e *SyncEngine) finishRun(id int64, status string, synced, failed int, bytes int64, token string) {
	if err := e.store.FinishRun(id, status, synced, failed, bytes, token); err != nil {
		log.Printf("[sync] failed to finish run: %v", err)
	}
}

// DiffEntry represents a file that would be synced in a dry run.
type DiffEntry struct {
	FileID        string `json:"file_id"`
	Name          string `json:"name"`
	MimeType      string `json:"mime_type"`
	Size          int64  `json:"size"`
	DriveModified string `json:"drive_modified"`
	LocalPath     string `json:"local_path"`
	Action        string `json:"action"` // "new" | "update" | "delete"
}

// DryRun lists files that would be synced without downloading.
func (e *SyncEngine) DryRun(ctx context.Context) ([]DiffEntry, error) {
	if e.drive == nil {
		return nil, fmt.Errorf("Drive client not initialized")
	}

	files, err := e.drive.ListAllRecursive(ctx, e.cfg.SyncFolderID)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	var entries []DiffEntry
	for _, f := range files {
		if f.MimeType == "application/vnd.google-apps.folder" {
			continue
		}

		local, _ := e.store.GetFile(f.Id)
		if !NeedsSync(f, local) {
			continue
		}

		action := "update"
		if local == nil {
			action = "new"
		}

		entries = append(entries, DiffEntry{
			FileID:        f.Id,
			Name:          f.Name,
			MimeType:      f.MimeType,
			Size:          f.Size,
			DriveModified: f.ModifiedTime,
			LocalPath:     e.localPath(f),
			Action:        action,
		})
	}

	return entries, nil
}

// isFatalError returns true for errors that should stop the entire sync.
func isFatalError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if os.IsPermission(err) {
		return true
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		if errno == syscall.ENOSPC {
			return true
		}
	}
	return false
}

