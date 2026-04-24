package ocr

import (
	"encoding/json"
	"fmt"
)

const deductionSystemPrompt = `Jsi specialista na analyzu dokladu pro nezdanitelne casti zakladu dane (odpocty) z ceskeho danoveho priznani DPFO.

Analyzuj prilozeny doklad a urc, ke ktere kategorii odpoctu patri, a extrahuj udaje potrebne pro vyplneni priznani.

Vrat POUZE platny JSON objekt (bez markdown, bez komentaru) s nasledujici strukturou:
{
  "category": "jedna z hodnot: mortgage|life_insurance|pension|donation|union_dues|unknown",
  "provider_name": "nazev instituce, ktera potvrzeni vystavila (banka, pojistovna, penzijni spolecnost, obdarovany subjekt, odborova organizace)",
  "provider_ico": "ICO poskytovatele / obdarovaneho (pokud je uvedeno), prazdne pokud neni",
  "contract_number": "cislo smlouvy / variabilni symbol / cislo potvrzeni",
  "document_date": "datum vystaveni dokladu ve formatu YYYY-MM-DD",
  "period_year": rok za ktery se odpocet uplatnuje (cele cislo, napr. 2025),
  "amount_czk": castka v CZK v korunach jako desetinne cislo (napr. 12345.50),
  "purpose": "ucel daru nebo jiny popis - pouze pro category=donation a union_dues, jinak prazdne",
  "description_suggestion": "kratky nazev pro odpocet - napr. 'Hypotecni urok - Ceska sporitelna', 'Zivotni pojisteni NN 2025' (ceske)",
  "confidence": mira jistoty extrakce 0.0-1.0,
  "raw_text": "neupraveny text z dokladu"
}

Jak urcit kategorii:
- "mortgage" = potvrzeni banky / stavebni sporitelny o zaplacenych urocich z hypoteky nebo uveru ze stavebniho sporeni. Klicova slova: "hypotecni uver", "urok", "Cesta sporitelna", "KB Hypoteka", "stavebni sporeni", "Raiffeisenbank Hypoteka".
- "life_insurance" = potvrzeni pojistovny o zaplacenem pojistnem na soukrome zivotni pojisteni. Klicova slova: "zivotni pojisteni", "pojistne", "NN", "Allianz", "Ceska podnikatelska pojistovna", "Kooperativa".
- "pension" = potvrzeni penzijni spolecnosti o zaplacenych prispevcich na penzijni sporeni/doplnkove penzijni sporeni (po odecteni statniho prispevku). Klicova slova: "penzijni spolecnost", "doplnkove penzijni sporeni", "prispevky ucastnika".
- "donation" = darovaci smlouva nebo potvrzeni obdarovaneho. Klicova slova: "darovaci smlouva", "potvrzeni o daru", "obdarovany", "ucel daru".
- "union_dues" = potvrzeni odborove organizace o zaplacenych clenskych prispevcich. Klicova slova: "odborova organizace", "odborovy svaz", "clenske prispevky".
- "unknown" = pokud se nepodari spolehlive urcit kategorii.

Dulezite:
- amount_czk je castka, kterou lze odecist - u penzijniho sporeni to JE castka PO odecteni statniho prispevku (pokud je rozdil, vezmi tu nizsi)
- U hypoteky amount_czk = soucet urocenych plateb za rok
- period_year je rok, za ktery se odpocet uplatnuje (ne rok vystaveni dokladu, pokud se lisi)
- Datum vzdy ve formatu YYYY-MM-DD
- Pokud udaj neni na dokladu, pouzij prazdny retezec pro textova pole, 0 pro cisla
- Pro confidence pouzij hodnotu podle toho, jak jsi si jisty spravnosti extrakce`

const deductionUserPrompt = `Analyzuj tento doklad pro nezdanitelnou cast zakladu dane (odpocet) a extrahuj vsechna pozadovana data do JSON formatu podle zadane struktury.`

// DeductionSystemPrompt returns the system prompt for tax deduction document extraction.
func DeductionSystemPrompt() string {
	return deductionSystemPrompt
}

// DeductionUserPrompt returns the user prompt for tax deduction document extraction.
func DeductionUserPrompt() string {
	return deductionUserPrompt
}

// DeductionExtractionResponse is the expected JSON structure from the AI model
// when processing a tax deduction proof document.
type DeductionExtractionResponse struct {
	Category              string  `json:"category"`
	ProviderName          string  `json:"provider_name"`
	ProviderICO           string  `json:"provider_ico"`
	ContractNumber        string  `json:"contract_number"`
	DocumentDate          string  `json:"document_date"`
	PeriodYear            int     `json:"period_year"`
	AmountCZK             float64 `json:"amount_czk"`
	Purpose               string  `json:"purpose"`
	DescriptionSuggestion string  `json:"description_suggestion"`
	Confidence            float64 `json:"confidence"`
	RawText               string  `json:"raw_text"`
}

// ParseDeductionJSON parses the AI model's JSON output into a DeductionExtractionResponse.
// Exported for testing.
func ParseDeductionJSON(content string) (*DeductionExtractionResponse, error) {
	content = stripCodeFences(content)

	var resp DeductionExtractionResponse
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		return nil, fmt.Errorf("parsing deduction JSON from model output: %w", err)
	}

	return &resp, nil
}
