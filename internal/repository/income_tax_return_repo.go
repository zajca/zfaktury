package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// IncomeTaxReturnRepository handles persistence of IncomeTaxReturn entities.
type IncomeTaxReturnRepository struct {
	db *sql.DB
}

// NewIncomeTaxReturnRepository creates a new IncomeTaxReturnRepository.
func NewIncomeTaxReturnRepository(db *sql.DB) *IncomeTaxReturnRepository {
	return &IncomeTaxReturnRepository{db: db}
}

// incomeTaxReturnColumns is the list of columns to select for an IncomeTaxReturn row.
const incomeTaxReturnColumns = `id, year, filing_type,
	total_revenue, actual_expenses, flat_rate_percent, flat_rate_amount, used_expenses,
	tax_base, total_deductions, tax_base_rounded, tax_at_15, tax_at_23, total_tax,
	credit_basic, credit_spouse, credit_disability, credit_student, total_credits,
	tax_after_credits, child_benefit, tax_after_benefit,
	prepayments, tax_due,
	capital_income_gross, capital_income_tax, capital_income_net,
	other_income_gross, other_income_expenses, other_income_exempt, other_income_net,
	deduction_mortgage, deduction_life_insurance, deduction_pension, deduction_donation, deduction_union_dues,
	xml_data, status, filed_at, created_at, updated_at`

// scanIncomeTaxReturn scans an IncomeTaxReturn from a row.
func scanIncomeTaxReturn(s scanner) (*domain.IncomeTaxReturn, error) {
	itr := &domain.IncomeTaxReturn{}
	var filedAtStr sql.NullString
	var createdAtStr, updatedAtStr string
	var xmlData []byte

	err := s.Scan(
		&itr.ID, &itr.Year, &itr.FilingType,
		&itr.TotalRevenue, &itr.ActualExpenses, &itr.FlatRatePercent, &itr.FlatRateAmount, &itr.UsedExpenses,
		&itr.TaxBase, &itr.TotalDeductions, &itr.TaxBaseRounded, &itr.TaxAt15, &itr.TaxAt23, &itr.TotalTax,
		&itr.CreditBasic, &itr.CreditSpouse, &itr.CreditDisability, &itr.CreditStudent, &itr.TotalCredits,
		&itr.TaxAfterCredits, &itr.ChildBenefit, &itr.TaxAfterBenefit,
		&itr.Prepayments, &itr.TaxDue,
		&itr.CapitalIncomeGross, &itr.CapitalIncomeTax, &itr.CapitalIncomeNet,
		&itr.OtherIncomeGross, &itr.OtherIncomeExpenses, &itr.OtherIncomeExempt, &itr.OtherIncomeNet,
		&itr.DeductionMortgage, &itr.DeductionLifeInsurance, &itr.DeductionPension, &itr.DeductionDonation, &itr.DeductionUnionDues,
		&xmlData, &itr.Status, &filedAtStr,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	itr.XMLData = xmlData

	itr.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning income_tax_return created_at: %w", err)
	}
	itr.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning income_tax_return updated_at: %w", err)
	}
	itr.FiledAt, err = parseDatePtr(time.RFC3339, filedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning income_tax_return filed_at: %w", err)
	}

	return itr, nil
}

// Create inserts a new income tax return into the database.
func (r *IncomeTaxReturnRepository) Create(ctx context.Context, itr *domain.IncomeTaxReturn) error {
	now := time.Now()
	itr.CreatedAt = now
	itr.UpdatedAt = now
	if itr.Status == "" {
		itr.Status = domain.FilingStatusDraft
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO income_tax_returns (
			year, filing_type,
			total_revenue, actual_expenses, flat_rate_percent, flat_rate_amount, used_expenses,
			tax_base, total_deductions, tax_base_rounded, tax_at_15, tax_at_23, total_tax,
			credit_basic, credit_spouse, credit_disability, credit_student, total_credits,
			tax_after_credits, child_benefit, tax_after_benefit,
			prepayments, tax_due,
			capital_income_gross, capital_income_tax, capital_income_net,
			other_income_gross, other_income_expenses, other_income_exempt, other_income_net,
			deduction_mortgage, deduction_life_insurance, deduction_pension, deduction_donation, deduction_union_dues,
			xml_data, status, filed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		itr.Year, itr.FilingType,
		itr.TotalRevenue, itr.ActualExpenses, itr.FlatRatePercent, itr.FlatRateAmount, itr.UsedExpenses,
		itr.TaxBase, itr.TotalDeductions, itr.TaxBaseRounded, itr.TaxAt15, itr.TaxAt23, itr.TotalTax,
		itr.CreditBasic, itr.CreditSpouse, itr.CreditDisability, itr.CreditStudent, itr.TotalCredits,
		itr.TaxAfterCredits, itr.ChildBenefit, itr.TaxAfterBenefit,
		itr.Prepayments, itr.TaxDue,
		itr.CapitalIncomeGross, itr.CapitalIncomeTax, itr.CapitalIncomeNet,
		itr.OtherIncomeGross, itr.OtherIncomeExpenses, itr.OtherIncomeExempt, itr.OtherIncomeNet,
		itr.DeductionMortgage, itr.DeductionLifeInsurance, itr.DeductionPension, itr.DeductionDonation, itr.DeductionUnionDues,
		nil, itr.Status, nil,
		itr.CreatedAt.Format(time.RFC3339), itr.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting income_tax_return: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for income_tax_return: %w", err)
	}
	itr.ID = id
	return nil
}

// Update modifies an existing income tax return.
func (r *IncomeTaxReturnRepository) Update(ctx context.Context, itr *domain.IncomeTaxReturn) error {
	itr.UpdatedAt = time.Now()

	var filedAt any
	if itr.FiledAt != nil {
		filedAt = itr.FiledAt.Format(time.RFC3339)
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE income_tax_returns SET
			year = ?, filing_type = ?,
			total_revenue = ?, actual_expenses = ?, flat_rate_percent = ?, flat_rate_amount = ?, used_expenses = ?,
			tax_base = ?, total_deductions = ?, tax_base_rounded = ?, tax_at_15 = ?, tax_at_23 = ?, total_tax = ?,
			credit_basic = ?, credit_spouse = ?, credit_disability = ?, credit_student = ?, total_credits = ?,
			tax_after_credits = ?, child_benefit = ?, tax_after_benefit = ?,
			prepayments = ?, tax_due = ?,
			capital_income_gross = ?, capital_income_tax = ?, capital_income_net = ?,
			other_income_gross = ?, other_income_expenses = ?, other_income_exempt = ?, other_income_net = ?,
			deduction_mortgage = ?, deduction_life_insurance = ?, deduction_pension = ?, deduction_donation = ?, deduction_union_dues = ?,
			xml_data = ?, status = ?, filed_at = ?, updated_at = ?
		WHERE id = ?`,
		itr.Year, itr.FilingType,
		itr.TotalRevenue, itr.ActualExpenses, itr.FlatRatePercent, itr.FlatRateAmount, itr.UsedExpenses,
		itr.TaxBase, itr.TotalDeductions, itr.TaxBaseRounded, itr.TaxAt15, itr.TaxAt23, itr.TotalTax,
		itr.CreditBasic, itr.CreditSpouse, itr.CreditDisability, itr.CreditStudent, itr.TotalCredits,
		itr.TaxAfterCredits, itr.ChildBenefit, itr.TaxAfterBenefit,
		itr.Prepayments, itr.TaxDue,
		itr.CapitalIncomeGross, itr.CapitalIncomeTax, itr.CapitalIncomeNet,
		itr.OtherIncomeGross, itr.OtherIncomeExpenses, itr.OtherIncomeExempt, itr.OtherIncomeNet,
		itr.DeductionMortgage, itr.DeductionLifeInsurance, itr.DeductionPension, itr.DeductionDonation, itr.DeductionUnionDues,
		itr.XMLData, itr.Status, filedAt,
		itr.UpdatedAt.Format(time.RFC3339), itr.ID,
	)
	if err != nil {
		return fmt.Errorf("updating income_tax_return %d: %w", itr.ID, err)
	}
	return nil
}

// Delete removes an income tax return by ID.
func (r *IncomeTaxReturnRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM income_tax_returns WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting income_tax_return %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for income_tax_return %d delete: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("income_tax_return %d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// GetByID retrieves an income tax return by its ID.
func (r *IncomeTaxReturnRepository) GetByID(ctx context.Context, id int64) (*domain.IncomeTaxReturn, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+incomeTaxReturnColumns+` FROM income_tax_returns WHERE id = ?`, id)
	itr, err := scanIncomeTaxReturn(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("income_tax_return %d: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying income_tax_return %d: %w", id, err)
	}
	return itr, nil
}

// List retrieves all income tax returns for a given year.
func (r *IncomeTaxReturnRepository) List(ctx context.Context, year int) ([]domain.IncomeTaxReturn, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+incomeTaxReturnColumns+` FROM income_tax_returns WHERE year = ? ORDER BY created_at ASC`,
		year,
	)
	if err != nil {
		return nil, fmt.Errorf("listing income_tax_returns for year %d: %w", year, err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.IncomeTaxReturn
	for rows.Next() {
		itr, err := scanIncomeTaxReturn(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning income_tax_return row: %w", err)
		}
		result = append(result, *itr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating income_tax_return rows: %w", err)
	}
	return result, nil
}

// GetByYear retrieves an income tax return for a specific year and filing type.
func (r *IncomeTaxReturnRepository) GetByYear(ctx context.Context, year int, filingType string) (*domain.IncomeTaxReturn, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+incomeTaxReturnColumns+` FROM income_tax_returns WHERE year = ? AND filing_type = ?`,
		year, filingType,
	)
	itr, err := scanIncomeTaxReturn(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("income_tax_return for year %d (%s): %w", year, filingType, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying income_tax_return by year: %w", err)
	}
	return itr, nil
}

// LinkInvoices associates invoices with an income tax return via the junction table.
func (r *IncomeTaxReturnRepository) LinkInvoices(ctx context.Context, incomeTaxReturnID int64, invoiceIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction for linking invoices: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `DELETE FROM income_tax_return_invoices WHERE income_tax_return_id = ?`, incomeTaxReturnID)
	if err != nil {
		return fmt.Errorf("clearing existing invoice links for income_tax_return %d: %w", incomeTaxReturnID, err)
	}

	for _, invID := range invoiceIDs {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO income_tax_return_invoices (income_tax_return_id, invoice_id) VALUES (?, ?)`,
			incomeTaxReturnID, invID,
		)
		if err != nil {
			return fmt.Errorf("linking invoice %d to income_tax_return %d: %w", invID, incomeTaxReturnID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing invoice links for income_tax_return %d: %w", incomeTaxReturnID, err)
	}
	return nil
}

// LinkExpenses associates expenses with an income tax return via the junction table.
func (r *IncomeTaxReturnRepository) LinkExpenses(ctx context.Context, incomeTaxReturnID int64, expenseIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction for linking expenses: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `DELETE FROM income_tax_return_expenses WHERE income_tax_return_id = ?`, incomeTaxReturnID)
	if err != nil {
		return fmt.Errorf("clearing existing expense links for income_tax_return %d: %w", incomeTaxReturnID, err)
	}

	for _, expID := range expenseIDs {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO income_tax_return_expenses (income_tax_return_id, expense_id) VALUES (?, ?)`,
			incomeTaxReturnID, expID,
		)
		if err != nil {
			return fmt.Errorf("linking expense %d to income_tax_return %d: %w", expID, incomeTaxReturnID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing expense links for income_tax_return %d: %w", incomeTaxReturnID, err)
	}
	return nil
}

// GetLinkedInvoiceIDs returns invoice IDs linked to an income tax return.
func (r *IncomeTaxReturnRepository) GetLinkedInvoiceIDs(ctx context.Context, incomeTaxReturnID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT invoice_id FROM income_tax_return_invoices WHERE income_tax_return_id = ? ORDER BY invoice_id`,
		incomeTaxReturnID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying linked invoices for income_tax_return %d: %w", incomeTaxReturnID, err)
	}
	defer func() { _ = rows.Close() }()

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

// GetLinkedExpenseIDs returns expense IDs linked to an income tax return.
func (r *IncomeTaxReturnRepository) GetLinkedExpenseIDs(ctx context.Context, incomeTaxReturnID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT expense_id FROM income_tax_return_expenses WHERE income_tax_return_id = ? ORDER BY expense_id`,
		incomeTaxReturnID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying linked expenses for income_tax_return %d: %w", incomeTaxReturnID, err)
	}
	defer func() { _ = rows.Close() }()

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
