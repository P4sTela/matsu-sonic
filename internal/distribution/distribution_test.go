package distribution

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/P4sTela/matsu-sonic/internal/config"
)

func TestLocalTarget_Distribute(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	os.MkdirAll(srcDir, 0o755)

	// Create source file
	srcFile := filepath.Join(srcDir, "test.txt")
	os.WriteFile(srcFile, []byte("hello world"), 0o644)

	target := &LocalTarget{BasePath: destDir}

	path, err := target.Distribute(context.Background(), srcFile, "subdir/test.txt")
	if err != nil {
		t.Fatalf("Distribute: %v", err)
	}

	expected := filepath.Join(destDir, "subdir", "test.txt")
	if path != expected {
		t.Errorf("path = %q, want %q", path, expected)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("content = %q, want %q", string(data), "hello world")
	}
}

func TestLocalTarget_TestConnection(t *testing.T) {
	tmpDir := t.TempDir()
	target := &LocalTarget{BasePath: filepath.Join(tmpDir, "new-dir")}

	if err := target.TestConnection(context.Background()); err != nil {
		t.Errorf("TestConnection: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(filepath.Join(tmpDir, "new-dir")); os.IsNotExist(err) {
		t.Error("base path should be created")
	}
}

func TestLocalTarget_ListContents(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0o644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0o755)

	target := &LocalTarget{BasePath: tmpDir}
	entries, err := target.ListContents(context.Background(), "")
	if err != nil {
		t.Fatalf("ListContents: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name] = true
		if e.Name == "subdir" && !e.IsDir {
			t.Error("subdir should be a directory")
		}
	}
	if !names["a.txt"] || !names["subdir"] {
		t.Errorf("unexpected entries: %v", entries)
	}
}

func TestSMBTarget_NotImplemented(t *testing.T) {
	target := &SMBTarget{}

	if _, err := target.Distribute(context.Background(), "", ""); err != ErrNotImplemented {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
	if err := target.TestConnection(context.Background()); err != ErrNotImplemented {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
	if _, err := target.ListContents(context.Background(), ""); err != ErrNotImplemented {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
}

func TestManager(t *testing.T) {
	configs := []config.DistTargetConf{
		{Name: "local1", Type: "local", Path: "/tmp/test"},
		{Name: "smb1", Type: "smb", Server: "nas"},
	}

	mgr := NewManager(configs)

	local, err := mgr.Get("local1")
	if err != nil {
		t.Fatalf("Get local1: %v", err)
	}
	if local.Type() != "local" {
		t.Errorf("type = %q, want %q", local.Type(), "local")
	}

	smb, err := mgr.Get("smb1")
	if err != nil {
		t.Fatalf("Get smb1: %v", err)
	}
	if smb.Type() != "smb" {
		t.Errorf("type = %q, want %q", smb.Type(), "smb")
	}

	_, err = mgr.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent target")
	}
}
