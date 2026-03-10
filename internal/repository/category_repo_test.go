package repository

import (
	"context"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestCategoryRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	cat := &domain.ExpenseCategory{
		Key:       "test_cat",
		LabelCS:   "Testovaci kategorie",
		LabelEN:   "Test category",
		Color:     "#FF0000",
		SortOrder: 50,
	}

	if err := repo.Create(ctx, cat); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if cat.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if cat.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestCategoryRepository_Create_DuplicateKey(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	cat1 := &domain.ExpenseCategory{
		Key:     "duplicate_key",
		LabelCS: "Prvni",
		LabelEN: "First",
	}
	if err := repo.Create(ctx, cat1); err != nil {
		t.Fatalf("Create() first: %v", err)
	}

	cat2 := &domain.ExpenseCategory{
		Key:     "duplicate_key",
		LabelCS: "Druha",
		LabelEN: "Second",
	}
	err := repo.Create(ctx, cat2)
	if err == nil {
		t.Error("expected error for duplicate key")
	}
}

func TestCategoryRepository_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	cat := &domain.ExpenseCategory{
		Key:       "get_test",
		LabelCS:   "Test",
		LabelEN:   "Test",
		Color:     "#123456",
		SortOrder: 10,
	}
	if err := repo.Create(ctx, cat); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, cat.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.Key != "get_test" {
		t.Errorf("Key = %q, want %q", got.Key, "get_test")
	}
	if got.LabelCS != "Test" {
		t.Errorf("LabelCS = %q, want %q", got.LabelCS, "Test")
	}
	if got.Color != "#123456" {
		t.Errorf("Color = %q, want %q", got.Color, "#123456")
	}
	if got.SortOrder != 10 {
		t.Errorf("SortOrder = %d, want %d", got.SortOrder, 10)
	}
}

func TestCategoryRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent category")
	}
}

func TestCategoryRepository_GetByKey(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	// Default categories are seeded by migration, try fetching one.
	got, err := repo.GetByKey(ctx, "software")
	if err != nil {
		t.Fatalf("GetByKey() error: %v", err)
	}

	if got.Key != "software" {
		t.Errorf("Key = %q, want %q", got.Key, "software")
	}
	if got.LabelCS != "Software a licence" {
		t.Errorf("LabelCS = %q, want %q", got.LabelCS, "Software a licence")
	}
	if !got.IsDefault {
		t.Error("expected IsDefault to be true for seeded category")
	}
}

func TestCategoryRepository_GetByKey_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	_, err := repo.GetByKey(ctx, "nonexistent_key")
	if err == nil {
		t.Error("expected error for non-existent key")
	}
}

func TestCategoryRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	cat := &domain.ExpenseCategory{
		Key:     "update_test",
		LabelCS: "Pred",
		LabelEN: "Before",
		Color:   "#000000",
	}
	if err := repo.Create(ctx, cat); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	cat.LabelCS = "Po"
	cat.LabelEN = "After"
	cat.Color = "#FFFFFF"
	if err := repo.Update(ctx, cat); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, cat.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.LabelCS != "Po" {
		t.Errorf("LabelCS = %q, want %q", got.LabelCS, "Po")
	}
	if got.LabelEN != "After" {
		t.Errorf("LabelEN = %q, want %q", got.LabelEN, "After")
	}
	if got.Color != "#FFFFFF" {
		t.Errorf("Color = %q, want %q", got.Color, "#FFFFFF")
	}
}

func TestCategoryRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	cat := &domain.ExpenseCategory{
		Key:     "delete_test",
		LabelCS: "Smazat",
		LabelEN: "Delete",
	}
	if err := repo.Create(ctx, cat); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := repo.Delete(ctx, cat.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := repo.GetByID(ctx, cat.ID)
	if err == nil {
		t.Error("expected error when getting deleted category")
	}
}

func TestCategoryRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent category")
	}
}

func TestCategoryRepository_List_Order(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	categories, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	// Default categories should be seeded (16 total).
	if len(categories) < 16 {
		t.Errorf("expected at least 16 default categories, got %d", len(categories))
	}

	// Verify ordering by sort_order.
	for i := 1; i < len(categories); i++ {
		if categories[i].SortOrder < categories[i-1].SortOrder {
			t.Errorf("categories not sorted: [%d].SortOrder=%d < [%d].SortOrder=%d",
				i, categories[i].SortOrder, i-1, categories[i-1].SortOrder)
		}
	}
}

func TestCategoryRepository_List_ExcludesSoftDeleted(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCategoryRepository(db)
	ctx := context.Background()

	cat := &domain.ExpenseCategory{
		Key:     "soft_delete_test",
		LabelCS: "Smazat",
		LabelEN: "Delete",
	}
	if err := repo.Create(ctx, cat); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	beforeList, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() before error: %v", err)
	}
	beforeCount := len(beforeList)

	if err := repo.Delete(ctx, cat.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	afterList, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() after error: %v", err)
	}

	if len(afterList) != beforeCount-1 {
		t.Errorf("expected %d categories after delete, got %d", beforeCount-1, len(afterList))
	}
}
