package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

// TestCrossCompanyLeakDetection exhaustively asserts that every per-company
// repository's Get and List methods refuse to surface rows belonging to a
// different company. Populated incrementally across Phase 3 tasks as each
// repo gains the companyID parameter:
//
//   - T20: Contact, Category, Sequence (this file's initial coverage)
//   - T21: Invoice, Expense + child docs/items, recurring invoices/expenses,
//     status history, payment reminders
//   - T22: Settings, VAT (return + control statement + VIES), income tax,
//     social/health insurance, tax year settings, tax prepayments,
//     spouse/child/personal credits, tax deductions + documents,
//     investment documents, capital income, security transactions,
//     fakturoid import log
//
// Each leakCase seeds an entity under company id=1 and asserts that
// company id=2 cannot see it via GetByID (ErrNotFound) or List (empty).
type leakCase struct {
	name string
	// seed inserts a representative entity for companyID and returns its ID.
	seed func(t *testing.T, db *sql.DB, companyID int64) int64
	// getByOtherCompany attempts GetByID with the wrong company; must
	// return domain.ErrNotFound.
	getByOtherCompany func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error
	// listByOtherCompany returns the number of rows the wrong company can
	// see via List; must be 0.
	listByOtherCompany func(t *testing.T, db *sql.DB, wrongCompanyID int64) int
}

// setupLeakDetectorDB returns a fresh in-memory test DB with the default
// company (id=1) plus a second company (id=2) so cross-company queries have
// a valid second tenant to target.
func setupLeakDetectorDB(t *testing.T) *sql.DB {
	t.Helper()
	db := testutil.NewTestDB(t)
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`
		INSERT INTO companies (id, name, legal_name, ico, vat_registered, created_at, updated_at)
		VALUES (2, 'Other Company', 'Other Company', '99999999', 0, ?, ?)`, now, now); err != nil {
		t.Fatalf("seeding second company: %v", err)
	}
	return db
}

var leakCases = []leakCase{
	{
		name: "ContactRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			c := testutil.SeedContact(t, db, companyID, &domain.Contact{
				Name: "Leak-test Contact",
				ICO:  "11223344",
			})
			return c.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewContactRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewContactRepository(db)
			_, total, err := repo.List(context.Background(), wrongCompanyID, domain.ContactFilter{})
			if err != nil {
				t.Fatalf("ContactRepository.List(other company) error: %v", err)
			}
			return total
		},
	},
	{
		name: "CategoryRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			return testutil.SeedCategory(t, db, companyID, "leak_test_cat",
				"Leak-test", "Leak test")
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewCategoryRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewCategoryRepository(db)
			// Seed-default categories live under company 1 only (the
			// migration backfill), so company 2 sees only what tests put
			// there. The leak-test category was seeded into company 1 in
			// the seed() callback, so company 2's list must be empty of
			// THAT particular row; but it will still contain the 16
			// default seeded categories (which were inserted by migration
			// 010 before company_id existed, so they currently live in
			// company 1 too). For this test we only care that company 2
			// cannot see company 1's data — count must be 0.
			list, err := repo.List(context.Background(), wrongCompanyID)
			if err != nil {
				t.Fatalf("CategoryRepository.List(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "SequenceRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			return testutil.SeedInvoiceSequence(t, db, companyID, "LK", 2030)
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewSequenceRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewSequenceRepository(db)
			list, err := repo.List(context.Background(), wrongCompanyID)
			if err != nil {
				t.Fatalf("SequenceRepository.List(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "InvoiceRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			customer := testutil.SeedContact(t, db, companyID, &domain.Contact{Name: "Leak-test Customer"})
			inv := testutil.SeedInvoice(t, db, companyID, customer.ID, []domain.InvoiceItem{
				{Description: "leak-test", Quantity: 100, Unit: "ks", UnitPrice: 10000, VATRatePercent: 21},
			})
			return inv.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewInvoiceRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewInvoiceRepository(db)
			_, total, err := repo.List(context.Background(), wrongCompanyID, domain.InvoiceFilter{})
			if err != nil {
				t.Fatalf("InvoiceRepository.List(other company) error: %v", err)
			}
			return total
		},
	},
	{
		name: "ExpenseRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			exp := testutil.SeedExpense(t, db, companyID, &domain.Expense{
				Description: "Leak-test Expense",
			})
			return exp.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewExpenseRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewExpenseRepository(db)
			_, total, err := repo.List(context.Background(), wrongCompanyID, domain.ExpenseFilter{})
			if err != nil {
				t.Fatalf("ExpenseRepository.List(other company) error: %v", err)
			}
			return total
		},
	},
	{
		name: "RecurringInvoiceRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			customer := testutil.SeedContact(t, db, companyID, &domain.Contact{Name: "Leak-test RI Customer"})
			repo := NewRecurringInvoiceRepository(db)
			ri := &domain.RecurringInvoice{
				Name:          "Leak-test RI",
				CustomerID:    customer.ID,
				Frequency:     domain.FrequencyMonthly,
				NextIssueDate: time.Now(),
				CurrencyCode:  domain.CurrencyCZK,
				ExchangeRate:  100,
				PaymentMethod: "bank_transfer",
				IsActive:      true,
				Items: []domain.RecurringInvoiceItem{
					{Description: "leak", Quantity: 100, Unit: "ks", UnitPrice: 10000, VATRatePercent: 21},
				},
			}
			if err := repo.Create(context.Background(), companyID, ri); err != nil {
				t.Fatalf("seeding recurring invoice: %v", err)
			}
			return ri.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewRecurringInvoiceRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewRecurringInvoiceRepository(db)
			list, err := repo.List(context.Background(), wrongCompanyID)
			if err != nil {
				t.Fatalf("RecurringInvoiceRepository.List(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "RecurringExpenseRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewRecurringExpenseRepository(db)
			re := &domain.RecurringExpense{
				Name:            "Leak-test RE",
				Description:     "leak-test",
				Amount:          domain.NewAmount(1000, 0),
				CurrencyCode:    domain.CurrencyCZK,
				Frequency:       "monthly",
				NextIssueDate:   time.Now(),
				IsActive:        true,
				BusinessPercent: 100,
				PaymentMethod:   "bank_transfer",
			}
			if err := repo.Create(context.Background(), companyID, re); err != nil {
				t.Fatalf("seeding recurring expense: %v", err)
			}
			return re.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewRecurringExpenseRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewRecurringExpenseRepository(db)
			_, total, err := repo.List(context.Background(), wrongCompanyID, 0, 0)
			if err != nil {
				t.Fatalf("RecurringExpenseRepository.List(other company) error: %v", err)
			}
			return total
		},
	},
	{
		name: "InvoiceDocumentRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			customer := testutil.SeedContact(t, db, companyID, &domain.Contact{Name: "Leak-test ID Customer"})
			inv := testutil.SeedInvoice(t, db, companyID, customer.ID, nil)
			repo := NewInvoiceDocumentRepository(db)
			doc := &domain.InvoiceDocument{
				InvoiceID:   inv.ID,
				Filename:    "leak.pdf",
				ContentType: "application/pdf",
				StoragePath: "/tmp/leak.pdf",
				Size:        100,
			}
			if err := repo.Create(context.Background(), companyID, doc); err != nil {
				t.Fatalf("seeding invoice document: %v", err)
			}
			return doc.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewInvoiceDocumentRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			// InvoiceDocumentRepository has no plain List, so probe via ListByInvoiceID
			// with a non-existent invoice ID. The repo filters by company_id first;
			// any returned rows would mean a leak.
			repo := NewInvoiceDocumentRepository(db)
			// Seed invoice exists in company 1 — list it under company 2.
			list, err := repo.ListByInvoiceID(context.Background(), wrongCompanyID, 1)
			if err != nil {
				t.Fatalf("InvoiceDocumentRepository.ListByInvoiceID(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "ExpenseDocumentRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			exp := testutil.SeedExpense(t, db, companyID, nil)
			repo := NewDocumentRepository(db)
			doc := &domain.ExpenseDocument{
				ExpenseID:   exp.ID,
				Filename:    "leak.pdf",
				ContentType: "application/pdf",
				StoragePath: "/tmp/leak.pdf",
				Size:        100,
			}
			if err := repo.Create(context.Background(), companyID, doc); err != nil {
				t.Fatalf("seeding expense document: %v", err)
			}
			return doc.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewDocumentRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewDocumentRepository(db)
			list, err := repo.ListByExpenseID(context.Background(), wrongCompanyID, 1)
			if err != nil {
				t.Fatalf("DocumentRepository.ListByExpenseID(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "StatusHistoryRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			customer := testutil.SeedContact(t, db, companyID, &domain.Contact{Name: "Leak-test SH Customer"})
			inv := testutil.SeedInvoice(t, db, companyID, customer.ID, nil)
			repo := NewStatusHistoryRepository(db)
			change := &domain.InvoiceStatusChange{
				InvoiceID: inv.ID,
				OldStatus: domain.InvoiceStatusDraft,
				NewStatus: domain.InvoiceStatusSent,
				ChangedAt: time.Now(),
				Note:      "leak-test",
			}
			if err := repo.Create(context.Background(), companyID, change); err != nil {
				t.Fatalf("seeding status history: %v", err)
			}
			return inv.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			// StatusHistoryRepository has no GetByID; probe via ListByInvoiceID.
			// Empty list under wrong company indicates the row is hidden as expected.
			repo := NewStatusHistoryRepository(db)
			list, err := repo.ListByInvoiceID(context.Background(), wrongCompanyID, entityID)
			if err != nil {
				return err
			}
			if len(list) > 0 {
				return nil // leak: should have been empty
			}
			return domain.ErrNotFound
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewStatusHistoryRepository(db)
			// Invoice id=1 belongs to company 1 in the seed; under company 2 should yield nothing.
			list, err := repo.ListByInvoiceID(context.Background(), wrongCompanyID, 1)
			if err != nil {
				t.Fatalf("StatusHistoryRepository.ListByInvoiceID(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "ReminderRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			customer := testutil.SeedContact(t, db, companyID, &domain.Contact{Name: "Leak-test Rem Customer"})
			inv := testutil.SeedInvoice(t, db, companyID, customer.ID, nil)
			repo := NewReminderRepository(db)
			rem := &domain.PaymentReminder{
				InvoiceID:      inv.ID,
				ReminderNumber: 1,
				SentAt:         time.Now(),
				SentTo:         "leak@example.com",
				Subject:        "leak-test",
				BodyPreview:    "leak-test",
			}
			if err := repo.Create(context.Background(), companyID, rem); err != nil {
				t.Fatalf("seeding reminder: %v", err)
			}
			return inv.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			// ReminderRepository has no GetByID; probe via ListByInvoiceID.
			repo := NewReminderRepository(db)
			list, err := repo.ListByInvoiceID(context.Background(), wrongCompanyID, entityID)
			if err != nil {
				return err
			}
			if len(list) > 0 {
				return nil // leak
			}
			return domain.ErrNotFound
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewReminderRepository(db)
			list, err := repo.ListByInvoiceID(context.Background(), wrongCompanyID, 1)
			if err != nil {
				t.Fatalf("ReminderRepository.ListByInvoiceID(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "SettingsRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewSettingsRepository(db)
			if err := repo.Set(context.Background(), companyID, "company_name", "Leak-test"); err != nil {
				t.Fatalf("seeding setting: %v", err)
			}
			// No int id; return 0 (we probe via Get/GetAll keyed by name).
			return 0
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewSettingsRepository(db)
			_, err := repo.Get(context.Background(), wrongCompanyID, "company_name")
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewSettingsRepository(db)
			all, err := repo.GetAll(context.Background(), wrongCompanyID)
			if err != nil {
				t.Fatalf("SettingsRepository.GetAll(other company) error: %v", err)
			}
			return len(all)
		},
	},
	{
		name: "VATReturnRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewVATReturnRepository(db)
			vr := &domain.VATReturn{
				Period:     domain.TaxPeriod{Year: 2030, Quarter: 1},
				FilingType: domain.FilingTypeRegular,
				Status:     domain.FilingStatusDraft,
			}
			if err := repo.Create(context.Background(), companyID, vr); err != nil {
				t.Fatalf("seeding vat_return: %v", err)
			}
			return vr.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewVATReturnRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewVATReturnRepository(db)
			list, err := repo.List(context.Background(), wrongCompanyID, 2030)
			if err != nil {
				t.Fatalf("VATReturnRepository.List(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "VATControlStatementRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewVATControlStatementRepository(db)
			cs := &domain.VATControlStatement{
				Period:     domain.TaxPeriod{Year: 2030, Month: 1},
				FilingType: domain.FilingTypeRegular,
				Status:     domain.FilingStatusDraft,
			}
			if err := repo.Create(context.Background(), companyID, cs); err != nil {
				t.Fatalf("seeding control statement: %v", err)
			}
			return cs.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewVATControlStatementRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewVATControlStatementRepository(db)
			list, err := repo.List(context.Background(), wrongCompanyID, 2030)
			if err != nil {
				t.Fatalf("VATControlStatementRepository.List(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "VIESSummaryRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewVIESSummaryRepository(db)
			vs := &domain.VIESSummary{
				Period:     domain.TaxPeriod{Year: 2030, Quarter: 1},
				FilingType: domain.FilingTypeRegular,
				Status:     domain.FilingStatusDraft,
			}
			if err := repo.Create(context.Background(), companyID, vs); err != nil {
				t.Fatalf("seeding vies summary: %v", err)
			}
			return vs.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewVIESSummaryRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewVIESSummaryRepository(db)
			list, err := repo.List(context.Background(), wrongCompanyID, 2030)
			if err != nil {
				t.Fatalf("VIESSummaryRepository.List(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "IncomeTaxReturnRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewIncomeTaxReturnRepository(db)
			itr := &domain.IncomeTaxReturn{
				Year:       2030,
				FilingType: domain.FilingTypeRegular,
				Status:     domain.FilingStatusDraft,
			}
			if err := repo.Create(context.Background(), companyID, itr); err != nil {
				t.Fatalf("seeding income tax return: %v", err)
			}
			return itr.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewIncomeTaxReturnRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewIncomeTaxReturnRepository(db)
			list, err := repo.List(context.Background(), wrongCompanyID, 2030)
			if err != nil {
				t.Fatalf("IncomeTaxReturnRepository.List(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "SocialInsuranceOverviewRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewSocialInsuranceOverviewRepository(db)
			sio := &domain.SocialInsuranceOverview{
				Year:       2030,
				FilingType: domain.FilingTypeRegular,
				Status:     domain.FilingStatusDraft,
			}
			if err := repo.Create(context.Background(), companyID, sio); err != nil {
				t.Fatalf("seeding social insurance overview: %v", err)
			}
			return sio.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewSocialInsuranceOverviewRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewSocialInsuranceOverviewRepository(db)
			list, err := repo.List(context.Background(), wrongCompanyID, 2030)
			if err != nil {
				t.Fatalf("SocialInsuranceOverviewRepository.List(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "HealthInsuranceOverviewRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewHealthInsuranceOverviewRepository(db)
			hi := &domain.HealthInsuranceOverview{
				Year:       2030,
				FilingType: domain.FilingTypeRegular,
				Status:     domain.FilingStatusDraft,
			}
			if err := repo.Create(context.Background(), companyID, hi); err != nil {
				t.Fatalf("seeding health insurance overview: %v", err)
			}
			return hi.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewHealthInsuranceOverviewRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewHealthInsuranceOverviewRepository(db)
			list, err := repo.List(context.Background(), wrongCompanyID, 2030)
			if err != nil {
				t.Fatalf("HealthInsuranceOverviewRepository.List(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "TaxYearSettingsRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewTaxYearSettingsRepository(db)
			tys := &domain.TaxYearSettings{Year: 2030, FlatRatePercent: 40}
			if err := repo.Upsert(context.Background(), companyID, tys); err != nil {
				t.Fatalf("seeding tax_year_settings: %v", err)
			}
			return int64(tys.Year)
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewTaxYearSettingsRepository(db)
			_, err := repo.GetByYear(context.Background(), wrongCompanyID, int(entityID))
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			// TaxYearSettings has no List; reuse GetByYear and check ErrNotFound for the seeded year.
			repo := NewTaxYearSettingsRepository(db)
			_, err := repo.GetByYear(context.Background(), wrongCompanyID, 2030)
			if err == nil {
				return 1 // leak
			}
			return 0
		},
	},
	{
		name: "TaxPrepaymentRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewTaxPrepaymentRepository(db)
			prepayments := []domain.TaxPrepayment{
				{Year: 2030, Month: 1, TaxAmount: domain.Amount(100)},
			}
			if err := repo.UpsertAll(context.Background(), companyID, 2030, prepayments); err != nil {
				t.Fatalf("seeding tax_prepayments: %v", err)
			}
			return 2030
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			// TaxPrepayments has no GetByID; probe via ListByYear.
			repo := NewTaxPrepaymentRepository(db)
			list, err := repo.ListByYear(context.Background(), wrongCompanyID, int(entityID))
			if err != nil {
				return err
			}
			if len(list) > 0 {
				return nil // leak
			}
			return domain.ErrNotFound
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewTaxPrepaymentRepository(db)
			list, err := repo.ListByYear(context.Background(), wrongCompanyID, 2030)
			if err != nil {
				t.Fatalf("TaxPrepaymentRepository.ListByYear(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "TaxSpouseCreditRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewTaxSpouseCreditRepository(db)
			credit := &domain.TaxSpouseCredit{
				Year:          2030,
				SpouseName:    "Leak-test Spouse",
				MonthsClaimed: 12,
			}
			if err := repo.Upsert(context.Background(), companyID, credit); err != nil {
				t.Fatalf("seeding spouse credit: %v", err)
			}
			return credit.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewTaxSpouseCreditRepository(db)
			_, err := repo.GetByYear(context.Background(), wrongCompanyID, 2030)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewTaxSpouseCreditRepository(db)
			_, err := repo.GetByYear(context.Background(), wrongCompanyID, 2030)
			if err == nil {
				return 1 // leak
			}
			return 0
		},
	},
	{
		name: "TaxChildCreditRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewTaxChildCreditRepository(db)
			credit := &domain.TaxChildCredit{
				Year:          2030,
				ChildName:     "Leak-test Child",
				ChildOrder:    1,
				MonthsClaimed: 12,
			}
			if err := repo.Create(context.Background(), companyID, credit); err != nil {
				t.Fatalf("seeding child credit: %v", err)
			}
			return credit.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			// TaxChildCredit has no GetByID; probe via ListByYear.
			repo := NewTaxChildCreditRepository(db)
			list, err := repo.ListByYear(context.Background(), wrongCompanyID, 2030)
			if err != nil {
				return err
			}
			if len(list) > 0 {
				return nil // leak
			}
			return domain.ErrNotFound
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewTaxChildCreditRepository(db)
			list, err := repo.ListByYear(context.Background(), wrongCompanyID, 2030)
			if err != nil {
				t.Fatalf("TaxChildCreditRepository.ListByYear(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "TaxPersonalCreditsRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewTaxPersonalCreditsRepository(db)
			credits := &domain.TaxPersonalCredits{
				Year:      2030,
				IsStudent: false,
			}
			if err := repo.Upsert(context.Background(), companyID, credits); err != nil {
				t.Fatalf("seeding personal credits: %v", err)
			}
			return 2030
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewTaxPersonalCreditsRepository(db)
			_, err := repo.GetByYear(context.Background(), wrongCompanyID, int(entityID))
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewTaxPersonalCreditsRepository(db)
			_, err := repo.GetByYear(context.Background(), wrongCompanyID, 2030)
			if err == nil {
				return 1 // leak
			}
			return 0
		},
	},
	{
		name: "TaxDeductionRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewTaxDeductionRepository(db)
			ded := &domain.TaxDeduction{
				Year:          2030,
				Category:      "mortgage",
				Description:   "Leak-test deduction",
				ClaimedAmount: domain.Amount(100),
			}
			if err := repo.Create(context.Background(), companyID, ded); err != nil {
				t.Fatalf("seeding tax deduction: %v", err)
			}
			return ded.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewTaxDeductionRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewTaxDeductionRepository(db)
			list, err := repo.ListByYear(context.Background(), wrongCompanyID, 2030)
			if err != nil {
				t.Fatalf("TaxDeductionRepository.ListByYear(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "TaxDeductionDocumentRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			// Need a parent deduction for the FK.
			dedRepo := NewTaxDeductionRepository(db)
			ded := &domain.TaxDeduction{Year: 2030, Category: "mortgage", Description: "Parent"}
			if err := dedRepo.Create(context.Background(), companyID, ded); err != nil {
				t.Fatalf("seeding parent deduction: %v", err)
			}
			repo := NewTaxDeductionDocumentRepository(db)
			doc := &domain.TaxDeductionDocument{
				TaxDeductionID: ded.ID,
				Filename:       "leak.pdf",
				ContentType:    "application/pdf",
				StoragePath:    "/tmp/leak.pdf",
				Size:           100,
			}
			if err := repo.Create(context.Background(), companyID, doc); err != nil {
				t.Fatalf("seeding tax_deduction_document: %v", err)
			}
			return doc.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewTaxDeductionDocumentRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewTaxDeductionDocumentRepository(db)
			// Probe with the seeded deduction's id (1); company 2 must not see it.
			list, err := repo.ListByDeductionID(context.Background(), wrongCompanyID, 1)
			if err != nil {
				t.Fatalf("TaxDeductionDocumentRepository.ListByDeductionID(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "InvestmentDocumentRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewInvestmentDocumentRepository(db)
			doc := &domain.InvestmentDocument{
				Year:             2030,
				Platform:         "portu",
				Filename:         "leak.pdf",
				ContentType:      "application/pdf",
				StoragePath:      "/tmp/leak.pdf",
				Size:             100,
				ExtractionStatus: domain.ExtractionPending,
			}
			if err := repo.Create(context.Background(), companyID, doc); err != nil {
				t.Fatalf("seeding investment_document: %v", err)
			}
			return doc.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewInvestmentDocumentRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewInvestmentDocumentRepository(db)
			list, err := repo.ListByYear(context.Background(), wrongCompanyID, 2030)
			if err != nil {
				t.Fatalf("InvestmentDocumentRepository.ListByYear(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "CapitalIncomeRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewCapitalIncomeRepository(db)
			entry := &domain.CapitalIncomeEntry{
				Year:        2030,
				Category:    domain.CapitalCategoryDividendCZ,
				Description: "Leak-test",
				IncomeDate:  time.Now(),
				GrossAmount: domain.Amount(1000),
				NetAmount:   domain.Amount(1000),
			}
			if err := repo.Create(context.Background(), companyID, entry); err != nil {
				t.Fatalf("seeding capital_income_entry: %v", err)
			}
			return entry.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewCapitalIncomeRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewCapitalIncomeRepository(db)
			list, err := repo.ListByYear(context.Background(), wrongCompanyID, 2030)
			if err != nil {
				t.Fatalf("CapitalIncomeRepository.ListByYear(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "SecurityTransactionRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewSecurityTransactionRepository(db)
			tx := &domain.SecurityTransaction{
				Year:            2030,
				AssetType:       domain.AssetTypeStock,
				AssetName:       "Leak-test ASSET",
				TransactionType: domain.TransactionTypeBuy,
				TransactionDate: time.Now(),
				Quantity:        100,
				UnitPrice:       domain.Amount(1000),
				TotalAmount:     domain.Amount(100000),
				CurrencyCode:    domain.CurrencyCZK,
				ExchangeRate:    100,
			}
			if err := repo.Create(context.Background(), companyID, tx); err != nil {
				t.Fatalf("seeding security_transaction: %v", err)
			}
			return tx.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			repo := NewSecurityTransactionRepository(db)
			_, err := repo.GetByID(context.Background(), wrongCompanyID, entityID)
			return err
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewSecurityTransactionRepository(db)
			list, err := repo.ListByYear(context.Background(), wrongCompanyID, 2030)
			if err != nil {
				t.Fatalf("SecurityTransactionRepository.ListByYear(other company) error: %v", err)
			}
			return len(list)
		},
	},
	{
		name: "FakturoidImportLogRepository",
		seed: func(t *testing.T, db *sql.DB, companyID int64) int64 {
			repo := NewFakturoidImportLogRepository(db)
			entry := &domain.FakturoidImportLog{
				FakturoidEntityType: "subject",
				FakturoidID:         12345,
				LocalEntityType:     "contact",
				LocalID:             1,
			}
			if err := repo.Create(context.Background(), companyID, entry); err != nil {
				t.Fatalf("seeding fakturoid_import_log: %v", err)
			}
			return entry.ID
		},
		getByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID, entityID int64) error {
			// FakturoidImportLog has no GetByID; probe via FindByFakturoidID.
			repo := NewFakturoidImportLogRepository(db)
			entry, err := repo.FindByFakturoidID(context.Background(), wrongCompanyID, "subject", 12345)
			if err != nil {
				return err
			}
			if entry != nil {
				return nil // leak
			}
			return domain.ErrNotFound
		},
		listByOtherCompany: func(t *testing.T, db *sql.DB, wrongCompanyID int64) int {
			repo := NewFakturoidImportLogRepository(db)
			list, err := repo.ListByEntityType(context.Background(), wrongCompanyID, "subject")
			if err != nil {
				t.Fatalf("FakturoidImportLogRepository.ListByEntityType(other company) error: %v", err)
			}
			return len(list)
		},
	},
}

func TestCrossCompanyLeakDetection(t *testing.T) {
	for _, lc := range leakCases {
		t.Run(lc.name+"_GetByID", func(t *testing.T) {
			db := setupLeakDetectorDB(t)
			id := lc.seed(t, db, 1)

			err := lc.getByOtherCompany(t, db, 2, id)
			if err == nil {
				t.Errorf("%s: company 2 can read company 1's entity %d (no error)", lc.name, id)
				return
			}
			if !errors.Is(err, domain.ErrNotFound) {
				t.Errorf("%s: got %v, want ErrNotFound", lc.name, err)
			}
		})
		t.Run(lc.name+"_List", func(t *testing.T) {
			db := setupLeakDetectorDB(t)
			lc.seed(t, db, 1)

			n := lc.listByOtherCompany(t, db, 2)
			if n != 0 {
				t.Errorf("%s: company 2 can list %d of company 1's rows; want 0", lc.name, n)
			}
		})
	}
}
