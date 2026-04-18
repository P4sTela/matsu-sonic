package handler

import (
	"github.com/P4sTela/matsu-sonic/internal/sync"
)

// --- Common responses ---

type StatusResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// --- Auth ---

type AuthUser struct {
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
}

type AuthTestResponse struct {
	Status string   `json:"status"`
	User   AuthUser `json:"user"`
}

// --- Sync ---

type SyncStartResponse struct {
	Status string `json:"status"`
	Mode   string `json:"mode"`
}

type SyncStatusResponse struct {
	IsRunning bool                  `json:"is_running"`
	Progress  sync.ProgressSnapshot `json:"progress"`
}

// --- Files ---

type DeleteFilesRequest struct {
	FileIDs []string `json:"file_ids"`
}

type DeleteFilesResponse struct {
	Deleted int `json:"deleted"`
}

// --- Browse ---

type BrowseItem struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
}

type BrowseResponse struct {
	Current string       `json:"current"`
	Parent  string       `json:"parent"`
	Cwd     string       `json:"cwd"`
	Items   []BrowseItem `json:"items"`
}

type MakeDirRequest struct {
	Path string `json:"path"`
}

type MakeDirResponse struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

// --- Drive Browse ---

type DriveBrowseItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsFolder bool   `json:"is_folder"`
	MimeType string `json:"mime_type"`
}

type DriveBrowseResponse struct {
	FolderID   string            `json:"folder_id"`
	FolderName string            `json:"folder_name"`
	ParentID   string            `json:"parent_id"`
	Source     string            `json:"source"`
	Items      []DriveBrowseItem `json:"items"`
}

// --- Distribution ---

type DistributeRequest struct {
	FileIDs    []string `json:"file_ids"`
	TargetName string   `json:"target_name"`
	DestDir    string   `json:"dest_dir"`
}

type DistributeResult struct {
	FileID string `json:"file_id"`
	Status string `json:"status"`
	Path   string `json:"path,omitempty"`
	Error  string `json:"error,omitempty"`
}

// --- Verify ---

type VerifyResult struct {
	FileID   string `json:"file_id"`
	Name     string `json:"name"`
	Status   string `json:"status"` // "ok", "mismatch", "missing", "skipped"
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
}

type VerifyResponse struct {
	Total    int            `json:"total"`
	Ok       int            `json:"ok"`
	Mismatch int            `json:"mismatch"`
	Missing  int            `json:"missing"`
	Skipped  int            `json:"skipped"`
	Results  []VerifyResult `json:"results"`
}

// --- Revisions ---

type RevisionDownloadRequest struct {
	DestDir string `json:"dest_dir"`
}

type RevisionDownloadResponse struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}
