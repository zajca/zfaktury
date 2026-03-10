package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// ReminderRepository handles persistence of PaymentReminder entities.
type ReminderRepository struct {
	db *sql.DB
}

// NewReminderRepository creates a new ReminderRepository.
func NewReminderRepository(db *sql.DB) *ReminderRepository {
	return &ReminderRepository{db: db}
}

// Create inserts a new payment reminder. Sets CreatedAt to now.
func (r *ReminderRepository) Create(ctx context.Context, reminder *domain.PaymentReminder) error {
	reminder.CreatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO payment_reminders (
			invoice_id, reminder_number, sent_at, sent_to, subject, body_preview, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		reminder.InvoiceID, reminder.ReminderNumber,
		reminder.SentAt.Format(time.RFC3339), reminder.SentTo,
		reminder.Subject, reminder.BodyPreview,
		reminder.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting payment reminder: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for payment reminder: %w", err)
	}
	reminder.ID = id
	return nil
}

// ListByInvoiceID returns all reminders for the given invoice, ordered by sent_at ASC.
func (r *ReminderRepository) ListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.PaymentReminder, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, invoice_id, reminder_number, sent_at, sent_to, subject, body_preview, created_at
		FROM payment_reminders
		WHERE invoice_id = ?
		ORDER BY sent_at ASC`, invoiceID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing reminders for invoice %d: %w", invoiceID, err)
	}
	defer rows.Close()

	var reminders []domain.PaymentReminder
	for rows.Next() {
		var rem domain.PaymentReminder
		var sentAtStr, createdAtStr string

		if err := rows.Scan(
			&rem.ID, &rem.InvoiceID, &rem.ReminderNumber,
			&sentAtStr, &rem.SentTo, &rem.Subject, &rem.BodyPreview,
			&createdAtStr,
		); err != nil {
			return nil, fmt.Errorf("scanning payment reminder row: %w", err)
		}

		rem.SentAt, _ = time.Parse(time.RFC3339, sentAtStr)
		rem.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

		reminders = append(reminders, rem)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating payment reminder rows: %w", err)
	}
	return reminders, nil
}

// CountByInvoiceID returns the number of reminders sent for the given invoice.
func (r *ReminderRepository) CountByInvoiceID(ctx context.Context, invoiceID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM payment_reminders WHERE invoice_id = ?`, invoiceID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting reminders for invoice %d: %w", invoiceID, err)
	}
	return count, nil
}
