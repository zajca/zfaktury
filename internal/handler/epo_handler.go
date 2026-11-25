package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/epo"
	"github.com/zajca/zfaktury/internal/service"
)

// EPOHandler handles the "open in EPO" endpoint.
type EPOHandler struct {
	svc *service.EPOService
}

// NewEPOHandler creates a new EPOHandler.
func NewEPOHandler(svc *service.EPOService) *EPOHandler {
	return &EPOHandler{svc: svc}
}

// Routes returns a chi.Router with the EPO routes mounted.
func (h *EPOHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/{kind}/{id}/open-form", h.OpenForm)
	return r
}

// epoFindingResponse is one finding reported by EPO.
type epoFindingResponse struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Field   string `json:"field"`
	Section string `json:"section"`
	Row     string `json:"row"`
	Text    string `json:"text"`
}

// epoOpenFormResponse is the successful result of the open-form call.
type epoOpenFormResponse struct {
	URL      string               `json:"url"`
	Warnings []epoFindingResponse `json:"warnings"`
}

// epoErrorResponse is the error body; Findings is set for EPO rejections.
type epoErrorResponse struct {
	Error    string               `json:"error"`
	Findings []epoFindingResponse `json:"findings,omitempty"`
}

func epoFindingsFromDomain(findings []epo.Finding) []epoFindingResponse {
	out := make([]epoFindingResponse, 0, len(findings))
	for _, f := range findings {
		out = append(out, epoFindingResponse{
			Type:    f.Type,
			Code:    f.Code,
			Field:   f.Field,
			Section: f.Section,
			Row:     f.Row,
			Text:    f.Text,
		})
	}
	return out
}

var epoKinds = map[string]service.EPOKind{
	string(service.EPOKindVATReturn):        service.EPOKindVATReturn,
	string(service.EPOKindControlStatement): service.EPOKindControlStatement,
	string(service.EPOKindVIESSummary):      service.EPOKindVIESSummary,
	string(service.EPOKindIncomeTax):        service.EPOKindIncomeTax,
}

// OpenForm handles POST /{kind}/{id}/open-form. The response carries a bearer
// link to the prefilled form, so it is never logged.
func (h *EPOHandler) OpenForm(w http.ResponseWriter, r *http.Request) {
	kind, ok := epoKinds[chi.URLParam(r, "kind")]
	if !ok {
		respondError(w, http.StatusBadRequest, "Neznámý typ podání.")
		return
	}
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Neplatné ID podání.")
		return
	}

	link, err := h.svc.OpenForm(r.Context(), kind, id)
	if err != nil {
		h.respondOpenFormError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, epoOpenFormResponse{
		URL:      link.URL,
		Warnings: epoFindingsFromDomain(link.Warnings),
	})
}

func (h *EPOHandler) respondOpenFormError(w http.ResponseWriter, err error) {
	var rejected *epo.RejectedError
	switch {
	case errors.As(err, &rejected):
		respondJSON(w, http.StatusUnprocessableEntity, epoErrorResponse{
			Error:    "EPO v podání našlo chyby, formulář nebyl otevřen.",
			Findings: epoFindingsFromDomain(rejected.Findings),
		})
	case errors.Is(err, domain.ErrXMLNotGenerated):
		respondError(w, http.StatusConflict, "Nejdřív vygenerujte XML.")
	case errors.Is(err, domain.ErrNotFound):
		respondError(w, http.StatusNotFound, "Podání nebylo nalezeno.")
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusBadRequest, "Neplatný požadavek.")
	case errors.Is(err, epo.ErrPackaging), errors.Is(err, epo.ErrRequestTooLarge):
		slog.Error("EPO open-form request could not be packed", "error", err)
		respondError(w, http.StatusInternalServerError, "Podání se nepodařilo připravit pro EPO. Zkuste to znovu, nebo stáhněte XML a nahrajte ho ručně.")
	case errors.Is(err, epo.ErrInvalidXML):
		respondError(w, http.StatusUnprocessableEntity, "XML podání nelze odeslat do EPO. Vygenerujte ho znovu.")
	case errors.Is(err, epo.ErrEPOTimeout):
		respondError(w, http.StatusGatewayTimeout, "EPO neodpovědělo včas. Zkuste to znovu, nebo stáhněte XML a nahrajte ho ručně.")
	case errors.Is(err, epo.ErrEPOUnavailable), errors.Is(err, epo.ErrUnexpectedResponse):
		respondError(w, http.StatusBadGateway, "EPO je nedostupné, stáhněte XML a nahrajte ho ručně.")
	default:
		slog.Error("EPO open-form failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}
