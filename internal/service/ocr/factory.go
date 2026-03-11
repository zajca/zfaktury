package ocr

import "fmt"

// NewProvider creates an OCR provider based on the provider name.
// If model is empty, the provider's default model is used.
// If baseURL is non-empty, it overrides the provider's default endpoint.
func NewProvider(providerName, apiKey, model, baseURL string) (Provider, error) {
	switch providerName {
	case "openai", "":
		p := NewOpenAIProvider(apiKey, model)
		if baseURL != "" {
			p.SetBaseURL(baseURL)
		}
		return p, nil

	case "openrouter":
		p := NewOpenRouterProvider(apiKey, model)
		if baseURL != "" {
			p.SetBaseURL(baseURL)
		}
		return p, nil

	case "mistral":
		p := NewMistralProvider(apiKey, model)
		if baseURL != "" {
			p.SetBaseURL(baseURL)
		}
		return p, nil

	case "gemini":
		p := NewGeminiProvider(apiKey, model)
		if baseURL != "" {
			p.SetBaseURL(baseURL)
		}
		return p, nil

	case "claude":
		p := NewAnthropicProvider(apiKey, model)
		if baseURL != "" {
			p.SetBaseURL(baseURL)
		}
		return p, nil

	default:
		return nil, fmt.Errorf("unknown OCR provider: %q; supported: openai, openrouter, gemini, mistral, claude", providerName)
	}
}
