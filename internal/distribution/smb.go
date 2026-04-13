package distribution

import "context"

// SMBTarget distributes files to an SMB/CIFS share.
// Currently a stub — all methods return ErrNotImplemented.
type SMBTarget struct {
	Server   string
	Share    string
	Username string
	Password string
	Domain   string
}

func (t *SMBTarget) Type() string { return "smb" }

func (t *SMBTarget) Distribute(_ context.Context, _ string, _ string) (string, error) {
	return "", ErrNotImplemented
}

func (t *SMBTarget) TestConnection(_ context.Context) error {
	return ErrNotImplemented
}

func (t *SMBTarget) ListContents(_ context.Context, _ string) ([]DirEntry, error) {
	return nil, ErrNotImplemented
}
