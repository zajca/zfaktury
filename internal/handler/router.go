package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/zajca/zfaktury/internal/service"
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
	cfg RouterConfig,
) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
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
		invoiceHandler := NewInvoiceHandler(invoiceSvc)
		expenseHandler := NewExpenseHandler(expenseSvc)

		api.Mount("/contacts", contactHandler.Routes())
		api.Mount("/invoices", invoiceHandler.Routes())
		api.Mount("/expenses", expenseHandler.Routes())
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
