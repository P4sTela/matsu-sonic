package handler

import (
	"net/http"
	"os"
	"path/filepath"
)

// BrowseDirectory lists local directory contents for the directory picker.
func (h *Handler) BrowseDirectory(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path, _ = os.Getwd()
	}

	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		writeError(w, http.StatusBadRequest, "absolute path required")
		return
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	type item struct {
		Name  string `json:"name"`
		Path  string `json:"path"`
		IsDir bool   `json:"is_dir"`
	}

	var items []item
	for _, e := range entries {
		// Skip hidden files
		if len(e.Name()) > 0 && e.Name()[0] == '.' {
			continue
		}
		items = append(items, item{
			Name:  e.Name(),
			Path:  filepath.Join(path, e.Name()),
			IsDir: e.IsDir(),
		})
	}

	cwd, _ := os.Getwd()

	writeJSON(w, http.StatusOK, map[string]any{
		"current": path,
		"parent":  filepath.Dir(path),
		"cwd":     cwd,
		"items":   items,
	})
}

// MakeDirectory creates a new directory.
func (h *Handler) MakeDirectory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := decodeJSON(r, &req); err != nil || req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	req.Path = filepath.Clean(req.Path)
	if !filepath.IsAbs(req.Path) {
		writeError(w, http.StatusBadRequest, "absolute path required")
		return
	}

	if err := os.MkdirAll(req.Path, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "created", "path": req.Path})
}
