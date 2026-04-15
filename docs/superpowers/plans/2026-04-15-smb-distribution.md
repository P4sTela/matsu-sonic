# SMB Distribution Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement SMB/CIFS file distribution so users can copy synced Google Drive files to Windows shared folders by specifying `\\server\share` and credentials.

**Architecture:** Replace the existing SMBTarget stub with a real implementation using `github.com/hirochachacha/go-smb2`. The SMB target connects via NTLM authentication, mounts the named share, and copies files preserving directory hierarchy. Frontend adds SMB-specific fields to the target creation dialog.

**Tech Stack:** Go + go-smb2 (SMB2/3 client), React + TypeScript (frontend)

---

### Task 1: Add go-smb2 dependency

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add the dependency**

```bash
cd /Users/p4stela/ghq/github.com/P4sTela/matsu-sonic
go get github.com/hirochachacha/go-smb2
```

- [ ] **Step 2: Verify it was added**

```bash
grep go-smb2 go.mod
```

Expected: a line like `github.com/hirochachacha/go-smb2 v...`

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add go-smb2 dependency for SMB distribution"
```

---

### Task 2: Implement SMBTarget

**Files:**
- Modify: `internal/distribution/smb.go`

The `SMBTarget` struct already exists with the right fields. Replace the stub methods with real implementations.

- [ ] **Step 1: Write the SMBTarget implementation**

Replace the entire contents of `internal/distribution/smb.go` with:

```go
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
	// Ensure parent directory exists
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
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/p4stela/ghq/github.com/P4sTela/matsu-sonic
go build ./internal/distribution/
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/distribution/smb.go
git commit -m "feat: implement SMBTarget with go-smb2"
```

---

### Task 3: Update SMB tests

**Files:**
- Modify: `internal/distribution/distribution_test.go`

Since we can't run a real SMB server in unit tests, replace the "not implemented" test with a test that verifies the struct implements the Target interface and has the correct type.

- [ ] **Step 1: Update the SMB test**

Replace the `TestSMBTarget_NotImplemented` function in `internal/distribution/distribution_test.go` with:

```go
func TestSMBTarget_ImplementsInterface(t *testing.T) {
	target := &SMBTarget{
		Server:   "testserver",
		Share:    "testshare",
		Username: "user",
		Password: "pass",
		Domain:   "WORKGROUP",
	}

	// Verify it implements Target interface
	var _ Target = target

	if target.Type() != "smb" {
		t.Errorf("type = %q, want %q", target.Type(), "smb")
	}
}
```

- [ ] **Step 2: Run all distribution tests**

```bash
cd /Users/p4stela/ghq/github.com/P4sTela/matsu-sonic
go test ./internal/distribution/ -v
```

Expected: all tests PASS (LocalTarget tests + SMBTarget interface test + Manager test)

- [ ] **Step 3: Commit**

```bash
git add internal/distribution/distribution_test.go
git commit -m "test: update SMB test from not-implemented check to interface verification"
```

---

### Task 4: Add password field to frontend DistTarget type

**Files:**
- Modify: `frontend/src/api/types.ts`

The `DistTarget` type is missing the `password` and `domain` fields.

- [ ] **Step 1: Update DistTarget interface**

In `frontend/src/api/types.ts`, replace the `DistTarget` interface with:

```typescript
export interface DistTarget {
  name: string;
  type: "local" | "smb";
  path: string;
  server?: string;
  share?: string;
  username?: string;
  password?: string;
  domain?: string;
}
```

- [ ] **Step 2: Commit**

```bash
cd /Users/p4stela/ghq/github.com/P4sTela/matsu-sonic
git add frontend/src/api/types.ts
git commit -m "feat: add password and domain fields to DistTarget type"
```

---

### Task 5: Update DistributePage with SMB form fields

**Files:**
- Modify: `frontend/src/pages/DistributePage.tsx`

Add a type selector (local/smb) to the add-target dialog. When `smb` is selected, show server/share/username/password/domain fields instead of path.

- [ ] **Step 1: Update the add-target dialog**

In `frontend/src/pages/DistributePage.tsx`, replace the dialog's `<div className="space-y-4">` block (inside `<DialogContent>`) with:

```tsx
<div className="space-y-4">
  <div>
    <Label>Name</Label>
    <Input
      value={newTarget.name || ""}
      onChange={(e) => setNewTarget({ ...newTarget, name: e.target.value })}
      placeholder="e.g. backup-drive"
    />
  </div>
  <div>
    <Label>Type</Label>
    <select
      value={newTarget.type || "local"}
      onChange={(e) =>
        setNewTarget({ ...newTarget, type: e.target.value as "local" | "smb" })
      }
      className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm"
    >
      <option value="local">Local Path</option>
      <option value="smb">SMB Share</option>
    </select>
  </div>
  {newTarget.type === "smb" ? (
    <>
      <div>
        <Label>Server</Label>
        <Input
          value={newTarget.server || ""}
          onChange={(e) => setNewTarget({ ...newTarget, server: e.target.value })}
          placeholder="e.g. 192.168.1.10 or PC-NAME"
        />
      </div>
      <div>
        <Label>Share</Label>
        <Input
          value={newTarget.share || ""}
          onChange={(e) => setNewTarget({ ...newTarget, share: e.target.value })}
          placeholder="e.g. shared-folder"
        />
      </div>
      <div>
        <Label>Username</Label>
        <Input
          value={newTarget.username || ""}
          onChange={(e) => setNewTarget({ ...newTarget, username: e.target.value })}
          placeholder="user"
        />
      </div>
      <div>
        <Label>Password</Label>
        <Input
          type="password"
          value={newTarget.password || ""}
          onChange={(e) => setNewTarget({ ...newTarget, password: e.target.value })}
          placeholder="password"
        />
      </div>
      <div>
        <Label>Domain (optional)</Label>
        <Input
          value={newTarget.domain || ""}
          onChange={(e) => setNewTarget({ ...newTarget, domain: e.target.value })}
          placeholder="WORKGROUP"
        />
      </div>
    </>
  ) : (
    <div>
      <Label>Path</Label>
      <Input
        value={newTarget.path || ""}
        onChange={(e) => setNewTarget({ ...newTarget, path: e.target.value })}
        placeholder="/path/to/destination"
      />
    </div>
  )}
  <Button onClick={handleAdd} className="w-full">
    <Send className="mr-2 h-4 w-4" />
    Add
  </Button>
</div>
```

- [ ] **Step 2: Update the handleAdd validation**

Replace the `handleAdd` function with:

```typescript
const handleAdd = async () => {
  if (!newTarget.name) return;
  if (newTarget.type === "smb") {
    if (!newTarget.server || !newTarget.share) return;
  } else {
    if (!newTarget.path) return;
  }
  try {
    await api.addTarget(newTarget as DistTarget);
    setDialogOpen(false);
    setNewTarget({ type: "local" });
    load();
  } catch {
    // TODO: show error
  }
};
```

- [ ] **Step 3: Update target list to show SMB info**

In the target list rendering, replace the `<span>` that shows `t.path` with:

```tsx
<span className="ml-2 text-sm text-muted-foreground">
  {t.type === "smb" ? `\\\\${t.server}\\${t.share}` : t.path}
</span>
```

- [ ] **Step 4: Verify frontend builds**

```bash
cd /Users/p4stela/ghq/github.com/P4sTela/matsu-sonic/frontend
npm run build
```

Expected: build succeeds with no errors

- [ ] **Step 5: Commit**

```bash
cd /Users/p4stela/ghq/github.com/P4sTela/matsu-sonic
git add frontend/src/pages/DistributePage.tsx
git commit -m "feat: add SMB fields to distribution target dialog"
```

---

### Task 6: Full build verification

**Files:** None (verification only)

- [ ] **Step 1: Run Go tests**

```bash
cd /Users/p4stela/ghq/github.com/P4sTela/matsu-sonic
go test ./... -v
```

Expected: all tests pass

- [ ] **Step 2: Build the full binary**

```bash
cd /Users/p4stela/ghq/github.com/P4sTela/matsu-sonic
make build
```

Expected: binary builds successfully

- [ ] **Step 3: Verify the binary starts**

```bash
./matsu-sonic -version
```

Expected: prints version string
