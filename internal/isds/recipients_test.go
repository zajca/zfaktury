package isds

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

// wantTaxOfficeBoxes is the expected production table, written out
// independently of taxOfficeBoxes so that a typo in either shows up.
var wantTaxOfficeBoxes = map[int]string{
	451: "7nyn2d9",
	452: "6sxny3p",
	453: "scdnz6b",
	454: "gf9n2e3",
	455: "q9in2ev",
	456: "qjfn2bj",
	457: "zcqn2bf",
	458: "fj8ny4i",
	459: "95zn2bb",
	460: "tqjn2cy",
	461: "qdhny4c",
	462: "25nnz67",
	463: "4hun2cu",
	464: "n69nz5y",
	13:  "7cs8cge",
}

// testBoxFor returns a synthetic test box id for a c_ufo code.
func testBoxFor(cUfo int) string {
	return "t" + strconv.Itoa(1000000 + cUfo)[1:]
}

func fullTestBoxes() TestRecipientBoxes {
	boxes := TestRecipientBoxes{CSSZ: "9tsaf6s", TaxOffices: map[string]string{}}
	for cUfo := range wantTaxOfficeBoxes {
		boxes.TaxOffices[strconv.Itoa(cUfo)] = testBoxFor(cUfo)
	}
	return boxes
}

func TestTaxOfficeTable_MatchesSource(t *testing.T) {
	if len(taxOfficeBoxes) != len(wantTaxOfficeBoxes) {
		t.Fatalf("table has %d rows, want %d", len(taxOfficeBoxes), len(wantTaxOfficeBoxes))
	}
	seen := map[string]bool{}
	for _, row := range taxOfficeBoxes {
		want, ok := wantTaxOfficeBoxes[row.CUfo]
		if !ok {
			t.Errorf("unexpected c_ufo %d in table", row.CUfo)
			continue
		}
		if row.BoxID != want {
			t.Errorf("c_ufo %d: box = %q, want %q", row.CUfo, row.BoxID, want)
		}
		if err := ValidateBoxID(row.BoxID); err != nil {
			t.Errorf("c_ufo %d: table box id malformed: %v", row.CUfo, err)
		}
		if seen[row.BoxID] {
			t.Errorf("box %q appears twice", row.BoxID)
		}
		seen[row.BoxID] = true
		if !strings.Contains(row.Name, "inanční úřad") {
			t.Errorf("c_ufo %d: unexpected name %q", row.CUfo, row.Name)
		}
	}
}

func TestRecipients_ProductionTaxOffice_EveryRow(t *testing.T) {
	r, err := NewRecipients(RecipientEnvProduction, TestRecipientBoxes{})
	if err != nil {
		t.Fatalf("NewRecipients: %v", err)
	}
	for cUfo, wantBox := range wantTaxOfficeBoxes {
		got, err := r.Lookup(RecipientKindTaxOffice, cUfo)
		if err != nil {
			t.Errorf("c_ufo %d: %v", cUfo, err)
			continue
		}
		if got.BoxID != wantBox {
			t.Errorf("c_ufo %d: box = %q, want %q", cUfo, got.BoxID, wantBox)
		}
		if got.Name == "" {
			t.Errorf("c_ufo %d: empty display name", cUfo)
		}
	}
}

func TestRecipients_ProductionCSSZ(t *testing.T) {
	r, err := NewRecipients(RecipientEnvProduction, TestRecipientBoxes{})
	if err != nil {
		t.Fatalf("NewRecipients: %v", err)
	}
	got, err := r.Lookup(RecipientKindCSSZ, 0)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got.BoxID != "5ffu6xk" {
		t.Errorf("box = %q, want 5ffu6xk", got.BoxID)
	}
	if !strings.Contains(got.Name, "ČSSZ") && !strings.Contains(got.Name, "Česká správa") {
		t.Errorf("name = %q, want a ČSSZ name", got.Name)
	}
}

func TestRecipients_UnknownCode(t *testing.T) {
	for _, env := range []RecipientEnvironment{RecipientEnvProduction, RecipientEnvTest} {
		r, err := NewRecipients(env, fullTestBoxes())
		if err != nil {
			t.Fatalf("NewRecipients(%s): %v", env, err)
		}
		for _, cUfo := range []int{0, 1, 12, 14, 450, 465, -451} {
			_, err := r.Lookup(RecipientKindTaxOffice, cUfo)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("env %s c_ufo %d: err = %v, want ErrInvalidInput", env, cUfo, err)
			}
		}
	}
}

func TestRecipients_UnknownKind(t *testing.T) {
	r, err := NewRecipients(RecipientEnvProduction, TestRecipientBoxes{})
	if err != nil {
		t.Fatalf("NewRecipients: %v", err)
	}
	if _, err := r.Lookup(RecipientKind("health_insurer"), 451); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput", err)
	}
	if _, err := r.Lookup(RecipientKind(""), 451); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("empty kind: err = %v, want ErrInvalidInput", err)
	}
}

func TestRecipients_TestEnvironment_EveryRow(t *testing.T) {
	r, err := NewRecipients(RecipientEnvTest, fullTestBoxes())
	if err != nil {
		t.Fatalf("NewRecipients: %v", err)
	}
	for cUfo := range wantTaxOfficeBoxes {
		got, err := r.Lookup(RecipientKindTaxOffice, cUfo)
		if err != nil {
			t.Errorf("c_ufo %d: %v", cUfo, err)
			continue
		}
		if got.BoxID != testBoxFor(cUfo) {
			t.Errorf("c_ufo %d: box = %q, want %q", cUfo, got.BoxID, testBoxFor(cUfo))
		}
		if !strings.Contains(got.Name, "testovací") {
			t.Errorf("c_ufo %d: name %q does not mark the test environment", cUfo, got.Name)
		}
	}
	got, err := r.Lookup(RecipientKindCSSZ, 0)
	if err != nil {
		t.Fatalf("CSSZ: %v", err)
	}
	if got.BoxID != "9tsaf6s" {
		t.Errorf("CSSZ box = %q, want 9tsaf6s", got.BoxID)
	}
	if !strings.Contains(got.Name, "testovací") {
		t.Errorf("CSSZ name %q does not mark the test environment", got.Name)
	}
}

func TestRecipients_TestEnvironment_MissingConfig(t *testing.T) {
	r, err := NewRecipients(RecipientEnvTest, TestRecipientBoxes{})
	if err != nil {
		t.Fatalf("NewRecipients: %v", err)
	}
	if _, err := r.Lookup(RecipientKindCSSZ, 0); !errors.Is(err, domain.ErrMissingSetting) {
		t.Errorf("CSSZ: err = %v, want ErrMissingSetting", err)
	}
	for cUfo := range wantTaxOfficeBoxes {
		if _, err := r.Lookup(RecipientKindTaxOffice, cUfo); !errors.Is(err, domain.ErrMissingSetting) {
			t.Errorf("c_ufo %d: err = %v, want ErrMissingSetting", cUfo, err)
		}
	}

	partial := TestRecipientBoxes{TaxOffices: map[string]string{"451": "abcdefg"}}
	r, err = NewRecipients(RecipientEnvTest, partial)
	if err != nil {
		t.Fatalf("NewRecipients partial: %v", err)
	}
	if _, err := r.Lookup(RecipientKindTaxOffice, 451); err != nil {
		t.Errorf("configured c_ufo 451: %v", err)
	}
	if _, err := r.Lookup(RecipientKindTaxOffice, 452); !errors.Is(err, domain.ErrMissingSetting) {
		t.Errorf("unconfigured c_ufo 452: err = %v, want ErrMissingSetting", err)
	}
}

func TestNewRecipients_MalformedBoxID(t *testing.T) {
	malformed := []string{"", "abc", "abcdefgh", "ABCDEFG", "abc-efg", "abcdef!", "abcdéfg", " bcdefg", "abcdef\n"}
	for _, id := range malformed {
		_, err := NewRecipients(RecipientEnvTest, TestRecipientBoxes{CSSZ: id})
		if id == "" {
			// An empty CSSZ box means "not configured" and fails at lookup.
			if err != nil {
				t.Errorf("empty CSSZ box: NewRecipients err = %v, want nil", err)
			}
			continue
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("CSSZ box %q: err = %v, want ErrInvalidInput", id, err)
		}
		_, err = NewRecipients(RecipientEnvTest, TestRecipientBoxes{TaxOffices: map[string]string{"451": id}})
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("tax office box %q: err = %v, want ErrInvalidInput", id, err)
		}
	}
	if err := ValidateBoxID("9tsaf6s"); err != nil {
		t.Errorf("ValidateBoxID(9tsaf6s) = %v", err)
	}
}

func TestNewRecipients_BadTestTaxOfficeKeys(t *testing.T) {
	for _, key := range []string{"abc", "", "450", "465", "-1", "451 "} {
		_, err := NewRecipients(RecipientEnvTest, TestRecipientBoxes{TaxOffices: map[string]string{key: "abcdefg"}})
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("key %q: err = %v, want ErrInvalidInput", key, err)
		}
	}
}

func TestNewRecipients_RejectsUnknownEnvironment(t *testing.T) {
	for _, env := range []RecipientEnvironment{"", "prod", "TEST", "staging"} {
		if _, err := NewRecipients(env, TestRecipientBoxes{}); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("env %q: err = %v, want ErrInvalidInput", env, err)
		}
	}
	for _, s := range []string{"", "prod", "Test"} {
		if _, err := ParseRecipientEnvironment(s); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("Parse(%q): err = %v, want ErrInvalidInput", s, err)
		}
	}
	for s, want := range map[string]RecipientEnvironment{"test": RecipientEnvTest, "production": RecipientEnvProduction} {
		got, err := ParseRecipientEnvironment(s)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %q, %v", s, got, err)
		}
	}
}

// A test lookup must never yield a production box, whether the configuration
// is complete, partial, empty or tries to name a production box.
func TestRecipients_TestLookupNeverReturnsProductionBox(t *testing.T) {
	configs := map[string]TestRecipientBoxes{
		"empty":    {},
		"complete": fullTestBoxes(),
		"partial":  {CSSZ: "9tsaf6s", TaxOffices: map[string]string{"451": "abcdefg"}},
	}
	for name, cfg := range configs {
		r, err := NewRecipients(RecipientEnvTest, cfg)
		if err != nil {
			t.Fatalf("%s: NewRecipients: %v", name, err)
		}
		lookups := []struct {
			kind RecipientKind
			cUfo int
		}{{RecipientKindCSSZ, 0}}
		for cUfo := range wantTaxOfficeBoxes {
			lookups = append(lookups, struct {
				kind RecipientKind
				cUfo int
			}{RecipientKindTaxOffice, cUfo})
		}
		for _, l := range lookups {
			got, err := r.Lookup(l.kind, l.cUfo)
			if err != nil {
				continue
			}
			if isProductionBox(got.BoxID) {
				t.Errorf("%s: test lookup %s/%d returned production box %q", name, l.kind, l.cUfo, got.BoxID)
			}
		}
	}

	prodBoxes := []string{CSSZEpodaniBoxID}
	for _, box := range wantTaxOfficeBoxes {
		prodBoxes = append(prodBoxes, box)
	}
	for _, box := range prodBoxes {
		if _, err := NewRecipients(RecipientEnvTest, TestRecipientBoxes{CSSZ: box}); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("CSSZ test box %q (production): err = %v, want ErrInvalidInput", box, err)
		}
		if _, err := NewRecipients(RecipientEnvTest, TestRecipientBoxes{TaxOffices: map[string]string{"452": box}}); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("tax office test box %q (production): err = %v, want ErrInvalidInput", box, err)
		}
	}
}

func TestRecipients_Environment(t *testing.T) {
	r, err := NewRecipients(RecipientEnvTest, TestRecipientBoxes{})
	if err != nil {
		t.Fatalf("NewRecipients: %v", err)
	}
	if r.Environment() != RecipientEnvTest {
		t.Errorf("Environment() = %q", r.Environment())
	}
}
