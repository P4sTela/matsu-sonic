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
		home, _ := os.UserHomeDir()
		path = home
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

	writeJSON(w, http.StatusOK, map[string]any{
		"current": path,
		"parent":  filepath.Dir(path),
		"items":   items,
	})
}
