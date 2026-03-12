package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func seedSecurityTransaction(t *testing.T, repo *SecurityTransactionRepository, tx *domain.SecurityTransaction) *domain.SecurityTransaction {
	t.Helper()
	if tx == nil {
		tx = &domain.SecurityTransaction{}
	}
	if tx.Year == 0 {
		tx.Year = 2025
	}
	if tx.AssetType == "" {
		tx.AssetType = domain.AssetTypeETF
	}
	if tx.AssetName == "" {
		tx.AssetName = "Vanguard S&P 500 ETF"
	}
	if tx.TransactionType == "" {
		tx.TransactionType = domain.TransactionTypeBuy
	}
	if tx.TransactionDate.IsZero() {
		tx.TransactionDate = time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	}
	if tx.Quantity == 0 {
		tx.Quantity = 10000 // 1 share (1/10000 units)
	}
	if tx.UnitPrice == 0 {
		tx.UnitPrice = domain.NewAmount(5000, 0)
	}
	if tx.TotalAmount == 0 {
		tx.TotalAmount = domain.NewAmount(5000, 0)
	}
	if tx.CurrencyCode == "" {
		tx.CurrencyCode = "USD"
	}
	if tx.ExchangeRate == 0 {
		tx.ExchangeRate = 230000 // 23.0 * 10000
	}

	ctx := context.Background()
	if err := repo.Create(ctx, tx); err != nil {
		t.Fatalf("seedSecurityTransaction: %v", err)
	}
	return tx
}

func TestSecurityTransactionRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	tx := &domain.SecurityTransaction{
		Year:            2025,
		AssetType:       domain.AssetTypeStock,
		AssetName:       "Apple Inc.",
		ISIN:            "US0378331005",
		TransactionType: domain.TransactionTypeBuy,
		TransactionDate: time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC),
		Quantity:        20000, // 2 shares
		UnitPrice:       domain.NewAmount(15000, 0),
		TotalAmount:     domain.NewAmount(30000, 0),
		Fees:            domain.NewAmount(50, 0),
		CurrencyCode:    "USD",
		ExchangeRate:    230000,
	}

	if err := repo.Create(ctx, tx); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if tx.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if tx.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestSecurityTransactionRepository_Create_WithDocumentID(t *testing.T) {
	db := testutil.NewTestDB(t)
	docRepo := NewInvestmentDocumentRepository(db)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	doc := seedInvestmentDocument(t, docRepo, nil)
	docID := doc.ID

	tx := &domain.SecurityTransaction{
		Year:            2025,
		DocumentID:      &docID,
		AssetType:       domain.AssetTypeETF,
		AssetName:       "iShares Core MSCI World",
		TransactionType: domain.TransactionTypeBuy,
		TransactionDate: time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC),
		Quantity:        50000,
		UnitPrice:       domain.NewAmount(8000, 0),
		TotalAmount:     domain.NewAmount(40000, 0),
		CurrencyCode:    "EUR",
		ExchangeRate:    250000,
	}

	if err := repo.Create(ctx, tx); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, tx.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.DocumentID == nil || *got.DocumentID != docID {
		t.Errorf("DocumentID = %v, want %d", got.DocumentID, docID)
	}
}

func TestSecurityTransactionRepository_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	seeded := seedSecurityTransaction(t, repo, &domain.SecurityTransaction{
		AssetType: domain.AssetTypeStock,
		AssetName: "Microsoft Corp.",
		ISIN:      "US5949181045",
		Fees:      domain.NewAmount(25, 0),
	})

	got, err := repo.GetByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.AssetName != "Microsoft Corp." {
		t.Errorf("AssetName = %q, want %q", got.AssetName, "Microsoft Corp.")
	}
	if got.ISIN != "US5949181045" {
		t.Errorf("ISIN = %q, want %q", got.ISIN, "US5949181045")
	}
	if got.Fees != domain.NewAmount(25, 0) {
		t.Errorf("Fees = %d, want %d", got.Fees, domain.NewAmount(25, 0))
	}
}

func TestSecurityTransactionRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent transaction")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestSecurityTransactionRepository_ListByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	seedSecurityTransaction(t, repo, &domain.SecurityTransaction{Year: 2025, TransactionDate: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)})
	seedSecurityTransaction(t, repo, &domain.SecurityTransaction{Year: 2025, TransactionDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)})
	seedSecurityTransaction(t, repo, &domain.SecurityTransaction{Year: 2024, TransactionDate: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)})

	txs, err := repo.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear() error: %v", err)
	}
	if len(txs) != 2 {
		t.Errorf("len(txs) = %d, want 2", len(txs))
	}

	// Verify ordered by transaction_date ASC.
	if len(txs) >= 2 && txs[0].TransactionDate.After(txs[1].TransactionDate) {
		t.Error("expected transactions ordered by transaction_date ASC")
	}
}

func TestSecurityTransactionRepository_ListByYear_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	txs, err := repo.ListByYear(ctx, 2099)
	if err != nil {
		t.Fatalf("ListByYear() error: %v", err)
	}
	if len(txs) != 0 {
		t.Errorf("expected empty result, got %d", len(txs))
	}
}

func TestSecurityTransactionRepository_ListByDocumentID(t *testing.T) {
	db := testutil.NewTestDB(t)
	docRepo := NewInvestmentDocumentRepository(db)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	doc := seedInvestmentDocument(t, docRepo, nil)
	docID := doc.ID

	seedSecurityTransaction(t, repo, &domain.SecurityTransaction{DocumentID: &docID})
	seedSecurityTransaction(t, repo, &domain.SecurityTransaction{DocumentID: &docID})
	seedSecurityTransaction(t, repo, nil) // no document_id

	txs, err := repo.ListByDocumentID(ctx, docID)
	if err != nil {
		t.Fatalf("ListByDocumentID() error: %v", err)
	}
	if len(txs) != 2 {
		t.Errorf("len(txs) = %d, want 2", len(txs))
	}
}

func TestSecurityTransactionRepository_ListBuysForFIFO(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	// Two buy transactions for the same asset.
	seedSecurityTransaction(t, repo, &domain.SecurityTransaction{
		AssetName:       "AAPL",
		AssetType:       domain.AssetTypeStock,
		TransactionType: domain.TransactionTypeBuy,
		TransactionDate: time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
	})
	seedSecurityTransaction(t, repo, &domain.SecurityTransaction{
		AssetName:       "AAPL",
		AssetType:       domain.AssetTypeStock,
		TransactionType: domain.TransactionTypeBuy,
		TransactionDate: time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC),
	})
	// Sell transaction -- should NOT be returned.
	seedSecurityTransaction(t, repo, &domain.SecurityTransaction{
		AssetName:       "AAPL",
		AssetType:       domain.AssetTypeStock,
		TransactionType: domain.TransactionTypeSell,
		TransactionDate: time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC),
	})
	// Different asset -- should NOT be returned.
	seedSecurityTransaction(t, repo, &domain.SecurityTransaction{
		AssetName:       "MSFT",
		AssetType:       domain.AssetTypeStock,
		TransactionType: domain.TransactionTypeBuy,
	})

	buys, err := repo.ListBuysForFIFO(ctx, "AAPL", domain.AssetTypeStock)
	if err != nil {
		t.Fatalf("ListBuysForFIFO() error: %v", err)
	}
	if len(buys) != 2 {
		t.Errorf("len(buys) = %d, want 2", len(buys))
	}

	// Verify FIFO ordering by transaction_date ASC.
	if len(buys) >= 2 && buys[0].TransactionDate.After(buys[1].TransactionDate) {
		t.Error("expected buys ordered by transaction_date ASC for FIFO")
	}
}

func TestSecurityTransactionRepository_ListSellsByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	seedSecurityTransaction(t, repo, &domain.SecurityTransaction{
		Year:            2025,
		TransactionType: domain.TransactionTypeSell,
	})
	seedSecurityTransaction(t, repo, &domain.SecurityTransaction{
		Year:            2025,
		TransactionType: domain.TransactionTypeSell,
	})
	// Buy -- should NOT be returned.
	seedSecurityTransaction(t, repo, &domain.SecurityTransaction{
		Year:            2025,
		TransactionType: domain.TransactionTypeBuy,
	})
	// Different year sell -- should NOT be returned.
	seedSecurityTransaction(t, repo, &domain.SecurityTransaction{
		Year:            2024,
		TransactionType: domain.TransactionTypeSell,
		TransactionDate: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
	})

	sells, err := repo.ListSellsByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListSellsByYear() error: %v", err)
	}
	if len(sells) != 2 {
		t.Errorf("len(sells) = %d, want 2", len(sells))
	}
}

func TestSecurityTransactionRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	seeded := seedSecurityTransaction(t, repo, nil)

	seeded.AssetName = "Updated Asset"
	seeded.TotalAmount = domain.NewAmount(10000, 0)
	seeded.TimeTestExempt = true
	if err := repo.Update(ctx, seeded); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.AssetName != "Updated Asset" {
		t.Errorf("AssetName = %q, want %q", got.AssetName, "Updated Asset")
	}
	if got.TotalAmount != domain.NewAmount(10000, 0) {
		t.Errorf("TotalAmount = %d, want %d", got.TotalAmount, domain.NewAmount(10000, 0))
	}
	if !got.TimeTestExempt {
		t.Error("expected TimeTestExempt to be true")
	}
}

func TestSecurityTransactionRepository_Update_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	tx := &domain.SecurityTransaction{
		ID:              99999,
		Year:            2025,
		AssetType:       domain.AssetTypeStock,
		AssetName:       "Ghost",
		TransactionType: domain.TransactionTypeBuy,
		TransactionDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		CurrencyCode:    "USD",
	}
	err := repo.Update(ctx, tx)
	if err == nil {
		t.Error("expected error for non-existent transaction")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestSecurityTransactionRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	seeded := seedSecurityTransaction(t, repo, nil)

	if err := repo.Delete(ctx, seeded.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := repo.GetByID(ctx, seeded.ID)
	if err == nil {
		t.Error("expected error when getting deleted transaction")
	}
}

func TestSecurityTransactionRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent transaction")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestSecurityTransactionRepository_UpdateFIFOResults(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	seeded := seedSecurityTransaction(t, repo, &domain.SecurityTransaction{
		TransactionType: domain.TransactionTypeSell,
	})

	costBasis := domain.NewAmount(3000, 0)
	computedGain := domain.NewAmount(2000, 0)
	exemptAmount := domain.NewAmount(2000, 0)

	if err := repo.UpdateFIFOResults(ctx, seeded.ID, costBasis, computedGain, exemptAmount, true); err != nil {
		t.Fatalf("UpdateFIFOResults() error: %v", err)
	}

	got, err := repo.GetByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.CostBasis != costBasis {
		t.Errorf("CostBasis = %d, want %d", got.CostBasis, costBasis)
	}
	if got.ComputedGain != computedGain {
		t.Errorf("ComputedGain = %d, want %d", got.ComputedGain, computedGain)
	}
	if got.ExemptAmount != exemptAmount {
		t.Errorf("ExemptAmount = %d, want %d", got.ExemptAmount, exemptAmount)
	}
	if !got.TimeTestExempt {
		t.Error("expected TimeTestExempt to be true")
	}
}

func TestSecurityTransactionRepository_UpdateFIFOResults_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	err := repo.UpdateFIFOResults(ctx, 99999, 0, 0, 0, false)
	if err == nil {
		t.Error("expected error for non-existent transaction")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestSecurityTransactionRepository_DeleteByDocumentID(t *testing.T) {
	db := testutil.NewTestDB(t)
	docRepo := NewInvestmentDocumentRepository(db)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	doc := seedInvestmentDocument(t, docRepo, nil)
	docID := doc.ID

	seedSecurityTransaction(t, repo, &domain.SecurityTransaction{DocumentID: &docID})
	seedSecurityTransaction(t, repo, &domain.SecurityTransaction{DocumentID: &docID})

	if err := repo.DeleteByDocumentID(ctx, docID); err != nil {
		t.Fatalf("DeleteByDocumentID() error: %v", err)
	}

	txs, err := repo.ListByDocumentID(ctx, docID)
	if err != nil {
		t.Fatalf("ListByDocumentID() error: %v", err)
	}
	if len(txs) != 0 {
		t.Errorf("expected 0 transactions after delete, got %d", len(txs))
	}
}

func TestSecurityTransactionRepository_DeleteByDocumentID_NoEntries(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSecurityTransactionRepository(db)
	ctx := context.Background()

	// Should not error even when no transactions exist for the document.
	if err := repo.DeleteByDocumentID(ctx, 99999); err != nil {
		t.Fatalf("DeleteByDocumentID() should not error for non-existent document, got: %v", err)
	}
}
