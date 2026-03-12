package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// SecurityTransactionRepository handles persistence of SecurityTransaction entities.
type SecurityTransactionRepository struct {
	db *sql.DB
}

// NewSecurityTransactionRepository creates a new SecurityTransactionRepository.
func NewSecurityTransactionRepository(db *sql.DB) *SecurityTransactionRepository {
	return &SecurityTransactionRepository{db: db}
}

// securityTransactionColumns is the list of columns to select for a SecurityTransaction row.
const securityTransactionColumns = `id, year, document_id, asset_type, asset_name, isin, transaction_type, transaction_date, quantity, unit_price, total_amount, fees, currency_code, exchange_rate, cost_basis, computed_gain, time_test_exempt, exempt_amount, created_at, updated_at`

// scanSecurityTransaction scans a SecurityTransaction from a row.
func scanSecurityTransaction(s scanner) (*domain.SecurityTransaction, error) {
	t := &domain.SecurityTransaction{}
	var documentID sql.NullInt64
	var isin sql.NullString
	var transactionDateStr string
	var timeTestExempt int
	var createdAtStr, updatedAtStr string

	err := s.Scan(
		&t.ID, &t.Year, &documentID, &t.AssetType, &t.AssetName, &isin,
		&t.TransactionType, &transactionDateStr, &t.Quantity,
		&t.UnitPrice, &t.TotalAmount, &t.Fees,
		&t.CurrencyCode, &t.ExchangeRate,
		&t.CostBasis, &t.ComputedGain, &timeTestExempt, &t.ExemptAmount,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	if documentID.Valid {
		t.DocumentID = &documentID.Int64
	}
	if isin.Valid {
		t.ISIN = isin.String
	}

	t.TimeTestExempt = timeTestExempt != 0

	t.TransactionDate, err = parseDate("2006-01-02", transactionDateStr)
	if err != nil {
		return nil, fmt.Errorf("scanning security_transaction transaction_date: %w", err)
	}
	t.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning security_transaction created_at: %w", err)
	}
	t.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning security_transaction updated_at: %w", err)
	}

	return t, nil
}

// Create inserts a new security transaction into the database.
func (r *SecurityTransactionRepository) Create(ctx context.Context, tx *domain.SecurityTransaction) error {
	now := time.Now()
	tx.CreatedAt = now
	tx.UpdatedAt = now

	var documentID *int64
	if tx.DocumentID != nil {
		documentID = tx.DocumentID
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO security_transactions (year, document_id, asset_type, asset_name, isin, transaction_type, transaction_date, quantity, unit_price, total_amount, fees, currency_code, exchange_rate, cost_basis, computed_gain, time_test_exempt, exempt_amount, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tx.Year, documentID, tx.AssetType, tx.AssetName, tx.ISIN,
		tx.TransactionType, tx.TransactionDate.Format("2006-01-02"),
		tx.Quantity, tx.UnitPrice, tx.TotalAmount, tx.Fees,
		tx.CurrencyCode, tx.ExchangeRate,
		tx.CostBasis, tx.ComputedGain, boolToInt(tx.TimeTestExempt), tx.ExemptAmount,
		tx.CreatedAt.Format(time.RFC3339), tx.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting security_transaction: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for security_transaction: %w", err)
	}
	tx.ID = id
	return nil
}

// Update modifies an existing security transaction.
func (r *SecurityTransactionRepository) Update(ctx context.Context, tx *domain.SecurityTransaction) error {
	tx.UpdatedAt = time.Now()

	var documentID *int64
	if tx.DocumentID != nil {
		documentID = tx.DocumentID
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE security_transactions SET
			year = ?, document_id = ?, asset_type = ?, asset_name = ?, isin = ?,
			transaction_type = ?, transaction_date = ?, quantity = ?,
			unit_price = ?, total_amount = ?, fees = ?,
			currency_code = ?, exchange_rate = ?,
			cost_basis = ?, computed_gain = ?, time_test_exempt = ?, exempt_amount = ?,
			updated_at = ?
		WHERE id = ?`,
		tx.Year, documentID, tx.AssetType, tx.AssetName, tx.ISIN,
		tx.TransactionType, tx.TransactionDate.Format("2006-01-02"),
		tx.Quantity, tx.UnitPrice, tx.TotalAmount, tx.Fees,
		tx.CurrencyCode, tx.ExchangeRate,
		tx.CostBasis, tx.ComputedGain, boolToInt(tx.TimeTestExempt), tx.ExemptAmount,
		tx.UpdatedAt.Format(time.RFC3339), tx.ID,
	)
	if err != nil {
		return fmt.Errorf("updating security_transaction %d: %w", tx.ID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for security_transaction %d: %w", tx.ID, err)
	}
	if rows == 0 {
		return fmt.Errorf("security_transaction %d: %w", tx.ID, domain.ErrNotFound)
	}
	return nil
}

// Delete removes a security transaction by ID.
func (r *SecurityTransactionRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM security_transactions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting security_transaction %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for security_transaction %d delete: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("security_transaction %d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// GetByID retrieves a security transaction by its ID.
func (r *SecurityTransactionRepository) GetByID(ctx context.Context, id int64) (*domain.SecurityTransaction, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+securityTransactionColumns+` FROM security_transactions WHERE id = ?`, id)
	t, err := scanSecurityTransaction(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("security_transaction %d: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying security_transaction %d: %w", id, err)
	}
	return t, nil
}

// ListByYear retrieves all security transactions for a given year, ordered by transaction_date.
func (r *SecurityTransactionRepository) ListByYear(ctx context.Context, year int) ([]domain.SecurityTransaction, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+securityTransactionColumns+` FROM security_transactions WHERE year = ? ORDER BY transaction_date ASC, id ASC`,
		year,
	)
	if err != nil {
		return nil, fmt.Errorf("listing security_transactions for year %d: %w", year, err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.SecurityTransaction
	for rows.Next() {
		t, err := scanSecurityTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning security_transaction row: %w", err)
		}
		result = append(result, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating security_transaction rows: %w", err)
	}
	return result, nil
}

// ListByDocumentID retrieves all security transactions for a given document ID.
func (r *SecurityTransactionRepository) ListByDocumentID(ctx context.Context, documentID int64) ([]domain.SecurityTransaction, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+securityTransactionColumns+` FROM security_transactions WHERE document_id = ? ORDER BY transaction_date ASC, id ASC`,
		documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing security_transactions for document %d: %w", documentID, err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.SecurityTransaction
	for rows.Next() {
		t, err := scanSecurityTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning security_transaction row: %w", err)
		}
		result = append(result, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating security_transaction rows: %w", err)
	}
	return result, nil
}

// ListBuysForFIFO retrieves all buy transactions for a given asset, ordered by date for FIFO matching.
func (r *SecurityTransactionRepository) ListBuysForFIFO(ctx context.Context, assetName, assetType string) ([]domain.SecurityTransaction, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+securityTransactionColumns+` FROM security_transactions WHERE asset_name = ? AND asset_type = ? AND transaction_type = 'buy' ORDER BY transaction_date ASC, id ASC`,
		assetName, assetType,
	)
	if err != nil {
		return nil, fmt.Errorf("listing buy transactions for FIFO (%s/%s): %w", assetName, assetType, err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.SecurityTransaction
	for rows.Next() {
		t, err := scanSecurityTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning security_transaction row: %w", err)
		}
		result = append(result, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating security_transaction rows: %w", err)
	}
	return result, nil
}

// ListSellsByYear retrieves all sell transactions for a given year, ordered by date.
func (r *SecurityTransactionRepository) ListSellsByYear(ctx context.Context, year int) ([]domain.SecurityTransaction, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+securityTransactionColumns+` FROM security_transactions WHERE year = ? AND transaction_type = 'sell' ORDER BY transaction_date ASC`,
		year,
	)
	if err != nil {
		return nil, fmt.Errorf("listing sell transactions for year %d: %w", year, err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.SecurityTransaction
	for rows.Next() {
		t, err := scanSecurityTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning security_transaction row: %w", err)
		}
		result = append(result, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating security_transaction rows: %w", err)
	}
	return result, nil
}

// UpdateFIFOResults updates the FIFO calculation results for a security transaction.
func (r *SecurityTransactionRepository) UpdateFIFOResults(ctx context.Context, id int64, costBasis, computedGain, exemptAmount domain.Amount, timeTestExempt bool) error {
	now := time.Now()

	result, err := r.db.ExecContext(ctx, `
		UPDATE security_transactions SET cost_basis = ?, computed_gain = ?, exempt_amount = ?, time_test_exempt = ?, updated_at = ? WHERE id = ?`,
		costBasis, computedGain, exemptAmount, boolToInt(timeTestExempt), now.Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("updating FIFO results for security_transaction %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for security_transaction %d FIFO update: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("security_transaction %d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// DeleteByDocumentID removes all security transactions for a given document ID.
func (r *SecurityTransactionRepository) DeleteByDocumentID(ctx context.Context, documentID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM security_transactions WHERE document_id = ?`, documentID)
	if err != nil {
		return fmt.Errorf("deleting security_transactions for document %d: %w", documentID, err)
	}
	return nil
}
