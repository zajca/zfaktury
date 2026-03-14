package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// configTemplate is the default configuration file template.
const configTemplate = `# ZFaktury configuration
# See https://github.com/zajca/zfaktury for documentation.

# [database]
# path = ""                        # Custom DB path (default: <data_dir>/zfaktury.db)

# [log]
# path = ""                        # Log file path (default: stderr only)
# level = "info"                   # debug, info, warn, error

# [server]
# port = 8080

# [smtp]
# host = ""
# port = 587
# username = ""
# password = ""
# from = ""

# [fio]
# api_token = ""

# [backup]
# destination = ""              # Backup directory (default: <data_dir>/backups)
# schedule = ""                 # Cron expression, e.g. "0 2 * * *"
# retention_count = 10          # Keep last N backups, 0 = keep all
# [backup.s3]                   # S3-compatible storage (AWS S3, MinIO, Backblaze B2, Cloudflare R2)
# endpoint = ""                 # e.g. "s3.amazonaws.com" or "minio.example.com:9000"
# region = ""                   # e.g. "eu-central-1"
# bucket = ""                   # Bucket name
# access_key = ""
# secret_key = ""
# use_ssl = true

# [ocr]
# provider = ""
# api_key = ""
# model = ""
# base_url = ""
`

// Config holds the application configuration loaded from config.toml.
type Config struct {
	DataDir  string         `toml:"data_dir"`
	Database DatabaseConfig `toml:"database"`
	Log      LogConfig      `toml:"log"`
	Server   ServerConfig   `toml:"server"`
	Backup   BackupConfig   `toml:"backup"`
	SMTP     SMTPConfig     `toml:"smtp"`
	FIO      FIOConfig      `toml:"fio"`
	OCR      OCRConfig      `toml:"ocr"`
}

// DatabaseConfig holds database settings.
type DatabaseConfig struct {
	Path        string `toml:"path"`
	JournalMode string `toml:"journal_mode"` // wal (default) or delete
}

// BackupConfig holds backup settings.
type BackupConfig struct {
	Destination    string   `toml:"destination"`     // Backup directory (default: DataDir/backups)
	Schedule       string   `toml:"schedule"`        // Cron expression, e.g. "0 2 * * *"
	RetentionCount int      `toml:"retention_count"` // Keep last N backups, 0 = keep all
	S3             S3Config `toml:"s3"`
}

// S3Config holds S3-compatible storage settings.
type S3Config struct {
	Endpoint  string `toml:"endpoint"`
	Region    string `toml:"region"`
	Bucket    string `toml:"bucket"`
	AccessKey string `toml:"access_key"`
	SecretKey string `toml:"secret_key"`
	UseSSL    bool   `toml:"use_ssl"`
}

// IsConfigured returns true when both Endpoint and Bucket are non-empty.
func (c *S3Config) IsConfigured() bool {
	return c.Endpoint != "" && c.Bucket != ""
}

// LogConfig holds logging settings.
type LogConfig struct {
	Path  string `toml:"path"`  // empty = stderr only
	Level string `toml:"level"` // debug, info, warn, error (default: info)
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port int  `toml:"port"`
	Dev  bool `toml:"dev"`
}

// SMTPConfig holds email sending settings.
type SMTPConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	From     string `toml:"from"`
}

// FIOConfig holds FIO Bank API settings.
type FIOConfig struct {
	APIToken string `toml:"api_token"`
}

// OCRConfig holds OCR service settings.
type OCRConfig struct {
	Provider string `toml:"provider"`
	APIKey   string `toml:"api_key"`
	Model    string `toml:"model"`
	BaseURL  string `toml:"base_url"`
}

// defaultDataDir returns the default data directory (~/.zfaktury).
func defaultDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".zfaktury"), nil
}

// ExpandHome replaces a leading ~ with the user's home directory.
func ExpandHome(path string) string {
	if path == "" || !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}

// DefaultConfigPath returns the default config file path (~/.zfaktury/config.toml).
func DefaultConfigPath() (string, error) {
	dataDir, err := defaultDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "config.toml"), nil
}

// WriteTemplate creates a config file with commented-out defaults at the given path.
// Parent directories are created if needed.
func WriteTemplate(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(configTemplate), 0o644); err != nil {
		return fmt.Errorf("writing config template: %w", err)
	}
	return nil
}

// Resolve determines the config file path and creates a template if needed.
// If explicit is non-empty, it is used as the config path. When the file does not exist:
//   - if initConfig is true, a template is created
//   - otherwise an error is returned suggesting --init-config
//
// If explicit is empty, the default path is used and a template is created
// automatically if it does not exist.
func Resolve(explicit string, initConfig bool) (string, error) {
	if explicit != "" {
		path := ExpandHome(explicit)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if !initConfig {
				return "", fmt.Errorf("config file not found: %s\nUse --init-config to create a default config file", path)
			}
			if err := WriteTemplate(path); err != nil {
				return "", fmt.Errorf("creating config file: %w", err)
			}
			slog.Info("created default config file", "path", path)
		}
		return path, nil
	}

	path, err := DefaultConfigPath()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := WriteTemplate(path); err != nil {
			return "", fmt.Errorf("creating config file: %w", err)
		}
		slog.Info("created default config file", "path", path)
	}
	return path, nil
}

// Load reads configuration from the given configPath.
// The config file must exist -- call WriteTemplate first if needed.
// If DataDir is not set, it defaults to ~/.zfaktury.
// The DataDir is created if it does not exist.
func Load(configPath string) (*Config, error) {
	dataDir, err := defaultDataDir()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		DataDir: dataDir,
		Server: ServerConfig{
			Port: 8080,
		},
		Backup: BackupConfig{
			S3: S3Config{
				UseSSL: true,
			},
		},
	}

	// Allow overriding data dir via environment variable
	if envDir := os.Getenv("ZFAKTURY_DATA_DIR"); envDir != "" {
		cfg.DataDir = envDir
	}

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data directory %s: %w", cfg.DataDir, err)
	}

	configPath = ExpandHome(configPath)

	if _, err := toml.DecodeFile(configPath, cfg); err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", configPath, err)
	}

	cfg.DataDir = ExpandHome(cfg.DataDir)
	cfg.Database.Path = ExpandHome(cfg.Database.Path)
	cfg.Log.Path = ExpandHome(cfg.Log.Path)
	cfg.Backup.Destination = ExpandHome(cfg.Backup.Destination)

	slog.Info("config loaded", "path", configPath, "data_dir", cfg.DataDir)
	return cfg, nil
}

// BackupDestination returns the backup directory path.
// If Backup.Destination is set, it is used directly.
// Otherwise defaults to DataDir/backups.
func (c *Config) BackupDestination() string {
	if c.Backup.Destination != "" {
		return c.Backup.Destination
	}
	return filepath.Join(c.DataDir, "backups")
}

// DatabasePath returns the path to the SQLite database file.
// If Database.Path is set in config, it is used directly.
// Otherwise defaults to DataDir/zfaktury.db.
func (c *Config) DatabasePath() string {
	if c.Database.Path != "" {
		return c.Database.Path
	}
	return filepath.Join(c.DataDir, "zfaktury.db")
}
