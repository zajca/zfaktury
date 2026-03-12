package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/zajca/zfaktury/internal/service"
)

// --- Report Response DTOs ---

type monthlyAmountResponse struct {
	Month  int   `json:"month"`
	Amount int64 `json:"amount"`
}

type quarterlyAmountResponse struct {
	Quarter int   `json:"quarter"`
	Amount  int64 `json:"amount"`
}

type categoryAmountResponse struct {
	Category string `json:"category"`
	Amount   int64  `json:"amount"`
}

type revenueReportResponse struct {
	Year      int                       `json:"year"`
	Monthly   []monthlyAmountResponse   `json:"monthly"`
	Quarterly []quarterlyAmountResponse `json:"quarterly"`
	Total     int64                     `json:"total"`
}

type expenseReportResponse struct {
	Year       int                       `json:"year"`
	Monthly    []monthlyAmountResponse   `json:"monthly"`
	Quarterly  []quarterlyAmountResponse `json:"quarterly"`
	Categories []categoryAmountResponse  `json:"categories"`
}

type topCustomerResponse struct {
	CustomerID   int64  `json:"customer_id"`
	CustomerName string `json:"customer_name"`
	Total        int64  `json:"total"`
	InvoiceCount int    `json:"invoice_count"`
}

type profitLossResponse struct {
	Year            int                     `json:"year"`
	MonthlyRevenue  []monthlyAmountResponse `json:"monthly_revenue"`
	MonthlyExpenses []monthlyAmountResponse `json:"monthly_expenses"`
}

// ReportHandler handles HTTP requests for report endpoints.
type ReportHandler struct {
	svc *service.ReportService
}

// NewReportHandler creates a new ReportHandler.
func NewReportHandler(svc *service.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

// parseReportYear extracts the year query parameter, defaulting to the current year.
func parseReportYear(r *http.Request) (int, error) {
	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		return time.Now().Year(), nil
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return 0, err
	}
	return year, nil
}

// Revenue handles GET requests for revenue reports.
func (h *ReportHandler) Revenue(w http.ResponseWriter, r *http.Request) {
	year, err := parseReportYear(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	report, err := h.svc.RevenueReport(r.Context(), year)
	if err != nil {
		slog.Error("failed to fetch revenue report", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to fetch revenue report")
		return
	}

	monthly := make([]monthlyAmountResponse, len(report.Monthly))
	for i, m := range report.Monthly {
		monthly[i] = monthlyAmountResponse{Month: m.Month, Amount: int64(m.Amount)}
	}

	quarterly := make([]quarterlyAmountResponse, len(report.Quarterly))
	for i, q := range report.Quarterly {
		quarterly[i] = quarterlyAmountResponse{Quarter: q.Quarter, Amount: int64(q.Amount)}
	}

	respondJSON(w, http.StatusOK, revenueReportResponse{
		Year:      report.Year,
		Monthly:   monthly,
		Quarterly: quarterly,
		Total:     int64(report.Total),
	})
}

// Expenses handles GET requests for expense reports.
func (h *ReportHandler) Expenses(w http.ResponseWriter, r *http.Request) {
	year, err := parseReportYear(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	report, err := h.svc.ExpenseReport(r.Context(), year)
	if err != nil {
		slog.Error("failed to fetch expense report", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to fetch expense report")
		return
	}

	monthly := make([]monthlyAmountResponse, len(report.Monthly))
	for i, m := range report.Monthly {
		monthly[i] = monthlyAmountResponse{Month: m.Month, Amount: int64(m.Amount)}
	}

	quarterly := make([]quarterlyAmountResponse, len(report.Quarterly))
	for i, q := range report.Quarterly {
		quarterly[i] = quarterlyAmountResponse{Quarter: q.Quarter, Amount: int64(q.Amount)}
	}

	categories := make([]categoryAmountResponse, len(report.Categories))
	for i, c := range report.Categories {
		categories[i] = categoryAmountResponse{Category: c.Category, Amount: int64(c.Amount)}
	}

	respondJSON(w, http.StatusOK, expenseReportResponse{
		Year:       report.Year,
		Monthly:    monthly,
		Quarterly:  quarterly,
		Categories: categories,
	})
}

// TopCustomers handles GET requests for top customers report.
func (h *ReportHandler) TopCustomers(w http.ResponseWriter, r *http.Request) {
	year, err := parseReportYear(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	customers, err := h.svc.TopCustomers(r.Context(), year)
	if err != nil {
		slog.Error("failed to fetch top customers", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to fetch top customers")
		return
	}

	resp := make([]topCustomerResponse, len(customers))
	for i, c := range customers {
		resp[i] = topCustomerResponse{
			CustomerID:   c.CustomerID,
			CustomerName: c.CustomerName,
			Total:        int64(c.Total),
			InvoiceCount: c.InvoiceCount,
		}
	}

	respondJSON(w, http.StatusOK, resp)
}

// ProfitLoss handles GET requests for profit/loss reports.
func (h *ReportHandler) ProfitLoss(w http.ResponseWriter, r *http.Request) {
	year, err := parseReportYear(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	report, err := h.svc.ProfitLoss(r.Context(), year)
	if err != nil {
		slog.Error("failed to fetch profit/loss report", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to fetch profit/loss report")
		return
	}

	revenue := make([]monthlyAmountResponse, len(report.MonthlyRevenue))
	for i, m := range report.MonthlyRevenue {
		revenue[i] = monthlyAmountResponse{Month: m.Month, Amount: int64(m.Amount)}
	}

	expenses := make([]monthlyAmountResponse, len(report.MonthlyExpenses))
	for i, m := range report.MonthlyExpenses {
		expenses[i] = monthlyAmountResponse{Month: m.Month, Amount: int64(m.Amount)}
	}

	respondJSON(w, http.StatusOK, profitLossResponse{
		Year:            report.Year,
		MonthlyRevenue:  revenue,
		MonthlyExpenses: expenses,
	})
}
