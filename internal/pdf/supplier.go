package pdf

import "github.com/zajca/zfaktury/internal/domain"

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

// SupplierFromCompany builds SupplierInfo from a company record. Migration 025
// made the company struct the single source of truth for supplier identity, so
// both the HTTP send-email handler and the recurring auto-send path map from it
// here to avoid duplicating the field mapping.
func SupplierFromCompany(c *domain.Company) SupplierInfo {
	if c == nil {
		return SupplierInfo{}
	}
	return SupplierInfo{
		Name:          c.LegalName,
		ICO:           c.ICO,
		DIC:           c.DIC,
		VATRegistered: c.VATRegistered,
		Street:        c.Street,
		City:          c.City,
		ZIP:           c.ZIP,
		Email:         c.Email,
		Phone:         c.Phone,
		BankAccount:   c.BankAccount,
		BankCode:      c.BankCode,
		IBAN:          c.IBAN,
		SWIFT:         c.SWIFT,
	}
}
