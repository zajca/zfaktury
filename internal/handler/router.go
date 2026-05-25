package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/zajca/zfaktury/internal/isdoc"
	"github.com/zajca/zfaktury/internal/pdf"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/service/cnb"
	"github.com/zajca/zfaktury/internal/service/email"
)

// RouterConfig holds configuration for the HTTP router.
type RouterConfig struct {
	// DevMode enables CORS for all origins (for Vite dev server).
	DevMode bool
	// DataDir is the application data directory for file storage.
	DataDir string
}

// NewRouter creates a chi router with all API routes mounted.
//
// Routes are split into two tiers:
//
//  1. Global tier — endpoints that are not scoped to a single company:
//     /health, /api/v1/companies (CRUD), /api/v1/audit-log, /api/v1/backups,
//     and the CNB exchange-rate proxy (a public, company-agnostic data feed).
//
//  2. Per-company tier — everything else, mounted under
//     /api/v1/companies/{companyID}/... and guarded by the WithCompany
//     middleware which resolves and validates the {companyID} path param
//     before delegating to the inner handler.
//
// Per-company handler internals still ignore the resolved company id;
// threading it through repositories/services/handlers is the job of the
// follow-up tasks T20-T22.
func NewRouter(
	companySvc *service.CompanyService,
	contactSvc *service.ContactService,
	invoiceSvc *service.InvoiceService,
	expenseSvc *service.ExpenseService,
	settingsSvc *service.SettingsService,
	sequenceSvc *service.SequenceService,
	categorySvc *service.CategoryService,
	documentSvc *service.DocumentService,
	recurringInvoiceSvc *service.RecurringInvoiceService,
	recurringExpenseSvc *service.RecurringExpenseService,
	ocrSvc *service.OCRService,
	importSvc *service.ImportService,
	overdueSvc *service.OverdueService,
	reminderSvc *service.ReminderService,
	cnbClient *cnb.Client,
	pdfGen *pdf.InvoicePDFGenerator,
	isdocGen *isdoc.ISDOCGenerator,
	vatReturnSvc *service.VATReturnService,
	vatControlSvc *service.VATControlStatementService,
	viesSvc *service.VIESSummaryService,
	incomeTaxSvc *service.IncomeTaxReturnService,
	socialInsuranceSvc *service.SocialInsuranceService,
	healthInsuranceSvc *service.HealthInsuranceService,
	taxYearSettingsSvc *service.TaxYearSettingsService,
	taxCreditsSvc *service.TaxCreditsService,
	taxDeductionDocSvc *service.TaxDeductionDocumentService,
	taxExtractionSvc *service.TaxDocumentExtractionService,
	investmentIncomeSvc *service.InvestmentIncomeService,
	investmentDocSvc *service.InvestmentDocumentService,
	investmentExtractionSvc *service.InvestmentExtractionService,
	invDocumentSvc *service.InvoiceDocumentService,
	fakturoidImportSvc *service.FakturoidImportService,
	dashboardSvc *service.DashboardService,
	reportSvc *service.ReportService,
	taxCalendarSvc *service.TaxCalendarService,
	emailSender *email.EmailSender,
	auditSvc *service.AuditService,
	backupSvc *service.BackupService,
	cfg RouterConfig,
) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(securityHeadersMiddleware)
	r.Use(slogMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(adaptiveTimeoutMiddleware)

	// CORS middleware for dev mode.
	if cfg.DevMode {
		r.Use(corsMiddleware)
	}

	// API routes with JSON content type.
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(jsonContentTypeMiddleware)

		// Construct all per-company handlers up-front. Their internals still
		// ignore the company id resolved by the WithCompany middleware — that
		// gets threaded through repos/services/handlers in T20-T22.
		contactHandler := NewContactHandler(contactSvc)
		invoiceHandler := NewInvoiceHandler(invoiceSvc, settingsSvc, pdfGen, isdocGen)
		expenseHandler := NewExpenseHandler(expenseSvc)
		categoryHandler := NewCategoryHandler(categorySvc)
		settingsHandler := NewSettingsHandler(settingsSvc)
		sequenceHandler := NewSequenceHandler(sequenceSvc)
		documentHandler := NewDocumentHandler(documentSvc)
		recurringInvoiceHandler := NewRecurringInvoiceHandler(recurringInvoiceSvc)
		recurringExpenseHandler := NewRecurringExpenseHandler(recurringExpenseSvc)

		var invDocHandler *InvoiceDocumentHandler
		if invDocumentSvc != nil {
			invDocHandler = NewInvoiceDocumentHandler(invDocumentSvc)
		}

		// -----------------------------------------------------------------
		// Global tier — endpoints not scoped to a single company.
		// -----------------------------------------------------------------

		// Company CRUD (the global registry of companies themselves).
		// List + Create are registered at /companies; Get/Update/Delete on a
		// specific company live INSIDE the per-company Route below, behind
		// WithCompany, because chi cannot disambiguate /companies/{id} (here)
		// from /companies/{companyID}/* (subrouter) when both are wildcards
		// matching /companies/1 exactly.
		companyHandler := NewCompanyHandler(companySvc)
		api.Get("/companies", companyHandler.List)
		api.Post("/companies", companyHandler.Create)

		// Audit log is cross-company by design (admin/diagnostic view).
		auditLogHandler := NewAuditLogHandler(auditSvc)
		api.Mount("/audit-log", auditLogHandler.Routes())

		// Backups operate at the database level, not per-company.
		if backupSvc != nil {
			backupHandler := NewBackupHandler(backupSvc)
			api.Mount("/backups", backupHandler.Routes())
		}

		// CNB exchange-rate is a public data feed and not scoped to a company.
		if cnbClient != nil {
			exchangeHandler := NewExchangeHandler(cnbClient)
			api.Mount("/exchange-rate", exchangeHandler.Routes())
		}

		// -----------------------------------------------------------------
		// Per-company tier — mounted under /companies/{companyID}/... with
		// the WithCompany middleware resolving and validating the path param.
		// -----------------------------------------------------------------
		api.Route("/companies/{companyID}", func(co chi.Router) {
			co.Use(WithCompany(companySvc))

			// Company resource itself (Get/Update/Delete by ID).
			// WithCompany already validated the company exists and is not soft-deleted.
			co.Get("/", companyHandler.Get)
			co.Put("/", companyHandler.Update)
			co.Delete("/", companyHandler.Delete)

			co.Mount("/contacts", contactHandler.Routes())

			// Use Route (not Mount) for /invoices so additional sub-routes can
			// be registered in the same group without being swallowed by
			// Mount's wildcard.
			co.Route("/invoices", func(inv chi.Router) {
				// Core invoice CRUD + actions.
				inv.Post("/", invoiceHandler.Create)
				inv.Get("/", invoiceHandler.List)
				inv.Get("/{id}", invoiceHandler.GetByID)
				inv.Put("/{id}", invoiceHandler.Update)
				inv.Delete("/{id}", invoiceHandler.Delete)
				inv.Post("/{id}/send", invoiceHandler.MarkAsSent)
				inv.Post("/{id}/mark-paid", invoiceHandler.MarkAsPaid)
				inv.Post("/{id}/duplicate", invoiceHandler.Duplicate)
				inv.Post("/{id}/settle", invoiceHandler.SettleProforma)
				inv.Post("/{id}/credit-note", invoiceHandler.CreateCreditNote)
				inv.Get("/{id}/pdf", invoiceHandler.DownloadPDF)
				inv.Get("/{id}/qr", invoiceHandler.QRPayment)
				inv.Get("/{id}/isdoc", invoiceHandler.ExportISDOC)
				inv.Post("/export/isdoc", invoiceHandler.ExportISDOCBatch)

				// Invoice document routes.
				if invDocHandler != nil {
					inv.Get("/{id}/documents", invDocHandler.ListByInvoice)
				}

				// Status history & overdue (conditional).
				if overdueSvc != nil {
					statusHistoryHandler := NewStatusHistoryHandler(overdueSvc)
					inv.Get("/{id}/history", statusHistoryHandler.GetHistory)
					inv.Post("/check-overdue", statusHistoryHandler.CheckOverdue)
				}

				// Payment reminders (conditional).
				if reminderSvc != nil {
					reminderHandler := NewReminderHandler(reminderSvc)
					inv.Post("/{id}/remind", reminderHandler.SendReminder)
					inv.Get("/{id}/reminders", reminderHandler.ListReminders)
				}

				// Send invoice via email (always registered, checks SMTP at
				// runtime).
				emailHandler := NewEmailHandler(invoiceSvc, settingsSvc, pdfGen, isdocGen, emailSender)
				inv.Post("/{id}/send-email", emailHandler.SendEmail)
			})

			// Email defaults (frontend pre-population).
			co.Get("/email/defaults", NewEmailHandler(invoiceSvc, settingsSvc, pdfGen, isdocGen, emailSender).GetDefaults)

			// Use Route (not Mount) for /expenses so the import sub-route is
			// not swallowed by Mount's wildcard. Document routes also live
			// here since Route captures the entire /expenses/* prefix.
			co.Route("/expenses", func(exp chi.Router) {
				exp.Post("/", expenseHandler.Create)
				exp.Get("/", expenseHandler.List)
				exp.Post("/review", expenseHandler.MarkTaxReviewed)
				exp.Post("/unreview", expenseHandler.UnmarkTaxReviewed)
				exp.Get("/{id}", expenseHandler.GetByID)
				exp.Put("/{id}", expenseHandler.Update)
				exp.Delete("/{id}", expenseHandler.Delete)

				// Document routes (moved from documentHandler.Routes()).
				exp.Post("/{id}/documents", documentHandler.Upload)
				exp.Get("/{id}/documents", documentHandler.ListByExpense)

				if importSvc != nil {
					importHandler := NewImportHandler(importSvc)
					exp.Post("/import", importHandler.Import)
				}
			})

			co.Mount("/expense-categories", categoryHandler.Routes())

			pdfSettingsHandler := NewPDFSettingsHandler(settingsSvc, invoiceSvc, pdfGen, cfg.DataDir)
			co.Route("/settings", func(sr chi.Router) {
				sr.Get("/", settingsHandler.GetAll)
				sr.Put("/", settingsHandler.Update)
				sr.Get("/pdf", pdfSettingsHandler.GetPDFSettings)
				sr.Put("/pdf", pdfSettingsHandler.UpdatePDFSettings)
				sr.Post("/logo", pdfSettingsHandler.UploadLogo)
				sr.Get("/logo", pdfSettingsHandler.GetLogo)
				sr.Delete("/logo", pdfSettingsHandler.DeleteLogo)
				sr.Get("/pdf-preview", pdfSettingsHandler.PreviewPDF)
			})

			co.Mount("/invoice-sequences", sequenceHandler.Routes())
			co.Mount("/", documentHandler.Routes())
			if invDocHandler != nil {
				co.Get("/invoice-documents/{id}", invDocHandler.GetByID)
				co.Get("/invoice-documents/{id}/download", invDocHandler.Download)
				co.Delete("/invoice-documents/{id}", invDocHandler.Delete)
			}
			co.Mount("/recurring-invoices", recurringInvoiceHandler.Routes())
			co.Mount("/recurring-expenses", recurringExpenseHandler.Routes())

			if ocrSvc != nil {
				ocrHandler := NewOCRHandler(ocrSvc)
				co.Post("/documents/{id}/ocr", ocrHandler.ProcessDocument)
			}

			vatReturnHandler := NewVATReturnHandler(vatReturnSvc)
			co.Mount("/vat-returns", vatReturnHandler.Routes())

			vatControlHandler := NewVATControlStatementHandler(vatControlSvc, settingsSvc)
			co.Mount("/vat-control-statements", vatControlHandler.Routes())

			viesHandler := NewVIESHandler(viesSvc, settingsSvc)
			co.Mount("/vies-summaries", viesHandler.Routes())

			incomeTaxHandler := NewIncomeTaxHandler(incomeTaxSvc)
			co.Mount("/income-tax-returns", incomeTaxHandler.Routes())

			socialInsuranceHandler := NewSocialInsuranceHandler(socialInsuranceSvc)
			co.Mount("/social-insurance", socialInsuranceHandler.Routes())

			healthInsuranceHandler := NewHealthInsuranceHandler(healthInsuranceSvc)
			co.Mount("/health-insurance", healthInsuranceHandler.Routes())

			taxYearSettingsHandler := NewTaxYearSettingsHandler(taxYearSettingsSvc)
			co.Mount("/tax-year-settings", taxYearSettingsHandler.Routes())

			co.Get("/tax-constants/{year}", handleGetTaxConstants)

			taxCreditsHandler := NewTaxCreditsHandler(taxCreditsSvc)
			co.Mount("/tax-credits", taxCreditsHandler.Routes())

			taxDeductionsHandler := NewTaxDeductionsHandler(taxCreditsSvc, taxDeductionDocSvc, taxExtractionSvc)
			co.Mount("/tax-deductions", taxDeductionsHandler.Routes())
			co.Mount("/tax-deduction-documents", taxDeductionsHandler.DocumentRoutes())

			if investmentIncomeSvc != nil {
				investmentHandler := NewInvestmentIncomeHandler(investmentIncomeSvc, investmentDocSvc, investmentExtractionSvc)
				co.Mount("/investments", investmentHandler.Routes())
			}

			fakturoidHandler := NewFakturoidHandler(fakturoidImportSvc)
			co.Mount("/import/fakturoid", fakturoidHandler.Routes())

			// Dashboard.
			dashboardHandler := NewDashboardHandler(dashboardSvc)
			co.Get("/dashboard", dashboardHandler.GetDashboard)

			// Reports.
			reportHandler := NewReportHandler(reportSvc)
			co.Get("/reports/revenue", reportHandler.Revenue)
			co.Get("/reports/expenses", reportHandler.Expenses)
			co.Get("/reports/top-customers", reportHandler.TopCustomers)
			co.Get("/reports/profit-loss", reportHandler.ProfitLoss)

			// Tax calendar.
			taxCalendarHandler := NewTaxCalendarHandler(taxCalendarSvc)
			co.Get("/reports/tax-calendar", taxCalendarHandler.GetCalendar)

			// CSV export.
			exportHandler := NewExportHandler(invoiceSvc, expenseSvc)
			co.Get("/export/invoices", exportHandler.ExportInvoices)
			co.Get("/export/expenses", exportHandler.ExportExpenses)
		})
	})

	// Health check endpoint.
	r.Get("/health", healthHandler)

	return r
}

// healthHandler returns a simple health check response.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// slogMiddleware logs each HTTP request using slog.
func slogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"bytes", ww.BytesWritten(),
		)
	})
}

// jsonContentTypeMiddleware sets the Content-Type header to application/json for API responses.
func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// securityHeadersMiddleware adds basic security headers to all responses.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// adaptiveTimeoutMiddleware applies a longer timeout for long-running import
// endpoints and the default 30s timeout for everything else.
//
// The Fakturoid import is now per-company at
// /api/v1/companies/{companyID}/import/fakturoid; matching is done with
// strings.Contains because the company-id segment is a path variable.
func adaptiveTimeoutMiddleware(next http.Handler) http.Handler {
	longTimeout := middleware.Timeout(10 * time.Minute)
	shortTimeout := middleware.Timeout(30 * time.Second)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/import/fakturoid") {
			longTimeout(next).ServeHTTP(w, r)
		} else {
			shortTimeout(next).ServeHTTP(w, r)
		}
	})
}

// corsMiddleware adds permissive CORS headers for development.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
