package isds

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/zajca/zfaktury/internal/domain"
)

// RecipientEnvironment selects which set of recipient boxes is resolved. It
// has no default: the caller must name one explicitly.
type RecipientEnvironment string

// The supported recipient environments.
const (
	RecipientEnvTest       RecipientEnvironment = "test"
	RecipientEnvProduction RecipientEnvironment = "production"
)

// ParseRecipientEnvironment converts a configuration value to a
// RecipientEnvironment. Anything but "test" and "production" is rejected.
func ParseRecipientEnvironment(s string) (RecipientEnvironment, error) {
	switch e := RecipientEnvironment(s); e {
	case RecipientEnvTest, RecipientEnvProduction:
		return e, nil
	default:
		return "", fmt.Errorf("%w: recipient environment must be %q or %q", domain.ErrInvalidInput, RecipientEnvProduction, RecipientEnvTest)
	}
}

// RecipientKind names the authority a filing goes to.
type RecipientKind string

// The supported recipient kinds.
const (
	// RecipientKindTaxOffice is the regional tax office (FÚ) selected by c_ufo.
	RecipientKindTaxOffice RecipientKind = "tax_office"
	// RecipientKindCSSZ is the ČSSZ e-Podání box.
	RecipientKindCSSZ RecipientKind = "cssz"
)

// CSSZEpodaniBoxID is the data box "e-podani ČSSZ" that receives bare-XML
// OSVČ overviews (ČSSZ "Podávací a dotazovací protokol pro e-podání" v1.47).
// The ISDS test environment documents the ČSSZ test box 9tsaf6s; it is never
// built in and must be set through TestRecipientBoxes.CSSZ.
const CSSZEpodaniBoxID = "5ffu6xk"

// Recipient is a resolved data box with its Czech display name.
type Recipient struct {
	BoxID string
	Name  string
}

// taxOfficeBox is one row of the regional tax office box table.
type taxOfficeBox struct {
	CUfo  int
	Name  string
	BoxID string
}

// taxOfficeBoxes lists the regional tax office (FÚ) data boxes by c_ufo.
// Source: the Finanční správa list "Identifikacni_kody_FR_a_FU.pdf" of
// 1.7.2023, cross-checked on 2026-11-24 against the ISDS open data
// "seznam_ds_ovm" (https://www.datovka.gov.cz/sds/datafile.do?format=xml&service=seznam_ds_ovm).
// Territorial workplaces (c_pracufo) have their own boxes but no official
// mapping table exists, so filings go to the regional office.
var taxOfficeBoxes = []taxOfficeBox{
	{451, "Finanční úřad pro hlavní město Prahu", "7nyn2d9"},
	{452, "Finanční úřad pro Středočeský kraj", "6sxny3p"},
	{453, "Finanční úřad pro Jihočeský kraj", "scdnz6b"},
	{454, "Finanční úřad pro Plzeňský kraj", "gf9n2e3"},
	{455, "Finanční úřad pro Karlovarský kraj", "q9in2ev"},
	{456, "Finanční úřad pro Ústecký kraj", "qjfn2bj"},
	{457, "Finanční úřad pro Liberecký kraj", "zcqn2bf"},
	{458, "Finanční úřad pro Královéhradecký kraj", "fj8ny4i"},
	{459, "Finanční úřad pro Pardubický kraj", "95zn2bb"},
	{460, "Finanční úřad pro Kraj Vysočina", "tqjn2cy"},
	{461, "Finanční úřad pro Jihomoravský kraj", "qdhny4c"},
	{462, "Finanční úřad pro Olomoucký kraj", "25nnz67"},
	{463, "Finanční úřad pro Moravskoslezský kraj", "4hun2cu"},
	{464, "Finanční úřad pro Zlínský kraj", "n69nz5y"},
	{13, "Specializovaný finanční úřad", "7cs8cge"},
}

const (
	csszName       = "Česká správa sociálního zabezpečení (e-Podání)"
	testNameSuffix = " (testovací schránka)"
)

// boxIDPattern is the format of an ISDS data box id: 7 characters [a-z0-9].
var boxIDPattern = regexp.MustCompile(`^[a-z0-9]{7}$`)

// ValidateBoxID reports whether id is a well-formed data box id.
func ValidateBoxID(id string) error {
	if !boxIDPattern.MatchString(id) {
		return fmt.Errorf("%w: data box id must be 7 characters [a-z0-9]", domain.ErrInvalidInput)
	}
	return nil
}

// TestRecipientBoxes holds the explicitly configured boxes of the ISDS test
// environment. It has no built-in values, so a test send can never reach a
// production box.
type TestRecipientBoxes struct {
	// CSSZ is the test box that stands in for CSSZEpodaniBoxID.
	CSSZ string
	// TaxOffices maps a c_ufo code to the test box standing in for that FÚ.
	TaxOffices map[string]string
}

// Recipients resolves filing recipients for one environment.
type Recipients struct {
	env  RecipientEnvironment
	test TestRecipientBoxes
	// testTaxOffices is TestRecipientBoxes.TaxOffices keyed by parsed c_ufo.
	testTaxOffices map[int]string
}

// NewRecipients validates the environment and, for the test environment, the
// configured boxes: every id must be well-formed, every c_ufo known, and no
// id may equal a production box. The production environment ignores test.
func NewRecipients(env RecipientEnvironment, test TestRecipientBoxes) (*Recipients, error) {
	switch env {
	case RecipientEnvProduction:
		return &Recipients{env: env}, nil
	case RecipientEnvTest:
	default:
		return nil, fmt.Errorf("%w: unknown recipient environment %q", domain.ErrInvalidInput, string(env))
	}

	r := &Recipients{env: env, test: test, testTaxOffices: make(map[int]string, len(test.TaxOffices))}
	if test.CSSZ != "" {
		if err := checkTestBox("cssz", test.CSSZ); err != nil {
			return nil, err
		}
	}
	for key, boxID := range test.TaxOffices {
		cUfo, err := strconv.Atoi(key)
		if err != nil {
			return nil, fmt.Errorf("%w: test tax office key %q is not a c_ufo code", domain.ErrInvalidInput, key)
		}
		if _, ok := findTaxOffice(cUfo); !ok {
			return nil, fmt.Errorf("%w: unknown c_ufo %d in test tax office boxes", domain.ErrInvalidInput, cUfo)
		}
		if err := checkTestBox("tax office "+key, boxID); err != nil {
			return nil, err
		}
		r.testTaxOffices[cUfo] = boxID
	}
	return r, nil
}

// checkTestBox rejects a malformed test box id and one that is a production
// recipient box, which would turn a test send into a production send.
func checkTestBox(label, boxID string) error {
	if err := ValidateBoxID(boxID); err != nil {
		return fmt.Errorf("test box for %s: %w", label, err)
	}
	if isProductionBox(boxID) {
		return fmt.Errorf("%w: test box for %s is a production recipient box", domain.ErrInvalidInput, label)
	}
	return nil
}

func isProductionBox(boxID string) bool {
	if boxID == CSSZEpodaniBoxID {
		return true
	}
	for _, row := range taxOfficeBoxes {
		if row.BoxID == boxID {
			return true
		}
	}
	return false
}

func findTaxOffice(cUfo int) (taxOfficeBox, bool) {
	for _, row := range taxOfficeBoxes {
		if row.CUfo == cUfo {
			return row, true
		}
	}
	return taxOfficeBox{}, false
}

// Environment returns the environment the lookup resolves for.
func (r *Recipients) Environment() RecipientEnvironment { return r.env }

// Lookup resolves the recipient of kind. cUfo is used for
// RecipientKindTaxOffice only. An unknown kind or c_ufo gives
// domain.ErrInvalidInput; a test environment without a configured box gives
// domain.ErrMissingSetting. The test environment never falls back to a
// production box.
func (r *Recipients) Lookup(kind RecipientKind, cUfo int) (Recipient, error) {
	switch kind {
	case RecipientKindTaxOffice:
		return r.lookupTaxOffice(cUfo)
	case RecipientKindCSSZ:
		return r.lookupCSSZ()
	default:
		return Recipient{}, fmt.Errorf("%w: unknown recipient kind %q", domain.ErrInvalidInput, string(kind))
	}
}

func (r *Recipients) lookupTaxOffice(cUfo int) (Recipient, error) {
	row, ok := findTaxOffice(cUfo)
	if !ok {
		return Recipient{}, fmt.Errorf("%w: unknown tax office code c_ufo %d", domain.ErrInvalidInput, cUfo)
	}
	switch r.env {
	case RecipientEnvProduction:
		return Recipient{BoxID: row.BoxID, Name: row.Name}, nil
	case RecipientEnvTest:
		boxID, ok := r.testTaxOffices[cUfo]
		if !ok {
			return Recipient{}, fmt.Errorf("%w: no ISDS test box configured for tax office c_ufo %d", domain.ErrMissingSetting, cUfo)
		}
		return Recipient{BoxID: boxID, Name: row.Name + testNameSuffix}, nil
	default:
		return Recipient{}, fmt.Errorf("%w: unknown recipient environment %q", domain.ErrInvalidInput, string(r.env))
	}
}

func (r *Recipients) lookupCSSZ() (Recipient, error) {
	switch r.env {
	case RecipientEnvProduction:
		return Recipient{BoxID: CSSZEpodaniBoxID, Name: csszName}, nil
	case RecipientEnvTest:
		if r.test.CSSZ == "" {
			return Recipient{}, fmt.Errorf("%w: no ISDS test box configured for ČSSZ", domain.ErrMissingSetting)
		}
		return Recipient{BoxID: r.test.CSSZ, Name: csszName + testNameSuffix}, nil
	default:
		return Recipient{}, fmt.Errorf("%w: unknown recipient environment %q", domain.ErrInvalidInput, string(r.env))
	}
}
