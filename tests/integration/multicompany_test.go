//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/testutil"
)

// TestMultiCompanyEndToEnd exercises the full multi-company flow per the
// migration-025 spec:
//
//  1. Two companies are created via CompanyService and given contacts.
//  2. An invoice in company A referencing a company-A contact succeeds.
//  3. An invoice in company A referencing a company-B contact is rejected
//     (cross-company isolation: the customer lookup against company A returns
//     ErrNotFound).
//  4. Both companies create the same sequence prefix+year (FV2026) — the
//     UNIQUE(company_id, prefix, year) partition allows the collision.
//  5. Soft-deleting company A while it still owns a contact fails with
//     ErrInUse.
//  6. After the invoice and contact in company A are soft-deleted, deleting
//     company A succeeds; company B remains listable.
//  7. Deleting the last remaining company (B) fails with ErrLastCompany.
func TestMultiCompanyEndToEnd(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t)

	// The default test DB seeds company id=1 ("Test Company"). We soft-delete
	// it so the active list starts empty, then create company A and B via
	// CompanyService. Soft-delete just sets deleted_at — the row stays in
	// place so existing FKs (e.g. from settings) keep resolving, but List
	// excludes it.
	if _, err := db.ExecContext(ctx, `UPDATE companies SET deleted_at = ? WHERE id = 1`, time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatalf("soft-deleting default company: %v", err)
	}

	// --- Repository + service wiring -------------------------------------------------

	companyRepo := repository.NewCompanyRepository(db)
	contactRepo := repository.NewContactRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	sequenceRepo := repository.NewSequenceRepository(db)

	contactSvc := service.NewContactService(contactRepo, nil, nil)
	sequenceSvc := service.NewSequenceService(sequenceRepo, nil)
	contactService := contactSvc
	invoiceSvc := service.NewInvoiceService(invoiceRepo, contactService, sequenceSvc, nil)

	// Only the contact checker is registered: this is sufficient to drive
	// the ErrInUse branch when company A still owns a contact.
	checkers := []service.EntityChecker{
		service.NewContactCompanyChecker(contactRepo),
	}
	companySvc := service.NewCompanyService(companyRepo, checkers, nil)

	// --- Step 1: Create two companies ------------------------------------------------

	companyA := domain.Company{
		Name:      "Company A",
		LegalName: "Company A s.r.o.",
		ICO:       "11111111",
	}
	companyAID, err := companySvc.Create(ctx, companyA)
	if err != nil {
		t.Fatalf("creating company A: %v", err)
	}

	companyB := domain.Company{
		Name:      "Company B",
		LegalName: "Company B s.r.o.",
		ICO:       "22222222",
	}
	companyBID, err := companySvc.Create(ctx, companyB)
	if err != nil {
		t.Fatalf("creating company B: %v", err)
	}

	if companyAID == companyBID {
		t.Fatalf("company A and B share an ID: %d", companyAID)
	}

	// --- Step 2: Create contacts in each company -------------------------------------

	contactA := &domain.Contact{
		Type: domain.ContactTypeCompany,
		Name: "Customer A",
		ICO:  "10000001",
	}
	if err := contactSvc.Create(ctx, companyAID, contactA); err != nil {
		t.Fatalf("creating contact A: %v", err)
	}

	contactB := &domain.Contact{
		Type: domain.ContactTypeCompany,
		Name: "Customer B",
		ICO:  "20000001",
	}
	if err := contactSvc.Create(ctx, companyBID, contactB); err != nil {
		t.Fatalf("creating contact B: %v", err)
	}

	// --- Step 3: Create invoice in company A using company-A contact (success) ------

	invoiceA := &domain.Invoice{
		Type:          domain.InvoiceTypeRegular,
		Status:        domain.InvoiceStatusDraft,
		IssueDate:     time.Now(),
		DueDate:       time.Now().AddDate(0, 0, 14),
		DeliveryDate:  time.Now(),
		CustomerID:    contactA.ID,
		CurrencyCode:  domain.CurrencyCZK,
		ExchangeRate:  100,
		PaymentMethod: "bank_transfer",
		InvoiceNumber: "FV2026-A001",
		Items: []domain.InvoiceItem{
			{Description: "leak-test", Quantity: 100, Unit: "ks", UnitPrice: 10000, VATRatePercent: 21},
		},
	}
	if err := invoiceSvc.Create(ctx, companyAID, invoiceA); err != nil {
		t.Fatalf("creating same-company invoice: %v", err)
	}

	// --- Step 4: Try to create invoice in company A using company-B contact (fail) --

	invoiceCross := &domain.Invoice{
		Type:          domain.InvoiceTypeRegular,
		Status:        domain.InvoiceStatusDraft,
		IssueDate:     time.Now(),
		DueDate:       time.Now().AddDate(0, 0, 14),
		DeliveryDate:  time.Now(),
		CustomerID:    contactB.ID, // belongs to company B
		CurrencyCode:  domain.CurrencyCZK,
		ExchangeRate:  100,
		PaymentMethod: "bank_transfer",
		InvoiceNumber: "FV2026-A002",
		Items: []domain.InvoiceItem{
			{Description: "cross", Quantity: 100, Unit: "ks", UnitPrice: 10000, VATRatePercent: 21},
		},
	}
	err = invoiceSvc.Create(ctx, companyAID, invoiceCross)
	if err == nil {
		t.Fatalf("expected cross-company invoice creation to fail, but it succeeded")
	}
	// InvoiceService validates the customer exists in the same company via
	// the contact repo's company-scoped GetByID; a mismatch surfaces as
	// ErrNotFound (or a wrapped version of it).
	t.Logf("cross-company invoice was rejected (good): %v", err)

	// --- Step 5: Both companies create FV2026-0001 sequence -------------------------

	seqA := &domain.InvoiceSequence{
		Prefix:        "FV",
		NextNumber:    1,
		Year:          2026,
		FormatPattern: "{prefix}{year}{number:04d}",
	}
	if err := sequenceSvc.Create(ctx, companyAID, seqA); err != nil {
		t.Fatalf("creating sequence in company A: %v", err)
	}

	seqB := &domain.InvoiceSequence{
		Prefix:        "FV",
		NextNumber:    1,
		Year:          2026,
		FormatPattern: "{prefix}{year}{number:04d}",
	}
	if err := sequenceSvc.Create(ctx, companyBID, seqB); err != nil {
		t.Fatalf("creating identical sequence in company B (UNIQUE partition failed): %v", err)
	}

	// --- Step 6: Soft-delete company A while it owns contacts → ErrInUse ------------

	err = companySvc.Delete(ctx, companyAID)
	if err == nil {
		t.Fatalf("expected ErrInUse deleting in-use company, got nil")
	}
	if !errors.Is(err, domain.ErrInUse) {
		t.Fatalf("expected ErrInUse, got %v", err)
	}

	// --- Step 7: Soft-delete the contact (parent of invoice) and the invoice --------

	if err := invoiceSvc.Delete(ctx, companyAID, invoiceA.ID); err != nil {
		t.Fatalf("deleting invoice: %v", err)
	}

	if err := contactSvc.Delete(ctx, companyAID, contactA.ID); err != nil {
		t.Fatalf("deleting contact: %v", err)
	}

	// Now company A is empty; delete it.
	if err := companySvc.Delete(ctx, companyAID); err != nil {
		t.Fatalf("deleting empty company A: %v", err)
	}

	// --- Step 8: companies.List returns only company B ------------------------------

	companies, err := companySvc.List(ctx)
	if err != nil {
		t.Fatalf("listing companies after delete: %v", err)
	}
	if len(companies) != 1 {
		t.Fatalf("expected 1 active company after delete, got %d", len(companies))
	}
	if companies[0].ID != companyBID {
		t.Fatalf("expected only company B in list, got id=%d (B=%d)", companies[0].ID, companyBID)
	}

	// --- Step 9: Try to delete the last company → ErrLastCompany --------------------

	err = companySvc.Delete(ctx, companyBID)
	if err == nil {
		t.Fatalf("expected ErrLastCompany deleting the only remaining company, got nil")
	}
	if !errors.Is(err, domain.ErrLastCompany) {
		t.Fatalf("expected ErrLastCompany, got %v", err)
	}
}
