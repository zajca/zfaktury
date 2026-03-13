package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
)

// MonthlyAmount holds a revenue or expense total for a single month.
type MonthlyAmount struct {
	Month  int
	Amount domain.Amount
}

// QuarterlyAmount holds a revenue or expense total for a single quarter.
type QuarterlyAmount struct {
	Quarter int
	Amount  domain.Amount
}

// CategoryAmount holds an expense total for a single category.
type CategoryAmount struct {
	Category string
	Amount   domain.Amount
}

// CustomerRevenue holds revenue and invoice count for a single customer.
type CustomerRevenue struct {
	CustomerID   int64
	CustomerName string
	Total        domain.Amount
	InvoiceCount int
}

// ReportRepository provides read-only aggregate queries for reports.
type ReportRepository struct {
	db *sql.DB
}

// NewReportRepository creates a new ReportRepository.
func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

// MonthlyRevenue returns revenue grouped by month for the given year.
// Only regular invoices are included, grouped by delivery_date.
func (r *ReportRepository) MonthlyRevenue(ctx context.Context, year int) ([]MonthlyAmount, error) {
	yearStr := fmt.Sprintf("%04d", year)
	rows, err := r.db.QueryContext(ctx, `
		SELECT CAST(strftime('%m', delivery_date) AS INTEGER) as month,
		       COALESCE(SUM(total_amount), 0) as total
		FROM invoices
		WHERE strftime('%Y', delivery_date) = ? AND type = 'regular' AND deleted_at IS NULL
		GROUP BY month ORDER BY month`, yearStr)
	if err != nil {
		return nil, fmt.Errorf("querying monthly revenue: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []MonthlyAmount
	for rows.Next() {
		var ma MonthlyAmount
		if err := rows.Scan(&ma.Month, &ma.Amount); err != nil {
			return nil, fmt.Errorf("scanning monthly revenue: %w", err)
		}
		result = append(result, ma)
	}
	return result, rows.Err()
}

// QuarterlyRevenue returns revenue grouped by quarter for the given year.
// Only regular invoices are included, grouped by delivery_date.
func (r *ReportRepository) QuarterlyRevenue(ctx context.Context, year int) ([]QuarterlyAmount, error) {
	yearStr := fmt.Sprintf("%04d", year)
	rows, err := r.db.QueryContext(ctx, `
		SELECT CASE
		    WHEN CAST(strftime('%m', delivery_date) AS INTEGER) <= 3 THEN 1
		    WHEN CAST(strftime('%m', delivery_date) AS INTEGER) <= 6 THEN 2
		    WHEN CAST(strftime('%m', delivery_date) AS INTEGER) <= 9 THEN 3
		    ELSE 4 END as quarter,
		    COALESCE(SUM(total_amount), 0) as total
		FROM invoices
		WHERE strftime('%Y', delivery_date) = ? AND type = 'regular' AND deleted_at IS NULL
		GROUP BY quarter ORDER BY quarter`, yearStr)
	if err != nil {
		return nil, fmt.Errorf("querying quarterly revenue: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []QuarterlyAmount
	for rows.Next() {
		var qa QuarterlyAmount
		if err := rows.Scan(&qa.Quarter, &qa.Amount); err != nil {
			return nil, fmt.Errorf("scanning quarterly revenue: %w", err)
		}
		result = append(result, qa)
	}
	return result, rows.Err()
}

// YearlyRevenue returns the total revenue for the given year.
// Only regular invoices are included, grouped by delivery_date.
func (r *ReportRepository) YearlyRevenue(ctx context.Context, year int) (domain.Amount, error) {
	yearStr := fmt.Sprintf("%04d", year)
	var total domain.Amount
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_amount), 0) FROM invoices
		WHERE strftime('%Y', delivery_date) = ? AND type = 'regular' AND deleted_at IS NULL`, yearStr).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("querying yearly revenue: %w", err)
	}
	return total, nil
}

// MonthlyExpenses returns expenses grouped by month for the given year.
// Grouped by issue_date.
func (r *ReportRepository) MonthlyExpenses(ctx context.Context, year int) ([]MonthlyAmount, error) {
	yearStr := fmt.Sprintf("%04d", year)
	rows, err := r.db.QueryContext(ctx, `
		SELECT CAST(strftime('%m', issue_date) AS INTEGER) as month,
		       COALESCE(SUM(amount), 0) as total
		FROM expenses
		WHERE strftime('%Y', issue_date) = ? AND deleted_at IS NULL
		GROUP BY month ORDER BY month`, yearStr)
	if err != nil {
		return nil, fmt.Errorf("querying monthly expenses: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []MonthlyAmount
	for rows.Next() {
		var ma MonthlyAmount
		if err := rows.Scan(&ma.Month, &ma.Amount); err != nil {
			return nil, fmt.Errorf("scanning monthly expenses: %w", err)
		}
		result = append(result, ma)
	}
	return result, rows.Err()
}

// QuarterlyExpenses returns expenses grouped by quarter for the given year.
// Grouped by issue_date.
func (r *ReportRepository) QuarterlyExpenses(ctx context.Context, year int) ([]QuarterlyAmount, error) {
	yearStr := fmt.Sprintf("%04d", year)
	rows, err := r.db.QueryContext(ctx, `
		SELECT CASE
		    WHEN CAST(strftime('%m', issue_date) AS INTEGER) <= 3 THEN 1
		    WHEN CAST(strftime('%m', issue_date) AS INTEGER) <= 6 THEN 2
		    WHEN CAST(strftime('%m', issue_date) AS INTEGER) <= 9 THEN 3
		    ELSE 4 END as quarter,
		    COALESCE(SUM(amount), 0) as total
		FROM expenses
		WHERE strftime('%Y', issue_date) = ? AND deleted_at IS NULL
		GROUP BY quarter ORDER BY quarter`, yearStr)
	if err != nil {
		return nil, fmt.Errorf("querying quarterly expenses: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []QuarterlyAmount
	for rows.Next() {
		var qa QuarterlyAmount
		if err := rows.Scan(&qa.Quarter, &qa.Amount); err != nil {
			return nil, fmt.Errorf("scanning quarterly expenses: %w", err)
		}
		result = append(result, qa)
	}
	return result, rows.Err()
}

// CategoryExpenses returns expenses grouped by category for the given year.
func (r *ReportRepository) CategoryExpenses(ctx context.Context, year int) ([]CategoryAmount, error) {
	yearStr := fmt.Sprintf("%04d", year)
	rows, err := r.db.QueryContext(ctx, `
		SELECT category, COALESCE(SUM(amount), 0) as total
		FROM expenses WHERE strftime('%Y', issue_date) = ? AND deleted_at IS NULL
		GROUP BY category ORDER BY total DESC`, yearStr)
	if err != nil {
		return nil, fmt.Errorf("querying category expenses: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []CategoryAmount
	for rows.Next() {
		var ca CategoryAmount
		if err := rows.Scan(&ca.Category, &ca.Amount); err != nil {
			return nil, fmt.Errorf("scanning category expenses: %w", err)
		}
		result = append(result, ca)
	}
	return result, rows.Err()
}

// TopCustomers returns the top customers by revenue for the given year.
func (r *ReportRepository) TopCustomers(ctx context.Context, year int, limit int) ([]CustomerRevenue, error) {
	yearStr := fmt.Sprintf("%04d", year)
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.name, COALESCE(SUM(i.total_amount), 0) as total, COUNT(i.id) as invoice_count
		FROM invoices i JOIN contacts c ON i.customer_id = c.id
		WHERE strftime('%Y', i.delivery_date) = ? AND i.type = 'regular' AND i.deleted_at IS NULL
		GROUP BY c.id ORDER BY total DESC LIMIT ?`, yearStr, limit)
	if err != nil {
		return nil, fmt.Errorf("querying top customers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []CustomerRevenue
	for rows.Next() {
		var cr CustomerRevenue
		if err := rows.Scan(&cr.CustomerID, &cr.CustomerName, &cr.Total, &cr.InvoiceCount); err != nil {
			return nil, fmt.Errorf("scanning top customers: %w", err)
		}
		result = append(result, cr)
	}
	return result, rows.Err()
}

// ProfitLossMonthly returns monthly revenue and expenses for the given year.
func (r *ReportRepository) ProfitLossMonthly(ctx context.Context, year int) (revenue []MonthlyAmount, expenses []MonthlyAmount, err error) {
	revenue, err = r.MonthlyRevenue(ctx, year)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching monthly revenue for profit/loss: %w", err)
	}
	expenses, err = r.MonthlyExpenses(ctx, year)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching monthly expenses for profit/loss: %w", err)
	}
	return revenue, expenses, nil
}
