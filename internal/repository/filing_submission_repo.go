package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// FilingSubmissionRepository persists data-box send attempts.
type FilingSubmissionRepository struct {
	db *sql.DB
}

// NewFilingSubmissionRepository creates a new FilingSubmissionRepository.
func NewFilingSubmissionRepository(db *sql.DB) *FilingSubmissionRepository {
	return &FilingSubmissionRepository{db: db}
}

// filingSubmissionCols lists the columns scanFilingSubmission reads. The
// receipt blob is never selected; has_receipt stands in for it.
const filingSubmissionCols = `id, document_kind, document_id, channel, environment,
	recipient_box_id, recipient_name, dm_id, dm_annotation, file_name, xml_sha256,
	status, dm_message_status, sent_at, delivered_at,
	COALESCE(length(receipt_zfo), 0) > 0, error_text, created_at`

func scanFilingSubmission(s scanner) (*domain.FilingSubmission, error) {
	var sub domain.FilingSubmission
	var kind, channel, status string
	var dmMessageStatus sql.NullInt64
	var sentAt, deliveredAt sql.NullString
	var createdAt string

	err := s.Scan(
		&sub.ID, &kind, &sub.DocumentID, &channel, &sub.Environment,
		&sub.RecipientBoxID, &sub.RecipientName, &sub.DmID, &sub.DmAnnotation, &sub.FileName, &sub.XMLSHA256,
		&status, &dmMessageStatus, &sentAt, &deliveredAt,
		&sub.HasReceipt, &sub.ErrorText, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	sub.DocumentKind = domain.FilingKind(kind)
	sub.Channel = domain.FilingChannel(channel)
	sub.Status = domain.FilingSubmissionStatus(status)
	if dmMessageStatus.Valid {
		v := int(dmMessageStatus.Int64)
		sub.DmMessageStatus = &v
	}

	if sub.SentAt, err = parseDatePtr(time.RFC3339, sentAt); err != nil {
		return nil, fmt.Errorf("parsing sent_at: %w", err)
	}
	if sub.DeliveredAt, err = parseDatePtr(time.RFC3339, deliveredAt); err != nil {
		return nil, fmt.Errorf("parsing delivered_at: %w", err)
	}
	if sub.CreatedAt, err = parseDate(time.RFC3339, createdAt); err != nil {
		return nil, fmt.Errorf("parsing created_at: %w", err)
	}
	return &sub, nil
}

// Create inserts a submission. It returns domain.ErrFilingAlreadySent when a
// blocking submission of the same bytes to the same environment exists.
func (r *FilingSubmissionRepository) Create(ctx context.Context, sub *domain.FilingSubmission) error {
	sub.CreatedAt = time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO filing_submissions (
			document_kind, document_id, channel, environment,
			recipient_box_id, recipient_name, dm_id, dm_annotation, file_name, xml_sha256,
			status, error_text, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(sub.DocumentKind), sub.DocumentID, string(sub.Channel), sub.Environment,
		sub.RecipientBoxID, sub.RecipientName, sub.DmID, sub.DmAnnotation, sub.FileName, sub.XMLSHA256,
		string(sub.Status), sub.ErrorText, sub.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("inserting filing submission: %w", domain.ErrFilingAlreadySent)
		}
		return fmt.Errorf("inserting filing submission: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting filing submission ID: %w", err)
	}
	sub.ID = id
	return nil
}

// GetByID returns one submission without its receipt blob.
func (r *FilingSubmissionRepository) GetByID(ctx context.Context, id int64) (*domain.FilingSubmission, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+filingSubmissionCols+` FROM filing_submissions WHERE id = ? AND deleted_at IS NULL`, id)
	sub, err := scanFilingSubmission(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("getting filing submission by ID: %w", err)
	}
	return sub, nil
}

// ListByDocument returns the submissions of one document, newest first.
func (r *FilingSubmissionRepository) ListByDocument(ctx context.Context, kind domain.FilingKind, documentID int64) ([]domain.FilingSubmission, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+filingSubmissionCols+` FROM filing_submissions
		 WHERE document_kind = ? AND document_id = ? AND deleted_at IS NULL
		 ORDER BY id DESC`, string(kind), documentID)
	if err != nil {
		return nil, fmt.Errorf("listing filing submissions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := []domain.FilingSubmission{}
	for rows.Next() {
		sub, err := scanFilingSubmission(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning filing submission: %w", err)
		}
		result = append(result, *sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating filing submissions: %w", err)
	}
	return result, nil
}

// MarkSent records that ISDS accepted the message: it stores the message id
// and moves a sending submission to sent.
func (r *FilingSubmissionRepository) MarkSent(ctx context.Context, id int64, dmID string, sentAt time.Time) error {
	return r.updateOne(ctx, "marking filing submission sent", `
		UPDATE filing_submissions SET dm_id = ?, status = ?, sent_at = ?, error_text = ''
		WHERE id = ? AND status = ? AND deleted_at IS NULL`,
		dmID, string(domain.SubmissionSent), sentAt.UTC().Format(time.RFC3339),
		id, string(domain.SubmissionSending))
}

// MarkFailed moves a sending submission to failed or unknown and stores the
// reason. Other statuses are rejected.
func (r *FilingSubmissionRepository) MarkFailed(ctx context.Context, id int64, status domain.FilingSubmissionStatus, errorText string) error {
	if status != domain.SubmissionFailed && status != domain.SubmissionUnknown {
		return fmt.Errorf("marking filing submission failed: %w: status %q", domain.ErrInvalidInput, status)
	}
	return r.updateOne(ctx, "marking filing submission failed", `
		UPDATE filing_submissions SET status = ?, error_text = ?
		WHERE id = ? AND status = ? AND deleted_at IS NULL`,
		string(status), errorText, id, string(domain.SubmissionSending))
}

// UpdateDelivery stores the delivery state read from ISDS. deliveredAt is set
// only for a delivered submission and an already stored value is kept.
func (r *FilingSubmissionRepository) UpdateDelivery(ctx context.Context, id int64, status domain.FilingSubmissionStatus, dmMessageStatus int, deliveredAt *time.Time) error {
	switch status {
	case domain.SubmissionSent, domain.SubmissionDelivered, domain.SubmissionUndeliverable:
	default:
		return fmt.Errorf("updating filing submission delivery: %w: status %q", domain.ErrInvalidInput, status)
	}
	return r.updateOne(ctx, "updating filing submission delivery", `
		UPDATE filing_submissions SET status = ?, dm_message_status = ?, delivered_at = COALESCE(delivered_at, ?)
		WHERE id = ? AND status IN (?, ?, ?) AND deleted_at IS NULL`,
		string(status), dmMessageStatus, formatUTCPtr(deliveredAt),
		id, string(domain.SubmissionSent), string(domain.SubmissionDelivered), string(domain.SubmissionUndeliverable))
}

// SaveReceipt stores the signed sent message (ZFO) of a submission.
func (r *FilingSubmissionRepository) SaveReceipt(ctx context.Context, id int64, zfo []byte) error {
	if len(zfo) == 0 {
		return fmt.Errorf("saving filing receipt: %w: empty receipt", domain.ErrInvalidInput)
	}
	return r.updateOne(ctx, "saving filing receipt",
		`UPDATE filing_submissions SET receipt_zfo = ? WHERE id = ? AND deleted_at IS NULL`, zfo, id)
}

// GetReceipt returns the stored ZFO of a submission, or domain.ErrNotFound
// when the submission or its receipt does not exist.
func (r *FilingSubmissionRepository) GetReceipt(ctx context.Context, id int64) ([]byte, error) {
	var zfo []byte
	err := r.db.QueryRowContext(ctx,
		`SELECT receipt_zfo FROM filing_submissions WHERE id = ? AND deleted_at IS NULL`, id).Scan(&zfo)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("getting filing receipt: %w", err)
	}
	if len(zfo) == 0 {
		return nil, domain.ErrNotFound
	}
	return zfo, nil
}

// updateOne runs a single-row update and reports domain.ErrNotFound when no
// row matched (missing, or not in a state the update applies to).
func (r *FilingSubmissionRepository) updateOne(ctx context.Context, what, query string, args ...any) error {
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: checking rows affected: %w", what, err)
	}
	if n == 0 {
		return fmt.Errorf("%s: %w", what, domain.ErrNotFound)
	}
	return nil
}

func formatUTCPtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
