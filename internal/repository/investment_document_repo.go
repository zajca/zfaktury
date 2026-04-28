package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// InvestmentDocumentRepository handles persistence of InvestmentDocument entities.
type InvestmentDocumentRepository struct {
	db *sql.DB
}

// NewInvestmentDocumentRepository creates a new InvestmentDocumentRepository.
func NewInvestmentDocumentRepository(db *sql.DB) *InvestmentDocumentRepository {
	return &InvestmentDocumentRepository{db: db}
}

// investmentDocumentColumns is the list of columns to select for an InvestmentDocument row.
const investmentDocumentColumns = `id, year, platform, kind, filename, content_type, storage_path, size, extraction_status, extraction_error, created_at, updated_at`

// scanInvestmentDocument scans an InvestmentDocument from a row.
func scanInvestmentDocument(s scanner) (*domain.InvestmentDocument, error) {
	d := &domain.InvestmentDocument{}
	var extractionError sql.NullString
	var createdAtStr, updatedAtStr string

	err := s.Scan(
		&d.ID, &d.Year, &d.Platform, &d.Kind, &d.Filename, &d.ContentType, &d.StoragePath,
		&d.Size, &d.ExtractionStatus, &extractionError,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	if extractionError.Valid {
		d.ExtractionError = extractionError.String
	}

	d.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning investment_document created_at: %w", err)
	}
	d.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning investment_document updated_at: %w", err)
	}

	return d, nil
}

// Create inserts a new investment document into the database.
func (r *InvestmentDocumentRepository) Create(ctx context.Context, doc *domain.InvestmentDocument) error {
	now := time.Now()
	doc.CreatedAt = now
	doc.UpdatedAt = now

	if doc.Kind == "" {
		doc.Kind = domain.InvestmentDocKindStatement
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO investment_documents (year, platform, kind, filename, content_type, storage_path, size, extraction_status, extraction_error, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		doc.Year, doc.Platform, doc.Kind, doc.Filename, doc.ContentType, doc.StoragePath,
		doc.Size, doc.ExtractionStatus, doc.ExtractionError,
		doc.CreatedAt.Format(time.RFC3339), doc.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting investment_document: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for investment_document: %w", err)
	}
	doc.ID = id
	return nil
}

// GetByID retrieves an investment document by its ID.
func (r *InvestmentDocumentRepository) GetByID(ctx context.Context, id int64) (*domain.InvestmentDocument, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+investmentDocumentColumns+` FROM investment_documents WHERE id = ?`, id)
	doc, err := scanInvestmentDocument(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("investment_document %d: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying investment_document %d: %w", id, err)
	}
	return doc, nil
}

// ListByYear retrieves all investment documents for a given year, ordered by created_at.
func (r *InvestmentDocumentRepository) ListByYear(ctx context.Context, year int) ([]domain.InvestmentDocument, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+investmentDocumentColumns+` FROM investment_documents WHERE year = ? ORDER BY created_at DESC`,
		year,
	)
	if err != nil {
		return nil, fmt.Errorf("listing investment_documents for year %d: %w", year, err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.InvestmentDocument
	for rows.Next() {
		doc, err := scanInvestmentDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning investment_document row: %w", err)
		}
		result = append(result, *doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating investment_document rows: %w", err)
	}
	return result, nil
}

// Delete removes an investment document by ID.
func (r *InvestmentDocumentRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM investment_documents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting investment_document %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for investment_document %d delete: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("investment_document %d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// UpdateExtraction updates the extraction status and error for an investment document.
func (r *InvestmentDocumentRepository) UpdateExtraction(ctx context.Context, id int64, status string, extractionError string) error {
	now := time.Now()

	result, err := r.db.ExecContext(ctx, `
		UPDATE investment_documents SET extraction_status = ?, extraction_error = ?, updated_at = ? WHERE id = ?`,
		status, extractionError, now.Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("updating extraction for investment_document %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for investment_document %d extraction update: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("investment_document %d: %w", id, domain.ErrNotFound)
	}
	return nil
}
