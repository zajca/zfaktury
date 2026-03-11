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
	anthropicDefaultURL   = "https://api.anthropic.com/v1/messages"
	anthropicDefaultModel = "claude-sonnet-4-20250514"
	anthropicAPIVersion   = "2023-06-01"
)

// AnthropicProvider implements the Provider interface using Anthropic's Messages API.
type AnthropicProvider struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewAnthropicProvider creates a provider for Anthropic's Claude API.
func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	if model == "" {
		model = anthropicDefaultModel
	}
	return &AnthropicProvider{
		apiKey:     apiKey,
		baseURL:    anthropicDefaultURL,
		model:      model,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the provider name.
func (p *AnthropicProvider) Name() string {
	return "claude"
}

// SetBaseURL overrides the API endpoint URL.
func (p *AnthropicProvider) SetBaseURL(url string) {
	p.baseURL = url
}

// ProcessImage sends an image to Anthropic's Messages API and extracts structured invoice data.
func (p *AnthropicProvider) ProcessImage(ctx context.Context, imageData []byte, contentType string) (*domain.OCRResult, error) {
	if err := validateContentType(contentType); err != nil {
		return nil, err
	}

	b64Data := base64.StdEncoding.EncodeToString(imageData)

	reqBody := p.buildMessagesRequest(b64Data, contentType)

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshalling anthropic request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating anthropic request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling anthropic API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	const maxResponseBytes = 2 << 20 // 2 MB
	limited := io.LimitReader(resp.Body, maxResponseBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("reading anthropic response: %w", err)
	}
	if int64(len(body)) > maxResponseBytes {
		return nil, fmt.Errorf("anthropic response too large (> %d bytes)", maxResponseBytes)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic API returned status %d: %s", resp.StatusCode, truncate(string(body), 500))
	}

	return parseAnthropicResponse(body)
}

// Anthropic Messages API request types.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string                `json:"role"`
	Content []anthropicContentPart `json:"content"`
}

type anthropicContentPart struct {
	Type   string                `json:"type"`
	Text   string                `json:"text,omitempty"`
	Source *anthropicImageSource `json:"source,omitempty"`
}

type anthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// Anthropic Messages API response types.
type anthropicResponse struct {
	Content []anthropicResponseContent `json:"content"`
	Error   *anthropicError            `json:"error,omitempty"`
}

type anthropicResponseContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (p *AnthropicProvider) buildMessagesRequest(b64Data, mediaType string) anthropicRequest {
	return anthropicRequest{
		Model:     p.model,
		MaxTokens: defaultMaxTokens,
		System:    systemPrompt,
		Messages: []anthropicMessage{
			{
				Role: "user",
				Content: []anthropicContentPart{
					{
						Type: "image",
						Source: &anthropicImageSource{
							Type:      "base64",
							MediaType: mediaType,
							Data:      b64Data,
						},
					},
					{
						Type: "text",
						Text: userPrompt,
					},
				},
			},
		},
	}
}

// parseAnthropicResponse extracts the OCR result from the Anthropic Messages API response.
func parseAnthropicResponse(body []byte) (*domain.OCRResult, error) {
	var resp anthropicResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing anthropic response JSON: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("anthropic API error: %s", resp.Error.Message)
	}

	if len(resp.Content) == 0 {
		return nil, fmt.Errorf("anthropic returned no content")
	}

	// Find the first text content block.
	for _, block := range resp.Content {
		if block.Type == "text" {
			return ParseOCRJSON(block.Text)
		}
	}

	return nil, fmt.Errorf("anthropic returned no text content")
}
