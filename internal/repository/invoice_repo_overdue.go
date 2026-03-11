package repository

import (
	"context"
	"fmt"
	"time"
)

// ListOverdueCandidateIDs returns IDs of invoices that are in 'sent' status
// with a due_date before the given date and not soft-deleted.
func (r *InvoiceRepository) ListOverdueCandidateIDs(ctx context.Context, beforeDate time.Time) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id FROM invoices
		WHERE status = 'sent' AND due_date < ? AND deleted_at IS NULL`,
		beforeDate.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("listing overdue candidate invoices: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning overdue candidate id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating overdue candidate rows: %w", err)
	}
	return ids, nil
}
