package config

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
}

// Config holds all application settings.
type Config struct {
	AuthMethod      string            `json:"auth_method"`       // "oauth" | "service_account"
	CredentialsPath string            `json:"credentials_path"`
	TokenPath       string            `json:"token_path"`
	SyncFolderID    string            `json:"sync_folder_id"`
	LocalSyncDir    string            `json:"local_sync_dir"`
	Scopes          []string          `json:"scopes"`
	ExportFormats   map[string]Export `json:"export_formats"`
	ChunkSizeMB     int              `json:"chunk_size_mb"`
	MaxWorkers      int              `json:"max_workers"`
	RevisionNaming  string           `json:"revision_naming"`
	DistTargets     []DistTargetConf `json:"distribution_targets"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		AuthMethod:     "oauth",
		TokenPath:      "token.json",
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
