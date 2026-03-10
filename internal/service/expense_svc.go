package service

import (
	"context"
	"errors"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// ExpenseService provides business logic for expense management.
type ExpenseService struct {
	repo repository.ExpenseRepo
}

// NewExpenseService creates a new ExpenseService.
func NewExpenseService(repo repository.ExpenseRepo) *ExpenseService {
	return &ExpenseService{repo: repo}
}

// Create validates and persists a new expense.
func (s *ExpenseService) Create(ctx context.Context, expense *domain.Expense) error {
	if expense.Description == "" {
		return errors.New("expense description is required")
	}
	if expense.Amount == 0 {
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

	// Calculate VAT amount from rate if not set.
	if expense.VATAmount == 0 && expense.VATRatePercent > 0 {
		expense.VATAmount = expense.Amount.Multiply(float64(expense.VATRatePercent) / (100.0 + float64(expense.VATRatePercent)))
	}

	return s.repo.Create(ctx, expense)
}

// Update validates and updates an existing expense.
func (s *ExpenseService) Update(ctx context.Context, expense *domain.Expense) error {
	if expense.ID == 0 {
		return errors.New("expense ID is required")
	}
	if expense.Description == "" {
		return errors.New("expense description is required")
	}
	if expense.Amount == 0 {
		return errors.New("expense amount is required")
	}
	if expense.BusinessPercent < 0 || expense.BusinessPercent > 100 {
		return errors.New("business share must be between 0 and 100")
	}

	// Recalculate VAT amount from rate if not set.
	if expense.VATAmount == 0 && expense.VATRatePercent > 0 {
		expense.VATAmount = expense.Amount.Multiply(float64(expense.VATRatePercent) / (100.0 + float64(expense.VATRatePercent)))
	}

	return s.repo.Update(ctx, expense)
}

// Delete removes an expense by ID (soft delete).
func (s *ExpenseService) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return errors.New("expense ID is required")
	}
	return s.repo.Delete(ctx, id)
}

// GetByID retrieves an expense by its ID.
func (s *ExpenseService) GetByID(ctx context.Context, id int64) (*domain.Expense, error) {
	if id == 0 {
		return nil, errors.New("expense ID is required")
	}
	return s.repo.GetByID(ctx, id)
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
	return s.repo.List(ctx, filter)
}
