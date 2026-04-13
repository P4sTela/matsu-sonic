package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	cfgpkg "github.com/P4sTela/matsu-sonic/internal/config"
)

// ListTargets returns all configured distribution targets.
func (h *Handler) ListTargets(w http.ResponseWriter, r *http.Request) {
	targets := h.Config.DistTargets
	if targets == nil {
		targets = []cfgpkg.DistTargetConf{}
	}
	writeJSON(w, http.StatusOK, targets)
}

// AddTarget adds a new distribution target.
func (h *Handler) AddTarget(w http.ResponseWriter, r *http.Request) {
	var target cfgpkg.DistTargetConf
	if err := decodeJSON(r, &target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if target.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Check for duplicates
	for _, t := range h.Config.DistTargets {
		if t.Name == target.Name {
			writeError(w, http.StatusConflict, "target already exists")
			return
		}
	}

	h.Config.DistTargets = append(h.Config.DistTargets, target)
	cfgpkg.Save(h.ConfigPath, *h.Config)

	writeJSON(w, http.StatusOK, map[string]string{"status": "added"})
}

// RemoveTarget removes a distribution target by name.
func (h *Handler) RemoveTarget(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	found := false
	var filtered []cfgpkg.DistTargetConf
	for _, t := range h.Config.DistTargets {
		if t.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, t)
	}

	if !found {
		writeError(w, http.StatusNotFound, "target not found")
		return
	}

	h.Config.DistTargets = filtered
	cfgpkg.Save(h.ConfigPath, *h.Config)

	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// TestTarget tests connectivity to a distribution target.
func (h *Handler) TestTarget(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var target *cfgpkg.DistTargetConf
	for i := range h.Config.DistTargets {
		if h.Config.DistTargets[i].Name == name {
			target = &h.Config.DistTargets[i]
			break
		}
	}

	if target == nil {
		writeError(w, http.StatusNotFound, "target not found")
		return
	}

	// TODO: actual target test via distribution package
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Distribute distributes files to a target.
func (h *Handler) Distribute(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileIDs    []string `json:"file_ids"`
		TargetName string   `json:"target_name"`
		DestDir    string   `json:"dest_dir"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// TODO: actual distribution via distribution package
	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

// ListDistJobs returns recent distribution jobs.
func (h *Handler) ListDistJobs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	jobs, err := h.Store.ListDistJobs(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, jobs)
}
