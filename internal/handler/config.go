package handler

import (
	"encoding/json"
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
// It marshals the current config to JSON, overlays the request body,
// then unmarshals back — achieving a partial merge.
func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	// Start from current config as JSON
	base, err := json.Marshal(h.Config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Unmarshal into a map, then overlay the request body
	var merged map[string]any
	json.Unmarshal(base, &merged)

	var update map[string]any
	if err := decodeJSON(r, &update); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	for k, v := range update {
		merged[k] = v
	}

	// Marshal back and decode into Config struct
	data, err := json.Marshal(merged)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var cfg cfgpkg.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid config values")
		return
	}

	// Apply and save
	*h.Config = cfg
	if err := cfgpkg.Save(h.ConfigPath, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Reinitialize Drive client if credentials changed
	go h.ReinitDrive()

	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}
