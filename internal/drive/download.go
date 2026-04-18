package drive

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// DownloadFile downloads a file from Drive to destPath.
// For Google Docs types, it exports using the configured format.
// expectedMD5 is verified after download for binary files (pass "" to skip).
// progress is called with a value between 0.0 and 1.0.
func (d *DriveClient) DownloadFile(ctx context.Context, fileID, destPath, mimeType string, fileSize int64, expectedMD5 string, progress func(float64)) (int64, error) {
	if strings.HasPrefix(mimeType, "application/vnd.google-apps.") {
		return d.exportFile(ctx, fileID, destPath, mimeType, progress)
	}
	return d.downloadBinary(ctx, fileID, destPath, fileSize, expectedMD5, progress)
}

func (d *DriveClient) downloadBinary(ctx context.Context, fileID, destPath string, fileSize int64, expectedMD5 string, progress func(float64)) (int64, error) {
	resp, err := d.Service.Files.Get(fileID).Context(ctx).Download()
	if err != nil {
		return 0, fmt.Errorf("download %s: %w", fileID, err)
	}
	defer resp.Body.Close()

	return d.writeToFile(destPath, resp.Body, fileSize, expectedMD5, progress)
}

func (d *DriveClient) exportFile(ctx context.Context, fileID, destPath, mimeType string, progress func(float64)) (int64, error) {
	exportMime := "application/pdf"
	if exp, ok := d.Config.ExportFormats[mimeType]; ok {
		exportMime = exp.MimeType
	}

	resp, err := d.Service.Files.Export(fileID, exportMime).Context(ctx).Download()
	if err != nil {
		return 0, fmt.Errorf("export %s: %w", fileID, err)
	}
	defer resp.Body.Close()

	return d.writeToFile(destPath, resp.Body, 0, "", progress)
}

func (d *DriveClient) writeToFile(destPath string, body io.Reader, totalSize int64, expectedMD5 string, progress func(float64)) (int64, error) {
	out, err := os.Create(destPath)
	if err != nil {
		return 0, fmt.Errorf("create %s: %w", destPath, err)
	}
	defer out.Close()

	chunkSize := int64(d.Config.ChunkSizeMB) * 1024 * 1024
	if chunkSize <= 0 {
		chunkSize = 10 * 1024 * 1024
	}
	// Cap buffer at 32MB to prevent excessive memory allocation
	const maxBufSize = 32 * 1024 * 1024
	if chunkSize > maxBufSize {
		chunkSize = maxBufSize
	}

	hasher := md5.New()
	reader := body
	if expectedMD5 != "" {
		reader = io.TeeReader(body, hasher)
	}

	buf := make([]byte, chunkSize)
	var written int64

	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			nw, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return written, writeErr
			}
			written += int64(nw)

			if progress != nil && totalSize > 0 {
				progress(float64(written) / float64(totalSize))
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return written, readErr
		}
	}

	if progress != nil {
		progress(1.0)
	}

	if expectedMD5 != "" {
		actual := hex.EncodeToString(hasher.Sum(nil))
		if actual != expectedMD5 {
			os.Remove(destPath)
			return written, fmt.Errorf("checksum mismatch for %s: expected %s, got %s", destPath, expectedMD5, actual)
		}
	}

	return written, nil
}
