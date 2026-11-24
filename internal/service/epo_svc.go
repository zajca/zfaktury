package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/epo"
)

// EPOKind names a document whose stored XML can be opened as an EPO form.
type EPOKind string

// EPO document kinds, as used in the API path.
const (
	EPOKindVATReturn        EPOKind = "vat-return"
	EPOKindControlStatement EPOKind = "control-statement"
	EPOKindVIESSummary      EPOKind = "vies-summary"
	EPOKindIncomeTax        EPOKind = "income-tax-return"
)

// EPOXMLSource returns the stored XML of one document.
type EPOXMLSource interface {
	GetXMLData(ctx context.Context, id int64) ([]byte, error)
}

// EPOClient is the part of the EPO client the service uses.
type EPOClient interface {
	Check(ctx context.Context, xmlData []byte) ([]epo.Finding, error)
	OpenForm(ctx context.Context, xmlData []byte) (string, error)
}

// EPOFormLink is the result of a successful open-form request. The URL is a
// bearer link to a form prefilled with the taxpayer's data.
type EPOFormLink struct {
	URL      string
	Warnings []epo.Finding
}

// EPOService opens stored filing XML as a prefilled form in the EPO portal.
type EPOService struct {
	sources map[EPOKind]EPOXMLSource
	client  EPOClient
}

// NewEPOService creates an EPOService. Only kinds present in sources can be opened.
func NewEPOService(sources map[EPOKind]EPOXMLSource, client EPOClient) *EPOService {
	return &EPOService{sources: sources, client: client}
}

// OpenForm validates the stored XML with EPO's test mode and, when EPO finds
// nothing blocking, uploads it again to obtain the form link. A blocking
// finding yields *epo.RejectedError carrying every finding. Neither the XML
// nor the link is logged.
func (s *EPOService) OpenForm(ctx context.Context, kind EPOKind, id int64) (*EPOFormLink, error) {
	start := time.Now()
	source, ok := s.sources[kind]
	if !ok {
		return nil, fmt.Errorf("opening EPO form: %w: unknown kind %q", domain.ErrInvalidInput, kind)
	}
	if id <= 0 {
		return nil, fmt.Errorf("opening EPO form: %w: document ID is required", domain.ErrInvalidInput)
	}

	xmlData, err := source.GetXMLData(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("opening EPO form for %s %d: %w", kind, id, err)
	}
	if len(xmlData) == 0 {
		return nil, fmt.Errorf("opening EPO form for %s %d: %w", kind, id, domain.ErrXMLNotGenerated)
	}

	log := func(result string, findings int, err error) {
		attrs := []any{
			"kind", string(kind),
			"id", id,
			"xml_bytes", len(xmlData),
			"result", result,
			"findings", findings,
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if err != nil {
			attrs = append(attrs, "error", err.Error())
			slog.Warn("EPO open-form failed", attrs...)
			return
		}
		slog.Info("EPO open-form finished", attrs...)
	}

	findings, err := s.client.Check(ctx, xmlData)
	if err != nil {
		log("error", 0, err)
		return nil, fmt.Errorf("checking %s %d with EPO: %w", kind, id, err)
	}
	for _, f := range findings {
		if f.IsBlocking() {
			log("rejected", len(findings), nil)
			return nil, fmt.Errorf("checking %s %d with EPO: %w", kind, id, &epo.RejectedError{Findings: findings})
		}
	}

	link, err := s.client.OpenForm(ctx, xmlData)
	if err != nil {
		var rejected *epo.RejectedError
		if errors.As(err, &rejected) {
			log("rejected", len(rejected.Findings), nil)
		} else {
			log("error", len(findings), err)
		}
		return nil, fmt.Errorf("opening EPO form for %s %d: %w", kind, id, err)
	}

	log("url", len(findings), nil)
	return &EPOFormLink{URL: link, Warnings: findings}, nil
}
