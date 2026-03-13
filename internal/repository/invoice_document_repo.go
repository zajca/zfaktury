package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// InvoiceDocumentRepository handles persistence of InvoiceDocument entities.
type InvoiceDocumentRepository struct {
	db *sql.DB
}

// NewInvoiceDocumentRepository creates a new InvoiceDocumentRepository.
func NewInvoiceDocumentRepository(db *sql.DB) *InvoiceDocumentRepository {
	return &InvoiceDocumentRepository{db: db}
}

// Create inserts a new invoice document into the database.
func (r *InvoiceDocumentRepository) Create(ctx context.Context, doc *domain.InvoiceDocument) error {
	now := time.Now()
	doc.CreatedAt = now

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO invoice_documents (invoice_id, filename, content_type, storage_path, size, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		doc.InvoiceID, doc.Filename, doc.ContentType, doc.StoragePath, doc.Size, doc.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting invoice document: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for invoice document: %w", err)
	}
	doc.ID = id
	return nil
}

// GetByID retrieves a single invoice document by ID.
func (r *InvoiceDocumentRepository) GetByID(ctx context.Context, id int64) (*domain.InvoiceDocument, error) {
	doc := &domain.InvoiceDocument{}
	var createdAtStr string
	var deletedAtStr sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT id, invoice_id, filename, content_type, storage_path, size, created_at, deleted_at
		FROM invoice_documents
		WHERE id = ? AND deleted_at IS NULL`, id,
	).Scan(
		&doc.ID, &doc.InvoiceID, &doc.Filename, &doc.ContentType,
		&doc.StoragePath, &doc.Size, &createdAtStr, &deletedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("invoice document %d not found: %w", id, err)
		}
		return nil, fmt.Errorf("querying invoice document %d: %w", id, err)
	}

	doc.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning invoice document: %w", err)
	}
	doc.DeletedAt, err = parseDatePtr(time.RFC3339, deletedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning invoice document: %w", err)
	}

	return doc, nil
}

// ListByInvoiceID retrieves all active documents for a given invoice.
func (r *InvoiceDocumentRepository) ListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.InvoiceDocument, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, invoice_id, filename, content_type, storage_path, size, created_at, deleted_at
		FROM invoice_documents
		WHERE invoice_id = ? AND deleted_at IS NULL
		ORDER BY created_at ASC`, invoiceID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing invoice documents for invoice %d: %w", invoiceID, err)
	}
	defer func() { _ = rows.Close() }()

	var docs []domain.InvoiceDocument
	for rows.Next() {
		var doc domain.InvoiceDocument
		var createdAtStr string
		var deletedAtStr sql.NullString

		if err := rows.Scan(
			&doc.ID, &doc.InvoiceID, &doc.Filename, &doc.ContentType,
			&doc.StoragePath, &doc.Size, &createdAtStr, &deletedAtStr,
		); err != nil {
			return nil, fmt.Errorf("scanning invoice document row: %w", err)
		}

		doc.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("scanning invoice document row: %w", err)
		}
		doc.DeletedAt, err = parseDatePtr(time.RFC3339, deletedAtStr)
		if err != nil {
			return nil, fmt.Errorf("scanning invoice document row: %w", err)
		}

		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating invoice document rows: %w", err)
	}

	return docs, nil
}

// Delete performs a soft delete on an invoice document.
func (r *InvoiceDocumentRepository) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
		UPDATE invoice_documents SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`,
		now.Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("soft-deleting invoice document %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for invoice document %d: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("invoice document %d not found or already deleted", id)
	}
	return nil
}

// CountByInvoiceID returns the number of active documents for a given invoice.
func (r *InvoiceDocumentRepository) CountByInvoiceID(ctx context.Context, invoiceID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM invoice_documents
		WHERE invoice_id = ? AND deleted_at IS NULL`, invoiceID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting documents for invoice %d: %w", invoiceID, err)
	}
	return count, nil
}
