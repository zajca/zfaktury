package filename

import (
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

func TestPeriodNames(t *testing.T) {
	monthly := domain.TaxPeriod{Year: 2025, Month: 3}
	quarterly := domain.TaxPeriod{Year: 2025, Quarter: 1}
	december := domain.TaxPeriod{Year: 2025, Month: 12}
	q4 := domain.TaxPeriod{Year: 2025, Quarter: 4}

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"vat monthly regular", VATReturn(monthly, domain.FilingTypeRegular), "dph-2025-03.xml"},
		{"vat december", VATReturn(december, domain.FilingTypeRegular), "dph-2025-12.xml"},
		{"vat quarterly", VATReturn(quarterly, domain.FilingTypeRegular), "dph-2025-q1.xml"},
		{"vat quarter 4", VATReturn(q4, domain.FilingTypeRegular), "dph-2025-q4.xml"},
		{"vat corrective", VATReturn(monthly, domain.FilingTypeCorrective), "dph-2025-03-opravne.xml"},
		{"vat supplementary", VATReturn(quarterly, domain.FilingTypeSupplementary), "dph-2025-q1-dodatecne.xml"},
		{"vat empty filing type", VATReturn(monthly, ""), "dph-2025-03.xml"},
		{"vat unknown filing type", VATReturn(monthly, "other"), "dph-2025-03.xml"},
		{"vat period without month or quarter", VATReturn(domain.TaxPeriod{Year: 2025}, ""), "dph-2025.xml"},
		{"control regular", ControlStatement(monthly, domain.FilingTypeRegular), "kh-2025-03.xml"},
		{"control quarterly", ControlStatement(quarterly, domain.FilingTypeRegular), "kh-2025-q1.xml"},
		{"control corrective", ControlStatement(monthly, domain.FilingTypeCorrective), "kh-2025-03-opravne.xml"},
		{"control follow-up", ControlStatement(monthly, domain.FilingTypeSupplementary), "kh-2025-03-nasledne.xml"},
		{"summary monthly", SummaryStatement(monthly, domain.FilingTypeRegular), "sh-2025-03.xml"},
		{"summary quarterly", SummaryStatement(quarterly, domain.FilingTypeRegular), "sh-2025-q1.xml"},
		{"summary corrective", SummaryStatement(quarterly, domain.FilingTypeCorrective), "sh-2025-q1-opravne.xml"},
		{"summary supplementary", SummaryStatement(monthly, domain.FilingTypeSupplementary), "sh-2025-03-dodatecne.xml"},
		{"income tax regular", IncomeTax(2025, domain.FilingTypeRegular), "dpfo-2025.xml"},
		{"income tax corrective", IncomeTax(2025, domain.FilingTypeCorrective), "dpfo-2025-opravne.xml"},
		{"income tax supplementary", IncomeTax(2025, domain.FilingTypeSupplementary), "dpfo-2025-dodatecne.xml"},
		{"income tax bundle", IncomeTaxBundle(2025, domain.FilingTypeRegular), "dpfo-2025.zip"},
		{"income tax bundle corrective", IncomeTaxBundle(2025, domain.FilingTypeCorrective), "dpfo-2025-opravne.zip"},
		{"social regular", SocialInsurance(2025, domain.FilingTypeRegular), "cssz-2025.xml"},
		{"social corrective", SocialInsurance(2025, domain.FilingTypeCorrective), "cssz-2025-opravne.xml"},
		{"social supplementary", SocialInsurance(2025, domain.FilingTypeSupplementary), "cssz-2025-dodatecne.xml"},
		{"health regular", HealthInsurance(2025, domain.FilingTypeRegular), "zp-2025.xml"},
		{"health corrective", HealthInsurance(2025, domain.FilingTypeCorrective), "zp-2025-opravne.xml"},
		{"health supplementary", HealthInsurance(2025, domain.FilingTypeSupplementary), "zp-2025-dodatecne.xml"},
		{"isdoc export", ISDOCExport(), "faktury-isdoc.zip"},
		{"invoices csv", InvoicesCSV(2025), "faktury-2025.csv"},
		{"expenses csv", ExpensesCSV(2025), "vydaje-2025.csv"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestInvoiceNames(t *testing.T) {
	tests := []struct {
		number   string
		wantBase string
	}{
		{"2025001", "faktura-2025001"},
		{"FV-2025-001", "faktura-fv-2025-001"},
		{"FV/2025/001", "faktura-fv-2025-001"},
		{"FV 2025/001", "faktura-fv-2025-001"},
		{"  2025/001  ", "faktura-2025-001"},
		{"../etc/passwd", "faktura-etc-passwd"},
		{`a\b:c*d?e"f<g>h|i`, "faktura-a-b-c-d-e-f-g-h-i"},
		{"FA--001", "faktura-fa-001"},
		{"-001-", "faktura-001"},
		{"cislo č. 5", "faktura-cislo-5"},
		{"", "faktura"},
		{"///", "faktura"},
	}
	for _, tt := range tests {
		t.Run(tt.number, func(t *testing.T) {
			if got := InvoicePDF(tt.number); got != tt.wantBase+".pdf" {
				t.Errorf("InvoicePDF(%q) = %q, want %q", tt.number, got, tt.wantBase+".pdf")
			}
			if got := InvoiceISDOC(tt.number); got != tt.wantBase+".isdoc" {
				t.Errorf("InvoiceISDOC(%q) = %q, want %q", tt.number, got, tt.wantBase+".isdoc")
			}
		})
	}
}

func TestSanitizeInvoiceNumber(t *testing.T) {
	tests := []struct{ in, want string }{
		{"AbC-123", "abc-123"},
		{"a  b", "a-b"},
		{"-", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := SanitizeInvoiceNumber(tt.in); got != tt.want {
			t.Errorf("SanitizeInvoiceNumber(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestContentDisposition(t *testing.T) {
	if got, want := ContentDisposition("dph-2025-03.xml"), "attachment; filename=dph-2025-03.xml"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
