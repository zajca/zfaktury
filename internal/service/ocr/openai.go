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
	defaultTimeout   = 25 * time.Second
	defaultMaxTokens = 4096
)

// OpenAICompatibleProvider implements the Provider interface using the OpenAI-compatible
// Chat Completions API. Works with OpenAI, OpenRouter, Gemini, and Mistral.
type OpenAICompatibleProvider struct {
	name       string
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOpenAIProvider creates a provider for OpenAI's API.
func NewOpenAIProvider(apiKey, model string) *OpenAICompatibleProvider {
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAICompatibleProvider{
		name:       "openai",
		apiKey:     apiKey,
		baseURL:    "https://api.openai.com/v1/chat/completions",
		model:      model,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// NewOpenRouterProvider creates a provider for OpenRouter's API.
func NewOpenRouterProvider(apiKey, model string) *OpenAICompatibleProvider {
	if model == "" {
		model = "google/gemini-2.0-flash-001"
	}
	return &OpenAICompatibleProvider{
		name:       "openrouter",
		apiKey:     apiKey,
		baseURL:    "https://openrouter.ai/api/v1/chat/completions",
		model:      model,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// NewMistralProvider creates a provider for Mistral's API.
func NewMistralProvider(apiKey, model string) *OpenAICompatibleProvider {
	if model == "" {
		model = "pixtral-large-latest"
	}
	return &OpenAICompatibleProvider{
		name:       "mistral",
		apiKey:     apiKey,
		baseURL:    "https://api.mistral.ai/v1/chat/completions",
		model:      model,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// NewGeminiProvider creates a provider for Google Gemini's OpenAI-compatible API.
func NewGeminiProvider(apiKey, model string) *OpenAICompatibleProvider {
	if model == "" {
		model = "gemini-2.0-flash"
	}
	return &OpenAICompatibleProvider{
		name:       "gemini",
		apiKey:     apiKey,
		baseURL:    "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions",
		model:      model,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// Name returns the provider name.
func (p *OpenAICompatibleProvider) Name() string {
	return p.name
}

// SetBaseURL overrides the API endpoint URL.
func (p *OpenAICompatibleProvider) SetBaseURL(url string) {
	p.baseURL = url
}

// ProcessImage sends an image to the Chat Completions API and extracts structured invoice data.
func (p *OpenAICompatibleProvider) ProcessImage(ctx context.Context, imageData []byte, contentType string) (*domain.OCRResult, error) {
	if err := validateContentType(contentType); err != nil {
		return nil, err
	}

	b64Data := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", contentType, b64Data)

	reqBody := p.buildChatRequest(dataURL)

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshalling %s request: %w", p.name, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating %s request: %w", p.name, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling %s API: %w", p.name, err)
	}
	defer func() { _ = resp.Body.Close() }()

	const maxResponseBytes = 2 << 20 // 2 MB
	limited := io.LimitReader(resp.Body, maxResponseBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("reading %s response: %w", p.name, err)
	}
	if int64(len(body)) > maxResponseBytes {
		return nil, fmt.Errorf("%s response too large (> %d bytes)", p.name, maxResponseBytes)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s API returned status %d: %s", p.name, resp.StatusCode, truncate(string(body), 500))
	}

	return parseChatResponse(body, p.name)
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

// chatResponse represents the relevant parts of the Chat Completions API response.
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

func (p *OpenAICompatibleProvider) buildChatRequest(dataURL string) chatRequest {
	return chatRequest{
		Model: p.model,
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
		MaxTokens:   defaultMaxTokens,
		Temperature: 0.1,
	}
}

// ProcessWithPrompt sends data to the API with custom system and user prompts and returns the raw response text.
func (p *OpenAICompatibleProvider) ProcessWithPrompt(ctx context.Context, imageData []byte, contentType string, sysPrompt, usrPrompt string) (string, error) {
	if err := validateContentType(contentType); err != nil {
		return "", err
	}

	b64Data := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", contentType, b64Data)

	reqBody := chatRequest{
		Model: p.model,
		Messages: []chatMessage{
			{
				Role: "system",
				Content: []contentPart{
					{Type: "text", Text: sysPrompt},
				},
			},
			{
				Role: "user",
				Content: []contentPart{
					{Type: "text", Text: usrPrompt},
					{Type: "image_url", ImageURL: &imageURL{URL: dataURL}},
				},
			},
		},
		MaxTokens:   defaultMaxTokens,
		Temperature: 0.1,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshalling %s request: %w", p.name, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("creating %s request: %w", p.name, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("calling %s API: %w", p.name, err)
	}
	defer func() { _ = resp.Body.Close() }()

	const maxResponseBytes = 2 << 20 // 2 MB
	limited := io.LimitReader(resp.Body, maxResponseBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("reading %s response: %w", p.name, err)
	}
	if int64(len(body)) > maxResponseBytes {
		return "", fmt.Errorf("%s response too large (> %d bytes)", p.name, maxResponseBytes)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s API returned status %d: %s", p.name, resp.StatusCode, truncate(string(body), 500))
	}

	return parseChatResponseRaw(body, p.name)
}

// parseChatResponseRaw extracts the raw text content from the Chat Completions API response.
func parseChatResponseRaw(body []byte, providerName string) (string, error) {
	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("parsing %s response JSON: %w", providerName, err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("%s API error: %s", providerName, chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("%s returned no choices", providerName)
	}

	return chatResp.Choices[0].Message.Content, nil
}

// parseChatResponse extracts the OCR result from the Chat Completions API response body.
func parseChatResponse(body []byte, providerName string) (*domain.OCRResult, error) {
	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("parsing %s response JSON: %w", providerName, err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("%s API error: %s", providerName, chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("%s returned no choices", providerName)
	}

	content := chatResp.Choices[0].Message.Content
	return ParseOCRJSON(content)
}
