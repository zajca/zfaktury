package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// RecurringExpenseRepository handles persistence of RecurringExpense entities.
type RecurringExpenseRepository struct {
	db *sql.DB
}

// NewRecurringExpenseRepository creates a new RecurringExpenseRepository.
func NewRecurringExpenseRepository(db *sql.DB) *RecurringExpenseRepository {
	return &RecurringExpenseRepository{db: db}
}

// Create inserts a new recurring expense into the database.
func (r *RecurringExpenseRepository) Create(ctx context.Context, re *domain.RecurringExpense) error {
	now := time.Now()
	re.CreatedAt = now
	re.UpdatedAt = now

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO recurring_expenses (
			name, vendor_id, category, description,
			amount, currency_code, exchange_rate,
			vat_rate_percent, vat_amount,
			is_tax_deductible, business_percent, payment_method,
			notes, frequency, next_issue_date, end_date, is_active,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		re.Name, re.VendorID, re.Category, re.Description,
		re.Amount, re.CurrencyCode, re.ExchangeRate,
		re.VATRatePercent, re.VATAmount,
		re.IsTaxDeductible, re.BusinessPercent, re.PaymentMethod,
		re.Notes, re.Frequency, re.NextIssueDate.Format("2006-01-02"),
		formatNullableDate(re.EndDate), re.IsActive,
		re.CreatedAt.Format(time.RFC3339), re.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting recurring expense: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for recurring expense: %w", err)
	}
	re.ID = id
	return nil
}

// Update modifies an existing recurring expense.
func (r *RecurringExpenseRepository) Update(ctx context.Context, re *domain.RecurringExpense) error {
	re.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		UPDATE recurring_expenses SET
			name = ?, vendor_id = ?, category = ?, description = ?,
			amount = ?, currency_code = ?, exchange_rate = ?,
			vat_rate_percent = ?, vat_amount = ?,
			is_tax_deductible = ?, business_percent = ?, payment_method = ?,
			notes = ?, frequency = ?, next_issue_date = ?, end_date = ?, is_active = ?,
			updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`,
		re.Name, re.VendorID, re.Category, re.Description,
		re.Amount, re.CurrencyCode, re.ExchangeRate,
		re.VATRatePercent, re.VATAmount,
		re.IsTaxDeductible, re.BusinessPercent, re.PaymentMethod,
		re.Notes, re.Frequency, re.NextIssueDate.Format("2006-01-02"),
		formatNullableDate(re.EndDate), re.IsActive,
		re.UpdatedAt.Format(time.RFC3339), re.ID,
	)
	if err != nil {
		return fmt.Errorf("updating recurring expense %d: %w", re.ID, err)
	}
	return nil
}

// Delete performs a soft delete on a recurring expense.
func (r *RecurringExpenseRepository) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	nowStr := now.Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx, `
		UPDATE recurring_expenses SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		nowStr, nowStr, id,
	)
	if err != nil {
		return fmt.Errorf("soft-deleting recurring expense %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for recurring expense %d: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("recurring expense %d not found or already deleted", id)
	}
	return nil
}

// GetByID retrieves a single recurring expense by ID, including vendor data if available.
func (r *RecurringExpenseRepository) GetByID(ctx context.Context, id int64) (*domain.RecurringExpense, error) {
	re := &domain.RecurringExpense{}
	var nextIssueDateStr string
	var endDateStr sql.NullString
	var createdAtStr string
	var updatedAtStr string
	var deletedAtStr sql.NullString
	var vendorID sql.NullInt64
	var vendorName sql.NullString
	var vendorType sql.NullString
	var vendorICO sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT
			re.id, re.name, re.vendor_id, re.category, re.description,
			re.amount, re.currency_code, re.exchange_rate,
			re.vat_rate_percent, re.vat_amount,
			re.is_tax_deductible, re.business_percent, re.payment_method,
			re.notes, re.frequency, re.next_issue_date, re.end_date, re.is_active,
			re.created_at, re.updated_at, re.deleted_at,
			c.name, c.type, c.ico
		FROM recurring_expenses re
		LEFT JOIN contacts c ON c.id = re.vendor_id
		WHERE re.id = ? AND re.deleted_at IS NULL`, id,
	).Scan(
		&re.ID, &re.Name, &vendorID, &re.Category, &re.Description,
		&re.Amount, &re.CurrencyCode, &re.ExchangeRate,
		&re.VATRatePercent, &re.VATAmount,
		&re.IsTaxDeductible, &re.BusinessPercent, &re.PaymentMethod,
		&re.Notes, &re.Frequency, &nextIssueDateStr, &endDateStr, &re.IsActive,
		&createdAtStr, &updatedAtStr, &deletedAtStr,
		&vendorName, &vendorType, &vendorICO,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("recurring expense %d not found: %w", id, err)
		}
		return nil, fmt.Errorf("querying recurring expense %d: %w", id, err)
	}

	re.NextIssueDate, err = parseDate(time.DateOnly, nextIssueDateStr)
	if err != nil {
		return nil, fmt.Errorf("scanning recurring expense: %w", err)
	}
	re.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning recurring expense: %w", err)
	}
	re.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning recurring expense: %w", err)
	}
	re.EndDate, err = parseDatePtr(time.DateOnly, endDateStr)
	if err != nil {
		return nil, fmt.Errorf("scanning recurring expense: %w", err)
	}
	re.DeletedAt, err = parseDatePtr(time.RFC3339, deletedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning recurring expense: %w", err)
	}
	if vendorID.Valid {
		vid := vendorID.Int64
		re.VendorID = &vid
		if vendorName.Valid {
			re.Vendor = &domain.Contact{
				ID:   vid,
				Name: vendorName.String,
				Type: vendorType.String,
				ICO:  vendorICO.String,
			}
		}
	}

	return re, nil
}

// List retrieves all non-deleted recurring expenses with pagination.
func (r *RecurringExpenseRepository) List(ctx context.Context, limit, offset int) ([]domain.RecurringExpense, int, error) {
	// Count.
	var total int
	if err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM recurring_expenses WHERE deleted_at IS NULL",
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting recurring expenses: %w", err)
	}

	query := `SELECT
			re.id, re.name, re.vendor_id, re.category, re.description,
			re.amount, re.currency_code, re.exchange_rate,
			re.vat_rate_percent, re.vat_amount,
			re.is_tax_deductible, re.business_percent, re.payment_method,
			re.notes, re.frequency, re.next_issue_date, re.end_date, re.is_active,
			re.created_at, re.updated_at, re.deleted_at,
			COALESCE(c.name, '') AS vendor_name
		FROM recurring_expenses re
		LEFT JOIN contacts c ON c.id = re.vendor_id
		WHERE re.deleted_at IS NULL
		ORDER BY re.next_issue_date ASC`

	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		return r.scanList(ctx, query, total, limit, offset)
	}

	return r.scanList(ctx, query, total)
}

// ListActive retrieves all active, non-deleted recurring expenses.
func (r *RecurringExpenseRepository) ListActive(ctx context.Context) ([]domain.RecurringExpense, error) {
	query := `SELECT
			re.id, re.name, re.vendor_id, re.category, re.description,
			re.amount, re.currency_code, re.exchange_rate,
			re.vat_rate_percent, re.vat_amount,
			re.is_tax_deductible, re.business_percent, re.payment_method,
			re.notes, re.frequency, re.next_issue_date, re.end_date, re.is_active,
			re.created_at, re.updated_at, re.deleted_at,
			COALESCE(c.name, '') AS vendor_name
		FROM recurring_expenses re
		LEFT JOIN contacts c ON c.id = re.vendor_id
		WHERE re.deleted_at IS NULL AND re.is_active = 1
		ORDER BY re.next_issue_date ASC`

	items, _, err := r.scanList(ctx, query, 0)
	return items, err
}

// ListDue retrieves all active, non-deleted recurring expenses where next_issue_date <= asOfDate.
func (r *RecurringExpenseRepository) ListDue(ctx context.Context, asOfDate time.Time) ([]domain.RecurringExpense, error) {
	query := `SELECT
			re.id, re.name, re.vendor_id, re.category, re.description,
			re.amount, re.currency_code, re.exchange_rate,
			re.vat_rate_percent, re.vat_amount,
			re.is_tax_deductible, re.business_percent, re.payment_method,
			re.notes, re.frequency, re.next_issue_date, re.end_date, re.is_active,
			re.created_at, re.updated_at, re.deleted_at,
			COALESCE(c.name, '') AS vendor_name
		FROM recurring_expenses re
		LEFT JOIN contacts c ON c.id = re.vendor_id
		WHERE re.deleted_at IS NULL AND re.is_active = 1 AND re.next_issue_date <= ?
		ORDER BY re.next_issue_date ASC`

	rows, err := r.db.QueryContext(ctx, query, asOfDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("listing due recurring expenses: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return r.scanRows(rows)
}

// Deactivate sets is_active = 0 for the given recurring expense.
func (r *RecurringExpenseRepository) Deactivate(ctx context.Context, id int64) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
		UPDATE recurring_expenses SET is_active = 0, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		now.Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("deactivating recurring expense %d: %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for recurring expense %d deactivation: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("recurring expense %d not found or already deleted", id)
	}
	return nil
}

// Activate sets is_active = 1 for the given recurring expense.
func (r *RecurringExpenseRepository) Activate(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE recurring_expenses SET is_active = 1, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		now.Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("activating recurring expense %d: %w", id, err)
	}
	return nil
}

// scanList executes the query and scans the results into a list.
func (r *RecurringExpenseRepository) scanList(ctx context.Context, query string, total int, args ...any) ([]domain.RecurringExpense, int, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing recurring expenses: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items, err := r.scanRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// scanRows scans recurring expense rows from a query result.
func (r *RecurringExpenseRepository) scanRows(rows *sql.Rows) ([]domain.RecurringExpense, error) {
	var items []domain.RecurringExpense
	var err error
	for rows.Next() {
		var re domain.RecurringExpense
		var nextIssueDateStr string
		var endDateStr sql.NullString
		var createdAtStr string
		var updatedAtStr string
		var deletedAtStr sql.NullString
		var vendorID sql.NullInt64
		var vendorName string

		if err := rows.Scan(
			&re.ID, &re.Name, &vendorID, &re.Category, &re.Description,
			&re.Amount, &re.CurrencyCode, &re.ExchangeRate,
			&re.VATRatePercent, &re.VATAmount,
			&re.IsTaxDeductible, &re.BusinessPercent, &re.PaymentMethod,
			&re.Notes, &re.Frequency, &nextIssueDateStr, &endDateStr, &re.IsActive,
			&createdAtStr, &updatedAtStr, &deletedAtStr,
			&vendorName,
		); err != nil {
			return nil, fmt.Errorf("scanning recurring expense row: %w", err)
		}

		re.NextIssueDate, err = parseDate(time.DateOnly, nextIssueDateStr)
		if err != nil {
			return nil, fmt.Errorf("scanning recurring expense row: %w", err)
		}
		re.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("scanning recurring expense row: %w", err)
		}
		re.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("scanning recurring expense row: %w", err)
		}
		re.EndDate, err = parseDatePtr(time.DateOnly, endDateStr)
		if err != nil {
			return nil, fmt.Errorf("scanning recurring expense row: %w", err)
		}
		re.DeletedAt, err = parseDatePtr(time.RFC3339, deletedAtStr)
		if err != nil {
			return nil, fmt.Errorf("scanning recurring expense row: %w", err)
		}
		if vendorID.Valid {
			vid := vendorID.Int64
			re.VendorID = &vid
			if vendorName != "" {
				re.Vendor = &domain.Contact{ID: vid, Name: vendorName}
			}
		}

		items = append(items, re)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating recurring expense rows: %w", err)
	}
	return items, nil
}

// formatNullableDate formats a nullable time pointer as a date string for SQL.
func formatNullableDate(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}
