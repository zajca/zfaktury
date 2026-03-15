package server

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/zajca/zfaktury/internal/config"
)

// setupLogging configures the default slog logger based on config.
// If a log path is configured, logs are written to both stderr and the file.
// Returns the opened file (if any) so the caller can defer Close.
func setupLogging(cfg config.LogConfig) (*os.File, error) {
	level := slog.LevelInfo
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: level}

	if cfg.Path != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.Path), 0o755); err != nil {
			return nil, fmt.Errorf("creating log directory: %w", err)
		}
		f, err := os.OpenFile(cfg.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("opening log file: %w", err)
		}
		w := io.MultiWriter(os.Stderr, f)
		slog.SetDefault(slog.New(slog.NewTextHandler(w, opts)))
		return f, nil
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, opts)))
	return nil, nil
}
