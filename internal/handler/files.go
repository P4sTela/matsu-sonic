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
