package drive

import (
	"context"
	"fmt"

	"github.com/P4sTela/matsu-sonic/internal/config"
	driveapi "google.golang.org/api/drive/v3"
)

// DriveClient wraps the Google Drive API service.
type DriveClient struct {
	Service *driveapi.Service
	Config  *config.Config
}

// NewDriveClient creates an authenticated Drive client.
func NewDriveClient(ctx context.Context, cfg *config.Config) (*DriveClient, error) {
	svc, err := Authenticate(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &DriveClient{Service: svc, Config: cfg}, nil
}

// ListFolder lists files in a single folder (non-recursive).
func (d *DriveClient) ListFolder(ctx context.Context, folderID string) ([]*driveapi.File, error) {
	var files []*driveapi.File
	query := fmt.Sprintf("'%s' in parents and trashed = false", folderID)
	pageToken := ""

	for {
		call := d.Service.Files.List().
			Context(ctx).
			Q(query).
			Fields("nextPageToken, files(id, name, mimeType, md5Checksum, size, modifiedTime, parents)").
			PageSize(1000)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		result, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("list folder %s: %w", folderID, err)
		}

		files = append(files, result.Files...)

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	return files, nil
}

// ListAllRecursive lists all files under a folder using BFS.
func (d *DriveClient) ListAllRecursive(ctx context.Context, folderID string) ([]*driveapi.File, error) {
	var allFiles []*driveapi.File
	queue := []string{folderID}

	for len(queue) > 0 {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		current := queue[0]
		queue = queue[1:]

		files, err := d.ListFolder(ctx, current)
		if err != nil {
			return nil, err
		}

		for _, f := range files {
			allFiles = append(allFiles, f)
			if f.MimeType == "application/vnd.google-apps.folder" {
				queue = append(queue, f.Id)
			}
		}
	}

	return allFiles, nil
}

// GetFileMeta returns metadata for a single file.
func (d *DriveClient) GetFileMeta(ctx context.Context, fileID string) (*driveapi.File, error) {
	return d.Service.Files.Get(fileID).
		Context(ctx).
		Fields("id, name, mimeType, md5Checksum, size, modifiedTime, parents").
		Do()
}

// GetStartPageToken returns a token for tracking future changes.
func (d *DriveClient) GetStartPageToken(ctx context.Context) (string, error) {
	resp, err := d.Service.Changes.GetStartPageToken().Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return resp.StartPageToken, nil
}

// GetChanges returns changes since the given page token.
func (d *DriveClient) GetChanges(ctx context.Context, pageToken string) ([]*driveapi.Change, string, error) {
	var changes []*driveapi.Change
	token := pageToken

	for {
		resp, err := d.Service.Changes.List(token).
			Context(ctx).
			Fields("nextPageToken, newStartPageToken, changes(fileId, removed, file(id, name, mimeType, md5Checksum, size, modifiedTime, parents))").
			PageSize(1000).
			Do()
		if err != nil {
			return nil, "", err
		}

		changes = append(changes, resp.Changes...)

		if resp.NewStartPageToken != "" {
			return changes, resp.NewStartPageToken, nil
		}
		token = resp.NextPageToken
	}
}
