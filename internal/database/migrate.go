package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate runs all pending database migrations using goose.
func Migrate(db *sql.DB) error {
	goose.SetLogger(goose.NopLogger())
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("setting goose dialect: %w", err)
	}

	currentVersion, err := goose.GetDBVersion(db)
	if err != nil {
		currentVersion = 0
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	newVersion, err := goose.GetDBVersion(db)
	if err != nil {
		return fmt.Errorf("getting db version after migration: %w", err)
	}

	if newVersion > currentVersion {
		slog.Info("migrations applied", "from_version", currentVersion, "to_version", newVersion)
	} else {
		slog.Info("database schema up to date", "version", newVersion)
	}

	return nil
}
