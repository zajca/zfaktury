package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/fakturoid"
)

func TestMapSubjectToContact(t *testing.T) {
	subj := fakturoid.Subject{
		ID:             42,
		Name:           "Acme s.r.o.",
		RegistrationNo: "12345678",
		VatNo:          "CZ12345678",
		Street:         "Hlavni 1",
		City:           "Praha",
		Zip:            "11000",
		Country:        "CZ",
		BankAccount:    "12345/0100",
		IBAN:           "CZ6508000000192000145399",
		Email:          "info@acme.cz",
		Phone:          "+420123456789",
		Web:            "https://acme.cz",
		Due:            14,
	}

	contact := mapSubjectToContact(subj)

	if contact.Name != "Acme s.r.o." {
		t.Errorf("Name = %q, want %q", contact.Name, "Acme s.r.o.")
	}
	if contact.ICO != "12345678" {
		t.Errorf("ICO = %q, want %q", contact.ICO, "12345678")
	}
	if contact.DIC != "CZ12345678" {
		t.Errorf("DIC = %q, want %q", contact.DIC, "CZ12345678")
	}
	if contact.Street != "Hlavni 1" {
		t.Errorf("Street = %q, want %q", contact.Street, "Hlavni 1")
	}
	if contact.City != "Praha" {
		t.Errorf("City = %q, want %q", contact.City, "Praha")
	}
	if contact.ZIP != "11000" {
		t.Errorf("ZIP = %q, want %q", contact.ZIP, "11000")
	}
	if contact.Country != "CZ" {
		t.Errorf("Country = %q, want %q", contact.Country, "CZ")
	}
	if contact.IBAN != "CZ6508000000192000145399" {
		t.Errorf("IBAN = %q, want %q", contact.IBAN, "CZ6508000000192000145399")
	}
	if contact.Email != "info@acme.cz" {
		t.Errorf("Email = %q, want %q", contact.Email, "info@acme.cz")
	}
	if contact.Phone != "+420123456789" {
		t.Errorf("Phone = %q, want %q", contact.Phone, "+420123456789")
	}
	if contact.Web != "https://acme.cz" {
		t.Errorf("Web = %q, want %q", contact.Web, "https://acme.cz")
	}
	if contact.Type != domain.ContactTypeCompany {
		t.Errorf("Type = %q, want %q", contact.Type, domain.ContactTypeCompany)
	}
	if contact.PaymentTermsDays != 14 {
		t.Errorf("PaymentTermsDays = %d, want 14", contact.PaymentTermsDays)
	}

	// Bank account parsing: "12345/0100" -> BankAccount="12345", BankCode="0100"
	if contact.BankAccount != "12345" {
		t.Errorf("BankAccount = %q, want %q", contact.BankAccount, "12345")
	}
	if contact.BankCode != "0100" {
		t.Errorf("BankCode = %q, want %q", contact.BankCode, "0100")
	}
}

func TestMapSubjectToContact_NoBankAccount(t *testing.T) {
	subj := fakturoid.Subject{
		ID:   1,
		Name: "No Bank Inc.",
	}

	contact := mapSubjectToContact(subj)

	if contact.BankAccount != "" {
		t.Errorf("BankAccount = %q, want empty", contact.BankAccount)
	}
	if contact.BankCode != "" {
		t.Errorf("BankCode = %q, want empty", contact.BankCode)
	}
}

func TestMapSubjectToContact_BankAccountWithoutCode(t *testing.T) {
	subj := fakturoid.Subject{
		ID:          1,
		Name:        "Partial Bank",
		BankAccount: "999888777",
	}

	contact := mapSubjectToContact(subj)

	if contact.BankAccount != "999888777" {
		t.Errorf("BankAccount = %q, want %q", contact.BankAccount, "999888777")
	}
	if contact.BankCode != "" {
		t.Errorf("BankCode = %q, want empty (no slash in input)", contact.BankCode)
	}
}

func TestMapFakturoidInvoice(t *testing.T) {
	subjectMap := map[int64]int64{
		100: 5, // fakturoid subject 100 -> local contact 5
	}

	inv := fakturoid.Invoice{
		ID:                    42,
		Number:                "2024-001",
		DocumentType:          "invoice",
		Status:                "paid",
		IssuedOn:              "2024-03-15",
		DueOn:                 "2024-04-15",
		TaxableFulfillmentDue: "2024-03-15",
		VariableSymbol:        "2024001",
		SubjectID:             100,
		Currency:              "CZK",
		ExchangeRate:          1.0,
		Subtotal:              10000.0,
		Total:                 12100.0,
		Note:                  "Test invoice",
		Lines: []fakturoid.InvoiceLine{
			{
				Name:      "Consulting",
				Quantity:  2.0,
				UnitName:  "hours",
				UnitPrice: 5000.0,
				VatRate:   21.0,
			},
		},
		Payments: []fakturoid.Payment{
			{PaidOn: "2024-04-01"},
		},
	}

	invoice := mapFakturoidInvoice(inv, subjectMap)

	// Document type mapping
	if invoice.Type != domain.InvoiceTypeRegular {
		t.Errorf("Type = %q, want %q", invoice.Type, domain.InvoiceTypeRegular)
	}

	// Status mapping
	if invoice.Status != domain.InvoiceStatusPaid {
		t.Errorf("Status = %q, want %q", invoice.Status, domain.InvoiceStatusPaid)
	}

	// Basic fields
	if invoice.InvoiceNumber != "2024-001" {
		t.Errorf("InvoiceNumber = %q, want %q", invoice.InvoiceNumber, "2024-001")
	}
	if invoice.VariableSymbol != "2024001" {
		t.Errorf("VariableSymbol = %q, want %q", invoice.VariableSymbol, "2024001")
	}
	if invoice.CurrencyCode != "CZK" {
		t.Errorf("CurrencyCode = %q, want %q", invoice.CurrencyCode, "CZK")
	}
	if invoice.Notes != "Test invoice" {
		t.Errorf("Notes = %q, want %q", invoice.Notes, "Test invoice")
	}

	// Date parsing
	expectedIssue := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	if !invoice.IssueDate.Equal(expectedIssue) {
		t.Errorf("IssueDate = %v, want %v", invoice.IssueDate, expectedIssue)
	}
	expectedDue := time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC)
	if !invoice.DueDate.Equal(expectedDue) {
		t.Errorf("DueDate = %v, want %v", invoice.DueDate, expectedDue)
	}
	expectedDelivery := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	if !invoice.DeliveryDate.Equal(expectedDelivery) {
		t.Errorf("DeliveryDate = %v, want %v", invoice.DeliveryDate, expectedDelivery)
	}

	// Paid date from first payment
	if invoice.PaidAt == nil {
		t.Fatal("PaidAt should not be nil for paid invoice")
	}
	expectedPaid := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	if !invoice.PaidAt.Equal(expectedPaid) {
		t.Errorf("PaidAt = %v, want %v", *invoice.PaidAt, expectedPaid)
	}

	// Customer resolved via subject map
	if invoice.CustomerID != 5 {
		t.Errorf("CustomerID = %d, want 5 (resolved from subjectMap)", invoice.CustomerID)
	}

	// Line items
	if len(invoice.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(invoice.Items))
	}
	item := invoice.Items[0]
	if item.Description != "Consulting" {
		t.Errorf("Items[0].Description = %q, want %q", item.Description, "Consulting")
	}
	if item.Unit != "hours" {
		t.Errorf("Items[0].Unit = %q, want %q", item.Unit, "hours")
	}
	if item.VATRatePercent != 21 {
		t.Errorf("Items[0].VATRatePercent = %d, want 21", item.VATRatePercent)
	}
	if item.SortOrder != 1 {
		t.Errorf("Items[0].SortOrder = %d, want 1", item.SortOrder)
	}
	// Quantity: domain.FromFloat(2.0) = 200
	if item.Quantity != domain.FromFloat(2.0) {
		t.Errorf("Items[0].Quantity = %d, want %d", item.Quantity, domain.FromFloat(2.0))
	}
	// UnitPrice: domain.FromFloat(5000.0) = 500000
	if item.UnitPrice != domain.FromFloat(5000.0) {
		t.Errorf("Items[0].UnitPrice = %d, want %d", item.UnitPrice, domain.FromFloat(5000.0))
	}

	// CalculateTotals should have been called -- verify totals are computed from items
	if invoice.SubtotalAmount == 0 {
		t.Error("SubtotalAmount should be non-zero after CalculateTotals")
	}
	if invoice.TotalAmount == 0 {
		t.Error("TotalAmount should be non-zero after CalculateTotals")
	}
}

func TestMapFakturoidInvoice_DocumentTypes(t *testing.T) {
	subjectMap := map[int64]int64{}

	tests := []struct {
		docType  string
		wantType string
	}{
		{"invoice", domain.InvoiceTypeRegular},
		{"proforma", domain.InvoiceTypeProforma},
		{"credit_note", domain.InvoiceTypeCreditNote},
		{"unknown", domain.InvoiceTypeRegular},
	}

	for _, tt := range tests {
		t.Run(tt.docType, func(t *testing.T) {
			inv := fakturoid.Invoice{DocumentType: tt.docType}
			result := mapFakturoidInvoice(inv, subjectMap)
			if result.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", result.Type, tt.wantType)
			}
		})
	}
}

func TestMapFakturoidInvoice_StatusMapping(t *testing.T) {
	subjectMap := map[int64]int64{}

	tests := []struct {
		status     string
		wantStatus string
	}{
		{"paid", domain.InvoiceStatusPaid},
		{"overdue", domain.InvoiceStatusOverdue},
		{"cancelled", domain.InvoiceStatusCancelled},
		{"sent", domain.InvoiceStatusSent},
		{"open", domain.InvoiceStatusSent},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			inv := fakturoid.Invoice{Status: tt.status}
			result := mapFakturoidInvoice(inv, subjectMap)
			if result.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", result.Status, tt.wantStatus)
			}
		})
	}
}

func TestMapFakturoidInvoice_UnresolvedSubject(t *testing.T) {
	subjectMap := map[int64]int64{}
	inv := fakturoid.Invoice{SubjectID: 999}
	result := mapFakturoidInvoice(inv, subjectMap)

	if result.CustomerID != 999 {
		t.Errorf("CustomerID = %d, want 999 (unresolved fakturoid ID)", result.CustomerID)
	}
}

func TestMapFakturoidExpense(t *testing.T) {
	subjectMap := map[int64]int64{
		200: 10, // fakturoid subject 200 -> local contact 10
	}

	exp := fakturoid.Expense{
		ID:             77,
		OriginalNumber: "FV-2024-100",
		IssuedOn:       "2024-06-01",
		SubjectID:      200,
		Description:    "Office supplies",
		Total:          1210.0,
		Currency:       "CZK",
		ExchangeRate:   1.0,
		PaymentMethod:  "cash",
		PrivateNote:    "Internal note",
		Lines: []fakturoid.ExpenseLine{
			{
				Name:      "Paper",
				Quantity:  10.0,
				UnitPrice: 100.0,
				VatRate:   21.0,
			},
		},
	}

	expense := mapFakturoidExpense(exp, subjectMap)

	if expense.ExpenseNumber != "FV-2024-100" {
		t.Errorf("ExpenseNumber = %q, want %q", expense.ExpenseNumber, "FV-2024-100")
	}
	if expense.Description != "Office supplies" {
		t.Errorf("Description = %q, want %q", expense.Description, "Office supplies")
	}
	if expense.Amount != domain.FromFloat(1210.0) {
		t.Errorf("Amount = %d, want %d", expense.Amount, domain.FromFloat(1210.0))
	}
	if expense.CurrencyCode != "CZK" {
		t.Errorf("CurrencyCode = %q, want %q", expense.CurrencyCode, "CZK")
	}
	if expense.Notes != "Internal note" {
		t.Errorf("Notes = %q, want %q", expense.Notes, "Internal note")
	}
	if !expense.IsTaxDeductible {
		t.Error("IsTaxDeductible should be true")
	}
	if expense.BusinessPercent != 100 {
		t.Errorf("BusinessPercent = %d, want 100", expense.BusinessPercent)
	}

	// Date parsing
	expectedDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	if !expense.IssueDate.Equal(expectedDate) {
		t.Errorf("IssueDate = %v, want %v", expense.IssueDate, expectedDate)
	}

	// Payment method
	if expense.PaymentMethod != "cash" {
		t.Errorf("PaymentMethod = %q, want %q", expense.PaymentMethod, "cash")
	}

	// Vendor resolved via subject map
	if expense.VendorID == nil {
		t.Fatal("VendorID should not be nil")
	}
	if *expense.VendorID != 10 {
		t.Errorf("VendorID = %d, want 10 (resolved from subjectMap)", *expense.VendorID)
	}

	// VAT calculation from lines
	if expense.VATRatePercent != 21 {
		t.Errorf("VATRatePercent = %d, want 21", expense.VATRatePercent)
	}
	if expense.VATAmount == 0 {
		t.Error("VATAmount should be non-zero for lines with VAT")
	}
}

func TestMapFakturoidExpense_DescriptionFallbackToLineName(t *testing.T) {
	exp := fakturoid.Expense{
		ID:          1,
		Description: "", // empty description
		Lines: []fakturoid.ExpenseLine{
			{Name: "First line item", Quantity: 1, UnitPrice: 100, VatRate: 0},
		},
	}

	expense := mapFakturoidExpense(exp, map[int64]int64{})

	if expense.Description != "First line item" {
		t.Errorf("Description = %q, want %q (fallback to first line name)", expense.Description, "First line item")
	}
}

func TestMapFakturoidExpense_DescriptionFallbackDefault(t *testing.T) {
	exp := fakturoid.Expense{
		ID:          1,
		Description: "",
		Lines:       nil, // no lines either
	}

	expense := mapFakturoidExpense(exp, map[int64]int64{})

	if expense.Description != "Import z Fakturoidu" {
		t.Errorf("Description = %q, want %q (default fallback)", expense.Description, "Import z Fakturoidu")
	}
}

func TestMapFakturoidExpense_PaymentMethodDefault(t *testing.T) {
	exp := fakturoid.Expense{
		ID:            1,
		PaymentMethod: "bank", // anything except "cash"
	}

	expense := mapFakturoidExpense(exp, map[int64]int64{})

	if expense.PaymentMethod != "bank_transfer" {
		t.Errorf("PaymentMethod = %q, want %q", expense.PaymentMethod, "bank_transfer")
	}
}

func TestMapFakturoidExpense_NoVendor(t *testing.T) {
	exp := fakturoid.Expense{
		ID:        1,
		SubjectID: 0, // no vendor
	}

	expense := mapFakturoidExpense(exp, map[int64]int64{})

	if expense.VendorID != nil {
		t.Errorf("VendorID = %v, want nil (no subject)", expense.VendorID)
	}
}

func TestMapFakturoidExpense_VATFromMultipleLines(t *testing.T) {
	exp := fakturoid.Expense{
		ID:    1,
		Total: 1500.0,
		Lines: []fakturoid.ExpenseLine{
			{Name: "Item A", Quantity: 1, UnitPrice: 1000, VatRate: 21},
			{Name: "Item B", Quantity: 1, UnitPrice: 500, VatRate: 12},
		},
	}

	expense := mapFakturoidExpense(exp, map[int64]int64{})

	// Dominant rate should be 21 (larger amount: 1000 > 500)
	if expense.VATRatePercent != 21 {
		t.Errorf("VATRatePercent = %d, want 21 (dominant rate)", expense.VATRatePercent)
	}
	if expense.VATAmount == 0 {
		t.Error("VATAmount should be non-zero for lines with VAT")
	}
}

func TestImportAll_ImportsNewSkipsDuplicates(t *testing.T) {
	client := &fakturoidMockClient{
		subjects: []fakturoid.Subject{
			{ID: 1, Name: "New Company", RegistrationNo: "11111111"},
			{ID: 2, Name: "Existing Co", RegistrationNo: "22222222"},
		},
		invoices: []fakturoid.Invoice{
			{ID: 10, Number: "INV-001", SubjectID: 1, DocumentType: "invoice", Status: "sent", IssuedOn: "2024-01-01", DueOn: "2024-02-01",
				Lines: []fakturoid.InvoiceLine{{Name: "Service", Quantity: 1, UnitPrice: 1000, VatRate: 21}}},
		},
		expenses: []fakturoid.Expense{
			{ID: 20, OriginalNumber: "EXP-001", Description: "Office", Total: 500, IssuedOn: "2024-01-15"},
		},
	}

	contactRepo := &fakturoidMockContactRepo{
		findByICOResult: map[string]*domain.Contact{
			"22222222": {ID: 99, Name: "Existing Co", ICO: "22222222"},
		},
	}

	importRepo := &fakturoidMockImportRepo{}
	invoiceRepo := &fakturoidMockInvoiceRepo{}
	expenseRepo := &fakturoidMockExpenseRepo{}

	contactSvc := &ContactService{repo: contactRepo}
	invoiceSvc := &InvoiceService{repo: invoiceRepo, contacts: contactSvc}
	expenseSvc := &ExpenseService{repo: expenseRepo}

	svc := NewFakturoidImportService(
		importRepo, contactRepo, invoiceRepo, expenseRepo,
		contactSvc, invoiceSvc, expenseSvc, nil, nil,
	)

	result, err := svc.ImportAll(context.Background(), client, false)
	if err != nil {
		t.Fatalf("ImportAll failed: %v", err)
	}

	// 1 new contact created, 1 duplicate skipped
	if result.ContactsCreated != 1 {
		t.Errorf("ContactsCreated = %d, want 1", result.ContactsCreated)
	}
	if result.ContactsSkipped != 1 {
		t.Errorf("ContactsSkipped = %d, want 1", result.ContactsSkipped)
	}

	// 1 new invoice created
	if result.InvoicesCreated != 1 {
		t.Errorf("InvoicesCreated = %d, want 1", result.InvoicesCreated)
	}

	// 1 new expense created
	if result.ExpensesCreated != 1 {
		t.Errorf("ExpensesCreated = %d, want 1", result.ExpensesCreated)
	}
}

// Mock types for Fakturoid import tests

type fakturoidMockClient struct {
	subjects       []fakturoid.Subject
	invoices       []fakturoid.Invoice
	expenses       []fakturoid.Expense
	callCount      int
	attachmentData []byte
	attachmentErr  error
}

func (m *fakturoidMockClient) ListSubjects(_ context.Context) ([]fakturoid.Subject, error) {
	m.callCount++
	return m.subjects, nil
}

func (m *fakturoidMockClient) ListInvoices(_ context.Context) ([]fakturoid.Invoice, error) {
	m.callCount++
	return m.invoices, nil
}

func (m *fakturoidMockClient) ListExpenses(_ context.Context) ([]fakturoid.Expense, error) {
	m.callCount++
	return m.expenses, nil
}

func (m *fakturoidMockClient) DownloadAttachment(_ context.Context, _ string) ([]byte, string, error) {
	if m.attachmentErr != nil {
		return nil, "", m.attachmentErr
	}
	if m.attachmentData != nil {
		return m.attachmentData, "application/pdf", nil
	}
	return nil, "", fmt.Errorf("not implemented in test")
}

type fakturoidMockImportRepo struct{}

func (m *fakturoidMockImportRepo) Create(_ context.Context, _ *domain.FakturoidImportLog) error {
	return nil
}
func (m *fakturoidMockImportRepo) FindByFakturoidID(_ context.Context, _ string, _ int64) (*domain.FakturoidImportLog, error) {
	return nil, nil
}
func (m *fakturoidMockImportRepo) ListByEntityType(_ context.Context, _ string) ([]domain.FakturoidImportLog, error) {
	return nil, nil
}

type fakturoidMockContactRepo struct {
	findByICOResult map[string]*domain.Contact
}

func (m *fakturoidMockContactRepo) Create(_ context.Context, c *domain.Contact) error {
	c.ID = 1000 // assign a fake ID
	return nil
}
func (m *fakturoidMockContactRepo) Update(_ context.Context, _ *domain.Contact) error { return nil }
func (m *fakturoidMockContactRepo) Delete(_ context.Context, _ int64) error           { return nil }
func (m *fakturoidMockContactRepo) GetByID(_ context.Context, _ int64) (*domain.Contact, error) {
	return nil, nil
}
func (m *fakturoidMockContactRepo) FindByICO(_ context.Context, ico string) (*domain.Contact, error) {
	if m.findByICOResult != nil {
		if c, ok := m.findByICOResult[ico]; ok {
			return c, nil
		}
	}
	return nil, nil
}
func (m *fakturoidMockContactRepo) List(_ context.Context, _ domain.ContactFilter) ([]domain.Contact, int, error) {
	return nil, 0, nil
}

type fakturoidMockInvoiceRepo struct{}

func (m *fakturoidMockInvoiceRepo) Create(_ context.Context, inv *domain.Invoice) error {
	inv.ID = 2000
	return nil
}
func (m *fakturoidMockInvoiceRepo) Update(_ context.Context, _ *domain.Invoice) error { return nil }
func (m *fakturoidMockInvoiceRepo) Delete(_ context.Context, _ int64) error           { return nil }
func (m *fakturoidMockInvoiceRepo) GetByID(_ context.Context, _ int64) (*domain.Invoice, error) {
	return nil, nil
}
func (m *fakturoidMockInvoiceRepo) List(_ context.Context, _ domain.InvoiceFilter) ([]domain.Invoice, int, error) {
	return nil, 0, nil
}
func (m *fakturoidMockInvoiceRepo) UpdateStatus(_ context.Context, _ int64, _ string) error {
	return nil
}
func (m *fakturoidMockInvoiceRepo) GetNextNumber(_ context.Context, _ int64) (string, error) {
	return "INV-001", nil
}
func (m *fakturoidMockInvoiceRepo) FindByRelatedInvoice(_ context.Context, _ int64, _ string) (*domain.Invoice, error) {
	return nil, nil
}
func (m *fakturoidMockInvoiceRepo) GetRelatedInvoices(_ context.Context, _ int64) ([]domain.Invoice, error) {
	return nil, nil
}

type fakturoidMockExpenseRepo struct{}

func (m *fakturoidMockExpenseRepo) Create(_ context.Context, exp *domain.Expense) error {
	exp.ID = 3000
	return nil
}
func (m *fakturoidMockExpenseRepo) Update(_ context.Context, _ *domain.Expense) error { return nil }
func (m *fakturoidMockExpenseRepo) Delete(_ context.Context, _ int64) error           { return nil }
func (m *fakturoidMockExpenseRepo) GetByID(_ context.Context, _ int64) (*domain.Expense, error) {
	return nil, nil
}
func (m *fakturoidMockExpenseRepo) List(_ context.Context, _ domain.ExpenseFilter) ([]domain.Expense, int, error) {
	return nil, 0, nil
}
func (m *fakturoidMockExpenseRepo) MarkTaxReviewed(_ context.Context, _ []int64) error {
	return nil
}
func (m *fakturoidMockExpenseRepo) UnmarkTaxReviewed(_ context.Context, _ []int64) error {
	return nil
}

// Mock document repos for attachment download tests

type fakturoidMockDocumentRepo struct {
	docs  []domain.ExpenseDocument
	count int
}

func (m *fakturoidMockDocumentRepo) Create(_ context.Context, doc *domain.ExpenseDocument) error {
	doc.ID = int64(len(m.docs)) + 1
	m.docs = append(m.docs, *doc)
	return nil
}
func (m *fakturoidMockDocumentRepo) GetByID(_ context.Context, _ int64) (*domain.ExpenseDocument, error) {
	return nil, nil
}
func (m *fakturoidMockDocumentRepo) ListByExpenseID(_ context.Context, _ int64) ([]domain.ExpenseDocument, error) {
	return nil, nil
}
func (m *fakturoidMockDocumentRepo) Delete(_ context.Context, _ int64) error { return nil }
func (m *fakturoidMockDocumentRepo) CountByExpenseID(_ context.Context, _ int64) (int, error) {
	return m.count, nil
}

type fakturoidMockInvDocumentRepo struct {
	docs  []domain.InvoiceDocument
	count int
}

func (m *fakturoidMockInvDocumentRepo) Create(_ context.Context, doc *domain.InvoiceDocument) error {
	doc.ID = int64(len(m.docs)) + 1
	m.docs = append(m.docs, *doc)
	return nil
}
func (m *fakturoidMockInvDocumentRepo) GetByID(_ context.Context, _ int64) (*domain.InvoiceDocument, error) {
	return nil, nil
}
func (m *fakturoidMockInvDocumentRepo) ListByInvoiceID(_ context.Context, _ int64) ([]domain.InvoiceDocument, error) {
	return nil, nil
}
func (m *fakturoidMockInvDocumentRepo) Delete(_ context.Context, _ int64) error { return nil }
func (m *fakturoidMockInvDocumentRepo) CountByInvoiceID(_ context.Context, _ int64) (int, error) {
	return m.count, nil
}

func TestImportAll_DownloadsAttachments(t *testing.T) {
	pdfData := []byte("%PDF-1.4 test content")

	client := &fakturoidMockClient{
		subjects: []fakturoid.Subject{
			{ID: 1, Name: "Test Co", RegistrationNo: "99999999"},
		},
		invoices: []fakturoid.Invoice{
			{
				ID:       10,
				Number:   "INV-ATT-001",
				IssuedOn: "2024-01-15",
				DueOn:    "2024-02-15",
				Lines:    []fakturoid.InvoiceLine{{Name: "Test", Quantity: 1, UnitPrice: 1000}},
				Attachments: []fakturoid.Attachment{
					{ID: 100, Filename: "receipt.pdf", ContentType: "application/pdf", DownloadURL: "https://example.com/att/100"},
				},
				SubjectID: 1,
			},
		},
		expenses: []fakturoid.Expense{
			{
				ID:          20,
				IssuedOn:    "2024-01-20",
				Description: "Office supplies",
				Total:       500,
				Lines:       []fakturoid.ExpenseLine{{Name: "Pens", Quantity: 1, UnitPrice: 500}},
				Attachments: []fakturoid.Attachment{
					{ID: 200, Filename: "receipt.pdf", ContentType: "application/pdf", DownloadURL: "https://example.com/att/200"},
					{ID: 201, Filename: "contract.pdf", ContentType: "application/pdf", DownloadURL: "https://example.com/att/201"},
				},
			},
		},
		attachmentData: pdfData,
	}

	contactRepo := &fakturoidMockContactRepo{}
	importRepo := &fakturoidMockImportRepo{}
	invoiceRepo := &fakturoidMockInvoiceRepo{}
	expenseRepo := &fakturoidMockExpenseRepo{}
	docRepo := &fakturoidMockDocumentRepo{}
	invDocRepo := &fakturoidMockInvDocumentRepo{}

	dataDir := t.TempDir()

	contactSvc := &ContactService{repo: contactRepo}
	invoiceSvc := &InvoiceService{repo: invoiceRepo, contacts: contactSvc}
	expenseSvc := &ExpenseService{repo: expenseRepo}
	documentSvc := NewDocumentService(docRepo, dataDir, nil)
	invDocumentSvc := NewInvoiceDocumentService(invDocRepo, dataDir, nil)

	svc := NewFakturoidImportService(
		importRepo, contactRepo, invoiceRepo, expenseRepo,
		contactSvc, invoiceSvc, expenseSvc, documentSvc, invDocumentSvc,
	)

	result, err := svc.ImportAll(context.Background(), client, true)
	if err != nil {
		t.Fatalf("ImportAll with attachments failed: %v", err)
	}

	// 1 invoice attachment + 2 expense attachments = 3 total
	if result.AttachmentsDownloaded != 3 {
		t.Errorf("AttachmentsDownloaded = %d, want 3", result.AttachmentsDownloaded)
	}
	if result.AttachmentsSkipped != 0 {
		t.Errorf("AttachmentsSkipped = %d, want 0", result.AttachmentsSkipped)
	}
}

func TestImportAll_AttachmentDownloadError(t *testing.T) {
	client := &fakturoidMockClient{
		subjects: []fakturoid.Subject{
			{ID: 1, Name: "Error Test Co", RegistrationNo: "88888888"},
		},
		invoices: []fakturoid.Invoice{
			{
				ID:        10,
				Number:    "INV-ERR-001",
				IssuedOn:  "2024-01-15",
				DueOn:     "2024-02-15",
				SubjectID: 1,
				Lines:     []fakturoid.InvoiceLine{{Name: "Test", Quantity: 1, UnitPrice: 1000}},
				Attachments: []fakturoid.Attachment{
					{ID: 100, Filename: "receipt.pdf", DownloadURL: "https://example.com/att/100"},
				},
			},
		},
		expenses:      []fakturoid.Expense{},
		attachmentErr: fmt.Errorf("network error"),
	}

	contactRepo := &fakturoidMockContactRepo{}
	importRepo := &fakturoidMockImportRepo{}
	invoiceRepo := &fakturoidMockInvoiceRepo{}
	expenseRepo := &fakturoidMockExpenseRepo{}

	dataDir := t.TempDir()
	invDocRepo := &fakturoidMockInvDocumentRepo{}
	invDocumentSvc := NewInvoiceDocumentService(invDocRepo, dataDir, nil)

	contactSvc := &ContactService{repo: contactRepo}
	invoiceSvc := &InvoiceService{repo: invoiceRepo, contacts: contactSvc}
	expenseSvc := &ExpenseService{repo: expenseRepo}

	svc := NewFakturoidImportService(
		importRepo, contactRepo, invoiceRepo, expenseRepo,
		contactSvc, invoiceSvc, expenseSvc, nil, invDocumentSvc,
	)

	result, err := svc.ImportAll(context.Background(), client, true)
	if err != nil {
		t.Fatalf("ImportAll failed: %v", err)
	}

	if result.AttachmentsDownloaded != 0 {
		t.Errorf("AttachmentsDownloaded = %d, want 0", result.AttachmentsDownloaded)
	}
	if result.AttachmentsSkipped != 1 {
		t.Errorf("AttachmentsSkipped = %d, want 1", result.AttachmentsSkipped)
	}
}
