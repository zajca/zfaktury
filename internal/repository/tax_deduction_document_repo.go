package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// TaxDeductionDocumentRepository handles persistence of TaxDeductionDocument entities.
type TaxDeductionDocumentRepository struct {
	db *sql.DB
}

// NewTaxDeductionDocumentRepository creates a new TaxDeductionDocumentRepository.
func NewTaxDeductionDocumentRepository(db *sql.DB) *TaxDeductionDocumentRepository {
	return &TaxDeductionDocumentRepository{db: db}
}

// taxDeductionDocumentColumns is the list of columns to select for a TaxDeductionDocument row.
const taxDeductionDocumentColumns = `id, tax_deduction_id, filename, content_type, storage_path, size, extracted_amount, confidence, created_at, deleted_at`

// scanTaxDeductionDocument scans a TaxDeductionDocument from a row.
func scanTaxDeductionDocument(s scanner) (*domain.TaxDeductionDocument, error) {
	d := &domain.TaxDeductionDocument{}
	var createdAtStr string
	var deletedAtStr sql.NullString

	err := s.Scan(
		&d.ID, &d.TaxDeductionID, &d.Filename, &d.ContentType,
		&d.StoragePath, &d.Size, &d.ExtractedAmount, &d.Confidence,
		&createdAtStr, &deletedAtStr,
	)
	if err != nil {
		return nil, err
	}

	d.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning tax_deduction_document created_at: %w", err)
	}
	d.DeletedAt, err = parseDatePtr(time.RFC3339, deletedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning tax_deduction_document deleted_at: %w", err)
	}

	return d, nil
}

// Create inserts a new tax deduction document into the database.
func (r *TaxDeductionDocumentRepository) Create(ctx context.Context, doc *domain.TaxDeductionDocument) error {
	now := time.Now()
	doc.CreatedAt = now

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO tax_deduction_documents (tax_deduction_id, filename, content_type, storage_path, size, extracted_amount, confidence, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		doc.TaxDeductionID, doc.Filename, doc.ContentType, doc.StoragePath,
		doc.Size, doc.ExtractedAmount, doc.Confidence,
		doc.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting tax_deduction_document: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for tax_deduction_document: %w", err)
	}
	doc.ID = id
	return nil
}

// GetByID retrieves a single tax deduction document by ID (excluding soft-deleted).
func (r *TaxDeductionDocumentRepository) GetByID(ctx context.Context, id int64) (*domain.TaxDeductionDocument, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+taxDeductionDocumentColumns+` FROM tax_deduction_documents WHERE id = ? AND deleted_at IS NULL`, id,
	)
	doc, err := scanTaxDeductionDocument(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("tax_deduction_document %d: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying tax_deduction_document %d: %w", id, err)
	}
	return doc, nil
}

// ListByDeductionID retrieves all active documents for a given tax deduction.
func (r *TaxDeductionDocumentRepository) ListByDeductionID(ctx context.Context, deductionID int64) ([]domain.TaxDeductionDocument, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+taxDeductionDocumentColumns+` FROM tax_deduction_documents WHERE tax_deduction_id = ? AND deleted_at IS NULL ORDER BY created_at ASC`,
		deductionID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing tax_deduction_documents for deduction %d: %w", deductionID, err)
	}
	defer func() { _ = rows.Close() }()

	var docs []domain.TaxDeductionDocument
	for rows.Next() {
		doc, err := scanTaxDeductionDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning tax_deduction_document row: %w", err)
		}
		docs = append(docs, *doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tax_deduction_document rows: %w", err)
	}
	return docs, nil
}

// Delete performs a soft delete on a tax deduction document.
func (r *TaxDeductionDocumentRepository) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
		UPDATE tax_deduction_documents SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`,
		now.Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("soft-deleting tax_deduction_document %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for tax_deduction_document %d: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("tax_deduction_document %d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// UpdateExtraction updates the extracted amount and confidence for a document.
func (r *TaxDeductionDocumentRepository) UpdateExtraction(ctx context.Context, id int64, amount domain.Amount, confidence float64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tax_deduction_documents SET extracted_amount = ?, confidence = ? WHERE id = ? AND deleted_at IS NULL`,
		amount, confidence, id,
	)
	if err != nil {
		return fmt.Errorf("updating extraction for tax_deduction_document %d: %w", id, err)
	}
	return nil
}
