package taxrules

import (
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

// goldenFingerprints pins the content of every released rule-set ID. A value
// change must come with a new rule-set ID (for example ".v2") and a new entry
// here; editing an existing ID's values fails this test on purpose.
var goldenFingerprints = map[string]string{
	"cz-dpfo-2024.v6":             "e31c9c015055fe79b512fc166b192aff41248a7e35038d7e5daefacc9db2a72a",
	"cz-dpfo-2025.v6":             "4dd20e9f1c66e03e02bf0323ad07e7867713df5a234204e4537935cde3d404cd",
	"cz-dpfo-2026.v5-provisional": "327650bbc72a80d8c3f4523dbe25f0ab92fa8ce0474eb5d4bfa36ea50998a373",
}

func TestFingerprint_GoldenPerRuleSetID(t *testing.T) {
	seen := map[string]bool{}
	for _, year := range SupportedYears() {
		rules, err := GetRuleSet(year)
		if err != nil {
			t.Fatalf("GetRuleSet(%d) error: %v", year, err)
		}
		want, ok := goldenFingerprints[rules.ID]
		if !ok {
			t.Errorf("rule set %s has no golden fingerprint", rules.ID)
			continue
		}
		seen[rules.ID] = true
		got, err := rules.Fingerprint()
		if err != nil {
			t.Fatalf("Fingerprint(%s) error: %v", rules.ID, err)
		}
		if got != want {
			t.Errorf("fingerprint of %s changed:\n got  %s\n want %s\nchange the rule-set ID when values change", rules.ID, got, want)
		}
	}
	for id := range goldenFingerprints {
		if !seen[id] {
			t.Errorf("golden fingerprint %s has no rule set", id)
		}
	}
}

func TestFingerprint_IgnoresSourcesButNotValues(t *testing.T) {
	rules, err := GetRuleSet(2025)
	if err != nil {
		t.Fatalf("GetRuleSet(2025) error: %v", err)
	}
	base, err := rules.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint() error: %v", err)
	}

	edited := rules
	edited.Sources = []LegalSource{{Title: "Corrected title", URL: "https://example.invalid/fixed"}}
	got, err := edited.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint() error: %v", err)
	}
	if got != base {
		t.Error("fingerprint changed after editing Sources only")
	}

	edited = rules
	edited.Constants.BasicCredit += domain.NewAmount(1, 0)
	got, err = edited.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint() error: %v", err)
	}
	if got == base {
		t.Error("fingerprint did not change after editing a constant")
	}

	edited = rules
	edited.Constants.SecurityExemptionLimitAssetTypes = []string{domain.AssetTypeCrypto}
	got, err = edited.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint() error: %v", err)
	}
	if got == base {
		t.Error("fingerprint did not change after editing the limited asset types")
	}
}
