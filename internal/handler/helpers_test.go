package handler

import (
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// Invoices saved before DUZP existed hold a zero delivery date (0001-01-01 in
// the database); the JSON response must emit the effective taxable supply
// date, never the zero date.
func TestInvoiceFromDomain_ZeroDeliveryDateShowsIssueDate(t *testing.T) {
	inv := &domain.Invoice{
		IssueDate:    time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		DueDate:      time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
		DeliveryDate: time.Time{},
	}

	resp := invoiceFromDomain(inv)

	if resp.DeliveryDate != "2026-03-10" {
		t.Errorf("DeliveryDate = %q, want the issue date %q", resp.DeliveryDate, "2026-03-10")
	}
}
