package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/P4sTela/matsu-sonic/internal/store"
	driveapi "google.golang.org/api/drive/v3"
)

func TestNeedsSync_NewFile(t *testing.T) {
	remote := &driveapi.File{Id: "f1", Md5Checksum: "abc123"}
	if !NeedsSync(remote, nil) {
		t.Error("new file (local=nil) should need sync")
	}
}

func TestNeedsSync_SameMD5(t *testing.T) {
	remote := &driveapi.File{Id: "f1", Md5Checksum: "abc123", ModifiedTime: "2026-01-02T00:00:00Z"}
	local := &store.SyncedFile{FileID: "f1", MD5Checksum: "abc123", DriveModified: "2026-01-01T00:00:00Z"}

	if NeedsSync(remote, local) {
		t.Error("same MD5 should not need sync")
	}
}

func TestNeedsSync_DifferentMD5(t *testing.T) {
	remote := &driveapi.File{Id: "f1", Md5Checksum: "new123", ModifiedTime: "2026-01-02T00:00:00Z"}
	local := &store.SyncedFile{FileID: "f1", MD5Checksum: "old456", DriveModified: "2026-01-01T00:00:00Z"}

	if !NeedsSync(remote, local) {
		t.Error("different MD5 + newer remote should need sync")
	}
}

func TestNeedsSync_RemoteNewer(t *testing.T) {
	remote := &driveapi.File{Id: "f1", ModifiedTime: "2026-01-02T00:00:00Z"}
	local := &store.SyncedFile{FileID: "f1", DriveModified: "2026-01-01T00:00:00Z"}

	if !NeedsSync(remote, local) {
		t.Error("newer remote (no MD5) should need sync")
	}
}

func TestNeedsSync_RemoteOlder(t *testing.T) {
	remote := &driveapi.File{Id: "f1", ModifiedTime: "2026-01-01T00:00:00Z"}

	dir := t.TempDir()
	localPath := filepath.Join(dir, "file.pdf")
	os.WriteFile(localPath, []byte("data"), 0o644)

	local := &store.SyncedFile{FileID: "f1", DriveModified: "2026-01-02T00:00:00Z", LocalPath: localPath}

	if NeedsSync(remote, local) {
		t.Error("older remote with existing local file should not need sync")
	}
}

func TestNeedsSync_LocalFileMissing(t *testing.T) {
	remote := &driveapi.File{Id: "f1", ModifiedTime: "2026-01-01T00:00:00Z"}
	local := &store.SyncedFile{FileID: "f1", DriveModified: "2026-01-01T00:00:00Z", LocalPath: "/nonexistent/file.pdf"}

	if !NeedsSync(remote, local) {
		t.Error("missing local file should need sync")
	}
}

func TestNeedsSync_GoogleDocsNoMD5(t *testing.T) {
	remote := &driveapi.File{Id: "f1", Md5Checksum: "", ModifiedTime: "2026-01-02T00:00:00Z"}
	local := &store.SyncedFile{FileID: "f1", MD5Checksum: "", DriveModified: "2026-01-01T00:00:00Z"}

	if !NeedsSync(remote, local) {
		t.Error("Google Doc with newer modifiedTime should need sync")
	}
}
