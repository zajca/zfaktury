package database

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "modernc.org/sqlite"

	"github.com/zajca/zfaktury/internal/config"
)

// New opens a SQLite database connection with recommended pragmas for WAL mode.
func New(cfg *config.Config) (*sql.DB, error) {
	dbPath := cfg.DatabasePath()

	dsn := dbPath + "?_time_format=sqlite"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database at %s: %w", dbPath, err)
	}

	// Set SQLite pragmas for optimal performance and safety
	pragmas := []struct {
		name  string
		query string
	}{
		{"journal_mode", "PRAGMA journal_mode=WAL"},
		{"foreign_keys", "PRAGMA foreign_keys=ON"},
		{"busy_timeout", "PRAGMA busy_timeout=5000"},
	}

	for _, p := range pragmas {
		if _, err := db.Exec(p.query); err != nil {
			db.Close()
			return nil, fmt.Errorf("setting pragma %s: %w", p.name, err)
		}
	}

	// Verify the connection is working
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	slog.Info("database opened", "path", dbPath)
	return db, nil
}
