package config

import (
	"os"
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	dir := t.TempDir()
	key, err := loadOrCreateKey(dir)
	if err != nil {
		t.Fatalf("loadOrCreateKey: %v", err)
	}

	for _, plain := range []string{"hunter2", "", "日本語パスワード", "with spaces & symbols!"} {
		enc, err := encryptValue(key, plain)
		if err != nil {
			t.Fatalf("encryptValue(%q): %v", plain, err)
		}
		if plain != "" && !strings.HasPrefix(enc, encPrefix) {
			t.Errorf("expected %q to be encrypted with prefix, got %q", plain, enc)
		}
		dec, err := decryptValue(key, enc)
		if err != nil {
			t.Fatalf("decryptValue: %v", err)
		}
		if dec != plain {
			t.Errorf("round trip mismatch: got %q want %q", dec, plain)
		}
	}
}

func TestDecryptPlaintextBackCompat(t *testing.T) {
	dir := t.TempDir()
	key, _ := loadOrCreateKey(dir)
	// A value without the enc: prefix is treated as plaintext.
	got, err := decryptValue(key, "legacy-plaintext")
	if err != nil {
		t.Fatalf("decryptValue: %v", err)
	}
	if got != "legacy-plaintext" {
		t.Errorf("got %q, want legacy-plaintext", got)
	}
}

func TestSaveLoadEncryptsPassword(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.json"

	cfg := DefaultConfig()
	cfg.DistTargets = []DistTargetConf{
		{Name: "nas", Type: "smb", Server: "host", Share: "s", Password: "secret-pw"},
		{Name: "local", Type: "local", Path: "/tmp"},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// On disk the password must be encrypted.
	raw, _ := readFile(t, path)
	if strings.Contains(raw, "secret-pw") {
		t.Fatal("plaintext password found in config file on disk")
	}
	if !strings.Contains(raw, encPrefix) {
		t.Fatal("expected encrypted marker in config file")
	}

	// Loading must yield the plaintext password back.
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.DistTargets[0].Password != "secret-pw" {
		t.Errorf("got %q, want secret-pw", loaded.DistTargets[0].Password)
	}
}

func readFile(t *testing.T, path string) (string, error) {
	t.Helper()
	data, err := os.ReadFile(path)
	return string(data), err
}
