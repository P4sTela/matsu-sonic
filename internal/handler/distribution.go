package handler

import (
	"net/http"
	"strconv"

	cfgpkg "github.com/P4sTela/matsu-sonic/internal/config"
	"github.com/P4sTela/matsu-sonic/internal/distribution"
	"github.com/P4sTela/matsu-sonic/internal/store"
	"github.com/go-chi/chi/v5"
)

// ListTargets returns all configured distribution targets.
func (h *Handler) ListTargets(w http.ResponseWriter, r *http.Request) {
	targets := h.Config.DistTargets
	if targets == nil {
		targets = []cfgpkg.DistTargetConf{}
	}
	// Mask passwords before returning
	masked := make([]cfgpkg.DistTargetConf, len(targets))
	copy(masked, targets)
	for i := range masked {
		if masked[i].Password != "" {
			masked[i].Password = "********"
		}
	}
	writeJSON(w, http.StatusOK, masked)
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
	if h.DistManager != nil {
		h.DistManager.Reload(h.Config.DistTargets)
	}

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
	if h.DistManager != nil {
		h.DistManager.Reload(h.Config.DistTargets)
	}

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

	if h.DistManager == nil {
		writeError(w, http.StatusInternalServerError, "distribution manager not initialized")
		return
	}

	t, err := h.DistManager.Get(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	if err := t.TestConnection(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

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

	if h.DistManager == nil {
		writeError(w, http.StatusInternalServerError, "distribution manager not initialized")
		return
	}

	type result struct {
		FileID string `json:"file_id"`
		Status string `json:"status"`
		Path   string `json:"path,omitempty"`
		Error  string `json:"error,omitempty"`
	}

	// Collect valid files for batch distribution.
	type fileEntry struct {
		fileID    string
		localPath string
	}
	var files []distribution.FileCopy
	var entries []fileEntry
	var results []result

	for _, fileID := range req.FileIDs {
		f, err := h.Store.GetFile(fileID)
		if err != nil {
			results = append(results, result{FileID: fileID, Status: "failed", Error: "file not found"})
			continue
		}

		destRelative := f.Name
		if req.DestDir != "" {
			destRelative = req.DestDir + "/" + f.Name
		}

		files = append(files, distribution.FileCopy{Src: f.LocalPath, DestRelative: destRelative})
		entries = append(entries, fileEntry{fileID: fileID, localPath: f.LocalPath})
	}

	if len(files) > 0 {
		batchResults, err := h.DistManager.DistributeBatch(r.Context(), req.TargetName, files)
		if err != nil {
			// Target-level error (e.g. target not found): fail all files.
			for _, e := range entries {
				results = append(results, result{FileID: e.fileID, Status: "failed", Error: err.Error()})
				h.Store.InsertDistJob(store.DistJob{
					FileID:       e.fileID,
					SourcePath:   e.localPath,
					TargetType:   req.TargetName,
					TargetPath:   "",
					Status:       "failed",
					ErrorMessage: err.Error(),
				})
			}
		} else {
			for i, br := range batchResults {
				e := entries[i]
				if br.Err != nil {
					results = append(results, result{FileID: e.fileID, Status: "failed", Error: br.Err.Error()})
					h.Store.InsertDistJob(store.DistJob{
						FileID:       e.fileID,
						SourcePath:   e.localPath,
						TargetType:   req.TargetName,
						TargetPath:   files[i].DestRelative,
						Status:       "failed",
						ErrorMessage: br.Err.Error(),
					})
				} else {
					results = append(results, result{FileID: e.fileID, Status: "completed", Path: br.DestPath})
					h.Store.InsertDistJob(store.DistJob{
						FileID:     e.fileID,
						SourcePath: e.localPath,
						TargetType: req.TargetName,
						TargetPath: br.DestPath,
						Status:     "completed",
					})
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, results)
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
