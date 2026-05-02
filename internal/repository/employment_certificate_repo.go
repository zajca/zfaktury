package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// EmploymentCertificateRepository handles persistence of EmploymentCertificate entities.
type EmploymentCertificateRepository struct {
	db *sql.DB
}

// NewEmploymentCertificateRepository creates a new EmploymentCertificateRepository.
func NewEmploymentCertificateRepository(db *sql.DB) *EmploymentCertificateRepository {
	return &EmploymentCertificateRepository{db: db}
}

// employmentCertificateColumns is the list of columns to select for an
// EmploymentCertificate row.
const employmentCertificateColumns = `id, year, document_id, certificate_type,
	employer_name, employer_ico, employer_address, contract_type,
	period_from, period_to,
	gross_income, income_without_advance, foreign_tax_paid,
	advance_tax_withheld, annual_settlement_refund, monthly_bonus_paid,
	withheld_final_tax, include_withholding_in_dap,
	notes, status, deleted_at, created_at, updated_at`

// scanEmploymentCertificate scans an EmploymentCertificate from a row.
func scanEmploymentCertificate(s scanner) (*domain.EmploymentCertificate, error) {
	c := &domain.EmploymentCertificate{}
	var documentID sql.NullInt64
	var certificateType, contractType string
	var periodFromStr, periodToStr string
	var includeWithholding int
	var deletedAtStr sql.NullString
	var createdAtStr, updatedAtStr string

	err := s.Scan(
		&c.ID, &c.Year, &documentID, &certificateType,
		&c.EmployerName, &c.EmployerICO, &c.EmployerAddress, &contractType,
		&periodFromStr, &periodToStr,
		&c.GrossIncome, &c.IncomeWithoutAdvance, &c.ForeignTaxPaid,
		&c.AdvanceTaxWithheld, &c.AnnualSettlementRefund, &c.MonthlyBonusPaid,
		&c.WithheldFinalTax, &includeWithholding,
		&c.Notes, &c.Status, &deletedAtStr, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	if documentID.Valid {
		id := documentID.Int64
		c.DocumentID = &id
	}
	c.CertificateType = domain.CertificateType(certificateType)
	c.ContractType = domain.ContractType(contractType)
	c.IncludeWithholdingInDAP = includeWithholding != 0

	c.PeriodFrom, err = parseDate(time.DateOnly, periodFromStr)
	if err != nil {
		return nil, fmt.Errorf("scanning employment_certificate period_from: %w", err)
	}
	c.PeriodTo, err = parseDate(time.DateOnly, periodToStr)
	if err != nil {
		return nil, fmt.Errorf("scanning employment_certificate period_to: %w", err)
	}
	c.DeletedAt, err = parseDatePtr(time.RFC3339, deletedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning employment_certificate deleted_at: %w", err)
	}
	c.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning employment_certificate created_at: %w", err)
	}
	c.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning employment_certificate updated_at: %w", err)
	}

	return c, nil
}

// Create inserts a new employment certificate into the database. The UNIQUE
// (year, employer_ico, certificate_type, period_from, period_to) constraint
// uses ON CONFLICT REPLACE — re-inserting with the same key overwrites the
// existing row and changes its primary key.
func (r *EmploymentCertificateRepository) Create(ctx context.Context, cert *domain.EmploymentCertificate) error {
	now := time.Now()
	cert.CreatedAt = now
	cert.UpdatedAt = now

	if cert.CertificateType == "" {
		cert.CertificateType = domain.CertificateAdvance
	}
	if cert.ContractType == "" {
		cert.ContractType = domain.ContractDPC
	}
	if cert.Status == "" {
		cert.Status = "draft"
	}

	var documentID any
	if cert.DocumentID != nil {
		documentID = *cert.DocumentID
	}

	includeWithholding := 0
	if cert.IncludeWithholdingInDAP {
		includeWithholding = 1
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO employment_income_certificates (
			year, document_id, certificate_type,
			employer_name, employer_ico, employer_address, contract_type,
			period_from, period_to,
			gross_income, income_without_advance, foreign_tax_paid,
			advance_tax_withheld, annual_settlement_refund, monthly_bonus_paid,
			withheld_final_tax, include_withholding_in_dap,
			notes, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cert.Year, documentID, string(cert.CertificateType),
		cert.EmployerName, cert.EmployerICO, cert.EmployerAddress, string(cert.ContractType),
		cert.PeriodFrom.Format(time.DateOnly), cert.PeriodTo.Format(time.DateOnly),
		cert.GrossIncome, cert.IncomeWithoutAdvance, cert.ForeignTaxPaid,
		cert.AdvanceTaxWithheld, cert.AnnualSettlementRefund, cert.MonthlyBonusPaid,
		cert.WithheldFinalTax, includeWithholding,
		cert.Notes, cert.Status,
		cert.CreatedAt.Format(time.RFC3339), cert.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting employment_certificate: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for employment_certificate: %w", err)
	}
	cert.ID = id
	return nil
}

// GetByID retrieves a non-deleted employment certificate by its ID.
func (r *EmploymentCertificateRepository) GetByID(ctx context.Context, id int64) (*domain.EmploymentCertificate, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+employmentCertificateColumns+` FROM employment_income_certificates WHERE id = ? AND deleted_at IS NULL`,
		id,
	)
	cert, err := scanEmploymentCertificate(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("employment_certificate %d: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying employment_certificate %d: %w", id, err)
	}
	return cert, nil
}

// Update modifies an existing employment certificate.
func (r *EmploymentCertificateRepository) Update(ctx context.Context, cert *domain.EmploymentCertificate) error {
	cert.UpdatedAt = time.Now()

	var documentID any
	if cert.DocumentID != nil {
		documentID = *cert.DocumentID
	}

	includeWithholding := 0
	if cert.IncludeWithholdingInDAP {
		includeWithholding = 1
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE employment_income_certificates SET
			year = ?, document_id = ?, certificate_type = ?,
			employer_name = ?, employer_ico = ?, employer_address = ?, contract_type = ?,
			period_from = ?, period_to = ?,
			gross_income = ?, income_without_advance = ?, foreign_tax_paid = ?,
			advance_tax_withheld = ?, annual_settlement_refund = ?, monthly_bonus_paid = ?,
			withheld_final_tax = ?, include_withholding_in_dap = ?,
			notes = ?, status = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`,
		cert.Year, documentID, string(cert.CertificateType),
		cert.EmployerName, cert.EmployerICO, cert.EmployerAddress, string(cert.ContractType),
		cert.PeriodFrom.Format(time.DateOnly), cert.PeriodTo.Format(time.DateOnly),
		cert.GrossIncome, cert.IncomeWithoutAdvance, cert.ForeignTaxPaid,
		cert.AdvanceTaxWithheld, cert.AnnualSettlementRefund, cert.MonthlyBonusPaid,
		cert.WithheldFinalTax, includeWithholding,
		cert.Notes, cert.Status,
		cert.UpdatedAt.Format(time.RFC3339), cert.ID,
	)
	if err != nil {
		return fmt.Errorf("updating employment_certificate %d: %w", cert.ID, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for employment_certificate %d: %w", cert.ID, err)
	}
	if rows == 0 {
		return fmt.Errorf("employment_certificate %d: %w", cert.ID, domain.ErrNotFound)
	}
	return nil
}

// Delete performs a soft delete on an employment certificate.
func (r *EmploymentCertificateRepository) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx,
		`UPDATE employment_income_certificates SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		now.Format(time.RFC3339), now.Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("soft-deleting employment_certificate %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for employment_certificate %d delete: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("employment_certificate %d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// ListByYear retrieves all non-deleted certificates for a year, ordered by
// employer name and period.
func (r *EmploymentCertificateRepository) ListByYear(ctx context.Context, year int) ([]*domain.EmploymentCertificate, error) {
	return r.queryList(ctx, `WHERE year = ? AND deleted_at IS NULL ORDER BY employer_name, period_from`, year)
}

// ListConfirmedByYear retrieves all non-deleted certificates with status =
// 'confirmed' for a year. Used by Recalculate to aggregate §6 totals.
func (r *EmploymentCertificateRepository) ListConfirmedByYear(ctx context.Context, year int) ([]*domain.EmploymentCertificate, error) {
	return r.queryList(ctx, `WHERE year = ? AND status = 'confirmed' AND deleted_at IS NULL ORDER BY employer_name, period_from`, year)
}

func (r *EmploymentCertificateRepository) queryList(ctx context.Context, where string, args ...any) ([]*domain.EmploymentCertificate, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+employmentCertificateColumns+` FROM employment_income_certificates `+where,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("listing employment_certificates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []*domain.EmploymentCertificate
	for rows.Next() {
		cert, err := scanEmploymentCertificate(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning employment_certificate row: %w", err)
		}
		result = append(result, cert)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating employment_certificate rows: %w", err)
	}
	return result, nil
}
