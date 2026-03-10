package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// DocumentRepository handles persistence of ExpenseDocument entities.
type DocumentRepository struct {
	db *sql.DB
}

// NewDocumentRepository creates a new DocumentRepository.
func NewDocumentRepository(db *sql.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

// Create inserts a new expense document into the database.
func (r *DocumentRepository) Create(ctx context.Context, doc *domain.ExpenseDocument) error {
	now := time.Now()
	doc.CreatedAt = now

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO expense_documents (expense_id, filename, content_type, storage_path, size, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		doc.ExpenseID, doc.Filename, doc.ContentType, doc.StoragePath, doc.Size, doc.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting expense document: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for expense document: %w", err)
	}
	doc.ID = id
	return nil
}

// GetByID retrieves a single expense document by ID.
func (r *DocumentRepository) GetByID(ctx context.Context, id int64) (*domain.ExpenseDocument, error) {
	doc := &domain.ExpenseDocument{}
	var createdAtStr string
	var deletedAtStr sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT id, expense_id, filename, content_type, storage_path, size, created_at, deleted_at
		FROM expense_documents
		WHERE id = ? AND deleted_at IS NULL`, id,
	).Scan(
		&doc.ID, &doc.ExpenseID, &doc.Filename, &doc.ContentType,
		&doc.StoragePath, &doc.Size, &createdAtStr, &deletedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("expense document %d not found: %w", id, err)
		}
		return nil, fmt.Errorf("querying expense document %d: %w", id, err)
	}

	doc.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	if deletedAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, deletedAtStr.String)
		doc.DeletedAt = &t
	}

	return doc, nil
}

// ListByExpenseID retrieves all active documents for a given expense.
func (r *DocumentRepository) ListByExpenseID(ctx context.Context, expenseID int64) ([]domain.ExpenseDocument, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, expense_id, filename, content_type, storage_path, size, created_at, deleted_at
		FROM expense_documents
		WHERE expense_id = ? AND deleted_at IS NULL
		ORDER BY created_at ASC`, expenseID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing expense documents for expense %d: %w", expenseID, err)
	}
	defer rows.Close()

	var docs []domain.ExpenseDocument
	for rows.Next() {
		var doc domain.ExpenseDocument
		var createdAtStr string
		var deletedAtStr sql.NullString

		if err := rows.Scan(
			&doc.ID, &doc.ExpenseID, &doc.Filename, &doc.ContentType,
			&doc.StoragePath, &doc.Size, &createdAtStr, &deletedAtStr,
		); err != nil {
			return nil, fmt.Errorf("scanning expense document row: %w", err)
		}

		doc.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		if deletedAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, deletedAtStr.String)
			doc.DeletedAt = &t
		}

		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating expense document rows: %w", err)
	}

	return docs, nil
}

// Delete performs a soft delete on an expense document.
func (r *DocumentRepository) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
		UPDATE expense_documents SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`,
		now.Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("soft-deleting expense document %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for expense document %d: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("expense document %d not found or already deleted", id)
	}
	return nil
}

// CountByExpenseID returns the number of active documents for a given expense.
func (r *DocumentRepository) CountByExpenseID(ctx context.Context, expenseID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM expense_documents
		WHERE expense_id = ? AND deleted_at IS NULL`, expenseID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting documents for expense %d: %w", expenseID, err)
	}
	return count, nil
}
