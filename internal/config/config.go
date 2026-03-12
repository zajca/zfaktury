package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds the application configuration loaded from config.toml.
type Config struct {
	DataDir string       `toml:"data_dir"`
	Server  ServerConfig `toml:"server"`
	SMTP    SMTPConfig   `toml:"smtp"`
	FIO  FIOConfig `toml:"fio"`
	OCR  OCRConfig `toml:"ocr"`
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

// Load reads configuration from the given configPath, or from DataDir/config.toml if empty.
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
	}

	// Allow overriding data dir via environment variable
	if envDir := os.Getenv("ZFAKTURY_DATA_DIR"); envDir != "" {
		cfg.DataDir = envDir
	}

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data directory %s: %w", cfg.DataDir, err)
	}

	if configPath == "" {
		configPath = filepath.Join(cfg.DataDir, "config.toml")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		slog.Warn("config file not found, using defaults", "path", configPath)
		return cfg, nil
	}

	if _, err := toml.DecodeFile(configPath, cfg); err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", configPath, err)
	}

	slog.Info("config loaded", "path", configPath, "data_dir", cfg.DataDir)
	return cfg, nil
}

// DatabasePath returns the path to the SQLite database file.
func (c *Config) DatabasePath() string {
	return filepath.Join(c.DataDir, "zfaktury.db")
}
