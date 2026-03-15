package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/zajca/zfaktury/internal/ares"
	"github.com/zajca/zfaktury/internal/config"
	"github.com/zajca/zfaktury/internal/database"
	"github.com/zajca/zfaktury/internal/flock"
	"github.com/zajca/zfaktury/internal/handler"
	"github.com/zajca/zfaktury/internal/isdoc"
	"github.com/zajca/zfaktury/internal/pdf"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/service/cnb"
	"github.com/zajca/zfaktury/internal/service/email"
	"github.com/zajca/zfaktury/internal/service/ocr"
)

// App holds the wired application components.
type App struct {
	cfg     *config.Config
	db      *sql.DB
	lock    *flock.Lock
	logFile *os.File
	router  *chi.Mux
}

// Options configures how the application is initialized.
type Options struct {
	ConfigFile string
	InitConfig bool
	Port       int // 0 = let the OS pick a free port
	DevMode    bool
}

// New creates a fully wired application: config, logging, lock, DB, migrations,
// repositories, services, and the HTTP router.
func New(opts Options) (*App, error) {
	cfgPath, err := config.Resolve(opts.ConfigFile, opts.InitConfig)
	if err != nil {
		return nil, err
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	logFile, err := setupLogging(cfg.Log)
	if err != nil {
		return nil, fmt.Errorf("setting up logging: %w", err)
	}

	cfg.Server.Port = opts.Port
	cfg.Server.Dev = opts.DevMode

	if err := cfg.Validate(); err != nil {
		if logFile != nil {
			_ = logFile.Close()
		}
		return nil, err
	}

	// Acquire instance lock to prevent concurrent access.
	lockPath := filepath.Join(cfg.DataDir, ".zfaktury.lock")
	lock, err := flock.Acquire(lockPath)
	if err != nil {
		if logFile != nil {
			_ = logFile.Close()
		}
		return nil, fmt.Errorf("acquiring instance lock: %w", err)
	}

	db, err := database.New(cfg)
	if err != nil {
		_ = lock.Release()
		if logFile != nil {
			_ = logFile.Close()
		}
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := database.Migrate(db); err != nil {
		_ = db.Close()
		_ = lock.Release()
		if logFile != nil {
			_ = logFile.Close()
		}
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	router := wireRouter(cfg, db)

	// Mount frontend handler.
	if cfg.Server.Dev {
		mountDevProxy(router)
		slog.Info("dev mode enabled, proxying frontend to Vite at localhost:5173")
	} else {
		mountEmbeddedFrontend(router)
	}

	return &App{
		cfg:     cfg,
		db:      db,
		lock:    lock,
		logFile: logFile,
		router:  router,
	}, nil
}

// Router returns the chi router with all routes mounted.
func (a *App) Router() *chi.Mux {
	return a.router
}

// Config returns the resolved configuration.
func (a *App) Config() *config.Config {
	return a.cfg
}

// ListenAndServe starts the HTTP server and blocks until ctx is cancelled.
func (a *App) ListenAndServe(ctx context.Context) error {
	addr := fmt.Sprintf("127.0.0.1:%d", a.cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      a.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("starting server", "addr", addr, "dev", a.cfg.Server.Dev)
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
}

// ListenAndServeOnFreePort starts the HTTP server on a random free port
// and returns the listener address. The server runs until ctx is cancelled.
func (a *App) ListenAndServeOnFreePort(ctx context.Context) (net.Addr, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listening on free port: %w", err)
	}

	srv := &http.Server{
		Handler:      a.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("starting server", "addr", ln.Addr().String(), "dev", a.cfg.Server.Dev)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		slog.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
		slog.Info("server stopped")
	}()

	return ln.Addr(), nil
}

// Close releases the database, lock, and log file.
func (a *App) Close() error {
	if a.db != nil {
		_ = a.db.Close()
	}
	if a.lock != nil {
		_ = a.lock.Release()
	}
	if a.logFile != nil {
		_ = a.logFile.Close()
	}
	return nil
}

// wireRouter creates repositories, services, and the HTTP router.
func wireRouter(cfg *config.Config, db *sql.DB) *chi.Mux {
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

	// Wire backup service.
	backupHistoryRepo := repository.NewBackupHistoryRepository(db)
	var backupStorage service.BackupStorage
	if cfg.Backup.S3.IsConfigured() {
		s3Storage, err := service.NewS3Storage(cfg.Backup.S3)
		if err != nil {
			slog.Error("initializing S3 backup storage", "error", err)
		} else {
			backupStorage = s3Storage
			slog.Info("backup storage configured", "type", "s3", "endpoint", cfg.Backup.S3.Endpoint, "bucket", cfg.Backup.S3.Bucket)
		}
	}
	if backupStorage == nil {
		backupStorage = service.NewLocalStorage(cfg.BackupDestination())
		slog.Info("backup storage configured", "type", "local", "path", cfg.BackupDestination())
	}
	backupSvc := service.NewBackupService(backupHistoryRepo, db, cfg.Backup, cfg.DataDir, backupStorage)

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

	return handler.NewRouter(contactSvc, invoiceSvc, expenseSvc, settingsSvc, sequenceSvc, categorySvc, documentSvc, recurringInvoiceSvc, recurringExpenseSvc, ocrSvc, importSvc, overdueSvc, reminderSvc, cnbClient, pdfGen, isdocGen, vatReturnSvc, vatControlSvc, viesSvc, incomeTaxSvc, socialInsuranceSvc, healthInsuranceSvc, taxYearSettingsSvc, taxCreditsSvc, taxDeductionDocSvc, taxExtractionSvc, investmentIncomeSvc, investmentDocSvc, investmentExtractionSvc, invDocumentSvc, fakturoidImportSvc, dashboardSvc, reportSvc, taxCalendarSvc, emailSender, auditSvc, backupSvc, handler.RouterConfig{
		DevMode: cfg.Server.Dev,
		DataDir: cfg.DataDir,
	})
}
