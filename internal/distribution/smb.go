package distribution

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path"

	"github.com/hirochachacha/go-smb2"
)

// SMBTarget distributes files to an SMB/CIFS share.
type SMBTarget struct {
	Server   string
	Share    string
	Username string
	Password string
	Domain   string
}

func (t *SMBTarget) Type() string { return "smb" }

// mount connects to the SMB server and mounts the share.
// Caller must close both the returned *smb2.Share and net.Conn.
func (t *SMBTarget) mount(ctx context.Context) (*smb2.Share, net.Conn, error) {
	addr := t.Server + ":445"
	conn, err := new(net.Dialer).DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to %s: %w", addr, err)
	}

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     t.Username,
			Password: t.Password,
			Domain:   t.Domain,
		},
	}

	s, err := d.DialContext(ctx, conn)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("smb session: %w", err)
	}

	share, err := s.Mount(t.Share)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("mount share %q: %w", t.Share, err)
	}

	return share, conn, nil
}

// Distribute copies src to Share/destRelative, preserving directory structure.
func (t *SMBTarget) Distribute(ctx context.Context, src string, destRelative string) (string, error) {
	share, conn, err := t.mount(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	destPath := destRelative
	dir := path.Dir(destPath)
	if dir != "" && dir != "." {
		if err := share.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("mkdir %q: %w", dir, err)
		}
	}

	in, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	out, err := share.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create dest: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return "", fmt.Errorf("copy: %w", err)
	}

	fullPath := fmt.Sprintf(`\\%s\%s\%s`, t.Server, t.Share, destPath)
	return fullPath, nil
}

// TestConnection verifies the SMB share is accessible and writable.
func (t *SMBTarget) TestConnection(ctx context.Context) error {
	share, conn, err := t.mount(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	testFile := ".gdrive-sync-test"
	f, err := share.Create(testFile)
	if err != nil {
		return fmt.Errorf("cannot write to share: %w", err)
	}
	f.Close()
	share.Remove(testFile)

	return nil
}

// ListContents lists files and directories at the given path in the share.
func (t *SMBTarget) ListContents(ctx context.Context, dirPath string) ([]DirEntry, error) {
	share, conn, err := t.mount(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	target := dirPath
	if target == "" {
		target = "."
	}

	entries, err := share.ReadDir(target)
	if err != nil {
		return nil, err
	}

	var result []DirEntry
	for _, e := range entries {
		result = append(result, DirEntry{
			Name:  e.Name(),
			IsDir: e.IsDir(),
			Size:  e.Size(),
			Path:  path.Join(dirPath, e.Name()),
		})
	}
	return result, nil
}
