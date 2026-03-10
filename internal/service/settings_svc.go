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
	return s.repo.GetAll(ctx)
}

// Get retrieves a single setting by key.
func (s *SettingsService) Get(ctx context.Context, key string) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	return s.repo.Get(ctx, key)
}

// Set upserts a single setting.
func (s *SettingsService) Set(ctx context.Context, key, value string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	return s.repo.Set(ctx, key, value)
}

// SetBulk upserts multiple settings at once.
func (s *SettingsService) SetBulk(ctx context.Context, settings map[string]string) error {
	for key := range settings {
		if err := validateKey(key); err != nil {
			return err
		}
	}
	return s.repo.SetBulk(ctx, settings)
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
	SettingIBAN:          true,
	SettingSWIFT:         true,
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
