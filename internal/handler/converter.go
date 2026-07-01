package handler

import (
	"net/http"
	"strconv"

	"github.com/P4sTela/matsu-sonic/internal/store"
	"github.com/go-chi/chi/v5"
)

// ListConverters returns enabled converter configurations.
func (h *Handler) ListConverters(w http.ResponseWriter, r *http.Request) {
	var result []map[string]any
	for _, c := range h.Config.Converters {
		result = append(result, map[string]any{
			"name":             c.Name,
			"enabled":          c.Enabled,
			"input_pattern":    c.InputPattern,
			"output_extension": c.OutputExtension,
			"output_dir":       c.OutputDir,
			"auto_convert":     c.AutoConvert,
		})
	}
	if result == nil {
		result = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, result)
}

// ConvertFile starts a conversion for a file using the specified converter.
func (h *Handler) ConvertFile(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "fileID")
	var req struct {
		Converter string `json:"converter"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Converter == "" {
		writeError(w, http.StatusBadRequest, "converter is required")
		return
	}

	conv, err := h.ConvManager.Run(fileID, req.Converter)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "started",
		"job_id": conv.ID,
	})
}

// ReconvertFile deletes the existing conversion record and starts a new conversion.
func (h *Handler) ReconvertFile(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "fileID")
	var req struct {
		Converter string `json:"converter"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Converter == "" {
		writeError(w, http.StatusBadRequest, "converter is required")
		return
	}

	// Delete existing conversion record
	existing, err := h.Store.GetConversion(fileID, req.Converter)
	if err == nil && existing != nil {
		_ = h.Store.DeleteConversion(existing.ID)
	}

	conv, err := h.ConvManager.Run(fileID, req.Converter)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "started",
		"job_id": conv.ID,
	})
}

// ListFileConversions returns conversion records for a given file.
func (h *Handler) ListFileConversions(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "fileID")
	convs, err := h.Store.ListConversionsByFile(fileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if convs == nil {
		convs = []store.Conversion{}
	}
	writeJSON(w, http.StatusOK, convs)
}

// ListConversions returns all recent conversion records.
func (h *Handler) ListConversions(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	convs, err := h.Store.ListConversions(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if convs == nil {
		convs = []store.Conversion{}
	}
	writeJSON(w, http.StatusOK, convs)
}

// ListStaleConversions returns completed conversions whose original files have changed.
func (h *Handler) ListStaleConversions(w http.ResponseWriter, r *http.Request) {
	convs, err := h.Store.ListStaleConversions()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Filter to only truly stale ones
	var stale []store.Conversion
	for _, c := range convs {
		// Check original file state
		sf, err := h.Store.GetFile(c.FileID)
		if err != nil || sf == nil {
			continue
		}
		// Check if original has changed
		if sf.LocalSize != c.OriginalSize || sf.LocalModified != c.OriginalModified {
			stale = append(stale, c)
		}
	}
	if stale == nil {
		stale = []store.Conversion{}
	}
	writeJSON(w, http.StatusOK, stale)
}

// DeleteConversion removes a conversion record.
func (h *Handler) DeleteConversion(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.Store.DeleteConversion(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, StatusResponse{Status: "deleted"})
}
