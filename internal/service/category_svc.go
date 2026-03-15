package service

import (
	"context"
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
	repo  repository.CategoryRepo
	audit *AuditService
}

// NewCategoryService creates a new CategoryService.
func NewCategoryService(repo repository.CategoryRepo, audit *AuditService) *CategoryService {
	return &CategoryService{repo: repo, audit: audit}
}

// Create validates and persists a new expense category.
func (s *CategoryService) Create(ctx context.Context, cat *domain.ExpenseCategory) error {
	if err := s.validateCategory(cat); err != nil {
		return err
	}

	// Check for duplicate key.
	existing, err := s.repo.GetByKey(ctx, cat.Key)
	if err == nil && existing != nil {
		return fmt.Errorf("category with this key already exists: %w", domain.ErrDuplicateNumber)
	}

	if cat.Color == "" {
		cat.Color = "#6B7280"
	}

	if err := s.repo.Create(ctx, cat); err != nil {
		return fmt.Errorf("creating category: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "category", cat.ID, "create", nil, cat)
	}
	return nil
}

// Update validates and updates an existing expense category.
func (s *CategoryService) Update(ctx context.Context, cat *domain.ExpenseCategory) error {
	if cat.ID == 0 {
		return fmt.Errorf("category ID is required: %w", domain.ErrInvalidInput)
	}

	if err := s.validateCategory(cat); err != nil {
		return err
	}

	// Check for duplicate key (excluding self).
	existingByKey, err := s.repo.GetByKey(ctx, cat.Key)
	if err == nil && existingByKey != nil && existingByKey.ID != cat.ID {
		return fmt.Errorf("category with this key already exists: %w", domain.ErrDuplicateNumber)
	}

	// Fetch existing state for audit logging.
	existing, err := s.repo.GetByID(ctx, cat.ID)
	if err != nil {
		return fmt.Errorf("fetching category for update: %w", err)
	}

	if err := s.repo.Update(ctx, cat); err != nil {
		return fmt.Errorf("updating category: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "category", cat.ID, "update", existing, cat)
	}
	return nil
}

// Delete removes an expense category by ID (soft delete).
// Default categories (is_default=1) cannot be deleted.
func (s *CategoryService) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return fmt.Errorf("category ID is required: %w", domain.ErrInvalidInput)
	}

	cat, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching category for delete: %w", err)
	}

	if cat.IsDefault {
		return fmt.Errorf("default categories cannot be deleted: %w", domain.ErrInvalidInput)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting category: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "category", id, "delete", nil, nil)
	}
	return nil
}

// GetByID retrieves an expense category by its ID.
func (s *CategoryService) GetByID(ctx context.Context, id int64) (*domain.ExpenseCategory, error) {
	if id == 0 {
		return nil, fmt.Errorf("category ID is required: %w", domain.ErrInvalidInput)
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
		return fmt.Errorf("category key is required: %w", domain.ErrInvalidInput)
	}
	if !keyPattern.MatchString(cat.Key) {
		return fmt.Errorf("category key must be lowercase alphanumeric with underscores only: %w", domain.ErrInvalidInput)
	}
	if cat.LabelCS == "" {
		return fmt.Errorf("category Czech label is required: %w", domain.ErrInvalidInput)
	}
	if cat.LabelEN == "" {
		return fmt.Errorf("category English label is required: %w", domain.ErrInvalidInput)
	}
	if cat.Color != "" && !hexColorPattern.MatchString(cat.Color) {
		return fmt.Errorf("color must be a valid hex color (e.g. #FFF or #FF00FF): %w", domain.ErrInvalidInput)
	}
	return nil
}
