package handler

import (
	"log"
	"net/http"

	"github.com/P4sTela/matsu-sonic/internal/drive"
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

	writeJSON(w, http.StatusOK, AuthTestResponse{
		Status: "ok",
		User: AuthUser{
			DisplayName:  about.User.DisplayName,
			EmailAddress: about.User.EmailAddress,
		},
	})
}

// GetAuthStatus reports whether the Drive client is authenticated and whether
// an interactive authorization is currently awaiting user approval.
func (h *Handler) GetAuthStatus(w http.ResponseWriter, r *http.Request) {
	h.authMu.Lock()
	pending := h.authFlow != nil
	h.authMu.Unlock()

	drv := h.GetDrive()
	writeJSON(w, http.StatusOK, AuthStatusResponse{
		Authenticated: drv != nil && drv.Service != nil,
		Pending:       pending,
		AuthMethod:    h.Config.AuthMethod,
	})
}

// StartAuth begins an interactive OAuth flow and returns the authorization URL
// for the user to open. The exchange completes asynchronously in the browser
// callback, after which the Drive client is reinitialized.
func (h *Handler) StartAuth(w http.ResponseWriter, r *http.Request) {
	h.authMu.Lock()
	defer h.authMu.Unlock()

	if h.authFlow != nil {
		writeJSON(w, http.StatusOK, AuthStartResponse{AuthURL: h.authFlow.AuthURL})
		return
	}

	flow, err := drive.BeginOAuth(h.Config)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.authFlow = flow

	go func() {
		err := <-flow.Done
		if err != nil {
			log.Printf("[auth] interactive flow failed: %v", err)
		} else {
			log.Println("[auth] interactive flow completed; reinitializing Drive")
			h.ReinitDrive()
		}
		h.authMu.Lock()
		h.authFlow = nil
		h.authMu.Unlock()
	}()

	writeJSON(w, http.StatusOK, AuthStartResponse{AuthURL: flow.AuthURL})
}
