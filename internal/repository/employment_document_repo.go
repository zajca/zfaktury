package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// EmploymentDocumentRepository handles persistence of EmploymentDocument entities.
type EmploymentDocumentRepository struct {
	db *sql.DB
}

// NewEmploymentDocumentRepository creates a new EmploymentDocumentRepository.
func NewEmploymentDocumentRepository(db *sql.DB) *EmploymentDocumentRepository {
	return &EmploymentDocumentRepository{db: db}
}

// employmentDocumentColumns is the list of columns to select for an EmploymentDocument row.
const employmentDocumentColumns = `id, year, document_kind, filename, content_type, storage_path, size, extraction_status, extraction_error, created_at, updated_at`

// scanEmploymentDocument scans an EmploymentDocument from a row.
func scanEmploymentDocument(s scanner) (*domain.EmploymentDocument, error) {
	d := &domain.EmploymentDocument{}
	var kind string
	var extractionError sql.NullString
	var createdAtStr, updatedAtStr string

	err := s.Scan(
		&d.ID, &d.Year, &kind, &d.Filename, &d.ContentType, &d.StoragePath,
		&d.Size, &d.ExtractionStatus, &extractionError,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	d.Kind = domain.EmploymentDocumentKind(kind)
	if extractionError.Valid {
		d.ExtractionError = extractionError.String
	}

	d.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning employment_document created_at: %w", err)
	}
	d.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning employment_document updated_at: %w", err)
	}

	return d, nil
}

// Create inserts a new employment document into the database.
func (r *EmploymentDocumentRepository) Create(ctx context.Context, doc *domain.EmploymentDocument) error {
	now := time.Now()
	doc.CreatedAt = now
	doc.UpdatedAt = now

	if doc.Kind == "" {
		doc.Kind = domain.EmploymentDocAdvance
	}
	if doc.ExtractionStatus == "" {
		doc.ExtractionStatus = domain.ExtractionPending
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO employment_documents (year, document_kind, filename, content_type, storage_path, size, extraction_status, extraction_error, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		doc.Year, string(doc.Kind), doc.Filename, doc.ContentType, doc.StoragePath,
		doc.Size, doc.ExtractionStatus, doc.ExtractionError,
		doc.CreatedAt.Format(time.RFC3339), doc.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting employment_document: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for employment_document: %w", err)
	}
	doc.ID = id
	return nil
}

// GetByID retrieves an employment document by its ID.
func (r *EmploymentDocumentRepository) GetByID(ctx context.Context, id int64) (*domain.EmploymentDocument, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+employmentDocumentColumns+` FROM employment_documents WHERE id = ?`, id)
	doc, err := scanEmploymentDocument(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("employment_document %d: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying employment_document %d: %w", id, err)
	}
	return doc, nil
}

// ListByYear retrieves all employment documents for a given year, newest first.
func (r *EmploymentDocumentRepository) ListByYear(ctx context.Context, year int) ([]*domain.EmploymentDocument, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+employmentDocumentColumns+` FROM employment_documents WHERE year = ? ORDER BY created_at DESC`,
		year,
	)
	if err != nil {
		return nil, fmt.Errorf("listing employment_documents for year %d: %w", year, err)
	}
	defer func() { _ = rows.Close() }()

	var result []*domain.EmploymentDocument
	for rows.Next() {
		doc, err := scanEmploymentDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning employment_document row: %w", err)
		}
		result = append(result, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating employment_document rows: %w", err)
	}
	return result, nil
}

// Delete removes an employment document by ID.
func (r *EmploymentDocumentRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM employment_documents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting employment_document %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for employment_document %d delete: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("employment_document %d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// UpdateExtraction updates the extraction status and error for an employment document.
func (r *EmploymentDocumentRepository) UpdateExtraction(ctx context.Context, id int64, status, errMsg string) error {
	now := time.Now()

	result, err := r.db.ExecContext(ctx, `
		UPDATE employment_documents SET extraction_status = ?, extraction_error = ?, updated_at = ? WHERE id = ?`,
		status, errMsg, now.Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("updating extraction for employment_document %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for employment_document %d extraction update: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("employment_document %d: %w", id, domain.ErrNotFound)
	}
	return nil
}
