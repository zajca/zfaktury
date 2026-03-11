package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// ImportService orchestrates expense creation from an uploaded document.
type ImportService struct {
	expenses  *ExpenseService
	documents *DocumentService
	ocr       *OCRService // may be nil
}

// NewImportService creates a new ImportService.
func NewImportService(expenses *ExpenseService, documents *DocumentService, ocr *OCRService) *ImportService {
	return &ImportService{
		expenses:  expenses,
		documents: documents,
		ocr:       ocr,
	}
}

// ImportFromDocument creates a skeleton expense, uploads the document, and
// optionally runs OCR. Returns the combined result.
func (s *ImportService) ImportFromDocument(ctx context.Context, filename, contentType string, data io.Reader) (*domain.ImportResult, error) {
	// Create a skeleton expense with minimal placeholder data.
	expense := &domain.Expense{
		Description:     filename,
		Amount:          100, // 1 CZK minimum (in halere)
		IssueDate:       time.Now().Truncate(24 * time.Hour),
		CurrencyCode:    domain.CurrencyCZK,
		BusinessPercent: 100,
		PaymentMethod:   "bank_transfer",
	}

	if err := s.expenses.Create(ctx, expense); err != nil {
		return nil, fmt.Errorf("creating skeleton expense: %w", err)
	}

	// Upload the document linked to the new expense.
	doc, err := s.documents.Upload(ctx, expense.ID, filename, contentType, data)
	if err != nil {
		// Rollback: delete the skeleton expense.
		if delErr := s.expenses.Delete(ctx, expense.ID); delErr != nil {
			slog.Error("failed to rollback skeleton expense after upload failure", "error", delErr, "expense_id", expense.ID)
		}
		return nil, fmt.Errorf("uploading document: %w", err)
	}

	result := &domain.ImportResult{
		Expense:  *expense,
		Document: *doc,
	}

	// Run OCR if configured -- non-fatal on failure.
	if s.ocr != nil {
		ocrResult, ocrErr := s.ocr.ProcessDocument(ctx, doc.ID)
		if ocrErr != nil {
			slog.Warn("OCR processing failed during import, continuing without OCR", "error", ocrErr, "document_id", doc.ID)
		} else {
			result.OCR = ocrResult
		}
	}

	return result, nil
}
