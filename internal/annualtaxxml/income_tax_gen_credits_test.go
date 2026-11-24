package annualtaxxml

import (
	"bytes"
	"encoding/xml"
	"errors"
	"strconv"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

// attrBag collects every attribute of an element.
type attrBag struct {
	Attrs []xml.Attr `xml:",any,attr"`
}

type parsedIncomeTax struct {
	VetaD attrBag `xml:"DPFDP7>VetaD"`
	VetaS attrBag `xml:"DPFDP7>VetaS"`
	VetaT attrBag `xml:"DPFDP7>VetaT"`
}

func parseIncomeTax(t *testing.T, data []byte) (d, s, vt map[string]string) {
	t.Helper()
	var p parsedIncomeTax
	if err := xml.Unmarshal(data, &p); err != nil {
		t.Fatalf("parsing generated XML: %v", err)
	}
	toMap := func(b attrBag) map[string]string {
		m := make(map[string]string, len(b.Attrs))
		for _, a := range b.Attrs {
			m[a.Name.Local] = a.Value
		}
		return m
	}
	return toMap(p.VetaD), toMap(p.VetaS), toMap(p.VetaT)
}

// attrInt returns the integer value of an attribute; a missing attribute is 0.
func attrInt(t *testing.T, m map[string]string, key string) int64 {
	t.Helper()
	v, ok := m[key]
	if !ok {
		return 0
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		t.Fatalf("attribute %s=%q is not an integer: %v", key, v, err)
	}
	return n
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func newCreditTestSettings() map[string]string {
	s := section6BaseSettings()
	return s
}

type formulaCase struct {
	name     string
	itr      *domain.IncomeTaxReturn
	children []domain.TaxChildCredit
	spouse   *domain.TaxSpouseCredit
}

func formulaCases() []formulaCase {
	full := &domain.IncomeTaxReturn{
		Year: 2025, FilingType: domain.FilingTypeRegular,
		TotalRevenue: domain.NewAmount(2_000_000, 0), FlatRatePercent: 60,
		UsedExpenses: domain.NewAmount(1_200_000, 0),
		TotalTax:     domain.NewAmount(120_000, 0),
		CreditBasic:  domain.NewAmount(30_840, 0), CreditSpouse: domain.NewAmount(24_840, 0),
		CreditDisability: domain.NewAmount(2_520, 0), CreditDisability12: domain.NewAmount(2_520, 0),
		TotalCredits: domain.NewAmount(58_200, 0),
		ChildBenefit: domain.NewAmount(15_204, 0),
		Prepayments:  domain.NewAmount(36_000, 0),
		TaxDue:       domain.NewAmount(10_596, 0),
	}
	minimal := &domain.IncomeTaxReturn{
		Year: 2025, FilingType: domain.FilingTypeRegular,
		TotalRevenue: domain.NewAmount(350_000, 0), UsedExpenses: domain.NewAmount(210_000, 0),
		TotalTax:    domain.NewAmount(21_000, 0),
		CreditBasic: domain.NewAmount(30_840, 0), TotalCredits: domain.NewAmount(30_840, 0),
	}
	allCredits := &domain.IncomeTaxReturn{
		Year: 2025, FilingType: domain.FilingTypeRegular,
		TotalRevenue: domain.NewAmount(1_000_000, 0), UsedExpenses: domain.NewAmount(200_000, 0),
		TotalTax:    domain.NewAmount(200_000, 0),
		CreditBasic: domain.NewAmount(30_840, 0), CreditSpouse: domain.NewAmount(49_680, 0),
		CreditDisability:     domain.NewAmount(21_180, 0),
		CreditDisability3:    domain.NewAmount(5_040, 0),
		CreditDisabilityZTPP: domain.NewAmount(16_140, 0),
		TotalCredits:         domain.NewAmount(101_700, 0),
	}
	bonus := &domain.IncomeTaxReturn{
		Year: 2025, FilingType: domain.FilingTypeRegular,
		TotalRevenue: domain.NewAmount(350_000, 0), UsedExpenses: domain.NewAmount(210_000, 0),
		TotalTax:    domain.NewAmount(21_000, 0),
		CreditBasic: domain.NewAmount(30_840, 0), TotalCredits: domain.NewAmount(30_840, 0),
		ChildBenefit: domain.NewAmount(37_524, 0),
	}
	return []formulaCase{
		{
			name: "full", itr: full,
			children: []domain.TaxChildCredit{{ChildName: "Petr Novak", BirthNumber: "150310/0005", ChildOrder: 1, MonthsClaimed: 12}},
			spouse:   &domain.TaxSpouseCredit{SpouseName: "Marie Novakova", SpouseBirthNumber: "855101/0006", MonthsClaimed: 12},
		},
		{name: "minimal", itr: minimal},
		{
			name: "all credits", itr: allCredits,
			spouse: &domain.TaxSpouseCredit{SpouseName: "Marie Novakova", SpouseBirthNumber: "855101/0006", SpouseZTP: true, MonthsClaimed: 12},
		},
		{
			name: "bonus", itr: bonus,
			children: []domain.TaxChildCredit{
				{ChildName: "Petr Novak", ChildOrder: 1, MonthsClaimed: 12},
				{ChildName: "Anna Novakova", ChildOrder: 2, MonthsClaimed: 12},
			},
		},
	}
}

func TestIncomeTaxXML_FormulaConsistency(t *testing.T) {
	// Row formulas and month x rate products of the published EPO controls.
	omitWhenZero := []string{
		"kc_dztrata", "kc_op15_1a", "kc_op15_1c", "kc_manztpp", "kc_op15_1d", "kc_op15_1e1", "kc_op15_1e2",
		"uhrn_slevy35ba", "kc_dazvyhod", "kc_slevy35c", "kc_danbonus", "kc_db_po_odpd", "kc_zalpred",
	}

	for _, tc := range formulaCases() {
		t.Run(tc.name, func(t *testing.T) {
			data, err := GenerateIncomeTaxXML(tc.itr, newCreditTestSettings(), tc.children, tc.spouse)
			if err != nil {
				t.Fatalf("GenerateIncomeTaxXML: %v", err)
			}
			d, s, _ := parseIncomeTax(t, data)
			get := func(k string) int64 { return attrInt(t, d, k) }

			r60 := get("da_celod13")
			r70 := get("kc_op15_1a") + get("kc_op15_1c") + get("kc_manztpp") + get("kc_op15_1d") +
				get("kc_op15_1e1") + get("kc_op15_1e2")
			if got := get("uhrn_slevy35ba"); got != r70 {
				t.Errorf("ř.70 = %d, want sum of rows 64-68 = %d", got, r70)
			}
			r71 := maxInt64(0, r60-r70)
			if got := get("da_slevy35ba"); got != r71 {
				t.Errorf("ř.71 = %d, want max(0, ř.60 - ř.70) = %d", got, r71)
			}

			r72 := 1267*get("m_deti") + 1860*get("m_deti2") + 2320*get("m_deti3") +
				2*(1267*get("m_detiztpp")+1860*get("m_detiztpp2")+2320*get("m_detiztpp3"))
			if got := get("kc_dazvyhod"); got != r72 {
				t.Errorf("ř.72 = %d, want %d from the child months", got, r72)
			}
			r73 := get("kc_slevy35c")
			if got := get("da_slevy35c"); got != r71-r73 {
				t.Errorf("ř.74 = %d, want ř.71 - ř.73 = %d", got, r71-r73)
			}
			if got := get("kc_danbonus"); got != r72-r73 {
				t.Errorf("ř.76 = %d, want ř.72 - ř.73 = %d", got, r72-r73)
			}
			r75 := get("kc_dan_celk")
			r76 := get("kc_danbonus")
			if got := get("kc_dan_po_db"); got != maxInt64(0, r75-r76) {
				t.Errorf("ř.77 = %d, want %d", got, maxInt64(0, r75-r76))
			}
			if got := get("kc_db_po_odpd"); got != maxInt64(0, r76-r75) {
				t.Errorf("ř.77a = %d, want %d", got, maxInt64(0, r76-r75))
			}

			products := []struct {
				amount, months string
				rate           int64
			}{
				{"kc_op15_1c", "m_vyzmanzl", 2070},
				{"kc_manztpp", "m_manz", 4140},
				{"kc_op15_1d", "m_cinvduch", 210},
				{"kc_op15_1e1", "m_invduch", 420},
				{"kc_op15_1e2", "m_ztpp", 1345},
			}
			for _, p := range products {
				if got, want := get(p.amount), p.rate*get(p.months); got != want {
					t.Errorf("%s = %d, want %d x %s = %d", p.amount, got, p.rate, p.months, want)
				}
			}

			if v, ok := s["kc_odcelk"]; ok {
				want := attrInt(t, s, "kc_op28_5") + attrInt(t, s, "kc_op15_13") +
					attrInt(t, s, "kc_op15_12") + attrInt(t, s, "kc_op15_8")
				if attrInt(t, s, "kc_odcelk") != want {
					t.Errorf("ř.54 = %s, want sum of rows 46-53 = %d", v, want)
				}
			}

			for _, key := range omitWhenZero {
				if v, ok := d[key]; ok && v == "0" {
					t.Errorf("%s is emitted as \"0\"; an unclaimed item must be left out", key)
				}
			}
			if v, ok := s["kc_odcelk"]; ok && v == "0" {
				t.Error("kc_odcelk is emitted as \"0\"")
			}
		})
	}

	t.Run("bonus case values", func(t *testing.T) {
		tc := formulaCases()[3]
		data, err := GenerateIncomeTaxXML(tc.itr, newCreditTestSettings(), tc.children, tc.spouse)
		if err != nil {
			t.Fatalf("GenerateIncomeTaxXML: %v", err)
		}
		d, _, _ := parseIncomeTax(t, data)
		if d["da_slevy35ba"] != "0" || d["kc_danbonus"] != "37524" || d["kc_db_po_odpd"] != "37524" {
			t.Errorf("ř.71/76/77a = %s/%s/%s, want 0/37524/37524", d["da_slevy35ba"], d["kc_danbonus"], d["kc_db_po_odpd"])
		}
		if _, ok := d["kc_slevy35c"]; ok {
			t.Error("kc_slevy35c must be absent when ř.73 is 0")
		}
	})
}

func TestIncomeTaxXML_DisabilityRows(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*domain.IncomeTaxReturn)
		want   map[string]string
		total  string
	}{
		{
			name: "level 1",
			mutate: func(i *domain.IncomeTaxReturn) {
				i.CreditDisability = domain.NewAmount(2_520, 0)
				i.CreditDisability12 = domain.NewAmount(2_520, 0)
			},
			want:  map[string]string{"kc_op15_1d": "2520", "m_cinvduch": "12"},
			total: "33360",
		},
		{
			name: "level 2",
			mutate: func(i *domain.IncomeTaxReturn) {
				i.CreditDisability = domain.NewAmount(5_040, 0)
				i.CreditDisability3 = domain.NewAmount(5_040, 0)
			},
			want:  map[string]string{"kc_op15_1e1": "5040", "m_invduch": "12"},
			total: "35880",
		},
		{
			name: "ZTP/P",
			mutate: func(i *domain.IncomeTaxReturn) {
				i.CreditDisability = domain.NewAmount(16_140, 0)
				i.CreditDisabilityZTPP = domain.NewAmount(16_140, 0)
			},
			want:  map[string]string{"kc_op15_1e2": "16140", "m_ztpp": "12"},
			total: "46980",
		},
		{
			name: "level 2 with ZTP/P",
			mutate: func(i *domain.IncomeTaxReturn) {
				i.CreditDisability = domain.NewAmount(21_180, 0)
				i.CreditDisability3 = domain.NewAmount(5_040, 0)
				i.CreditDisabilityZTPP = domain.NewAmount(16_140, 0)
			},
			want: map[string]string{
				"kc_op15_1e1": "5040", "m_invduch": "12", "kc_op15_1e2": "16140", "m_ztpp": "12",
			},
			total: "52020",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			itr := &domain.IncomeTaxReturn{
				Year: 2025, FilingType: domain.FilingTypeRegular,
				TotalRevenue: domain.NewAmount(1_000_000, 0), UsedExpenses: domain.NewAmount(200_000, 0),
				TotalTax:    domain.NewAmount(200_000, 0),
				CreditBasic: domain.NewAmount(30_840, 0),
			}
			tt.mutate(itr)
			itr.TotalCredits = itr.CreditBasic + itr.CreditDisability

			data, err := GenerateIncomeTaxXML(itr, newCreditTestSettings(), nil, nil)
			if err != nil {
				t.Fatalf("GenerateIncomeTaxXML: %v", err)
			}
			d, _, _ := parseIncomeTax(t, data)
			for k, v := range tt.want {
				if d[k] != v {
					t.Errorf("%s = %q, want %q", k, d[k], v)
				}
			}
			if d["uhrn_slevy35ba"] != tt.total {
				t.Errorf("uhrn_slevy35ba = %q, want %q", d["uhrn_slevy35ba"], tt.total)
			}
		})
	}
}

func TestIncomeTaxXML_SpouseZTPPUsesRow65b(t *testing.T) {
	newItr := func(spouseCredit domain.Amount) *domain.IncomeTaxReturn {
		return &domain.IncomeTaxReturn{
			Year: 2025, FilingType: domain.FilingTypeRegular,
			TotalRevenue: domain.NewAmount(1_000_000, 0), UsedExpenses: domain.NewAmount(200_000, 0),
			TotalTax:    domain.NewAmount(200_000, 0),
			CreditBasic: domain.NewAmount(30_840, 0), CreditSpouse: spouseCredit,
			TotalCredits: domain.NewAmount(30_840, 0) + spouseCredit,
		}
	}

	t.Run("ZTP/P spouse, 12 months", func(t *testing.T) {
		spouse := &domain.TaxSpouseCredit{SpouseName: "Marie Novakova", SpouseBirthNumber: "855101/0006", SpouseZTP: true, MonthsClaimed: 12}
		data, err := GenerateIncomeTaxXML(newItr(domain.NewAmount(49_680, 0)), newCreditTestSettings(), nil, spouse)
		if err != nil {
			t.Fatalf("GenerateIncomeTaxXML: %v", err)
		}
		d, _, _ := parseIncomeTax(t, data)
		if d["kc_manztpp"] != "49680" || d["m_manz"] != "12" {
			t.Errorf("kc_manztpp/m_manz = %q/%q, want 49680/12", d["kc_manztpp"], d["m_manz"])
		}
		if _, ok := d["kc_op15_1c"]; ok {
			t.Error("kc_op15_1c must be absent for a ZTP/P spouse")
		}
	})

	t.Run("regular spouse, 7 months", func(t *testing.T) {
		spouse := &domain.TaxSpouseCredit{SpouseName: "Marie Novakova", SpouseBirthNumber: "855101/0006", MonthsClaimed: 7}
		data, err := GenerateIncomeTaxXML(newItr(domain.NewAmount(14_490, 0)), newCreditTestSettings(), nil, spouse)
		if err != nil {
			t.Fatalf("GenerateIncomeTaxXML: %v", err)
		}
		d, _, _ := parseIncomeTax(t, data)
		if d["kc_op15_1c"] != "14490" || d["m_vyzmanzl"] != "7" {
			t.Errorf("kc_op15_1c/m_vyzmanzl = %q/%q, want 14490/7", d["kc_op15_1c"], d["m_vyzmanzl"])
		}
		if d["manz_jmeno"] != "Marie" || d["manz_prijmeni"] != "Novakova" || d["manz_r_cislo"] != "8551010006" {
			t.Errorf("spouse identity = %q %q %q", d["manz_jmeno"], d["manz_prijmeni"], d["manz_r_cislo"])
		}
	})
}

func TestIncomeTaxXML_RejectsInconsistentInput(t *testing.T) {
	base := func() *domain.IncomeTaxReturn {
		return &domain.IncomeTaxReturn{
			Year: 2025, FilingType: domain.FilingTypeRegular,
			TotalRevenue: domain.NewAmount(1_000_000, 0), UsedExpenses: domain.NewAmount(200_000, 0),
			TotalTax:    domain.NewAmount(200_000, 0),
			CreditBasic: domain.NewAmount(30_840, 0), TotalCredits: domain.NewAmount(30_840, 0),
		}
	}
	tests := []struct {
		name     string
		mutate   func(*domain.IncomeTaxReturn)
		settings func() map[string]string
	}{
		{"student credit", func(i *domain.IncomeTaxReturn) {
			i.CreditStudent = domain.NewAmount(4_020, 0)
			i.TotalCredits += i.CreditStudent
		}, nil},
		{"spouse credit without spouse", func(i *domain.IncomeTaxReturn) {
			i.CreditSpouse = domain.NewAmount(24_840, 0)
			i.TotalCredits += i.CreditSpouse
		}, nil},
		{"legacy disability without split", func(i *domain.IncomeTaxReturn) {
			i.CreditDisability = domain.NewAmount(2_520, 0)
			i.TotalCredits += i.CreditDisability
		}, nil},
		{"total credits differ from rows", func(i *domain.IncomeTaxReturn) {
			i.TotalCredits += domain.NewAmount(1_000, 0)
		}, nil},
		{"child benefit without children", func(i *domain.IncomeTaxReturn) {
			i.ChildBenefit = domain.NewAmount(15_204, 0)
		}, nil},
		{"revenue without NACE", func(i *domain.IncomeTaxReturn) {}, func() map[string]string {
			s := newCreditTestSettings()
			delete(s, "c_okec")
			return s
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			itr := base()
			tt.mutate(itr)
			settings := newCreditTestSettings()
			if tt.settings != nil {
				settings = tt.settings()
			}
			_, err := GenerateIncomeTaxXML(itr, settings, nil, nil)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestIncomeTaxXML_UcSoust(t *testing.T) {
	itr := &domain.IncomeTaxReturn{
		Year: 2025, FilingType: domain.FilingTypeRegular,
		TotalRevenue: domain.NewAmount(350_000, 0), UsedExpenses: domain.NewAmount(210_000, 0),
		TotalTax:    domain.NewAmount(21_000, 0),
		CreditBasic: domain.NewAmount(30_840, 0), TotalCredits: domain.NewAmount(30_840, 0),
	}

	data, err := GenerateIncomeTaxXML(itr, newCreditTestSettings(), nil, nil)
	if err != nil {
		t.Fatalf("GenerateIncomeTaxXML: %v", err)
	}
	if !bytes.Contains(data, []byte(`uc_soust="1"`)) {
		t.Errorf("actual-expense filer must emit uc_soust=\"1\", got:\n%s", data)
	}

	itr.FlatRatePercent = 60
	data, err = GenerateIncomeTaxXML(itr, newCreditTestSettings(), nil, nil)
	if err != nil {
		t.Fatalf("GenerateIncomeTaxXML: %v", err)
	}
	if bytes.Contains(data, []byte(`uc_soust`)) {
		t.Errorf("flat-rate filer must not emit uc_soust, got:\n%s", data)
	}
}
