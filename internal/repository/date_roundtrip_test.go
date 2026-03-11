package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

// TestInvoiceDateRoundtrip verifies that all date/time fields survive a Create->GetByID cycle.
func TestInvoiceDateRoundtrip(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	issueDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	dueDate := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
	deliveryDate := time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)
	sentAt := time.Date(2026, 3, 16, 10, 30, 0, 0, time.UTC)
	paidAt := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)

	invoiceRepoCounter++
	inv := &domain.Invoice{
		SequenceID:    seqID,
		InvoiceNumber: fmt.Sprintf("FV2026%04d", invoiceRepoCounter),
		Type:          domain.InvoiceTypeRegular,
		Status:        domain.InvoiceStatusDraft,
		IssueDate:     issueDate,
		DueDate:       dueDate,
		DeliveryDate:  deliveryDate,
		CustomerID:    customerID,
		CurrencyCode:  domain.CurrencyCZK,
		ExchangeRate:  100,
		PaymentMethod: "bank_transfer",
		SentAt:        &sentAt,
		PaidAt:        &paidAt,
		Items: []domain.InvoiceItem{
			{Description: "Test", Quantity: 100, Unit: "ks", UnitPrice: 10000, VATRatePercent: 21},
		},
	}
	inv.CalculateTotals()

	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	// Date fields (date precision).
	assertDateEqual(t, "IssueDate", issueDate, got.IssueDate)
	assertDateEqual(t, "DueDate", dueDate, got.DueDate)
	assertDateEqual(t, "DeliveryDate", deliveryDate, got.DeliveryDate)

	// Timestamp fields (second precision).
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero after roundtrip")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero after roundtrip")
	}

	// Nullable timestamp fields.
	if got.SentAt == nil {
		t.Fatal("SentAt is nil after roundtrip")
	}
	assertTimestampEqual(t, "SentAt", sentAt, *got.SentAt)

	if got.PaidAt == nil {
		t.Fatal("PaidAt is nil after roundtrip")
	}
	assertTimestampEqual(t, "PaidAt", paidAt, *got.PaidAt)
}

// TestInvoiceUpdateDateRoundtrip verifies dates survive an Update->GetByID cycle.
func TestInvoiceUpdateDateRoundtrip(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	inv := makeRepoInvoice(customerID, seqID)
	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	newIssueDate := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	newDueDate := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	inv.IssueDate = newIssueDate
	inv.DueDate = newDueDate

	if err := repo.Update(ctx, inv); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	assertDateEqual(t, "IssueDate after update", newIssueDate, got.IssueDate)
	assertDateEqual(t, "DueDate after update", newDueDate, got.DueDate)
	if got.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero after update roundtrip")
	}
}

// TestInvoiceListDateRoundtrip verifies dates in List results.
func TestInvoiceListDateRoundtrip(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	issueDate := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	inv := makeRepoInvoice(customerID, seqID)
	inv.IssueDate = issueDate
	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	invoices, _, err := repo.List(ctx, domain.InvoiceFilter{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(invoices) == 0 {
		t.Fatal("expected at least 1 invoice in list")
	}

	assertDateEqual(t, "List IssueDate", issueDate, invoices[0].IssueDate)
	if invoices[0].CreatedAt.IsZero() {
		t.Error("List CreatedAt is zero")
	}
}

// TestExpenseDateRoundtrip verifies expense date fields survive Create->GetByID.
func TestExpenseDateRoundtrip(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	issueDate := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)

	e := &domain.Expense{
		Description:     "Date test expense",
		Amount:          domain.NewAmount(500, 0),
		IssueDate:       issueDate,
		CurrencyCode:    domain.CurrencyCZK,
		Category:        "supplies",
		BusinessPercent: 100,
		PaymentMethod:   "bank_transfer",
	}

	if err := repo.Create(ctx, e); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	assertDateEqual(t, "Expense IssueDate", issueDate, got.IssueDate)
	if got.CreatedAt.IsZero() {
		t.Error("Expense CreatedAt is zero after roundtrip")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("Expense UpdatedAt is zero after roundtrip")
	}
}

// TestExpenseUpdateDateRoundtrip verifies expense dates survive Update->GetByID.
func TestExpenseUpdateDateRoundtrip(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	seeded := testutil.SeedExpense(t, db, &domain.Expense{Description: "Update date test"})

	newIssueDate := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	seeded.IssueDate = newIssueDate

	if err := repo.Update(ctx, seeded); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	assertDateEqual(t, "Expense IssueDate after update", newIssueDate, got.IssueDate)
	if got.UpdatedAt.IsZero() {
		t.Error("Expense UpdatedAt is zero after update")
	}
}

// TestContactDateRoundtrip verifies contact timestamp fields survive Create->GetByID.
func TestContactDateRoundtrip(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	vatAt := time.Date(2026, 1, 15, 8, 0, 0, 0, time.UTC)
	c := &domain.Contact{
		Type:            domain.ContactTypeCompany,
		Name:            "Date Test s.r.o.",
		Country:         "CZ",
		VATUnreliableAt: &vatAt,
	}

	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.CreatedAt.IsZero() {
		t.Error("Contact CreatedAt is zero after roundtrip")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("Contact UpdatedAt is zero after roundtrip")
	}
	if got.VATUnreliableAt == nil {
		t.Fatal("Contact VATUnreliableAt is nil after roundtrip")
	}
	assertTimestampEqual(t, "Contact VATUnreliableAt", vatAt, *got.VATUnreliableAt)
}

// TestContactUpdateDateRoundtrip verifies contact dates survive Update->GetByID.
func TestContactUpdateDateRoundtrip(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	c := &domain.Contact{
		Type:    domain.ContactTypeCompany,
		Name:    "Update Date Test",
		Country: "CZ",
	}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	vatAt := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	c.VATUnreliableAt = &vatAt
	if err := repo.Update(ctx, c); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.VATUnreliableAt == nil {
		t.Fatal("VATUnreliableAt is nil after update roundtrip")
	}
	assertTimestampEqual(t, "VATUnreliableAt after update", vatAt, *got.VATUnreliableAt)
	if got.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero after update")
	}
}

// TestDocumentDateRoundtrip verifies document CreatedAt survives Create->GetByID.
func TestDocumentDateRoundtrip(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	expense := testutil.SeedExpense(t, db, nil)
	doc := &domain.ExpenseDocument{
		ExpenseID:   expense.ID,
		Filename:    "date_test.pdf",
		ContentType: "application/pdf",
		StoragePath: "/test/date_test.pdf",
		Size:        512,
	}

	if err := repo.Create(ctx, doc); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.CreatedAt.IsZero() {
		t.Error("Document CreatedAt is zero after roundtrip")
	}
}

// TestCategoryDateRoundtrip verifies category CreatedAt survives Create->GetByID.
func TestCategoryDateRoundtrip(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	cat := &domain.ExpenseCategory{
		Key:     "date_test_cat",
		LabelCS: "Datumovy test",
		LabelEN: "Date test",
	}

	if err := repo.Create(ctx, cat); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, cat.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.CreatedAt.IsZero() {
		t.Error("Category CreatedAt is zero after roundtrip")
	}
}

// TestRecurringInvoiceDateRoundtrip verifies all date fields on recurring invoices.
func TestRecurringInvoiceDateRoundtrip(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Date Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	nextIssue := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	ri := &domain.RecurringInvoice{
		Name:          "Date roundtrip test",
		CustomerID:    customer.ID,
		Frequency:     domain.FrequencyMonthly,
		NextIssueDate: nextIssue,
		EndDate:       &endDate,
		CurrencyCode:  domain.CurrencyCZK,
		ExchangeRate:  100,
		PaymentMethod: "bank_transfer",
		IsActive:      true,
		Items: []domain.RecurringInvoiceItem{
			{Description: "Test", Quantity: 100, Unit: "ks", UnitPrice: 10000, VATRatePercent: 21},
		},
	}

	if err := repo.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, ri.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	assertDateEqual(t, "RecurringInvoice NextIssueDate", nextIssue, got.NextIssueDate)
	if got.EndDate == nil {
		t.Fatal("RecurringInvoice EndDate is nil after roundtrip")
	}
	assertDateEqual(t, "RecurringInvoice EndDate", endDate, *got.EndDate)
	if got.CreatedAt.IsZero() {
		t.Error("RecurringInvoice CreatedAt is zero after roundtrip")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("RecurringInvoice UpdatedAt is zero after roundtrip")
	}
}

// TestRecurringExpenseDateRoundtrip verifies all date fields on recurring expenses.
func TestRecurringExpenseDateRoundtrip(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringExpenseRepository(db)
	ctx := context.Background()

	nextIssue := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)

	re := &domain.RecurringExpense{
		Name:            "Date roundtrip test",
		Description:     "Test",
		Amount:          domain.NewAmount(1000, 0),
		CurrencyCode:    domain.CurrencyCZK,
		Frequency:       "monthly",
		NextIssueDate:   nextIssue,
		EndDate:         &endDate,
		IsActive:        true,
		BusinessPercent: 100,
		PaymentMethod:   "bank_transfer",
	}

	if err := repo.Create(ctx, re); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, re.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	assertDateEqual(t, "RecurringExpense NextIssueDate", nextIssue, got.NextIssueDate)
	if got.EndDate == nil {
		t.Fatal("RecurringExpense EndDate is nil after roundtrip")
	}
	assertDateEqual(t, "RecurringExpense EndDate", endDate, *got.EndDate)
	if got.CreatedAt.IsZero() {
		t.Error("RecurringExpense CreatedAt is zero after roundtrip")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("RecurringExpense UpdatedAt is zero after roundtrip")
	}
}

// assertDateEqual compares two dates at day precision.
func assertDateEqual(t *testing.T, field string, want, got time.Time) {
	t.Helper()
	wantStr := want.Format("2006-01-02")
	gotStr := got.Format("2006-01-02")
	if wantStr != gotStr {
		t.Errorf("%s = %s, want %s", field, gotStr, wantStr)
	}
}

// assertTimestampEqual compares two timestamps at second precision.
func assertTimestampEqual(t *testing.T, field string, want, got time.Time) {
	t.Helper()
	wantStr := want.Format(time.RFC3339)
	gotStr := got.Format(time.RFC3339)
	if wantStr != gotStr {
		t.Errorf("%s = %s, want %s", field, gotStr, wantStr)
	}
}
