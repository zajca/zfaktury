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

// ReportService provides business logic for report generation. All methods
// are scoped to a single company via the companyID parameter.
type ReportService struct {
	repo repository.ReportRepo
}

// NewReportService creates a new ReportService.
func NewReportService(repo repository.ReportRepo) *ReportService {
	return &ReportService{repo: repo}
}

// RevenueReport returns aggregated revenue data for the given year within the given company.
func (s *ReportService) RevenueReport(ctx context.Context, companyID int64, year int) (*RevenueReport, error) {
	monthly, err := s.repo.MonthlyRevenue(ctx, companyID, year)
	if err != nil {
		return nil, fmt.Errorf("fetching revenue report: %w", err)
	}

	quarterly, err := s.repo.QuarterlyRevenue(ctx, companyID, year)
	if err != nil {
		return nil, fmt.Errorf("fetching revenue report: %w", err)
	}

	total, err := s.repo.YearlyRevenue(ctx, companyID, year)
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

// ExpenseReport returns aggregated expense data for the given year within the given company.
func (s *ReportService) ExpenseReport(ctx context.Context, companyID int64, year int) (*ExpenseReport, error) {
	monthly, err := s.repo.MonthlyExpenses(ctx, companyID, year)
	if err != nil {
		return nil, fmt.Errorf("fetching expense report: %w", err)
	}

	quarterly, err := s.repo.QuarterlyExpenses(ctx, companyID, year)
	if err != nil {
		return nil, fmt.Errorf("fetching expense report: %w", err)
	}

	categories, err := s.repo.CategoryExpenses(ctx, companyID, year)
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

// TopCustomers returns top customers by revenue for the given year within the given company.
func (s *ReportService) TopCustomers(ctx context.Context, companyID int64, year int) ([]repository.CustomerRevenue, error) {
	customers, err := s.repo.TopCustomers(ctx, companyID, year, 10)
	if err != nil {
		return nil, fmt.Errorf("fetching top customers: %w", err)
	}
	return customers, nil
}

// ProfitLoss returns monthly profit/loss data for the given year within the given company.
func (s *ReportService) ProfitLoss(ctx context.Context, companyID int64, year int) (*ProfitLossReport, error) {
	revenue, expenses, err := s.repo.ProfitLossMonthly(ctx, companyID, year)
	if err != nil {
		return nil, fmt.Errorf("fetching profit/loss report: %w", err)
	}

	return &ProfitLossReport{
		Year:            year,
		MonthlyRevenue:  revenue,
		MonthlyExpenses: expenses,
	}, nil
}
