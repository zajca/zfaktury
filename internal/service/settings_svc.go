package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/zajca/zfaktury/internal/repository"
)

// Known setting keys for the application.
const (
	SettingCompanyName   = "company_name"
	SettingICO           = "ico"
	SettingDIC           = "dic"
	SettingVATRegistered = "vat_registered"
	SettingStreet        = "street"
	SettingCity          = "city"
	SettingZIP           = "zip"
	SettingEmail         = "email"
	SettingPhone         = "phone"
	SettingBankAccount   = "bank_account"
	SettingBankCode      = "bank_code"
	SettingIBAN          = "iban"
	SettingSWIFT         = "swift"

	// Email settings.
	SettingEmailAttachPDF      = "email_attach_pdf"
	SettingEmailAttachISDOC    = "email_attach_isdoc"
	SettingEmailSubjectTpl     = "email_subject_template"
	SettingEmailBodyTpl        = "email_body_template"

	// Office codes (global, rarely change).
	SettingHealthInsCode      = "health_insurance_code"
	SettingFinancniUradCode   = "financni_urad_code"
	SettingCSSZCode           = "cssz_code"
)

// SettingsService provides business logic for application settings.
type SettingsService struct {
	repo *repository.SettingsRepository
}

// NewSettingsService creates a new SettingsService.
func NewSettingsService(repo *repository.SettingsRepository) *SettingsService {
	return &SettingsService{repo: repo}
}

// GetAll retrieves all settings.
func (s *SettingsService) GetAll(ctx context.Context) (map[string]string, error) {
	settings, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching all settings: %w", err)
	}
	return settings, nil
}

// Get retrieves a single setting by key.
func (s *SettingsService) Get(ctx context.Context, key string) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	val, err := s.repo.Get(ctx, key)
	if err != nil {
		return "", fmt.Errorf("fetching setting %q: %w", key, err)
	}
	return val, nil
}

// Set upserts a single setting.
func (s *SettingsService) Set(ctx context.Context, key, value string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := s.repo.Set(ctx, key, value); err != nil {
		return fmt.Errorf("setting %q: %w", key, err)
	}
	return nil
}

// SetBulk upserts multiple settings at once.
func (s *SettingsService) SetBulk(ctx context.Context, settings map[string]string) error {
	for key := range settings {
		if err := validateKey(key); err != nil {
			return err
		}
	}
	if err := s.repo.SetBulk(ctx, settings); err != nil {
		return fmt.Errorf("setting bulk settings: %w", err)
	}
	return nil
}

// knownKeys contains all valid setting keys.
var knownKeys = map[string]bool{
	SettingCompanyName:   true,
	SettingICO:           true,
	SettingDIC:           true,
	SettingVATRegistered: true,
	SettingStreet:        true,
	SettingCity:          true,
	SettingZIP:           true,
	SettingEmail:         true,
	SettingPhone:         true,
	SettingBankAccount:   true,
	SettingBankCode:      true,
	SettingIBAN:              true,
	SettingSWIFT:             true,
	SettingEmailAttachPDF:    true,
	SettingEmailAttachISDOC:  true,
	SettingEmailSubjectTpl:   true,
	SettingEmailBodyTpl:      true,
	SettingHealthInsCode:     true,
	SettingFinancniUradCode:  true,
	SettingCSSZCode:          true,
}

// validateKey checks that a setting key is valid and known.
func validateKey(key string) error {
	if key == "" {
		return errors.New("setting key is required")
	}
	if !knownKeys[key] {
		return fmt.Errorf("unknown setting key: %q", key)
	}
	return nil
}
