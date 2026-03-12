package ocr

import (
	"encoding/json"
	"fmt"
)

const investmentSystemPrompt = `Jsi specialista na analyzu brokerskych vypisu a investicnich reportu. Analyzuj dokument a extrahuj strukturovana data o investicnich prijmech.

Vrat POUZE platny JSON objekt (bez markdown, bez komentaru) s nasledujici strukturou:
{
  "platform": "identifikator platformy",
  "capital_entries": [
    {
      "category": "dividend_cz|dividend_foreign|interest",
      "description": "popis - napr. nazev akcie, typ dividendy",
      "income_date": "YYYY-MM-DD",
      "gross_amount": castka v CZK (cislo, napr. 1234.56),
      "withheld_tax_cz": srazena dan v CR (cislo),
      "withheld_tax_foreign": srazena dan v zahranici (cislo),
      "country_code": "2-pismenny kod zeme (US, DE, IE...)",
      "needs_declaring": true/false
    }
  ],
  "transactions": [
    {
      "asset_type": "stock|etf|bond|crypto",
      "asset_name": "nazev aktiva (napr. Vanguard S&P 500 ETF)",
      "isin": "ISIN kod pokud je dostupny",
      "transaction_type": "buy|sell",
      "transaction_date": "YYYY-MM-DD",
      "quantity": pocet kusu (cislo, napr. 1.5),
      "unit_price": cena za kus v originalni mene (cislo),
      "total_amount": celkova castka v originalni mene (cislo),
      "fees": poplatky (cislo),
      "currency_code": "kod meny (CZK, EUR, USD)",
      "exchange_rate": kurz k CZK (cislo, napr. 25.12 pro EUR)
    }
  ],
  "confidence": mira jistoty 0.0-1.0
}

Dulezite:
- Castky jsou v CZK jako desetinna cisla (napr. 1234.56 = 1234 Kc a 56 haleru), pokud neni uvedena jina mena
- Pro zahranicni transakce uved originalni menu a kurz k CZK
- Datum vzdy ve formatu YYYY-MM-DD
- category pro dividendy: "dividend_cz" pro ceske, "dividend_foreign" pro zahranicni
- needs_declaring = true pro prijmy ktere je treba uvest v danoven priznani
- Pokud udaj neni na dokladu, pouzij prazdny retezec pro textova pole, 0 pro cisla
- Pro confidence pouzij hodnotu podle toho, jak jsi si jisty spravnosti extrakce`

// InvestmentSystemPrompt returns the system prompt for investment extraction.
func InvestmentSystemPrompt() string {
	return investmentSystemPrompt
}

// InvestmentUserPrompt returns a platform-specific user prompt for investment extraction.
func InvestmentUserPrompt(platform string) string {
	return investmentUserPromptByPlatform(platform)
}

// investmentUserPromptByPlatform returns a platform-specific user prompt.
func investmentUserPromptByPlatform(platform string) string {
	base := "Analyzuj tento brokersky vypis a extrahuj vsechna data o investicnich prijmech (dividendy, uroky) a transakcich (nakupy, prodeje) do JSON formatu."

	hints := map[string]string{
		"portu":      " Dokument je z platformy Portu (cesky robo-advisor). Ocekavej ETF transakce, dividendy z ETF fondu, a meny EUR/USD/CZK.",
		"zonky":      " Dokument je z platformy Zonky (P2P pujcky). Ocekavej uroky z pujcek jako kapitalove prijmy v CZK.",
		"trading212": " Dokument je z platformy Trading212. Ocekavej akcie, ETF, dividendy v ruznych menach (GBP, USD, EUR).",
		"revolut":    " Dokument je z platformy Revolut. Ocekavej akcie, ETF, dividendy, mozne kryptomeny, v ruznych menach.",
		"other":      "",
	}

	hint, ok := hints[platform]
	if !ok {
		hint = ""
	}

	return base + hint
}

// InvestmentExtractionResponse is the expected JSON structure from the AI model.
type InvestmentExtractionResponse struct {
	Platform       string                 `json:"platform"`
	CapitalEntries []capitalEntryResponse `json:"capital_entries"`
	Transactions   []securityTxResponse   `json:"transactions"`
	Confidence     float64                `json:"confidence"`
}

type capitalEntryResponse struct {
	Category           string  `json:"category"`
	Description        string  `json:"description"`
	IncomeDate         string  `json:"income_date"`
	GrossAmount        float64 `json:"gross_amount"`
	WithheldTaxCZ      float64 `json:"withheld_tax_cz"`
	WithheldTaxForeign float64 `json:"withheld_tax_foreign"`
	CountryCode        string  `json:"country_code"`
	NeedsDeclaring     bool    `json:"needs_declaring"`
}

type securityTxResponse struct {
	AssetType       string  `json:"asset_type"`
	AssetName       string  `json:"asset_name"`
	ISIN            string  `json:"isin"`
	TransactionType string  `json:"transaction_type"`
	TransactionDate string  `json:"transaction_date"`
	Quantity        float64 `json:"quantity"`
	UnitPrice       float64 `json:"unit_price"`
	TotalAmount     float64 `json:"total_amount"`
	Fees            float64 `json:"fees"`
	CurrencyCode    string  `json:"currency_code"`
	ExchangeRate    float64 `json:"exchange_rate"`
}

// ParseInvestmentJSON parses the AI model's JSON output into an InvestmentExtractionResponse.
func ParseInvestmentJSON(content string) (*InvestmentExtractionResponse, error) {
	content = stripCodeFences(content)

	var resp InvestmentExtractionResponse
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		return nil, fmt.Errorf("parsing investment JSON from model output: %w", err)
	}

	return &resp, nil
}

// CzkToHalere converts a CZK float amount to halere (int64). Exported wrapper.
func CzkToHalere(czk float64) int64 {
	return czkToHalere(czk)
}

// QuantityToInt converts a float quantity to int64 (1/10000 units).
func QuantityToInt(qty float64) int64 {
	return int64(qty*10000 + 0.5)
}

// ExchangeRateToInt converts a float exchange rate to int64 (rate * 10000).
func ExchangeRateToInt(rate float64) int64 {
	return int64(rate*10000 + 0.5)
}
