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
	if h.Drive == nil || h.Drive.Service == nil {
		writeError(w, http.StatusInternalServerError, "Drive client not initialized")
		return
	}

	folderID := r.URL.Query().Get("folder_id")
	source := r.URL.Query().Get("source")

	type item struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		IsFolder bool   `json:"is_folder"`
		MimeType string `json:"mime_type"`
	}

	var items []item
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
			items = append(items, item{
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
		files, err := h.Drive.ListFolder(r.Context(), folderID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, f := range files {
			items = append(items, item{
				ID:       f.Id,
				Name:     f.Name,
				IsFolder: f.MimeType == "application/vnd.google-apps.folder",
				MimeType: f.MimeType,
			})
		}
		if folderID != "root" {
			meta, err := h.Drive.GetFileMeta(r.Context(), folderID)
			if err == nil {
				folderName = meta.Name
				if len(meta.Parents) > 0 {
					parentID = meta.Parents[0]
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"folder_id":   folderID,
		"folder_name": folderName,
		"parent_id":   parentID,
		"source":      source,
		"items":       items,
	})
}

func (h *Handler) listSharedItems(r *http.Request) ([]*driveapi.File, error) {
	var files []*driveapi.File
	pageToken := ""
	query := "sharedWithMe = true and trashed = false"

	for {
		call := h.Drive.Service.Files.List().
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
