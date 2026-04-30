// Package codebooks provides static reference data used in EPO submissions:
// the financial-office (ufo) codebook and the CZ-NACE 2025 economic-activity
// classification (used as c_nace in DPFDP7). Sources are embedded at compile
// time so lookups are zero-cost and the app stays self-contained.
//
// Sources:
//   - ufo: https://podpora.mojedane.gov.cz/cs/seznam-okruhu/rozhrani-pro-treti-strany/informace-k-ciselniku-ufo-platnem-od-1-1-4382
//   - CZ-NACE 2025 (level 5, 715 leaf subclasses):
//     https://apl2.czso.cz/iSMS/cisdata.jsp?kodcis=6105 (export CSV)
package codebooks

import (
	_ "embed"
	"encoding/csv"
	"sort"
	"strings"
	"sync"
)

// FinancialOffice represents a single entry in the EPO ufo codebook.
type FinancialOffice struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// FinancialOffices returns the EPO ufo codebook in display order (krajské FÚ
// 451..464, then SFÚ 13). Caller must not mutate the slice.
func FinancialOffices() []FinancialOffice {
	return financialOffices
}

// FinancialOfficeByCode looks up a c_ufo / c_ufo_cil entry. Returns nil if
// the code is not in the codebook.
func FinancialOfficeByCode(code string) *FinancialOffice {
	code = strings.TrimSpace(code)
	for i := range financialOffices {
		if financialOffices[i].Code == code {
			return &financialOffices[i]
		}
	}
	return nil
}

var financialOffices = []FinancialOffice{
	{Code: "451", Name: "Finanční úřad pro hlavní město Prahu"},
	{Code: "452", Name: "Finanční úřad pro Středočeský kraj"},
	{Code: "453", Name: "Finanční úřad pro Jihočeský kraj"},
	{Code: "454", Name: "Finanční úřad pro Plzeňský kraj"},
	{Code: "455", Name: "Finanční úřad pro Karlovarský kraj"},
	{Code: "456", Name: "Finanční úřad pro Ústecký kraj"},
	{Code: "457", Name: "Finanční úřad pro Liberecký kraj"},
	{Code: "458", Name: "Finanční úřad pro Královéhradecký kraj"},
	{Code: "459", Name: "Finanční úřad pro Pardubický kraj"},
	{Code: "460", Name: "Finanční úřad pro Kraj Vysočina"},
	{Code: "461", Name: "Finanční úřad pro Jihomoravský kraj"},
	{Code: "462", Name: "Finanční úřad pro Olomoucký kraj"},
	{Code: "463", Name: "Finanční úřad pro Moravskoslezský kraj"},
	{Code: "464", Name: "Finanční úřad pro Zlínský kraj"},
	{Code: "13", Name: "Specializovaný finanční úřad"},
}

// NACEEntry represents one CZ-NACE 2025 subclass (5-digit code) loaded from
// the CSU codebook. Code is the 5-digit form ("62109"); to emit it as the
// EPO c_nace attribute the value is right-padded with one zero ("621090").
type NACEEntry struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

//go:embed nace_2025.csv
var naceCSV []byte

var (
	naceOnce    sync.Once
	naceEntries []NACEEntry
	naceByCode  map[string]*NACEEntry
)

func loadNACE() {
	r := csv.NewReader(strings.NewReader(string(naceCSV)))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		return
	}
	if len(rows) == 0 {
		return
	}
	header := rows[0]
	codeCol, nameCol := -1, -1
	for i, h := range header {
		switch h {
		case "chodnota":
			codeCol = i
		case "text":
			nameCol = i
		}
	}
	if codeCol < 0 || nameCol < 0 {
		return
	}
	out := make([]NACEEntry, 0, len(rows)-1)
	idx := make(map[string]*NACEEntry, len(rows)-1)
	for _, row := range rows[1:] {
		if len(row) <= codeCol || len(row) <= nameCol {
			continue
		}
		entry := NACEEntry{Code: row[codeCol], Name: row[nameCol]}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	for i := range out {
		idx[out[i].Code] = &out[i]
	}
	naceEntries = out
	naceByCode = idx
}

// NACE returns the full CZ-NACE 2025 level-5 codebook sorted by code. Caller
// must not mutate the slice. The slice is built once on first call.
func NACE() []NACEEntry {
	naceOnce.Do(loadNACE)
	return naceEntries
}

// NACEByCode looks up a 5-digit CZ-NACE 2025 code. Returns nil if not found.
// Accepts the 5-digit form ("62109") or the 6-digit EPO form ("621090") --
// the trailing zero added for EPO is stripped before lookup.
func NACEByCode(code string) *NACEEntry {
	naceOnce.Do(loadNACE)
	code = strings.TrimSpace(code)
	if len(code) == 6 && strings.HasSuffix(code, "0") {
		code = code[:5]
	}
	return naceByCode[code]
}
