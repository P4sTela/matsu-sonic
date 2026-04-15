package handler

import (
	"fmt"
	"net/http"

	driveapi "google.golang.org/api/drive/v3"
)

// BrowseDrive lists folders in a Google Drive folder for the folder picker.
// Query params:
//   - folder_id: folder to browse (default: "root")
//   - source: "my_drive" (default) or "shared" (shared with me)
func (h *Handler) BrowseDrive(w http.ResponseWriter, r *http.Request) {
	drv := h.GetDrive()
	if drv == nil || drv.Service == nil {
		writeError(w, http.StatusServiceUnavailable, "Drive client not configured — set credentials first")
		return
	}

	folderID := r.URL.Query().Get("folder_id")
	source := r.URL.Query().Get("source")

	var items []DriveBrowseItem
	folderName := "My Drive"
	parentID := ""

	if source == "shared" && folderID == "" {
		// List top-level shared items
		folderName = "Shared with me"
		files, err := h.listSharedItems(r)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, f := range files {
			items = append(items, DriveBrowseItem{
				ID:       f.Id,
				Name:     f.Name,
				IsFolder: f.MimeType == "application/vnd.google-apps.folder",
				MimeType: f.MimeType,
			})
		}
	} else {
		if folderID == "" {
			folderID = "root"
		}
		files, err := drv.ListFolder(r.Context(), folderID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, f := range files {
			items = append(items, DriveBrowseItem{
				ID:       f.Id,
				Name:     f.Name,
				IsFolder: f.MimeType == "application/vnd.google-apps.folder",
				MimeType: f.MimeType,
			})
		}
		if folderID != "root" {
			meta, err := drv.GetFileMeta(r.Context(), folderID)
			if err == nil {
				folderName = meta.Name
				if len(meta.Parents) > 0 {
					parentID = meta.Parents[0]
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, DriveBrowseResponse{
		FolderID:   folderID,
		FolderName: folderName,
		ParentID:   parentID,
		Source:     source,
		Items:      items,
	})
}

func (h *Handler) listSharedItems(r *http.Request) ([]*driveapi.File, error) {
	drv := h.GetDrive()
	var files []*driveapi.File
	pageToken := ""
	query := "sharedWithMe = true and trashed = false"

	for {
		call := drv.Service.Files.List().
			Context(r.Context()).
			Q(query).
			Fields("nextPageToken, files(id, name, mimeType)").
			PageSize(1000)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		result, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("list shared: %w", err)
		}

		files = append(files, result.Files...)

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	return files, nil
}
