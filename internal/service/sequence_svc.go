package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// SequenceService provides business logic for invoice sequence management.
type SequenceService struct {
	repo repository.InvoiceSequenceRepo
}

// NewSequenceService creates a new SequenceService.
func NewSequenceService(repo repository.InvoiceSequenceRepo) *SequenceService {
	return &SequenceService{repo: repo}
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
	if err != nil && !strings.Contains(err.Error(), sql.ErrNoRows.Error()) {
		return fmt.Errorf("checking sequence uniqueness: %w", err)
	}

	return s.repo.Create(ctx, seq)
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

	// Prevent lowering next_number below already-used numbers.
	maxUsed, err := s.repo.MaxUsedNumber(ctx, seq.ID)
	if err != nil {
		return fmt.Errorf("checking max used number: %w", err)
	}
	if seq.NextNumber <= maxUsed {
		return fmt.Errorf("cannot set next number to %d, numbers up to %d have already been assigned", seq.NextNumber, maxUsed)
	}

	// Check uniqueness of prefix+year if they changed.
	existing, err := s.repo.GetByPrefixAndYear(ctx, seq.Prefix, seq.Year)
	if err == nil && existing != nil && existing.ID != seq.ID {
		return fmt.Errorf("sequence with prefix %q and year %d already exists", seq.Prefix, seq.Year)
	}

	return s.repo.Update(ctx, seq)
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

	return s.repo.Delete(ctx, id)
}

// GetByID retrieves a sequence by its ID.
func (s *SequenceService) GetByID(ctx context.Context, id int64) (*domain.InvoiceSequence, error) {
	if id == 0 {
		return nil, errors.New("sequence ID is required")
	}
	return s.repo.GetByID(ctx, id)
}

// List retrieves all invoice sequences.
func (s *SequenceService) List(ctx context.Context) ([]domain.InvoiceSequence, error) {
	return s.repo.List(ctx)
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
	if !strings.Contains(err.Error(), sql.ErrNoRows.Error()) {
		return nil, fmt.Errorf("looking up sequence: %w", err)
	}

	newSeq := &domain.InvoiceSequence{
		Prefix:        prefix,
		NextNumber:    1,
		Year:          year,
		FormatPattern: "{prefix}{year}{number:04d}",
	}
	if err := s.repo.Create(ctx, newSeq); err != nil {
		return nil, fmt.Errorf("creating sequence for year %d: %w", year, err)
	}
	return newSeq, nil
}

// FormatPreview returns a preview of the next formatted invoice number for a sequence.
func FormatPreview(seq *domain.InvoiceSequence) string {
	return fmt.Sprintf("%s%d%04d", seq.Prefix, seq.Year, seq.NextNumber)
}
