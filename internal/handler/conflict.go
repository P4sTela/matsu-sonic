package handler

import (
	"net/http"

	"github.com/P4sTela/matsu-sonic/internal/sync"
)

// GetConflicts scans all synced files and returns any whose local state
// differs from the baseline recorded at the time of the last successful sync.
func (h *Handler) GetConflicts(w http.ResponseWriter, r *http.Request) {
	conflicts, err := sync.DetectConflicts(h.Store)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if conflicts == nil {
		conflicts = []sync.Conflict{}
	}
	writeJSON(w, http.StatusOK, conflicts)
}
