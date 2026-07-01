package sync

import (
	"slices"
	"sync"
)

// ProgressEvent represents a single sync progress event.
type ProgressEvent struct {
	Type            string  `json:"type"` // "file_start" | "file_progress" | "file_done" | "file_skip" | "file_error" | "scan"
	FileID          string  `json:"file_id"`
	FileName        string  `json:"file_name"`
	FileProgress    float64 `json:"file_progress"` // 0.0 ~ 1.0
	BytesDownloaded int64   `json:"bytes_downloaded"`
	Error           string  `json:"error,omitempty"`
}

// ActiveDownload represents a single file being downloaded by a worker.
type ActiveDownload struct {
	FileID   string  `json:"file_id"`
	FileName string  `json:"file_name"`
	Progress float64 `json:"progress"` // 0.0 ~ 1.0
}

// ProgressSnapshot is a thread-safe copy of the current sync state.
type ProgressSnapshot struct {
	TotalFiles          int              `json:"total_files"`
	CompletedFiles      int              `json:"completed_files"`
	FailedFiles         int              `json:"failed_files"`
	SkippedFiles        int              `json:"skipped_files"`
	BytesDownloaded     int64            `json:"bytes_downloaded"`
	CurrentFile         string           `json:"current_file"`
	CurrentFileProgress float64          `json:"current_file_progress"`
	ActiveDownloads     []ActiveDownload `json:"active_downloads"`
	IsRunning           bool             `json:"is_running"`
	Errors              []string         `json:"errors"`
}

const maxErrors = 20

// ProgressTracker aggregates progress events into a snapshot.
type ProgressTracker struct {
	mu              sync.Mutex
	totalFiles      int
	completedFiles  int
	failedFiles     int
	skippedFiles    int
	bytesDownloaded int64
	currentFile     string
	currentProgress float64
	activeDownloads map[string]ActiveDownload
	activeOrder     []string // insertion-ordered list of file IDs for stable UI rendering
	isRunning       bool
	errors          []string
}

// NewProgressTracker creates a tracker with the given total file count.
func NewProgressTracker(totalFiles int) *ProgressTracker {
	return &ProgressTracker{
		totalFiles:      totalFiles,
		isRunning:       true,
		activeDownloads: make(map[string]ActiveDownload),
	}
}

// Apply processes a progress event and updates internal state.
func (t *ProgressTracker) Apply(e ProgressEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch e.Type {
	case "file_start":
		t.currentFile = e.FileName
		t.currentProgress = 0
		if _, exists := t.activeDownloads[e.FileID]; !exists {
			t.activeOrder = append(t.activeOrder, e.FileID)
		}
		t.activeDownloads[e.FileID] = ActiveDownload{
			FileID: e.FileID, FileName: e.FileName, Progress: 0,
		}
	case "file_progress":
		t.currentFile = e.FileName
		t.currentProgress = e.FileProgress
		if _, exists := t.activeDownloads[e.FileID]; !exists {
			t.activeOrder = append(t.activeOrder, e.FileID)
		}
		t.activeDownloads[e.FileID] = ActiveDownload{
			FileID: e.FileID, FileName: e.FileName, Progress: e.FileProgress,
		}
	case "file_done":
		t.completedFiles++
		t.currentProgress = 1.0
		t.bytesDownloaded += e.BytesDownloaded
		delete(t.activeDownloads, e.FileID)
		t.removeActiveOrder(e.FileID)
	case "file_skip":
		t.skippedFiles++
	case "file_error":
		t.failedFiles++
		delete(t.activeDownloads, e.FileID)
		t.removeActiveOrder(e.FileID)
		if len(t.errors) < maxErrors {
			t.errors = append(t.errors, e.Error)
		}
	case "scan":
		t.totalFiles = int(e.BytesDownloaded)
	}
}

// Snapshot returns a copy of the current progress state.
func (t *ProgressTracker) Snapshot() ProgressSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()

	errs := make([]string, len(t.errors))
	copy(errs, t.errors)

	active := make([]ActiveDownload, 0, len(t.activeOrder))
	for _, id := range t.activeOrder {
		if d, ok := t.activeDownloads[id]; ok {
			active = append(active, d)
		}
	}

	return ProgressSnapshot{
		TotalFiles:          t.totalFiles,
		CompletedFiles:      t.completedFiles,
		FailedFiles:         t.failedFiles,
		SkippedFiles:        t.skippedFiles,
		BytesDownloaded:     t.bytesDownloaded,
		CurrentFile:         t.currentFile,
		CurrentFileProgress: t.currentProgress,
		ActiveDownloads:     active,
		IsRunning:           t.isRunning,
		Errors:              errs,
	}
}

// removeActiveOrder removes the given file ID from the activeOrder slice.
func (t *ProgressTracker) removeActiveOrder(id string) {
	for i, v := range t.activeOrder {
		if v == id {
			t.activeOrder = slices.Delete(t.activeOrder, i, i+1)
			return
		}
	}
}

// SetRunning updates the running state.
func (t *ProgressTracker) SetRunning(running bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.isRunning = running
}
