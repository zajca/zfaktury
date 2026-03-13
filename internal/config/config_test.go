package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("getting home dir: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{"empty", "", ""},
		{"absolute", "/etc/config.toml", "/etc/config.toml"},
		{"relative", "config.toml", "config.toml"},
		{"tilde only", "~", home},
		{"tilde slash", "~/config.toml", filepath.Join(home, "config.toml")},
		{"tilde nested", "~/.zfaktury/me/config.toml", filepath.Join(home, ".zfaktury/me/config.toml")},
		{"tilde no slash", "~foo", filepath.Join(home, "foo")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandHome(tt.path)
			if got != tt.want {
				t.Errorf("ExpandHome(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestResolve_ExplicitMissing_NoInit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.toml")
	_, err := Resolve(path, false)
	if err == nil {
		t.Fatal("expected error for missing explicit config without --init-config")
	}
}

func TestResolve_ExplicitMissing_WithInit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.toml")
	got, err := Resolve(path, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != path {
		t.Errorf("Resolve() = %q, want %q", got, path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file was not created: %v", err)
	}
}

func TestResolve_ExplicitExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("# existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(path, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != path {
		t.Errorf("Resolve() = %q, want %q", got, path)
	}
}

func TestResolve_DefaultAutoCreates(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	got, err := Resolve("", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(tmpDir, ".zfaktury", "config.toml")
	if got != expected {
		t.Errorf("Resolve() = %q, want %q", got, expected)
	}
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("default config was not created: %v", err)
	}
}

func TestWriteTemplate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "config.toml")
	if err := WriteTemplate(path); err != nil {
		t.Fatalf("WriteTemplate() error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading template: %v", err)
	}
	if len(data) == 0 {
		t.Error("template file is empty")
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	content := `
data_dir = "` + dir + `"

[log]
level = "debug"

[server]
port = 9090
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("log level = %q, want debug", cfg.Log.Level)
	}
}
