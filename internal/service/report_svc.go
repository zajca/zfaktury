package service

import (
	"context"
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// RevenueReport holds aggregated revenue data for a year.
type RevenueReport struct {
	Year      int
	Monthly   []repository.MonthlyAmount
	Quarterly []repository.QuarterlyAmount
	Total     domain.Amount
}

// ExpenseReport holds aggregated expense data for a year.
type ExpenseReport struct {
	Year       int
	Monthly    []repository.MonthlyAmount
	Quarterly  []repository.QuarterlyAmount
	Categories []repository.CategoryAmount
}

// ProfitLossReport holds monthly revenue and expenses for a year.
type ProfitLossReport struct {
	Year            int
	MonthlyRevenue  []repository.MonthlyAmount
	MonthlyExpenses []repository.MonthlyAmount
}

// ReportService provides business logic for report generation.
type ReportService struct {
	repo repository.ReportRepo
}

// NewReportService creates a new ReportService.
func NewReportService(repo repository.ReportRepo) *ReportService {
	return &ReportService{repo: repo}
}

// RevenueReport returns aggregated revenue data for the given year.
func (s *ReportService) RevenueReport(ctx context.Context, year int) (*RevenueReport, error) {
	monthly, err := s.repo.MonthlyRevenue(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("fetching revenue report: %w", err)
	}

	quarterly, err := s.repo.QuarterlyRevenue(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("fetching revenue report: %w", err)
	}

	total, err := s.repo.YearlyRevenue(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("fetching revenue report: %w", err)
	}

	return &RevenueReport{
		Year:      year,
		Monthly:   monthly,
		Quarterly: quarterly,
		Total:     total,
	}, nil
}

// ExpenseReport returns aggregated expense data for the given year.
func (s *ReportService) ExpenseReport(ctx context.Context, year int) (*ExpenseReport, error) {
	monthly, err := s.repo.MonthlyExpenses(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("fetching expense report: %w", err)
	}

	quarterly, err := s.repo.QuarterlyExpenses(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("fetching expense report: %w", err)
	}

	categories, err := s.repo.CategoryExpenses(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("fetching expense report: %w", err)
	}

	return &ExpenseReport{
		Year:       year,
		Monthly:    monthly,
		Quarterly:  quarterly,
		Categories: categories,
	}, nil
}

// TopCustomers returns top customers by revenue for the given year.
func (s *ReportService) TopCustomers(ctx context.Context, year int) ([]repository.CustomerRevenue, error) {
	customers, err := s.repo.TopCustomers(ctx, year, 10)
	if err != nil {
		return nil, fmt.Errorf("fetching top customers: %w", err)
	}
	return customers, nil
}

// ProfitLoss returns monthly profit/loss data for the given year.
func (s *ReportService) ProfitLoss(ctx context.Context, year int) (*ProfitLossReport, error) {
	revenue, expenses, err := s.repo.ProfitLossMonthly(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("fetching profit/loss report: %w", err)
	}

	return &ProfitLossReport{
		Year:            year,
		MonthlyRevenue:  revenue,
		MonthlyExpenses: expenses,
	}, nil
}
