package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// overdueInvoiceRepo defines the invoice repository methods needed by OverdueService.
type overdueInvoiceRepo interface {
	ListOverdueCandidateIDs(ctx context.Context, beforeDate time.Time) ([]int64, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
}

// statusHistoryRepo defines the status history repository methods needed by OverdueService.
type statusHistoryRepo interface {
	Create(ctx context.Context, change *domain.InvoiceStatusChange) error
	ListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.InvoiceStatusChange, error)
}

// OverdueService handles overdue detection and invoice status history.
type OverdueService struct {
	invoiceRepo overdueInvoiceRepo
	historyRepo statusHistoryRepo
}

// NewOverdueService creates a new OverdueService.
func NewOverdueService(invoiceRepo overdueInvoiceRepo, historyRepo statusHistoryRepo) *OverdueService {
	return &OverdueService{
		invoiceRepo: invoiceRepo,
		historyRepo: historyRepo,
	}
}

// CheckOverdue finds all sent invoices past their due date, marks them as overdue,
// and records the status change in history. Returns the number of invoices marked.
func (s *OverdueService) CheckOverdue(ctx context.Context) (int, error) {
	now := time.Now()

	// First, find all candidates before updating.
	ids, err := s.invoiceRepo.ListOverdueCandidateIDs(ctx, now)
	if err != nil {
		return 0, err
	}

	if len(ids) == 0 {
		return 0, nil
	}

	count := 0
	for _, id := range ids {
		if err := s.invoiceRepo.UpdateStatus(ctx, id, domain.InvoiceStatusOverdue); err != nil {
			slog.Error("failed to mark invoice as overdue", "invoice_id", id, "error", err)
			continue
		}
		count++

		change := &domain.InvoiceStatusChange{
			InvoiceID: id,
			OldStatus: domain.InvoiceStatusSent,
			NewStatus: domain.InvoiceStatusOverdue,
			ChangedAt: now,
			Note:      "automatically marked as overdue",
		}
		if err := s.historyRepo.Create(ctx, change); err != nil {
			slog.Error("failed to record status change for overdue invoice", "invoice_id", id, "error", err)
		}
	}

	return count, nil
}

// GetHistory returns the status change history for a given invoice.
func (s *OverdueService) GetHistory(ctx context.Context, invoiceID int64) ([]domain.InvoiceStatusChange, error) {
	return s.historyRepo.ListByInvoiceID(ctx, invoiceID)
}
