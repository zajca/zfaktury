package repository

import (
	"context"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func seedDocument(t *testing.T, repo *DocumentRepository, expenseID int64, filename string) *domain.ExpenseDocument {
	t.Helper()
	doc := &domain.ExpenseDocument{
		ExpenseID:   expenseID,
		Filename:    filename,
		ContentType: "application/pdf",
		StoragePath: "/test/documents/" + filename,
		Size:        1024,
	}
	if err := repo.Create(context.Background(), doc); err != nil {
		t.Fatalf("seeding document: %v", err)
	}
	return doc
}

func TestDocumentRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	expenseRepo := NewExpenseRepository(db)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	expense := testutil.SeedExpense(t, db, nil)
	_ = expenseRepo

	doc := &domain.ExpenseDocument{
		ExpenseID:   expense.ID,
		Filename:    "receipt.pdf",
		ContentType: "application/pdf",
		StoragePath: "/data/documents/1/uuid_receipt.pdf",
		Size:        2048,
	}

	if err := repo.Create(ctx, doc); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if doc.ID == 0 {
		t.Error("expected non-zero ID after create")
	}
	if doc.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestDocumentRepository_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	expense := testutil.SeedExpense(t, db, nil)
	created := seedDocument(t, repo, expense.ID, "invoice.pdf")

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.Filename != "invoice.pdf" {
		t.Errorf("Filename = %q, want %q", got.Filename, "invoice.pdf")
	}
	if got.ExpenseID != expense.ID {
		t.Errorf("ExpenseID = %d, want %d", got.ExpenseID, expense.ID)
	}
	if got.ContentType != "application/pdf" {
		t.Errorf("ContentType = %q, want %q", got.ContentType, "application/pdf")
	}
	if got.Size != 1024 {
		t.Errorf("Size = %d, want 1024", got.Size)
	}
}

func TestDocumentRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent document")
	}
}

func TestDocumentRepository_ListByExpenseID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	expense1 := testutil.SeedExpense(t, db, nil)
	expense2 := testutil.SeedExpense(t, db, nil)

	seedDocument(t, repo, expense1.ID, "doc1.pdf")
	seedDocument(t, repo, expense1.ID, "doc2.png")
	seedDocument(t, repo, expense2.ID, "other.jpg")

	docs, err := repo.ListByExpenseID(ctx, expense1.ID)
	if err != nil {
		t.Fatalf("ListByExpenseID() error: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("len = %d, want 2", len(docs))
	}
}

func TestDocumentRepository_ListByExpenseID_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	expense := testutil.SeedExpense(t, db, nil)

	docs, err := repo.ListByExpenseID(ctx, expense.ID)
	if err != nil {
		t.Fatalf("ListByExpenseID() error: %v", err)
	}

	if len(docs) != 0 {
		t.Errorf("expected empty list, got %d docs", len(docs))
	}
}

func TestDocumentRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	expense := testutil.SeedExpense(t, db, nil)
	doc := seedDocument(t, repo, expense.ID, "todelete.pdf")

	if err := repo.Delete(ctx, doc.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// Should not be returned by GetByID after soft delete.
	_, err := repo.GetByID(ctx, doc.ID)
	if err == nil {
		t.Error("expected error when getting soft-deleted document")
	}
}

func TestDocumentRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent document")
	}
}

func TestDocumentRepository_Delete_ExcludesFromList(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	expense := testutil.SeedExpense(t, db, nil)
	toDelete := seedDocument(t, repo, expense.ID, "delete_me.pdf")
	seedDocument(t, repo, expense.ID, "keep_me.pdf")

	if err := repo.Delete(ctx, toDelete.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	docs, err := repo.ListByExpenseID(ctx, expense.ID)
	if err != nil {
		t.Fatalf("ListByExpenseID() error: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("expected 1 doc after delete, got %d", len(docs))
	}
	if docs[0].Filename != "keep_me.pdf" {
		t.Errorf("remaining doc = %q, want %q", docs[0].Filename, "keep_me.pdf")
	}
}

func TestDocumentRepository_CountByExpenseID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	expense := testutil.SeedExpense(t, db, nil)
	seedDocument(t, repo, expense.ID, "a.pdf")
	seedDocument(t, repo, expense.ID, "b.pdf")
	seedDocument(t, repo, expense.ID, "c.pdf")

	count, err := repo.CountByExpenseID(ctx, expense.ID)
	if err != nil {
		t.Fatalf("CountByExpenseID() error: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestDocumentRepository_CountByExpenseID_ExcludesDeleted(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDocumentRepository(db)
	ctx := context.Background()

	expense := testutil.SeedExpense(t, db, nil)
	toDelete := seedDocument(t, repo, expense.ID, "delete_me.pdf")
	seedDocument(t, repo, expense.ID, "keep.pdf")

	if err := repo.Delete(ctx, toDelete.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	count, err := repo.CountByExpenseID(ctx, expense.ID)
	if err != nil {
		t.Fatalf("CountByExpenseID() error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (deleted should be excluded)", count)
	}
}
