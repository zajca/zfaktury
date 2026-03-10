package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// InvoiceRepository handles persistence of Invoice entities.
type InvoiceRepository struct {
	db *sql.DB
}

// NewInvoiceRepository creates a new InvoiceRepository.
func NewInvoiceRepository(db *sql.DB) *InvoiceRepository {
	return &InvoiceRepository{db: db}
}

// Create inserts a new invoice with its items in a single transaction.
func (r *InvoiceRepository) Create(ctx context.Context, inv *domain.Invoice) error {
	now := time.Now()
	inv.CreatedAt = now
	inv.UpdatedAt = now

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction for invoice create: %w", err)
	}
	defer tx.Rollback()

	// Use NULL for zero sequence_id to avoid FK constraint violation.
	var seqID any
	if inv.SequenceID > 0 {
		seqID = inv.SequenceID
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO invoices (
			sequence_id, invoice_number, type, status,
			issue_date, due_date, delivery_date, variable_symbol, constant_symbol,
			customer_id, currency_code, exchange_rate,
			payment_method, bank_account, bank_code, iban, swift,
			subtotal_amount, vat_amount, total_amount, paid_amount,
			notes, internal_notes, sent_at, paid_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		seqID, inv.InvoiceNumber, inv.Type, inv.Status,
		inv.IssueDate, inv.DueDate, inv.DeliveryDate, inv.VariableSymbol, inv.ConstantSymbol,
		inv.CustomerID, inv.CurrencyCode, inv.ExchangeRate,
		inv.PaymentMethod, inv.BankAccount, inv.BankCode, inv.IBAN, inv.SWIFT,
		inv.SubtotalAmount, inv.VATAmount, inv.TotalAmount, inv.PaidAmount,
		inv.Notes, inv.InternalNotes, inv.SentAt, inv.PaidAt,
		inv.CreatedAt, inv.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting invoice: %w", err)
	}

	invoiceID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for invoice: %w", err)
	}
	inv.ID = invoiceID

	for i := range inv.Items {
		item := &inv.Items[i]
		item.InvoiceID = invoiceID

		itemResult, err := tx.ExecContext(ctx, `
			INSERT INTO invoice_items (
				invoice_id, description, quantity, unit, unit_price,
				vat_rate_percent, vat_amount, total_amount, sort_order
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.InvoiceID, item.Description, item.Quantity, item.Unit, item.UnitPrice,
			item.VATRatePercent, item.VATAmount, item.TotalAmount, item.SortOrder,
		)
		if err != nil {
			return fmt.Errorf("inserting invoice item %d: %w", i, err)
		}
		itemID, err := itemResult.LastInsertId()
		if err != nil {
			return fmt.Errorf("getting last insert id for invoice item %d: %w", i, err)
		}
		item.ID = itemID
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing invoice transaction: %w", err)
	}
	return nil
}

// Update modifies an existing invoice and replaces its items.
func (r *InvoiceRepository) Update(ctx context.Context, inv *domain.Invoice) error {
	inv.UpdatedAt = time.Now()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction for invoice update: %w", err)
	}
	defer tx.Rollback()

	var updateSeqID any
	if inv.SequenceID > 0 {
		updateSeqID = inv.SequenceID
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE invoices SET
			sequence_id = ?, invoice_number = ?, type = ?, status = ?,
			issue_date = ?, due_date = ?, delivery_date = ?, variable_symbol = ?, constant_symbol = ?,
			customer_id = ?, currency_code = ?, exchange_rate = ?,
			payment_method = ?, bank_account = ?, bank_code = ?, iban = ?, swift = ?,
			subtotal_amount = ?, vat_amount = ?, total_amount = ?, paid_amount = ?,
			notes = ?, internal_notes = ?, sent_at = ?, paid_at = ?,
			updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`,
		updateSeqID, inv.InvoiceNumber, inv.Type, inv.Status,
		inv.IssueDate, inv.DueDate, inv.DeliveryDate, inv.VariableSymbol, inv.ConstantSymbol,
		inv.CustomerID, inv.CurrencyCode, inv.ExchangeRate,
		inv.PaymentMethod, inv.BankAccount, inv.BankCode, inv.IBAN, inv.SWIFT,
		inv.SubtotalAmount, inv.VATAmount, inv.TotalAmount, inv.PaidAmount,
		inv.Notes, inv.InternalNotes, inv.SentAt, inv.PaidAt,
		inv.UpdatedAt, inv.ID,
	)
	if err != nil {
		return fmt.Errorf("updating invoice %d: %w", inv.ID, err)
	}

	// Delete existing items and re-insert.
	_, err = tx.ExecContext(ctx, `DELETE FROM invoice_items WHERE invoice_id = ?`, inv.ID)
	if err != nil {
		return fmt.Errorf("deleting old invoice items for invoice %d: %w", inv.ID, err)
	}

	for i := range inv.Items {
		item := &inv.Items[i]
		item.InvoiceID = inv.ID

		itemResult, err := tx.ExecContext(ctx, `
			INSERT INTO invoice_items (
				invoice_id, description, quantity, unit, unit_price,
				vat_rate_percent, vat_amount, total_amount, sort_order
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			item.InvoiceID, item.Description, item.Quantity, item.Unit, item.UnitPrice,
			item.VATRatePercent, item.VATAmount, item.TotalAmount, item.SortOrder,
		)
		if err != nil {
			return fmt.Errorf("inserting invoice item %d on update: %w", i, err)
		}
		itemID, err := itemResult.LastInsertId()
		if err != nil {
			return fmt.Errorf("getting last insert id for invoice item %d on update: %w", i, err)
		}
		item.ID = itemID
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing invoice update transaction: %w", err)
	}
	return nil
}

// Delete performs a soft delete on an invoice.
func (r *InvoiceRepository) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
		UPDATE invoices SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		now, now, id,
	)
	if err != nil {
		return fmt.Errorf("soft-deleting invoice %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for invoice %d: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("invoice %d not found or already deleted", id)
	}
	return nil
}

// GetByID retrieves an invoice with its items and customer data.
func (r *InvoiceRepository) GetByID(ctx context.Context, id int64) (*domain.Invoice, error) {
	inv := &domain.Invoice{}
	var seqID sql.NullInt64
	var issueDateStr, dueDateStr string
	var deliveryDateStr sql.NullString
	var createdAtStr string
	var updatedAtStr string
	var deletedAtStr sql.NullString
	var sentAtStr sql.NullString
	var paidAtStr sql.NullString
	var custID sql.NullInt64
	var custType, custName, custICO, custDIC sql.NullString
	var custStreet, custCity, custZIP, custCountry sql.NullString
	var custEmail, custPhone, custWeb sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT
			i.id, i.sequence_id, i.invoice_number, i.type, i.status,
			i.issue_date, i.due_date, i.delivery_date, i.variable_symbol, i.constant_symbol,
			i.customer_id, i.currency_code, i.exchange_rate,
			i.payment_method, i.bank_account, i.bank_code, i.iban, i.swift,
			i.subtotal_amount, i.vat_amount, i.total_amount, i.paid_amount,
			i.notes, i.internal_notes, i.sent_at, i.paid_at,
			i.created_at, i.updated_at, i.deleted_at,
			c.id, c.type, c.name, c.ico, c.dic,
			c.street, c.city, c.zip, c.country,
			c.email, c.phone, c.web
		FROM invoices i
		LEFT JOIN contacts c ON c.id = i.customer_id
		WHERE i.id = ? AND i.deleted_at IS NULL`, id,
	).Scan(
		&inv.ID, &seqID, &inv.InvoiceNumber, &inv.Type, &inv.Status,
		&issueDateStr, &dueDateStr, &deliveryDateStr, &inv.VariableSymbol, &inv.ConstantSymbol,
		&inv.CustomerID, &inv.CurrencyCode, &inv.ExchangeRate,
		&inv.PaymentMethod, &inv.BankAccount, &inv.BankCode, &inv.IBAN, &inv.SWIFT,
		&inv.SubtotalAmount, &inv.VATAmount, &inv.TotalAmount, &inv.PaidAmount,
		&inv.Notes, &inv.InternalNotes, &sentAtStr, &paidAtStr,
		&createdAtStr, &updatedAtStr, &deletedAtStr,
		&custID, &custType, &custName, &custICO, &custDIC,
		&custStreet, &custCity, &custZIP, &custCountry,
		&custEmail, &custPhone, &custWeb,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invoice %d not found: %w", id, err)
		}
		return nil, fmt.Errorf("querying invoice %d: %w", id, err)
	}

	if seqID.Valid {
		inv.SequenceID = seqID.Int64
	}
	inv.IssueDate, _ = time.Parse("2006-01-02", issueDateStr)
	inv.DueDate, _ = time.Parse("2006-01-02", dueDateStr)
	if deliveryDateStr.Valid {
		inv.DeliveryDate, _ = time.Parse("2006-01-02", deliveryDateStr.String)
	}
	inv.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	inv.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	if sentAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, sentAtStr.String)
		inv.SentAt = &t
	}
	if paidAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, paidAtStr.String)
		inv.PaidAt = &t
	}
	if deletedAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, deletedAtStr.String)
		inv.DeletedAt = &t
	}
	if custID.Valid {
		inv.Customer = &domain.Contact{
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
		SELECT id, invoice_id, description, quantity, unit, unit_price,
			vat_rate_percent, vat_amount, total_amount, sort_order
		FROM invoice_items
		WHERE invoice_id = ?
		ORDER BY sort_order ASC`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("querying items for invoice %d: %w", id, err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item domain.InvoiceItem
		if err := itemRows.Scan(
			&item.ID, &item.InvoiceID, &item.Description, &item.Quantity, &item.Unit, &item.UnitPrice,
			&item.VATRatePercent, &item.VATAmount, &item.TotalAmount, &item.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("scanning invoice item row: %w", err)
		}
		inv.Items = append(inv.Items, item)
	}
	if err := itemRows.Err(); err != nil {
		return nil, fmt.Errorf("iterating invoice item rows: %w", err)
	}

	return inv, nil
}

// List retrieves invoices matching the given filter with pagination.
// Returns the matching invoices (without items), total count, and any error.
func (r *InvoiceRepository) List(ctx context.Context, filter domain.InvoiceFilter) ([]domain.Invoice, int, error) {
	where := "i.deleted_at IS NULL"
	args := []any{}

	if filter.Status != "" {
		where += " AND i.status = ?"
		args = append(args, filter.Status)
	}
	if filter.CustomerID != nil {
		where += " AND i.customer_id = ?"
		args = append(args, *filter.CustomerID)
	}
	if filter.DateFrom != nil {
		where += " AND i.issue_date >= ?"
		args = append(args, *filter.DateFrom)
	}
	if filter.DateTo != nil {
		where += " AND i.issue_date <= ?"
		args = append(args, *filter.DateTo)
	}
	if filter.Search != "" {
		where += " AND (i.invoice_number LIKE ? OR i.variable_symbol LIKE ? OR c.name LIKE ?)"
		search := "%" + filter.Search + "%"
		args = append(args, search, search, search)
	}

	// Count.
	var total int
	countQuery := "SELECT COUNT(*) FROM invoices i LEFT JOIN contacts c ON c.id = i.customer_id WHERE " + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting invoices: %w", err)
	}

	// Fetch page.
	query := `SELECT
			i.id, i.sequence_id, i.invoice_number, i.type, i.status,
			i.issue_date, i.due_date, i.delivery_date, i.variable_symbol, i.constant_symbol,
			i.customer_id, i.currency_code, i.exchange_rate,
			i.payment_method, i.bank_account, i.bank_code, i.iban, i.swift,
			i.subtotal_amount, i.vat_amount, i.total_amount, i.paid_amount,
			i.notes, i.internal_notes, i.sent_at, i.paid_at,
			i.created_at, i.updated_at, i.deleted_at,
			COALESCE(c.name, '') AS customer_name
		FROM invoices i
		LEFT JOIN contacts c ON c.id = i.customer_id
		WHERE ` + where + ` ORDER BY i.issue_date DESC`

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", filter.Limit, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing invoices: %w", err)
	}
	defer rows.Close()

	var invoices []domain.Invoice
	for rows.Next() {
		var inv domain.Invoice
		var listSeqID sql.NullInt64
		var issueDateStr, dueDateStr string
		var deliveryDateStr sql.NullString
		var createdAtStr string
		var updatedAtStr string
		var deletedAtStr sql.NullString
		var sentAtStr sql.NullString
		var paidAtStr sql.NullString
		var customerName string

		if err := rows.Scan(
			&inv.ID, &listSeqID, &inv.InvoiceNumber, &inv.Type, &inv.Status,
			&issueDateStr, &dueDateStr, &deliveryDateStr, &inv.VariableSymbol, &inv.ConstantSymbol,
			&inv.CustomerID, &inv.CurrencyCode, &inv.ExchangeRate,
			&inv.PaymentMethod, &inv.BankAccount, &inv.BankCode, &inv.IBAN, &inv.SWIFT,
			&inv.SubtotalAmount, &inv.VATAmount, &inv.TotalAmount, &inv.PaidAmount,
			&inv.Notes, &inv.InternalNotes, &sentAtStr, &paidAtStr,
			&createdAtStr, &updatedAtStr, &deletedAtStr,
			&customerName,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning invoice row: %w", err)
		}

		if listSeqID.Valid {
			inv.SequenceID = listSeqID.Int64
		}
		inv.IssueDate, _ = time.Parse("2006-01-02", issueDateStr)
		inv.DueDate, _ = time.Parse("2006-01-02", dueDateStr)
		if deliveryDateStr.Valid {
			inv.DeliveryDate, _ = time.Parse("2006-01-02", deliveryDateStr.String)
		}
		inv.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		inv.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		if sentAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, sentAtStr.String)
			inv.SentAt = &t
		}
		if paidAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, paidAtStr.String)
			inv.PaidAt = &t
		}
		if deletedAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, deletedAtStr.String)
			inv.DeletedAt = &t
		}
		if customerName != "" {
			inv.Customer = &domain.Contact{ID: inv.CustomerID, Name: customerName}
		}

		invoices = append(invoices, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating invoice rows: %w", err)
	}
	return invoices, total, nil
}

// UpdateStatus changes the status of an invoice.
func (r *InvoiceRepository) UpdateStatus(ctx context.Context, id int64, status string) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
		UPDATE invoices SET status = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		status, now, id,
	)
	if err != nil {
		return fmt.Errorf("updating status of invoice %d to %s: %w", id, status, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for invoice %d status update: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("invoice %d not found or already deleted", id)
	}
	return nil
}

// GetNextNumber atomically increments the sequence counter and returns the formatted invoice number.
func (r *InvoiceRepository) GetNextNumber(ctx context.Context, sequenceID int64) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("beginning transaction for next number: %w", err)
	}
	defer tx.Rollback()

	var seq domain.InvoiceSequence
	err = tx.QueryRowContext(ctx, `
		SELECT id, prefix, next_number, year, format_pattern
		FROM invoice_sequences WHERE id = ?`, sequenceID,
	).Scan(&seq.ID, &seq.Prefix, &seq.NextNumber, &seq.Year, &seq.FormatPattern)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("invoice sequence %d not found: %w", sequenceID, err)
		}
		return "", fmt.Errorf("querying invoice sequence %d: %w", sequenceID, err)
	}

	// NOTE: format_pattern is not yet implemented; using hardcoded format.
	// This must stay consistent with service.FormatPreview.
	number := fmt.Sprintf("%s%d%04d", seq.Prefix, seq.Year, seq.NextNumber)

	_, err = tx.ExecContext(ctx, `
		UPDATE invoice_sequences SET next_number = next_number + 1 WHERE id = ?`, sequenceID)
	if err != nil {
		return "", fmt.Errorf("incrementing sequence %d: %w", sequenceID, err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("committing sequence increment: %w", err)
	}
	return number, nil
}
