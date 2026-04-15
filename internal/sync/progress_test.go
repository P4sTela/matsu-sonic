package sync

import (
	"testing"
)

func TestProgressTracker_Apply(t *testing.T) {
	tracker := NewProgressTracker(5)

	tracker.Apply(ProgressEvent{Type: "file_start", FileName: "a.pdf"})
	snap := tracker.Snapshot()
	if snap.CurrentFile != "a.pdf" {
		t.Errorf("CurrentFile = %q, want %q", snap.CurrentFile, "a.pdf")
	}

	tracker.Apply(ProgressEvent{Type: "file_progress", FileName: "a.pdf", FileProgress: 0.5})
	snap = tracker.Snapshot()
	if snap.CurrentFileProgress != 0.5 {
		t.Errorf("CurrentFileProgress = %f, want 0.5", snap.CurrentFileProgress)
	}
	if snap.BytesDownloaded != 0 {
		t.Errorf("BytesDownloaded = %d, want 0", snap.BytesDownloaded)
	}

	tracker.Apply(ProgressEvent{Type: "file_done", FileName: "a.pdf", BytesDownloaded: 500})
	snap = tracker.Snapshot()
	if snap.CompletedFiles != 1 {
		t.Errorf("CompletedFiles = %d, want 1", snap.CompletedFiles)
	}
	if snap.BytesDownloaded != 500 {
		t.Errorf("BytesDownloaded = %d, want 500", snap.BytesDownloaded)
	}
}

func TestProgressTracker_Skip(t *testing.T) {
	tracker := NewProgressTracker(3)
	tracker.Apply(ProgressEvent{Type: "file_skip", FileName: "b.pdf"})
	tracker.Apply(ProgressEvent{Type: "file_skip", FileName: "c.pdf"})

	snap := tracker.Snapshot()
	if snap.SkippedFiles != 2 {
		t.Errorf("SkippedFiles = %d, want 2", snap.SkippedFiles)
	}
}

func TestProgressTracker_Error(t *testing.T) {
	tracker := NewProgressTracker(2)
	tracker.Apply(ProgressEvent{Type: "file_error", FileName: "bad.pdf", Error: "network timeout"})

	snap := tracker.Snapshot()
	if snap.FailedFiles != 1 {
		t.Errorf("FailedFiles = %d, want 1", snap.FailedFiles)
	}
	if len(snap.Errors) != 1 || snap.Errors[0] != "network timeout" {
		t.Errorf("Errors = %v, want [network timeout]", snap.Errors)
	}
}

func TestProgressTracker_MaxErrors(t *testing.T) {
	tracker := NewProgressTracker(30)
	for i := 0; i < 25; i++ {
		tracker.Apply(ProgressEvent{Type: "file_error", Error: "err"})
	}

	snap := tracker.Snapshot()
	if len(snap.Errors) != maxErrors {
		t.Errorf("Errors len = %d, want %d", len(snap.Errors), maxErrors)
	}
}

func TestProgressTracker_SetRunning(t *testing.T) {
	tracker := NewProgressTracker(1)
	if !tracker.Snapshot().IsRunning {
		t.Error("new tracker should be running")
	}

	tracker.SetRunning(false)
	if tracker.Snapshot().IsRunning {
		t.Error("should not be running after SetRunning(false)")
	}
}

func TestProgressTracker_SnapshotCopy(t *testing.T) {
	tracker := NewProgressTracker(1)
	tracker.Apply(ProgressEvent{Type: "file_error", Error: "err1"})

	snap := tracker.Snapshot()
	snap.Errors[0] = "modified"

	snap2 := tracker.Snapshot()
	if snap2.Errors[0] != "err1" {
		t.Error("Snapshot should return a copy, not a reference")
	}
}
