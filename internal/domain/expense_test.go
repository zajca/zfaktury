package domain

import "testing"

func TestExpense_CalculateTotals_NoItems(t *testing.T) {
	e := &Expense{
		Amount:    50000,
		VATAmount: 8678,
	}
	e.CalculateTotals()

	// Should leave fields untouched when no items.
	if e.Amount != 50000 {
		t.Errorf("Amount = %d, want 50000", e.Amount)
	}
	if e.VATAmount != 8678 {
		t.Errorf("VATAmount = %d, want 8678", e.VATAmount)
	}
}

func TestExpense_CalculateTotals_SingleItemNoVAT(t *testing.T) {
	e := &Expense{
		Items: []ExpenseItem{
			{
				Description:    "Office supplies",
				Quantity:       100,   // 1.00
				UnitPrice:      50000, // 500.00 CZK
				VATRatePercent: 0,
			},
		},
	}

	e.CalculateTotals()

	// subtotal = 100 * 50000 / 100 = 50000
	if e.Amount != 50000 {
		t.Errorf("Amount = %d, want 50000", e.Amount)
	}
	if e.VATAmount != 0 {
		t.Errorf("VATAmount = %d, want 0", e.VATAmount)
	}
	if e.VATRatePercent != 0 {
		t.Errorf("VATRatePercent = %d, want 0", e.VATRatePercent)
	}
	if e.Items[0].TotalAmount != 50000 {
		t.Errorf("Item TotalAmount = %d, want 50000", e.Items[0].TotalAmount)
	}
}

func TestExpense_CalculateTotals_SingleItemWithVAT21(t *testing.T) {
	e := &Expense{
		Items: []ExpenseItem{
			{
				Description:    "Consulting",
				Quantity:       200,    // 2.00
				UnitPrice:      100000, // 1000.00 CZK
				VATRatePercent: 21,
			},
		},
	}

	e.CalculateTotals()

	// subtotal = 200 * 100000 / 100 = 200000
	// VAT = 200000 * 0.21 = 42000
	// total = 242000
	if e.VATAmount != 42000 {
		t.Errorf("VATAmount = %d, want 42000", e.VATAmount)
	}
	if e.Amount != 242000 {
		t.Errorf("Amount = %d, want 242000", e.Amount)
	}
	if e.VATRatePercent != 21 {
		t.Errorf("VATRatePercent = %d, want 21", e.VATRatePercent)
	}
	if e.Items[0].VATAmount != 42000 {
		t.Errorf("Item VATAmount = %d, want 42000", e.Items[0].VATAmount)
	}
	if e.Items[0].TotalAmount != 242000 {
		t.Errorf("Item TotalAmount = %d, want 242000", e.Items[0].TotalAmount)
	}
}

func TestExpense_CalculateTotals_MultipleItemsDominantRate(t *testing.T) {
	e := &Expense{
		Items: []ExpenseItem{
			{
				Description:    "Item A",
				Quantity:       100,   // 1.00
				UnitPrice:      10000, // 100.00 CZK
				VATRatePercent: 12,
			},
			{
				Description:    "Item B",
				Quantity:       300,   // 3.00
				UnitPrice:      20000, // 200.00 CZK
				VATRatePercent: 21,
			},
		},
	}

	e.CalculateTotals()

	// Item A: subtotal = 100*10000/100 = 10000, VAT = 10000*0.12 = 1200, total = 11200
	// Item B: subtotal = 300*20000/100 = 60000, VAT = 60000*0.21 = 12600, total = 72600
	expectedVAT := Amount(1200 + 12600)
	expectedAmount := Amount(11200 + 72600)

	if e.VATAmount != expectedVAT {
		t.Errorf("VATAmount = %d, want %d", e.VATAmount, expectedVAT)
	}
	if e.Amount != expectedAmount {
		t.Errorf("Amount = %d, want %d", e.Amount, expectedAmount)
	}
	// Item B has higher subtotal (60000 > 10000), so dominant rate should be 21.
	if e.VATRatePercent != 21 {
		t.Errorf("VATRatePercent = %d, want 21 (dominant rate)", e.VATRatePercent)
	}
}
