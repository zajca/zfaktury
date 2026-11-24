package taxrules

import (
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

// goldenFingerprints pins the content of every released rule-set ID. A value
// change must come with a new rule-set ID (for example ".v2") and a new entry
// here; editing an existing ID's values fails this test on purpose.
var goldenFingerprints = map[string]string{
	"cz-dpfo-2024.v4":             "80153abf5babc88f48bb6951c3b2e958fafea5294ffa0c8b29f38a06034d26cb",
	"cz-dpfo-2025.v4":             "049268b29625576e48f235b8c2b53d118b71fec225bdd3ea8f4e3b21a84b5a65",
	"cz-dpfo-2026.v3-provisional": "49916b7eefccab4602631f3fb8ef276efdbf92c297edbe1ee9f19f6a8fea272d",
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
