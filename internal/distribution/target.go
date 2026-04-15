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
