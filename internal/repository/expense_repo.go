package repository

import (
	"errors"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// ExpenseRepository handles persistence of Expense entities.
type ExpenseRepository struct {
	db *sql.DB
}

// NewExpenseRepository creates a new ExpenseRepository.
func NewExpenseRepository(db *sql.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

// Create inserts a new expense into the database.
func (r *ExpenseRepository) Create(ctx context.Context, e *domain.Expense) error {
	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO expenses (
			vendor_id, expense_number, category, description,
			issue_date, amount, currency_code, exchange_rate,
			vat_rate_percent, vat_amount,
			is_tax_deductible, business_percent, payment_method,
			document_path, notes,
			tax_reviewed_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.VendorID, e.ExpenseNumber, e.Category, e.Description,
		e.IssueDate.Format("2006-01-02"), e.Amount, e.CurrencyCode, e.ExchangeRate,
		e.VATRatePercent, e.VATAmount,
		e.IsTaxDeductible, e.BusinessPercent, e.PaymentMethod,
		e.DocumentPath, e.Notes,
		nil,
		e.CreatedAt.Format(time.RFC3339), e.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting expense: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for expense: %w", err)
	}
	e.ID = id
	return nil
}

// Update modifies an existing expense.
func (r *ExpenseRepository) Update(ctx context.Context, e *domain.Expense) error {
	e.UpdatedAt = time.Now()

	var taxReviewedAt interface{}
	if e.TaxReviewedAt != nil {
		taxReviewedAt = e.TaxReviewedAt.Format(time.RFC3339)
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE expenses SET
			vendor_id = ?, expense_number = ?, category = ?, description = ?,
			issue_date = ?, amount = ?, currency_code = ?, exchange_rate = ?,
			vat_rate_percent = ?, vat_amount = ?,
			is_tax_deductible = ?, business_percent = ?, payment_method = ?,
			document_path = ?, notes = ?,
			tax_reviewed_at = ?,
			updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`,
		e.VendorID, e.ExpenseNumber, e.Category, e.Description,
		e.IssueDate.Format("2006-01-02"), e.Amount, e.CurrencyCode, e.ExchangeRate,
		e.VATRatePercent, e.VATAmount,
		e.IsTaxDeductible, e.BusinessPercent, e.PaymentMethod,
		e.DocumentPath, e.Notes,
		taxReviewedAt,
		e.UpdatedAt.Format(time.RFC3339), e.ID,
	)
	if err != nil {
		return fmt.Errorf("updating expense %d: %w", e.ID, err)
	}
	return nil
}

// Delete performs a soft delete on an expense.
func (r *ExpenseRepository) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	nowStr := now.Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx, `
		UPDATE expenses SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		nowStr, nowStr, id,
	)
	if err != nil {
		return fmt.Errorf("soft-deleting expense %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for expense %d: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("expense %d not found or already deleted", id)
	}
	return nil
}

// GetByID retrieves a single expense by ID, including vendor data if available.
func (r *ExpenseRepository) GetByID(ctx context.Context, id int64) (*domain.Expense, error) {
	e := &domain.Expense{}
	var issueDateStr string
	var createdAtStr string
	var updatedAtStr string
	var deletedAtStr sql.NullString
	var taxReviewedAtStr sql.NullString
	var vendorID sql.NullInt64
	var vendorName sql.NullString
	var vendorType sql.NullString
	var vendorICO sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT
			e.id, e.vendor_id, e.expense_number, e.category, e.description,
			e.issue_date, e.amount, e.currency_code, e.exchange_rate,
			e.vat_rate_percent, e.vat_amount,
			e.is_tax_deductible, e.business_percent, e.payment_method,
			e.document_path, e.notes,
			e.tax_reviewed_at,
			e.created_at, e.updated_at, e.deleted_at,
			c.name, c.type, c.ico
		FROM expenses e
		LEFT JOIN contacts c ON c.id = e.vendor_id
		WHERE e.id = ? AND e.deleted_at IS NULL`, id,
	).Scan(
		&e.ID, &vendorID, &e.ExpenseNumber, &e.Category, &e.Description,
		&issueDateStr, &e.Amount, &e.CurrencyCode, &e.ExchangeRate,
		&e.VATRatePercent, &e.VATAmount,
		&e.IsTaxDeductible, &e.BusinessPercent, &e.PaymentMethod,
		&e.DocumentPath, &e.Notes,
		&taxReviewedAtStr,
		&createdAtStr, &updatedAtStr, &deletedAtStr,
		&vendorName, &vendorType, &vendorICO,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("expense %d not found: %w", id, err)
		}
		return nil, fmt.Errorf("querying expense %d: %w", id, err)
	}

	e.IssueDate, err = parseDate(time.DateOnly, issueDateStr)
	if err != nil {
		return nil, fmt.Errorf("scanning expense: %w", err)
	}
	e.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning expense: %w", err)
	}
	e.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning expense: %w", err)
	}
	e.DeletedAt, err = parseDatePtr(time.RFC3339, deletedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning expense: %w", err)
	}
	e.TaxReviewedAt, err = parseDatePtr(time.RFC3339, taxReviewedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning expense: %w", err)
	}
	if vendorID.Valid {
		vid := vendorID.Int64
		e.VendorID = &vid
		if vendorName.Valid {
			e.Vendor = &domain.Contact{
				ID:   vid,
				Name: vendorName.String,
				Type: vendorType.String,
				ICO:  vendorICO.String,
			}
		}
	}

	return e, nil
}

// List retrieves expenses matching the given filter with pagination.
// Returns the matching expenses, total count, and any error.
func (r *ExpenseRepository) List(ctx context.Context, filter domain.ExpenseFilter) ([]domain.Expense, int, error) {
	where := "e.deleted_at IS NULL"
	args := []any{}

	if filter.Category != "" {
		where += " AND e.category = ?"
		args = append(args, filter.Category)
	}
	if filter.VendorID != nil {
		where += " AND e.vendor_id = ?"
		args = append(args, *filter.VendorID)
	}
	if filter.DateFrom != nil {
		where += " AND e.issue_date >= ?"
		args = append(args, *filter.DateFrom)
	}
	if filter.DateTo != nil {
		where += " AND e.issue_date <= ?"
		args = append(args, *filter.DateTo)
	}
	if filter.Search != "" {
		where += " AND (e.expense_number LIKE ? OR e.description LIKE ? OR COALESCE(c.name, '') LIKE ?)"
		search := "%" + filter.Search + "%"
		args = append(args, search, search, search)
	}
	if filter.TaxReviewed != nil {
		if *filter.TaxReviewed {
			where += " AND e.tax_reviewed_at IS NOT NULL"
		} else {
			where += " AND e.tax_reviewed_at IS NULL"
		}
	}

	// Count.
	var total int
	countQuery := "SELECT COUNT(*) FROM expenses e LEFT JOIN contacts c ON c.id = e.vendor_id WHERE " + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting expenses: %w", err)
	}

	// Fetch page.
	query := `SELECT
			e.id, e.vendor_id, e.expense_number, e.category, e.description,
			e.issue_date, e.amount, e.currency_code, e.exchange_rate,
			e.vat_rate_percent, e.vat_amount,
			e.is_tax_deductible, e.business_percent, e.payment_method,
			e.document_path, e.notes,
			e.tax_reviewed_at,
			e.created_at, e.updated_at, e.deleted_at,
			COALESCE(c.name, '') AS vendor_name
		FROM expenses e
		LEFT JOIN contacts c ON c.id = e.vendor_id
		WHERE ` + where + ` ORDER BY e.issue_date DESC`

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", filter.Limit, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing expenses: %w", err)
	}
	defer rows.Close()

	var expenses []domain.Expense
	for rows.Next() {
		var e domain.Expense
		var issueDateStr string
		var createdAtStr string
		var updatedAtStr string
		var deletedAtStr sql.NullString
		var taxReviewedAtStr sql.NullString
		var vendorID sql.NullInt64
		var vendorName string

		if err := rows.Scan(
			&e.ID, &vendorID, &e.ExpenseNumber, &e.Category, &e.Description,
			&issueDateStr, &e.Amount, &e.CurrencyCode, &e.ExchangeRate,
			&e.VATRatePercent, &e.VATAmount,
			&e.IsTaxDeductible, &e.BusinessPercent, &e.PaymentMethod,
			&e.DocumentPath, &e.Notes,
			&taxReviewedAtStr,
			&createdAtStr, &updatedAtStr, &deletedAtStr,
			&vendorName,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning expense row: %w", err)
		}

		e.IssueDate, err = parseDate(time.DateOnly, issueDateStr)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning expense row: %w", err)
		}
		e.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning expense row: %w", err)
		}
		e.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning expense row: %w", err)
		}
		e.DeletedAt, err = parseDatePtr(time.RFC3339, deletedAtStr)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning expense row: %w", err)
		}
		e.TaxReviewedAt, err = parseDatePtr(time.RFC3339, taxReviewedAtStr)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning expense row: %w", err)
		}
		if vendorID.Valid {
			vid := vendorID.Int64
			e.VendorID = &vid
			if vendorName != "" {
				e.Vendor = &domain.Contact{ID: vid, Name: vendorName}
			}
		}

		expenses = append(expenses, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating expense rows: %w", err)
	}
	return expenses, total, nil
}

// MarkTaxReviewed sets tax_reviewed_at to the current timestamp for the given expense IDs.
func (r *ExpenseRepository) MarkTaxReviewed(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(ids)+1)
	args[0] = time.Now().Format(time.RFC3339)
	for i, id := range ids {
		args[i+1] = id
	}
	_, err := r.db.ExecContext(ctx,
		fmt.Sprintf("UPDATE expenses SET tax_reviewed_at = ? WHERE id IN (%s) AND deleted_at IS NULL", placeholders),
		args...,
	)
	return err
}

// UnmarkTaxReviewed clears tax_reviewed_at for the given expense IDs.
func (r *ExpenseRepository) UnmarkTaxReviewed(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	_, err := r.db.ExecContext(ctx,
		fmt.Sprintf("UPDATE expenses SET tax_reviewed_at = NULL WHERE id IN (%s) AND deleted_at IS NULL", placeholders),
		args...,
	)
	return err
}
