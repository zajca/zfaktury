package handler

import (
	"log/slog"
	"net/http"
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
}

// NewRouter creates a chi router with all API routes mounted.
func NewRouter(
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
	cfg RouterConfig,
) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(securityHeadersMiddleware)
	r.Use(slogMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// CORS middleware for dev mode.
	if cfg.DevMode {
		r.Use(corsMiddleware)
	}

	// API routes with JSON content type.
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(jsonContentTypeMiddleware)

		contactHandler := NewContactHandler(contactSvc)
		invoiceHandler := NewInvoiceHandler(invoiceSvc, settingsSvc, pdfGen, isdocGen)
		expenseHandler := NewExpenseHandler(expenseSvc)
		categoryHandler := NewCategoryHandler(categorySvc)
		settingsHandler := NewSettingsHandler(settingsSvc)
		sequenceHandler := NewSequenceHandler(sequenceSvc)
		documentHandler := NewDocumentHandler(documentSvc)
		recurringInvoiceHandler := NewRecurringInvoiceHandler(recurringInvoiceSvc)
		recurringExpenseHandler := NewRecurringExpenseHandler(recurringExpenseSvc)

		api.Mount("/contacts", contactHandler.Routes())
		// Use Route (not Mount) for /invoices so additional sub-routes can be
		// registered in the same group without being swallowed by Mount's wildcard.
		var invDocHandler *InvoiceDocumentHandler
		if invDocumentSvc != nil {
			invDocHandler = NewInvoiceDocumentHandler(invDocumentSvc)
		}

		api.Route("/invoices", func(inv chi.Router) {
			// Core invoice CRUD + actions
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

			// Invoice document routes
			if invDocHandler != nil {
				inv.Get("/{id}/documents", invDocHandler.ListByInvoice)
			}

			// Status history & overdue (conditional)
			if overdueSvc != nil {
				statusHistoryHandler := NewStatusHistoryHandler(overdueSvc)
				inv.Get("/{id}/history", statusHistoryHandler.GetHistory)
				inv.Post("/check-overdue", statusHistoryHandler.CheckOverdue)
			}

			// Payment reminders (conditional)
			if reminderSvc != nil {
				reminderHandler := NewReminderHandler(reminderSvc)
				inv.Post("/{id}/remind", reminderHandler.SendReminder)
				inv.Get("/{id}/reminders", reminderHandler.ListReminders)
			}

			// Send invoice via email (always registered, checks SMTP at runtime)
			emailHandler := NewEmailHandler(invoiceSvc, settingsSvc, pdfGen, isdocGen, emailSender)
			inv.Post("/{id}/send-email", emailHandler.SendEmail)
		})

		// Email defaults (always available for frontend pre-population)
		api.Get("/email/defaults", NewEmailHandler(invoiceSvc, settingsSvc, pdfGen, isdocGen, emailSender).GetDefaults)

		// Use Route (not Mount) for /expenses so the import sub-route
		// is not swallowed by Mount's wildcard. Document routes must also
		// live here since Route captures the entire /expenses/* prefix.
		api.Route("/expenses", func(exp chi.Router) {
			exp.Post("/", expenseHandler.Create)
			exp.Get("/", expenseHandler.List)
			exp.Post("/review", expenseHandler.MarkTaxReviewed)
			exp.Post("/unreview", expenseHandler.UnmarkTaxReviewed)
			exp.Get("/{id}", expenseHandler.GetByID)
			exp.Put("/{id}", expenseHandler.Update)
			exp.Delete("/{id}", expenseHandler.Delete)

			// Document routes (moved from documentHandler.Routes())
			exp.Post("/{id}/documents", documentHandler.Upload)
			exp.Get("/{id}/documents", documentHandler.ListByExpense)

			if importSvc != nil {
				importHandler := NewImportHandler(importSvc)
				exp.Post("/import", importHandler.Import)
			}
		})
		api.Mount("/expense-categories", categoryHandler.Routes())
		api.Mount("/settings", settingsHandler.Routes())
		api.Mount("/invoice-sequences", sequenceHandler.Routes())
		api.Mount("/", documentHandler.Routes())
		if invDocHandler != nil {
			api.Get("/invoice-documents/{id}", invDocHandler.GetByID)
			api.Get("/invoice-documents/{id}/download", invDocHandler.Download)
			api.Delete("/invoice-documents/{id}", invDocHandler.Delete)
		}
		api.Mount("/recurring-invoices", recurringInvoiceHandler.Routes())
		api.Mount("/recurring-expenses", recurringExpenseHandler.Routes())

		if ocrSvc != nil {
			ocrHandler := NewOCRHandler(ocrSvc)
			api.Post("/documents/{id}/ocr", ocrHandler.ProcessDocument)
		}

		if cnbClient != nil {
			exchangeHandler := NewExchangeHandler(cnbClient)
			api.Mount("/exchange-rate", exchangeHandler.Routes())
		}

		vatReturnHandler := NewVATReturnHandler(vatReturnSvc)
		api.Mount("/vat-returns", vatReturnHandler.Routes())

		vatControlHandler := NewVATControlStatementHandler(vatControlSvc, settingsSvc)
		api.Mount("/vat-control-statements", vatControlHandler.Routes())

		viesHandler := NewVIESHandler(viesSvc, settingsSvc)
		api.Mount("/vies-summaries", viesHandler.Routes())

		incomeTaxHandler := NewIncomeTaxHandler(incomeTaxSvc)
		api.Mount("/income-tax-returns", incomeTaxHandler.Routes())

		socialInsuranceHandler := NewSocialInsuranceHandler(socialInsuranceSvc)
		api.Mount("/social-insurance", socialInsuranceHandler.Routes())

		healthInsuranceHandler := NewHealthInsuranceHandler(healthInsuranceSvc)
		api.Mount("/health-insurance", healthInsuranceHandler.Routes())

		taxYearSettingsHandler := NewTaxYearSettingsHandler(taxYearSettingsSvc)
		api.Mount("/tax-year-settings", taxYearSettingsHandler.Routes())

		api.Get("/tax-constants/{year}", handleGetTaxConstants)

		taxCreditsHandler := NewTaxCreditsHandler(taxCreditsSvc)
		api.Mount("/tax-credits", taxCreditsHandler.Routes())

		taxDeductionsHandler := NewTaxDeductionsHandler(taxCreditsSvc, taxDeductionDocSvc, taxExtractionSvc)
		api.Mount("/tax-deductions", taxDeductionsHandler.Routes())
		api.Mount("/tax-deduction-documents", taxDeductionsHandler.DocumentRoutes())

		if investmentIncomeSvc != nil {
			investmentHandler := NewInvestmentIncomeHandler(investmentIncomeSvc, investmentDocSvc, investmentExtractionSvc)
			api.Mount("/investments", investmentHandler.Routes())
		}

		fakturoidHandler := NewFakturoidHandler(fakturoidImportSvc)
		api.Mount("/import/fakturoid", fakturoidHandler.Routes())

		// Dashboard
		dashboardHandler := NewDashboardHandler(dashboardSvc)
		api.Get("/dashboard", dashboardHandler.GetDashboard)

		// Reports
		reportHandler := NewReportHandler(reportSvc)
		api.Get("/reports/revenue", reportHandler.Revenue)
		api.Get("/reports/expenses", reportHandler.Expenses)
		api.Get("/reports/top-customers", reportHandler.TopCustomers)
		api.Get("/reports/profit-loss", reportHandler.ProfitLoss)

		// Tax calendar
		taxCalendarHandler := NewTaxCalendarHandler(taxCalendarSvc)
		api.Get("/reports/tax-calendar", taxCalendarHandler.GetCalendar)

		// CSV export
		exportHandler := NewExportHandler(invoiceSvc, expenseSvc)
		api.Get("/export/invoices", exportHandler.ExportInvoices)
		api.Get("/export/expenses", exportHandler.ExportExpenses)

		// Audit log
		auditLogHandler := NewAuditLogHandler(auditSvc)
		api.Mount("/audit-log", auditLogHandler.Routes())
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
