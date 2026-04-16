package handler

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/P4sTela/matsu-sonic/internal/store"
)

// ListRevisions lists revisions for a file from the Drive API.
func (h *Handler) ListRevisions(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "fileID")

	drv := h.GetDrive()
	if drv == nil {
		writeError(w, http.StatusServiceUnavailable, "Drive client not configured — set credentials first")
		return
	}

	revisions, err := drv.ListRevisions(r.Context(), fileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, revisions)
}

// DownloadRevision downloads a specific revision of a file.
func (h *Handler) DownloadRevision(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "fileID")
	revID := chi.URLParam(r, "revID")

	var req RevisionDownloadRequest
	decodeJSON(r, &req)

	// Determine destination path
	destDir := req.DestDir
	if destDir == "" {
		destDir = h.Config.LocalSyncDir
	}

	// Build filename using revision naming pattern
	f, err := h.Store.GetFile(fileID)
	baseName := fileID
	if err == nil && f != nil {
		baseName = f.Name
	}

	naming := h.Config.RevisionNaming
	stem := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	suffix := filepath.Ext(baseName)
	name := strings.ReplaceAll(naming, "{stem}", stem)
	name = strings.ReplaceAll(name, "{rev_id}", revID)
	name = strings.ReplaceAll(name, "{suffix}", suffix)

	destPath := filepath.Join(destDir, name)

	drv := h.GetDrive()
	if drv == nil {
		writeError(w, http.StatusServiceUnavailable, "Drive client not configured — set credentials first")
		return
	}

	size, err := drv.DownloadRevision(r.Context(), fileID, revID, destPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Record in DB
	h.Store.InsertRevision(store.DownloadedRevision{
		FileID:     fileID,
		RevisionID: revID,
		LocalPath:  destPath,
		Size:       size,
	})

	writeJSON(w, http.StatusOK, RevisionDownloadResponse{
		Path: destPath,
		Size: size,
	})
}

// ListDownloadedRevisions returns locally downloaded revisions from the DB.
func (h *Handler) ListDownloadedRevisions(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "fileID")

	revisions, err := h.Store.ListDownloadedRevisions(fileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if revisions == nil {
		revisions = []store.DownloadedRevision{}
	}

	writeJSON(w, http.StatusOK, revisions)
}

// RegisterRoutes registers all handler routes on the given Chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api", func(r chi.Router) {
		r.Get("/config", h.GetConfig)
		r.Post("/config", h.UpdateConfig)

		r.Post("/auth/test", h.TestAuth)

		r.Post("/sync/full", h.StartFullSync)
		r.Post("/sync/incremental", h.StartIncrementalSync)
		r.Post("/sync/cancel", h.CancelSync)
		r.Get("/sync/status", h.GetSyncStatus)
		r.Get("/sync/diff", h.GetSyncDiff)
		r.Post("/sync/reset", h.ResetSync)
		r.Get("/sync/history", h.GetSyncHistory)

		r.Get("/files", h.ListFiles)
		r.Post("/files/delete", h.DeleteFiles)
		r.Post("/files/verify", h.VerifyFiles)
		r.Post("/files/resync", h.ResyncFiles)
		r.Get("/files/{fileID}", h.GetFile)
		r.Get("/files/{fileID}/revisions", h.ListRevisions)
		r.Post("/files/{fileID}/revisions/{revID}/download", h.DownloadRevision)
		r.Get("/files/{fileID}/revisions/downloaded", h.ListDownloadedRevisions)

		r.Get("/distribution/targets", h.ListTargets)
		r.Post("/distribution/targets", h.AddTarget)
		r.Delete("/distribution/targets/{name}", h.RemoveTarget)
		r.Post("/distribution/targets/{name}/test", h.TestTarget)
		r.Post("/distribute", h.Distribute)
		r.Get("/distribution/jobs", h.ListDistJobs)

		r.Get("/browse", h.BrowseDirectory)
		r.Post("/mkdir", h.MakeDirectory)
		r.Get("/drive/browse", h.BrowseDrive)
	})
}
