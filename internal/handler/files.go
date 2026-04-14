package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ListFiles returns all synced files, optionally filtered by search.
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	files, err := h.Store.ListFiles(search)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, files)
}

// GetFile returns a single synced file by ID.
func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "fileID")
	f, err := h.Store.GetFile(fileID)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	writeJSON(w, http.StatusOK, f)
}

// DeleteFiles removes selected file records from the database.
func (h *Handler) DeleteFiles(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileIDs []string `json:"file_ids"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	deleted := 0
	for _, id := range req.FileIDs {
		if err := h.Store.DeleteFile(id); err == nil {
			deleted++
		}
	}

	// Clear change tokens so the next incremental sync re-discovers deleted files via full sync.
	if deleted > 0 {
		_ = h.Store.ClearChangeTokens()
	}

	writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
}
