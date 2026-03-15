# RFC-012: Isolate Financial Calculations and XML Generation with Full Coverage

**Status:** Draft
**Date:** 2026-03-14

## Summary

Extract all pure financial calculations into a new `internal/calc/` package with zero I/O dependencies. Achieve 100% test coverage on `calc/`, `vatxml/`, `annualtaxxml/`, `isdoc/`. Add golden file (snapshot) tests for all 6 XML generators.

## Background

ZFaktury handles money (invoicing, VAT, income tax, social/health insurance) and generates XML for Czech government systems (DPHDP3, DPHKH1, DPHSHV, DPFDP5, CSSZ OSVC, ISDOC). These are the highest-risk areas of the application -- bugs here mean incorrect tax filings or wrong invoices.

Financial calculation logic is currently embedded in service methods that mix DB I/O (fetching invoices, expenses, persisting results) with pure math. This makes it impossible to test the math in isolation without a database. XML generators have assertion-based tests but no snapshot tests to catch unintended output changes.

### Requirements

1. Extract all pure financial calculations into `internal/calc/` with zero I/O dependencies
2. Achieve 100% test coverage on `calc/`, `vatxml/`, `annualtaxxml/`, `isdoc/`
3. Add golden file (snapshot) tests for all 6 XML generators
4. No business logic or external behavior changes -- pure structural refactor + test hardening

### Non-goals

- Changing any business logic or external behavior
- Refactoring XML generators themselves (only adding tests)
- Modifying the service layer API surface

## Current State Analysis

### Financial Calculations (mixed with DB I/O)

| Calculation | Location | Pure Logic Lines | DB I/O Lines |
|---|---|---|---|
| VAT return | `service/vat_return_svc.go:167-271` | ~60 lines of VAT grouping | ~40 lines of DB fetch |
| Income tax | `service/income_tax_return_svc.go:179-283` | ~80 lines of progressive tax | ~30 lines of DB fetch |
| Social insurance | `service/social_insurance_svc.go:182-229` | ~40 lines of insurance calc | ~20 lines of DB fetch |
| Health insurance | `service/health_insurance_svc.go:174-213` | ~35 lines (identical formula) | ~20 lines of DB fetch |
| Tax credits | `service/tax_credits_svc.go:290-421` | ~50 lines of credit computation | ~30 lines of DB fetch |
| Annual base | `service/annual_tax_base.go:45-105` | ~40 lines of filtering/summing | ~20 lines of DB fetch |
| Tax constants | `service/annual_tax_constants.go` | 145 lines (already pure) | 0 |
| Invoice totals | `domain/invoice.go:107-125` | 18 lines (already pure) | 0 |
| Expense totals | `domain/expense.go:56-90` | 34 lines (already pure) | 0 |
| Amount type | `domain/money.go` | 72 lines (already pure) | 0 |

### XML Generators (already isolated, need snapshot tests)

| Generator | Package | Lines | Test Coverage | Has Snapshot Tests |
|---|---|---|---|---|
| VAT Return (DPHDP3) | `vatxml/vat_return_gen.go` | 132 | ~91% | No |
| Control Statement (DPHKH1) | `vatxml/control_statement_gen.go` | 162 | ~91% | No |
| VIES Summary (DPHSHV) | `vatxml/vies_gen.go` | 84 | ~91% | No |
| Income Tax (DPFDP5) | `annualtaxxml/income_tax_gen.go` | 82 | ~20% | No |
| Social Insurance (CSSZ) | `annualtaxxml/social_insurance_gen.go` | 141 | ~20% | No |
| ISDOC Invoice | `isdoc/generator.go` | 350 | ~98% | No |

### Determinism Issues for Snapshot Tests

- `vatxml/vat_return_gen.go:66` -- uses `time.Now().Format("02.01.2006")` for `DPoddp` field
- Must be injectable for deterministic tests

## Design

### New Package: `internal/calc/`

Pure calculation functions. **Dependencies: only `domain.Amount`**. No context, no repos, no I/O.

Each function takes a lightweight input struct and returns a result struct. Services become thin orchestrators: **fetch -> map to input -> calc -> map result back -> persist**.

```
                  ┌─────────────────────────────────────────────┐
                  │              Service Layer                   │
                  │  (orchestration: fetch -> calc -> persist)   │
                  └────────┬──────────────────┬─────────────────┘
                           │                  │
                   ┌───────▼───────┐  ┌───────▼───────┐
                   │  Repository   │  │  internal/calc │
                   │  (DB I/O)     │  │  (pure math)   │
                   └───────────────┘  └───────────────┘
```

### Calc Package Structure

```
internal/calc/
├── constants.go          # TaxYearConstants (moved from service)
├── constants_test.go
├── vat.go                # CalculateVATReturn()
├── vat_test.go
├── income_tax.go         # CalculateIncomeTax()
├── income_tax_test.go
├── insurance.go          # CalculateInsurance(), ResolveUsedExpenses()
├── insurance_test.go
├── credits.go            # ComputeSpouseCredit(), ComputePersonalCredits(), ComputeChildBenefit(), ComputeDeductions()
├── credits_test.go
├── annual_base.go        # CalculateAnnualTotals()
└── annual_base_test.go
```

### Golden File Infrastructure

Shared helper in `internal/testutil/golden.go`:

```go
package testutil

var Update = flag.Bool("update", false, "update golden files")

func AssertGolden(t *testing.T, goldenPath string, actual []byte) {
    t.Helper()
    if *Update {
        os.MkdirAll(filepath.Dir(goldenPath), 0o755)
        os.WriteFile(goldenPath, actual, 0o644)
        return
    }
    expected, err := os.ReadFile(goldenPath)
    if err != nil {
        t.Fatalf("golden file %s not found (run with -update): %v", goldenPath, err)
    }
    if !bytes.Equal(expected, actual) {
        t.Errorf("output differs from golden file %s\n%s", goldenPath, unifiedDiff(expected, actual))
    }
}
```

Usage: `go test ./internal/vatxml/ -update` to regenerate golden files.

Golden files live in `<package>/testdata/*.golden.xml`.

### API Design

#### VAT Return Calculation

```go
type VATInvoiceInput struct {
    Type  string // domain.InvoiceTypeCreditNote, etc.
    Items []VATItemInput
}
type VATItemInput struct {
    Base           domain.Amount // pre-computed: quantity * unitPrice / 100
    VATAmount      domain.Amount
    VATRatePercent int
}
type VATExpenseInput struct {
    Amount          domain.Amount
    VATAmount       domain.Amount
    VATRatePercent  int
    BusinessPercent int // 0 means 100
}
type VATResult struct {
    OutputVATBase21   domain.Amount
    OutputVATAmount21 domain.Amount
    OutputVATBase12   domain.Amount
    OutputVATAmount12 domain.Amount
    InputVATBase21    domain.Amount
    InputVATAmount21  domain.Amount
    InputVATBase12    domain.Amount
    InputVATAmount12  domain.Amount
    TotalOutputVAT    domain.Amount
    TotalInputVAT     domain.Amount
    NetVAT            domain.Amount
}
func CalculateVATReturn(invoices []VATInvoiceInput, expenses []VATExpenseInput) VATResult
```

#### Income Tax Calculation

```go
type IncomeTaxInput struct {
    TotalRevenue     domain.Amount
    ActualExpenses   domain.Amount
    FlatRatePercent  int
    Constants        TaxYearConstants
    SpouseCredit     domain.Amount
    DisabilityCredit domain.Amount
    StudentCredit    domain.Amount
    ChildBenefit     domain.Amount
    TotalDeductions  domain.Amount
    Prepayments      domain.Amount
    CapitalIncomeNet domain.Amount
    OtherIncomeNet   domain.Amount
}
type IncomeTaxResult struct {
    FlatRateAmount  domain.Amount
    UsedExpenses    domain.Amount
    TaxBase         domain.Amount
    TaxBaseRounded  domain.Amount // rounded down to 100 CZK
    TaxAt15         domain.Amount
    TaxAt23         domain.Amount
    TotalTax        domain.Amount
    CreditBasic     domain.Amount
    TotalCredits    domain.Amount
    TaxAfterCredits domain.Amount
    TaxAfterBenefit domain.Amount
    TaxDue          domain.Amount
}
func CalculateIncomeTax(input IncomeTaxInput) IncomeTaxResult
```

#### Insurance Calculation

```go
type InsuranceInput struct {
    Revenue        domain.Amount
    UsedExpenses   domain.Amount
    MinMonthlyBase domain.Amount // constants.SocialMinMonthly or HealthMinMonthly
    RatePermille   int           // 292 (social) or 135 (health)
    Prepayments    domain.Amount
}
type InsuranceResult struct {
    TaxBase             domain.Amount
    AssessmentBase      domain.Amount
    MinAssessmentBase   domain.Amount
    FinalAssessmentBase domain.Amount
    TotalInsurance      domain.Amount
    Difference          domain.Amount
    NewMonthlyPrepay    domain.Amount
}
func CalculateInsurance(input InsuranceInput) InsuranceResult
```

#### Shared Expense Resolution

```go
func ResolveUsedExpenses(revenue, actualExpenses domain.Amount, flatRatePercent int, caps map[int]domain.Amount) domain.Amount
```

Used by income tax, social insurance, and health insurance (all 3 do the same flat-rate-vs-actual logic).

#### Credits and Deductions

```go
func ComputeSpouseCredit(spouseIncome domain.Amount, monthsClaimed int, spouseZTP bool, constants TaxYearConstants) domain.Amount
func ComputePersonalCredits(disabilityLevel int, isStudent bool, studentMonths int, constants TaxYearConstants) (disability, student domain.Amount)
func ComputeChildBenefit(children []ChildCreditInput, constants TaxYearConstants) domain.Amount

type DeductionInput struct {
    Category      string
    ClaimedAmount domain.Amount
}
type DeductionResult struct {
    AllowedAmounts []domain.Amount
    TotalAllowed   domain.Amount
}
func ComputeDeductions(deductions []DeductionInput, taxBase domain.Amount, constants TaxYearConstants) DeductionResult
```

#### Annual Base Aggregation

```go
type InvoiceForBase struct {
    ID             int64
    Type           string
    Status         string
    DeliveryDate   time.Time
    IssueDate      time.Time
    SubtotalAmount domain.Amount
}
type ExpenseForBase struct {
    ID              int64
    IssueDate       time.Time
    Amount          domain.Amount
    VATAmount       domain.Amount
    BusinessPercent int
    TaxReviewed     bool
}
type AnnualBaseResult struct {
    Revenue    domain.Amount
    Expenses   domain.Amount
    InvoiceIDs []int64
    ExpenseIDs []int64
}
func CalculateAnnualTotals(invoices []InvoiceForBase, expenses []ExpenseForBase, year int) AnnualBaseResult
```

### Service Refactoring Pattern

Before (mixed):
```go
func (s *Service) Recalculate(ctx context.Context, id int64) {
    invoices := s.repo.List(ctx, ...)     // DB I/O
    for _, inv := range invoices {         // pure math mixed in
        result += inv.Amount * rate
    }
    s.repo.Update(ctx, result)             // DB I/O
}
```

After (separated):
```go
func (s *Service) Recalculate(ctx context.Context, id int64) {
    invoices := s.repo.List(ctx, ...)                        // DB I/O
    inputs := mapToCalcInputs(invoices)                      // mapping
    result := calc.CalculateVATReturn(inputs.invs, inputs.exps) // pure math
    mapResultToEntity(vr, result)                             // mapping
    s.repo.Update(ctx, vr)                                   // DB I/O
}
```

## Task List

### Phase 1: Foundation -- `internal/calc/` package

#### 1.1 Move tax constants to calc package
- [ ] Create `internal/calc/constants.go` -- move `TaxYearConstants` struct, `taxConstantsDB` map, `GetTaxConstants()` from `service/annual_tax_constants.go`
- [ ] Delete `internal/service/annual_tax_constants.go`
- [ ] Update all imports in service package (`income_tax_return_svc.go`, `social_insurance_svc.go`, `health_insurance_svc.go`, `tax_credits_svc.go`)
- [ ] Create `internal/calc/constants_test.go` -- test known years (2024, 2025, 2026) and unknown years (1999, 9999)
- [ ] Verify: `CGO_ENABLED=0 go build ./...` and `go test ./...` pass

#### 1.2 Extract VAT return calculation
- [ ] Create `internal/calc/vat.go` with `CalculateVATReturn(invoices, expenses) VATResult`
- [ ] Create `internal/calc/vat_test.go` -- table-driven tests covering: zero inputs, single 21%/12% invoice, mixed rates, credit notes, 0% VAT (excluded), expenses with business percent 0/50/100, large amounts, invariant checks (TotalOutputVAT = sum, NetVAT = output - input)
- [ ] Refactor `service/vat_return_svc.go:Recalculate` to use `calc.CalculateVATReturn()`
- [ ] Verify: existing tests pass unchanged

#### 1.3 Extract income tax calculation
- [ ] Create `internal/calc/income_tax.go` with `CalculateIncomeTax(IncomeTaxInput) IncomeTaxResult`
- [ ] Create `internal/calc/income_tax_test.go` -- table-driven tests covering: flat rate with/without cap, actual expenses, tax base clamped to 0, rounding to 100 CZK, progressive tax (below/above/at threshold), credits exceeding tax, child benefit (negative allowed), prepayment refund, full realistic 2025 scenario
- [ ] Refactor `service/income_tax_return_svc.go:Recalculate` to use `calc.CalculateIncomeTax()`
- [ ] Verify: existing service tests pass

#### 1.4 Extract insurance calculation
- [ ] Create `internal/calc/insurance.go` with `CalculateInsurance(InsuranceInput) InsuranceResult` and `ResolveUsedExpenses()`
- [ ] Create `internal/calc/insurance_test.go` -- table-driven: revenue above/below minimum, social rate 292 and health rate 135 scenarios, monthly prepayment rounding, negative difference
- [ ] Create `internal/calc/expenses_test.go` -- tests for `ResolveUsedExpenses`: flat rate within/at cap, different percentages (60/80), flat rate 0% returns actual
- [ ] Refactor `service/social_insurance_svc.go` and `service/health_insurance_svc.go` to use `calc.CalculateInsurance()` + `calc.ResolveUsedExpenses()`
- [ ] Verify: existing service tests pass

#### 1.5 Extract tax credits calculation
- [ ] Create `internal/calc/credits.go` with `ComputeSpouseCredit()`, `ComputePersonalCredits()`, `ComputeChildBenefit()`, `ComputeDeductions()`
- [ ] Create `internal/calc/credits_test.go` -- spouse (income below/at/above limit, ZTP, proportional months), personal (disability levels 0-3, student months), child benefit (orders 1-3, ZTP, proportional), deductions (single/multiple per category, cap sharing, donation 15% of base, empty list)
- [ ] Refactor `service/tax_credits_svc.go` compute methods to use calc functions
- [ ] Verify: existing service tests pass

#### 1.6 Extract annual base aggregation
- [ ] Create `internal/calc/annual_base.go` with `CalculateAnnualTotals(invoices, expenses, year) AnnualBaseResult`
- [ ] Create `internal/calc/annual_base_test.go` -- regular/proforma/draft/credit note invoices, DeliveryDate vs IssueDate, tax-reviewed vs not, business percent 0/50/100, mixed scenario with ID verification
- [ ] Refactor `service/annual_tax_base.go:CalculateAnnualBase` to use `calc.CalculateAnnualTotals()`
- [ ] Verify: existing tests pass

### Phase 2: Golden File Infrastructure

#### 2.1 Create golden file test helper
- [ ] Create `internal/testutil/golden.go` with `AssertGolden(t, goldenPath, actual)` and `-update` flag support
- [ ] Verify: builds and is importable

#### 2.2 Fix determinism in XML generators
- [ ] `vatxml/vat_return_gen.go:66` -- add `SubmissionDate time.Time` to `TaxpayerInfo` struct, use it instead of `time.Now()` (zero value falls back to `time.Now()`)
- [ ] Check all other generators for `time.Now()` calls -- fix any found
- [ ] Verify: existing tests pass

### Phase 3: Snapshot Tests for XML Generators

#### 3.1 VAT Return XML (`vatxml/`)
- [ ] `TestVATReturnGenerator_Generate_Golden_Regular` -- standard Q1 2025 return with 21%+12% output VAT, 21% input VAT
- [ ] `TestVATReturnGenerator_Generate_Golden_Zero` -- all zero amounts
- [ ] `TestVATReturnGenerator_Generate_Golden_Negative` -- refund scenario
- [ ] `TestVATReturnGenerator_Generate_Golden_Corrective` -- corrective filing
- [ ] Generate golden files: `cd internal/vatxml && go test -run Golden -update`

#### 3.2 Control Statement XML (`vatxml/`)
- [ ] `TestControlStatement_Golden_A4A5` -- A4 lines (>10k) + A5 aggregated
- [ ] `TestControlStatement_Golden_B2B3` -- B2/B3 input sections
- [ ] `TestControlStatement_Golden_Corrective` -- corrective filing

#### 3.3 VIES Summary XML (`vatxml/`)
- [ ] `TestVIESSummary_Golden_Regular` -- standard quarterly VIES with 2 partner countries
- [ ] `TestVIESSummary_Golden_Empty` -- no lines

#### 3.4 Income Tax Return XML (`annualtaxxml/`)
- [ ] `TestIncomeTaxXML_Golden_Full` -- full return: revenue, expenses, progressive tax, credits, child benefit, prepayments
- [ ] `TestIncomeTaxXML_Golden_Minimal` -- minimal return: revenue only, no credits

#### 3.5 Social Insurance XML (`annualtaxxml/`)
- [ ] `TestSocialInsuranceXML_Golden_Regular` -- standard overview with flat rate flag
- [ ] `TestSocialInsuranceXML_Golden_ActualExpenses` -- no flat rate

#### 3.6 ISDOC Invoice XML (`isdoc/`)
- [ ] `TestISDOC_Golden_Regular` -- standard invoice with 21% VAT, bank payment
- [ ] `TestISDOC_Golden_CreditNote` -- credit note (document type 2)
- [ ] `TestISDOC_Golden_ForeignCurrency` -- EUR invoice with exchange rate
- [ ] `TestISDOC_Golden_CashPayment` -- cash payment (payment means 10)

### Phase 4: Fill Coverage Gaps to 100%

#### 4.1 `vatxml/` coverage gap (~9%)
- [ ] Run coverage analysis, add tests for uncovered branches (FilingTypeCode edge cases, `formatDPPD` with invalid date, `stripCountryPrefix` edge cases, `ToWholeCZK` negative/zero)

#### 4.2 `annualtaxxml/` coverage gap (~80%)
- [ ] Full test suite for `GenerateIncomeTaxXML` -- nil input, valid roundtrip, all VetaD/VetaP fields
- [ ] Full test suite for `GenerateSocialInsuranceXML` -- nil input, valid XML, CSSZ namespace, flat rate flag, `CSSZFilingTypeCode` variants
- [ ] Test `ToWholeCZK` if not covered

#### 4.3 `isdoc/` coverage gap (~2%)
- [ ] Run coverage analysis, add targeted tests for uncovered lines

#### 4.4 `calc/` coverage (new package)
- [ ] Verify 100% coverage, add targeted tests if any gaps

### Phase 5: Coverage Enforcement

#### 5.1 Makefile target
- [ ] Add `coverage-critical` target to `Makefile`:
  ```makefile
  .PHONY: coverage-critical
  coverage-critical:
  	@echo "Checking critical package coverage..."
  	CGO_ENABLED=0 go test ./internal/calc/ -coverprofile=/tmp/calc.cov -count=1
  	@go tool cover -func=/tmp/calc.cov | tail -1
  	CGO_ENABLED=0 go test ./internal/vatxml/ -coverprofile=/tmp/vatxml.cov -count=1
  	@go tool cover -func=/tmp/vatxml.cov | tail -1
  	CGO_ENABLED=0 go test ./internal/annualtaxxml/ -coverprofile=/tmp/annualtaxxml.cov -count=1
  	@go tool cover -func=/tmp/annualtaxxml.cov | tail -1
  	CGO_ENABLED=0 go test ./internal/isdoc/ -coverprofile=/tmp/isdoc.cov -count=1
  	@go tool cover -func=/tmp/isdoc.cov | tail -1
  ```

### Phase 6: Final Verification

- [ ] `CGO_ENABLED=0 go build ./...` -- compiles cleanly
- [ ] `CGO_ENABLED=0 go test ./...` -- all tests pass (no regression)
- [ ] `CGO_ENABLED=0 go test ./internal/calc/ -cover` -- 100%
- [ ] `CGO_ENABLED=0 go test ./internal/vatxml/ -cover` -- 100%
- [ ] `CGO_ENABLED=0 go test ./internal/annualtaxxml/ -cover` -- 100%
- [ ] `CGO_ENABLED=0 go test ./internal/isdoc/ -cover` -- 100%
- [ ] `go test ./internal/vatxml/ -run Golden` -- all golden files match
- [ ] `go test ./internal/annualtaxxml/ -run Golden` -- all golden files match
- [ ] `go test ./internal/isdoc/ -run Golden` -- all golden files match
- [ ] `go vet ./...` -- no issues
- [ ] Review: no business logic changed, only structural refactor

## Files Changed Summary

### New files

| File | Purpose |
|---|---|
| `internal/calc/constants.go` | Tax year constants (moved from service) |
| `internal/calc/constants_test.go` | Constants lookup tests |
| `internal/calc/vat.go` | Pure VAT return calculation |
| `internal/calc/vat_test.go` | VAT calculation tests |
| `internal/calc/income_tax.go` | Pure income tax calculation |
| `internal/calc/income_tax_test.go` | Income tax tests |
| `internal/calc/insurance.go` | Pure social/health insurance calculation |
| `internal/calc/insurance_test.go` | Insurance tests |
| `internal/calc/expenses_test.go` | ResolveUsedExpenses tests |
| `internal/calc/credits.go` | Pure tax credits/deductions calculation |
| `internal/calc/credits_test.go` | Credits tests |
| `internal/calc/annual_base.go` | Pure annual revenue/expense aggregation |
| `internal/calc/annual_base_test.go` | Annual base tests |
| `internal/testutil/golden.go` | Golden file comparison helper |
| `internal/vatxml/testdata/*.golden.xml` | ~8 golden files |
| `internal/annualtaxxml/testdata/*.golden.xml` | ~4 golden files |
| `internal/isdoc/testdata/*.golden.xml` | ~4 golden files |

### Modified files

| File | Change |
|---|---|
| `service/annual_tax_constants.go` | **Deleted** (moved to calc) |
| `service/vat_return_svc.go` | Recalculate uses `calc.CalculateVATReturn()` |
| `service/income_tax_return_svc.go` | Recalculate uses `calc.CalculateIncomeTax()` |
| `service/social_insurance_svc.go` | Recalculate uses `calc.CalculateInsurance()` |
| `service/health_insurance_svc.go` | Recalculate uses `calc.CalculateInsurance()` |
| `service/tax_credits_svc.go` | Compute* methods use `calc.Compute*()` |
| `service/annual_tax_base.go` | Uses `calc.CalculateAnnualTotals()` |
| `vatxml/vat_return_gen.go` | SubmissionDate field for determinism |
| `vatxml/*_test.go` | Golden file snapshot tests added |
| `annualtaxxml/*_test.go` | Full test suites + golden file tests |
| `isdoc/generator_test.go` | Golden file snapshot tests added |
| `Makefile` | `coverage-critical` target |

### Unchanged (already pure, already tested)

| File | Why unchanged |
|---|---|
| `domain/money.go` | Already pure, already tested |
| `domain/invoice.go:CalculateTotals` | Already pure, already tested |
| `domain/expense.go:CalculateTotals` | Already pure, already tested |
| `pdf/invoice_pdf.go` | Amount formatting only, uses domain.Amount methods |
| `pdf/qr_payment.go` | Uses ToCZK() conversion only |

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Subtle math difference during extraction | Run existing integration tests after each extraction step; no logic changes allowed |
| Golden file drift on unrelated changes | Golden files regenerated with `-update` flag; CI catches unexpected drift |
| `domain.Amount.Multiply` float rounding | Existing behavior preserved; calc functions use same Multiply calls |
| Import cycles calc <-> domain | calc depends only on domain.Amount; no reverse dependency |
