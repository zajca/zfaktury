package annualtaxxml

import (
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

func TestToWholeCZK(t *testing.T) {
	tests := []struct {
		input domain.Amount
		want  int64
	}{
		{0, 0},
		{100, 1},
		{150, 1},
		{199, 1},
		{200, 2},
		{-150, -1},
		{10050, 100},
	}
	for _, tt := range tests {
		got := ToWholeCZK(tt.input)
		if got != tt.want {
			t.Errorf("ToWholeCZK(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestDPFOFilingTypeCode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{domain.FilingTypeRegular, "B"},
		{domain.FilingTypeCorrective, "O"},
		{domain.FilingTypeSupplementary, "D"},
		{"unknown", "B"},
	}
	for _, tt := range tests {
		got := DPFOFilingTypeCode(tt.input)
		if got != tt.want {
			t.Errorf("DPFOFilingTypeCode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCSSZFilingTypeCode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{domain.FilingTypeRegular, "N"},
		{domain.FilingTypeCorrective, "O"},
		{domain.FilingTypeSupplementary, "Z"},
		{"unknown", "N"},
	}
	for _, tt := range tests {
		got := CSSZFilingTypeCode(tt.input)
		if got != tt.want {
			t.Errorf("CSSZFilingTypeCode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
