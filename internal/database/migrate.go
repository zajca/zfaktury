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

// MigrationsFS returns the embedded migrations filesystem for use in tests.
func MigrationsFS() embed.FS { return migrationsFS }

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

	// Disable FK checks during migrations (table recreation may temporarily break references).
	if _, err := db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("disabling foreign keys for migration: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	// Re-enable FK checks after migrations.
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("re-enabling foreign keys after migration: %w", err)
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
