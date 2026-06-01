package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/zajca/zfaktury/internal/repository"
)

// RecurringScheduler generates due recurring invoices for every company once a
// day at a configured local-time hour, auto-sending the invoices whose template
// opts into auto-send. It uses only the standard library time package -- no cron
// dependency.
type RecurringScheduler struct {
	companies repository.CompanyRepo
	recurring *RecurringInvoiceService
	hour      int
}

// NewRecurringScheduler creates a scheduler that runs daily at the given
// local-time hour (0-23).
func NewRecurringScheduler(companies repository.CompanyRepo, recurring *RecurringInvoiceService, hour int) *RecurringScheduler {
	return &RecurringScheduler{
		companies: companies,
		recurring: recurring,
		hour:      hour,
	}
}

// startupDelay is how long after process start the first catch-up run fires, so
// a freshly-due template is not stuck until the next scheduled hour.
const startupDelay = 1 * time.Minute

// Run blocks until ctx is cancelled. It performs one catch-up run shortly after
// start and then runs once per day at the configured hour.
func (s *RecurringScheduler) Run(ctx context.Context) {
	slog.Info("recurring scheduler started", "hour", s.hour)

	timer := time.NewTimer(startupDelay)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("recurring scheduler stopped")
			return
		case <-timer.C:
			s.runOnce(ctx)
			d := durationUntilHour(time.Now(), s.hour)
			timer.Reset(d)
			slog.Info("recurring scheduler sleeping until next run", "next_run_in", d.String())
		}
	}
}

// durationUntilHour returns the duration from now until the next occurrence of
// the given local-time hour.
func durationUntilHour(now time.Time, hour int) time.Duration {
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next.Sub(now)
}

// runOnce processes due recurring invoices for every company. Per-company
// failures are logged and skipped so one company cannot abort the rest.
func (s *RecurringScheduler) runOnce(ctx context.Context) {
	companies, err := s.companies.List(ctx)
	if err != nil {
		slog.Error("recurring scheduler: failed to list companies", "error", err)
		return
	}

	total, totalSent := 0, 0
	for i := range companies {
		co := companies[i]
		count, err := s.recurring.ProcessDue(ctx, co.ID)
		if err != nil {
			slog.Error("recurring scheduler: failed to process company", "error", err, "company_id", co.ID)
			continue
		}
		if count > 0 {
			slog.Info("recurring scheduler: generated invoices", "company_id", co.ID, "count", count)
		}
		total += count

		// Email the unsent drafts of auto-send templates (this run's and any
		// earlier failures). Independent of generation so a generate error for
		// one company doesn't block sending for it or others.
		sent, err := s.recurring.SweepAutoSend(ctx, co.ID)
		if err != nil {
			slog.Error("recurring scheduler: auto-send sweep failed", "error", err, "company_id", co.ID)
			continue
		}
		if sent > 0 {
			slog.Info("recurring scheduler: auto-sent invoices", "company_id", co.ID, "count", sent)
		}
		totalSent += sent
	}
	slog.Info("recurring scheduler run complete", "companies", len(companies), "generated", total, "auto_sent", totalSent)
}
