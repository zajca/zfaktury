package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/zajca/zfaktury/internal/calc"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// InvestmentIncomeService provides business logic for investment income management.
type InvestmentIncomeService struct {
	capitalRepo  repository.CapitalIncomeRepo
	securityRepo repository.SecurityTransactionRepo
	audit        *AuditService
}

// NewInvestmentIncomeService creates a new InvestmentIncomeService.
func NewInvestmentIncomeService(
	capitalRepo repository.CapitalIncomeRepo,
	securityRepo repository.SecurityTransactionRepo,
	audit *AuditService,
) *InvestmentIncomeService {
	return &InvestmentIncomeService{
		capitalRepo:  capitalRepo,
		securityRepo: securityRepo,
		audit:        audit,
	}
}

// --- Capital Income CRUD (§8) ---

// CreateCapitalEntry validates and creates a new capital income entry.
func (s *InvestmentIncomeService) CreateCapitalEntry(ctx context.Context, entry *domain.CapitalIncomeEntry) error {
	if err := validateCapitalCategory(entry.Category); err != nil {
		return err
	}
	entry.NetAmount = entry.GrossAmount - entry.WithheldTaxCZ - entry.WithheldTaxForeign
	if err := s.capitalRepo.Create(ctx, entry); err != nil {
		return fmt.Errorf("creating capital entry: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "capital_income", entry.ID, "create", nil, entry)
	}
	return nil
}

// UpdateCapitalEntry validates and updates an existing capital income entry.
func (s *InvestmentIncomeService) UpdateCapitalEntry(ctx context.Context, entry *domain.CapitalIncomeEntry) error {
	if err := validateCapitalCategory(entry.Category); err != nil {
		return err
	}
	var existing *domain.CapitalIncomeEntry
	if s.audit != nil {
		var err error
		existing, err = s.capitalRepo.GetByID(ctx, entry.ID)
		if err != nil {
			return fmt.Errorf("fetching capital entry for audit: %w", err)
		}
	}
	entry.NetAmount = entry.GrossAmount - entry.WithheldTaxCZ - entry.WithheldTaxForeign
	if err := s.capitalRepo.Update(ctx, entry); err != nil {
		return fmt.Errorf("updating capital entry: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "capital_income", entry.ID, "update", existing, entry)
	}
	return nil
}

// DeleteCapitalEntry deletes a capital income entry by ID.
func (s *InvestmentIncomeService) DeleteCapitalEntry(ctx context.Context, id int64) error {
	if err := s.capitalRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting capital entry: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "capital_income", id, "delete", nil, nil)
	}
	return nil
}

// GetCapitalEntry retrieves a capital income entry by ID.
func (s *InvestmentIncomeService) GetCapitalEntry(ctx context.Context, id int64) (*domain.CapitalIncomeEntry, error) {
	entry, err := s.capitalRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching capital entry: %w", err)
	}
	return entry, nil
}

// ListCapitalEntries retrieves all capital income entries for a given year.
func (s *InvestmentIncomeService) ListCapitalEntries(ctx context.Context, year int) ([]domain.CapitalIncomeEntry, error) {
	entries, err := s.capitalRepo.ListByYear(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("listing capital entries: %w", err)
	}
	return entries, nil
}

// ComputeCapitalIncomeTotals returns aggregated gross, tax, and net amounts for capital income in a year.
func (s *InvestmentIncomeService) ComputeCapitalIncomeTotals(ctx context.Context, year int) (gross, tax, net domain.Amount, err error) {
	gross, tax, net, err = s.capitalRepo.SumByYear(ctx, year)
	if err != nil {
		err = fmt.Errorf("computing capital income totals: %w", err)
	}
	return
}

// --- Security Transaction CRUD (§10) ---

// CreateSecurityTransaction validates and creates a new security transaction.
func (s *InvestmentIncomeService) CreateSecurityTransaction(ctx context.Context, tx *domain.SecurityTransaction) error {
	if err := validateAssetType(tx.AssetType); err != nil {
		return err
	}
	if err := validateTransactionType(tx.TransactionType); err != nil {
		return err
	}
	if err := s.securityRepo.Create(ctx, tx); err != nil {
		return fmt.Errorf("creating security transaction: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "security_transaction", tx.ID, "create", nil, tx)
	}
	return nil
}

// UpdateSecurityTransaction validates and updates an existing security transaction.
func (s *InvestmentIncomeService) UpdateSecurityTransaction(ctx context.Context, tx *domain.SecurityTransaction) error {
	if err := validateAssetType(tx.AssetType); err != nil {
		return err
	}
	if err := validateTransactionType(tx.TransactionType); err != nil {
		return err
	}
	var existing *domain.SecurityTransaction
	if s.audit != nil {
		var err error
		existing, err = s.securityRepo.GetByID(ctx, tx.ID)
		if err != nil {
			return fmt.Errorf("fetching security transaction for audit: %w", err)
		}
	}
	if err := s.securityRepo.Update(ctx, tx); err != nil {
		return fmt.Errorf("updating security transaction: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "security_transaction", tx.ID, "update", existing, tx)
	}
	return nil
}

// DeleteSecurityTransaction deletes a security transaction by ID.
func (s *InvestmentIncomeService) DeleteSecurityTransaction(ctx context.Context, id int64) error {
	if err := s.securityRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting security transaction: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "security_transaction", id, "delete", nil, nil)
	}
	return nil
}

// GetSecurityTransaction retrieves a security transaction by ID.
func (s *InvestmentIncomeService) GetSecurityTransaction(ctx context.Context, id int64) (*domain.SecurityTransaction, error) {
	tx, err := s.securityRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching security transaction: %w", err)
	}
	return tx, nil
}

// ListSecurityTransactions retrieves all security transactions for a given year.
func (s *InvestmentIncomeService) ListSecurityTransactions(ctx context.Context, year int) ([]domain.SecurityTransaction, error) {
	txs, err := s.securityRepo.ListByYear(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("listing security transactions: %w", err)
	}
	return txs, nil
}

// --- FIFO Calculation ---

// assetGroupKey uniquely identifies a group of the same asset for FIFO matching.
type assetGroupKey struct {
	AssetName string
	AssetType string
}

// RecalculateFIFO recomputes FIFO cost basis, gains, and time test exemptions for all sells in a year.
func (s *InvestmentIncomeService) RecalculateFIFO(ctx context.Context, year int) error {
	constants, err := calc.GetTaxConstants(year)
	if err != nil {
		return fmt.Errorf("getting tax constants for FIFO: %w", err)
	}

	sells, err := s.securityRepo.ListSellsByYear(ctx, year)
	if err != nil {
		return fmt.Errorf("listing sells for FIFO: %w", err)
	}
	if len(sells) == 0 {
		return nil
	}

	// Group sells by (assetName, assetType).
	groups := make(map[assetGroupKey][]domain.SecurityTransaction)
	for _, sell := range sells {
		key := assetGroupKey{AssetName: sell.AssetName, AssetType: sell.AssetType}
		groups[key] = append(groups[key], sell)
	}

	// Track cumulative exempt amount across all groups for the year.
	var cumulativeExempt domain.Amount

	// Sort group keys for deterministic processing.
	keys := make([]assetGroupKey, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].AssetName != keys[j].AssetName {
			return keys[i].AssetName < keys[j].AssetName
		}
		return keys[i].AssetType < keys[j].AssetType
	})

	for _, key := range keys {
		groupSells := groups[key]

		// Sort sells by transaction date (FIFO order).
		sort.Slice(groupSells, func(i, j int) bool {
			return groupSells[i].TransactionDate.Before(groupSells[j].TransactionDate)
		})

		// Load all buys for this asset group.
		buys, err := s.securityRepo.ListBuysForFIFO(ctx, key.AssetName, key.AssetType)
		if err != nil {
			return fmt.Errorf("listing buys for FIFO asset %s/%s: %w", key.AssetName, key.AssetType, err)
		}

		// Track consumed quantity per buy (shared across all sells of this group).
		consumed := make(map[int64]int64) // buyID -> consumed quantity

		for _, sell := range groupSells {
			remainingQty := sell.Quantity
			var costBasis domain.Amount
			allBuysExempt := true
			timeTestCutoff := sell.TransactionDate.AddDate(-constants.TimeTestYears, 0, 0)

			for _, buy := range buys {
				if remainingQty <= 0 {
					break
				}

				availableQty := buy.Quantity - consumed[buy.ID]
				if availableQty <= 0 {
					continue
				}

				// Determine how much to consume from this buy.
				takeQty := availableQty
				if takeQty > remainingQty {
					takeQty = remainingQty
				}

				// Calculate proportional cost basis from this buy.
				// costBasis portion = buy.TotalAmount * (takeQty / buy.Quantity)
				buyCost := domain.Amount(int64(buy.TotalAmount) * takeQty / buy.Quantity)
				costBasis += buyCost

				// Check time test: buy must be before the cutoff date.
				if !buy.TransactionDate.Before(timeTestCutoff) {
					allBuysExempt = false
				}

				consumed[buy.ID] += takeQty
				remainingQty -= takeQty
			}

			// If we couldn't match all sell quantity to buys, mark as not exempt.
			if remainingQty > 0 {
				allBuysExempt = false
			}

			computedGain := sell.TotalAmount - sell.Fees - costBasis
			timeTestExempt := allBuysExempt && computedGain > 0

			var exemptAmount domain.Amount
			if timeTestExempt {
				exemptAmount = computedGain
				// Apply SecurityExemptionLimit if configured (> 0).
				if constants.SecurityExemptionLimit > 0 {
					remaining := constants.SecurityExemptionLimit - cumulativeExempt
					if remaining <= 0 {
						exemptAmount = 0
						timeTestExempt = false
					} else if exemptAmount > remaining {
						exemptAmount = remaining
					}
				}
				cumulativeExempt += exemptAmount
			}

			if err := s.securityRepo.UpdateFIFOResults(ctx, sell.ID, costBasis, computedGain, exemptAmount, timeTestExempt); err != nil {
				return fmt.Errorf("updating FIFO results for sell %d: %w", sell.ID, err)
			}
		}
	}

	return nil
}

// --- Year Summary ---

// GetYearSummary aggregates investment income data for a given year.
func (s *InvestmentIncomeService) GetYearSummary(ctx context.Context, year int) (*domain.InvestmentYearSummary, error) {
	capitalGross, capitalTax, capitalNet, err := s.ComputeCapitalIncomeTotals(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("computing capital income for year summary: %w", err)
	}

	sells, err := s.securityRepo.ListSellsByYear(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("listing sells for year summary: %w", err)
	}

	var otherGross, otherExpenses, otherExempt domain.Amount
	for _, sell := range sells {
		otherGross += sell.TotalAmount - sell.Fees
		otherExpenses += sell.CostBasis
		otherExempt += sell.ExemptAmount
	}
	otherNet := otherGross - otherExpenses - otherExempt

	return &domain.InvestmentYearSummary{
		Year:                year,
		CapitalIncomeGross:  capitalGross,
		CapitalIncomeTax:    capitalTax,
		CapitalIncomeNet:    capitalNet,
		OtherIncomeGross:    otherGross,
		OtherIncomeExpenses: otherExpenses,
		OtherIncomeExempt:   otherExempt,
		OtherIncomeNet:      otherNet,
	}, nil
}

// --- Validation helpers ---

func validateCapitalCategory(category string) error {
	switch category {
	case domain.CapitalCategoryDividendCZ, domain.CapitalCategoryDividendForeign, domain.CapitalCategoryInterest,
		domain.CapitalCategoryCoupon, domain.CapitalCategoryFundDist, domain.CapitalCategoryOther:
		return nil
	default:
		return fmt.Errorf("invalid capital income category %q: %w", category, domain.ErrInvalidInput)
	}
}

func validateAssetType(assetType string) error {
	switch assetType {
	case domain.AssetTypeStock, domain.AssetTypeETF, domain.AssetTypeBond, domain.AssetTypeFund,
		domain.AssetTypeCrypto, domain.AssetTypeOther:
		return nil
	default:
		return fmt.Errorf("invalid asset type %q: %w", assetType, domain.ErrInvalidInput)
	}
}

func validateTransactionType(txType string) error {
	switch txType {
	case domain.TransactionTypeBuy, domain.TransactionTypeSell:
		return nil
	default:
		return fmt.Errorf("invalid transaction type %q: %w", txType, domain.ErrInvalidInput)
	}
}
