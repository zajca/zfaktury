package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// VIESSummaryRepository handles persistence of VIES summary entities.
type VIESSummaryRepository struct {
	db *sql.DB
}

// NewVIESSummaryRepository creates a new VIESSummaryRepository.
func NewVIESSummaryRepository(db *sql.DB) *VIESSummaryRepository {
	return &VIESSummaryRepository{db: db}
}

// scanVIESSummaryRow scans the core VIES summary columns from a row.
func scanVIESSummaryRow(s scanner) (*domain.VIESSummary, error) {
	vs := &domain.VIESSummary{}
	var xmlData []byte
	var filedAtStr sql.NullString
	var createdAtStr, updatedAtStr string

	err := s.Scan(
		&vs.ID, &vs.Period.Year, &vs.Period.Quarter, &vs.FilingType,
		&xmlData, &vs.Status, &filedAtStr,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	vs.XMLData = xmlData

	createdAt, err := parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at: %w", err)
	}
	vs.CreatedAt = createdAt

	updatedAt, err := parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at: %w", err)
	}
	vs.UpdatedAt = updatedAt

	filedAt, err := parseDatePtr(time.RFC3339, filedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing filed_at: %w", err)
	}
	vs.FiledAt = filedAt

	return vs, nil
}

// scanVIESSummaryLine scans a single VIES summary line row.
func scanVIESSummaryLine(s scanner) (*domain.VIESSummaryLine, error) {
	line := &domain.VIESSummaryLine{}
	err := s.Scan(
		&line.ID, &line.VIESSummaryID,
		&line.PartnerDIC, &line.CountryCode,
		&line.TotalAmount, &line.ServiceCode,
	)
	if err != nil {
		return nil, err
	}
	return line, nil
}

const viesSummaryCols = `id, year, quarter, filing_type, xml_data, status, filed_at, created_at, updated_at`

// Create inserts a new VIES summary into the database.
func (r *VIESSummaryRepository) Create(ctx context.Context, vs *domain.VIESSummary) error {
	now := time.Now()
	vs.CreatedAt = now
	vs.UpdatedAt = now

	if vs.Status == "" {
		vs.Status = domain.FilingStatusDraft
	}

	var filedAt any
	if vs.FiledAt != nil {
		filedAt = vs.FiledAt.Format(time.RFC3339)
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO vies_summaries (
			year, quarter, filing_type, xml_data, status, filed_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		vs.Period.Year, vs.Period.Quarter, vs.FilingType,
		vs.XMLData, vs.Status, filedAt,
		vs.CreatedAt.Format(time.RFC3339), vs.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting VIES summary: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting VIES summary ID: %w", err)
	}
	vs.ID = id
	return nil
}

// Update updates an existing VIES summary.
func (r *VIESSummaryRepository) Update(ctx context.Context, vs *domain.VIESSummary) error {
	vs.UpdatedAt = time.Now()

	var filedAt any
	if vs.FiledAt != nil {
		filedAt = vs.FiledAt.Format(time.RFC3339)
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE vies_summaries SET
			year = ?, quarter = ?, filing_type = ?,
			xml_data = ?, status = ?, filed_at = ?,
			updated_at = ?
		WHERE id = ?`,
		vs.Period.Year, vs.Period.Quarter, vs.FilingType,
		vs.XMLData, vs.Status, filedAt,
		vs.UpdatedAt.Format(time.RFC3339), vs.ID,
	)
	if err != nil {
		return fmt.Errorf("updating VIES summary: %w", err)
	}
	return nil
}

// Delete removes a VIES summary by ID.
func (r *VIESSummaryRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM vies_summaries WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting VIES summary: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// GetByID retrieves a VIES summary by ID.
func (r *VIESSummaryRepository) GetByID(ctx context.Context, id int64) (*domain.VIESSummary, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+viesSummaryCols+` FROM vies_summaries WHERE id = ?`, id,
	)
	vs, err := scanVIESSummaryRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("getting VIES summary by ID: %w", err)
	}
	return vs, nil
}

// List retrieves all VIES summaries for a given year.
func (r *VIESSummaryRepository) List(ctx context.Context, year int) ([]domain.VIESSummary, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+viesSummaryCols+` FROM vies_summaries WHERE year = ? ORDER BY quarter ASC`, year,
	)
	if err != nil {
		return nil, fmt.Errorf("listing VIES summaries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.VIESSummary
	for rows.Next() {
		vs, err := scanVIESSummaryRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning VIES summary: %w", err)
		}
		result = append(result, *vs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating VIES summaries: %w", err)
	}
	return result, nil
}

// GetByPeriod retrieves a VIES summary for a specific year, quarter, and filing type.
func (r *VIESSummaryRepository) GetByPeriod(ctx context.Context, year, quarter int, filingType string) (*domain.VIESSummary, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+viesSummaryCols+` FROM vies_summaries WHERE year = ? AND quarter = ? AND filing_type = ?`,
		year, quarter, filingType,
	)
	vs, err := scanVIESSummaryRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("getting VIES summary by period: %w", err)
	}
	return vs, nil
}

// CreateLines inserts multiple VIES summary lines.
func (r *VIESSummaryRepository) CreateLines(ctx context.Context, lines []domain.VIESSummaryLine) error {
	for _, line := range lines {
		result, err := r.db.ExecContext(ctx, `
			INSERT INTO vies_summary_lines (
				vies_summary_id, partner_dic, country_code, total_amount, service_code
			) VALUES (?, ?, ?, ?, ?)`,
			line.VIESSummaryID, line.PartnerDIC, line.CountryCode,
			line.TotalAmount, line.ServiceCode,
		)
		if err != nil {
			return fmt.Errorf("inserting VIES summary line: %w", err)
		}
		_, err = result.LastInsertId()
		if err != nil {
			return fmt.Errorf("getting VIES summary line ID: %w", err)
		}
	}
	return nil
}

// DeleteLines removes all lines for a given VIES summary.
func (r *VIESSummaryRepository) DeleteLines(ctx context.Context, viesSummaryID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM vies_summary_lines WHERE vies_summary_id = ?`, viesSummaryID)
	if err != nil {
		return fmt.Errorf("deleting VIES summary lines: %w", err)
	}
	return nil
}

// GetLines retrieves all lines for a given VIES summary.
func (r *VIESSummaryRepository) GetLines(ctx context.Context, viesSummaryID int64) ([]domain.VIESSummaryLine, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, vies_summary_id, partner_dic, country_code, total_amount, service_code
		FROM vies_summary_lines WHERE vies_summary_id = ? ORDER BY country_code, partner_dic`,
		viesSummaryID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing VIES summary lines: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.VIESSummaryLine
	for rows.Next() {
		line, err := scanVIESSummaryLine(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning VIES summary line: %w", err)
		}
		result = append(result, *line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating VIES summary lines: %w", err)
	}
	return result, nil
}
