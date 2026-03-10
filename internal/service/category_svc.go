package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// keyPattern validates that a category key is lowercase alphanumeric with underscores.
var keyPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

// hexColorPattern validates CSS hex color values (#RGB or #RRGGBB).
var hexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{3}([0-9a-fA-F]{3})?$`)

// CategoryService provides business logic for expense category management.
type CategoryService struct {
	repo repository.CategoryRepo
}

// NewCategoryService creates a new CategoryService.
func NewCategoryService(repo repository.CategoryRepo) *CategoryService {
	return &CategoryService{repo: repo}
}

// Create validates and persists a new expense category.
func (s *CategoryService) Create(ctx context.Context, cat *domain.ExpenseCategory) error {
	if err := s.validateCategory(cat); err != nil {
		return err
	}

	// Check for duplicate key.
	existing, err := s.repo.GetByKey(ctx, cat.Key)
	if err == nil && existing != nil {
		return errors.New("category with this key already exists")
	}

	if cat.Color == "" {
		cat.Color = "#6B7280"
	}

	if err := s.repo.Create(ctx, cat); err != nil {
		return fmt.Errorf("creating category: %w", err)
	}
	return nil
}

// Update validates and updates an existing expense category.
func (s *CategoryService) Update(ctx context.Context, cat *domain.ExpenseCategory) error {
	if cat.ID == 0 {
		return errors.New("category ID is required")
	}

	if err := s.validateCategory(cat); err != nil {
		return err
	}

	// Check for duplicate key (excluding self).
	existing, err := s.repo.GetByKey(ctx, cat.Key)
	if err == nil && existing != nil && existing.ID != cat.ID {
		return errors.New("category with this key already exists")
	}

	if err := s.repo.Update(ctx, cat); err != nil {
		return fmt.Errorf("updating category: %w", err)
	}
	return nil
}

// Delete removes an expense category by ID (soft delete).
// Default categories (is_default=1) cannot be deleted.
func (s *CategoryService) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return errors.New("category ID is required")
	}

	cat, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching category for delete: %w", err)
	}

	if cat.IsDefault {
		return errors.New("default categories cannot be deleted")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting category: %w", err)
	}
	return nil
}

// GetByID retrieves an expense category by its ID.
func (s *CategoryService) GetByID(ctx context.Context, id int64) (*domain.ExpenseCategory, error) {
	if id == 0 {
		return nil, errors.New("category ID is required")
	}
	cat, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching category: %w", err)
	}
	return cat, nil
}

// List retrieves all expense categories.
func (s *CategoryService) List(ctx context.Context) ([]domain.ExpenseCategory, error) {
	cats, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing categories: %w", err)
	}
	return cats, nil
}

// validateCategory performs common validation for create and update operations.
func (s *CategoryService) validateCategory(cat *domain.ExpenseCategory) error {
	cat.Key = strings.TrimSpace(cat.Key)
	cat.LabelCS = strings.TrimSpace(cat.LabelCS)
	cat.LabelEN = strings.TrimSpace(cat.LabelEN)

	if cat.Key == "" {
		return errors.New("category key is required")
	}
	if !keyPattern.MatchString(cat.Key) {
		return errors.New("category key must be lowercase alphanumeric with underscores only")
	}
	if cat.LabelCS == "" {
		return errors.New("category Czech label is required")
	}
	if cat.LabelEN == "" {
		return errors.New("category English label is required")
	}
	if cat.Color != "" && !hexColorPattern.MatchString(cat.Color) {
		return errors.New("color must be a valid hex color (e.g. #FFF or #FF00FF)")
	}
	return nil
}
