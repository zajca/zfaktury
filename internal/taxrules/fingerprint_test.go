package taxrules

import (
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

// goldenFingerprints pins the content of every released rule-set ID. A value
// change must come with a new rule-set ID (for example ".v2") and a new entry
// here; editing an existing ID's values fails this test on purpose.
var goldenFingerprints = map[string]string{
	"cz-dpfo-2024.v1":             "533265f0885b881fefaef3a01123f72dfeb77ca05a09b3ba7e5cced3334600d4",
	"cz-dpfo-2025.v1":             "b57b7504b0f16402b976671ef5437afb7f7f9869913f302e1c3fc05116a969c4",
	"cz-dpfo-2026.v0-provisional": "e9a75cfe821ed129a96610c65be40a71ab9355c5b4774887ed9cd48002f74611",
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
