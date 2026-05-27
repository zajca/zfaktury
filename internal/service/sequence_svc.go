package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/format"
	"github.com/zajca/zfaktury/internal/repository"
)

// SequenceService provides business logic for invoice sequence management.
type SequenceService struct {
	repo  repository.InvoiceSequenceRepo
	audit *AuditService
}

// NewSequenceService creates a new SequenceService.
func NewSequenceService(repo repository.InvoiceSequenceRepo, audit *AuditService) *SequenceService {
	return &SequenceService{repo: repo, audit: audit}
}

// Create validates uniqueness (prefix+year) within the company and persists a
// new invoice sequence.
func (s *SequenceService) Create(ctx context.Context, companyID int64, seq *domain.InvoiceSequence) error {
	if seq.Prefix == "" {
		return fmt.Errorf("prefix is required: %w", domain.ErrInvalidInput)
	}
	if seq.Year == 0 {
		return fmt.Errorf("year is required: %w", domain.ErrInvalidInput)
	}
	if seq.NextNumber <= 0 {
		seq.NextNumber = 1
	}
	if seq.FormatPattern == "" {
		seq.FormatPattern = "{prefix}{year}{number:04d}"
	}
	if err := format.ValidatePattern(seq.FormatPattern); err != nil {
		return fmt.Errorf("validating format pattern: %w", err)
	}

	// Check uniqueness of prefix+year within the company.
	existing, err := s.repo.GetByPrefixAndYear(ctx, companyID, seq.Prefix, seq.Year)
	if err == nil && existing != nil {
		return fmt.Errorf("sequence with prefix %q and year %d already exists: %w", seq.Prefix, seq.Year, domain.ErrDuplicateNumber)
	}
	// Only proceed if the error indicates not found.
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("checking sequence uniqueness: %w", err)
	}

	if err := s.repo.Create(ctx, companyID, seq); err != nil {
		return fmt.Errorf("creating sequence: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "sequence", seq.ID, "create", nil, seq)
	}
	return nil
}

// Update validates and updates an existing invoice sequence within the given company.
// Prevents lowering next_number below already-used numbers.
func (s *SequenceService) Update(ctx context.Context, companyID int64, seq *domain.InvoiceSequence) error {
	if seq.ID == 0 {
		return fmt.Errorf("sequence ID is required: %w", domain.ErrInvalidInput)
	}
	if seq.Prefix == "" {
		return fmt.Errorf("prefix is required: %w", domain.ErrInvalidInput)
	}
	if seq.Year == 0 {
		return fmt.Errorf("year is required: %w", domain.ErrInvalidInput)
	}
	if seq.NextNumber <= 0 {
		return fmt.Errorf("next number must be positive: %w", domain.ErrInvalidInput)
	}
	if seq.FormatPattern == "" {
		seq.FormatPattern = "{prefix}{year}{number:04d}"
	}
	if err := format.ValidatePattern(seq.FormatPattern); err != nil {
		return fmt.Errorf("validating format pattern: %w", err)
	}

	// Fetch existing for audit logging.
	existing, err := s.repo.GetByID(ctx, companyID, seq.ID)
	if err != nil {
		return fmt.Errorf("fetching existing sequence: %w", err)
	}

	// Prevent lowering next_number below already-used numbers.
	maxUsed, err := s.repo.MaxUsedNumber(ctx, companyID, seq.ID)
	if err != nil {
		return fmt.Errorf("checking max used number: %w", err)
	}
	if seq.NextNumber <= maxUsed {
		return fmt.Errorf("cannot set next number to %d, numbers up to %d have already been assigned: %w", seq.NextNumber, maxUsed, domain.ErrInvalidInput)
	}

	// Check uniqueness of prefix+year (within company) if they changed.
	duplicate, err := s.repo.GetByPrefixAndYear(ctx, companyID, seq.Prefix, seq.Year)
	if err == nil && duplicate != nil && duplicate.ID != seq.ID {
		return fmt.Errorf("sequence with prefix %q and year %d already exists: %w", seq.Prefix, seq.Year, domain.ErrDuplicateNumber)
	}

	if err := s.repo.Update(ctx, companyID, seq); err != nil {
		return fmt.Errorf("updating sequence: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "sequence", seq.ID, "update", existing, seq)
	}
	return nil
}

// Delete removes an invoice sequence by ID (soft delete) within the given company.
// Returns an error if invoices reference this sequence.
func (s *SequenceService) Delete(ctx context.Context, companyID, id int64) error {
	if id == 0 {
		return fmt.Errorf("sequence ID is required: %w", domain.ErrInvalidInput)
	}

	count, err := s.repo.CountInvoicesBySequenceID(ctx, companyID, id)
	if err != nil {
		return fmt.Errorf("checking invoice references: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete sequence: %d invoices reference it: %w", count, domain.ErrInvalidInput)
	}

	if err := s.repo.Delete(ctx, companyID, id); err != nil {
		return fmt.Errorf("deleting sequence: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "sequence", id, "delete", nil, nil)
	}
	return nil
}

// GetByID retrieves a sequence by its ID within the given company.
func (s *SequenceService) GetByID(ctx context.Context, companyID, id int64) (*domain.InvoiceSequence, error) {
	if id == 0 {
		return nil, fmt.Errorf("sequence ID is required: %w", domain.ErrInvalidInput)
	}
	seq, err := s.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, fmt.Errorf("fetching sequence: %w", err)
	}
	return seq, nil
}

// List retrieves all invoice sequences for the given company.
func (s *SequenceService) List(ctx context.Context, companyID int64) ([]domain.InvoiceSequence, error) {
	seqs, err := s.repo.List(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("listing sequences: %w", err)
	}
	return seqs, nil
}

// GetOrCreateForYear retrieves an existing sequence for the given prefix and year
// within the given company, or creates a new one if it doesn't exist.
func (s *SequenceService) GetOrCreateForYear(ctx context.Context, companyID int64, prefix string, year int) (*domain.InvoiceSequence, error) {
	if prefix == "" {
		return nil, fmt.Errorf("prefix is required: %w", domain.ErrInvalidInput)
	}
	if year == 0 {
		return nil, fmt.Errorf("year is required: %w", domain.ErrInvalidInput)
	}

	seq, err := s.repo.GetByPrefixAndYear(ctx, companyID, prefix, year)
	if err == nil {
		return seq, nil
	}

	// Not found -- create a new sequence for this year.
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("looking up sequence: %w", err)
	}

	newSeq := &domain.InvoiceSequence{
		Prefix:        prefix,
		NextNumber:    1,
		Year:          year,
		FormatPattern: "{prefix}{year}{number:04d}",
	}
	if err := s.repo.Create(ctx, companyID, newSeq); err != nil {
		// Race condition: another goroutine may have created the sequence concurrently.
		// Retry the lookup; if found, use that sequence.
		retrySeq, retryErr := s.repo.GetByPrefixAndYear(ctx, companyID, prefix, year)
		if retryErr == nil {
			return retrySeq, nil
		}
		return nil, fmt.Errorf("creating sequence for year %d: %w", year, err)
	}
	return newSeq, nil
}

// FormatPreview returns the formatted invoice number the sequence would
// produce for its current next_number. Backed by internal/format.Render so
// preview, persistence, and GetNextNumber stay in lockstep. The empty-pattern
// fallback mirrors Create/Update — every layer normalises empty to the legacy
// default so renders never see an empty pattern.
func FormatPreview(seq *domain.InvoiceSequence) string {
	pattern := seq.FormatPattern
	if pattern == "" {
		pattern = "{prefix}{year}{number:04d}"
	}
	return format.Render(pattern, seq.Prefix, seq.Year, seq.NextNumber)
}

// SequenceCompanyChecker reports whether a company has any non-deleted
// invoice sequences. It satisfies the EntityChecker interface so
// CompanyService.Delete can refuse to soft-delete a company that still
// owns sequences.
type SequenceCompanyChecker struct {
	repo repository.InvoiceSequenceRepo
}

// NewSequenceCompanyChecker creates a new SequenceCompanyChecker.
func NewSequenceCompanyChecker(repo repository.InvoiceSequenceRepo) *SequenceCompanyChecker {
	return &SequenceCompanyChecker{repo: repo}
}

// CountNonDeletedForCompany returns the number of non-deleted invoice
// sequences belonging to the given company.
func (c *SequenceCompanyChecker) CountNonDeletedForCompany(ctx context.Context, companyID int64) (int, error) {
	list, err := c.repo.List(ctx, companyID)
	if err != nil {
		return 0, fmt.Errorf("counting sequences for company %d: %w", companyID, err)
	}
	return len(list), nil
}
