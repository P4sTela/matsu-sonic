package handler

import (
	"net/http"

	cfgpkg "github.com/P4sTela/matsu-sonic/internal/config"
)

// GetConfig returns the current configuration (passwords masked).
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := *h.Config
	// Mask sensitive fields
	for i := range cfg.DistTargets {
		if cfg.DistTargets[i].Password != "" {
			cfg.DistTargets[i].Password = "********"
		}
	}
	writeJSON(w, http.StatusOK, cfg)
}

// UpdateConfig partially updates and saves the configuration.
func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var update map[string]any
	if err := decodeJSON(r, &update); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := cfgpkg.Save(h.ConfigPath, *h.Config); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}
