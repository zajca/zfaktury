package service

import (
	"context"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func newCategoryService(t *testing.T) (*CategoryService, *repository.CategoryRepository) {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := repository.NewCategoryRepository(db)
	svc := NewCategoryService(repo, nil)
	return svc, repo
}

func TestCategoryService_Create_Valid(t *testing.T) {
	svc, _ := newCategoryService(t)
	ctx := context.Background()

	cat := &domain.ExpenseCategory{
		Key:     "new_category",
		LabelCS: "Nova kategorie",
		LabelEN: "New category",
	}

	if err := svc.Create(ctx, cat); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if cat.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if cat.Color != "#6B7280" {
		t.Errorf("Color = %q, want default #6B7280", cat.Color)
	}
}

func TestCategoryService_Create_MissingKey(t *testing.T) {
	svc, _ := newCategoryService(t)
	ctx := context.Background()

	cat := &domain.ExpenseCategory{
		LabelCS: "Test",
		LabelEN: "Test",
	}

	err := svc.Create(ctx, cat)
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestCategoryService_Create_InvalidKeyFormat(t *testing.T) {
	svc, _ := newCategoryService(t)
	ctx := context.Background()

	tests := []struct {
		name string
		key  string
	}{
		{"uppercase", "INVALID"},
		{"spaces", "has spaces"},
		{"dashes", "has-dashes"},
		{"special chars", "has@special"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat := &domain.ExpenseCategory{
				Key:     tt.key,
				LabelCS: "Test",
				LabelEN: "Test",
			}
			err := svc.Create(ctx, cat)
			if err == nil {
				t.Errorf("expected error for invalid key %q", tt.key)
			}
		})
	}
}

func TestCategoryService_Create_MissingLabelCS(t *testing.T) {
	svc, _ := newCategoryService(t)
	ctx := context.Background()

	cat := &domain.ExpenseCategory{
		Key:     "valid_key",
		LabelEN: "English",
	}

	err := svc.Create(ctx, cat)
	if err == nil {
		t.Error("expected error for missing Czech label")
	}
}

func TestCategoryService_Create_MissingLabelEN(t *testing.T) {
	svc, _ := newCategoryService(t)
	ctx := context.Background()

	cat := &domain.ExpenseCategory{
		Key:     "valid_key",
		LabelCS: "Czech",
	}

	err := svc.Create(ctx, cat)
	if err == nil {
		t.Error("expected error for missing English label")
	}
}

func TestCategoryService_Create_DuplicateKey(t *testing.T) {
	svc, _ := newCategoryService(t)
	ctx := context.Background()

	// "software" is a default category from seed data.
	cat := &domain.ExpenseCategory{
		Key:     "software",
		LabelCS: "Duplikat",
		LabelEN: "Duplicate",
	}

	err := svc.Create(ctx, cat)
	if err == nil {
		t.Error("expected error for duplicate key")
	}
}

func TestCategoryService_Update_Valid(t *testing.T) {
	svc, _ := newCategoryService(t)
	ctx := context.Background()

	cat := &domain.ExpenseCategory{
		Key:     "update_svc",
		LabelCS: "Pred",
		LabelEN: "Before",
	}
	if err := svc.Create(ctx, cat); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	cat.LabelCS = "Po"
	cat.LabelEN = "After"
	if err := svc.Update(ctx, cat); err != nil {
		t.Fatalf("Update() error: %v", err)
	}
}

func TestCategoryService_Update_MissingID(t *testing.T) {
	svc, _ := newCategoryService(t)
	ctx := context.Background()

	cat := &domain.ExpenseCategory{
		Key:     "test",
		LabelCS: "Test",
		LabelEN: "Test",
	}
	err := svc.Update(ctx, cat)
	if err == nil {
		t.Error("expected error for missing ID")
	}
}

func TestCategoryService_Delete_Default_Protected(t *testing.T) {
	svc, _ := newCategoryService(t)
	ctx := context.Background()

	// List to find a default category.
	categories, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	var defaultCat *domain.ExpenseCategory
	for i := range categories {
		if categories[i].IsDefault {
			defaultCat = &categories[i]
			break
		}
	}

	if defaultCat == nil {
		t.Fatal("no default categories found in seed data")
	}

	err = svc.Delete(ctx, defaultCat.ID)
	if err == nil {
		t.Error("expected error when deleting default category")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("error = %v, want %v", err, domain.ErrInvalidInput)
	}
}

func TestCategoryService_Delete_Custom(t *testing.T) {
	svc, _ := newCategoryService(t)
	ctx := context.Background()

	cat := &domain.ExpenseCategory{
		Key:     "delete_svc",
		LabelCS: "Smazat",
		LabelEN: "Delete",
	}
	if err := svc.Create(ctx, cat); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.Delete(ctx, cat.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := svc.GetByID(ctx, cat.ID)
	if err == nil {
		t.Error("expected error when getting deleted category")
	}
}

func TestCategoryService_List(t *testing.T) {
	svc, _ := newCategoryService(t)
	ctx := context.Background()

	categories, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(categories) < 16 {
		t.Errorf("expected at least 16 default categories, got %d", len(categories))
	}
}
