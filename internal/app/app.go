package app

import (
	"database/sql"

	"github.com/zajca/zfaktury/internal/config"
)

// App is the dependency injection container holding all application dependencies.
type App struct {
	Config *config.Config
	DB     *sql.DB
}

// NewApp creates a new App instance and wires all dependencies.
func NewApp(cfg *config.Config, db *sql.DB) *App {
	return &App{
		Config: cfg,
		DB:     db,
	}
}
