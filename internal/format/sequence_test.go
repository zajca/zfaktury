package format

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

type renderCase struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Prefix  string `json:"prefix"`
	Year    int    `json:"year"`
	Number  int    `json:"number"`
	Want    string `json:"want"`
}

type validationCase struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
}

type fixtureFile struct {
	RenderCases      []renderCase     `json:"render_cases"`
	ValidationErrors []validationCase `json:"validation_errors"`
}

func loadFixture(t *testing.T) fixtureFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "render_cases.json"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	var f fixtureFile
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}
	if len(f.RenderCases) == 0 || len(f.ValidationErrors) == 0 {
		t.Fatal("fixture is empty")
	}
	return f
}

func TestRender_Fixtures(t *testing.T) {
	f := loadFixture(t)
	for _, tc := range f.RenderCases {
		t.Run(tc.Name, func(t *testing.T) {
			got := Render(tc.Pattern, tc.Prefix, tc.Year, tc.Number)
			if got != tc.Want {
				t.Errorf("Render(%q, %q, %d, %d) = %q, want %q",
					tc.Pattern, tc.Prefix, tc.Year, tc.Number, got, tc.Want)
			}
		})
	}
}

func TestValidatePattern_ValidFixturesPass(t *testing.T) {
	f := loadFixture(t)
	for _, tc := range f.RenderCases {
		t.Run(tc.Name, func(t *testing.T) {
			if err := ValidatePattern(tc.Pattern); err != nil {
				t.Errorf("ValidatePattern(%q) returned error: %v", tc.Pattern, err)
			}
		})
	}
}

func TestValidatePattern_InvalidFixturesFail(t *testing.T) {
	f := loadFixture(t)
	for _, tc := range f.ValidationErrors {
		t.Run(tc.Name, func(t *testing.T) {
			err := ValidatePattern(tc.Pattern)
			if err == nil {
				t.Fatalf("ValidatePattern(%q) returned nil, want error", tc.Pattern)
			}
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("error does not wrap ErrInvalidInput: %v", err)
			}
		})
	}
}

func TestRender_LegacyParity(t *testing.T) {
	const pattern = "{prefix}{year}{number:04d}"
	cases := []struct {
		prefix string
		year   int
		number int
		want   string
	}{
		{"FV", 2026, 1, "FV20260001"},
		{"ZF", 2025, 42, "ZF20250042"},
		{"DN", 2026, 9999, "DN20269999"},
	}
	for _, c := range cases {
		got := Render(pattern, c.prefix, c.year, c.number)
		if got != c.want {
			t.Errorf("legacy parity broken: Render(%q,%d,%d) = %q, want %q",
				c.prefix, c.year, c.number, got, c.want)
		}
	}
}
