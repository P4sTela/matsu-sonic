package handler

import (
	"context"
	"net/http"
	"strconv"
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
