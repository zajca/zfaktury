package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

// newInvestmentIncomeService creates a test InvestmentIncomeService backed by real SQLite.
func newInvestmentIncomeService(t *testing.T) (*InvestmentIncomeService, *repository.CapitalIncomeRepository, *repository.SecurityTransactionRepository) {
	t.Helper()
	db := testutil.NewTestDB(t)
	capitalRepo := repository.NewCapitalIncomeRepository(db)
	securityRepo := repository.NewSecurityTransactionRepository(db)
	svc := NewInvestmentIncomeService(capitalRepo, securityRepo)
	return svc, capitalRepo, securityRepo
}

// testSecTx creates a SecurityTransaction with required fields populated.
func testSecTx(year int, assetType, assetName, txType string, txDate time.Time, qty int64, totalAmount domain.Amount) *domain.SecurityTransaction {
	return &domain.SecurityTransaction{
		Year:            year,
		AssetType:       assetType,
		AssetName:       assetName,
		ISIN:            "TEST000000",
		TransactionType: txType,
		TransactionDate: txDate,
		Quantity:        qty,
		TotalAmount:     totalAmount,
		CurrencyCode:    "CZK",
	}
}

// --- Capital Income CRUD ---

func TestInvestmentIncomeService_CreateCapitalEntry_Valid(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	entry := &domain.CapitalIncomeEntry{
		Year:               2025,
		Category:           domain.CapitalCategoryDividendCZ,
		Description:        "Test dividend",
		IncomeDate:         time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		GrossAmount:        10000,
		WithheldTaxCZ:      1500,
		WithheldTaxForeign: 0,
		CountryCode:        "CZ",
		NeedsDeclaring:     false,
	}

	if err := svc.CreateCapitalEntry(ctx, entry); err != nil {
		t.Fatalf("CreateCapitalEntry() error: %v", err)
	}
	if entry.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if entry.NetAmount != 8500 {
		t.Errorf("NetAmount = %d, want 8500", entry.NetAmount)
	}
}

func TestInvestmentIncomeService_CreateCapitalEntry_InvalidCategory(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	entry := &domain.CapitalIncomeEntry{
		Year:        2025,
		Category:    "invalid_cat",
		IncomeDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		GrossAmount: 5000,
	}

	err := svc.CreateCapitalEntry(ctx, entry)
	if err == nil {
		t.Error("expected error for invalid category")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestInvestmentIncomeService_CreateCapitalEntry_AllCategories(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	categories := []string{
		domain.CapitalCategoryDividendCZ,
		domain.CapitalCategoryDividendForeign,
		domain.CapitalCategoryInterest,
		domain.CapitalCategoryCoupon,
		domain.CapitalCategoryFundDist,
		domain.CapitalCategoryOther,
	}

	for _, cat := range categories {
		t.Run(cat, func(t *testing.T) {
			entry := &domain.CapitalIncomeEntry{
				Year:        2025,
				Category:    cat,
				Description: "Test " + cat,
				IncomeDate:  time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
				GrossAmount: 1000,
			}
			if err := svc.CreateCapitalEntry(ctx, entry); err != nil {
				t.Errorf("CreateCapitalEntry(%s) error: %v", cat, err)
			}
		})
	}
}

func TestInvestmentIncomeService_UpdateCapitalEntry_Valid(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	entry := &domain.CapitalIncomeEntry{
		Year:               2025,
		Category:           domain.CapitalCategoryInterest,
		Description:        "Before update",
		IncomeDate:         time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		GrossAmount:        5000,
		WithheldTaxCZ:      750,
		WithheldTaxForeign: 250,
	}
	if err := svc.CreateCapitalEntry(ctx, entry); err != nil {
		t.Fatalf("Create: %v", err)
	}

	entry.Description = "After update"
	entry.GrossAmount = 8000
	entry.WithheldTaxCZ = 1200
	entry.WithheldTaxForeign = 0
	if err := svc.UpdateCapitalEntry(ctx, entry); err != nil {
		t.Fatalf("UpdateCapitalEntry() error: %v", err)
	}
	if entry.NetAmount != 6800 {
		t.Errorf("NetAmount = %d, want 6800", entry.NetAmount)
	}

	got, err := svc.GetCapitalEntry(ctx, entry.ID)
	if err != nil {
		t.Fatalf("GetCapitalEntry() error: %v", err)
	}
	if got.Description != "After update" {
		t.Errorf("Description = %q, want %q", got.Description, "After update")
	}
}

func TestInvestmentIncomeService_UpdateCapitalEntry_InvalidCategory(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	entry := &domain.CapitalIncomeEntry{
		Year:        2025,
		Category:    domain.CapitalCategoryInterest,
		IncomeDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		GrossAmount: 5000,
	}
	if err := svc.CreateCapitalEntry(ctx, entry); err != nil {
		t.Fatalf("Create: %v", err)
	}

	entry.Category = "bad"
	err := svc.UpdateCapitalEntry(ctx, entry)
	if err == nil {
		t.Error("expected error for invalid category on update")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestInvestmentIncomeService_DeleteCapitalEntry(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	entry := &domain.CapitalIncomeEntry{
		Year:        2025,
		Category:    domain.CapitalCategoryDividendCZ,
		IncomeDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		GrossAmount: 1000,
	}
	if err := svc.CreateCapitalEntry(ctx, entry); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.DeleteCapitalEntry(ctx, entry.ID); err != nil {
		t.Fatalf("DeleteCapitalEntry() error: %v", err)
	}

	_, err := svc.GetCapitalEntry(ctx, entry.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestInvestmentIncomeService_GetCapitalEntry_NotFound(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	_, err := svc.GetCapitalEntry(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent entry")
	}
}

func TestInvestmentIncomeService_ListCapitalEntries(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		entry := &domain.CapitalIncomeEntry{
			Year:        2025,
			Category:    domain.CapitalCategoryDividendCZ,
			Description: "Year 2025 entry",
			IncomeDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			GrossAmount: 1000,
		}
		if err := svc.CreateCapitalEntry(ctx, entry); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}
	entry := &domain.CapitalIncomeEntry{
		Year:        2024,
		Category:    domain.CapitalCategoryDividendCZ,
		IncomeDate:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		GrossAmount: 500,
	}
	if err := svc.CreateCapitalEntry(ctx, entry); err != nil {
		t.Fatalf("Create 2024: %v", err)
	}

	entries, err := svc.ListCapitalEntries(ctx, 2025)
	if err != nil {
		t.Fatalf("ListCapitalEntries() error: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("len = %d, want 3", len(entries))
	}
}

func TestInvestmentIncomeService_ComputeCapitalIncomeTotals(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	// SumByYear only counts entries where needs_declaring = true.
	entries := []domain.CapitalIncomeEntry{
		{Year: 2025, Category: domain.CapitalCategoryDividendForeign, IncomeDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), GrossAmount: 10000, WithheldTaxCZ: 1500, NeedsDeclaring: true},
		{Year: 2025, Category: domain.CapitalCategoryInterest, IncomeDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), GrossAmount: 5000, WithheldTaxCZ: 750, NeedsDeclaring: true},
	}
	for i := range entries {
		if err := svc.CreateCapitalEntry(ctx, &entries[i]); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	gross, tax, net, err := svc.ComputeCapitalIncomeTotals(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeCapitalIncomeTotals() error: %v", err)
	}
	if gross != 15000 {
		t.Errorf("gross = %d, want 15000", gross)
	}
	if tax != 2250 {
		t.Errorf("tax = %d, want 2250", tax)
	}
	if net != 12750 {
		t.Errorf("net = %d, want 12750", net)
	}
}

func TestInvestmentIncomeService_ComputeCapitalIncomeTotals_EmptyYear(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	gross, tax, net, err := svc.ComputeCapitalIncomeTotals(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeCapitalIncomeTotals() error: %v", err)
	}
	if gross != 0 || tax != 0 || net != 0 {
		t.Errorf("expected all zeros, got gross=%d tax=%d net=%d", gross, tax, net)
	}
}

func TestInvestmentIncomeService_ComputeCapitalIncomeTotals_ExcludesNonDeclaring(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	// This entry has NeedsDeclaring=false, so it should NOT be counted.
	entry := &domain.CapitalIncomeEntry{
		Year:           2025,
		Category:       domain.CapitalCategoryDividendCZ,
		IncomeDate:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		GrossAmount:    5000,
		WithheldTaxCZ:  750,
		NeedsDeclaring: false,
	}
	if err := svc.CreateCapitalEntry(ctx, entry); err != nil {
		t.Fatalf("Create: %v", err)
	}

	gross, _, _, err := svc.ComputeCapitalIncomeTotals(ctx, 2025)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if gross != 0 {
		t.Errorf("gross = %d, want 0 (non-declaring entry should be excluded)", gross)
	}
}

// --- Security Transaction CRUD ---

func TestInvestmentIncomeService_CreateSecurityTransaction_Valid(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	tx := testSecTx(2025, domain.AssetTypeStock, "AAPL", domain.TransactionTypeBuy,
		time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), 10000, 1500000)
	tx.ISIN = "US0378331005"
	tx.UnitPrice = 1500000
	tx.Fees = 5000

	if err := svc.CreateSecurityTransaction(ctx, tx); err != nil {
		t.Fatalf("CreateSecurityTransaction() error: %v", err)
	}
	if tx.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestInvestmentIncomeService_CreateSecurityTransaction_InvalidAssetType(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	tx := &domain.SecurityTransaction{
		Year:            2025,
		AssetType:       "invalid",
		TransactionType: domain.TransactionTypeBuy,
		TransactionDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Quantity:        10000,
	}

	err := svc.CreateSecurityTransaction(ctx, tx)
	if err == nil {
		t.Error("expected error for invalid asset type")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestInvestmentIncomeService_CreateSecurityTransaction_InvalidTransactionType(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	tx := &domain.SecurityTransaction{
		Year:            2025,
		AssetType:       domain.AssetTypeStock,
		TransactionType: "hold",
		TransactionDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Quantity:        10000,
	}

	err := svc.CreateSecurityTransaction(ctx, tx)
	if err == nil {
		t.Error("expected error for invalid transaction type")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestInvestmentIncomeService_CreateSecurityTransaction_AllAssetTypes(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	types := []string{
		domain.AssetTypeStock,
		domain.AssetTypeETF,
		domain.AssetTypeBond,
		domain.AssetTypeFund,
		domain.AssetTypeCrypto,
		domain.AssetTypeOther,
	}

	for _, at := range types {
		t.Run(at, func(t *testing.T) {
			tx := testSecTx(2025, at, "Test-"+at, domain.TransactionTypeBuy,
				time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 10000, 100000)
			if err := svc.CreateSecurityTransaction(ctx, tx); err != nil {
				t.Errorf("CreateSecurityTransaction(%s) error: %v", at, err)
			}
		})
	}
}

func TestInvestmentIncomeService_UpdateSecurityTransaction_Valid(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	tx := testSecTx(2025, domain.AssetTypeETF, "VWCE", domain.TransactionTypeBuy,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 50000, 4000000)
	tx.UnitPrice = 800000
	if err := svc.CreateSecurityTransaction(ctx, tx); err != nil {
		t.Fatalf("Create: %v", err)
	}

	tx.AssetName = "VWCE.DE"
	tx.TotalAmount = 4100000
	if err := svc.UpdateSecurityTransaction(ctx, tx); err != nil {
		t.Fatalf("UpdateSecurityTransaction() error: %v", err)
	}

	got, err := svc.GetSecurityTransaction(ctx, tx.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.AssetName != "VWCE.DE" {
		t.Errorf("AssetName = %q, want %q", got.AssetName, "VWCE.DE")
	}
}

func TestInvestmentIncomeService_UpdateSecurityTransaction_InvalidAssetType(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	tx := testSecTx(2025, domain.AssetTypeStock, "X", domain.TransactionTypeBuy,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 10000, 100000)
	if err := svc.CreateSecurityTransaction(ctx, tx); err != nil {
		t.Fatalf("Create: %v", err)
	}

	tx.AssetType = "invalid"
	err := svc.UpdateSecurityTransaction(ctx, tx)
	if err == nil {
		t.Error("expected error for invalid asset type on update")
	}
}

func TestInvestmentIncomeService_UpdateSecurityTransaction_InvalidTransactionType(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	tx := testSecTx(2025, domain.AssetTypeStock, "X", domain.TransactionTypeBuy,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 10000, 100000)
	if err := svc.CreateSecurityTransaction(ctx, tx); err != nil {
		t.Fatalf("Create: %v", err)
	}

	tx.TransactionType = "invalid"
	err := svc.UpdateSecurityTransaction(ctx, tx)
	if err == nil {
		t.Error("expected error for invalid transaction type on update")
	}
}

func TestInvestmentIncomeService_DeleteSecurityTransaction(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	tx := testSecTx(2025, domain.AssetTypeStock, "MSFT", domain.TransactionTypeBuy,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 10000, 500000)
	if err := svc.CreateSecurityTransaction(ctx, tx); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.DeleteSecurityTransaction(ctx, tx.ID); err != nil {
		t.Fatalf("DeleteSecurityTransaction() error: %v", err)
	}

	_, err := svc.GetSecurityTransaction(ctx, tx.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestInvestmentIncomeService_GetSecurityTransaction_NotFound(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	_, err := svc.GetSecurityTransaction(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent transaction")
	}
}

func TestInvestmentIncomeService_ListSecurityTransactions(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		tx := testSecTx(2025, domain.AssetTypeStock, "AAPL", domain.TransactionTypeBuy,
			time.Date(2025, 1, 1+i, 0, 0, 0, 0, time.UTC), 10000, 100000)
		if err := svc.CreateSecurityTransaction(ctx, tx); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	txs, err := svc.ListSecurityTransactions(ctx, 2025)
	if err != nil {
		t.Fatalf("ListSecurityTransactions() error: %v", err)
	}
	if len(txs) != 2 {
		t.Errorf("len = %d, want 2", len(txs))
	}
}

// --- FIFO Calculation ---

func TestInvestmentIncomeService_RecalculateFIFO_NoSells(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	buy := testSecTx(2025, domain.AssetTypeStock, "AAPL", domain.TransactionTypeBuy,
		time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), 100000, 1000000)
	if err := svc.CreateSecurityTransaction(ctx, buy); err != nil {
		t.Fatalf("Create buy: %v", err)
	}

	if err := svc.RecalculateFIFO(ctx, 2025); err != nil {
		t.Fatalf("RecalculateFIFO() error: %v", err)
	}
}

func TestInvestmentIncomeService_RecalculateFIFO_SimpleSell(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	// Buy 10 shares at total 10M halere.
	buy := testSecTx(2025, domain.AssetTypeStock, "AAPL", domain.TransactionTypeBuy,
		time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), 100000, 10000000)
	buy.UnitPrice = 1000000
	if err := svc.CreateSecurityTransaction(ctx, buy); err != nil {
		t.Fatalf("Create buy: %v", err)
	}

	// Sell 5 shares at total 7.5M halere with 5000 halere fees.
	sell := testSecTx(2025, domain.AssetTypeStock, "AAPL", domain.TransactionTypeSell,
		time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), 50000, 7500000)
	sell.UnitPrice = 1500000
	sell.Fees = 5000
	if err := svc.CreateSecurityTransaction(ctx, sell); err != nil {
		t.Fatalf("Create sell: %v", err)
	}

	if err := svc.RecalculateFIFO(ctx, 2025); err != nil {
		t.Fatalf("RecalculateFIFO() error: %v", err)
	}

	got, err := svc.GetSecurityTransaction(ctx, sell.ID)
	if err != nil {
		t.Fatalf("Get sell: %v", err)
	}

	// Cost basis = buy.TotalAmount * (50000/100000) = 5000000
	if got.CostBasis != 5000000 {
		t.Errorf("CostBasis = %d, want 5000000", got.CostBasis)
	}

	// Gain = 7500000 - 5000 - 5000000 = 2495000
	expectedGain := domain.Amount(2495000)
	if got.ComputedGain != expectedGain {
		t.Errorf("ComputedGain = %d, want %d", got.ComputedGain, expectedGain)
	}

	// Time test: buy 2022, sell 2025 (3+ years). Should be exempt.
	if !got.TimeTestExempt {
		t.Error("expected TimeTestExempt = true (held > 3 years)")
	}
}

func TestInvestmentIncomeService_RecalculateFIFO_TimeTestNotMet(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	buy := testSecTx(2025, domain.AssetTypeStock, "GOOG", domain.TransactionTypeBuy,
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 10000, 1000000)
	if err := svc.CreateSecurityTransaction(ctx, buy); err != nil {
		t.Fatalf("Create buy: %v", err)
	}

	sell := testSecTx(2025, domain.AssetTypeStock, "GOOG", domain.TransactionTypeSell,
		time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), 10000, 2000000)
	if err := svc.CreateSecurityTransaction(ctx, sell); err != nil {
		t.Fatalf("Create sell: %v", err)
	}

	if err := svc.RecalculateFIFO(ctx, 2025); err != nil {
		t.Fatalf("RecalculateFIFO() error: %v", err)
	}

	got, err := svc.GetSecurityTransaction(ctx, sell.ID)
	if err != nil {
		t.Fatalf("Get sell: %v", err)
	}
	if got.TimeTestExempt {
		t.Error("expected TimeTestExempt = false (held < 3 years)")
	}
	if got.ExemptAmount != 0 {
		t.Errorf("ExemptAmount = %d, want 0", got.ExemptAmount)
	}
}

func TestInvestmentIncomeService_RecalculateFIFO_MultipleBuys(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	buy1 := testSecTx(2025, domain.AssetTypeETF, "VWCE", domain.TransactionTypeBuy,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), 50000, 5000000)
	if err := svc.CreateSecurityTransaction(ctx, buy1); err != nil {
		t.Fatalf("Create buy1: %v", err)
	}

	buy2 := testSecTx(2025, domain.AssetTypeETF, "VWCE", domain.TransactionTypeBuy,
		time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), 50000, 10000000)
	if err := svc.CreateSecurityTransaction(ctx, buy2); err != nil {
		t.Fatalf("Create buy2: %v", err)
	}

	// Sell 7 shares. FIFO: 5 from buy1 + 2 from buy2.
	sell := testSecTx(2025, domain.AssetTypeETF, "VWCE", domain.TransactionTypeSell,
		time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), 70000, 21000000)
	if err := svc.CreateSecurityTransaction(ctx, sell); err != nil {
		t.Fatalf("Create sell: %v", err)
	}

	if err := svc.RecalculateFIFO(ctx, 2025); err != nil {
		t.Fatalf("RecalculateFIFO() error: %v", err)
	}

	got, err := svc.GetSecurityTransaction(ctx, sell.ID)
	if err != nil {
		t.Fatalf("Get sell: %v", err)
	}

	// Cost basis = all buy1 (5000000) + 2/5 of buy2 (10000000 * 20000/50000 = 4000000) = 9000000
	expectedCostBasis := domain.Amount(9000000)
	if got.CostBasis != expectedCostBasis {
		t.Errorf("CostBasis = %d, want %d", got.CostBasis, expectedCostBasis)
	}
}

func TestInvestmentIncomeService_RecalculateFIFO_ExemptionLimit2025(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	// SecurityExemptionLimit for 2025 = NewAmount(100_000_000, 0) = 10_000_000_000 halere.
	// Create two sells whose combined exempt gains exceed the limit.
	buy1 := testSecTx(2025, domain.AssetTypeStock, "BIG1", domain.TransactionTypeBuy,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), 10000, 100)
	if err := svc.CreateSecurityTransaction(ctx, buy1); err != nil {
		t.Fatalf("Create buy1: %v", err)
	}

	// First sell: gain just under the limit.
	sell1 := testSecTx(2025, domain.AssetTypeStock, "BIG1", domain.TransactionTypeSell,
		time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), 10000, 9_000_000_000)
	if err := svc.CreateSecurityTransaction(ctx, sell1); err != nil {
		t.Fatalf("Create sell1: %v", err)
	}

	buy2 := testSecTx(2025, domain.AssetTypeStock, "BIG2", domain.TransactionTypeBuy,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), 10000, 100)
	if err := svc.CreateSecurityTransaction(ctx, buy2); err != nil {
		t.Fatalf("Create buy2: %v", err)
	}

	// Second sell: gain that would push total over the limit.
	sell2 := testSecTx(2025, domain.AssetTypeStock, "BIG2", domain.TransactionTypeSell,
		time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC), 10000, 3_000_000_000)
	if err := svc.CreateSecurityTransaction(ctx, sell2); err != nil {
		t.Fatalf("Create sell2: %v", err)
	}

	if err := svc.RecalculateFIFO(ctx, 2025); err != nil {
		t.Fatalf("RecalculateFIFO() error: %v", err)
	}

	got1, err := svc.GetSecurityTransaction(ctx, sell1.ID)
	if err != nil {
		t.Fatalf("Get sell1: %v", err)
	}
	got2, err := svc.GetSecurityTransaction(ctx, sell2.ID)
	if err != nil {
		t.Fatalf("Get sell2: %v", err)
	}

	if !got1.TimeTestExempt {
		t.Error("sell1: expected TimeTestExempt = true")
	}
	if !got2.TimeTestExempt {
		t.Error("sell2: expected TimeTestExempt = true")
	}

	// Total exempt should be capped at the limit (10_000_000_000).
	totalExempt := got1.ExemptAmount + got2.ExemptAmount
	limit := domain.NewAmount(100_000_000, 0) // 10_000_000_000 halere
	if totalExempt > limit {
		t.Errorf("total exempt = %d, should not exceed limit %d", totalExempt, limit)
	}
	// sell2's exempt amount should be less than its full gain (it's capped).
	sell2Gain := got2.ComputedGain
	if got2.ExemptAmount >= sell2Gain {
		t.Errorf("sell2 exempt %d should be less than full gain %d (capped by limit)", got2.ExemptAmount, sell2Gain)
	}
}

func TestInvestmentIncomeService_RecalculateFIFO_NoLimit2024(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	buy := testSecTx(2024, domain.AssetTypeStock, "BIG", domain.TransactionTypeBuy,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), 10000, 100000)
	if err := svc.CreateSecurityTransaction(ctx, buy); err != nil {
		t.Fatalf("Create buy: %v", err)
	}

	sell := testSecTx(2024, domain.AssetTypeStock, "BIG", domain.TransactionTypeSell,
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), 10000, 200000000)
	if err := svc.CreateSecurityTransaction(ctx, sell); err != nil {
		t.Fatalf("Create sell: %v", err)
	}

	if err := svc.RecalculateFIFO(ctx, 2024); err != nil {
		t.Fatalf("RecalculateFIFO() error: %v", err)
	}

	got, err := svc.GetSecurityTransaction(ctx, sell.ID)
	if err != nil {
		t.Fatalf("Get sell: %v", err)
	}

	// No limit in 2024 -- full gain exempt.
	expectedGain := domain.Amount(200000000 - 100000)
	if got.ExemptAmount != expectedGain {
		t.Errorf("ExemptAmount = %d, want %d (no limit in 2024)", got.ExemptAmount, expectedGain)
	}
}

func TestInvestmentIncomeService_RecalculateFIFO_InvalidYear(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	err := svc.RecalculateFIFO(ctx, 2000)
	if err == nil {
		t.Error("expected error for unsupported year")
	}
}

func TestInvestmentIncomeService_RecalculateFIFO_SellWithNoMatchingBuys(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	sell := testSecTx(2025, domain.AssetTypeStock, "ORPHAN", domain.TransactionTypeSell,
		time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), 10000, 500000)
	if err := svc.CreateSecurityTransaction(ctx, sell); err != nil {
		t.Fatalf("Create sell: %v", err)
	}

	if err := svc.RecalculateFIFO(ctx, 2025); err != nil {
		t.Fatalf("RecalculateFIFO() error: %v", err)
	}

	got, err := svc.GetSecurityTransaction(ctx, sell.ID)
	if err != nil {
		t.Fatalf("Get sell: %v", err)
	}

	if got.TimeTestExempt {
		t.Error("expected TimeTestExempt = false when no matching buys exist")
	}
	if got.CostBasis != 0 {
		t.Errorf("CostBasis = %d, want 0", got.CostBasis)
	}
}

// --- Year Summary ---

func TestInvestmentIncomeService_GetYearSummary(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	// Add capital income (needs_declaring=true for it to count in totals).
	entry := &domain.CapitalIncomeEntry{
		Year:           2025,
		Category:       domain.CapitalCategoryDividendForeign,
		IncomeDate:     time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		GrossAmount:    10000,
		WithheldTaxCZ:  1500,
		NeedsDeclaring: true,
	}
	if err := svc.CreateCapitalEntry(ctx, entry); err != nil {
		t.Fatalf("Create capital: %v", err)
	}

	buy := testSecTx(2025, domain.AssetTypeStock, "TSLA", domain.TransactionTypeBuy,
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), 10000, 500000)
	if err := svc.CreateSecurityTransaction(ctx, buy); err != nil {
		t.Fatalf("Create buy: %v", err)
	}

	sell := testSecTx(2025, domain.AssetTypeStock, "TSLA", domain.TransactionTypeSell,
		time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), 10000, 1000000)
	sell.Fees = 5000
	if err := svc.CreateSecurityTransaction(ctx, sell); err != nil {
		t.Fatalf("Create sell: %v", err)
	}

	if err := svc.RecalculateFIFO(ctx, 2025); err != nil {
		t.Fatalf("RecalculateFIFO: %v", err)
	}

	summary, err := svc.GetYearSummary(ctx, 2025)
	if err != nil {
		t.Fatalf("GetYearSummary() error: %v", err)
	}

	if summary.Year != 2025 {
		t.Errorf("Year = %d, want 2025", summary.Year)
	}
	if summary.CapitalIncomeGross != 10000 {
		t.Errorf("CapitalIncomeGross = %d, want 10000", summary.CapitalIncomeGross)
	}
	if summary.CapitalIncomeTax != 1500 {
		t.Errorf("CapitalIncomeTax = %d, want 1500", summary.CapitalIncomeTax)
	}
	if summary.CapitalIncomeNet != 8500 {
		t.Errorf("CapitalIncomeNet = %d, want 8500", summary.CapitalIncomeNet)
	}
	if summary.OtherIncomeGross <= 0 {
		t.Error("expected OtherIncomeGross > 0")
	}
}

func TestInvestmentIncomeService_GetYearSummary_Empty(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	summary, err := svc.GetYearSummary(ctx, 2025)
	if err != nil {
		t.Fatalf("GetYearSummary() error: %v", err)
	}

	if summary.Year != 2025 {
		t.Errorf("Year = %d, want 2025", summary.Year)
	}
	if summary.CapitalIncomeGross != 0 {
		t.Errorf("CapitalIncomeGross = %d, want 0", summary.CapitalIncomeGross)
	}
	if summary.OtherIncomeGross != 0 {
		t.Errorf("OtherIncomeGross = %d, want 0", summary.OtherIncomeGross)
	}
}

func TestInvestmentIncomeService_NetAmountCalculation(t *testing.T) {
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	entry := &domain.CapitalIncomeEntry{
		Year:               2025,
		Category:           domain.CapitalCategoryDividendForeign,
		IncomeDate:         time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		GrossAmount:        20000,
		WithheldTaxCZ:      3000,
		WithheldTaxForeign: 2000,
	}
	if err := svc.CreateCapitalEntry(ctx, entry); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if entry.NetAmount != 15000 {
		t.Errorf("NetAmount = %d, want 15000", entry.NetAmount)
	}
}
