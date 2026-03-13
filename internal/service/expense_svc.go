package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// ExpenseService provides business logic for expense management.
type ExpenseService struct {
	repo  repository.ExpenseRepo
	audit *AuditService
}

// NewExpenseService creates a new ExpenseService.
func NewExpenseService(repo repository.ExpenseRepo, audit *AuditService) *ExpenseService {
	return &ExpenseService{repo: repo, audit: audit}
}

// Create validates and persists a new expense.
func (s *ExpenseService) Create(ctx context.Context, expense *domain.Expense) error {
	if expense.Description == "" {
		return errors.New("expense description is required")
	}
	if expense.Amount == 0 && len(expense.Items) == 0 {
		return errors.New("expense amount is required")
	}
	if expense.IssueDate.IsZero() {
		return errors.New("expense issue date is required")
	}
	if expense.CurrencyCode == "" {
		expense.CurrencyCode = domain.CurrencyCZK
	}
	if expense.BusinessPercent == 0 {
		expense.BusinessPercent = 100
	}
	if expense.BusinessPercent < 0 || expense.BusinessPercent > 100 {
		return errors.New("business share must be between 0 and 100")
	}

	// When items are present, recalculate totals from them.
	if len(expense.Items) > 0 {
		expense.CalculateTotals()
	} else if expense.VATAmount == 0 && expense.VATRatePercent > 0 {
		// Calculate VAT amount from rate if not set (flat-amount path).
		expense.VATAmount = expense.Amount.Multiply(float64(expense.VATRatePercent) / (100.0 + float64(expense.VATRatePercent)))
	}

	if err := s.repo.Create(ctx, expense); err != nil {
		return fmt.Errorf("creating expense: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "expense", expense.ID, "create", nil, expense)
	}
	return nil
}

// Update validates and updates an existing expense.
func (s *ExpenseService) Update(ctx context.Context, expense *domain.Expense) error {
	if expense.ID == 0 {
		return errors.New("expense ID is required")
	}
	if expense.Description == "" {
		return errors.New("expense description is required")
	}
	if expense.Amount == 0 && len(expense.Items) == 0 {
		return errors.New("expense amount is required")
	}
	if expense.BusinessPercent < 0 || expense.BusinessPercent > 100 {
		return errors.New("business share must be between 0 and 100")
	}

	// When items are present, recalculate totals from them.
	if len(expense.Items) > 0 {
		expense.CalculateTotals()
	} else if expense.VATAmount == 0 && expense.VATRatePercent > 0 {
		// Recalculate VAT amount from rate if not set (flat-amount path).
		expense.VATAmount = expense.Amount.Multiply(float64(expense.VATRatePercent) / (100.0 + float64(expense.VATRatePercent)))
	}

	existing, err := s.repo.GetByID(ctx, expense.ID)
	if err != nil {
		return fmt.Errorf("fetching expense for audit: %w", err)
	}
	if err := s.repo.Update(ctx, expense); err != nil {
		return fmt.Errorf("updating expense: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "expense", expense.ID, "update", existing, expense)
	}
	return nil
}

// Delete removes an expense by ID (soft delete).
func (s *ExpenseService) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return errors.New("expense ID is required")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting expense: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "expense", id, "delete", nil, nil)
	}
	return nil
}

// GetByID retrieves an expense by its ID.
func (s *ExpenseService) GetByID(ctx context.Context, id int64) (*domain.Expense, error) {
	if id == 0 {
		return nil, errors.New("expense ID is required")
	}
	exp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching expense: %w", err)
	}
	return exp, nil
}

// List retrieves expenses matching the given filter.
// Returns the expenses, total count, and any error.
func (s *ExpenseService) List(ctx context.Context, filter domain.ExpenseFilter) ([]domain.Expense, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	expenses, count, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("listing expenses: %w", err)
	}
	return expenses, count, nil
}

const maxBulkIDs = 500

// MarkTaxReviewed marks the given expense IDs as tax-reviewed.
func (s *ExpenseService) MarkTaxReviewed(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("no expense IDs provided")
	}
	if len(ids) > maxBulkIDs {
		return errors.New("too many IDs, maximum is 500")
	}
	if err := s.repo.MarkTaxReviewed(ctx, dedupIDs(ids)); err != nil {
		return fmt.Errorf("marking expenses as tax reviewed: %w", err)
	}
	return nil
}

// UnmarkTaxReviewed removes the tax review mark from the given expense IDs.
func (s *ExpenseService) UnmarkTaxReviewed(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New("no expense IDs provided")
	}
	if len(ids) > maxBulkIDs {
		return errors.New("too many IDs, maximum is 500")
	}
	if err := s.repo.UnmarkTaxReviewed(ctx, dedupIDs(ids)); err != nil {
		return fmt.Errorf("unmarking expenses tax review: %w", err)
	}
	return nil
}

// dedupIDs removes duplicate int64 values preserving order.
func dedupIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}
