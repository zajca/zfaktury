package service

import (
	"context"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// DashboardData holds all aggregated data for the dashboard view.
type DashboardData struct {
	RevenueCurrentMonth  domain.Amount
	ExpensesCurrentMonth domain.Amount
	UnpaidCount          int
	UnpaidTotal          domain.Amount
	OverdueCount         int
	OverdueTotal         domain.Amount
	MonthlyRevenue       []repository.MonthlyAmount
	MonthlyExpenses      []repository.MonthlyAmount
	RecentInvoices       []repository.RecentInvoice
	RecentExpenses       []repository.RecentExpense
}

// DashboardService provides business logic for the dashboard.
type DashboardService struct {
	repo repository.DashboardRepo
}

// NewDashboardService creates a new DashboardService.
func NewDashboardService(repo repository.DashboardRepo) *DashboardService {
	return &DashboardService{repo: repo}
}

// GetDashboard fetches all dashboard data for the current month and year.
func (s *DashboardService) GetDashboard(ctx context.Context) (*DashboardData, error) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	revenue, err := s.repo.RevenueCurrentMonth(ctx, year, month)
	if err != nil {
		return nil, fmt.Errorf("fetching revenue: %w", err)
	}

	expenses, err := s.repo.ExpensesCurrentMonth(ctx, year, month)
	if err != nil {
		return nil, fmt.Errorf("fetching expenses: %w", err)
	}

	unpaidCount, unpaidTotal, err := s.repo.UnpaidInvoices(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching unpaid invoices: %w", err)
	}

	overdueCount, overdueTotal, err := s.repo.OverdueInvoices(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching overdue invoices: %w", err)
	}

	monthlyRevenue, err := s.repo.MonthlyRevenue(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("fetching monthly revenue: %w", err)
	}

	monthlyExpenses, err := s.repo.MonthlyExpenses(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("fetching monthly expenses: %w", err)
	}

	recentInvoices, err := s.repo.RecentInvoices(ctx, 5)
	if err != nil {
		return nil, fmt.Errorf("fetching recent invoices: %w", err)
	}

	recentExpenses, err := s.repo.RecentExpenses(ctx, 5)
	if err != nil {
		return nil, fmt.Errorf("fetching recent expenses: %w", err)
	}

	return &DashboardData{
		RevenueCurrentMonth:  revenue,
		ExpensesCurrentMonth: expenses,
		UnpaidCount:          unpaidCount,
		UnpaidTotal:          unpaidTotal,
		OverdueCount:         overdueCount,
		OverdueTotal:         overdueTotal,
		MonthlyRevenue:       monthlyRevenue,
		MonthlyExpenses:      monthlyExpenses,
		RecentInvoices:       recentInvoices,
		RecentExpenses:       recentExpenses,
	}, nil
}
