package ocr

import (
	"encoding/json"
	"fmt"
)

const employmentSystemPrompt = `Jsi specialista na analyzu ceskych Potvrzeni o zdanitelnych prijmech ze zavisle cinnosti (§ 6 ZDP). Analyzuj prilozeny dokument a extrahuj strukturovana data potrebna pro vyplneni Priznani k dani z prijmu fyzickych osob (DPFO).

Identifikace varianty formulare (podle hlavicky / nazvu dokumentu):
- "25 5460 MFin 5460 - vzor c. 33" nebo "Potvrzeni o zdanitelnych prijmech ze zavisle cinnosti" -> certificate_type = "advance" (zalohove zdaneni; sraz zaloh dle § 38h ZDP)
- "25 5460/A MFin 5460/A - vzor c. 12" nebo "Potvrzeni o vyplacenych prijmech ... srazenou dani zvlastni sazbou" -> certificate_type = "withholding" (srazkove zdaneni dle § 36 odst. 6/7 ZDP)

Identifikace zamestnavatele a obdobi:
- employer_name = nazev plátce dane (zamestnavatele)
- employer_ico = ICO plátce (8-mistne cislo)
- employer_address = adresa sidla plátce
- period_from / period_to = zdanovaci obdobi (zacatek / konec; format YYYY-MM-DD)

Detekce typu pracovniho pomeru / smlouvy (z textu Potvrzeni nebo prilozenych poznamek):
- "Dohoda o pracovni cinnosti" / "DPC" -> contract_type = "dpc"
- "Dohoda o provedeni prace" / "DPP" -> contract_type = "dpp"
- "Pracovni pomer" / "HPP" / "hlavni pracovni pomer" -> contract_type = "hpp"
- jinak / nelze urcit -> contract_type = "other"

Extrakce castek pro variantu vzor 33 (advance) — radek po radku z Potvrzeni:
- r.2 = uhrn zuctovanych prijmu ze zavisle cinnosti
- r.4 = dalsi zdanitelne prijmy (napr. nepenezni plneni, benefity nad limit)
- r.5 = uhrn mesicnich danovych bonusu vyplacenych zamestnavatelem (cast 1)
- r.8 = uhrn srazenych zaloh na dan po slevach
- r.13 = uhrn mesicnich danovych bonusu (cast 2 / pripadny doplatek z RZ)
- polozka "vraceny preplatek z rocniho zuctovani" (pokud je uvedena)

Mapovani na vystupni JSON (pro vzor 33):
- gross_income_czk = r.2 + r.4  (DULEZITE: NIKOLIV jen r.2 nebo jen r.4)
- monthly_bonus_paid_czk = r.5 + r.13  (per oficialni Pokyny 2025 a XSD doc na atributu kc_vyplbonus: "soucet r.5 a 13")
- advance_tax_withheld_czk = r.8
- annual_settlement_refund_czk = vraceny preplatek z rocniho zuctovani (snizuje sraz. zalohy)
- foreign_tax_paid_czk = dan zaplacena v zahranici dle § 6 odst. 13 ZDP (muze chybet -> 0)
- income_without_advance_czk = cast prijmu, u kterych plátce nemel povinnost srazit zalohy dle § 38h ZDP (typicky prijmy zamestnancu zahranicnich zastupitelskych uradu v CR a prijmy ze zahranicniho zamestnavatele bez stale provozovny v CR; muze chybet -> 0)

Extrakce castek pro variantu vzor 12 (withholding):
- r.2 = uhrn vyplacenych prijmu
- polozka "srazena dan zvlastni sazbou" (typicky uvedena pod r.2)

Mapovani na vystupni JSON (pro vzor 12):
- gross_income_czk = r.2
- withheld_final_tax_czk = srazena dan zvlastni sazbou
- pole urcena pro vzor 33 (advance_tax_withheld_czk, monthly_bonus_paid_czk, annual_settlement_refund_czk, foreign_tax_paid_czk, income_without_advance_czk) zustavaji 0

Vrat POUZE platny JSON objekt (bez markdown, bez komentaru) s nasledujici strukturou:
{
  "certificate_type": "advance|withholding",
  "employer_name": "...",
  "employer_ico": "...",
  "employer_address": "...",
  "contract_type": "dpc|dpp|hpp|other",
  "period_from": "YYYY-MM-DD",
  "period_to": "YYYY-MM-DD",
  "gross_income_czk": 0.0,
  "income_without_advance_czk": 0.0,
  "foreign_tax_paid_czk": 0.0,
  "advance_tax_withheld_czk": 0.0,
  "annual_settlement_refund_czk": 0.0,
  "monthly_bonus_paid_czk": 0.0,
  "withheld_final_tax_czk": 0.0,
  "confidence": 0.0,
  "raw_text": "..."
}

Dulezite:
- Castky jsou v CZK jako desetinna cisla (napr. 1234.56 = 1234 Kc a 56 haleru). Tato vrstva NEPREVADI na halere — to dela aplikacni vrstva.
- Datum vzdy ve formatu YYYY-MM-DD.
- Pokud udaj neni na dokladu, pouzij prazdny retezec pro textova pole, 0 pro cisla.
- raw_text omez na maximalne 2000 znaku (pro ucely auditu).
- confidence v intervalu [0.0, 1.0] podle toho, jak jsi si jisty spravnosti extrakce (priblizny vodítka: < 0.5 = nizka jistota, 0.5–0.8 = stredni, > 0.8 = vysoka).
- ICO uved jako retezec presne 8 cislic (vcetne pripadnych vodicich nul).`

const employmentUserPrompt = `Analyzuj prilozene Potvrzeni o zdanitelnych prijmech ze zavisle cinnosti (vzor 33) nebo Potvrzeni o vyplacenych prijmech a srazene dani zvlastni sazbou (vzor 12) a extrahuj vsechna pozadovana data do JSON formatu podle zadane struktury.`

// EmploymentSystemPrompt returns the Czech system prompt for §6 employment income
// certificate extraction (Potvrzeni o zdanitelnych prijmech ze zavisle cinnosti).
func EmploymentSystemPrompt() string {
	return employmentSystemPrompt
}

// EmploymentUserPrompt returns the user instruction prefix for employment certificate OCR.
func EmploymentUserPrompt() string {
	return employmentUserPrompt
}

// EmploymentExtractionResponse is the expected JSON structure returned by the AI model
// when processing a §6 employment income certificate.
//
// JSON tags are present here because this is the OCR-layer DTO; downstream services
// convert these float CZK values to domain.Amount (halere) before persisting.
type EmploymentExtractionResponse struct {
	CertificateType           string  `json:"certificate_type"`
	EmployerName              string  `json:"employer_name"`
	EmployerICO               string  `json:"employer_ico"`
	EmployerAddress           string  `json:"employer_address"`
	ContractType              string  `json:"contract_type"`
	PeriodFrom                string  `json:"period_from"`
	PeriodTo                  string  `json:"period_to"`
	GrossIncomeCZK            float64 `json:"gross_income_czk"`
	IncomeWithoutAdvanceCZK   float64 `json:"income_without_advance_czk"`
	ForeignTaxPaidCZK         float64 `json:"foreign_tax_paid_czk"`
	AdvanceTaxWithheldCZK     float64 `json:"advance_tax_withheld_czk"`
	AnnualSettlementRefundCZK float64 `json:"annual_settlement_refund_czk"`
	MonthlyBonusPaidCZK       float64 `json:"monthly_bonus_paid_czk"`
	WithheldFinalTaxCZK       float64 `json:"withheld_final_tax_czk"`
	Confidence                float64 `json:"confidence"`
	RawText                   string  `json:"raw_text"`
}

// ParseEmploymentResponse parses the raw AI model output into an EmploymentExtractionResponse.
// Markdown code fences are stripped before JSON parsing.
func ParseEmploymentResponse(raw string) (*EmploymentExtractionResponse, error) {
	content := stripCodeFences(raw)

	var resp EmploymentExtractionResponse
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		return nil, fmt.Errorf("parsing employment JSON from model output: %w", err)
	}

	return &resp, nil
}
