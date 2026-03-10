package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

const (
	openAIAPIURL       = "https://api.openai.com/v1/chat/completions"
	openAIModel        = "gpt-4o"
	openAITimeout      = 25 * time.Second
	openAIMaxTokens    = 4096
)

// OpenAIProvider implements the Provider interface using OpenAI's vision API.
type OpenAIProvider struct {
	apiKey     string
	httpClient *http.Client
}

// NewOpenAIProvider creates a new OpenAI-based OCR provider.
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: openAITimeout,
		},
	}
}

// Name returns the provider name.
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// ProcessImage sends an image to OpenAI's vision model and extracts structured invoice data.
func (p *OpenAIProvider) ProcessImage(ctx context.Context, imageData []byte, contentType string) (*domain.OCRResult, error) {
	if err := validateContentType(contentType); err != nil {
		return nil, err
	}

	b64Data := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", contentType, b64Data)

	reqBody := buildChatRequest(dataURL)

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshalling OpenAI request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating OpenAI request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	const maxResponseBytes = 2 << 20 // 2 MB
	limited := io.LimitReader(resp.Body, maxResponseBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("reading OpenAI response: %w", err)
	}
	if int64(len(body)) > maxResponseBytes {
		return nil, fmt.Errorf("OpenAI response too large (> %d bytes)", maxResponseBytes)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, truncate(string(body), 500))
	}

	return parseOpenAIResponse(body)
}

// validateContentType checks that the content type is supported for OCR processing.
func validateContentType(contentType string) error {
	switch contentType {
	case "image/jpeg", "image/png", "application/pdf":
		return nil
	default:
		return fmt.Errorf("unsupported content type for OCR: %q; supported: image/jpeg, image/png, application/pdf", contentType)
	}
}

// chatRequest represents the OpenAI Chat Completions API request body.
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string        `json:"role"`
	Content []contentPart `json:"content,omitempty"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

// chatResponse represents the relevant parts of the OpenAI Chat Completions API response.
type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *apiError    `json:"error,omitempty"`
}

type chatChoice struct {
	Message chatResponseMessage `json:"message"`
}

type chatResponseMessage struct {
	Content string `json:"content"`
}

type apiError struct {
	Message string `json:"message"`
}

// ocrJSONResponse is the expected JSON structure from the model's output.
type ocrJSONResponse struct {
	VendorName     string            `json:"vendor_name"`
	VendorICO      string            `json:"vendor_ico"`
	VendorDIC      string            `json:"vendor_dic"`
	InvoiceNumber  string            `json:"invoice_number"`
	IssueDate      string            `json:"issue_date"`
	DueDate        string            `json:"due_date"`
	TotalAmount    float64           `json:"total_amount"`
	VATAmount      float64           `json:"vat_amount"`
	VATRatePercent int               `json:"vat_rate_percent"`
	CurrencyCode   string            `json:"currency_code"`
	Description    string            `json:"description"`
	Items          []ocrItemResponse `json:"items"`
	RawText        string            `json:"raw_text"`
	Confidence     float64           `json:"confidence"`
}

type ocrItemResponse struct {
	Description    string  `json:"description"`
	Quantity       float64 `json:"quantity"`
	UnitPrice      float64 `json:"unit_price"`
	VATRatePercent int     `json:"vat_rate_percent"`
	TotalAmount    float64 `json:"total_amount"`
}

const systemPrompt = `Jsi OCR asistent pro zpracovani faktur a uctenek. Analyzuj obrazek a extrahuj strukturovana data.

Vrat POUZE platny JSON objekt (bez markdown, bez komentaru) s nasledujici strukturou:
{
  "vendor_name": "nazev dodavatele",
  "vendor_ico": "ICO dodavatele",
  "vendor_dic": "DIC dodavatele",
  "invoice_number": "cislo faktury/dokladu",
  "issue_date": "datum vystaveni ve formatu YYYY-MM-DD",
  "due_date": "datum splatnosti ve formatu YYYY-MM-DD",
  "total_amount": celkova castka v CZK (cislo, napr. 1234.56),
  "vat_amount": castka DPH v CZK (cislo),
  "vat_rate_percent": sazba DPH v procentech (cele cislo, napr. 21),
  "currency_code": "kod meny (CZK, EUR, USD)",
  "description": "kratky popis dokladu",
  "items": [
    {
      "description": "popis polozky",
      "quantity": mnozstvi (cislo, napr. 1.5),
      "unit_price": jednotkova cena v CZK (cislo),
      "vat_rate_percent": sazba DPH polozky (cele cislo),
      "total_amount": celkova cena polozky v CZK (cislo)
    }
  ],
  "raw_text": "neupraveny text z dokladu",
  "confidence": mira jistoty 0.0-1.0
}

Dulezite:
- Castky jsou v korunach (CZK) jako desetinna cisla (napr. 1234.56 = 1234 Kc a 56 haleru)
- Pokud udaj neni na dokladu, pouzij prazdny retezec pro textova pole, 0 pro cisla
- Datum vzdy ve formatu YYYY-MM-DD
- Pro confidence pouzij hodnotu podle toho, jak jsi si jisty spravnosti extrakce`

const userPrompt = `Analyzuj tento doklad (faktura/uctenka) a extrahuj vsechna dostupna data do JSON formatu podle zadane struktury.`

func buildChatRequest(dataURL string) chatRequest {
	return chatRequest{
		Model: openAIModel,
		Messages: []chatMessage{
			{
				Role: "system",
				Content: []contentPart{
					{Type: "text", Text: systemPrompt},
				},
			},
			{
				Role: "user",
				Content: []contentPart{
					{Type: "text", Text: userPrompt},
					{Type: "image_url", ImageURL: &imageURL{URL: dataURL}},
				},
			},
		},
		MaxTokens:   openAIMaxTokens,
		Temperature: 0.1,
	}
}

// parseOpenAIResponse extracts the OCR result from the API response body.
func parseOpenAIResponse(body []byte) (*domain.OCRResult, error) {
	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("parsing OpenAI response JSON: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI returned no choices")
	}

	content := chatResp.Choices[0].Message.Content
	return ParseOCRJSON(content)
}

// ParseOCRJSON parses the model's JSON output into a domain.OCRResult.
// Exported for testing.
func ParseOCRJSON(content string) (*domain.OCRResult, error) {
	// Strip potential markdown code fences.
	content = stripCodeFences(content)

	var ocrResp ocrJSONResponse
	if err := json.Unmarshal([]byte(content), &ocrResp); err != nil {
		return nil, fmt.Errorf("parsing OCR JSON from model output: %w", err)
	}

	result := &domain.OCRResult{
		VendorName:     ocrResp.VendorName,
		VendorICO:      ocrResp.VendorICO,
		VendorDIC:      ocrResp.VendorDIC,
		InvoiceNumber:  ocrResp.InvoiceNumber,
		IssueDate:      ocrResp.IssueDate,
		DueDate:        ocrResp.DueDate,
		TotalAmount:    domain.Amount(czkToHalere(ocrResp.TotalAmount)),
		VATAmount:      domain.Amount(czkToHalere(ocrResp.VATAmount)),
		VATRatePercent: ocrResp.VATRatePercent,
		CurrencyCode:   ocrResp.CurrencyCode,
		Description:    ocrResp.Description,
		RawText:        ocrResp.RawText,
		Confidence:     ocrResp.Confidence,
	}

	for _, item := range ocrResp.Items {
		result.Items = append(result.Items, domain.OCRItem{
			Description:    item.Description,
			Quantity:       domain.Amount(floatToCents(item.Quantity)),
			UnitPrice:      domain.Amount(czkToHalere(item.UnitPrice)),
			VATRatePercent: item.VATRatePercent,
			TotalAmount:    domain.Amount(czkToHalere(item.TotalAmount)),
		})
	}

	return result, nil
}

// czkToHalere converts a CZK float amount to halere (int64).
// For example, 1234.56 becomes 123456.
func czkToHalere(czk float64) int64 {
	return int64(czk*100 + 0.5)
}

// floatToCents converts a float quantity to cents (int64).
// For example, 1.5 becomes 150.
func floatToCents(f float64) int64 {
	return int64(f*100 + 0.5)
}

// stripCodeFences removes markdown code fences from the content if present.
func stripCodeFences(s string) string {
	// Remove leading ```json or ``` and trailing ```
	if len(s) > 7 && s[:7] == "```json" {
		s = s[7:]
	} else if len(s) > 3 && s[:3] == "```" {
		s = s[3:]
	}
	if len(s) > 3 && s[len(s)-3:] == "```" {
		s = s[:len(s)-3]
	}
	// Trim whitespace that may remain.
	for len(s) > 0 && (s[0] == '\n' || s[0] == '\r' || s[0] == ' ') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}

// truncate returns the first n characters of s, appending "..." if truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
