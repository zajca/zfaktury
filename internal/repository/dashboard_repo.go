package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// RecentInvoice holds summary data for a recently created invoice.
type RecentInvoice struct {
	ID            int64
	InvoiceNumber string
	CustomerID    int64
	TotalAmount   domain.Amount
	Status        string
	IssueDate     time.Time
}

// RecentExpense holds summary data for a recently created expense.
type RecentExpense struct {
	ID          int64
	Description string
	Category    string
	Amount      domain.Amount
	IssueDate   time.Time
}

// DashboardRepository provides read-only aggregate queries for the dashboard.
type DashboardRepository struct {
	db *sql.DB
}

// NewDashboardRepository creates a new DashboardRepository.
func NewDashboardRepository(db *sql.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// RevenueCurrentMonth returns the total revenue (sum of total_amount) for regular invoices
// delivered in the given year and month.
func (r *DashboardRepository) RevenueCurrentMonth(ctx context.Context, year int, month int) (domain.Amount, error) {
	ym := fmt.Sprintf("%04d-%02d", year, month)
	var total int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(total_amount), 0)
		 FROM invoices
		 WHERE strftime('%Y-%m', delivery_date) = ?
		   AND type = 'regular'
		   AND deleted_at IS NULL`, ym).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("querying revenue for %s: %w", ym, err)
	}
	return domain.Amount(total), nil
}

// ExpensesCurrentMonth returns the total expenses (sum of amount) for expenses
// issued in the given year and month.
func (r *DashboardRepository) ExpensesCurrentMonth(ctx context.Context, year int, month int) (domain.Amount, error) {
	ym := fmt.Sprintf("%04d-%02d", year, month)
	var total int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount), 0)
		 FROM expenses
		 WHERE strftime('%Y-%m', issue_date) = ?
		   AND deleted_at IS NULL`, ym).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("querying expenses for %s: %w", ym, err)
	}
	return domain.Amount(total), nil
}

// UnpaidInvoices returns the count and total amount of unpaid invoices
// (draft, sent, or overdue status, excluding credit notes).
func (r *DashboardRepository) UnpaidInvoices(ctx context.Context) (count int, total domain.Amount, err error) {
	var totalVal int64
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(total_amount), 0)
		 FROM invoices
		 WHERE status IN ('draft', 'sent', 'overdue')
		   AND type != 'credit_note'
		   AND deleted_at IS NULL`).Scan(&count, &totalVal)
	if err != nil {
		return 0, 0, fmt.Errorf("querying unpaid invoices: %w", err)
	}
	return count, domain.Amount(totalVal), nil
}

// OverdueInvoices returns the count and total amount of overdue invoices.
func (r *DashboardRepository) OverdueInvoices(ctx context.Context) (count int, total domain.Amount, err error) {
	var totalVal int64
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(total_amount), 0)
		 FROM invoices
		 WHERE status = 'overdue'
		   AND deleted_at IS NULL`).Scan(&count, &totalVal)
	if err != nil {
		return 0, 0, fmt.Errorf("querying overdue invoices: %w", err)
	}
	return count, domain.Amount(totalVal), nil
}

// MonthlyRevenue returns the monthly revenue breakdown for regular invoices
// delivered in the given year.
func (r *DashboardRepository) MonthlyRevenue(ctx context.Context, year int) ([]MonthlyAmount, error) {
	yearStr := fmt.Sprintf("%04d", year)
	rows, err := r.db.QueryContext(ctx,
		`SELECT strftime('%m', delivery_date) AS month, SUM(total_amount)
		 FROM invoices
		 WHERE strftime('%Y', delivery_date) = ?
		   AND type = 'regular'
		   AND deleted_at IS NULL
		 GROUP BY month`, yearStr)
	if err != nil {
		return nil, fmt.Errorf("querying monthly revenue for %d: %w", year, err)
	}
	defer rows.Close()

	var result []MonthlyAmount
	for rows.Next() {
		var monthStr string
		var amount int64
		if err := rows.Scan(&monthStr, &amount); err != nil {
			return nil, fmt.Errorf("scanning monthly revenue row: %w", err)
		}
		m, err := strconv.Atoi(monthStr)
		if err != nil {
			return nil, fmt.Errorf("parsing month %q: %w", monthStr, err)
		}
		result = append(result, MonthlyAmount{Month: m, Amount: domain.Amount(amount)})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating monthly revenue rows: %w", err)
	}
	return result, nil
}

// MonthlyExpenses returns the monthly expenses breakdown for expenses
// issued in the given year.
func (r *DashboardRepository) MonthlyExpenses(ctx context.Context, year int) ([]MonthlyAmount, error) {
	yearStr := fmt.Sprintf("%04d", year)
	rows, err := r.db.QueryContext(ctx,
		`SELECT strftime('%m', issue_date) AS month, SUM(amount)
		 FROM expenses
		 WHERE strftime('%Y', issue_date) = ?
		   AND deleted_at IS NULL
		 GROUP BY month`, yearStr)
	if err != nil {
		return nil, fmt.Errorf("querying monthly expenses for %d: %w", year, err)
	}
	defer rows.Close()

	var result []MonthlyAmount
	for rows.Next() {
		var monthStr string
		var amount int64
		if err := rows.Scan(&monthStr, &amount); err != nil {
			return nil, fmt.Errorf("scanning monthly expense row: %w", err)
		}
		m, err := strconv.Atoi(monthStr)
		if err != nil {
			return nil, fmt.Errorf("parsing month %q: %w", monthStr, err)
		}
		result = append(result, MonthlyAmount{Month: m, Amount: domain.Amount(amount)})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating monthly expense rows: %w", err)
	}
	return result, nil
}

// RecentInvoices returns the most recently created invoices (not soft-deleted).
func (r *DashboardRepository) RecentInvoices(ctx context.Context, limit int) ([]RecentInvoice, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, invoice_number, customer_id, total_amount, status, issue_date
		 FROM invoices
		 WHERE deleted_at IS NULL
		 ORDER BY created_at DESC
		 LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("querying recent invoices: %w", err)
	}
	defer rows.Close()

	var result []RecentInvoice
	for rows.Next() {
		var ri RecentInvoice
		var issueDateStr string
		if err := rows.Scan(&ri.ID, &ri.InvoiceNumber, &ri.CustomerID, &ri.TotalAmount, &ri.Status, &issueDateStr); err != nil {
			return nil, fmt.Errorf("scanning recent invoice row: %w", err)
		}
		issueDate, err := parseDate("2006-01-02", issueDateStr)
		if err != nil {
			return nil, fmt.Errorf("parsing recent invoice issue_date: %w", err)
		}
		ri.IssueDate = issueDate
		result = append(result, ri)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating recent invoice rows: %w", err)
	}
	return result, nil
}

// RecentExpenses returns the most recently created expenses (not soft-deleted).
func (r *DashboardRepository) RecentExpenses(ctx context.Context, limit int) ([]RecentExpense, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, description, category, amount, issue_date
		 FROM expenses
		 WHERE deleted_at IS NULL
		 ORDER BY created_at DESC
		 LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("querying recent expenses: %w", err)
	}
	defer rows.Close()

	var result []RecentExpense
	for rows.Next() {
		var re RecentExpense
		var issueDateStr string
		if err := rows.Scan(&re.ID, &re.Description, &re.Category, &re.Amount, &issueDateStr); err != nil {
			return nil, fmt.Errorf("scanning recent expense row: %w", err)
		}
		issueDate, err := parseDate("2006-01-02", issueDateStr)
		if err != nil {
			return nil, fmt.Errorf("parsing recent expense issue_date: %w", err)
		}
		re.IssueDate = issueDate
		result = append(result, re)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating recent expense rows: %w", err)
	}
	return result, nil
}
