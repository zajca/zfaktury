package isdoc

import "encoding/xml"

// ISDOC 6.0.2 XML namespace.
const ISDOCNamespace = "urn:isdoc:invoice:6.0.2"

// Invoice is the root element of an ISDOC 6.0.2 document.
type Invoice struct {
	XMLName            xml.Name            `xml:"Invoice"`
	Xmlns              string              `xml:"xmlns,attr"`
	Version            string              `xml:"version,attr"`
	DocumentType       int                 `xml:"DocumentType"`
	ID                 string              `xml:"ID"`
	UUID               string              `xml:"UUID"`
	IssuingSystem      string              `xml:"IssuingSystem"`
	IssueDate          string              `xml:"IssueDate"`
	TaxPointDate       string              `xml:"TaxPointDate"`
	VATApplicable      bool                `xml:"VATApplicable"`
	Note               string              `xml:"Note,omitempty"`
	LocalCurrencyCode  string              `xml:"LocalCurrencyCode"`
	ForeignCurrencyCode string             `xml:"ForeignCurrencyCode,omitempty"`
	CurrRate           string              `xml:"CurrRate,omitempty"`
	RefCurrRate        string              `xml:"RefCurrRate,omitempty"`
	AccountingSupplierParty AccountingParty `xml:"AccountingSupplierParty"`
	AccountingCustomerParty AccountingParty `xml:"AccountingCustomerParty"`
	InvoiceLines       InvoiceLines        `xml:"InvoiceLines"`
	TaxTotal           TaxTotal            `xml:"TaxTotal"`
	LegalMonetaryTotal LegalMonetaryTotal  `xml:"LegalMonetaryTotal"`
	PaymentMeans       *PaymentMeans       `xml:"PaymentMeans,omitempty"`
}

// AccountingParty represents either the supplier or customer party.
type AccountingParty struct {
	Party Party `xml:"Party"`
}

// Party contains party identification and address details.
type Party struct {
	PartyIdentification PartyIdentification `xml:"PartyIdentification"`
	PartyName           PartyName           `xml:"PartyName"`
	PostalAddress       PostalAddress       `xml:"PostalAddress"`
	PartyTaxScheme      *PartyTaxScheme     `xml:"PartyTaxScheme,omitempty"`
	Contact             *PartyContact       `xml:"Contact,omitempty"`
}

// PartyIdentification holds the party's business ID (ICO).
type PartyIdentification struct {
	UserID string `xml:"UserID"`
	ID     string `xml:"ID"`
}

// PartyName holds the party name.
type PartyName struct {
	Name string `xml:"Name"`
}

// PostalAddress represents a postal address.
type PostalAddress struct {
	StreetName   string `xml:"StreetName"`
	BuildingNumber string `xml:"BuildingNumber,omitempty"`
	CityName     string `xml:"CityName"`
	PostalZone   string `xml:"PostalZone"`
	Country      Country `xml:"Country"`
}

// Country holds a country identification code.
type Country struct {
	IdentificationCode string `xml:"IdentificationCode"`
	Name               string `xml:"Name,omitempty"`
}

// PartyTaxScheme holds VAT identification (DIC).
type PartyTaxScheme struct {
	CompanyID string    `xml:"CompanyID"`
	TaxScheme TaxScheme `xml:"TaxScheme"`
}

// TaxScheme identifies the tax scheme (e.g. VAT).
type TaxScheme struct {
	Name string `xml:"Name"`
}

// PartyContact holds contact details for a party.
type PartyContact struct {
	Telephone       string `xml:"Telephone,omitempty"`
	ElectronicMail  string `xml:"ElectronicMail,omitempty"`
}

// InvoiceLines is a container for invoice line items.
type InvoiceLines struct {
	InvoiceLine []InvoiceLine `xml:"InvoiceLine"`
}

// InvoicedQuantity represents a quantity with an optional unit code attribute.
type InvoicedQuantity struct {
	Value    string `xml:",chardata"`
	UnitCode string `xml:"unitCode,attr,omitempty"`
}

// InvoiceLine represents a single line item on the invoice.
type InvoiceLine struct {
	ID                    string            `xml:"ID"`
	InvoicedQuantity      InvoicedQuantity  `xml:"InvoicedQuantity"`
	LineExtensionAmount   string            `xml:"LineExtensionAmount"`
	LineExtensionAmountTaxInclusive string `xml:"LineExtensionAmountTaxInclusive"`
	LineExtensionTaxAmount string     `xml:"LineExtensionTaxAmount"`
	UnitPrice             string      `xml:"UnitPrice"`
	UnitPriceTaxInclusive string      `xml:"UnitPriceTaxInclusive"`
	ClassifiedTaxCategory ClassifiedTaxCategory `xml:"ClassifiedTaxCategory"`
	Item                  Item        `xml:"Item,omitempty"`
}

// ClassifiedTaxCategory holds the VAT rate for a line item.
type ClassifiedTaxCategory struct {
	Percent       string    `xml:"Percent"`
	VATCalculationMethod int `xml:"VATCalculationMethod"`
}

// Item holds line item description.
type Item struct {
	Description string `xml:"Description"`
}

// TaxTotal contains aggregated tax information.
type TaxTotal struct {
	TaxSubTotal []TaxSubTotal `xml:"TaxSubTotal"`
	TaxAmount   string        `xml:"TaxAmount"`
}

// TaxSubTotal is a breakdown of tax by rate.
type TaxSubTotal struct {
	TaxableAmount string      `xml:"TaxableAmount"`
	TaxAmount     string      `xml:"TaxAmount"`
	TaxInclusiveAmount string `xml:"TaxInclusiveAmount"`
	AlreadyClaimedTaxableAmount string `xml:"AlreadyClaimedTaxableAmount"`
	AlreadyClaimedTaxAmount string `xml:"AlreadyClaimedTaxAmount"`
	AlreadyClaimedTaxInclusiveAmount string `xml:"AlreadyClaimedTaxInclusiveAmount"`
	DifferenceTaxableAmount string `xml:"DifferenceTaxableAmount"`
	DifferenceTaxAmount string `xml:"DifferenceTaxAmount"`
	DifferenceTaxInclusiveAmount string `xml:"DifferenceTaxInclusiveAmount"`
	TaxCategory  TaxCategory `xml:"TaxCategory"`
}

// TaxCategory identifies a tax rate category.
type TaxCategory struct {
	Percent string `xml:"Percent"`
}

// LegalMonetaryTotal holds the invoice totals.
type LegalMonetaryTotal struct {
	TaxExclusiveAmount string `xml:"TaxExclusiveAmount"`
	TaxInclusiveAmount string `xml:"TaxInclusiveAmount"`
	AlreadyClaimedTaxExclusiveAmount string `xml:"AlreadyClaimedTaxExclusiveAmount"`
	AlreadyClaimedTaxInclusiveAmount string `xml:"AlreadyClaimedTaxInclusiveAmount"`
	DifferenceTaxExclusiveAmount string `xml:"DifferenceTaxExclusiveAmount"`
	DifferenceTaxInclusiveAmount string `xml:"DifferenceTaxInclusiveAmount"`
	PaidDepositsAmount string `xml:"PaidDepositsAmount"`
	PayableAmount      string `xml:"PayableAmount"`
}

// PaymentMeans holds payment information.
type PaymentMeans struct {
	Payment    Payment    `xml:"Payment"`
	PaymentDueDate string `xml:"PaymentDueDate"`
}

// Payment contains payment details like bank account.
type Payment struct {
	PaidAmount       string        `xml:"PaidAmount"`
	PaymentMeansCode int           `xml:"PaymentMeansCode"`
	Details          *PaymentDetails `xml:"Details,omitempty"`
}

// PaymentDetails holds bank account details.
type PaymentDetails struct {
	PaymentDueDate string `xml:"PaymentDueDate,omitempty"`
	ID             string `xml:"ID,omitempty"`
	BankCode       string `xml:"BankCode,omitempty"`
	Name           string `xml:"Name,omitempty"`
	IBAN           string `xml:"IBAN,omitempty"`
	BIC            string `xml:"BIC,omitempty"`
	VariableSymbol string `xml:"VariableSymbol,omitempty"`
	ConstantSymbol string `xml:"ConstantSymbol,omitempty"`
}
