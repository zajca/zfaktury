package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
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

// Create validates uniqueness (prefix+year) and persists a new invoice sequence.
func (s *SequenceService) Create(ctx context.Context, seq *domain.InvoiceSequence) error {
	if seq.Prefix == "" {
		return errors.New("prefix is required")
	}
	if seq.Year == 0 {
		return errors.New("year is required")
	}
	if seq.NextNumber <= 0 {
		seq.NextNumber = 1
	}
	if seq.FormatPattern == "" {
		seq.FormatPattern = "{prefix}{year}{number:04d}"
	}

	// Check uniqueness of prefix+year.
	existing, err := s.repo.GetByPrefixAndYear(ctx, seq.Prefix, seq.Year)
	if err == nil && existing != nil {
		return fmt.Errorf("sequence with prefix %q and year %d already exists", seq.Prefix, seq.Year)
	}
	// Only proceed if the error indicates not found.
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("checking sequence uniqueness: %w", err)
	}

	if err := s.repo.Create(ctx, seq); err != nil {
		return fmt.Errorf("creating sequence: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "sequence", seq.ID, "create", nil, seq)
	}
	return nil
}

// Update validates and updates an existing invoice sequence.
// Prevents lowering next_number below already-used numbers.
func (s *SequenceService) Update(ctx context.Context, seq *domain.InvoiceSequence) error {
	if seq.ID == 0 {
		return errors.New("sequence ID is required")
	}
	if seq.Prefix == "" {
		return errors.New("prefix is required")
	}
	if seq.Year == 0 {
		return errors.New("year is required")
	}
	if seq.NextNumber <= 0 {
		return errors.New("next number must be positive")
	}

	// Fetch existing for audit logging.
	existing, err := s.repo.GetByID(ctx, seq.ID)
	if err != nil {
		return fmt.Errorf("fetching existing sequence: %w", err)
	}

	// Prevent lowering next_number below already-used numbers.
	maxUsed, err := s.repo.MaxUsedNumber(ctx, seq.ID)
	if err != nil {
		return fmt.Errorf("checking max used number: %w", err)
	}
	if seq.NextNumber <= maxUsed {
		return fmt.Errorf("cannot set next number to %d, numbers up to %d have already been assigned", seq.NextNumber, maxUsed)
	}

	// Check uniqueness of prefix+year if they changed.
	duplicate, err := s.repo.GetByPrefixAndYear(ctx, seq.Prefix, seq.Year)
	if err == nil && duplicate != nil && duplicate.ID != seq.ID {
		return fmt.Errorf("sequence with prefix %q and year %d already exists", seq.Prefix, seq.Year)
	}

	if err := s.repo.Update(ctx, seq); err != nil {
		return fmt.Errorf("updating sequence: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "sequence", seq.ID, "update", existing, seq)
	}
	return nil
}

// Delete removes an invoice sequence by ID (soft delete).
// Returns an error if invoices reference this sequence.
func (s *SequenceService) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return errors.New("sequence ID is required")
	}

	count, err := s.repo.CountInvoicesBySequenceID(ctx, id)
	if err != nil {
		return fmt.Errorf("checking invoice references: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete sequence: %d invoices reference it", count)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting sequence: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "sequence", id, "delete", nil, nil)
	}
	return nil
}

// GetByID retrieves a sequence by its ID.
func (s *SequenceService) GetByID(ctx context.Context, id int64) (*domain.InvoiceSequence, error) {
	if id == 0 {
		return nil, errors.New("sequence ID is required")
	}
	seq, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching sequence: %w", err)
	}
	return seq, nil
}

// List retrieves all invoice sequences.
func (s *SequenceService) List(ctx context.Context) ([]domain.InvoiceSequence, error) {
	seqs, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing sequences: %w", err)
	}
	return seqs, nil
}

// GetOrCreateForYear retrieves an existing sequence for the given prefix and year,
// or creates a new one if it doesn't exist.
func (s *SequenceService) GetOrCreateForYear(ctx context.Context, prefix string, year int) (*domain.InvoiceSequence, error) {
	if prefix == "" {
		return nil, errors.New("prefix is required")
	}
	if year == 0 {
		return nil, errors.New("year is required")
	}

	seq, err := s.repo.GetByPrefixAndYear(ctx, prefix, year)
	if err == nil {
		return seq, nil
	}

	// Not found -- create a new sequence for this year.
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("looking up sequence: %w", err)
	}

	newSeq := &domain.InvoiceSequence{
		Prefix:        prefix,
		NextNumber:    1,
		Year:          year,
		FormatPattern: "{prefix}{year}{number:04d}",
	}
	if err := s.repo.Create(ctx, newSeq); err != nil {
		// Race condition: another goroutine may have created the sequence concurrently.
		// Retry the lookup; if found, use that sequence.
		retrySeq, retryErr := s.repo.GetByPrefixAndYear(ctx, prefix, year)
		if retryErr == nil {
			return retrySeq, nil
		}
		return nil, fmt.Errorf("creating sequence for year %d: %w", year, err)
	}
	return newSeq, nil
}

// FormatPreview returns a preview of the next formatted invoice number for a sequence.
// NOTE: format_pattern is not yet implemented; using hardcoded format.
// This must stay consistent with InvoiceRepository.GetNextNumber.
func FormatPreview(seq *domain.InvoiceSequence) string {
	return fmt.Sprintf("%s%d%04d", seq.Prefix, seq.Year, seq.NextNumber)
}
