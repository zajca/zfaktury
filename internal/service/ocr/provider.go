package ocr

import (
	"context"

	"github.com/zajca/zfaktury/internal/domain"
)

// Provider extracts structured data from document images.
type Provider interface {
	ProcessImage(ctx context.Context, imageData []byte, contentType string) (*domain.OCRResult, error)
	ProcessWithPrompt(ctx context.Context, imageData []byte, contentType string, systemPrompt, userPrompt string) (string, error)
	Name() string
}
