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
	svc := NewInvestmentIncomeService(capitalRepo, securityRepo, nil)
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

	// 2025 limit: 40 000 000 CZK = 4_000_000_000 halere of gross time-test income.
	// Two sells with 3e9 + 3e9 halere of income exceed it, so 4/6 of each gain is exempt.
	for _, name := range []string{"BIG1", "BIG2"} {
		buy := testSecTx(2025, domain.AssetTypeStock, name, domain.TransactionTypeBuy,
			time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), 10000, 1_000_000_000)
		if err := svc.CreateSecurityTransaction(ctx, buy); err != nil {
			t.Fatalf("Create buy %s: %v", name, err)
		}
	}
	sell1 := testSecTx(2025, domain.AssetTypeStock, "BIG1", domain.TransactionTypeSell,
		time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), 10000, 3_000_000_000)
	sell2 := testSecTx(2025, domain.AssetTypeStock, "BIG2", domain.TransactionTypeSell,
		time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC), 10000, 3_000_000_000)
	for _, s := range []*domain.SecurityTransaction{sell1, sell2} {
		if err := svc.CreateSecurityTransaction(ctx, s); err != nil {
			t.Fatalf("Create sell: %v", err)
		}
	}

	if err := svc.RecalculateFIFO(ctx, 2025); err != nil {
		t.Fatalf("RecalculateFIFO() error: %v", err)
	}

	for _, id := range []int64{sell1.ID, sell2.ID} {
		got, err := svc.GetSecurityTransaction(ctx, id)
		if err != nil {
			t.Fatalf("Get sell: %v", err)
		}
		// Gain = 3e9 - 1e9 = 2e9; exempt share = 4e9 / 6e9 -> 2e9 * 2/3 = 1_333_333_333.
		if got.ComputedGain != 2_000_000_000 {
			t.Errorf("ComputedGain = %d, want 2000000000", got.ComputedGain)
		}
		if got.ExemptAmount != 1_333_333_333 {
			t.Errorf("ExemptAmount = %d, want 1333333333", got.ExemptAmount)
		}
		if !got.TimeTestExempt {
			t.Error("expected TimeTestExempt = true for the partially exempt sell")
		}
	}
}

// recalcSingleSell stores one time-test-satisfying sale and returns its FIFO result.
func recalcSingleSell(t *testing.T, year int, assetType string, cost, proceeds domain.Amount) *domain.SecurityTransaction {
	t.Helper()
	svc, _, _ := newInvestmentIncomeService(t)
	ctx := context.Background()

	buy := testSecTx(year, assetType, "LIM", domain.TransactionTypeBuy,
		time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC), 100, cost)
	if err := svc.CreateSecurityTransaction(ctx, buy); err != nil {
		t.Fatalf("Create buy: %v", err)
	}
	sell := testSecTx(year, assetType, "LIM", domain.TransactionTypeSell,
		time.Date(year, 6, 1, 0, 0, 0, 0, time.UTC), 100, proceeds)
	if err := svc.CreateSecurityTransaction(ctx, sell); err != nil {
		t.Fatalf("Create sell: %v", err)
	}
	if err := svc.RecalculateFIFO(ctx, year); err != nil {
		t.Fatalf("RecalculateFIFO() error: %v", err)
	}
	got, err := svc.GetSecurityTransaction(ctx, sell.ID)
	if err != nil {
		t.Fatalf("Get sell: %v", err)
	}
	return got
}

func TestInvestmentIncomeService_RecalculateFIFO_ExemptionLimitBoundary2025(t *testing.T) {
	const limit = domain.Amount(4_000_000_000) // 40 000 000 CZK in halere

	tests := []struct {
		name       string
		proceeds   domain.Amount
		wantExempt domain.Amount
	}{
		{"below limit is fully exempt", limit - 100, (limit - 100) - 1_000_000_000},
		{"at limit is fully exempt", limit, limit - 1_000_000_000},
		// Income 8e9 is twice the limit: half of the 7e9 gain stays exempt.
		{"above limit exempts the proportional share", 2 * limit, 3_500_000_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recalcSingleSell(t, 2025, domain.AssetTypeStock, 1_000_000_000, tt.proceeds)
			if got.ExemptAmount != tt.wantExempt {
				t.Errorf("ExemptAmount = %d, want %d", got.ExemptAmount, tt.wantExempt)
			}
			if !got.TimeTestExempt {
				t.Error("expected TimeTestExempt = true")
			}
		})
	}
}

// From 2026 the limit stays only for crypto-assets (MF press release).
func TestInvestmentIncomeService_RecalculateFIFO_ExemptionLimit2026(t *testing.T) {
	stock := recalcSingleSell(t, 2026, domain.AssetTypeStock, 1_000_000_000, 8_000_000_000)
	if stock.ExemptAmount != stock.ComputedGain {
		t.Errorf("stock ExemptAmount = %d, want full gain %d", stock.ExemptAmount, stock.ComputedGain)
	}

	crypto := recalcSingleSell(t, 2026, domain.AssetTypeCrypto, 1_000_000_000, 8_000_000_000)
	if crypto.ExemptAmount != 3_500_000_000 {
		t.Errorf("crypto ExemptAmount = %d, want 3500000000", crypto.ExemptAmount)
	}
}

func TestProportionalAmount_NoOverflow(t *testing.T) {
	// 9e12 halere gain x 4e9 limit exceeds int64 as a plain product.
	got := proportionalAmount(9_000_000_000_000, 4_000_000_000, 8_000_000_000)
	if got != 4_500_000_000_000 {
		t.Errorf("proportionalAmount = %d, want 4500000000000", got)
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

// The crypto time-test exemption (§ 4 odst. 1 písm. zk), zákon 32/2025 Sb.)
// applies only to sales on or after 15 Feb 2025, no matter when the coins were
// bought; earlier crypto sales stay taxable even when held more than 3 years.
// Securities are unaffected by the cutoff.
func TestInvestmentIncomeService_RecalculateFIFO_CryptoTimeTestCutoff(t *testing.T) {
	utc := func(y int, m time.Month, d int) time.Time {
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
	tests := []struct {
		name       string
		year       int
		assetType  string
		buyDate    time.Time
		sellDate   time.Time
		wantExempt bool
	}{
		{"crypto bought 2021 sold 2025-02-14 is taxable", 2025, domain.AssetTypeCrypto, utc(2021, 1, 1), utc(2025, 2, 14), false},
		{"crypto bought 2021 sold 2025-02-15 is exempt", 2025, domain.AssetTypeCrypto, utc(2021, 1, 1), utc(2025, 2, 15), true},
		{"crypto bought 2021 sold 2025-01-31 is taxable", 2025, domain.AssetTypeCrypto, utc(2021, 1, 1), utc(2025, 1, 31), false},
		{"crypto bought 2020 sold 2024-06-01 is taxable", 2024, domain.AssetTypeCrypto, utc(2020, 1, 1), utc(2024, 6, 1), false},
		{"security bought 2021 sold 2025-02-14 is exempt", 2025, domain.AssetTypeStock, utc(2021, 1, 1), utc(2025, 2, 14), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _ := newInvestmentIncomeService(t)
			ctx := context.Background()

			buy := testSecTx(tt.year, tt.assetType, "ASSET", domain.TransactionTypeBuy, tt.buyDate, 100, 100000)
			if err := svc.CreateSecurityTransaction(ctx, buy); err != nil {
				t.Fatalf("Create buy: %v", err)
			}
			// Proceeds 300 000 halere against cost 100 000 halere -> gain 200 000.
			sell := testSecTx(tt.year, tt.assetType, "ASSET", domain.TransactionTypeSell, tt.sellDate, 100, 300000)
			if err := svc.CreateSecurityTransaction(ctx, sell); err != nil {
				t.Fatalf("Create sell: %v", err)
			}

			if err := svc.RecalculateFIFO(ctx, tt.year); err != nil {
				t.Fatalf("RecalculateFIFO() error: %v", err)
			}

			got, err := svc.GetSecurityTransaction(ctx, sell.ID)
			if err != nil {
				t.Fatalf("Get sell: %v", err)
			}
			if got.TimeTestExempt != tt.wantExempt {
				t.Errorf("TimeTestExempt = %v, want %v", got.TimeTestExempt, tt.wantExempt)
			}
			if tt.wantExempt {
				if got.ExemptAmount != got.ComputedGain {
					t.Errorf("ExemptAmount = %d, want full gain %d", got.ExemptAmount, got.ComputedGain)
				}
			} else if got.ExemptAmount != 0 {
				t.Errorf("ExemptAmount = %d, want 0", got.ExemptAmount)
			}
		})
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
