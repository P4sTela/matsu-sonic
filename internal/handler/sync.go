package handler

import (
	"net/http"
	"strconv"
)

// StartFullSync starts a full sync in the background.
func (h *Handler) StartFullSync(w http.ResponseWriter, r *http.Request) {
	if h.Engine.IsRunning() {
		writeError(w, http.StatusConflict, "sync already running")
		return
	}

	go h.Engine.StartFull(r.Context())

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "started",
		"mode":   "full",
	})
}

// StartIncrementalSync starts an incremental sync in the background.
func (h *Handler) StartIncrementalSync(w http.ResponseWriter, r *http.Request) {
	if h.Engine.IsRunning() {
		writeError(w, http.StatusConflict, "sync already running")
		return
	}

	go h.Engine.StartIncremental(r.Context())

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "started",
		"mode":   "incremental",
	})
}

// CancelSync cancels the running sync.
func (h *Handler) CancelSync(w http.ResponseWriter, r *http.Request) {
	h.Engine.Cancel()
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancel_requested"})
}

// GetSyncStatus returns current sync status and progress.
func (h *Handler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"is_running": h.Engine.IsRunning(),
		"progress":   h.Engine.Status(),
	})
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
