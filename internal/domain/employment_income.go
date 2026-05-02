package domain

import "time"

// EmploymentDocumentKind classifies an uploaded employment document.
type EmploymentDocumentKind string

const (
	// EmploymentDocAdvance corresponds to "Potvrzení o zdanitelných příjmech ze
	// závislé činnosti" form 25 5460 vzor 33 (zálohové).
	EmploymentDocAdvance EmploymentDocumentKind = "advance"
	// EmploymentDocWithholding corresponds to form 25 5460/A vzor 12 (srážková daň).
	EmploymentDocWithholding EmploymentDocumentKind = "withholding"
	// EmploymentDocBonus corresponds to "Potvrzení o vyplaceném daňovém bonusu".
	// Reserved for future use; no upload UI in MVP.
	EmploymentDocBonus EmploymentDocumentKind = "bonus"
)

// CertificateType classifies the parsed Potvrzení certificate.
type CertificateType string

const (
	// CertificateAdvance maps to vzor 33 (zálohové daně).
	CertificateAdvance CertificateType = "advance"
	// CertificateWithholding maps to vzor 12 (srážková daň §36/6/7).
	CertificateWithholding CertificateType = "withholding"
)

// ContractType describes the underlying employment contract.
type ContractType string

const (
	// ContractDPC = Dohoda o pracovní činnosti.
	ContractDPC ContractType = "dpc"
	// ContractDPP = Dohoda o provedení práce.
	ContractDPP ContractType = "dpp"
	// ContractHPP = Hlavní pracovní poměr.
	ContractHPP ContractType = "hpp"
	// ContractOther covers any other contract form.
	ContractOther ContractType = "other"
)

// EmploymentDocument represents an uploaded Potvrzení (PDF/JPG/PNG/WEBP) for a
// given year. Extraction status follows the same pending/extracted/failed
// lifecycle as InvestmentDocument.
type EmploymentDocument struct {
	ID               int64
	Year             int
	Kind             EmploymentDocumentKind
	Filename         string
	ContentType      string
	StoragePath      string
	Size             int64
	ExtractionStatus string
	ExtractionError  string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// EmploymentCertificate is one parsed/manually entered Potvrzení o zdanitelných
// příjmech ze závislé činnosti. A single EmploymentCertificate maps to a single
// (employer, certificate_type, period) tuple per year.
type EmploymentCertificate struct {
	ID                      int64
	Year                    int
	DocumentID              *int64
	CertificateType         CertificateType
	EmployerName            string
	EmployerICO             string
	EmployerAddress         string
	ContractType            ContractType
	PeriodFrom              time.Time
	PeriodTo                time.Time
	GrossIncome             Amount // ř.2 + ř.4 Potvrzení -> ř.31 DAP
	IncomeWithoutAdvance    Amount // bez záloh dle §38h -> ř.35 DAP
	ForeignTaxPaid          Amount // §6 odst.13 -> ř.33 DAP
	AdvanceTaxWithheld      Amount // ř.8 Potvrzení -> ř.84 DAP
	AnnualSettlementRefund  Amount // vrácený přeplatek z RZ
	MonthlyBonusPaid        Amount // ř.5 + ř.13 Potvrzení -> ř.89 DAP
	WithheldFinalTax        Amount
	IncludeWithholdingInDAP bool
	Notes                   string
	Status                  string
	DeletedAt               *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
