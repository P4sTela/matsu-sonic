package handler

import (
	"net/http"
)

// TestAuth tests the Drive API authentication.
func (h *Handler) TestAuth(w http.ResponseWriter, r *http.Request) {
	drv := h.GetDrive()
	if drv == nil || drv.Service == nil {
		writeError(w, http.StatusServiceUnavailable, "Drive client not configured — set credentials first")
		return
	}

	about, err := drv.Service.About.Get().Fields("user(displayName, emailAddress)").Context(r.Context()).Do()
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"user": map[string]string{
			"displayName":  about.User.DisplayName,
			"emailAddress": about.User.EmailAddress,
		},
	})
}
