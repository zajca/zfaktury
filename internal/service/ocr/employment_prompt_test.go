package ocr

import (
	"strings"
	"testing"
)

func TestParseEmploymentResponse_AdvanceVariant(t *testing.T) {
	input := `{
		"certificate_type": "advance",
		"employer_name": "Acme s.r.o.",
		"employer_ico": "12345678",
		"employer_address": "Vaclavske namesti 1, 110 00 Praha 1",
		"contract_type": "dpc",
		"period_from": "2025-01-01",
		"period_to": "2025-12-31",
		"gross_income_czk": 240000.00,
		"income_without_advance_czk": 0.00,
		"foreign_tax_paid_czk": 0.00,
		"advance_tax_withheld_czk": 36000.00,
		"annual_settlement_refund_czk": 1500.00,
		"monthly_bonus_paid_czk": 15300.00,
		"withheld_final_tax_czk": 0.00,
		"confidence": 0.92,
		"raw_text": "Potvrzeni o zdanitelnych prijmech ze zavisle cinnosti..."
	}`

	resp, err := ParseEmploymentResponse(input)
	if err != nil {
		t.Fatalf("ParseEmploymentResponse() error: %v", err)
	}

	if resp.CertificateType != "advance" {
		t.Errorf("CertificateType = %q, want %q", resp.CertificateType, "advance")
	}
	if resp.EmployerName != "Acme s.r.o." {
		t.Errorf("EmployerName = %q, want %q", resp.EmployerName, "Acme s.r.o.")
	}
	if resp.EmployerICO != "12345678" {
		t.Errorf("EmployerICO = %q, want %q", resp.EmployerICO, "12345678")
	}
	if resp.ContractType != "dpc" {
		t.Errorf("ContractType = %q, want %q", resp.ContractType, "dpc")
	}
	if resp.PeriodFrom != "2025-01-01" {
		t.Errorf("PeriodFrom = %q, want %q", resp.PeriodFrom, "2025-01-01")
	}
	if resp.PeriodTo != "2025-12-31" {
		t.Errorf("PeriodTo = %q, want %q", resp.PeriodTo, "2025-12-31")
	}
	if resp.GrossIncomeCZK != 240000.00 {
		t.Errorf("GrossIncomeCZK = %f, want %f", resp.GrossIncomeCZK, 240000.00)
	}
	if resp.AdvanceTaxWithheldCZK != 36000.00 {
		t.Errorf("AdvanceTaxWithheldCZK = %f, want %f", resp.AdvanceTaxWithheldCZK, 36000.00)
	}
	if resp.AnnualSettlementRefundCZK != 1500.00 {
		t.Errorf("AnnualSettlementRefundCZK = %f, want %f", resp.AnnualSettlementRefundCZK, 1500.00)
	}
	if resp.MonthlyBonusPaidCZK != 15300.00 {
		t.Errorf("MonthlyBonusPaidCZK = %f, want %f", resp.MonthlyBonusPaidCZK, 15300.00)
	}
	if resp.WithheldFinalTaxCZK != 0.00 {
		t.Errorf("WithheldFinalTaxCZK = %f, want 0", resp.WithheldFinalTaxCZK)
	}
	if resp.Confidence != 0.92 {
		t.Errorf("Confidence = %f, want %f", resp.Confidence, 0.92)
	}
	if !strings.Contains(resp.RawText, "Potvrzeni") {
		t.Errorf("RawText = %q, expected to contain 'Potvrzeni'", resp.RawText)
	}
}

func TestParseEmploymentResponse_WithholdingVariant(t *testing.T) {
	input := `{
		"certificate_type": "withholding",
		"employer_name": "Beta a.s.",
		"employer_ico": "87654321",
		"employer_address": "Brno, Stara 5",
		"contract_type": "dpp",
		"period_from": "2025-03-01",
		"period_to": "2025-08-31",
		"gross_income_czk": 50000.00,
		"income_without_advance_czk": 0.00,
		"foreign_tax_paid_czk": 0.00,
		"advance_tax_withheld_czk": 0.00,
		"annual_settlement_refund_czk": 0.00,
		"monthly_bonus_paid_czk": 0.00,
		"withheld_final_tax_czk": 7500.00,
		"confidence": 0.88,
		"raw_text": "Potvrzeni 25 5460/A vzor c. 12 ..."
	}`

	resp, err := ParseEmploymentResponse(input)
	if err != nil {
		t.Fatalf("ParseEmploymentResponse() error: %v", err)
	}

	if resp.CertificateType != "withholding" {
		t.Errorf("CertificateType = %q, want %q", resp.CertificateType, "withholding")
	}
	if resp.ContractType != "dpp" {
		t.Errorf("ContractType = %q, want %q", resp.ContractType, "dpp")
	}
	if resp.GrossIncomeCZK != 50000.00 {
		t.Errorf("GrossIncomeCZK = %f, want %f", resp.GrossIncomeCZK, 50000.00)
	}
	if resp.WithheldFinalTaxCZK != 7500.00 {
		t.Errorf("WithheldFinalTaxCZK = %f, want %f", resp.WithheldFinalTaxCZK, 7500.00)
	}
	if resp.AdvanceTaxWithheldCZK != 0.00 {
		t.Errorf("AdvanceTaxWithheldCZK = %f, want 0 for withholding variant", resp.AdvanceTaxWithheldCZK)
	}
	if resp.MonthlyBonusPaidCZK != 0.00 {
		t.Errorf("MonthlyBonusPaidCZK = %f, want 0 for withholding variant", resp.MonthlyBonusPaidCZK)
	}
}

func TestParseEmploymentResponse_MissingFieldsDefaultToZero(t *testing.T) {
	// Minimal JSON: only the certificate type and one amount, all others should default.
	input := `{
		"certificate_type": "advance",
		"gross_income_czk": 120000.0
	}`

	resp, err := ParseEmploymentResponse(input)
	if err != nil {
		t.Fatalf("ParseEmploymentResponse() error: %v", err)
	}

	if resp.CertificateType != "advance" {
		t.Errorf("CertificateType = %q, want %q", resp.CertificateType, "advance")
	}
	if resp.EmployerName != "" {
		t.Errorf("EmployerName = %q, want empty string", resp.EmployerName)
	}
	if resp.EmployerICO != "" {
		t.Errorf("EmployerICO = %q, want empty string", resp.EmployerICO)
	}
	if resp.EmployerAddress != "" {
		t.Errorf("EmployerAddress = %q, want empty string", resp.EmployerAddress)
	}
	if resp.ContractType != "" {
		t.Errorf("ContractType = %q, want empty string", resp.ContractType)
	}
	if resp.PeriodFrom != "" {
		t.Errorf("PeriodFrom = %q, want empty string", resp.PeriodFrom)
	}
	if resp.PeriodTo != "" {
		t.Errorf("PeriodTo = %q, want empty string", resp.PeriodTo)
	}
	if resp.GrossIncomeCZK != 120000.0 {
		t.Errorf("GrossIncomeCZK = %f, want %f", resp.GrossIncomeCZK, 120000.0)
	}
	if resp.IncomeWithoutAdvanceCZK != 0.0 {
		t.Errorf("IncomeWithoutAdvanceCZK = %f, want 0", resp.IncomeWithoutAdvanceCZK)
	}
	if resp.ForeignTaxPaidCZK != 0.0 {
		t.Errorf("ForeignTaxPaidCZK = %f, want 0", resp.ForeignTaxPaidCZK)
	}
	if resp.AdvanceTaxWithheldCZK != 0.0 {
		t.Errorf("AdvanceTaxWithheldCZK = %f, want 0", resp.AdvanceTaxWithheldCZK)
	}
	if resp.AnnualSettlementRefundCZK != 0.0 {
		t.Errorf("AnnualSettlementRefundCZK = %f, want 0", resp.AnnualSettlementRefundCZK)
	}
	if resp.MonthlyBonusPaidCZK != 0.0 {
		t.Errorf("MonthlyBonusPaidCZK = %f, want 0", resp.MonthlyBonusPaidCZK)
	}
	if resp.WithheldFinalTaxCZK != 0.0 {
		t.Errorf("WithheldFinalTaxCZK = %f, want 0", resp.WithheldFinalTaxCZK)
	}
	if resp.Confidence != 0.0 {
		t.Errorf("Confidence = %f, want 0", resp.Confidence)
	}
	if resp.RawText != "" {
		t.Errorf("RawText = %q, want empty string", resp.RawText)
	}
}

func TestParseEmploymentResponse_StripsCodeFences(t *testing.T) {
	input := "```json\n{\"certificate_type\": \"advance\", \"employer_name\": \"Gamma s.r.o.\", \"gross_income_czk\": 60000.0, \"confidence\": 0.7}\n```"

	resp, err := ParseEmploymentResponse(input)
	if err != nil {
		t.Fatalf("ParseEmploymentResponse() error: %v", err)
	}

	if resp.CertificateType != "advance" {
		t.Errorf("CertificateType = %q, want %q", resp.CertificateType, "advance")
	}
	if resp.EmployerName != "Gamma s.r.o." {
		t.Errorf("EmployerName = %q, want %q", resp.EmployerName, "Gamma s.r.o.")
	}
	if resp.GrossIncomeCZK != 60000.0 {
		t.Errorf("GrossIncomeCZK = %f, want %f", resp.GrossIncomeCZK, 60000.0)
	}
}

func TestParseEmploymentResponse_StripsBareCodeFences(t *testing.T) {
	input := "```\n{\"certificate_type\": \"withholding\", \"gross_income_czk\": 30000.0}\n```"

	resp, err := ParseEmploymentResponse(input)
	if err != nil {
		t.Fatalf("ParseEmploymentResponse() error: %v", err)
	}

	if resp.CertificateType != "withholding" {
		t.Errorf("CertificateType = %q, want %q", resp.CertificateType, "withholding")
	}
	if resp.GrossIncomeCZK != 30000.0 {
		t.Errorf("GrossIncomeCZK = %f, want %f", resp.GrossIncomeCZK, 30000.0)
	}
}

func TestParseEmploymentResponse_MalformedJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"plain text", "not valid json at all"},
		{"unterminated object", `{"certificate_type": "advance"`},
		{"trailing garbage", `{"certificate_type": "advance"} extra`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseEmploymentResponse(tt.input)
			if err == nil {
				t.Errorf("expected error for malformed input %q", tt.input)
			}
		})
	}
}

func TestParseEmploymentResponse_LowConfidencePropagation(t *testing.T) {
	// The OCR-layer module must not silently filter / warn on low confidence;
	// it just reads the field through. Downstream services decide what to do.
	input := `{
		"certificate_type": "advance",
		"gross_income_czk": 100000.0,
		"confidence": 0.32,
		"raw_text": "blurry scan"
	}`

	resp, err := ParseEmploymentResponse(input)
	if err != nil {
		t.Fatalf("ParseEmploymentResponse() error: %v", err)
	}

	if resp.Confidence != 0.32 {
		t.Errorf("Confidence = %f, want %f (must be propagated verbatim)", resp.Confidence, 0.32)
	}
	if resp.GrossIncomeCZK != 100000.0 {
		t.Errorf("GrossIncomeCZK = %f, want %f", resp.GrossIncomeCZK, 100000.0)
	}
}

func TestParseEmploymentResponse_AmountsAsRawFloat(t *testing.T) {
	// Verify that the OCR layer does NOT convert CZK to halere — callers do that
	// via domain.AmountFromFloat or similar helpers.
	input := `{
		"certificate_type": "advance",
		"gross_income_czk": 1234.56,
		"advance_tax_withheld_czk": 78.90,
		"monthly_bonus_paid_czk": 0.01
	}`

	resp, err := ParseEmploymentResponse(input)
	if err != nil {
		t.Fatalf("ParseEmploymentResponse() error: %v", err)
	}

	if resp.GrossIncomeCZK != 1234.56 {
		t.Errorf("GrossIncomeCZK = %f, want %f (must stay as float CZK, no halere conversion)", resp.GrossIncomeCZK, 1234.56)
	}
	if resp.AdvanceTaxWithheldCZK != 78.90 {
		t.Errorf("AdvanceTaxWithheldCZK = %f, want %f", resp.AdvanceTaxWithheldCZK, 78.90)
	}
	if resp.MonthlyBonusPaidCZK != 0.01 {
		t.Errorf("MonthlyBonusPaidCZK = %f, want %f", resp.MonthlyBonusPaidCZK, 0.01)
	}
}

func TestEmploymentSystemPrompt_ContainsKeyInstructions(t *testing.T) {
	prompt := EmploymentSystemPrompt()
	if prompt == "" {
		t.Fatal("expected non-empty system prompt")
	}

	mustContain := []string{
		"JSON",
		"certificate_type",
		"advance",
		"withholding",
		"vzor c. 33",
		"vzor c. 12",
		"r.2",
		"r.4",
		"r.5",
		"r.8",
		"r.13",
		"gross_income_czk",
		"monthly_bonus_paid_czk",
		"r.5 + r.13", // critical correctness note: NOT just r.13
		"income_without_advance_czk",
		"§ 38h",
		"contract_type",
		"dpc",
		"dpp",
		"hpp",
		"YYYY-MM-DD",
		"raw_text",
		"confidence",
		"2000",
	}
	for _, s := range mustContain {
		if !strings.Contains(prompt, s) {
			t.Errorf("expected system prompt to contain %q, but it did not", s)
		}
	}
}

func TestEmploymentSystemPrompt_GrossIncomeFormulaDocumented(t *testing.T) {
	// Critical: the prompt must explicitly say gross_income_czk for advance variant
	// is r.2 + r.4 (not just one of them).
	prompt := EmploymentSystemPrompt()
	if !strings.Contains(prompt, "r.2 + r.4") {
		t.Error("expected system prompt to specify 'gross_income_czk = r.2 + r.4' for advance variant")
	}
}

func TestEmploymentUserPrompt_NonEmpty(t *testing.T) {
	prompt := EmploymentUserPrompt()
	if prompt == "" {
		t.Fatal("expected non-empty user prompt")
	}
	if !strings.Contains(prompt, "Potvrzeni") {
		t.Errorf("expected user prompt to mention 'Potvrzeni', got: %q", prompt)
	}
}
