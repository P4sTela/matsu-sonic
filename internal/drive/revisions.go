package drive

import (
	"context"
	"fmt"
	"io"
	"os"

	driveapi "google.golang.org/api/drive/v3"
)

// ListRevisions returns all revisions for a file from the Drive API.
func (d *DriveClient) ListRevisions(ctx context.Context, fileID string) ([]*driveapi.Revision, error) {
	resp, err := d.Service.Revisions.List(fileID).
		Context(ctx).
		Fields("revisions(id, modifiedTime, size, lastModifyingUser, mimeType, keepForever, originalFilename)").
		Do()
	if err != nil {
		return nil, fmt.Errorf("list revisions for %s: %w", fileID, err)
	}
	return resp.Revisions, nil
}

// DownloadRevision downloads a specific revision of a file.
func (d *DriveClient) DownloadRevision(ctx context.Context, fileID, revisionID, destPath string) (int64, error) {
	resp, err := d.Service.Revisions.Get(fileID, revisionID).
		Context(ctx).
		Download()
	if err != nil {
		return 0, fmt.Errorf("download revision %s/%s: %w", fileID, revisionID, err)
	}
	defer resp.Body.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return 0, fmt.Errorf("create %s: %w", destPath, err)
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return written, fmt.Errorf("write revision: %w", err)
	}

	return written, nil
}
