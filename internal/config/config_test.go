package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.AuthMethod != "oauth" {
		t.Errorf("AuthMethod = %q, want %q", cfg.AuthMethod, "oauth")
	}
	if cfg.ChunkSizeMB != 10 {
		t.Errorf("ChunkSizeMB = %d, want 10", cfg.ChunkSizeMB)
	}
	if cfg.MaxWorkers != 3 {
		t.Errorf("MaxWorkers = %d, want 3", cfg.MaxWorkers)
	}
	if cfg.RevisionNaming != "{stem}.rev{rev_id}{suffix}" {
		t.Errorf("RevisionNaming = %q, want default pattern", cfg.RevisionNaming)
	}
	if len(cfg.ExportFormats) != 3 {
		t.Errorf("ExportFormats has %d entries, want 3", len(cfg.ExportFormats))
	}
}

func TestLoadCreatesFileWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.json")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AuthMethod != "oauth" {
		t.Errorf("AuthMethod = %q, want %q", cfg.AuthMethod, "oauth")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("config file was not created")
	}
}

func TestLoadReadsExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	original := Config{
		AuthMethod:   "service_account",
		SyncFolderID: "folder123",
		LocalSyncDir: "/tmp/sync",
		ChunkSizeMB:  20,
		MaxWorkers:   5,
	}

	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AuthMethod != "service_account" {
		t.Errorf("AuthMethod = %q, want %q", cfg.AuthMethod, "service_account")
	}
	if cfg.SyncFolderID != "folder123" {
		t.Errorf("SyncFolderID = %q, want %q", cfg.SyncFolderID, "folder123")
	}
	if cfg.ChunkSizeMB != 20 {
		t.Errorf("ChunkSizeMB = %d, want 20", cfg.ChunkSizeMB)
	}
	if cfg.MaxWorkers != 5 {
		t.Errorf("MaxWorkers = %d, want 5", cfg.MaxWorkers)
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	partial := map[string]any{"sync_folder_id": "abc"}
	data, _ := json.Marshal(partial)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.SyncFolderID != "abc" {
		t.Errorf("SyncFolderID = %q, want %q", cfg.SyncFolderID, "abc")
	}
	if cfg.ChunkSizeMB != 10 {
		t.Errorf("ChunkSizeMB = %d, want default 10", cfg.ChunkSizeMB)
	}
	if cfg.MaxWorkers != 3 {
		t.Errorf("MaxWorkers = %d, want default 3", cfg.MaxWorkers)
	}
	if cfg.AuthMethod != "oauth" {
		t.Errorf("AuthMethod = %q, want default %q", cfg.AuthMethod, "oauth")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := DefaultConfig()
	cfg.SyncFolderID = "test-folder"
	cfg.LocalSyncDir = "/data/test"
	cfg.DistTargets = []DistTargetConf{
		{Name: "backup", Type: "local", Path: "/mnt/backup"},
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.SyncFolderID != cfg.SyncFolderID {
		t.Errorf("SyncFolderID = %q, want %q", loaded.SyncFolderID, cfg.SyncFolderID)
	}
	if loaded.LocalSyncDir != cfg.LocalSyncDir {
		t.Errorf("LocalSyncDir = %q, want %q", loaded.LocalSyncDir, cfg.LocalSyncDir)
	}
	if len(loaded.DistTargets) != 1 {
		t.Fatalf("DistTargets len = %d, want 1", len(loaded.DistTargets))
	}
	if loaded.DistTargets[0].Name != "backup" {
		t.Errorf("DistTargets[0].Name = %q, want %q", loaded.DistTargets[0].Name, "backup")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := os.WriteFile(path, []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("write error: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("Load() expected error for invalid JSON, got nil")
	}
}
