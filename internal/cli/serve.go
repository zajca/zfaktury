package cli

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
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
	"github.com/zajca/zfaktury/web"
)

var (
	servePort int
	serveDev  bool
)

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "HTTP server port")
	serveCmd.Flags().BoolVar(&serveDev, "dev", false, "Enable development mode (proxy frontend to Vite)")
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Long:  "Start the ZFaktury HTTP server serving both the API and the frontend.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		cfg.Server.Port = servePort
		cfg.Server.Dev = serveDev

		db, err := database.New(cfg)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}
		defer db.Close()

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

		// Wire ARES client.
		aresClient := ares.NewClient()

		// Wire generators.
		pdfGen := pdf.NewInvoicePDFGenerator()
		isdocGen := isdoc.NewISDOCGenerator()

		// Wire services.
		contactSvc := service.NewContactService(contactRepo, aresClient)
		sequenceSvc := service.NewSequenceService(sequenceRepo)
		invoiceSvc := service.NewInvoiceService(invoiceRepo, contactSvc, sequenceSvc)
		expenseSvc := service.NewExpenseService(expenseRepo)
		settingsSvc := service.NewSettingsService(settingsRepo)
		categorySvc := service.NewCategoryService(categoryRepo)

		router := handler.NewRouter(contactSvc, invoiceSvc, expenseSvc, settingsSvc, sequenceSvc, categorySvc, pdfGen, isdocGen, handler.RouterConfig{
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
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
			f.Close()
		}
		fileServer.ServeHTTP(w, req)
	})
}
