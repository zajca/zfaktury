package ocr

import (
	"strings"
	"testing"
)

func TestDeductionSystemPrompt_MentionsAllCategories(t *testing.T) {
	p := DeductionSystemPrompt()
	for _, want := range []string{"mortgage", "life_insurance", "pension", "donation", "union_dues"} {
		if !strings.Contains(p, want) {
			t.Errorf("system prompt missing category %q", want)
		}
	}
}

func TestDeductionUserPrompt_NotEmpty(t *testing.T) {
	if DeductionUserPrompt() == "" {
		t.Fatal("user prompt is empty")
	}
}

func TestParseDeductionJSON_HappyPath(t *testing.T) {
	raw := `{
		"category": "mortgage",
		"provider_name": "Česká spořitelna",
		"provider_ico": "45244782",
		"contract_number": "12345/2025",
		"document_date": "2026-01-15",
		"period_year": 2025,
		"amount_czk": 45000.50,
		"purpose": "",
		"description_suggestion": "Úroky z hypotéky 2025",
		"confidence": 0.92,
		"raw_text": "POTVRZENÍ O ZAPLACENÝCH ÚROCÍCH..."
	}`

	got, err := ParseDeductionJSON(raw)
	if err != nil {
		t.Fatalf("ParseDeductionJSON: %v", err)
	}
	if got.Category != "mortgage" {
		t.Errorf("Category = %q, want mortgage", got.Category)
	}
	if got.ProviderName != "Česká spořitelna" {
		t.Errorf("ProviderName = %q", got.ProviderName)
	}
	if got.ProviderICO != "45244782" {
		t.Errorf("ProviderICO = %q", got.ProviderICO)
	}
	if got.ContractNumber != "12345/2025" {
		t.Errorf("ContractNumber = %q", got.ContractNumber)
	}
	if got.DocumentDate != "2026-01-15" {
		t.Errorf("DocumentDate = %q", got.DocumentDate)
	}
	if got.PeriodYear != 2025 {
		t.Errorf("PeriodYear = %d", got.PeriodYear)
	}
	if got.AmountCZK != 45000.50 {
		t.Errorf("AmountCZK = %v", got.AmountCZK)
	}
	if got.DescriptionSuggestion != "Úroky z hypotéky 2025" {
		t.Errorf("DescriptionSuggestion = %q", got.DescriptionSuggestion)
	}
	if got.Confidence != 0.92 {
		t.Errorf("Confidence = %v", got.Confidence)
	}
}

func TestParseDeductionJSON_StripsCodeFences(t *testing.T) {
	raw := "```json\n" + `{"category": "donation", "purpose": "Dar nemocnici", "amount_czk": 1000, "confidence": 0.6}` + "\n```"

	got, err := ParseDeductionJSON(raw)
	if err != nil {
		t.Fatalf("ParseDeductionJSON with fences: %v", err)
	}
	if got.Category != "donation" {
		t.Errorf("Category = %q", got.Category)
	}
	if got.Purpose != "Dar nemocnici" {
		t.Errorf("Purpose = %q", got.Purpose)
	}
}

func TestParseDeductionJSON_EmptyAndUnknownCategory(t *testing.T) {
	raw := `{"category": "unknown", "amount_czk": 0, "confidence": 0.1}`
	got, err := ParseDeductionJSON(raw)
	if err != nil {
		t.Fatalf("ParseDeductionJSON: %v", err)
	}
	if got.Category != "unknown" {
		t.Errorf("Category = %q, want unknown (caller normalises)", got.Category)
	}
	if got.AmountCZK != 0 {
		t.Errorf("AmountCZK = %v", got.AmountCZK)
	}
}

func TestParseDeductionJSON_InvalidJSON(t *testing.T) {
	if _, err := ParseDeductionJSON("not a json"); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseDeductionJSON_AllSupportedCategories(t *testing.T) {
	cats := []string{"mortgage", "life_insurance", "pension", "donation", "union_dues"}
	for _, c := range cats {
		raw := `{"category": "` + c + `", "amount_czk": 1000, "confidence": 0.8}`
		got, err := ParseDeductionJSON(raw)
		if err != nil {
			t.Fatalf("ParseDeductionJSON(%s): %v", c, err)
		}
		if got.Category != c {
			t.Errorf("Category = %q, want %q", got.Category, c)
		}
	}
}
