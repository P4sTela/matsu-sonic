package distribution

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalTarget distributes files to a local filesystem path.
type LocalTarget struct {
	BasePath string
}

func (t *LocalTarget) Type() string { return "local" }

// Distribute copies src to BasePath/destRelative, preserving directory structure.
func (t *LocalTarget) Distribute(_ context.Context, src string, destRelative string) (string, error) {
	destPath := filepath.Join(t.BasePath, destRelative)

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	in, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create dest: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return "", fmt.Errorf("copy: %w", err)
	}

	// Preserve modification time
	srcInfo, err := os.Stat(src)
	if err == nil {
		os.Chtimes(destPath, srcInfo.ModTime(), srcInfo.ModTime())
	}

	return destPath, nil
}

// TestConnection verifies the base path is writable.
func (t *LocalTarget) TestConnection(_ context.Context) error {
	if err := os.MkdirAll(t.BasePath, 0o755); err != nil {
		return fmt.Errorf("cannot create base path: %w", err)
	}

	testFile := filepath.Join(t.BasePath, ".gdrive-sync-test")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("cannot write to base path: %w", err)
	}
	f.Close()
	os.Remove(testFile)

	return nil
}

// ListContents lists files and directories at the given path under BasePath.
func (t *LocalTarget) ListContents(_ context.Context, path string) ([]DirEntry, error) {
	fullPath := filepath.Join(t.BasePath, path)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	var result []DirEntry
	for _, e := range entries {
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		result = append(result, DirEntry{
			Name:  e.Name(),
			IsDir: e.IsDir(),
			Size:  size,
			Path:  filepath.Join(path, e.Name()),
		})
	}
	return result, nil
}
