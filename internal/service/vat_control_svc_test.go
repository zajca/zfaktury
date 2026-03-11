package service

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// --- Mock implementations ---

type mockVATControlStatementRepo struct {
	statements map[int64]*domain.VATControlStatement
	lines      map[int64][]domain.VATControlStatementLine
	nextID     int64
	nextLineID int64
}

func newMockVATControlStatementRepo() *mockVATControlStatementRepo {
	return &mockVATControlStatementRepo{
		statements: make(map[int64]*domain.VATControlStatement),
		lines:      make(map[int64][]domain.VATControlStatementLine),
		nextID:     1,
		nextLineID: 1,
	}
}

func (m *mockVATControlStatementRepo) Create(_ context.Context, cs *domain.VATControlStatement) error {
	cs.ID = m.nextID
	m.nextID++
	cs.CreatedAt = time.Now()
	cs.UpdatedAt = time.Now()
	clone := *cs
	m.statements[cs.ID] = &clone
	return nil
}

func (m *mockVATControlStatementRepo) Update(_ context.Context, cs *domain.VATControlStatement) error {
	cs.UpdatedAt = time.Now()
	clone := *cs
	m.statements[cs.ID] = &clone
	return nil
}

func (m *mockVATControlStatementRepo) Delete(_ context.Context, id int64) error {
	if _, ok := m.statements[id]; !ok {
		return domain.ErrNotFound
	}
	delete(m.statements, id)
	delete(m.lines, id)
	return nil
}

func (m *mockVATControlStatementRepo) GetByID(_ context.Context, id int64) (*domain.VATControlStatement, error) {
	cs, ok := m.statements[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *cs
	return &clone, nil
}

func (m *mockVATControlStatementRepo) List(_ context.Context, year int) ([]domain.VATControlStatement, error) {
	var result []domain.VATControlStatement
	for _, cs := range m.statements {
		if cs.Period.Year == year {
			result = append(result, *cs)
		}
	}
	return result, nil
}

func (m *mockVATControlStatementRepo) GetByPeriod(_ context.Context, year, month int, filingType string) (*domain.VATControlStatement, error) {
	for _, cs := range m.statements {
		if cs.Period.Year == year && cs.Period.Month == month && cs.FilingType == filingType {
			clone := *cs
			return &clone, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockVATControlStatementRepo) CreateLines(_ context.Context, lines []domain.VATControlStatementLine) error {
	if len(lines) == 0 {
		return nil
	}
	csID := lines[0].ControlStatementID
	for i := range lines {
		lines[i].ID = m.nextLineID
		m.nextLineID++
	}
	m.lines[csID] = append(m.lines[csID], lines...)
	return nil
}

func (m *mockVATControlStatementRepo) DeleteLines(_ context.Context, controlStatementID int64) error {
	delete(m.lines, controlStatementID)
	return nil
}

func (m *mockVATControlStatementRepo) GetLines(_ context.Context, controlStatementID int64) ([]domain.VATControlStatementLine, error) {
	return m.lines[controlStatementID], nil
}

type vcMockInvoiceRepo struct {
	invoices []domain.Invoice
}

func (m *vcMockInvoiceRepo) Create(_ context.Context, _ *domain.Invoice) error { return nil }
func (m *vcMockInvoiceRepo) Update(_ context.Context, _ *domain.Invoice) error { return nil }
func (m *vcMockInvoiceRepo) Delete(_ context.Context, _ int64) error           { return nil }
func (m *vcMockInvoiceRepo) UpdateStatus(_ context.Context, _ int64, _ string) error {
	return nil
}
func (m *vcMockInvoiceRepo) GetNextNumber(_ context.Context, _ int64) (string, error) {
	return "", nil
}
func (m *vcMockInvoiceRepo) GetRelatedInvoices(_ context.Context, _ int64) ([]domain.Invoice, error) {
	return nil, nil
}

func (m *vcMockInvoiceRepo) GetByID(_ context.Context, id int64) (*domain.Invoice, error) {
	for _, inv := range m.invoices {
		if inv.ID == id {
			clone := inv
			return &clone, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *vcMockInvoiceRepo) List(_ context.Context, _ domain.InvoiceFilter) ([]domain.Invoice, int, error) {
	return m.invoices, len(m.invoices), nil
}

type vcMockExpenseRepo struct {
	expenses []domain.Expense
}

func (m *vcMockExpenseRepo) Create(_ context.Context, _ *domain.Expense) error { return nil }
func (m *vcMockExpenseRepo) Update(_ context.Context, _ *domain.Expense) error { return nil }
func (m *vcMockExpenseRepo) Delete(_ context.Context, _ int64) error           { return nil }
func (m *vcMockExpenseRepo) MarkTaxReviewed(_ context.Context, _ []int64) error {
	return nil
}
func (m *vcMockExpenseRepo) UnmarkTaxReviewed(_ context.Context, _ []int64) error {
	return nil
}

func (m *vcMockExpenseRepo) GetByID(_ context.Context, id int64) (*domain.Expense, error) {
	for _, e := range m.expenses {
		if e.ID == id {
			clone := e
			return &clone, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *vcMockExpenseRepo) List(_ context.Context, _ domain.ExpenseFilter) ([]domain.Expense, int, error) {
	return m.expenses, len(m.expenses), nil
}

type vcMockContactRepo struct {
	contacts map[int64]*domain.Contact
}

func (m *vcMockContactRepo) Create(_ context.Context, _ *domain.Contact) error { return nil }
func (m *vcMockContactRepo) Update(_ context.Context, _ *domain.Contact) error { return nil }
func (m *vcMockContactRepo) Delete(_ context.Context, _ int64) error           { return nil }
func (m *vcMockContactRepo) FindByICO(_ context.Context, _ string) (*domain.Contact, error) {
	return nil, domain.ErrNotFound
}
func (m *vcMockContactRepo) List(_ context.Context, _ domain.ContactFilter) ([]domain.Contact, int, error) {
	return nil, 0, nil
}

func (m *vcMockContactRepo) GetByID(_ context.Context, id int64) (*domain.Contact, error) {
	c, ok := m.contacts[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *c
	return &clone, nil
}

// --- Tests ---

func TestVATControlStatementService_Create(t *testing.T) {
	repo := newMockVATControlStatementRepo()
	svc := NewVATControlStatementService(repo, &vcMockInvoiceRepo{}, &vcMockExpenseRepo{}, &vcMockContactRepo{})

	cs := &domain.VATControlStatement{
		Period:     domain.TaxPeriod{Year: 2025, Month: 3},
		FilingType: domain.FilingTypeRegular,
	}

	err := svc.Create(context.Background(), cs)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}
	if cs.ID == 0 {
		t.Error("Create() should set ID")
	}
	if cs.Status != domain.FilingStatusDraft {
		t.Errorf("Create() should set status to draft, got %q", cs.Status)
	}
}

func TestVATControlStatementService_Create_Duplicate(t *testing.T) {
	repo := newMockVATControlStatementRepo()
	svc := NewVATControlStatementService(repo, &vcMockInvoiceRepo{}, &vcMockExpenseRepo{}, &vcMockContactRepo{})

	cs1 := &domain.VATControlStatement{
		Period:     domain.TaxPeriod{Year: 2025, Month: 3},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(context.Background(), cs1); err != nil {
		t.Fatalf("first Create() returned error: %v", err)
	}

	cs2 := &domain.VATControlStatement{
		Period:     domain.TaxPeriod{Year: 2025, Month: 3},
		FilingType: domain.FilingTypeRegular,
	}
	err := svc.Create(context.Background(), cs2)
	if err == nil {
		t.Error("second Create() should return error for duplicate")
	}
}

func TestVATControlStatementService_Create_InvalidMonth(t *testing.T) {
	repo := newMockVATControlStatementRepo()
	svc := NewVATControlStatementService(repo, &vcMockInvoiceRepo{}, &vcMockExpenseRepo{}, &vcMockContactRepo{})

	cs := &domain.VATControlStatement{
		Period: domain.TaxPeriod{Year: 2025, Month: 13},
	}
	err := svc.Create(context.Background(), cs)
	if err == nil {
		t.Error("Create() should return error for invalid month")
	}
}

func TestVATControlStatementService_Delete_Filed(t *testing.T) {
	repo := newMockVATControlStatementRepo()
	svc := NewVATControlStatementService(repo, &vcMockInvoiceRepo{}, &vcMockExpenseRepo{}, &vcMockContactRepo{})

	cs := &domain.VATControlStatement{
		Period:     domain.TaxPeriod{Year: 2025, Month: 3},
		FilingType: domain.FilingTypeRegular,
	}
	_ = svc.Create(context.Background(), cs)

	// Mark as filed directly in repo.
	stored := repo.statements[cs.ID]
	stored.Status = domain.FilingStatusFiled

	err := svc.Delete(context.Background(), cs.ID)
	if err == nil {
		t.Error("Delete() should return error for filed statement")
	}
}

func TestVATControlStatementService_Recalculate(t *testing.T) {
	csRepo := newMockVATControlStatementRepo()
	contactRepo := &vcMockContactRepo{
		contacts: map[int64]*domain.Contact{
			100: {ID: 100, Name: "Big Customer", DIC: "CZ12345678"},
			200: {ID: 200, Name: "Small Customer", DIC: "CZ87654321"},
			300: {ID: 300, Name: "Foreign Customer", DIC: "DE123456789"},
			400: {ID: 400, Name: "Vendor CZ", DIC: "CZ55667788"},
		},
	}

	// Create invoices: one big (>10000 CZK), one small, one foreign.
	invoiceRepo := &vcMockInvoiceRepo{
		invoices: []domain.Invoice{
			{
				ID:            1,
				CustomerID:    100,
				InvoiceNumber: "FV20250001",
				Status:        domain.InvoiceStatusSent,
				DeliveryDate:  time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
				TotalAmount:   domain.NewAmount(15000, 0), // 15000 CZK > threshold
				Items: []domain.InvoiceItem{
					{
						Quantity:       domain.NewAmount(1, 0),      // 100 = 1.00
						UnitPrice:      domain.NewAmount(12396, 69), // 12396.69 CZK
						VATRatePercent: 21,
						VATAmount:      domain.NewAmount(2603, 31),
					},
				},
			},
			{
				ID:            2,
				CustomerID:    200,
				InvoiceNumber: "FV20250002",
				Status:        domain.InvoiceStatusPaid,
				DeliveryDate:  time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC),
				TotalAmount:   domain.NewAmount(5000, 0), // 5000 CZK < threshold
				Items: []domain.InvoiceItem{
					{
						Quantity:       domain.NewAmount(1, 0),
						UnitPrice:      domain.NewAmount(4132, 23),
						VATRatePercent: 21,
						VATAmount:      domain.NewAmount(867, 77),
					},
				},
			},
			{
				ID:            3,
				CustomerID:    300,
				InvoiceNumber: "FV20250003",
				Status:        domain.InvoiceStatusSent,
				DeliveryDate:  time.Date(2025, 3, 25, 0, 0, 0, 0, time.UTC),
				TotalAmount:   domain.NewAmount(8000, 0),
				Items: []domain.InvoiceItem{
					{
						Quantity:       domain.NewAmount(1, 0),
						UnitPrice:      domain.NewAmount(8000, 0),
						VATRatePercent: 21,
						VATAmount:      domain.NewAmount(1680, 0),
					},
				},
			},
		},
	}

	// Create expenses: one big, one small.
	vendorID := int64(400)
	expenseRepo := &vcMockExpenseRepo{
		expenses: []domain.Expense{
			{
				ID:              10,
				VendorID:        &vendorID,
				ExpenseNumber:   "VF2025001",
				IssueDate:       time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC),
				Amount:          domain.NewAmount(15000, 0),
				VATRatePercent:  21,
				VATAmount:       domain.NewAmount(2603, 31),
				IsTaxDeductible: true,
			},
			{
				ID:              11,
				VendorID:        &vendorID,
				ExpenseNumber:   "VF2025002",
				IssueDate:       time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC),
				Amount:          domain.NewAmount(3000, 0),
				VATRatePercent:  21,
				VATAmount:       domain.NewAmount(520, 66),
				IsTaxDeductible: true,
			},
		},
	}

	svc := NewVATControlStatementService(csRepo, invoiceRepo, expenseRepo, contactRepo)

	// Create the control statement.
	cs := &domain.VATControlStatement{
		Period:     domain.TaxPeriod{Year: 2025, Month: 3},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(context.Background(), cs); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	// Recalculate.
	if err := svc.Recalculate(context.Background(), cs.ID); err != nil {
		t.Fatalf("Recalculate() returned error: %v", err)
	}

	// Check lines.
	lines, err := svc.GetLines(context.Background(), cs.ID)
	if err != nil {
		t.Fatalf("GetLines() returned error: %v", err)
	}

	// Expect: 1 A4 line (big invoice), 1 A5 line (small invoice), 1 B2 line (big expense), 1 B3 line (small expense).
	// Foreign invoice (ID=3) should be excluded.
	sectionCounts := make(map[string]int)
	for _, line := range lines {
		sectionCounts[line.Section]++
	}

	if sectionCounts["A4"] != 1 {
		t.Errorf("expected 1 A4 line, got %d", sectionCounts["A4"])
	}
	if sectionCounts["A5"] != 1 {
		t.Errorf("expected 1 A5 line, got %d", sectionCounts["A5"])
	}
	if sectionCounts["B2"] != 1 {
		t.Errorf("expected 1 B2 line, got %d", sectionCounts["B2"])
	}
	if sectionCounts["B3"] != 1 {
		t.Errorf("expected 1 B3 line, got %d", sectionCounts["B3"])
	}

	// Verify A4 line has correct partner DIC and document number.
	for _, line := range lines {
		if line.Section == "A4" {
			if line.PartnerDIC != "CZ12345678" {
				t.Errorf("A4 line partner DIC = %q, want CZ12345678", line.PartnerDIC)
			}
			if line.DocumentNumber != "FV20250001" {
				t.Errorf("A4 line document number = %q, want FV20250001", line.DocumentNumber)
			}
			if line.InvoiceID == nil || *line.InvoiceID != 1 {
				t.Error("A4 line should reference invoice ID 1")
			}
		}
		if line.Section == "B2" {
			if line.PartnerDIC != "CZ55667788" {
				t.Errorf("B2 line partner DIC = %q, want CZ55667788", line.PartnerDIC)
			}
			if line.ExpenseID == nil || *line.ExpenseID != 10 {
				t.Error("B2 line should reference expense ID 10")
			}
		}
	}

	// Verify status updated to ready.
	updated, _ := svc.GetByID(context.Background(), cs.ID)
	if updated.Status != domain.FilingStatusReady {
		t.Errorf("status after recalculate = %q, want %q", updated.Status, domain.FilingStatusReady)
	}
}

func TestVATControlStatementService_MarkFiled(t *testing.T) {
	repo := newMockVATControlStatementRepo()
	svc := NewVATControlStatementService(repo, &vcMockInvoiceRepo{}, &vcMockExpenseRepo{}, &vcMockContactRepo{})

	cs := &domain.VATControlStatement{
		Period:     domain.TaxPeriod{Year: 2025, Month: 3},
		FilingType: domain.FilingTypeRegular,
	}
	_ = svc.Create(context.Background(), cs)

	err := svc.MarkFiled(context.Background(), cs.ID)
	if err != nil {
		t.Fatalf("MarkFiled() returned error: %v", err)
	}

	updated, _ := svc.GetByID(context.Background(), cs.ID)
	if updated.Status != domain.FilingStatusFiled {
		t.Errorf("status = %q, want %q", updated.Status, domain.FilingStatusFiled)
	}
	if updated.FiledAt == nil {
		t.Error("FiledAt should be set after MarkFiled")
	}
}
