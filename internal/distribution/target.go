package distribution

import (
	"context"
)

// DirEntry represents a file or directory in a distribution target.
type DirEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
	Path  string `json:"path"`
}

// Target defines the interface for a distribution destination.
type Target interface {
	Type() string
	Distribute(ctx context.Context, src string, destRelative string) (string, error)
	TestConnection(ctx context.Context) error
	ListContents(ctx context.Context, path string) ([]DirEntry, error)
}

// FileCopy represents a single file to distribute.
type FileCopy struct {
	Src          string
	DestRelative string
}

// FileCopyResult represents the result of a single file copy.
type FileCopyResult struct {
	DestPath string
	Err      error
}

// BatchDistributor is optionally implemented by targets that can batch
// multiple file operations over a single connection for better performance.
type BatchDistributor interface {
	DistributeMany(ctx context.Context, files []FileCopy) []FileCopyResult
}
