package cli

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/zajca/zfaktury/internal/config"
	"github.com/zajca/zfaktury/internal/database"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
)

var seedForce bool

func init() {
	seedCmd.Flags().BoolVar(&seedForce, "force", false, "Clear existing data before seeding")
	rootCmd.AddCommand(seedCmd)
}

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Populate database with demo data",
	Long:  "Seed the database with realistic demo data for a Czech freelance web developer.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSeed()
	},
}

// noopARES is a stub ARES client that is never called during seeding.
type noopARES struct{}

func (noopARES) LookupByICO(_ context.Context, _ string) (*domain.Contact, error) {
	return nil, fmt.Errorf("ARES lookup not available in seed mode")
}

func runSeed() error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	db, err := database.New(cfg)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	// Wire repositories.
	contactRepo := repository.NewContactRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	sequenceRepo := repository.NewSequenceRepository(db)
	recurringInvoiceRepo := repository.NewRecurringInvoiceRepository(db)
	recurringExpenseRepo := repository.NewRecurringExpenseRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	auditLogRepo := repository.NewAuditLogRepository(db)
	vatReturnRepo := repository.NewVATReturnRepository(db)
	vatControlRepo := repository.NewVATControlStatementRepository(db)
	viesRepo := repository.NewVIESSummaryRepository(db)
	incomeTaxReturnRepo := repository.NewIncomeTaxReturnRepository(db)
	socialInsuranceRepo := repository.NewSocialInsuranceOverviewRepository(db)
	healthInsuranceRepo := repository.NewHealthInsuranceOverviewRepository(db)
	taxYearSettingsRepo := repository.NewTaxYearSettingsRepository(db)
	taxPrepaymentRepo := repository.NewTaxPrepaymentRepository(db)
	taxSpouseCreditRepo := repository.NewTaxSpouseCreditRepository(db)
	taxChildCreditRepo := repository.NewTaxChildCreditRepository(db)
	taxPersonalCreditsRepo := repository.NewTaxPersonalCreditsRepository(db)
	taxDeductionRepo := repository.NewTaxDeductionRepository(db)
	capitalIncomeRepo := repository.NewCapitalIncomeRepository(db)
	securityTransactionRepo := repository.NewSecurityTransactionRepository(db)

	// Wire services.
	auditSvc := service.NewAuditService(auditLogRepo)
	settingsSvc := service.NewSettingsService(settingsRepo, auditSvc)
	contactSvc := service.NewContactService(contactRepo, noopARES{}, auditSvc)
	sequenceSvc := service.NewSequenceService(sequenceRepo, auditSvc)
	invoiceSvc := service.NewInvoiceService(invoiceRepo, contactSvc, sequenceSvc, auditSvc)
	expenseSvc := service.NewExpenseService(expenseRepo, auditSvc)
	recurringInvoiceSvc := service.NewRecurringInvoiceService(recurringInvoiceRepo, invoiceSvc, auditSvc)
	recurringExpenseSvc := service.NewRecurringExpenseService(recurringExpenseRepo, expenseSvc, auditSvc)
	vatReturnSvc := service.NewVATReturnService(vatReturnRepo, invoiceRepo, expenseRepo, settingsRepo, nil)
	vatControlSvc := service.NewVATControlStatementService(vatControlRepo, invoiceRepo, expenseRepo, contactRepo, nil)
	viesSvc := service.NewVIESSummaryService(viesRepo, invoiceRepo, contactRepo, nil)
	taxYearSettingsSvc := service.NewTaxYearSettingsService(taxYearSettingsRepo, taxPrepaymentRepo, nil)
	taxCreditsSvc := service.NewTaxCreditsService(taxSpouseCreditRepo, taxChildCreditRepo, taxPersonalCreditsRepo, taxDeductionRepo, auditSvc)
	incomeTaxSvc := service.NewIncomeTaxReturnService(incomeTaxReturnRepo, invoiceRepo, expenseRepo, settingsRepo, taxYearSettingsRepo, taxPrepaymentRepo, taxCreditsSvc, nil)
	socialInsuranceSvc := service.NewSocialInsuranceService(socialInsuranceRepo, invoiceRepo, expenseRepo, settingsRepo, taxYearSettingsRepo, taxPrepaymentRepo, nil)
	healthInsuranceSvc := service.NewHealthInsuranceService(healthInsuranceRepo, invoiceRepo, expenseRepo, settingsRepo, taxYearSettingsRepo, taxPrepaymentRepo, nil)
	investmentIncomeSvc := service.NewInvestmentIncomeService(capitalIncomeRepo, securityTransactionRepo, nil)

	ctx := context.Background()

	// Check if data already exists.
	if !seedForce {
		contacts, _, err := contactRepo.List(ctx, domain.ContactFilter{Limit: 1})
		if err != nil {
			return fmt.Errorf("checking existing data: %w", err)
		}
		if len(contacts) > 0 {
			fmt.Println("Database already contains data. Use --force to clear and re-seed.")
			return nil
		}
	}

	if seedForce {
		if err := clearDatabase(db); err != nil {
			return fmt.Errorf("clearing database: %w", err)
		}
		slog.Info("cleared existing data")
	}

	if err := seedSettings(ctx, settingsSvc); err != nil {
		return fmt.Errorf("seeding settings: %w", err)
	}
	slog.Info("seeded business settings")

	contactIDs, err := seedContacts(ctx, contactSvc)
	if err != nil {
		return fmt.Errorf("seeding contacts: %w", err)
	}
	slog.Info("seeded contacts", "count", len(contactIDs))

	invoiceCount, err := seedInvoices(ctx, invoiceSvc, invoiceRepo, contactIDs)
	if err != nil {
		return fmt.Errorf("seeding invoices: %w", err)
	}
	slog.Info("seeded invoices", "count", invoiceCount)

	expenseCount, err := seedExpenses(ctx, expenseSvc, contactIDs)
	if err != nil {
		return fmt.Errorf("seeding expenses: %w", err)
	}
	slog.Info("seeded expenses", "count", expenseCount)

	riCount, err := seedRecurringInvoices(ctx, recurringInvoiceSvc, contactIDs)
	if err != nil {
		return fmt.Errorf("seeding recurring invoices: %w", err)
	}
	slog.Info("seeded recurring invoices", "count", riCount)

	reCount, err := seedRecurringExpenses(ctx, recurringExpenseSvc)
	if err != nil {
		return fmt.Errorf("seeding recurring expenses: %w", err)
	}
	slog.Info("seeded recurring expenses", "count", reCount)

	if err := seedTaxYearSettings(ctx, taxYearSettingsSvc); err != nil {
		return fmt.Errorf("seeding tax year settings: %w", err)
	}
	slog.Info("seeded tax year settings and prepayments")

	if err := seedTaxCredits(ctx, taxCreditsSvc); err != nil {
		return fmt.Errorf("seeding tax credits: %w", err)
	}
	slog.Info("seeded tax credits and deductions")

	if err := seedVATReturns(ctx, vatReturnSvc); err != nil {
		return fmt.Errorf("seeding VAT returns: %w", err)
	}
	slog.Info("seeded VAT returns")

	if err := seedVATControlStatements(ctx, vatControlSvc); err != nil {
		return fmt.Errorf("seeding VAT control statements: %w", err)
	}
	slog.Info("seeded VAT control statements")

	if err := seedVIESSummaries(ctx, viesSvc); err != nil {
		return fmt.Errorf("seeding VIES summaries: %w", err)
	}
	slog.Info("seeded VIES summaries")

	if err := seedIncomeTaxReturn(ctx, incomeTaxSvc); err != nil {
		return fmt.Errorf("seeding income tax return: %w", err)
	}
	slog.Info("seeded income tax return")

	if err := seedSocialInsurance(ctx, socialInsuranceSvc); err != nil {
		return fmt.Errorf("seeding social insurance: %w", err)
	}
	slog.Info("seeded social insurance overview")

	if err := seedHealthInsurance(ctx, healthInsuranceSvc); err != nil {
		return fmt.Errorf("seeding health insurance: %w", err)
	}
	slog.Info("seeded health insurance overview")

	if err := seedInvestments(ctx, investmentIncomeSvc); err != nil {
		return fmt.Errorf("seeding investments: %w", err)
	}
	slog.Info("seeded investment income")

	fmt.Println("Seed complete: settings, contacts, invoices, expenses, recurring templates, VAT, tax returns, investments")
	return nil
}

func clearDatabase(db *sql.DB) error {
	// Temporarily disable FK checks so we can delete in any order.
	if _, err := db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("disabling foreign keys: %w", err)
	}
	defer func() { _, _ = db.Exec("PRAGMA foreign_keys = ON") }()

	// All user-data tables in deletion order (children before parents).
	tables := []string{
		"audit_log",
		"expense_documents",
		"invoice_status_history",
		"payment_reminders",
		"invoice_items",
		"invoices",
		"invoice_sequences",
		"recurring_invoice_items",
		"recurring_invoices",
		"recurring_expenses",
		"expenses",
		"contacts",
		"settings",
		"vat_returns",
		"vat_return_invoices",
		"vat_return_expenses",
		"vat_control_statements",
		"vat_control_statement_lines",
		"vies_summaries",
		"vies_summary_lines",
		"income_tax_returns",
		"income_tax_return_invoices",
		"income_tax_return_expenses",
		"social_insurance_overviews",
		"health_insurance_overviews",
		"tax_year_settings",
		"tax_prepayments",
		"tax_spouse_credits",
		"tax_child_credits",
		"tax_personal_credits",
		"tax_deductions",
		"tax_deduction_documents",
		"capital_income_entries",
		"security_transactions",
		"investment_documents",
	}
	for _, table := range tables {
		if _, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			return fmt.Errorf("clearing %s: %w", table, err)
		}
	}
	return nil
}

// contactIDs maps alias to database ID.
type contactIDs map[string]int64

func seedContacts(ctx context.Context, svc *service.ContactService) (contactIDs, error) {
	ids := make(contactIDs)

	contacts := []struct {
		alias   string
		contact domain.Contact
	}{
		{"starter", domain.Contact{
			Type: domain.ContactTypeCompany, Name: "Webova Agentura Starter s.r.o.",
			ICO: "12345678", DIC: "CZ12345678",
			Street: "Vinohradska 42", City: "Praha", ZIP: "12000", Country: "CZ",
			Email: "info@starter-web.cz", Phone: "+420 601 123 456",
			BankAccount: "1234567890", BankCode: "0100",
			PaymentTermsDays: 14, IsFavorite: true,
		}},
		{"kava", domain.Contact{
			Type: domain.ContactTypeCompany, Name: "Kava Digital a.s.",
			ICO: "27654321", DIC: "CZ27654321",
			Street: "Masarykova 15", City: "Brno", ZIP: "60200", Country: "CZ",
			Email: "fakturace@kavadigital.cz", Phone: "+420 541 234 567",
			BankAccount: "9876543210", BankCode: "0300",
			PaymentTermsDays: 30, IsFavorite: true,
		}},
		{"restaurace", domain.Contact{
			Type: domain.ContactTypeCompany, Name: "Restaurace Pod Lipou s.r.o.",
			ICO: "45678901", DIC: "CZ45678901",
			Street: "Namesti Miru 8", City: "Olomouc", ZIP: "77900", Country: "CZ",
			Email: "provoz@podlipou.cz", Phone: "+420 585 345 678",
			BankAccount: "1122334455", BankCode: "0600",
			PaymentTermsDays: 14,
		}},
		{"fitlife", domain.Contact{
			Type: domain.ContactTypeCompany, Name: "FitLife Coaching s.r.o.",
			ICO: "56789012", DIC: "CZ56789012",
			Street: "Americka 28", City: "Plzen", ZIP: "30100", Country: "CZ",
			Email: "admin@fitlife.cz", Phone: "+420 377 456 789",
			BankAccount: "5566778899", BankCode: "0100",
			PaymentTermsDays: 21,
		}},
		{"petr", domain.Contact{
			Type: domain.ContactTypeIndividual, Name: "Petr Svoboda",
			Street: "Nad Vodovodem 12", City: "Praha", ZIP: "10000", Country: "CZ",
			Email: "petr.svoboda@email.cz", Phone: "+420 777 111 222",
			BankAccount: "2233445566", BankCode: "0800",
			PaymentTermsDays: 7,
		}},
		{"marie", domain.Contact{
			Type: domain.ContactTypeIndividual, Name: "Marie Kralova",
			Street: "Husova 3", City: "Ceske Budejovice", ZIP: "37001", Country: "CZ",
			Email: "marie.kralova@seznam.cz", Phone: "+420 608 333 444",
			BankAccount: "6677889900", BankCode: "2010",
			PaymentTermsDays: 14,
		}},
		{"technoserv", domain.Contact{
			Type: domain.ContactTypeCompany, Name: "TechnoServ s.r.o.",
			ICO: "67890123", DIC: "CZ67890123",
			Street: "Sokolska 18", City: "Ostrava", ZIP: "70200", Country: "CZ",
			Email: "obchod@technoserv.cz", Phone: "+420 596 567 890",
			BankAccount: "3344556677", BankCode: "0100",
			PaymentTermsDays: 30,
		}},
		{"blesk", domain.Contact{
			Type: domain.ContactTypeCompany, Name: "Kreativni Studio Blesk s.r.o.",
			ICO: "78901234", DIC: "CZ78901234",
			Street: "Palachova 5", City: "Hradec Kralove", ZIP: "50002", Country: "CZ",
			Email: "studio@blesk-kreativ.cz", Phone: "+420 495 678 901",
			BankAccount: "4455667788", BankCode: "0300",
			PaymentTermsDays: 14, IsFavorite: true,
		}},
		{"dataflow", domain.Contact{
			Type: domain.ContactTypeCompany, Name: "DataFlow Analytics GmbH",
			DIC: "DE812345678",
			Street: "Friedrichstrasse 120", City: "Berlin", ZIP: "10117", Country: "DE",
			Email: "billing@dataflow-analytics.de", Phone: "+49 30 123 4567",
			IBAN: "DE89370400440532013000", SWIFT: "COBADEFFXXX",
			PaymentTermsDays: 30,
		}},
		{"ucetnictvi", domain.Contact{
			Type: domain.ContactTypeCompany, Name: "Ucetnictvi Novotna s.r.o.",
			ICO: "89012345", DIC: "CZ89012345",
			Street: "Jugoslavska 22", City: "Praha", ZIP: "12000", Country: "CZ",
			Email: "novotna@ucetnictvi-novotna.cz", Phone: "+420 222 789 012",
			BankAccount: "7788990011", BankCode: "2010",
			PaymentTermsDays: 14,
		}},
	}

	for _, c := range contacts {
		contact := c.contact
		if err := svc.Create(ctx, &contact); err != nil {
			return nil, fmt.Errorf("creating contact %s: %w", c.alias, err)
		}
		ids[c.alias] = contact.ID
	}
	return ids, nil
}

func seedInvoices(ctx context.Context, svc *service.InvoiceService, repo repository.InvoiceRepo, ids contactIDs) (int, error) {
	d := func(year, month, day int) time.Time {
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}
	count := 0

	// Helper: create, send, and pay an invoice.
	createPaid := func(inv *domain.Invoice, paidAt time.Time) error {
		if err := svc.Create(ctx, inv); err != nil {
			return err
		}
		if err := svc.MarkAsSent(ctx, inv.ID); err != nil {
			return err
		}
		if err := svc.MarkAsPaid(ctx, inv.ID, inv.TotalAmount, paidAt); err != nil {
			return err
		}
		count++
		return nil
	}

	// Helper: create and send (unpaid).
	createSent := func(inv *domain.Invoice) error {
		if err := svc.Create(ctx, inv); err != nil {
			return err
		}
		if err := svc.MarkAsSent(ctx, inv.ID); err != nil {
			return err
		}
		count++
		return nil
	}

	// Helper: create draft only.
	createDraft := func(inv *domain.Invoice) error {
		if err := svc.Create(ctx, inv); err != nil {
			return err
		}
		count++
		return nil
	}

	// --- 2025: all paid (10 invoices) ---

	// 1. Starter — web redesign, 80h x 1200 CZK, 21% VAT
	if err := createPaid(&domain.Invoice{
		CustomerID: ids["starter"], IssueDate: d(2025, 1, 15), DueDate: d(2025, 1, 29), DeliveryDate: d(2025, 1, 15),
		CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "Web redesign - design and development", Quantity: 8000, Unit: "hod", UnitPrice: 120000, VATRatePercent: 21},
		},
	}, d(2025, 1, 27)); err != nil {
		return 0, fmt.Errorf("invoice 1: %w", err)
	}

	// 2. Kava Digital — e-shop, 120h x 1500 CZK, 21% VAT
	if err := createPaid(&domain.Invoice{
		CustomerID: ids["kava"], IssueDate: d(2025, 2, 1), DueDate: d(2025, 3, 3), DeliveryDate: d(2025, 2, 1),
		CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "E-shop development - SvelteKit + Go backend", Quantity: 12000, Unit: "hod", UnitPrice: 150000, VATRatePercent: 21},
		},
	}, d(2025, 2, 25)); err != nil {
		return 0, fmt.Errorf("invoice 2: %w", err)
	}

	// 3. Restaurace — website + hosting, ~25K + VAT
	if err := createPaid(&domain.Invoice{
		CustomerID: ids["restaurace"], IssueDate: d(2025, 3, 10), DueDate: d(2025, 3, 24), DeliveryDate: d(2025, 3, 10),
		CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "Restaurant website - design and implementation", Quantity: 100, Unit: "ks", UnitPrice: 2200000, VATRatePercent: 21},
			{Description: "Web hosting - annual", Quantity: 100, Unit: "ks", UnitPrice: 300000, VATRatePercent: 21},
		},
	}, d(2025, 3, 22)); err != nil {
		return 0, fmt.Errorf("invoice 3: %w", err)
	}

	// 4. Petr Svoboda — portfolio, 40h x 800, 0% VAT (individual, non-VAT)
	if err := createPaid(&domain.Invoice{
		CustomerID: ids["petr"], IssueDate: d(2025, 4, 5), DueDate: d(2025, 4, 12), DeliveryDate: d(2025, 4, 5),
		CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "Personal portfolio website", Quantity: 4000, Unit: "hod", UnitPrice: 80000, VATRatePercent: 0},
		},
	}, d(2025, 4, 10)); err != nil {
		return 0, fmt.Errorf("invoice 4: %w", err)
	}

	// 5. FitLife — landing page, ~45K + VAT
	if err := createPaid(&domain.Invoice{
		CustomerID: ids["fitlife"], IssueDate: d(2025, 5, 1), DueDate: d(2025, 5, 22), DeliveryDate: d(2025, 5, 1),
		CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "Landing page - design", Quantity: 100, Unit: "ks", UnitPrice: 1500000, VATRatePercent: 21},
			{Description: "Landing page - development and deployment", Quantity: 100, Unit: "ks", UnitPrice: 3000000, VATRatePercent: 21},
		},
	}, d(2025, 5, 18)); err != nil {
		return 0, fmt.Errorf("invoice 5: %w", err)
	}

	// 6. Kava Digital — maintenance Q2, 20h x 1500, 21% VAT
	if err := createPaid(&domain.Invoice{
		CustomerID: ids["kava"], IssueDate: d(2025, 7, 1), DueDate: d(2025, 7, 31), DeliveryDate: d(2025, 6, 30),
		CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "E-shop maintenance Q2 2025", Quantity: 2000, Unit: "hod", UnitPrice: 150000, VATRatePercent: 21},
		},
	}, d(2025, 7, 15)); err != nil {
		return 0, fmt.Errorf("invoice 6: %w", err)
	}

	// 7. DataFlow GmbH — API integration, 60h x 1800 CZK, 0% VAT (reverse charge), EUR
	if err := createPaid(&domain.Invoice{
		CustomerID: ids["dataflow"], IssueDate: d(2025, 8, 1), DueDate: d(2025, 8, 31), DeliveryDate: d(2025, 8, 1),
		CurrencyCode: domain.CurrencyEUR, ExchangeRate: domain.NewAmount(25, 34), PaymentMethod: "bank_transfer",
		Notes: "Reverse charge - VAT to be paid by the recipient according to Article 196 of the Council Directive 2006/112/EC.",
		Items: []domain.InvoiceItem{
			{Description: "API integration - data pipeline connectors", Quantity: 6000, Unit: "hod", UnitPrice: 7200, VATRatePercent: 0},
		},
	}, d(2025, 8, 28)); err != nil {
		return 0, fmt.Errorf("invoice 7: %w", err)
	}

	// 8. Starter — SEO optimization, ~18K + VAT
	if err := createPaid(&domain.Invoice{
		CustomerID: ids["starter"], IssueDate: d(2025, 9, 1), DueDate: d(2025, 9, 15), DeliveryDate: d(2025, 9, 1),
		CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "SEO optimization and technical audit", Quantity: 100, Unit: "ks", UnitPrice: 1800000, VATRatePercent: 21},
		},
	}, d(2025, 9, 12)); err != nil {
		return 0, fmt.Errorf("invoice 8: %w", err)
	}

	// 9. Marie — photo portfolio, 30h x 900, 0% VAT
	if err := createPaid(&domain.Invoice{
		CustomerID: ids["marie"], IssueDate: d(2025, 10, 1), DueDate: d(2025, 10, 15), DeliveryDate: d(2025, 10, 1),
		CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "Photography portfolio website", Quantity: 3000, Unit: "hod", UnitPrice: 90000, VATRatePercent: 0},
		},
	}, d(2025, 10, 14)); err != nil {
		return 0, fmt.Errorf("invoice 9: %w", err)
	}

	// 10. Blesk — proforma design collab, ~50K + VAT
	if err := createPaid(&domain.Invoice{
		CustomerID: ids["blesk"], IssueDate: d(2025, 11, 1), DueDate: d(2025, 11, 15), DeliveryDate: d(2025, 11, 1),
		Type: domain.InvoiceTypeProforma, CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "Design collaboration - UI/UX for client project", Quantity: 100, Unit: "ks", UnitPrice: 5000000, VATRatePercent: 21},
		},
	}, d(2025, 11, 10)); err != nil {
		return 0, fmt.Errorf("invoice 10: %w", err)
	}

	// --- 2026: mixed statuses (8 invoices) ---

	// 11. Kava Digital — annual maintenance, paid
	if err := createPaid(&domain.Invoice{
		CustomerID: ids["kava"], IssueDate: d(2026, 1, 2), DueDate: d(2026, 2, 1), DeliveryDate: d(2026, 1, 2),
		CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "Annual maintenance contract 2026", Quantity: 100, Unit: "ks", UnitPrice: 12000000, VATRatePercent: 21},
		},
	}, d(2026, 1, 20)); err != nil {
		return 0, fmt.Errorf("invoice 11: %w", err)
	}

	// 12. FitLife — mobile app design, sent (unpaid)
	if err := createSent(&domain.Invoice{
		CustomerID: ids["fitlife"], IssueDate: d(2026, 2, 1), DueDate: d(2026, 2, 22), DeliveryDate: d(2026, 2, 1),
		CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "Mobile app UI design - wireframes and mockups", Quantity: 6000, Unit: "hod", UnitPrice: 140000, VATRatePercent: 21},
		},
	}); err != nil {
		return 0, fmt.Errorf("invoice 12: %w", err)
	}

	// 13. Starter — monthly hosting, sent
	if err := createSent(&domain.Invoice{
		CustomerID: ids["starter"], IssueDate: d(2026, 3, 1), DueDate: d(2026, 3, 15), DeliveryDate: d(2026, 3, 1),
		CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "Monthly web hosting - March 2026", Quantity: 100, Unit: "ks", UnitPrice: 500000, VATRatePercent: 21},
		},
	}); err != nil {
		return 0, fmt.Errorf("invoice 13: %w", err)
	}

	// 14. Restaurace — menu update, overdue (due 2026-01-19)
	inv14 := &domain.Invoice{
		CustomerID: ids["restaurace"], IssueDate: d(2026, 1, 5), DueDate: d(2026, 1, 19), DeliveryDate: d(2026, 1, 5),
		CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "Restaurant menu website update", Quantity: 100, Unit: "ks", UnitPrice: 850000, VATRatePercent: 21},
		},
	}
	if err := svc.Create(ctx, inv14); err != nil {
		return 0, fmt.Errorf("invoice 14 create: %w", err)
	}
	if err := svc.MarkAsSent(ctx, inv14.ID); err != nil {
		return 0, fmt.Errorf("invoice 14 send: %w", err)
	}
	if err := repo.UpdateStatus(ctx, inv14.ID, domain.InvoiceStatusOverdue); err != nil {
		return 0, fmt.Errorf("invoice 14 overdue: %w", err)
	}
	count++

	// 15. DataFlow GmbH — data pipeline, draft
	if err := createDraft(&domain.Invoice{
		CustomerID: ids["dataflow"], IssueDate: d(2026, 3, 1), DueDate: d(2026, 3, 31), DeliveryDate: d(2026, 3, 1),
		CurrencyCode: domain.CurrencyEUR, ExchangeRate: domain.NewAmount(25, 10), PaymentMethod: "bank_transfer",
		Notes: "Reverse charge - VAT to be paid by the recipient according to Article 196 of the Council Directive 2006/112/EC.",
		Items: []domain.InvoiceItem{
			{Description: "Data pipeline - ETL development", Quantity: 8000, Unit: "hod", UnitPrice: 7200, VATRatePercent: 0},
		},
	}); err != nil {
		return 0, fmt.Errorf("invoice 15: %w", err)
	}

	// 16. Petr Svoboda — blog redesign, draft
	if err := createDraft(&domain.Invoice{
		CustomerID: ids["petr"], IssueDate: d(2026, 3, 5), DueDate: d(2026, 3, 12), DeliveryDate: d(2026, 3, 5),
		CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "Personal blog redesign", Quantity: 2500, Unit: "hod", UnitPrice: 90000, VATRatePercent: 0},
		},
	}); err != nil {
		return 0, fmt.Errorf("invoice 16: %w", err)
	}

	// 17. Blesk — proforma UI kit, sent
	if err := createSent(&domain.Invoice{
		CustomerID: ids["blesk"], IssueDate: d(2026, 2, 15), DueDate: d(2026, 3, 1), DeliveryDate: d(2026, 2, 15),
		Type: domain.InvoiceTypeProforma, CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "UI kit design - components and design system", Quantity: 100, Unit: "ks", UnitPrice: 6500000, VATRatePercent: 21},
		},
	}); err != nil {
		return 0, fmt.Errorf("invoice 17: %w", err)
	}

	// 18. Kava Digital — credit note on inv #11, -12K
	// Find invoice 11 to get its ID (it's the Kava 2026 annual maintenance).
	kavaID := ids["kava"]
	kavaInvoices, _, err := repo.List(ctx, domain.InvoiceFilter{CustomerID: &kavaID})
	if err != nil {
		return 0, fmt.Errorf("finding kava invoices: %w", err)
	}
	var inv11ID int64
	for _, inv := range kavaInvoices {
		if inv.IssueDate.Year() == 2026 && inv.Type == domain.InvoiceTypeRegular {
			inv11ID = inv.ID
			break
		}
	}
	if inv11ID == 0 {
		return 0, fmt.Errorf("could not find invoice 11 for credit note")
	}
	if err := createSent(&domain.Invoice{
		CustomerID: ids["kava"], IssueDate: d(2026, 2, 10), DueDate: d(2026, 2, 24), DeliveryDate: d(2026, 2, 10),
		Type: domain.InvoiceTypeCreditNote, RelatedInvoiceID: &inv11ID, RelationType: domain.RelationTypeCreditNote,
		CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{Description: "Credit note - overcharge correction on annual maintenance", Quantity: 100, Unit: "ks", UnitPrice: -1200000, VATRatePercent: 21},
		},
	}); err != nil {
		return 0, fmt.Errorf("invoice 18: %w", err)
	}

	return count, nil
}

func seedExpenses(ctx context.Context, svc *service.ExpenseService, ids contactIDs) (int, error) {
	d := func(year, month, day int) time.Time {
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}
	count := 0

	technoservID := ids["technoserv"]
	ucetnictviID := ids["ucetnictvi"]

	expenses := []domain.Expense{
		// --- 2025 (10 expenses) ---
		{Description: "JetBrains All Products Pack - annual license", Category: "software", IssueDate: d(2025, 1, 10),
			Amount: 549000, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 100, PaymentMethod: "card"},
		{Description: "GitHub Team - annual subscription", Category: "software", IssueDate: d(2025, 1, 15),
			Amount: 480000, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 100, PaymentMethod: "card"},
		{Description: "MacBook Pro 14\" M3 Pro", Category: "hardware", IssueDate: d(2025, 2, 20),
			Amount: 5299000, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 100, PaymentMethod: "bank_transfer"},
		{Description: "Dell U2723QE 4K Monitor", Category: "hardware", IssueDate: d(2025, 3, 5),
			Amount: 1299000, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 100, PaymentMethod: "bank_transfer",
			VendorID: &technoservID},
		{Description: "Vodafone business tariff - annual", Category: "telecom", IssueDate: d(2025, 4, 1),
			Amount: 720000, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 80, PaymentMethod: "bank_transfer"},
		{Description: "O2 internet - annual", Category: "telecom", IssueDate: d(2025, 4, 1),
			Amount: 840000, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 60, PaymentMethod: "bank_transfer"},
		{Description: "WebExpo 2025 - conference ticket", Category: "education", IssueDate: d(2025, 6, 10),
			Amount: 499000, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 100, PaymentMethod: "card"},
		{Description: "Accounting services Q1-Q2 2025", Category: "accounting", IssueDate: d(2025, 7, 1),
			Amount: 1200000, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 100, PaymentMethod: "bank_transfer",
			VendorID: &ucetnictviID},
		{Description: "Train Praha-Brno (client meeting)", Category: "travel", IssueDate: d(2025, 8, 15),
			Amount: 89000, VATRatePercent: 12, IsTaxDeductible: true, BusinessPercent: 100, PaymentMethod: "card"},
		{Description: "Adobe Creative Cloud - annual subscription", Category: "software", IssueDate: d(2025, 9, 1),
			Amount: 1558800, CurrencyCode: domain.CurrencyCZK, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 100, PaymentMethod: "card"},

		// --- 2026 (8 expenses) ---
		{Description: "JetBrains All Products Pack - annual renewal", Category: "software", IssueDate: d(2026, 1, 10),
			Amount: 569000, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 100, PaymentMethod: "card"},
		{Description: "GitHub Team - annual renewal", Category: "software", IssueDate: d(2026, 1, 15),
			Amount: 480000, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 100, PaymentMethod: "card"},
		{Description: "Ergonomic office chair", Category: "office_supplies", IssueDate: d(2026, 1, 25),
			Amount: 899000, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 100, PaymentMethod: "bank_transfer"},
		{Description: "Vodafone business tariff - annual renewal", Category: "telecom", IssueDate: d(2026, 2, 1),
			Amount: 720000, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 80, PaymentMethod: "bank_transfer"},
		{Description: "Accounting services Q3-Q4 2025", Category: "accounting", IssueDate: d(2026, 1, 5),
			Amount: 1200000, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 100, PaymentMethod: "bank_transfer",
			VendorID: &ucetnictviID},
		{Description: "React Advanced 2026 - conference ticket", Category: "education", IssueDate: d(2026, 2, 15),
			Amount: 1250000, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 100, PaymentMethod: "card"},
		{Description: "Coworking space - monthly pass", Category: "rent", IssueDate: d(2026, 3, 1),
			Amount: 450000, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 100, PaymentMethod: "bank_transfer"},
		{Description: "USB-C hub and cables", Category: "office_supplies", IssueDate: d(2026, 3, 10),
			Amount: 249000, VATRatePercent: 21, IsTaxDeductible: true, BusinessPercent: 100, PaymentMethod: "card"},
	}

	for i, exp := range expenses {
		e := exp
		if err := svc.Create(ctx, &e); err != nil {
			return 0, fmt.Errorf("expense %d: %w", i+1, err)
		}
		count++
	}
	return count, nil
}

func seedRecurringInvoices(ctx context.Context, svc *service.RecurringInvoiceService, ids contactIDs) (int, error) {
	templates := []domain.RecurringInvoice{
		{
			Name: "Monthly hosting - Starter", CustomerID: ids["starter"],
			Frequency: domain.FrequencyMonthly, NextIssueDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer", IsActive: true,
			Items: []domain.RecurringInvoiceItem{
				{Description: "Monthly web hosting", Quantity: 100, Unit: "ks", UnitPrice: 500000, VATRatePercent: 21},
			},
		},
		{
			Name: "Quarterly maintenance - Kava Digital", CustomerID: ids["kava"],
			Frequency: domain.FrequencyQuarterly, NextIssueDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			CurrencyCode: domain.CurrencyCZK, PaymentMethod: "bank_transfer", IsActive: true,
			Items: []domain.RecurringInvoiceItem{
				{Description: "E-shop quarterly maintenance", Quantity: 2000, Unit: "hod", UnitPrice: 150000, VATRatePercent: 21},
			},
		},
	}

	for i, t := range templates {
		tmpl := t
		if err := svc.Create(ctx, &tmpl); err != nil {
			return 0, fmt.Errorf("recurring invoice %d: %w", i+1, err)
		}
	}
	return len(templates), nil
}

func seedRecurringExpenses(ctx context.Context, svc *service.RecurringExpenseService) (int, error) {
	templates := []domain.RecurringExpense{
		{
			Name: "Coworking space", Category: "rent", Description: "Monthly coworking space pass",
			Amount: 450000, CurrencyCode: domain.CurrencyCZK, VATRatePercent: 21,
			IsTaxDeductible: true, BusinessPercent: 100, PaymentMethod: "bank_transfer",
			Frequency: domain.FrequencyMonthly, NextIssueDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			IsActive: true,
		},
	}

	for i, t := range templates {
		tmpl := t
		if err := svc.Create(ctx, &tmpl); err != nil {
			return 0, fmt.Errorf("recurring expense %d: %w", i+1, err)
		}
	}
	return len(templates), nil
}

func seedTaxYearSettings(ctx context.Context, svc *service.TaxYearSettingsService) error {
	// 2025: actual expenses (flat rate 0%), with monthly prepayments.
	prepayments2025 := make([]domain.TaxPrepayment, 12)
	for m := 1; m <= 12; m++ {
		prepayments2025[m-1] = domain.TaxPrepayment{
			Year:         2025,
			Month:        m,
			TaxAmount:    domain.Amount(450000),  // 4 500 CZK DPFO
			SocialAmount: domain.Amount(320000),  // 3 200 CZK CSSZ
			HealthAmount: domain.Amount(280000),  // 2 800 CZK ZP
		}
	}
	if err := svc.Save(ctx, 2025, 0, prepayments2025); err != nil {
		return fmt.Errorf("saving 2025 settings: %w", err)
	}

	// 2026: actual expenses, higher prepayments (grew business).
	prepayments2026 := make([]domain.TaxPrepayment, 12)
	for m := 1; m <= 12; m++ {
		prepayments2026[m-1] = domain.TaxPrepayment{
			Year:         2026,
			Month:        m,
			TaxAmount:    domain.Amount(550000),  // 5 500 CZK DPFO
			SocialAmount: domain.Amount(380000),  // 3 800 CZK CSSZ
			HealthAmount: domain.Amount(320000),  // 3 200 CZK ZP
		}
	}
	return svc.Save(ctx, 2026, 0, prepayments2026)
}

func seedTaxCredits(ctx context.Context, svc *service.TaxCreditsService) error {
	// 2025 tax credits for Jan Novak.

	// Spouse credit — wife on maternity, income under 68k limit.
	if err := svc.UpsertSpouse(ctx, &domain.TaxSpouseCredit{
		Year:              2025,
		SpouseName:        "Eva Novakova",
		SpouseBirthNumber: "9055011234",
		SpouseIncome:      domain.Amount(4500000), // 45 000 CZK (under 68 000 limit)
		SpouseZTP:         false,
		MonthsClaimed:     12,
	}); err != nil {
		return fmt.Errorf("spouse credit: %w", err)
	}

	// Two children.
	if err := svc.CreateChild(ctx, &domain.TaxChildCredit{
		Year:          2025,
		ChildName:     "Tomas Novak",
		BirthNumber:   "1905151234",
		ChildOrder:    1,
		MonthsClaimed: 12,
		ZTP:           false,
	}); err != nil {
		return fmt.Errorf("child 1 credit: %w", err)
	}
	if err := svc.CreateChild(ctx, &domain.TaxChildCredit{
		Year:          2025,
		ChildName:     "Anna Novakova",
		BirthNumber:   "2155201234",
		ChildOrder:    2,
		MonthsClaimed: 12,
		ZTP:           false,
	}); err != nil {
		return fmt.Errorf("child 2 credit: %w", err)
	}

	// Personal credits — taxpayer basic (automatic), no disability/student.
	if err := svc.UpsertPersonal(ctx, &domain.TaxPersonalCredits{
		Year:            2025,
		IsStudent:       false,
		StudentMonths:   0,
		DisabilityLevel: 0,
	}); err != nil {
		return fmt.Errorf("personal credits: %w", err)
	}

	// Deductions — mortgage interest and pension savings.
	if err := svc.CreateDeduction(ctx, &domain.TaxDeduction{
		Year:          2025,
		Category:      "mortgage",
		Description:   "Mortgage interest - flat in Praha 2",
		ClaimedAmount: domain.Amount(8500000), // 85 000 CZK
	}); err != nil {
		return fmt.Errorf("mortgage deduction: %w", err)
	}
	if err := svc.CreateDeduction(ctx, &domain.TaxDeduction{
		Year:          2025,
		Category:      "pension",
		Description:   "Supplementary pension savings (doplnkove penzijni sporeni)",
		ClaimedAmount: domain.Amount(2400000), // 24 000 CZK
	}); err != nil {
		return fmt.Errorf("pension deduction: %w", err)
	}
	if err := svc.CreateDeduction(ctx, &domain.TaxDeduction{
		Year:          2025,
		Category:      "life_insurance",
		Description:   "Life insurance (zivotni pojisteni)",
		ClaimedAmount: domain.Amount(1200000), // 12 000 CZK
	}); err != nil {
		return fmt.Errorf("life insurance deduction: %w", err)
	}

	return nil
}

func seedVATReturns(ctx context.Context, svc *service.VATReturnService) error {
	// Quarterly VAT returns for 2025 — all filed.
	quarters := []struct {
		quarter int
		status  string
	}{
		{1, domain.FilingStatusFiled},
		{2, domain.FilingStatusFiled},
		{3, domain.FilingStatusFiled},
		{4, domain.FilingStatusFiled},
	}
	for _, q := range quarters {
		vr := &domain.VATReturn{
			Period:     domain.TaxPeriod{Year: 2025, Quarter: q.quarter},
			FilingType: domain.FilingTypeRegular,
			Status:     q.status,
		}
		if err := svc.Create(ctx, vr); err != nil {
			return fmt.Errorf("VAT return Q%d/2025: %w", q.quarter, err)
		}
	}

	// Q1 2026 — draft (in progress).
	vr := &domain.VATReturn{
		Period:     domain.TaxPeriod{Year: 2026, Quarter: 1},
		FilingType: domain.FilingTypeRegular,
	}
	return svc.Create(ctx, vr)
}

func seedVATControlStatements(ctx context.Context, svc *service.VATControlStatementService) error {
	// Monthly control statements for 2025 — all filed.
	for m := 1; m <= 12; m++ {
		cs := &domain.VATControlStatement{
			Period:     domain.TaxPeriod{Year: 2025, Month: m},
			FilingType: domain.FilingTypeRegular,
			Status:     domain.FilingStatusFiled,
		}
		if err := svc.Create(ctx, cs); err != nil {
			return fmt.Errorf("control statement %d/2025: %w", m, err)
		}
	}

	// January and February 2026 — filed. March — draft.
	for m := 1; m <= 3; m++ {
		status := domain.FilingStatusFiled
		if m == 3 {
			status = domain.FilingStatusDraft
		}
		cs := &domain.VATControlStatement{
			Period:     domain.TaxPeriod{Year: 2026, Month: m},
			FilingType: domain.FilingTypeRegular,
			Status:     status,
		}
		if err := svc.Create(ctx, cs); err != nil {
			return fmt.Errorf("control statement %d/2026: %w", m, err)
		}
	}
	return nil
}

func seedVIESSummaries(ctx context.Context, svc *service.VIESSummaryService) error {
	// Q3 2025 — DataFlow GmbH invoice was in August (reverse charge EU service).
	vs := &domain.VIESSummary{
		Period:     domain.TaxPeriod{Year: 2025, Quarter: 3},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusFiled,
	}
	return svc.Create(ctx, vs)
}

func seedIncomeTaxReturn(ctx context.Context, svc *service.IncomeTaxReturnService) error {
	// 2025 income tax return — filed.
	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusFiled,
	}
	return svc.Create(ctx, itr)
}

func seedSocialInsurance(ctx context.Context, svc *service.SocialInsuranceService) error {
	// 2025 social insurance overview — filed.
	sio := &domain.SocialInsuranceOverview{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusFiled,
	}
	return svc.Create(ctx, sio)
}

func seedHealthInsurance(ctx context.Context, svc *service.HealthInsuranceService) error {
	// 2025 health insurance overview — filed.
	hio := &domain.HealthInsuranceOverview{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusFiled,
	}
	return svc.Create(ctx, hio)
}

func seedInvestments(ctx context.Context, svc *service.InvestmentIncomeService) error {
	d := func(year, month, day int) time.Time {
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	}

	// Capital income (§8) — dividends and interest from 2025.
	capitalEntries := []domain.CapitalIncomeEntry{
		{
			Year: 2025, Category: "dividend_cz",
			Description: "Dividend - CEZ a.s.", IncomeDate: d(2025, 6, 15),
			GrossAmount: domain.Amount(1500000), WithheldTaxCZ: domain.Amount(225000), // 15% tax
			CountryCode: "CZ", NeedsDeclaring: false,
		},
		{
			Year: 2025, Category: "dividend_foreign",
			Description: "Dividend - Apple Inc. (AAPL)", IncomeDate: d(2025, 8, 10),
			GrossAmount: domain.Amount(850000), WithheldTaxForeign: domain.Amount(127500), // 15% US withholding
			CountryCode: "US", NeedsDeclaring: true,
		},
		{
			Year: 2025, Category: "interest",
			Description: "Interest - Portu savings account", IncomeDate: d(2025, 12, 31),
			GrossAmount: domain.Amount(320000), WithheldTaxCZ: domain.Amount(48000), // 15% tax
			CountryCode: "CZ", NeedsDeclaring: false,
		},
		{
			Year: 2025, Category: "dividend_cz",
			Description: "Dividend - Komercni banka a.s.", IncomeDate: d(2025, 5, 20),
			GrossAmount: domain.Amount(680000), WithheldTaxCZ: domain.Amount(102000),
			CountryCode: "CZ", NeedsDeclaring: false,
		},
	}
	for i, e := range capitalEntries {
		entry := e
		if err := svc.CreateCapitalEntry(ctx, &entry); err != nil {
			return fmt.Errorf("capital entry %d: %w", i+1, err)
		}
	}

	// Security transactions (§10) — ETF buys and a sell.
	transactions := []domain.SecurityTransaction{
		// Buy VWCE ETF — March 2022 (long-term hold, time-test exempt).
		{
			Year: 2022, AssetType: "etf", AssetName: "Vanguard FTSE All-World UCITS ETF",
			ISIN: "IE00BK5BQT80", TransactionType: "buy", TransactionDate: d(2022, 3, 15),
			Quantity: 500000, UnitPrice: domain.Amount(9500), TotalAmount: domain.Amount(4750000),
			Fees: domain.Amount(500), CurrencyCode: domain.CurrencyEUR, ExchangeRate: 249500,
		},
		// Buy VWCE ETF — September 2023.
		{
			Year: 2023, AssetType: "etf", AssetName: "Vanguard FTSE All-World UCITS ETF",
			ISIN: "IE00BK5BQT80", TransactionType: "buy", TransactionDate: d(2023, 9, 10),
			Quantity: 300000, UnitPrice: domain.Amount(10200), TotalAmount: domain.Amount(3060000),
			Fees: domain.Amount(500), CurrencyCode: domain.CurrencyEUR, ExchangeRate: 245000,
		},
		// Sell VWCE ETF — October 2025 (partial, 30 units, time-test applies to March 2022 lot).
		{
			Year: 2025, AssetType: "etf", AssetName: "Vanguard FTSE All-World UCITS ETF",
			ISIN: "IE00BK5BQT80", TransactionType: "sell", TransactionDate: d(2025, 10, 5),
			Quantity: 300000, UnitPrice: domain.Amount(11800), TotalAmount: domain.Amount(3540000),
			Fees: domain.Amount(500), CurrencyCode: domain.CurrencyEUR, ExchangeRate: 252000,
		},
		// Buy Bitcoin — January 2024.
		{
			Year: 2024, AssetType: "crypto", AssetName: "Bitcoin",
			TransactionType: "buy", TransactionDate: d(2024, 1, 20),
			Quantity: 5000, UnitPrice: domain.Amount(95000000), TotalAmount: domain.Amount(47500000),
			Fees: domain.Amount(50000), CurrencyCode: domain.CurrencyCZK,
		},
		// Sell Bitcoin — November 2025 (under 3 years, taxable).
		{
			Year: 2025, AssetType: "crypto", AssetName: "Bitcoin",
			TransactionType: "sell", TransactionDate: d(2025, 11, 15),
			Quantity: 2500, UnitPrice: domain.Amount(210000000), TotalAmount: domain.Amount(52500000),
			Fees: domain.Amount(75000), CurrencyCode: domain.CurrencyCZK,
		},
	}
	for i, t := range transactions {
		tx := t
		if err := svc.CreateSecurityTransaction(ctx, &tx); err != nil {
			return fmt.Errorf("security transaction %d: %w", i+1, err)
		}
	}

	// Recalculate FIFO for 2025 sell transactions.
	if err := svc.RecalculateFIFO(ctx, 2025); err != nil {
		return fmt.Errorf("recalculating FIFO: %w", err)
	}

	return nil
}

func seedSettings(ctx context.Context, svc *service.SettingsService) error {
	settings := map[string]string{
		service.SettingCompanyName:   "Jan Novak",
		service.SettingICO:           "01234567",
		service.SettingDIC:           "CZ8501011234",
		service.SettingVATRegistered: "true",
		service.SettingStreet:        "Korunni 65",
		service.SettingCity:          "Praha 2",
		service.SettingZIP:           "12000",
		service.SettingEmail:         "jan.novak@example.cz",
		service.SettingPhone:         "+420 777 888 999",
		service.SettingBankAccount:   "1234567890",
		service.SettingBankCode:      "0100",
		service.SettingIBAN:          "CZ6508000000001234567890",
		service.SettingSWIFT:         "KOMBCZPPXXX",

		service.SettingEmailSubjectTpl: "Faktura {{.InvoiceNumber}} - Jan Novak",
		service.SettingEmailBodyTpl:    "Dobry den,\n\nv priloze zasilam fakturu c. {{.InvoiceNumber}} na castku {{.TotalAmount}} {{.Currency}}.\n\nDekuji za spolupráci.\n\nJan Novak",
		service.SettingEmailAttachPDF:   "true",
		service.SettingEmailAttachISDOC: "true",

		service.SettingFinancniUradCode: "451",
		service.SettingCSSZCode:         "11",
		service.SettingHealthInsCode:    "111",
	}
	return svc.SetBulk(ctx, settings)
}
