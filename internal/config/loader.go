package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Load reads a Config from the JSON file at path.
// If the file does not exist, it returns DefaultConfig and creates the file.
func Load(path string) (Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			applyDefaults(&cfg, filepath.Dir(path))
			if err := Save(path, cfg); err != nil {
				return cfg, err
			}
			return cfg, nil
		}
		return cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	if err := decryptSecrets(&cfg, filepath.Dir(path)); err != nil {
		return cfg, err
	}

	applyDefaults(&cfg, filepath.Dir(path))
	return cfg, nil
}

// Save writes the Config as indented JSON to the given path.
// It creates parent directories if needed.
func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	cfg, err := encryptSecrets(cfg, filepath.Dir(path))
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// applyDefaults fills zero-value fields with sensible defaults. configDir is the
// directory holding config.json; app-local files (like the OAuth token) default
// to live alongside it so everything stays self-contained / portable.
func applyDefaults(cfg *Config, configDir string) {
	if cfg.ChunkSizeMB == 0 {
		cfg.ChunkSizeMB = 10
	}
	if cfg.MaxWorkers == 0 {
		cfg.MaxWorkers = 3
	}
	if cfg.RevisionNaming == "" {
		cfg.RevisionNaming = "{stem}.rev{rev_id}{suffix}"
	}
	if cfg.AuthMethod == "" {
		cfg.AuthMethod = "oauth"
	}
	// Record the config directory so relative paths (e.g. token_path) resolve
	// against it. token_path itself is intentionally left untouched so no
	// absolute path is ever persisted into config.json (keeps it portable).
	cfg.SetConfigDir(configDir)
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"https://www.googleapis.com/auth/drive.readonly"}
	}
}
