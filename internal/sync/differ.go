package sync

import (
	"os"

	"github.com/P4sTela/matsu-sonic/internal/store"
	driveapi "google.golang.org/api/drive/v3"
)

// NeedsSync determines whether a remote file needs to be downloaded.
func NeedsSync(remote *driveapi.File, local *store.SyncedFile) bool {
	// New file — never synced
	if local == nil {
		return true
	}

	// MD5 match means identical content (not available for Google Docs)
	if remote.Md5Checksum != "" && remote.Md5Checksum == local.MD5Checksum {
		return false
	}

	// Remote is newer
	if remote.ModifiedTime > local.DriveModified {
		return true
	}

	// Local file missing from disk
	if local.LocalPath != "" {
		if _, err := os.Stat(local.LocalPath); os.IsNotExist(err) {
			return true
		}
	}

	return false
}
