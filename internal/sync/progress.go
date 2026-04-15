package sync

import "sync"

// ProgressEvent represents a single sync progress event.
type ProgressEvent struct {
	Type            string  `json:"type"`              // "file_start" | "file_progress" | "file_done" | "file_skip" | "file_error" | "scan"
	FileID          string  `json:"file_id"`
	FileName        string  `json:"file_name"`
	FileProgress    float64 `json:"file_progress"`     // 0.0 ~ 1.0
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
		t.activeDownloads[e.FileID] = ActiveDownload{
			FileID: e.FileID, FileName: e.FileName, Progress: 0,
		}
	case "file_progress":
		t.currentFile = e.FileName
		t.currentProgress = e.FileProgress
		t.activeDownloads[e.FileID] = ActiveDownload{
			FileID: e.FileID, FileName: e.FileName, Progress: e.FileProgress,
		}
	case "file_done":
		t.completedFiles++
		t.currentProgress = 1.0
		t.bytesDownloaded += e.BytesDownloaded
		delete(t.activeDownloads, e.FileID)
	case "file_skip":
		t.skippedFiles++
	case "file_error":
		t.failedFiles++
		delete(t.activeDownloads, e.FileID)
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

	active := make([]ActiveDownload, 0, len(t.activeDownloads))
	for _, d := range t.activeDownloads {
		active = append(active, d)
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

// SetRunning updates the running state.
func (t *ProgressTracker) SetRunning(running bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.isRunning = running
}
