package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the runtime configuration for the panel.
type Config struct {
	// Listen is the address the HTTP server binds to.
	Listen string `json:"listen"`
	// DataDir is where the panel stores its SQLite database and runtime state.
	DataDir string `json:"data_dir"`
	// JWTSecret signs authentication tokens. Generated on first run if empty.
	JWTSecret string `json:"jwt_secret"`
	// FileRoot constrains the file manager. Empty means the whole filesystem ("/").
	FileRoot string `json:"file_root"`
	// BackupDir is where archive backups are written.
	BackupDir string `json:"backup_dir"`

	// FTPUser/FTPGroup are the system user/group that pure-ftpd virtual users
	// map to. They must exist on the host for FTP management to work.
	FTPUser  string `json:"ftp_user"`
	FTPGroup string `json:"ftp_group"`

	// TLS configures HTTPS for the panel itself.
	TLSEnabled        bool   `json:"tls_enabled"`
	TLSCert           string `json:"tls_cert"`
	TLSKey            string `json:"tls_key"`
	TLSAutoSelfSigned bool   `json:"tls_auto_self_signed"`
}

// DBPath returns the path to the SQLite database file.
func (c *Config) DBPath() string {
	return filepath.Join(c.DataDir, "digitarlopanel.db")
}

// Default returns a configuration with sensible defaults.
func Default() *Config {
	return &Config{
		Listen:    ":8088",
		DataDir:   "/var/lib/digitarlopanel",
		FileRoot:  "/",
		BackupDir: "/var/backups/digitarlopanel",
		FTPUser:   "ftpuser",
		FTPGroup:  "ftpgroup",
	}
}

// Load reads the configuration from path, creating it with defaults if missing.
// A JWT secret is generated and persisted on first run.
func Load(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	case os.IsNotExist(err):
		// keep defaults, will be persisted below
	default:
		return nil, err
	}

	if cfg.DataDir == "" {
		cfg.DataDir = Default().DataDir
	}
	if cfg.Listen == "" {
		cfg.Listen = Default().Listen
	}
	if cfg.FileRoot == "" {
		cfg.FileRoot = "/"
	}
	if cfg.BackupDir == "" {
		cfg.BackupDir = Default().BackupDir
	}
	if cfg.FTPUser == "" {
		cfg.FTPUser = Default().FTPUser
	}
	if cfg.FTPGroup == "" {
		cfg.FTPGroup = Default().FTPGroup
	}
	if cfg.JWTSecret == "" {
		secret, err := randomSecret(32)
		if err != nil {
			return nil, err
		}
		cfg.JWTSecret = secret
	}

	if err := os.MkdirAll(cfg.DataDir, 0o750); err != nil {
		return nil, err
	}
	if err := Save(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes the configuration to path with restrictive permissions.
func Save(path string, cfg *Config) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func randomSecret(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
