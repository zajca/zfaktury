package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// CategoryRepository handles persistence of ExpenseCategory entities.
type CategoryRepository struct {
	db *sql.DB
}

// NewCategoryRepository creates a new CategoryRepository.
func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// Create inserts a new expense category into the database.
func (r *CategoryRepository) Create(ctx context.Context, cat *domain.ExpenseCategory) error {
	now := time.Now()
	cat.CreatedAt = now

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO expense_categories (key, label_cs, label_en, color, sort_order, is_default, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		cat.Key, cat.LabelCS, cat.LabelEN, cat.Color, cat.SortOrder, cat.IsDefault, cat.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting expense category: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for expense category: %w", err)
	}
	cat.ID = id
	return nil
}

// Update modifies an existing expense category.
func (r *CategoryRepository) Update(ctx context.Context, cat *domain.ExpenseCategory) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE expense_categories SET
			key = ?, label_cs = ?, label_en = ?, color = ?, sort_order = ?
		WHERE id = ? AND deleted_at IS NULL`,
		cat.Key, cat.LabelCS, cat.LabelEN, cat.Color, cat.SortOrder, cat.ID,
	)
	if err != nil {
		return fmt.Errorf("updating expense category %d: %w", cat.ID, err)
	}
	return nil
}

// Delete performs a soft delete on an expense category.
func (r *CategoryRepository) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
		UPDATE expense_categories SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("soft-deleting expense category %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for expense category %d: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("expense category %d not found or already deleted", id)
	}
	return nil
}

// GetByID retrieves a single expense category by ID.
func (r *CategoryRepository) GetByID(ctx context.Context, id int64) (*domain.ExpenseCategory, error) {
	cat := &domain.ExpenseCategory{}
	var deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, `
		SELECT id, key, label_cs, label_en, color, sort_order, is_default, created_at, deleted_at
		FROM expense_categories
		WHERE id = ? AND deleted_at IS NULL`, id,
	).Scan(
		&cat.ID, &cat.Key, &cat.LabelCS, &cat.LabelEN, &cat.Color,
		&cat.SortOrder, &cat.IsDefault, &cat.CreatedAt, &deletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("expense category %d not found: %w", id, err)
		}
		return nil, fmt.Errorf("querying expense category %d: %w", id, err)
	}

	if deletedAt.Valid {
		cat.DeletedAt = &deletedAt.Time
	}
	return cat, nil
}

// GetByKey retrieves a single expense category by its unique key.
func (r *CategoryRepository) GetByKey(ctx context.Context, key string) (*domain.ExpenseCategory, error) {
	cat := &domain.ExpenseCategory{}
	var deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, `
		SELECT id, key, label_cs, label_en, color, sort_order, is_default, created_at, deleted_at
		FROM expense_categories
		WHERE key = ? AND deleted_at IS NULL`, key,
	).Scan(
		&cat.ID, &cat.Key, &cat.LabelCS, &cat.LabelEN, &cat.Color,
		&cat.SortOrder, &cat.IsDefault, &cat.CreatedAt, &deletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("expense category with key %q not found: %w", key, err)
		}
		return nil, fmt.Errorf("querying expense category by key %q: %w", key, err)
	}

	if deletedAt.Valid {
		cat.DeletedAt = &deletedAt.Time
	}
	return cat, nil
}

// List retrieves all non-deleted expense categories ordered by sort_order.
func (r *CategoryRepository) List(ctx context.Context) ([]domain.ExpenseCategory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, key, label_cs, label_en, color, sort_order, is_default, created_at, deleted_at
		FROM expense_categories
		WHERE deleted_at IS NULL
		ORDER BY sort_order ASC, key ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing expense categories: %w", err)
	}
	defer rows.Close()

	var categories []domain.ExpenseCategory
	for rows.Next() {
		var cat domain.ExpenseCategory
		var deletedAt sql.NullTime

		if err := rows.Scan(
			&cat.ID, &cat.Key, &cat.LabelCS, &cat.LabelEN, &cat.Color,
			&cat.SortOrder, &cat.IsDefault, &cat.CreatedAt, &deletedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning expense category row: %w", err)
		}

		if deletedAt.Valid {
			cat.DeletedAt = &deletedAt.Time
		}
		categories = append(categories, cat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating expense category rows: %w", err)
	}
	return categories, nil
}
