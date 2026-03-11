package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// VATControlStatementRepository handles persistence of VATControlStatement entities.
type VATControlStatementRepository struct {
	db *sql.DB
}

// NewVATControlStatementRepository creates a new VATControlStatementRepository.
func NewVATControlStatementRepository(db *sql.DB) *VATControlStatementRepository {
	return &VATControlStatementRepository{db: db}
}

// scanControlStatement scans a control statement row.
// Column order: id, year, month, filing_type, xml_data, status, filed_at, created_at, updated_at
func scanControlStatement(s scanner) (*domain.VATControlStatement, error) {
	cs := &domain.VATControlStatement{}
	var xmlData []byte
	var filedAtStr sql.NullString
	var createdAtStr, updatedAtStr string

	err := s.Scan(
		&cs.ID, &cs.Period.Year, &cs.Period.Month, &cs.FilingType,
		&xmlData, &cs.Status, &filedAtStr,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	cs.XMLData = xmlData

	cs.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning control statement created_at: %w", err)
	}
	cs.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning control statement updated_at: %w", err)
	}
	cs.FiledAt, err = parseDatePtr(time.RFC3339, filedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning control statement filed_at: %w", err)
	}

	return cs, nil
}

// scanControlStatementLine scans a control statement line row.
// Column order: id, control_statement_id, section, partner_dic, document_number,
// dppd, base, vat, vat_rate_percent, invoice_id, expense_id
func scanControlStatementLine(s scanner) (domain.VATControlStatementLine, error) {
	var line domain.VATControlStatementLine
	var invoiceID, expenseID sql.NullInt64

	err := s.Scan(
		&line.ID, &line.ControlStatementID, &line.Section,
		&line.PartnerDIC, &line.DocumentNumber, &line.DPPD,
		&line.Base, &line.VAT, &line.VATRatePercent,
		&invoiceID, &expenseID,
	)
	if err != nil {
		return line, err
	}

	if invoiceID.Valid {
		line.InvoiceID = &invoiceID.Int64
	}
	if expenseID.Valid {
		line.ExpenseID = &expenseID.Int64
	}

	return line, nil
}

// Create inserts a new control statement.
func (r *VATControlStatementRepository) Create(ctx context.Context, cs *domain.VATControlStatement) error {
	now := time.Now()
	cs.CreatedAt = now
	cs.UpdatedAt = now

	var filedAt any
	if cs.FiledAt != nil {
		filedAt = cs.FiledAt.Format(time.RFC3339)
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO vat_control_statements (
			year, month, filing_type, xml_data, status, filed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		cs.Period.Year, cs.Period.Month, cs.FilingType,
		cs.XMLData, cs.Status, filedAt,
		cs.CreatedAt.Format(time.RFC3339), cs.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting control statement: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for control statement: %w", err)
	}
	cs.ID = id
	return nil
}

// Update modifies an existing control statement.
func (r *VATControlStatementRepository) Update(ctx context.Context, cs *domain.VATControlStatement) error {
	cs.UpdatedAt = time.Now()

	var filedAt any
	if cs.FiledAt != nil {
		filedAt = cs.FiledAt.Format(time.RFC3339)
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE vat_control_statements SET
			year = ?, month = ?, filing_type = ?, xml_data = ?,
			status = ?, filed_at = ?, updated_at = ?
		WHERE id = ?`,
		cs.Period.Year, cs.Period.Month, cs.FilingType, cs.XMLData,
		cs.Status, filedAt, cs.UpdatedAt.Format(time.RFC3339), cs.ID,
	)
	if err != nil {
		return fmt.Errorf("updating control statement %d: %w", cs.ID, err)
	}
	return nil
}

// Delete removes a control statement and its lines.
func (r *VATControlStatementRepository) Delete(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction for control statement delete: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `DELETE FROM vat_control_statement_lines WHERE control_statement_id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting control statement lines for %d: %w", id, err)
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM vat_control_statements WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting control statement %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for control statement %d: %w", id, err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing control statement delete: %w", err)
	}
	return nil
}

// GetByID retrieves a control statement by ID.
func (r *VATControlStatementRepository) GetByID(ctx context.Context, id int64) (*domain.VATControlStatement, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, year, month, filing_type, xml_data, status, filed_at, created_at, updated_at
		FROM vat_control_statements WHERE id = ?`, id,
	)

	cs, err := scanControlStatement(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("querying control statement %d: %w", id, err)
	}
	return cs, nil
}

// List retrieves all control statements for a given year.
func (r *VATControlStatementRepository) List(ctx context.Context, year int) ([]domain.VATControlStatement, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, year, month, filing_type, xml_data, status, filed_at, created_at, updated_at
		FROM vat_control_statements WHERE year = ?
		ORDER BY month ASC`, year,
	)
	if err != nil {
		return nil, fmt.Errorf("listing control statements for year %d: %w", year, err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.VATControlStatement
	for rows.Next() {
		cs, err := scanControlStatement(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning control statement row: %w", err)
		}
		result = append(result, *cs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating control statement rows: %w", err)
	}
	return result, nil
}

// GetByPeriod retrieves a control statement for a specific period and filing type.
func (r *VATControlStatementRepository) GetByPeriod(ctx context.Context, year, month int, filingType string) (*domain.VATControlStatement, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, year, month, filing_type, xml_data, status, filed_at, created_at, updated_at
		FROM vat_control_statements WHERE year = ? AND month = ? AND filing_type = ?`,
		year, month, filingType,
	)

	cs, err := scanControlStatement(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("querying control statement for %d/%d: %w", year, month, err)
	}
	return cs, nil
}

// CreateLines inserts multiple control statement lines.
func (r *VATControlStatementRepository) CreateLines(ctx context.Context, lines []domain.VATControlStatementLine) error {
	if len(lines) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction for control statement lines: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO vat_control_statement_lines (
			control_statement_id, section, partner_dic, document_number,
			dppd, base, vat, vat_rate_percent, invoice_id, expense_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("preparing control statement line insert: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for i, line := range lines {
		result, err := stmt.ExecContext(ctx,
			line.ControlStatementID, line.Section, line.PartnerDIC, line.DocumentNumber,
			line.DPPD, line.Base, line.VAT, line.VATRatePercent,
			line.InvoiceID, line.ExpenseID,
		)
		if err != nil {
			return fmt.Errorf("inserting control statement line %d: %w", i, err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("getting last insert id for control statement line %d: %w", i, err)
		}
		lines[i].ID = id
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing control statement lines: %w", err)
	}
	return nil
}

// DeleteLines removes all lines for a given control statement.
func (r *VATControlStatementRepository) DeleteLines(ctx context.Context, controlStatementID int64) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM vat_control_statement_lines WHERE control_statement_id = ?`, controlStatementID,
	)
	if err != nil {
		return fmt.Errorf("deleting control statement lines for %d: %w", controlStatementID, err)
	}
	return nil
}

// GetLines retrieves all lines for a given control statement.
func (r *VATControlStatementRepository) GetLines(ctx context.Context, controlStatementID int64) ([]domain.VATControlStatementLine, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, control_statement_id, section, partner_dic, document_number,
			dppd, base, vat, vat_rate_percent, invoice_id, expense_id
		FROM vat_control_statement_lines WHERE control_statement_id = ?
		ORDER BY section ASC, id ASC`, controlStatementID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying control statement lines for %d: %w", controlStatementID, err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.VATControlStatementLine
	for rows.Next() {
		line, err := scanControlStatementLine(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning control statement line: %w", err)
		}
		result = append(result, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating control statement line rows: %w", err)
	}
	return result, nil
}
