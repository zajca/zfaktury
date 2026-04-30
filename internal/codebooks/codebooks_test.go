package codebooks

import (
	"strings"
	"testing"
)

func TestFinancialOffices_Sequence(t *testing.T) {
	offices := FinancialOffices()
	if len(offices) != 15 {
		t.Fatalf("expected 15 offices (14 kraj + 1 SFÚ), got %d", len(offices))
	}
	wantSeq := []string{
		"451", "452", "453", "454", "455", "456", "457", "458",
		"459", "460", "461", "462", "463", "464", "13",
	}
	for i, code := range wantSeq {
		if offices[i].Code != code {
			t.Errorf("offices[%d].Code = %q, want %q", i, offices[i].Code, code)
		}
	}
}

func TestFinancialOfficeByCode(t *testing.T) {
	tests := []struct {
		code     string
		wantName string
		wantNil  bool
	}{
		{"464", "Finanční úřad pro Zlínský kraj", false},
		{"451", "Finanční úřad pro hlavní město Prahu", false},
		{"13", "Specializovaný finanční úřad", false},
		{"  464  ", "Finanční úřad pro Zlínský kraj", false},
		{"581", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		got := FinancialOfficeByCode(tt.code)
		if tt.wantNil {
			if got != nil {
				t.Errorf("FinancialOfficeByCode(%q) = %+v, want nil", tt.code, got)
			}
			continue
		}
		if got == nil {
			t.Errorf("FinancialOfficeByCode(%q) = nil, want %q", tt.code, tt.wantName)
			continue
		}
		if got.Name != tt.wantName {
			t.Errorf("FinancialOfficeByCode(%q).Name = %q, want %q", tt.code, got.Name, tt.wantName)
		}
	}
}

func TestNACE_LoadsAllSubclasses(t *testing.T) {
	entries := NACE()
	// The CSU codebook level 5 (kodcis 6105) for CZ-NACE 2025 has 715 leaf
	// subclasses as of the 2025-01-01 release.
	if len(entries) < 700 || len(entries) > 800 {
		t.Errorf("NACE() returned %d entries, expected ~715", len(entries))
	}
}

func TestNACEByCode(t *testing.T) {
	tests := []struct {
		code     string
		wantNil  bool
		wantHint string // substring that must appear in returned name
	}{
		{"62109", false, "počítačové programování"},
		{"62101", false, "počítačových her"},
		{"621090", false, "počítačové programování"}, // 6-digit EPO form
		{"99999", true, ""},
		{"6210", true, ""}, // 4-digit class, not a leaf
		{"", true, ""},
	}
	for _, tt := range tests {
		got := NACEByCode(tt.code)
		if tt.wantNil {
			if got != nil {
				t.Errorf("NACEByCode(%q) = %+v, want nil", tt.code, got)
			}
			continue
		}
		if got == nil {
			t.Errorf("NACEByCode(%q) = nil, want match for %q", tt.code, tt.wantHint)
			continue
		}
		if !contains(got.Name, tt.wantHint) {
			t.Errorf("NACEByCode(%q).Name = %q, want substring %q", tt.code, got.Name, tt.wantHint)
		}
	}
}

func contains(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}
