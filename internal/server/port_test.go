package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zajca/zfaktury/internal/config"
)

func loadTestConfig(t *testing.T, serverSection string) *config.Config {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	content := "data_dir = \"" + dir + "\"\n" + serverSection
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	return cfg
}

func TestEffectivePort(t *testing.T) {
	tests := []struct {
		name          string
		serverSection string
		override      int
		want          int
	}{
		{name: "config port without flag", serverSection: "[server]\nport = 9090\n", override: 0, want: 9090},
		{name: "flag overrides config port", serverSection: "[server]\nport = 9090\n", override: 7000, want: 7000},
		{name: "neither config port nor flag", serverSection: "", override: 0, want: 8080},
		{name: "flag without config port", serverSection: "", override: 7000, want: 7000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := loadTestConfig(t, tt.serverSection)
			got := effectivePort(cfg.Server.Port, tt.override)
			if got != tt.want {
				t.Errorf("effectivePort() = %d, want %d", got, tt.want)
			}
		})
	}
}
