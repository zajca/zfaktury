package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// RecurringInvoiceRepository handles persistence of RecurringInvoice entities.
type RecurringInvoiceRepository struct {
	db *sql.DB
}

// NewRecurringInvoiceRepository creates a new RecurringInvoiceRepository.
func NewRecurringInvoiceRepository(db *sql.DB) *RecurringInvoiceRepository {
	return &RecurringInvoiceRepository{db: db}
}

// Create inserts a new recurring invoice with its items in a single transaction.
func (r *RecurringInvoiceRepository) Create(ctx context.Context, ri *domain.RecurringInvoice) error {
	now := time.Now()
	ri.CreatedAt = now
	ri.UpdatedAt = now

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction for recurring invoice create: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var endDate any
	if ri.EndDate != nil {
		endDate = ri.EndDate.Format("2006-01-02")
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO recurring_invoices (
			name, customer_id, frequency, next_issue_date, end_date,
			currency_code, exchange_rate, payment_method,
			bank_account, bank_code, iban, swift,
			constant_symbol, notes, is_active,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ri.Name, ri.CustomerID, ri.Frequency, ri.NextIssueDate.Format("2006-01-02"), endDate,
		ri.CurrencyCode, ri.ExchangeRate, ri.PaymentMethod,
		ri.BankAccount, ri.BankCode, ri.IBAN, ri.SWIFT,
		ri.ConstantSymbol, ri.Notes, ri.IsActive,
		ri.CreatedAt.Format(time.RFC3339), ri.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting recurring invoice: %w", err)
	}

	riID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for recurring invoice: %w", err)
	}
	ri.ID = riID

	for i := range ri.Items {
		item := &ri.Items[i]
		item.RecurringInvoiceID = riID

		itemResult, err := tx.ExecContext(ctx, `
			INSERT INTO recurring_invoice_items (
				recurring_invoice_id, description, quantity, unit, unit_price,
				vat_rate_percent, sort_order
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			item.RecurringInvoiceID, item.Description, item.Quantity, item.Unit, item.UnitPrice,
			item.VATRatePercent, item.SortOrder,
		)
		if err != nil {
			return fmt.Errorf("inserting recurring invoice item %d: %w", i, err)
		}
		itemID, err := itemResult.LastInsertId()
		if err != nil {
			return fmt.Errorf("getting last insert id for recurring invoice item %d: %w", i, err)
		}
		item.ID = itemID
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing recurring invoice transaction: %w", err)
	}
	return nil
}

// Update modifies an existing recurring invoice and replaces its items.
func (r *RecurringInvoiceRepository) Update(ctx context.Context, ri *domain.RecurringInvoice) error {
	ri.UpdatedAt = time.Now()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction for recurring invoice update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var endDate any
	if ri.EndDate != nil {
		endDate = ri.EndDate.Format("2006-01-02")
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE recurring_invoices SET
			name = ?, customer_id = ?, frequency = ?, next_issue_date = ?, end_date = ?,
			currency_code = ?, exchange_rate = ?, payment_method = ?,
			bank_account = ?, bank_code = ?, iban = ?, swift = ?,
			constant_symbol = ?, notes = ?, is_active = ?,
			updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`,
		ri.Name, ri.CustomerID, ri.Frequency, ri.NextIssueDate.Format("2006-01-02"), endDate,
		ri.CurrencyCode, ri.ExchangeRate, ri.PaymentMethod,
		ri.BankAccount, ri.BankCode, ri.IBAN, ri.SWIFT,
		ri.ConstantSymbol, ri.Notes, ri.IsActive,
		ri.UpdatedAt.Format(time.RFC3339), ri.ID,
	)
	if err != nil {
		return fmt.Errorf("updating recurring invoice %d: %w", ri.ID, err)
	}

	// Delete existing items and re-insert.
	_, err = tx.ExecContext(ctx, `DELETE FROM recurring_invoice_items WHERE recurring_invoice_id = ?`, ri.ID)
	if err != nil {
		return fmt.Errorf("deleting old recurring invoice items for %d: %w", ri.ID, err)
	}

	for i := range ri.Items {
		item := &ri.Items[i]
		item.RecurringInvoiceID = ri.ID

		itemResult, err := tx.ExecContext(ctx, `
			INSERT INTO recurring_invoice_items (
				recurring_invoice_id, description, quantity, unit, unit_price,
				vat_rate_percent, sort_order
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			item.RecurringInvoiceID, item.Description, item.Quantity, item.Unit, item.UnitPrice,
			item.VATRatePercent, item.SortOrder,
		)
		if err != nil {
			return fmt.Errorf("inserting recurring invoice item %d on update: %w", i, err)
		}
		itemID, err := itemResult.LastInsertId()
		if err != nil {
			return fmt.Errorf("getting last insert id for recurring invoice item %d on update: %w", i, err)
		}
		item.ID = itemID
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing recurring invoice update transaction: %w", err)
	}
	return nil
}

// Delete performs a soft delete on a recurring invoice.
func (r *RecurringInvoiceRepository) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	nowStr := now.Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx, `
		UPDATE recurring_invoices SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		nowStr, nowStr, id,
	)
	if err != nil {
		return fmt.Errorf("soft-deleting recurring invoice %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for recurring invoice %d: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("recurring invoice %d not found or already deleted", id)
	}
	return nil
}

// GetByID retrieves a recurring invoice with its items and customer data.
func (r *RecurringInvoiceRepository) GetByID(ctx context.Context, id int64) (*domain.RecurringInvoice, error) {
	ri := &domain.RecurringInvoice{}
	var nextIssueDateStr string
	var endDateStr sql.NullString
	var createdAtStr string
	var updatedAtStr string
	var deletedAtStr sql.NullString
	var custID sql.NullInt64
	var custType, custName, custICO, custDIC sql.NullString
	var custStreet, custCity, custZIP, custCountry sql.NullString
	var custEmail, custPhone, custWeb sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT
			ri.id, ri.name, ri.customer_id, ri.frequency, ri.next_issue_date, ri.end_date,
			ri.currency_code, ri.exchange_rate, ri.payment_method,
			ri.bank_account, ri.bank_code, ri.iban, ri.swift,
			ri.constant_symbol, ri.notes, ri.is_active,
			ri.created_at, ri.updated_at, ri.deleted_at,
			c.id, c.type, c.name, c.ico, c.dic,
			c.street, c.city, c.zip, c.country,
			c.email, c.phone, c.web
		FROM recurring_invoices ri
		LEFT JOIN contacts c ON c.id = ri.customer_id
		WHERE ri.id = ? AND ri.deleted_at IS NULL`, id,
	).Scan(
		&ri.ID, &ri.Name, &ri.CustomerID, &ri.Frequency, &nextIssueDateStr, &endDateStr,
		&ri.CurrencyCode, &ri.ExchangeRate, &ri.PaymentMethod,
		&ri.BankAccount, &ri.BankCode, &ri.IBAN, &ri.SWIFT,
		&ri.ConstantSymbol, &ri.Notes, &ri.IsActive,
		&createdAtStr, &updatedAtStr, &deletedAtStr,
		&custID, &custType, &custName, &custICO, &custDIC,
		&custStreet, &custCity, &custZIP, &custCountry,
		&custEmail, &custPhone, &custWeb,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("recurring invoice %d not found: %w", id, err)
		}
		return nil, fmt.Errorf("querying recurring invoice %d: %w", id, err)
	}

	ri.NextIssueDate, err = parseDate(time.DateOnly, nextIssueDateStr)
	if err != nil {
		return nil, fmt.Errorf("scanning recurring invoice: %w", err)
	}
	ri.EndDate, err = parseDatePtr(time.DateOnly, endDateStr)
	if err != nil {
		return nil, fmt.Errorf("scanning recurring invoice: %w", err)
	}
	ri.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning recurring invoice: %w", err)
	}
	ri.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning recurring invoice: %w", err)
	}
	ri.DeletedAt, err = parseDatePtr(time.RFC3339, deletedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning recurring invoice: %w", err)
	}

	if custID.Valid {
		ri.Customer = &domain.Contact{
			ID:      custID.Int64,
			Type:    custType.String,
			Name:    custName.String,
			ICO:     custICO.String,
			DIC:     custDIC.String,
			Street:  custStreet.String,
			City:    custCity.String,
			ZIP:     custZIP.String,
			Country: custCountry.String,
			Email:   custEmail.String,
			Phone:   custPhone.String,
			Web:     custWeb.String,
		}
	}

	// Fetch items.
	itemRows, err := r.db.QueryContext(ctx, `
		SELECT id, recurring_invoice_id, description, quantity, unit, unit_price,
			vat_rate_percent, sort_order
		FROM recurring_invoice_items
		WHERE recurring_invoice_id = ?
		ORDER BY sort_order ASC`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("querying items for recurring invoice %d: %w", id, err)
	}
	defer func() { _ = itemRows.Close() }()

	for itemRows.Next() {
		var item domain.RecurringInvoiceItem
		if err := itemRows.Scan(
			&item.ID, &item.RecurringInvoiceID, &item.Description, &item.Quantity, &item.Unit, &item.UnitPrice,
			&item.VATRatePercent, &item.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("scanning recurring invoice item row: %w", err)
		}
		ri.Items = append(ri.Items, item)
	}
	if err := itemRows.Err(); err != nil {
		return nil, fmt.Errorf("iterating recurring invoice item rows: %w", err)
	}

	return ri, nil
}

// List retrieves all non-deleted recurring invoices with customer name.
func (r *RecurringInvoiceRepository) List(ctx context.Context) ([]domain.RecurringInvoice, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			ri.id, ri.name, ri.customer_id, ri.frequency, ri.next_issue_date, ri.end_date,
			ri.currency_code, ri.exchange_rate, ri.payment_method,
			ri.bank_account, ri.bank_code, ri.iban, ri.swift,
			ri.constant_symbol, ri.notes, ri.is_active,
			ri.created_at, ri.updated_at, ri.deleted_at,
			COALESCE(c.name, '') AS customer_name
		FROM recurring_invoices ri
		LEFT JOIN contacts c ON c.id = ri.customer_id
		WHERE ri.deleted_at IS NULL
		ORDER BY ri.next_issue_date ASC`)
	if err != nil {
		return nil, fmt.Errorf("listing recurring invoices: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.RecurringInvoice
	for rows.Next() {
		var ri domain.RecurringInvoice
		var nextIssueDateStr string
		var endDateStr sql.NullString
		var createdAtStr string
		var updatedAtStr string
		var deletedAtStr sql.NullString
		var customerName string

		if err := rows.Scan(
			&ri.ID, &ri.Name, &ri.CustomerID, &ri.Frequency, &nextIssueDateStr, &endDateStr,
			&ri.CurrencyCode, &ri.ExchangeRate, &ri.PaymentMethod,
			&ri.BankAccount, &ri.BankCode, &ri.IBAN, &ri.SWIFT,
			&ri.ConstantSymbol, &ri.Notes, &ri.IsActive,
			&createdAtStr, &updatedAtStr, &deletedAtStr,
			&customerName,
		); err != nil {
			return nil, fmt.Errorf("scanning recurring invoice row: %w", err)
		}

		ri.NextIssueDate, err = parseDate(time.DateOnly, nextIssueDateStr)
		if err != nil {
			return nil, fmt.Errorf("scanning recurring invoice row: %w", err)
		}
		ri.EndDate, err = parseDatePtr(time.DateOnly, endDateStr)
		if err != nil {
			return nil, fmt.Errorf("scanning recurring invoice row: %w", err)
		}
		ri.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("scanning recurring invoice row: %w", err)
		}
		ri.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("scanning recurring invoice row: %w", err)
		}
		ri.DeletedAt, err = parseDatePtr(time.RFC3339, deletedAtStr)
		if err != nil {
			return nil, fmt.Errorf("scanning recurring invoice row: %w", err)
		}
		if customerName != "" {
			ri.Customer = &domain.Contact{ID: ri.CustomerID, Name: customerName}
		}

		result = append(result, ri)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating recurring invoice rows: %w", err)
	}
	return result, nil
}

// ListDue returns active recurring invoices where next_issue_date <= date.
func (r *RecurringInvoiceRepository) ListDue(ctx context.Context, date time.Time) ([]domain.RecurringInvoice, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			ri.id, ri.name, ri.customer_id, ri.frequency, ri.next_issue_date, ri.end_date,
			ri.currency_code, ri.exchange_rate, ri.payment_method,
			ri.bank_account, ri.bank_code, ri.iban, ri.swift,
			ri.constant_symbol, ri.notes, ri.is_active,
			ri.created_at, ri.updated_at, ri.deleted_at
		FROM recurring_invoices ri
		WHERE ri.deleted_at IS NULL
			AND ri.is_active = 1
			AND ri.next_issue_date <= ?
		ORDER BY ri.next_issue_date ASC`, date.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("listing due recurring invoices: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.RecurringInvoice
	for rows.Next() {
		var ri domain.RecurringInvoice
		var nextIssueDateStr string
		var endDateStr sql.NullString
		var createdAtStr string
		var updatedAtStr string
		var deletedAtStr sql.NullString

		if err := rows.Scan(
			&ri.ID, &ri.Name, &ri.CustomerID, &ri.Frequency, &nextIssueDateStr, &endDateStr,
			&ri.CurrencyCode, &ri.ExchangeRate, &ri.PaymentMethod,
			&ri.BankAccount, &ri.BankCode, &ri.IBAN, &ri.SWIFT,
			&ri.ConstantSymbol, &ri.Notes, &ri.IsActive,
			&createdAtStr, &updatedAtStr, &deletedAtStr,
		); err != nil {
			return nil, fmt.Errorf("scanning due recurring invoice row: %w", err)
		}

		ri.NextIssueDate, err = parseDate(time.DateOnly, nextIssueDateStr)
		if err != nil {
			return nil, fmt.Errorf("scanning due recurring invoice row: %w", err)
		}
		ri.EndDate, err = parseDatePtr(time.DateOnly, endDateStr)
		if err != nil {
			return nil, fmt.Errorf("scanning due recurring invoice row: %w", err)
		}
		ri.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("scanning due recurring invoice row: %w", err)
		}
		ri.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("scanning due recurring invoice row: %w", err)
		}
		ri.DeletedAt, err = parseDatePtr(time.RFC3339, deletedAtStr)
		if err != nil {
			return nil, fmt.Errorf("scanning due recurring invoice row: %w", err)
		}

		result = append(result, ri)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating due recurring invoice rows: %w", err)
	}
	_ = rows.Close()

	// Load items for each due recurring invoice (after closing the outer cursor
	// to avoid SQLite single-connection issues).
	for i := range result {
		ri := &result[i]
		itemRows, err := r.db.QueryContext(ctx, `
			SELECT id, recurring_invoice_id, description, quantity, unit, unit_price,
				vat_rate_percent, sort_order
			FROM recurring_invoice_items
			WHERE recurring_invoice_id = ?
			ORDER BY sort_order ASC`, ri.ID)
		if err != nil {
			return nil, fmt.Errorf("querying items for due recurring invoice %d: %w", ri.ID, err)
		}

		for itemRows.Next() {
			var item domain.RecurringInvoiceItem
			if err := itemRows.Scan(
				&item.ID, &item.RecurringInvoiceID, &item.Description, &item.Quantity, &item.Unit, &item.UnitPrice,
				&item.VATRatePercent, &item.SortOrder,
			); err != nil {
				_ = itemRows.Close() //nolint:sqlclosecheck // closed explicitly below after loop
				return nil, fmt.Errorf("scanning due recurring invoice item row: %w", err)
			}
			ri.Items = append(ri.Items, item)
		}
		_ = itemRows.Close()
		if err := itemRows.Err(); err != nil {
			return nil, fmt.Errorf("iterating due recurring invoice item rows: %w", err)
		}
	}

	return result, nil
}

// Deactivate sets is_active to false for a recurring invoice.
func (r *RecurringInvoiceRepository) Deactivate(ctx context.Context, id int64) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
		UPDATE recurring_invoices SET is_active = 0, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		now.Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("deactivating recurring invoice %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for recurring invoice %d deactivation: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("recurring invoice %d not found or already deleted", id)
	}
	return nil
}
