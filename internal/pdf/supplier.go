package pdf

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
