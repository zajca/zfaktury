package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// VATReturnRepository handles persistence of VATReturn entities.
type VATReturnRepository struct {
	db *sql.DB
}

// NewVATReturnRepository creates a new VATReturnRepository.
func NewVATReturnRepository(db *sql.DB) *VATReturnRepository {
	return &VATReturnRepository{db: db}
}

// vatReturnColumns is the list of columns to select for a VATReturn row.
const vatReturnColumns = `id, year, month, quarter, filing_type,
	output_vat_base_21, output_vat_amount_21, output_vat_base_12, output_vat_amount_12, output_vat_base_0,
	reverse_charge_base_21, reverse_charge_amount_21, reverse_charge_base_12, reverse_charge_amount_12,
	input_vat_base_21, input_vat_amount_21, input_vat_base_12, input_vat_amount_12,
	total_output_vat, total_input_vat, net_vat,
	xml_data, status, filed_at, created_at, updated_at`

// scanVATReturn scans a VATReturn from a row.
func scanVATReturn(s scanner) (*domain.VATReturn, error) {
	vr := &domain.VATReturn{}
	var filedAtStr sql.NullString
	var createdAtStr, updatedAtStr string
	var xmlData []byte

	err := s.Scan(
		&vr.ID, &vr.Period.Year, &vr.Period.Month, &vr.Period.Quarter, &vr.FilingType,
		&vr.OutputVATBase21, &vr.OutputVATAmount21, &vr.OutputVATBase12, &vr.OutputVATAmount12, &vr.OutputVATBase0,
		&vr.ReverseChargeBase21, &vr.ReverseChargeAmount21, &vr.ReverseChargeBase12, &vr.ReverseChargeAmount12,
		&vr.InputVATBase21, &vr.InputVATAmount21, &vr.InputVATBase12, &vr.InputVATAmount12,
		&vr.TotalOutputVAT, &vr.TotalInputVAT, &vr.NetVAT,
		&xmlData, &vr.Status, &filedAtStr,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	vr.XMLData = xmlData

	vr.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning vat_return created_at: %w", err)
	}
	vr.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning vat_return updated_at: %w", err)
	}
	vr.FiledAt, err = parseDatePtr(time.RFC3339, filedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning vat_return filed_at: %w", err)
	}

	return vr, nil
}

// Create inserts a new VAT return into the database.
func (r *VATReturnRepository) Create(ctx context.Context, vr *domain.VATReturn) error {
	now := time.Now()
	vr.CreatedAt = now
	vr.UpdatedAt = now
	if vr.Status == "" {
		vr.Status = domain.FilingStatusDraft
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO vat_returns (
			year, month, quarter, filing_type,
			output_vat_base_21, output_vat_amount_21, output_vat_base_12, output_vat_amount_12, output_vat_base_0,
			reverse_charge_base_21, reverse_charge_amount_21, reverse_charge_base_12, reverse_charge_amount_12,
			input_vat_base_21, input_vat_amount_21, input_vat_base_12, input_vat_amount_12,
			total_output_vat, total_input_vat, net_vat,
			xml_data, status, filed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		vr.Period.Year, vr.Period.Month, vr.Period.Quarter, vr.FilingType,
		vr.OutputVATBase21, vr.OutputVATAmount21, vr.OutputVATBase12, vr.OutputVATAmount12, vr.OutputVATBase0,
		vr.ReverseChargeBase21, vr.ReverseChargeAmount21, vr.ReverseChargeBase12, vr.ReverseChargeAmount12,
		vr.InputVATBase21, vr.InputVATAmount21, vr.InputVATBase12, vr.InputVATAmount12,
		vr.TotalOutputVAT, vr.TotalInputVAT, vr.NetVAT,
		nil, vr.Status, nil,
		vr.CreatedAt.Format(time.RFC3339), vr.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting vat_return: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for vat_return: %w", err)
	}
	vr.ID = id
	return nil
}

// Update modifies an existing VAT return.
func (r *VATReturnRepository) Update(ctx context.Context, vr *domain.VATReturn) error {
	vr.UpdatedAt = time.Now()

	var filedAt any
	if vr.FiledAt != nil {
		filedAt = vr.FiledAt.Format(time.RFC3339)
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE vat_returns SET
			year = ?, month = ?, quarter = ?, filing_type = ?,
			output_vat_base_21 = ?, output_vat_amount_21 = ?,
			output_vat_base_12 = ?, output_vat_amount_12 = ?,
			output_vat_base_0 = ?,
			reverse_charge_base_21 = ?, reverse_charge_amount_21 = ?,
			reverse_charge_base_12 = ?, reverse_charge_amount_12 = ?,
			input_vat_base_21 = ?, input_vat_amount_21 = ?,
			input_vat_base_12 = ?, input_vat_amount_12 = ?,
			total_output_vat = ?, total_input_vat = ?, net_vat = ?,
			xml_data = ?, status = ?, filed_at = ?, updated_at = ?
		WHERE id = ?`,
		vr.Period.Year, vr.Period.Month, vr.Period.Quarter, vr.FilingType,
		vr.OutputVATBase21, vr.OutputVATAmount21,
		vr.OutputVATBase12, vr.OutputVATAmount12,
		vr.OutputVATBase0,
		vr.ReverseChargeBase21, vr.ReverseChargeAmount21,
		vr.ReverseChargeBase12, vr.ReverseChargeAmount12,
		vr.InputVATBase21, vr.InputVATAmount21,
		vr.InputVATBase12, vr.InputVATAmount12,
		vr.TotalOutputVAT, vr.TotalInputVAT, vr.NetVAT,
		vr.XMLData, vr.Status, filedAt,
		vr.UpdatedAt.Format(time.RFC3339), vr.ID,
	)
	if err != nil {
		return fmt.Errorf("updating vat_return %d: %w", vr.ID, err)
	}
	return nil
}

// Delete removes a VAT return by ID.
func (r *VATReturnRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM vat_returns WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting vat_return %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for vat_return %d delete: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("vat_return %d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// GetByID retrieves a VAT return by its ID.
func (r *VATReturnRepository) GetByID(ctx context.Context, id int64) (*domain.VATReturn, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+vatReturnColumns+` FROM vat_returns WHERE id = ?`, id)
	vr, err := scanVATReturn(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("vat_return %d: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying vat_return %d: %w", id, err)
	}
	return vr, nil
}

// List retrieves all VAT returns for a given year.
func (r *VATReturnRepository) List(ctx context.Context, year int) ([]domain.VATReturn, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+vatReturnColumns+` FROM vat_returns WHERE year = ? ORDER BY month ASC, quarter ASC`,
		year,
	)
	if err != nil {
		return nil, fmt.Errorf("listing vat_returns for year %d: %w", year, err)
	}
	defer rows.Close()

	var result []domain.VATReturn
	for rows.Next() {
		vr, err := scanVATReturn(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning vat_return row: %w", err)
		}
		result = append(result, *vr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating vat_return rows: %w", err)
	}
	return result, nil
}

// GetByPeriod retrieves a VAT return for a specific period.
func (r *VATReturnRepository) GetByPeriod(ctx context.Context, year, month, quarter int, filingType string) (*domain.VATReturn, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+vatReturnColumns+` FROM vat_returns WHERE year = ? AND month = ? AND quarter = ? AND filing_type = ?`,
		year, month, quarter, filingType,
	)
	vr, err := scanVATReturn(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("vat_return for period %d/%d/Q%d (%s): %w", year, month, quarter, filingType, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying vat_return by period: %w", err)
	}
	return vr, nil
}

// LinkInvoices associates invoices with a VAT return via the junction table.
func (r *VATReturnRepository) LinkInvoices(ctx context.Context, vatReturnID int64, invoiceIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction for linking invoices: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `DELETE FROM vat_return_invoices WHERE vat_return_id = ?`, vatReturnID)
	if err != nil {
		return fmt.Errorf("clearing existing invoice links for vat_return %d: %w", vatReturnID, err)
	}

	for _, invID := range invoiceIDs {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO vat_return_invoices (vat_return_id, invoice_id) VALUES (?, ?)`,
			vatReturnID, invID,
		)
		if err != nil {
			return fmt.Errorf("linking invoice %d to vat_return %d: %w", invID, vatReturnID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing invoice links for vat_return %d: %w", vatReturnID, err)
	}
	return nil
}

// LinkExpenses associates expenses with a VAT return via the junction table.
func (r *VATReturnRepository) LinkExpenses(ctx context.Context, vatReturnID int64, expenseIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction for linking expenses: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `DELETE FROM vat_return_expenses WHERE vat_return_id = ?`, vatReturnID)
	if err != nil {
		return fmt.Errorf("clearing existing expense links for vat_return %d: %w", vatReturnID, err)
	}

	for _, expID := range expenseIDs {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO vat_return_expenses (vat_return_id, expense_id) VALUES (?, ?)`,
			vatReturnID, expID,
		)
		if err != nil {
			return fmt.Errorf("linking expense %d to vat_return %d: %w", expID, vatReturnID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing expense links for vat_return %d: %w", vatReturnID, err)
	}
	return nil
}

// GetLinkedInvoiceIDs returns invoice IDs linked to a VAT return.
func (r *VATReturnRepository) GetLinkedInvoiceIDs(ctx context.Context, vatReturnID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT invoice_id FROM vat_return_invoices WHERE vat_return_id = ? ORDER BY invoice_id`,
		vatReturnID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying linked invoices for vat_return %d: %w", vatReturnID, err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning linked invoice id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating linked invoice ids: %w", err)
	}
	return ids, nil
}

// GetLinkedExpenseIDs returns expense IDs linked to a VAT return.
func (r *VATReturnRepository) GetLinkedExpenseIDs(ctx context.Context, vatReturnID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT expense_id FROM vat_return_expenses WHERE vat_return_id = ? ORDER BY expense_id`,
		vatReturnID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying linked expenses for vat_return %d: %w", vatReturnID, err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning linked expense id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating linked expense ids: %w", err)
	}
	return ids, nil
}
