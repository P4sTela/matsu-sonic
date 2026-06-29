package handler

import (
	"context"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	msync "github.com/P4sTela/matsu-sonic/internal/sync"
)

// StartFullSync starts a full sync in the background.
func (h *Handler) StartFullSync(w http.ResponseWriter, r *http.Request) {
	if h.Engine.IsRunning() {
		writeError(w, http.StatusConflict, "sync already running")
		return
	}

	go h.Engine.StartFull(context.Background())

	writeJSON(w, http.StatusOK, SyncStartResponse{Status: "started", Mode: "full"})
}

// StartIncrementalSync starts an incremental sync in the background.
// Returns an error if no change token exists (full sync required first).
func (h *Handler) StartIncrementalSync(w http.ResponseWriter, r *http.Request) {
	if h.Engine.IsRunning() {
		writeError(w, http.StatusConflict, "sync already running")
		return
	}

	token, err := h.Store.GetLastChangeToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if token == "" {
		writeError(w, http.StatusBadRequest, "no change token found, run full sync first")
		return
	}

	go h.Engine.StartIncremental(context.Background())

	writeJSON(w, http.StatusOK, SyncStartResponse{Status: "started", Mode: "incremental"})
}

// CancelSync cancels the running sync.
func (h *Handler) CancelSync(w http.ResponseWriter, r *http.Request) {
	h.Engine.Cancel()
	writeJSON(w, http.StatusOK, StatusResponse{Status: "cancel_requested"})
}

// GetSyncStatus returns current sync status and progress.
func (h *Handler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, SyncStatusResponse{
		IsRunning: h.Engine.IsRunning(),
		Progress:  h.Engine.Status(),
	})
}

// ResetSync clears all sync data (DB records). Does not delete local files.
func (h *Handler) ResetSync(w http.ResponseWriter, r *http.Request) {
	if h.Engine.IsRunning() {
		writeError(w, http.StatusConflict, "sync is running")
		return
	}

	if err := h.Store.ClearAll(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, StatusResponse{Status: "reset"})
}

// GetSyncDiff returns a dry-run diff of files that would be synced.
func (h *Handler) GetSyncDiff(w http.ResponseWriter, r *http.Request) {
	entries, err := h.Engine.DryRun(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// PreviewSelect reports how many already-synced files match the given select
// patterns, without contacting Drive. It is a cheap preview against the local
// database to help validate patterns before saving.
func (h *Handler) PreviewSelect(w http.ResponseWriter, r *http.Request) {
	var req SyncPreviewRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	files, err := h.Store.ListFiles("")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	base := filepath.Clean(h.Config.LocalSyncDir)
	resp := SyncPreviewResponse{Samples: []string{}}
	for _, f := range files {
		if f.IsFolder {
			continue
		}
		resp.Total++
		rel := relativeSyncPath(base, f.LocalPath, f.Name)
		if msync.IsSelectedBy(req.Patterns, rel) {
			resp.Matched++
			if len(resp.Samples) < 10 {
				resp.Samples = append(resp.Samples, rel)
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// relativeSyncPath returns the forward-slash path of a synced file relative to
// the sync root, falling back to the file name when no usable path is stored.
func relativeSyncPath(base, localPath, name string) string {
	if localPath == "" {
		return name
	}
	rel := localPath
	if base != "" && base != "." {
		if r, err := filepath.Rel(base, localPath); err == nil && !strings.HasPrefix(r, "..") {
			rel = r
		}
	}
	return filepath.ToSlash(rel)
}

// GetSyncHistory returns recent sync runs.
func (h *Handler) GetSyncHistory(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	runs, err := h.Store.ListRuns(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, runs)
}
