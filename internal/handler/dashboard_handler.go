package handler

import (
	"log/slog"
	"net/http"

	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
)

// --- Dashboard Response DTOs ---

type dashboardResponse struct {
	RevenueCurrentMonth  int64                   `json:"revenue_current_month"`
	ExpensesCurrentMonth int64                   `json:"expenses_current_month"`
	UnpaidCount          int                     `json:"unpaid_count"`
	UnpaidTotal          int64                   `json:"unpaid_total"`
	OverdueCount         int                     `json:"overdue_count"`
	OverdueTotal         int64                   `json:"overdue_total"`
	MonthlyRevenue       []monthAmountResponse   `json:"monthly_revenue"`
	MonthlyExpenses      []monthAmountResponse   `json:"monthly_expenses"`
	RecentInvoices       []recentInvoiceResponse `json:"recent_invoices"`
	RecentExpenses       []recentExpenseResponse `json:"recent_expenses"`
}

type monthAmountResponse struct {
	Month  int   `json:"month"`
	Amount int64 `json:"amount"`
}

type recentInvoiceResponse struct {
	ID            int64  `json:"id"`
	InvoiceNumber string `json:"invoice_number"`
	CustomerID    int64  `json:"customer_id"`
	TotalAmount   int64  `json:"total_amount"`
	Status        string `json:"status"`
	IssueDate     string `json:"issue_date"`
}

type recentExpenseResponse struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Amount      int64  `json:"amount"`
	IssueDate   string `json:"issue_date"`
}

// DashboardHandler handles HTTP requests for the dashboard.
type DashboardHandler struct {
	svc *service.DashboardService
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// GetDashboard returns aggregated dashboard data.
func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.GetDashboard(r.Context())
	if err != nil {
		slog.Error("failed to get dashboard data", "error", err)
		respondError(w, http.StatusInternalServerError, "Failed to load dashboard data")
		return
	}

	resp := dashboardResponse{
		RevenueCurrentMonth:  int64(data.RevenueCurrentMonth),
		ExpensesCurrentMonth: int64(data.ExpensesCurrentMonth),
		UnpaidCount:          data.UnpaidCount,
		UnpaidTotal:          int64(data.UnpaidTotal),
		OverdueCount:         data.OverdueCount,
		OverdueTotal:         int64(data.OverdueTotal),
		MonthlyRevenue:       mapMonthAmounts(data.MonthlyRevenue),
		MonthlyExpenses:      mapMonthAmounts(data.MonthlyExpenses),
		RecentInvoices:       mapRecentInvoices(data.RecentInvoices),
		RecentExpenses:       mapRecentExpenses(data.RecentExpenses),
	}

	respondJSON(w, http.StatusOK, resp)
}

func mapMonthAmounts(items []repository.MonthlyAmount) []monthAmountResponse {
	result := make([]monthAmountResponse, len(items))
	for i, item := range items {
		result[i] = monthAmountResponse{
			Month:  item.Month,
			Amount: int64(item.Amount),
		}
	}
	return result
}

func mapRecentInvoices(items []repository.RecentInvoice) []recentInvoiceResponse {
	result := make([]recentInvoiceResponse, len(items))
	for i, item := range items {
		result[i] = recentInvoiceResponse{
			ID:            item.ID,
			InvoiceNumber: item.InvoiceNumber,
			CustomerID:    item.CustomerID,
			TotalAmount:   int64(item.TotalAmount),
			Status:        item.Status,
			IssueDate:     item.IssueDate.Format("2006-01-02"),
		}
	}
	return result
}

func mapRecentExpenses(items []repository.RecentExpense) []recentExpenseResponse {
	result := make([]recentExpenseResponse, len(items))
	for i, item := range items {
		result[i] = recentExpenseResponse{
			ID:          item.ID,
			Description: item.Description,
			Category:    item.Category,
			Amount:      int64(item.Amount),
			IssueDate:   item.IssueDate.Format("2006-01-02"),
		}
	}
	return result
}
