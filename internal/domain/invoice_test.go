package domain

import (
	"testing"
	"time"
)

func TestInvoice_CalculateTotals_NoVAT(t *testing.T) {
	inv := &Invoice{
		Items: []InvoiceItem{
			{
				Description:    "Service",
				Quantity:       100,   // 1.00
				UnitPrice:      50000, // 500.00 CZK
				VATRatePercent: 0,
			},
		},
	}

	inv.CalculateTotals()

	// subtotal = quantity * unit_price / 100 = 100 * 50000 / 100 = 50000
	if inv.SubtotalAmount != 50000 {
		t.Errorf("SubtotalAmount = %d, want 50000", inv.SubtotalAmount)
	}
	if inv.VATAmount != 0 {
		t.Errorf("VATAmount = %d, want 0", inv.VATAmount)
	}
	if inv.TotalAmount != 50000 {
		t.Errorf("TotalAmount = %d, want 50000", inv.TotalAmount)
	}
}

func TestInvoice_CalculateTotals_WithVAT21(t *testing.T) {
	inv := &Invoice{
		Items: []InvoiceItem{
			{
				Description:    "Consulting",
				Quantity:       200,    // 2.00
				UnitPrice:      100000, // 1000.00 CZK
				VATRatePercent: 21,
			},
		},
	}

	inv.CalculateTotals()

	// subtotal = 200 * 100000 / 100 = 200000 (2000.00 CZK)
	if inv.SubtotalAmount != 200000 {
		t.Errorf("SubtotalAmount = %d, want 200000", inv.SubtotalAmount)
	}

	// VAT = 200000 * 0.21 = 42000
	if inv.VATAmount != 42000 {
		t.Errorf("VATAmount = %d, want 42000", inv.VATAmount)
	}

	// Total = 200000 + 42000 = 242000
	if inv.TotalAmount != 242000 {
		t.Errorf("TotalAmount = %d, want 242000", inv.TotalAmount)
	}

	// Verify item-level amounts are set.
	if inv.Items[0].VATAmount != 42000 {
		t.Errorf("Item VATAmount = %d, want 42000", inv.Items[0].VATAmount)
	}
	if inv.Items[0].TotalAmount != 242000 {
		t.Errorf("Item TotalAmount = %d, want 242000", inv.Items[0].TotalAmount)
	}
}

func TestInvoice_CalculateTotals_MultipleItems(t *testing.T) {
	inv := &Invoice{
		Items: []InvoiceItem{
			{
				Description:    "Item A",
				Quantity:       100,   // 1.00
				UnitPrice:      10000, // 100.00 CZK
				VATRatePercent: 21,
			},
			{
				Description:    "Item B",
				Quantity:       300,  // 3.00
				UnitPrice:      5000, // 50.00 CZK
				VATRatePercent: 12,
			},
		},
	}

	inv.CalculateTotals()

	// Item A: subtotal = 100*10000/100 = 10000, VAT = 10000*0.21 = 2100, total = 12100
	// Item B: subtotal = 300*5000/100 = 15000, VAT = 15000*0.12 = 1800, total = 16800
	expectedSubtotal := Amount(10000 + 15000)
	expectedVAT := Amount(2100 + 1800)
	expectedTotal := Amount(12100 + 16800)

	if inv.SubtotalAmount != expectedSubtotal {
		t.Errorf("SubtotalAmount = %d, want %d", inv.SubtotalAmount, expectedSubtotal)
	}
	if inv.VATAmount != expectedVAT {
		t.Errorf("VATAmount = %d, want %d", inv.VATAmount, expectedVAT)
	}
	if inv.TotalAmount != expectedTotal {
		t.Errorf("TotalAmount = %d, want %d", inv.TotalAmount, expectedTotal)
	}
}

func TestInvoice_CalculateTotals_NoItems(t *testing.T) {
	inv := &Invoice{}
	inv.CalculateTotals()

	if inv.SubtotalAmount != 0 {
		t.Errorf("SubtotalAmount = %d, want 0", inv.SubtotalAmount)
	}
	if inv.VATAmount != 0 {
		t.Errorf("VATAmount = %d, want 0", inv.VATAmount)
	}
	if inv.TotalAmount != 0 {
		t.Errorf("TotalAmount = %d, want 0", inv.TotalAmount)
	}
}

func TestInvoice_IsOverdue(t *testing.T) {
	tests := []struct {
		name string
		inv  Invoice
		want bool
	}{
		{
			name: "past due date, draft",
			inv: Invoice{
				Status:  InvoiceStatusDraft,
				DueDate: time.Now().AddDate(0, 0, -1),
			},
			want: true,
		},
		{
			name: "future due date, sent",
			inv: Invoice{
				Status:  InvoiceStatusSent,
				DueDate: time.Now().AddDate(0, 0, 30),
			},
			want: false,
		},
		{
			name: "past due but paid",
			inv: Invoice{
				Status:  InvoiceStatusPaid,
				DueDate: time.Now().AddDate(0, 0, -10),
			},
			want: false,
		},
		{
			name: "past due but cancelled",
			inv: Invoice{
				Status:  InvoiceStatusCancelled,
				DueDate: time.Now().AddDate(0, 0, -10),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.inv.IsOverdue()
			if got != tt.want {
				t.Errorf("IsOverdue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvoice_IsPaid(t *testing.T) {
	tests := []struct {
		name string
		inv  Invoice
		want bool
	}{
		{
			name: "fully paid",
			inv:  Invoice{TotalAmount: 10000, PaidAmount: 10000},
			want: true,
		},
		{
			name: "overpaid",
			inv:  Invoice{TotalAmount: 10000, PaidAmount: 15000},
			want: true,
		},
		{
			name: "partially paid",
			inv:  Invoice{TotalAmount: 10000, PaidAmount: 5000},
			want: false,
		},
		{
			name: "not paid",
			inv:  Invoice{TotalAmount: 10000, PaidAmount: 0},
			want: false,
		},
		{
			name: "zero total zero paid",
			inv:  Invoice{TotalAmount: 0, PaidAmount: 0},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.inv.IsPaid()
			if got != tt.want {
				t.Errorf("IsPaid() = %v, want %v", got, tt.want)
			}
		})
	}
}
