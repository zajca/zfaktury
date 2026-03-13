package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/spf13/cobra"

	"github.com/zajca/zfaktury/internal/ares"
	"github.com/zajca/zfaktury/internal/config"
	"github.com/zajca/zfaktury/internal/database"
	"github.com/zajca/zfaktury/internal/handler"
	"github.com/zajca/zfaktury/internal/isdoc"
	"github.com/zajca/zfaktury/internal/pdf"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/service/cnb"
	"github.com/zajca/zfaktury/internal/service/email"
	"github.com/zajca/zfaktury/internal/service/ocr"
	"github.com/zajca/zfaktury/web"
)

var (
	servePort      int
	serveDev       bool
	serveInitConfig bool
)

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "HTTP server port")
	serveCmd.Flags().BoolVar(&serveDev, "dev", false, "Enable development mode (proxy frontend to Vite)")
	serveCmd.Flags().BoolVar(&serveInitConfig, "init-config", false, "Create a default config file if it does not exist")
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Long:  "Start the ZFaktury HTTP server serving both the API and the frontend.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := config.Resolve(configFile, serveInitConfig)
		if err != nil {
			return err
		}

		cfg, err := config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		logFile, err := setupLogging(cfg.Log)
		if err != nil {
			return fmt.Errorf("setting up logging: %w", err)
		}
		if logFile != nil {
			defer func() { _ = logFile.Close() }()
		}

		cfg.Server.Port = servePort
		cfg.Server.Dev = serveDev

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
		settingsRepo := repository.NewSettingsRepository(db)
		sequenceRepo := repository.NewSequenceRepository(db)
		categoryRepo := repository.NewCategoryRepository(db)
		documentRepo := repository.NewDocumentRepository(db)
		recurringInvoiceRepo := repository.NewRecurringInvoiceRepository(db)
		recurringExpenseRepo := repository.NewRecurringExpenseRepository(db)

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
		taxDeductionDocRepo := repository.NewTaxDeductionDocumentRepository(db)

		auditLogRepo := repository.NewAuditLogRepository(db)
		auditSvc := service.NewAuditService(auditLogRepo)

		// Wire ARES client.
		aresClient := ares.NewClient()

		// Wire email sender.
		emailSender := email.NewEmailSender(cfg.SMTP)
		if emailSender.IsConfigured() {
			slog.Info("email sender configured", "host", cfg.SMTP.Host)
		}
		// Wire generators.
		pdfGen := pdf.NewInvoicePDFGenerator()
		isdocGen := isdoc.NewISDOCGenerator()

		// Wire services.
		contactSvc := service.NewContactService(contactRepo, aresClient, auditSvc)
		sequenceSvc := service.NewSequenceService(sequenceRepo, auditSvc)
		invoiceSvc := service.NewInvoiceService(invoiceRepo, contactSvc, sequenceSvc, auditSvc)
		expenseSvc := service.NewExpenseService(expenseRepo, auditSvc)
		settingsSvc := service.NewSettingsService(settingsRepo, auditSvc)
		categorySvc := service.NewCategoryService(categoryRepo, auditSvc)
		documentSvc := service.NewDocumentService(documentRepo, cfg.DataDir, auditSvc)
		recurringInvoiceSvc := service.NewRecurringInvoiceService(recurringInvoiceRepo, invoiceSvc, auditSvc)
		recurringExpenseSvc := service.NewRecurringExpenseService(recurringExpenseRepo, expenseSvc, auditSvc)

		vatReturnSvc := service.NewVATReturnService(vatReturnRepo, invoiceRepo, expenseRepo, settingsRepo, auditSvc)
		vatControlSvc := service.NewVATControlStatementService(vatControlRepo, invoiceRepo, expenseRepo, contactRepo, auditSvc)
		viesSvc := service.NewVIESSummaryService(viesRepo, invoiceRepo, contactRepo, auditSvc)

		taxYearSettingsSvc := service.NewTaxYearSettingsService(taxYearSettingsRepo, taxPrepaymentRepo, auditSvc)

		taxCreditsSvc := service.NewTaxCreditsService(taxSpouseCreditRepo, taxChildCreditRepo, taxPersonalCreditsRepo, taxDeductionRepo, auditSvc)
		taxDeductionDocSvc := service.NewTaxDeductionDocumentService(taxDeductionDocRepo, taxDeductionRepo, cfg.DataDir, auditSvc)

		// Wire investment repos.
		investmentDocRepo := repository.NewInvestmentDocumentRepository(db)
		capitalIncomeRepo := repository.NewCapitalIncomeRepository(db)
		securityTransactionRepo := repository.NewSecurityTransactionRepository(db)

		// Wire investment services.
		investmentDocSvc := service.NewInvestmentDocumentService(investmentDocRepo, capitalIncomeRepo, securityTransactionRepo, cfg.DataDir, auditSvc)
		investmentIncomeSvc := service.NewInvestmentIncomeService(capitalIncomeRepo, securityTransactionRepo, auditSvc)

		incomeTaxSvc := service.NewIncomeTaxReturnService(incomeTaxReturnRepo, invoiceRepo, expenseRepo, settingsRepo, taxYearSettingsRepo, taxPrepaymentRepo, taxCreditsSvc, auditSvc)
		incomeTaxSvc.SetInvestmentService(investmentIncomeSvc)
		socialInsuranceSvc := service.NewSocialInsuranceService(socialInsuranceRepo, invoiceRepo, expenseRepo, settingsRepo, taxYearSettingsRepo, taxPrepaymentRepo, auditSvc)
		healthInsuranceSvc := service.NewHealthInsuranceService(healthInsuranceRepo, invoiceRepo, expenseRepo, settingsRepo, taxYearSettingsRepo, taxPrepaymentRepo, auditSvc)

		// Wire OCR service (conditional on API key).
		var ocrSvc *service.OCRService
		var taxExtractionSvc *service.TaxDocumentExtractionService
		var investmentExtractionSvc *service.InvestmentExtractionService
		if cfg.OCR.APIKey != "" {
			provider, err := ocr.NewProvider(cfg.OCR.Provider, cfg.OCR.APIKey, cfg.OCR.Model, cfg.OCR.BaseURL)
			if err != nil {
				slog.Warn("OCR disabled", "error", err)
			} else {
				ocrSvc = service.NewOCRService(provider, documentSvc)
				taxExtractionSvc = service.NewTaxDocumentExtractionService(provider, taxDeductionDocSvc, taxDeductionRepo, taxDeductionDocRepo)
				investmentExtractionSvc = service.NewInvestmentExtractionService(provider, investmentDocSvc, capitalIncomeRepo, securityTransactionRepo, investmentDocRepo)
				slog.Info("OCR service configured", "provider", provider.Name())
			}
		}

		// Wire import service (for upload-first expense creation).
		importSvc := service.NewImportService(expenseSvc, documentSvc, ocrSvc)

		// Wire invoice document repo and service.
		invDocumentRepo := repository.NewInvoiceDocumentRepository(db)
		invDocumentSvc := service.NewInvoiceDocumentService(invDocumentRepo, cfg.DataDir, auditSvc)

		// Wire Fakturoid import (credentials provided per-request via UI).
		fakturoidImportRepo := repository.NewFakturoidImportLogRepository(db)
		fakturoidImportSvc := service.NewFakturoidImportService(
			fakturoidImportRepo, contactRepo, invoiceRepo, expenseRepo,
			contactSvc, invoiceSvc, expenseSvc, documentSvc, invDocumentSvc,
		)

		// Wire dashboard and report repos/services.
		dashboardRepo := repository.NewDashboardRepository(db)
		dashboardSvc := service.NewDashboardService(dashboardRepo)
		reportRepo := repository.NewReportRepository(db)
		reportSvc := service.NewReportService(reportRepo)
		taxCalendarSvc := service.NewTaxCalendarService()

		// Wire CNB client.
		cnbClient := cnb.NewClient()

		// Wire status history repo and overdue service.
		statusHistoryRepo := repository.NewStatusHistoryRepository(db)
		overdueSvc := service.NewOverdueService(invoiceRepo, statusHistoryRepo)

		// Wire reminder service.
		reminderRepo := repository.NewReminderRepository(db)
		reminderSvc := service.NewReminderService(reminderRepo, invoiceRepo, emailSender, settingsSvc)

		router := handler.NewRouter(contactSvc, invoiceSvc, expenseSvc, settingsSvc, sequenceSvc, categorySvc, documentSvc, recurringInvoiceSvc, recurringExpenseSvc, ocrSvc, importSvc, overdueSvc, reminderSvc, cnbClient, pdfGen, isdocGen, vatReturnSvc, vatControlSvc, viesSvc, incomeTaxSvc, socialInsuranceSvc, healthInsuranceSvc, taxYearSettingsSvc, taxCreditsSvc, taxDeductionDocSvc, taxExtractionSvc, investmentIncomeSvc, investmentDocSvc, investmentExtractionSvc, invDocumentSvc, fakturoidImportSvc, dashboardSvc, reportSvc, taxCalendarSvc, emailSender, auditSvc, handler.RouterConfig{
			DevMode: cfg.Server.Dev,
		})

		// Serve frontend
		if cfg.Server.Dev {
			mountDevProxy(router)
			slog.Info("dev mode enabled, proxying frontend to Vite at localhost:5173")
		} else {
			mountEmbeddedFrontend(router)
		}

		addr := fmt.Sprintf("127.0.0.1:%d", cfg.Server.Port)
		srv := &http.Server{
			Addr:         addr,
			Handler:      router,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		// Graceful shutdown
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		go func() {
			slog.Info("starting server", "addr", addr, "dev", cfg.Server.Dev)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("server error", "error", err)
				os.Exit(1)
			}
		}()

		<-ctx.Done()
		slog.Info("shutting down server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown: %w", err)
		}

		slog.Info("server stopped")
		return nil
	},
}

// mountDevProxy sets up a reverse proxy to the Vite dev server for non-API requests.
func mountDevProxy(r *chi.Mux) {
	viteURL, _ := url.Parse("http://localhost:5173")
	proxy := httputil.NewSingleHostReverseProxy(viteURL)

	// Proxy everything that is not an API route
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		proxy.ServeHTTP(w, req)
	})
}

// setupLogging configures the default slog logger based on config.
// If a log path is configured, logs are written to both stderr and the file.
// Returns the opened file (if any) so the caller can defer Close.
func setupLogging(cfg config.LogConfig) (*os.File, error) {
	level := slog.LevelInfo
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: level}

	if cfg.Path != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.Path), 0o755); err != nil {
			return nil, fmt.Errorf("creating log directory: %w", err)
		}
		f, err := os.OpenFile(cfg.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("opening log file: %w", err)
		}
		w := io.MultiWriter(os.Stderr, f)
		slog.SetDefault(slog.New(slog.NewTextHandler(w, opts)))
		return f, nil
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, opts)))
	return nil, nil
}

// mountEmbeddedFrontend serves the embedded frontend build files.
func mountEmbeddedFrontend(r *chi.Mux) {
	distFS, err := fs.Sub(web.DistFS, "frontend/build")
	if err != nil {
		slog.Error("failed to create sub filesystem for frontend", "error", err)
		return
	}

	fileServer := http.FileServer(http.FS(distFS))

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		// Try to serve the file directly
		f, err := distFS.Open(req.URL.Path[1:]) // strip leading /
		if err != nil {
			// Fall back to index.html for SPA routing
			req.URL.Path = "/"
		} else {
			_ = f.Close()
		}
		fileServer.ServeHTTP(w, req)
	})
}
