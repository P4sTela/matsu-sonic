package config

import "path/filepath"

// Export defines a Google Docs export format mapping.
type Export struct {
	MimeType  string `json:"mime"`
	Extension string `json:"ext"`
}

// DistTargetConf defines a distribution target configuration.
type DistTargetConf struct {
	Name     string `json:"name"`
	Type     string `json:"type"`     // "local" | "smb"
	Path     string `json:"path"`     // local用
	Server   string `json:"server"`   // smb用
	Share    string `json:"share"`    // smb用
	Username string `json:"username"` // smb用
	Password string `json:"password"` // smb用
	Domain   string `json:"domain"`   // smb用
	// SelectPatterns limits which synced files are distributed to this target.
	// Empty means all files. Matched against the file path relative to the sync root.
	SelectPatterns []string `json:"select_patterns"`
	// Converter names a converter whose output file should be distributed
	// instead of the original synced file. Leave empty to distribute the original.
	Converter string `json:"converter,omitempty"`
}

// ConverterConf defines an external command-based file converter.
type ConverterConf struct {
	Name            string   `json:"name"`
	Enabled         bool     `json:"enabled"`
	InputPattern    string   `json:"input_pattern"`    // glob, e.g. "*.mp4"
	OutputExtension string   `json:"output_extension"` // e.g. ".mov"
	OutputDir       string   `json:"output_dir"`       // e.g. "converted/hap" (relative to sync root)
	Command         string   `json:"command"`          // e.g. "ffmpeg -i {{input}} -c:v hap {{output}}"
	Env             []string `json:"env,omitempty"`
	AutoConvert     bool     `json:"auto_convert"` // run automatically after sync
}

// Config holds all application settings.
type Config struct {
	AuthMethod       string            `json:"auth_method"` // "oauth" | "service_account"
	CredentialsPath  string            `json:"credentials_path"`
	TokenPath        string            `json:"token_path"`
	SyncFolderID     string            `json:"sync_folder_id"`
	LocalSyncDir     string            `json:"local_sync_dir"`
	Scopes           []string          `json:"scopes"`
	ExportFormats    map[string]Export `json:"export_formats"`
	ChunkSizeMB      int               `json:"chunk_size_mb"`
	MaxWorkers       int               `json:"max_workers"`
	RevisionNaming   string            `json:"revision_naming"`
	IgnorePatterns   []string          `json:"ignore_patterns"`
	SelectPatterns   []string          `json:"select_patterns"`   // 同期対象を限定する include パターン（空なら全件）
	ConflictStrategy string            `json:"conflict_strategy"` // "skip" | "overwrite"
	Converters       []ConverterConf   `json:"converters"`
	ConverterWorkers int               `json:"converter_workers"` // default 1
	DistTargets      []DistTargetConf  `json:"distribution_targets"`

	// configDir is the directory containing config.json. It is never persisted
	// (unexported), and is used to resolve relative paths like token_path so the
	// app folder stays portable.
	configDir string
}

// SetConfigDir records the directory of config.json so relative paths can be
// resolved against it at runtime.
func (c *Config) SetConfigDir(dir string) { c.configDir = dir }

// SecretDir returns the directory holding the local encryption key (secret.key),
// i.e. the config directory. Used to encrypt the OAuth token at rest.
func (c *Config) SecretDir() string { return c.configDir }

// ResolvedTokenPath returns the absolute path to the OAuth token file. An empty
// or relative token_path is resolved against the config directory, so no
// absolute path is ever baked into config.json and the whole folder remains
// portable.
func (c *Config) ResolvedTokenPath() string {
	p := c.TokenPath
	if p == "" {
		p = "token.json"
	}
	if filepath.IsAbs(p) || c.configDir == "" {
		return p
	}
	return filepath.Join(c.configDir, p)
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		AuthMethod:       "oauth",
		ConflictStrategy: "skip",
		ConverterWorkers: 1,
		// TokenPath is left empty and resolved at runtime via ResolvedTokenPath
		// (relative to the config dir) so no absolute path is persisted.
		Scopes:         []string{"https://www.googleapis.com/auth/drive.readonly"},
		ChunkSizeMB:    10,
		MaxWorkers:     3,
		RevisionNaming: "{stem}.rev{rev_id}{suffix}",
		ExportFormats: map[string]Export{
			"application/vnd.google-apps.document":     {MimeType: "application/pdf", Extension: ".pdf"},
			"application/vnd.google-apps.spreadsheet":  {MimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Extension: ".xlsx"},
			"application/vnd.google-apps.presentation": {MimeType: "application/pdf", Extension: ".pdf"},
		},
	}
}
