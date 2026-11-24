// Package filename builds the names of every generated file the application
// offers for download. Names are lowercase ASCII, hyphen-separated and follow
// <kind>-<period>[-<variant>].<ext>, where the period is the filing's period
// and never a database id.
package filename

import (
	"fmt"
	"mime"
	"strings"

	"github.com/zajca/zfaktury/internal/domain"
)

// Variant suffixes appended after the period.
const (
	variantCorrective    = "opravne"
	variantSupplementary = "dodatecne"
	variantFollowUp      = "nasledne"
)

// Kind prefixes and extensions.
const (
	kindVATReturn        = "dph"
	kindControlStatement = "kh"
	kindSummaryStatement = "sh"
	kindIncomeTax        = "dpfo"
	kindSocialInsurance  = "cssz"
	kindHealthInsurance  = "zp"
	kindInvoice          = "faktura"
	kindInvoicesCSV      = "faktury"
	kindExpensesCSV      = "vydaje"

	extXML   = "xml"
	extZIP   = "zip"
	extPDF   = "pdf"
	extISDOC = "isdoc"
	extCSV   = "csv"
)

// VATReturn returns the name of a VAT return XML, e.g. dph-2025-03.xml or dph-2025-q1-opravne.xml.
func VATReturn(period domain.TaxPeriod, filingType string) string {
	return build(kindVATReturn, periodPart(period), variant(filingType, false), extXML)
}

// ControlStatement returns the name of a control statement XML, e.g. kh-2025-03.xml.
// A supplementary control statement is a follow-up one (nasledne).
func ControlStatement(period domain.TaxPeriod, filingType string) string {
	return build(kindControlStatement, periodPart(period), variant(filingType, true), extXML)
}

// SummaryStatement returns the name of a VIES summary statement XML, e.g. sh-2025-q1.xml.
func SummaryStatement(period domain.TaxPeriod, filingType string) string {
	return build(kindSummaryStatement, periodPart(period), variant(filingType, false), extXML)
}

// IncomeTax returns the name of an income tax return XML, e.g. dpfo-2025.xml.
func IncomeTax(year int, filingType string) string {
	return build(kindIncomeTax, yearPart(year), variant(filingType, false), extXML)
}

// IncomeTaxBundle returns the name of an income tax return ZIP bundle, e.g. dpfo-2025.zip.
func IncomeTaxBundle(year int, filingType string) string {
	return build(kindIncomeTax, yearPart(year), variant(filingType, false), extZIP)
}

// SocialInsurance returns the name of a CSSZ overview XML, e.g. cssz-2025.xml.
func SocialInsurance(year int, filingType string) string {
	return build(kindSocialInsurance, yearPart(year), variant(filingType, false), extXML)
}

// HealthInsurance returns the name of a health insurance overview XML, e.g. zp-2025.xml.
func HealthInsurance(year int, filingType string) string {
	return build(kindHealthInsurance, yearPart(year), variant(filingType, false), extXML)
}

// InvoicePDF returns the name of an invoice PDF, e.g. faktura-2025001.pdf.
func InvoicePDF(invoiceNumber string) string {
	return build(kindInvoice, SanitizeInvoiceNumber(invoiceNumber), "", extPDF)
}

// InvoiceISDOC returns the name of an invoice ISDOC file, e.g. faktura-2025001.isdoc.
func InvoiceISDOC(invoiceNumber string) string {
	return build(kindInvoice, SanitizeInvoiceNumber(invoiceNumber), "", extISDOC)
}

// ISDOCExport returns the name of the ISDOC batch export archive.
func ISDOCExport() string {
	return build(kindInvoicesCSV, "isdoc", "", extZIP)
}

// InvoicesCSV returns the name of the yearly invoice CSV export, e.g. faktury-2025.csv.
func InvoicesCSV(year int) string {
	return build(kindInvoicesCSV, yearPart(year), "", extCSV)
}

// ExpensesCSV returns the name of the yearly expense CSV export, e.g. vydaje-2025.csv.
func ExpensesCSV(year int) string {
	return build(kindExpensesCSV, yearPart(year), "", extCSV)
}

// ContentDisposition returns an attachment Content-Disposition header value for name.
func ContentDisposition(name string) string {
	return mime.FormatMediaType("attachment", map[string]string{"filename": name})
}

// SanitizeInvoiceNumber lowercases an invoice number, replaces every character
// outside [a-z0-9-] with a hyphen, collapses repeated hyphens and trims them.
// A number with no usable character yields an empty string.
func SanitizeInvoiceNumber(number string) string {
	var b strings.Builder
	lastHyphen := true // swallows leading hyphens
	for _, r := range number {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
			lastHyphen = false
		default:
			if !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}

func yearPart(year int) string {
	return fmt.Sprintf("%04d", year)
}

// periodPart renders a monthly period as YYYY-MM and a quarterly one as YYYY-qN.
func periodPart(p domain.TaxPeriod) string {
	switch {
	case p.Month > 0:
		return fmt.Sprintf("%04d-%02d", p.Year, p.Month)
	case p.Quarter > 0:
		return fmt.Sprintf("%04d-q%d", p.Year, p.Quarter)
	default:
		return yearPart(p.Year)
	}
}

// variant maps a domain filing type to a name suffix; a regular or unknown
// filing type has none. followUp selects "nasledne" for a supplementary
// filing, which is how a control statement names it.
func variant(filingType string, followUp bool) string {
	switch filingType {
	case domain.FilingTypeCorrective:
		return variantCorrective
	case domain.FilingTypeSupplementary:
		if followUp {
			return variantFollowUp
		}
		return variantSupplementary
	default:
		return ""
	}
}

// build joins non-empty parts with hyphens and appends the extension.
func build(kind, period, variant, ext string) string {
	parts := []string{kind}
	for _, p := range []string{period, variant} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "-") + "." + ext
}
