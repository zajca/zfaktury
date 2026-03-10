package pdf

import (
	"context"

	"github.com/zajca/zfaktury/internal/service"
)

// SupplierInfo holds supplier details loaded from application settings for PDF generation.
type SupplierInfo struct {
	Name          string
	ICO           string
	DIC           string
	VATRegistered bool
	Street        string
	City          string
	ZIP           string
	Email         string
	Phone         string
	BankAccount   string
	BankCode      string
	IBAN          string
	SWIFT         string
	LogoPath      string
}

// LoadSupplierFromSettings loads supplier information from the settings service.
func LoadSupplierFromSettings(ctx context.Context, settingsSvc *service.SettingsService) (SupplierInfo, error) {
	settings, err := settingsSvc.GetAll(ctx)
	if err != nil {
		return SupplierInfo{}, err
	}

	return SupplierInfo{
		Name:          settings[service.SettingCompanyName],
		ICO:           settings[service.SettingICO],
		DIC:           settings[service.SettingDIC],
		VATRegistered: settings[service.SettingVATRegistered] == "true",
		Street:        settings[service.SettingStreet],
		City:          settings[service.SettingCity],
		ZIP:           settings[service.SettingZIP],
		Email:         settings[service.SettingEmail],
		Phone:         settings[service.SettingPhone],
		BankAccount:   settings[service.SettingBankAccount],
		BankCode:      settings[service.SettingBankCode],
		IBAN:          settings[service.SettingIBAN],
		SWIFT:         settings[service.SettingSWIFT],
	}, nil
}
