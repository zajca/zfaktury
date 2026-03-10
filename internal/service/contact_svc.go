package service

import (
	"context"
	"errors"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// ARESClient defines the interface for looking up companies via the ARES registry.
type ARESClient interface {
	LookupByICO(ctx context.Context, ico string) (*domain.Contact, error)
}

// ContactService provides business logic for contact management.
type ContactService struct {
	repo repository.ContactRepo
	ares ARESClient
}

// NewContactService creates a new ContactService.
func NewContactService(repo repository.ContactRepo, ares ARESClient) *ContactService {
	return &ContactService{
		repo: repo,
		ares: ares,
	}
}

// Create validates and persists a new contact.
func (s *ContactService) Create(ctx context.Context, contact *domain.Contact) error {
	if contact.Name == "" {
		return errors.New("contact name is required")
	}
	if contact.Type == "" {
		contact.Type = domain.ContactTypeCompany
	}
	if contact.Type != domain.ContactTypeCompany && contact.Type != domain.ContactTypeIndividual {
		return errors.New("contact type must be 'company' or 'individual'")
	}
	return s.repo.Create(ctx, contact)
}

// Update validates and updates an existing contact.
func (s *ContactService) Update(ctx context.Context, contact *domain.Contact) error {
	if contact.ID == 0 {
		return errors.New("contact ID is required")
	}
	if contact.Name == "" {
		return errors.New("contact name is required")
	}
	if contact.Type != domain.ContactTypeCompany && contact.Type != domain.ContactTypeIndividual {
		return errors.New("contact type must be 'company' or 'individual'")
	}
	return s.repo.Update(ctx, contact)
}

// Delete removes a contact by ID (soft delete).
func (s *ContactService) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return errors.New("contact ID is required")
	}
	return s.repo.Delete(ctx, id)
}

// GetByID retrieves a contact by its ID.
func (s *ContactService) GetByID(ctx context.Context, id int64) (*domain.Contact, error) {
	if id == 0 {
		return nil, errors.New("contact ID is required")
	}
	return s.repo.GetByID(ctx, id)
}

// List retrieves contacts matching the given filter.
// Returns the contacts, total count, and any error.
func (s *ContactService) List(ctx context.Context, filter domain.ContactFilter) ([]domain.Contact, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.List(ctx, filter)
}

// LookupARES looks up a company by ICO using the ARES registry.
func (s *ContactService) LookupARES(ctx context.Context, ico string) (*domain.Contact, error) {
	if ico == "" {
		return nil, errors.New("ICO is required")
	}
	if s.ares == nil {
		return nil, errors.New("ARES client is not configured")
	}
	return s.ares.LookupByICO(ctx, ico)
}
