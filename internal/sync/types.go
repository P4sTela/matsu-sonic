package sync

// DiffEntry represents a file that would be synced in a dry run.
type DiffEntry struct {
	FileID        string `json:"file_id"`
	Name          string `json:"name"`
	MimeType      string `json:"mime_type"`
	Size          int64  `json:"size"`
	DriveModified string `json:"drive_modified"`
	LocalPath     string `json:"local_path"`
	Action        string `json:"action"` // "new" | "update" | "delete"
}
