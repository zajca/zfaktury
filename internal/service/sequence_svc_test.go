package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func newSequenceTestStack(t *testing.T) (*SequenceService, *repository.SequenceRepository) {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := repository.NewSequenceRepository(db)
	svc := NewSequenceService(repo, nil)
	return svc, repo
}

func TestSequenceService_Create_Valid(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{
		Prefix: "FV",
		Year:   2026,
	}
	if err := svc.Create(ctx, seq); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if seq.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if seq.NextNumber != 1 {
		t.Errorf("NextNumber = %d, want 1 (default)", seq.NextNumber)
	}
	if seq.FormatPattern == "" {
		t.Error("expected FormatPattern to be set by default")
	}
}

func TestSequenceService_Create_MissingPrefix(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{Year: 2026}
	err := svc.Create(ctx, seq)
	if err == nil {
		t.Error("expected error for missing prefix")
	}
}

func TestSequenceService_Create_MissingYear(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{Prefix: "FV"}
	err := svc.Create(ctx, seq)
	if err == nil {
		t.Error("expected error for missing year")
	}
}

func TestSequenceService_Create_DuplicatePrefixYear(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq1 := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	if err := svc.Create(ctx, seq1); err != nil {
		t.Fatalf("Create() first error: %v", err)
	}

	seq2 := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	err := svc.Create(ctx, seq2)
	if err == nil {
		t.Error("expected error for duplicate prefix+year")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists', got: %v", err)
	}
}

func TestSequenceService_Update_Valid(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	if err := svc.Create(ctx, seq); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	seq.NextNumber = 5
	if err := svc.Update(ctx, seq); err != nil {
		t.Fatalf("Update() error: %v", err)
	}
}

func TestSequenceService_Update_PreventLoweringNextNumber(t *testing.T) {
	svc, repo := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{Prefix: "FV", Year: 2026, NextNumber: 5}
	if err := svc.Create(ctx, seq); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Simulate that numbers up to 4 have been used (next_number is 5, so max used = 4).
	// The MaxUsedNumber returns next_number - 1 = 4.
	// But we created with NextNumber=5, so after Create the DB value is 5.
	// Let's read back from DB to get current state.
	got, err := repo.GetByID(ctx, seq.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	// Try to set next_number to 3, which is below max used (4).
	got.NextNumber = 3
	err = svc.Update(ctx, got)
	if err == nil {
		t.Error("expected error when lowering next_number below used numbers")
	}
	if !strings.Contains(err.Error(), "already been assigned") {
		t.Errorf("error should mention 'already been assigned', got: %v", err)
	}
}

func TestSequenceService_Delete_NoReferences(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	if err := svc.Create(ctx, seq); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.Delete(ctx, seq.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// Verify deleted.
	_, err := svc.GetByID(ctx, seq.ID)
	if err == nil {
		t.Error("expected error for soft-deleted sequence")
	}
}

func TestSequenceService_Create_AfterDelete(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	first := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	if err := svc.Create(ctx, first); err != nil {
		t.Fatalf("Create() first error: %v", err)
	}
	if err := svc.Delete(ctx, first.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	recreated := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	if err := svc.Create(ctx, recreated); err != nil {
		t.Fatalf("Create() after delete error: %v", err)
	}

	duplicate := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	if err := svc.Create(ctx, duplicate); !errors.Is(err, domain.ErrDuplicateNumber) {
		t.Errorf("Create() duplicate error = %v, want ErrDuplicateNumber", err)
	}
}

func TestSequenceService_Create_AfterDelete_SkipsNumbersOfDeletedInvoices(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDB(t)
	sequenceSvc := NewSequenceService(repository.NewSequenceRepository(db), nil)
	contactSvc := NewContactService(repository.NewContactRepository(db), nil, nil)
	invoiceSvc := NewInvoiceService(repository.NewInvoiceRepository(db), contactSvc, sequenceSvc, nil)

	customer := &domain.Contact{Name: "Test Customer", Type: domain.ContactTypeCompany}
	if err := contactSvc.Create(ctx, customer); err != nil {
		t.Fatalf("creating customer: %v", err)
	}

	newInvoice := func(sequenceID int64) *domain.Invoice {
		return &domain.Invoice{
			CustomerID:    customer.ID,
			SequenceID:    sequenceID,
			Type:          domain.InvoiceTypeRegular,
			IssueDate:     time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			DueDate:       time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
			DeliveryDate:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			CurrencyCode:  domain.CurrencyCZK,
			PaymentMethod: "bank_transfer",
			Items: []domain.InvoiceItem{
				{Description: "Test service", Quantity: 100, Unit: "hod", UnitPrice: 100000, VATRatePercent: 21},
			},
		}
	}

	first := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	if err := sequenceSvc.Create(ctx, first); err != nil {
		t.Fatalf("creating sequence: %v", err)
	}
	oldInvoice := newInvoice(first.ID)
	if err := invoiceSvc.Create(ctx, oldInvoice); err != nil {
		t.Fatalf("creating first invoice: %v", err)
	}
	if err := invoiceSvc.Delete(ctx, oldInvoice.ID); err != nil {
		t.Fatalf("deleting invoice: %v", err)
	}
	if err := sequenceSvc.Delete(ctx, first.ID); err != nil {
		t.Fatalf("deleting sequence: %v", err)
	}

	recreated := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	if err := sequenceSvc.Create(ctx, recreated); err != nil {
		t.Fatalf("recreating sequence: %v", err)
	}
	if recreated.NextNumber != 2 {
		t.Errorf("recreated NextNumber = %d, want 2", recreated.NextNumber)
	}

	next := newInvoice(recreated.ID)
	if err := invoiceSvc.Create(ctx, next); err != nil {
		t.Fatalf("creating invoice in recreated sequence: %v", err)
	}
	if next.InvoiceNumber == oldInvoice.InvoiceNumber {
		t.Errorf("invoice number %q reused", next.InvoiceNumber)
	}
	if want := "FV20260002"; next.InvoiceNumber != want {
		t.Errorf("invoice number = %q, want %q", next.InvoiceNumber, want)
	}

	// A requested number above the deleted sequences' counter is kept.
	if err := invoiceSvc.Delete(ctx, next.ID); err != nil {
		t.Fatalf("deleting second invoice: %v", err)
	}
	if err := sequenceSvc.Delete(ctx, recreated.ID); err != nil {
		t.Fatalf("deleting recreated sequence: %v", err)
	}
	high := &domain.InvoiceSequence{Prefix: "FV", Year: 2026, NextNumber: 50}
	if err := sequenceSvc.Create(ctx, high); err != nil {
		t.Fatalf("creating sequence with high start: %v", err)
	}
	if high.NextNumber != 50 {
		t.Errorf("NextNumber = %d, want 50", high.NextNumber)
	}
}

// deletedInvoiceStack creates FV/2026 with one invoice, then deletes the invoice
// and the sequence, so a soft-deleted invoice still owns number FV20260001.
type deletedInvoiceStack struct {
	sequenceSvc *SequenceService
	invoiceSvc  *InvoiceService
	customerID  int64
	oldNumber   string
}

func newDeletedInvoiceStack(t *testing.T) *deletedInvoiceStack {
	t.Helper()
	ctx := context.Background()
	db := testutil.NewTestDB(t)
	sequenceSvc := NewSequenceService(repository.NewSequenceRepository(db), nil)
	contactSvc := NewContactService(repository.NewContactRepository(db), nil, nil)
	st := &deletedInvoiceStack{
		sequenceSvc: sequenceSvc,
		invoiceSvc:  NewInvoiceService(repository.NewInvoiceRepository(db), contactSvc, sequenceSvc, nil),
	}

	customer := &domain.Contact{Name: "Test Customer", Type: domain.ContactTypeCompany}
	if err := contactSvc.Create(ctx, customer); err != nil {
		t.Fatalf("creating customer: %v", err)
	}
	st.customerID = customer.ID

	first := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	if err := sequenceSvc.Create(ctx, first); err != nil {
		t.Fatalf("creating sequence: %v", err)
	}
	old := st.newInvoice(first.ID)
	if err := st.invoiceSvc.Create(ctx, old); err != nil {
		t.Fatalf("creating first invoice: %v", err)
	}
	st.oldNumber = old.InvoiceNumber
	if err := st.invoiceSvc.Delete(ctx, old.ID); err != nil {
		t.Fatalf("deleting invoice: %v", err)
	}
	if err := sequenceSvc.Delete(ctx, first.ID); err != nil {
		t.Fatalf("deleting sequence: %v", err)
	}
	return st
}

func (st *deletedInvoiceStack) newInvoice(sequenceID int64) *domain.Invoice {
	return &domain.Invoice{
		CustomerID:    st.customerID,
		SequenceID:    sequenceID,
		Type:          domain.InvoiceTypeRegular,
		IssueDate:     time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		DeliveryDate:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		CurrencyCode:  domain.CurrencyCZK,
		PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "Test service", Quantity: 100, Unit: "hod", UnitPrice: 100000, VATRatePercent: 21},
		},
	}
}

func TestInvoiceService_Create_WithoutSequence_SkipsNumbersOfDeletedInvoices(t *testing.T) {
	ctx := context.Background()
	st := newDeletedInvoiceStack(t)

	inv := st.newInvoice(0)
	if err := st.invoiceSvc.Create(ctx, inv); err != nil {
		t.Fatalf("creating invoice without sequence: %v", err)
	}
	if inv.InvoiceNumber == st.oldNumber {
		t.Errorf("invoice number %q reused", inv.InvoiceNumber)
	}
	if want := "FV20260002"; inv.InvoiceNumber != want {
		t.Errorf("invoice number = %q, want %q", inv.InvoiceNumber, want)
	}
}

func TestSequenceService_Update_ToDeletedPrefixYear_SkipsNumbersOfDeletedInvoices(t *testing.T) {
	ctx := context.Background()
	st := newDeletedInvoiceStack(t)

	other := &domain.InvoiceSequence{Prefix: "XX", Year: 2026}
	if err := st.sequenceSvc.Create(ctx, other); err != nil {
		t.Fatalf("creating other sequence: %v", err)
	}
	other.Prefix = "FV"
	if err := st.sequenceSvc.Update(ctx, other); err != nil {
		t.Fatalf("moving sequence onto deleted prefix/year: %v", err)
	}
	if other.NextNumber != 2 {
		t.Errorf("NextNumber = %d, want 2", other.NextNumber)
	}

	inv := st.newInvoice(other.ID)
	if err := st.invoiceSvc.Create(ctx, inv); err != nil {
		t.Fatalf("creating invoice in moved sequence: %v", err)
	}
	if inv.InvoiceNumber == st.oldNumber {
		t.Errorf("invoice number %q reused", inv.InvoiceNumber)
	}
	if want := "FV20260002"; inv.InvoiceNumber != want {
		t.Errorf("invoice number = %q, want %q", inv.InvoiceNumber, want)
	}
}

func TestSequenceService_Delete_ZeroID(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestSequenceService_Update_ZeroID(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{ID: 0, Prefix: "FV", Year: 2026, NextNumber: 1}
	if err := svc.Update(ctx, seq); err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestSequenceService_Update_MissingPrefix(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{ID: 1, Prefix: "", Year: 2026, NextNumber: 1}
	if err := svc.Update(ctx, seq); err == nil {
		t.Error("expected error for empty prefix")
	}
}

func TestSequenceService_Update_MissingYear(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{ID: 1, Prefix: "FV", Year: 0, NextNumber: 1}
	if err := svc.Update(ctx, seq); err == nil {
		t.Error("expected error for zero year")
	}
}

func TestSequenceService_Update_ZeroNextNumber(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{ID: 1, Prefix: "FV", Year: 2026, NextNumber: 0}
	if err := svc.Update(ctx, seq); err == nil {
		t.Error("expected error for zero next_number")
	}
}

func TestSequenceService_GetByID_Success(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	seq := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	if err := svc.Create(ctx, seq); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := svc.GetByID(ctx, seq.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Prefix != "FV" {
		t.Errorf("Prefix = %q, want FV", got.Prefix)
	}
}

func TestSequenceService_GetByID_ZeroID(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	if _, err := svc.GetByID(ctx, 0); err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestSequenceService_GetOrCreateForYear_EmptyPrefix(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	if _, err := svc.GetOrCreateForYear(ctx, "", 2026); err == nil {
		t.Error("expected error for empty prefix")
	}
}

func TestSequenceService_GetOrCreateForYear_ZeroYear(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	if _, err := svc.GetOrCreateForYear(ctx, "FV", 0); err == nil {
		t.Error("expected error for zero year")
	}
}

func TestSequenceService_GetOrCreateForYear_ExistingSequence(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	// Create a sequence first.
	seq := &domain.InvoiceSequence{Prefix: "FV", Year: 2026}
	if err := svc.Create(ctx, seq); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// GetOrCreate should return the existing one.
	got, err := svc.GetOrCreateForYear(ctx, "FV", 2026)
	if err != nil {
		t.Fatalf("GetOrCreateForYear() error: %v", err)
	}
	if got.ID != seq.ID {
		t.Errorf("ID = %d, want %d (should return existing)", got.ID, seq.ID)
	}
}

func TestSequenceService_GetOrCreateForYear_NewSequence(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	got, err := svc.GetOrCreateForYear(ctx, "ZF", 2027)
	if err != nil {
		t.Fatalf("GetOrCreateForYear() error: %v", err)
	}
	if got.ID == 0 {
		t.Error("expected non-zero ID for newly created sequence")
	}
	if got.Prefix != "ZF" {
		t.Errorf("Prefix = %q, want %q", got.Prefix, "ZF")
	}
	if got.Year != 2027 {
		t.Errorf("Year = %d, want %d", got.Year, 2027)
	}
	if got.NextNumber != 1 {
		t.Errorf("NextNumber = %d, want 1", got.NextNumber)
	}
}

func TestSequenceService_List(t *testing.T) {
	svc, _ := newSequenceTestStack(t)
	ctx := context.Background()

	svc.Create(ctx, &domain.InvoiceSequence{Prefix: "FV", Year: 2025})
	svc.Create(ctx, &domain.InvoiceSequence{Prefix: "FV", Year: 2026})

	sequences, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(sequences) != 2 {
		t.Errorf("len(sequences) = %d, want 2", len(sequences))
	}
}

func TestFormatPreview(t *testing.T) {
	tests := []struct {
		name   string
		seq    *domain.InvoiceSequence
		expect string
	}{
		{
			name:   "standard format",
			seq:    &domain.InvoiceSequence{Prefix: "FV", Year: 2026, NextNumber: 1},
			expect: "FV20260001",
		},
		{
			name:   "higher number",
			seq:    &domain.InvoiceSequence{Prefix: "ZF", Year: 2025, NextNumber: 42},
			expect: "ZF20250042",
		},
		{
			name:   "credit note prefix",
			seq:    &domain.InvoiceSequence{Prefix: "DN", Year: 2026, NextNumber: 100},
			expect: "DN20260100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPreview(tt.seq)
			if got != tt.expect {
				t.Errorf("FormatPreview() = %q, want %q", got, tt.expect)
			}
		})
	}
}
