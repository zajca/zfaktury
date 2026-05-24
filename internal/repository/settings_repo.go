package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// SettingsRepository handles persistence of key-value settings.
type SettingsRepository struct {
	db *sql.DB
}

// NewSettingsRepository creates a new SettingsRepository.
func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// GetAll retrieves all settings for the given company as a key-value map.
func (r *SettingsRepository) GetAll(ctx context.Context, companyID int64) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value FROM settings WHERE company_id = ?`, companyID)
	if err != nil {
		return nil, fmt.Errorf("querying all settings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scanning setting row: %w", err)
		}
		settings[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating setting rows: %w", err)
	}
	return settings, nil
}

// Get retrieves a single setting value by key within the given company.
//
// Returns a wrapped domain.ErrNotFound when no setting exists for the key.
func (r *SettingsRepository) Get(ctx context.Context, companyID int64, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE company_id = ? AND key = ?`, companyID, key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("setting %q not found: %w", key, domain.ErrNotFound)
		}
		return "", fmt.Errorf("querying setting %q: %w", key, err)
	}
	return value, nil
}

// Set upserts a single key-value setting within the given company.
//
// Uses ON CONFLICT(company_id, key) which matches the UNIQUE(company_id, key)
// constraint added by migration 025.
func (r *SettingsRepository) Set(ctx context.Context, companyID int64, key, value string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO settings (company_id, key, value, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(company_id, key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		companyID, key, value, now,
	)
	if err != nil {
		return fmt.Errorf("upserting setting %q: %w", key, err)
	}
	return nil
}

// SetBulk upserts multiple key-value settings within the given company in a
// single transaction.
func (r *SettingsRepository) SetBulk(ctx context.Context, companyID int64, settings map[string]string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction for bulk settings: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC().Format(time.RFC3339)
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO settings (company_id, key, value, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(company_id, key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`)
	if err != nil {
		return fmt.Errorf("preparing bulk settings statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for key, value := range settings {
		if _, err := stmt.ExecContext(ctx, companyID, key, value, now); err != nil {
			return fmt.Errorf("upserting setting %q in bulk: %w", key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing bulk settings: %w", err)
	}
	return nil
}
