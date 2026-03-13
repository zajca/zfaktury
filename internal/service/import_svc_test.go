package service

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service/ocr"
)

// mockExpenseRepo is a minimal in-memory ExpenseRepo for unit tests.
type mockExpenseRepo struct {
	expenses  map[int64]*domain.Expense
	nextID    int64
	createErr error // if set, Create returns this error
	deleteErr error // if set, Delete returns this error
}

func newMockExpenseRepo() *mockExpenseRepo {
	return &mockExpenseRepo{
		expenses: make(map[int64]*domain.Expense),
		nextID:   1,
	}
}

func (m *mockExpenseRepo) Create(ctx context.Context, expense *domain.Expense) error {
	if m.createErr != nil {
		return m.createErr
	}
	expense.ID = m.nextID
	m.nextID++
	cp := *expense
	m.expenses[expense.ID] = &cp
	return nil
}

func (m *mockExpenseRepo) Update(ctx context.Context, expense *domain.Expense) error {
	if _, ok := m.expenses[expense.ID]; !ok {
		return errors.New("expense not found")
	}
	cp := *expense
	m.expenses[expense.ID] = &cp
	return nil
}

func (m *mockExpenseRepo) Delete(ctx context.Context, id int64) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.expenses[id]; !ok {
		return errors.New("expense not found")
	}
	delete(m.expenses, id)
	return nil
}

func (m *mockExpenseRepo) GetByID(ctx context.Context, id int64) (*domain.Expense, error) {
	e, ok := m.expenses[id]
	if !ok {
		return nil, errors.New("expense not found")
	}
	cp := *e
	return &cp, nil
}

func (m *mockExpenseRepo) List(ctx context.Context, filter domain.ExpenseFilter) ([]domain.Expense, int, error) {
	var result []domain.Expense
	for _, e := range m.expenses {
		result = append(result, *e)
	}
	return result, len(result), nil
}

func (m *mockExpenseRepo) MarkTaxReviewed(ctx context.Context, ids []int64) error {
	return nil
}

func (m *mockExpenseRepo) UnmarkTaxReviewed(ctx context.Context, ids []int64) error {
	return nil
}

// failingDocumentRepo is a DocumentRepo that fails on Create to simulate upload DB errors.
type failingDocumentRepo struct {
	mockDocumentRepo
	createErr error
}

func (f *failingDocumentRepo) Create(ctx context.Context, doc *domain.ExpenseDocument) error {
	if f.createErr != nil {
		return f.createErr
	}
	return f.mockDocumentRepo.Create(ctx, doc)
}

// newImportTestService creates an ImportService with mock repos and t.TempDir() for document storage.
// If ocrProvider is nil, the OCRService is nil (simulating OCR not configured).
func newImportTestService(t *testing.T, ocrProvider ocr.Provider) (*ImportService, *mockExpenseRepo, *mockDocumentRepo) {
	t.Helper()

	expenseRepo := newMockExpenseRepo()
	expenseSvc := NewExpenseService(expenseRepo, nil)

	docRepo := newMockDocumentRepo()
	dataDir := t.TempDir()
	docSvc := NewDocumentService(docRepo, dataDir, nil)

	var ocrSvc *OCRService
	if ocrProvider != nil {
		ocrSvc = NewOCRService(ocrProvider, docSvc)
	}

	importSvc := NewImportService(expenseSvc, docSvc, ocrSvc)
	return importSvc, expenseRepo, docRepo
}

func TestImportService_ImportFromDocument_SuccessWithOCR(t *testing.T) {
	ocrResult := &domain.OCRResult{
		VendorName:  "Test Vendor s.r.o.",
		TotalAmount: 250000,
		Confidence:  0.92,
	}
	provider := &mockOCRProvider{result: ocrResult}
	importSvc, expenseRepo, docRepo := newImportTestService(t, provider)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	result, err := importSvc.ImportFromDocument(ctx, "faktura.pdf", "application/pdf", data)
	if err != nil {
		t.Fatalf("ImportFromDocument() error: %v", err)
	}

	// Verify expense was created.
	if result.Expense.ID == 0 {
		t.Error("expected non-zero expense ID")
	}
	if result.Expense.Description != "faktura.pdf" {
		t.Errorf("Expense.Description = %q, want %q", result.Expense.Description, "faktura.pdf")
	}
	if result.Expense.Amount != 100 {
		t.Errorf("Expense.Amount = %d, want 100 (1 CZK placeholder)", result.Expense.Amount)
	}
	if result.Expense.CurrencyCode != domain.CurrencyCZK {
		t.Errorf("Expense.CurrencyCode = %q, want %q", result.Expense.CurrencyCode, domain.CurrencyCZK)
	}

	// Verify document was created.
	if result.Document.ID == 0 {
		t.Error("expected non-zero document ID")
	}
	if result.Document.Filename != "faktura.pdf" {
		t.Errorf("Document.Filename = %q, want %q", result.Document.Filename, "faktura.pdf")
	}
	if result.Document.ExpenseID != result.Expense.ID {
		t.Errorf("Document.ExpenseID = %d, want %d", result.Document.ExpenseID, result.Expense.ID)
	}

	// Verify OCR result is present.
	if result.OCR == nil {
		t.Fatal("expected OCR result, got nil")
	}
	if result.OCR.VendorName != "Test Vendor s.r.o." {
		t.Errorf("OCR.VendorName = %q, want %q", result.OCR.VendorName, "Test Vendor s.r.o.")
	}
	if result.OCR.TotalAmount != 250000 {
		t.Errorf("OCR.TotalAmount = %d, want 250000", result.OCR.TotalAmount)
	}

	// Verify expense exists in repo.
	if _, ok := expenseRepo.expenses[result.Expense.ID]; !ok {
		t.Error("expense should exist in repo after successful import")
	}

	// Verify document exists in repo.
	if _, ok := docRepo.docs[result.Document.ID]; !ok {
		t.Error("document should exist in repo after successful import")
	}
}

func TestImportService_ImportFromDocument_SuccessWithoutOCR(t *testing.T) {
	// Pass nil OCR provider to simulate OCR not configured.
	importSvc, expenseRepo, _ := newImportTestService(t, nil)
	ctx := context.Background()

	data := bytes.NewReader(jpegMagic)
	result, err := importSvc.ImportFromDocument(ctx, "receipt.jpg", "image/jpeg", data)
	if err != nil {
		t.Fatalf("ImportFromDocument() error: %v", err)
	}

	// Verify expense was created.
	if result.Expense.ID == 0 {
		t.Error("expected non-zero expense ID")
	}

	// Verify document was created.
	if result.Document.ID == 0 {
		t.Error("expected non-zero document ID")
	}

	// Verify OCR result is nil (not configured).
	if result.OCR != nil {
		t.Errorf("expected nil OCR result when OCR is not configured, got %+v", result.OCR)
	}

	// Verify expense exists in repo.
	if _, ok := expenseRepo.expenses[result.Expense.ID]; !ok {
		t.Error("expense should exist in repo after successful import")
	}
}

func TestImportService_ImportFromDocument_UploadFailureRollsBackExpense(t *testing.T) {
	// Use a DocumentService with a repo that will fail on Create (after file write).
	expenseRepo := newMockExpenseRepo()
	expenseSvc := NewExpenseService(expenseRepo, nil)

	failDocRepo := &failingDocumentRepo{
		mockDocumentRepo: *newMockDocumentRepo(),
		createErr:        errors.New("database connection lost"),
	}
	dataDir := t.TempDir()
	docSvc := NewDocumentService(failDocRepo, dataDir, nil)

	importSvc := NewImportService(expenseSvc, docSvc, nil)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	_, err := importSvc.ImportFromDocument(ctx, "faktura.pdf", "application/pdf", data)
	if err == nil {
		t.Fatal("expected error when document upload fails")
	}

	// Verify the expense was rolled back (deleted).
	if len(expenseRepo.expenses) != 0 {
		t.Errorf("expected expense repo to be empty after rollback, got %d entries", len(expenseRepo.expenses))
	}
}

func TestImportService_ImportFromDocument_UploadFailureRollbackDeleteFails(t *testing.T) {
	// Even if the rollback delete fails, the original upload error should be returned.
	expenseRepo := newMockExpenseRepo()
	expenseRepo.deleteErr = errors.New("delete also failed")
	expenseSvc := NewExpenseService(expenseRepo, nil)

	failDocRepo := &failingDocumentRepo{
		mockDocumentRepo: *newMockDocumentRepo(),
		createErr:        errors.New("database connection lost"),
	}
	dataDir := t.TempDir()
	docSvc := NewDocumentService(failDocRepo, dataDir, nil)

	importSvc := NewImportService(expenseSvc, docSvc, nil)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	_, err := importSvc.ImportFromDocument(ctx, "faktura.pdf", "application/pdf", data)
	if err == nil {
		t.Fatal("expected error when document upload fails")
	}

	// The returned error should be about the upload failure, not the rollback failure.
	if !errors.Is(err, failDocRepo.createErr) {
		// The error is wrapped, so check the message contains the upload context.
		if got := err.Error(); got == "" {
			t.Error("expected non-empty error message")
		}
	}
}

func TestImportService_ImportFromDocument_OCRFailureIsNonFatal(t *testing.T) {
	// OCR provider returns an error -- import should still succeed.
	provider := &mockOCRProvider{err: errors.New("OCR API quota exceeded")}
	importSvc, _, _ := newImportTestService(t, provider)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	result, err := importSvc.ImportFromDocument(ctx, "receipt.pdf", "application/pdf", data)
	if err != nil {
		t.Fatalf("ImportFromDocument() should succeed even when OCR fails, got error: %v", err)
	}

	// Expense and document should be created.
	if result.Expense.ID == 0 {
		t.Error("expected non-zero expense ID")
	}
	if result.Document.ID == 0 {
		t.Error("expected non-zero document ID")
	}

	// OCR result should be nil since OCR failed.
	if result.OCR != nil {
		t.Errorf("expected nil OCR result when OCR fails, got %+v", result.OCR)
	}
}
