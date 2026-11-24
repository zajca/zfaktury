package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/epo"
	"github.com/zajca/zfaktury/internal/isds"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/xsdcheck"
)

const (
	// filingsPath is the mount path of the filing API. The dev CORS middleware
	// skips it so no foreign origin can trigger a send.
	filingsPath = "/api/v1/filings"

	// maxFilingBody bounds the JSON body of a filing request.
	maxFilingBody = 4 << 10

	// maxXSDFindings bounds the schema messages returned for one XML.
	maxXSDFindings = 20
)

const (
	// msgFilingBlockedHint tells the user how to get out of a send whose
	// outcome is not known.
	msgFilingBlockedHint = "Podání se znovu neodešle, dokud výsledek nepotvrdíte: zkontrolujte odeslané zprávy v datové schránce a podle nálezu u odeslání použijte akci resolve (attach s číslem zprávy, nebo notSent), případně podání podejte ručně přes EPO."

	msgFilingBlocked = "Toto podání už bylo do tohoto prostředí odesláno, nebo se výsledek dřívějšího odeslání nepodařilo zjistit. " + msgFilingBlockedHint
)

// filingOp names the action a request performs, so a failure is explained in
// terms of that action.
type filingOp string

const (
	opPreview   filingOp = "preview"
	opSend      filingOp = "send"
	opRefresh   filingOp = "refresh"
	opResolve   filingOp = "resolve"
	opLoginTest filingOp = "login-test"
	opRead      filingOp = "read"
)

// FilingHandler serves the data-box filing API: preview, confirmed send,
// delivery refresh and receipt download.
type FilingHandler struct {
	svc      *service.FilingService
	bindHost string
}

// NewFilingHandler creates a FilingHandler. The routes refuse to work unless
// bindHost is a loopback address.
func NewFilingHandler(svc *service.FilingService, bindHost string) *FilingHandler {
	return &FilingHandler{svc: svc, bindHost: bindHost}
}

// Routes returns a chi.Router with the filing routes mounted.
func (h *FilingHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(requireLoopbackBind(h.bindHost))
	r.Use(requireLocalHost)

	r.Get("/{kind}/{id}/isds/submissions", h.List)
	r.Get("/submissions/{sid}/receipt.zfo", h.Receipt)
	r.Group(func(w chi.Router) {
		w.Use(requireJSONContentType)
		w.Use(requireConfirmHeader)
		w.Post("/{kind}/{id}/isds/preview", h.Preview)
		w.Post("/{kind}/{id}/isds/send", h.Send)
		w.Post("/submissions/{sid}/refresh", h.Refresh)
		w.Post("/submissions/{sid}/resolve", h.Resolve)
	})
	return r
}

type filingPreviewRequest struct {
	Environment string `json:"environment"`
}

type filingSendRequest struct {
	Environment    string `json:"environment"`
	ConfirmToken   string `json:"confirmToken"`
	AcceptWarnings bool   `json:"acceptWarnings"`
}

type filingResolveRequest struct {
	Action string `json:"action"`
	DmID   string `json:"dmId"`
}

type filingPreviewResponse struct {
	Kind                   string               `json:"kind"`
	DocumentID             int64                `json:"documentId"`
	Environment            string               `json:"environment"`
	RecipientBoxID         string               `json:"recipientBoxId"`
	RecipientName          string               `json:"recipientName"`
	FileName               string               `json:"fileName"`
	Annotation             string               `json:"annotation"`
	SizeBytes              int                  `json:"sizeBytes"`
	SHA256                 string               `json:"sha256"`
	Findings               []epoFindingResponse `json:"findings"`
	RequiresAcceptWarnings bool                 `json:"requiresAcceptWarnings"`
	ConfirmToken           string               `json:"confirmToken"`
	ExpiresAt              string               `json:"expiresAt"`
}

// filingSubmissionResponse is one send attempt. It carries no failure text
// from ISDS and never any credentials.
type filingSubmissionResponse struct {
	ID              int64   `json:"id"`
	Kind            string  `json:"kind"`
	DocumentID      int64   `json:"documentId"`
	Channel         string  `json:"channel"`
	Environment     string  `json:"environment"`
	RecipientBoxID  string  `json:"recipientBoxId"`
	RecipientName   string  `json:"recipientName"`
	DmID            string  `json:"dmId"`
	Annotation      string  `json:"annotation"`
	FileName        string  `json:"fileName"`
	XMLSHA256       string  `json:"xmlSha256"`
	Status          string  `json:"status"`
	DmMessageStatus *int    `json:"dmMessageStatus"`
	SentAt          *string `json:"sentAt"`
	DeliveredAt     *string `json:"deliveredAt"`
	HasReceipt      bool    `json:"hasReceipt"`
	CreatedAt       string  `json:"createdAt"`
}

type filingRefreshResponse struct {
	Submission    filingSubmissionResponse `json:"submission"`
	DocumentFiled bool                     `json:"documentFiled"`
}

func timeString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

func filingSubmissionToResponse(s domain.FilingSubmission) filingSubmissionResponse {
	return filingSubmissionResponse{
		ID:              s.ID,
		Kind:            string(s.DocumentKind),
		DocumentID:      s.DocumentID,
		Channel:         string(s.Channel),
		Environment:     s.Environment,
		RecipientBoxID:  s.RecipientBoxID,
		RecipientName:   s.RecipientName,
		DmID:            s.DmID,
		Annotation:      s.DmAnnotation,
		FileName:        s.FileName,
		XMLSHA256:       s.XMLSHA256,
		Status:          string(s.Status),
		DmMessageStatus: s.DmMessageStatus,
		SentAt:          timeString(s.SentAt),
		DeliveredAt:     timeString(s.DeliveredAt),
		HasReceipt:      s.HasReceipt,
		CreatedAt:       s.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// decodeFilingBody reads a strict JSON body: unknown fields (for example a
// recipient box id) are an error, not silently ignored.
func decodeFilingBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxFilingBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		respondError(w, http.StatusBadRequest, msgISDSBadBody)
		return false
	}
	return true
}

// parseFilingTarget reads the kind and id of a document path.
func parseFilingTarget(w http.ResponseWriter, r *http.Request) (domain.FilingKind, int64, bool) {
	kind, err := service.ParseFilingKind(chi.URLParam(r, "kind"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Neznámý typ podání.")
		return "", 0, false
	}
	id, err := parseID(r)
	if err != nil || id <= 0 {
		respondError(w, http.StatusBadRequest, "Neplatné ID podání.")
		return "", 0, false
	}
	return kind, id, true
}

func parseSubmissionID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "sid"), 10, 64)
	if err != nil || id <= 0 {
		respondError(w, http.StatusBadRequest, "Neplatné ID odeslání.")
		return 0, false
	}
	return id, true
}

// Preview handles POST /{kind}/{id}/isds/preview.
func (h *FilingHandler) Preview(w http.ResponseWriter, r *http.Request) {
	kind, id, ok := parseFilingTarget(w, r)
	if !ok {
		return
	}
	var req filingPreviewRequest
	if !decodeFilingBody(w, r, &req) {
		return
	}

	p, err := h.svc.Preview(r.Context(), kind, id, req.Environment)
	if err != nil {
		respondFilingError(w, err, opPreview)
		return
	}
	respondJSON(w, http.StatusOK, filingPreviewResponse{
		Kind:                   string(p.Kind),
		DocumentID:             p.DocumentID,
		Environment:            p.Environment,
		RecipientBoxID:         p.RecipientBoxID,
		RecipientName:          p.RecipientName,
		FileName:               p.FileName,
		Annotation:             p.Annotation,
		SizeBytes:              p.SizeBytes,
		SHA256:                 p.SHA256,
		Findings:               epoFindingsFromDomain(p.Findings),
		RequiresAcceptWarnings: p.RequiresAcceptWarnings,
		ConfirmToken:           p.ConfirmToken,
		ExpiresAt:              p.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

// Send handles POST /{kind}/{id}/isds/send. The recipient is never read from
// the request.
func (h *FilingHandler) Send(w http.ResponseWriter, r *http.Request) {
	kind, id, ok := parseFilingTarget(w, r)
	if !ok {
		return
	}
	var req filingSendRequest
	if !decodeFilingBody(w, r, &req) {
		return
	}

	sub, err := h.svc.Send(r.Context(), kind, id, service.FilingSendRequest{
		Environment:    req.Environment,
		ConfirmToken:   req.ConfirmToken,
		AcceptWarnings: req.AcceptWarnings,
	})
	if err != nil {
		respondFilingError(w, err, opSend)
		return
	}
	respondJSON(w, http.StatusCreated, filingSubmissionToResponse(*sub))
}

// List handles GET /{kind}/{id}/isds/submissions.
func (h *FilingHandler) List(w http.ResponseWriter, r *http.Request) {
	kind, id, ok := parseFilingTarget(w, r)
	if !ok {
		return
	}
	subs, err := h.svc.List(r.Context(), kind, id)
	if err != nil {
		respondFilingError(w, err, opRead)
		return
	}
	out := make([]filingSubmissionResponse, 0, len(subs))
	for _, s := range subs {
		out = append(out, filingSubmissionToResponse(s))
	}
	respondJSON(w, http.StatusOK, out)
}

// Refresh handles POST /submissions/{sid}/refresh. It is the only place the
// delivery state is fetched, and only when the user asks for it.
func (h *FilingHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	sid, ok := parseSubmissionID(w, r)
	if !ok {
		return
	}
	res, err := h.svc.Refresh(r.Context(), sid)
	if err != nil {
		respondFilingError(w, err, opRefresh)
		return
	}
	respondJSON(w, http.StatusOK, filingRefreshResponse{
		Submission:    filingSubmissionToResponse(*res.Submission),
		DocumentFiled: res.DocumentFiled,
	})
}

// Resolve handles POST /submissions/{sid}/resolve: the user settles a
// submission whose outcome is unknown. ISDS is not contacted.
func (h *FilingHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	sid, ok := parseSubmissionID(w, r)
	if !ok {
		return
	}
	var req filingResolveRequest
	if !decodeFilingBody(w, r, &req) {
		return
	}
	sub, err := h.svc.Resolve(r.Context(), sid, service.FilingResolveRequest{Action: req.Action, DmID: req.DmID})
	if err != nil {
		respondFilingError(w, err, opResolve)
		return
	}
	respondJSON(w, http.StatusOK, filingSubmissionToResponse(*sub))
}

// Receipt handles GET /submissions/{sid}/receipt.zfo.
func (h *FilingHandler) Receipt(w http.ResponseWriter, r *http.Request) {
	sid, ok := parseSubmissionID(w, r)
	if !ok {
		return
	}
	zfo, name, err := h.svc.Receipt(r.Context(), sid)
	if err != nil {
		respondFilingError(w, err, opRead)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Length", strconv.Itoa(len(zfo)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(zfo)
}

// xsdFindings turns the schema messages of an invalid XML into findings.
func xsdFindings(messages string) []epoFindingResponse {
	var out []epoFindingResponse
	for _, line := range strings.Split(messages, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(out) == maxXSDFindings {
			break
		}
		out = append(out, epoFindingResponse{Type: epo.TypeStructure, Text: line})
	}
	return out
}

// respondFilingError maps every error of the filing flow to a status and a
// Czech message. Error text from ISDS, EPO, xmllint or the keyring is never
// forwarded, except the numeric ISDS status code.
func respondFilingError(w http.ResponseWriter, err error, op filingOp) {
	var rejected *epo.RejectedError
	var invalid *xsdcheck.InvalidError
	var statusErr *isds.StatusError
	var userMessage *domain.UserMessageError

	switch {
	case errors.As(err, &rejected):
		respondJSON(w, http.StatusUnprocessableEntity, epoErrorResponse{
			Error:    "EPO v podání našlo chyby, nic nebylo odesláno. Opravte podání a vygenerujte XML znovu.",
			Findings: epoFindingsFromDomain(rejected.Findings),
		})
	case errors.As(err, &invalid):
		respondJSON(w, http.StatusUnprocessableEntity, epoErrorResponse{
			Error:    "XML přehledu neodpovídá schématu ČSSZ, nic nebylo odesláno. Vygenerujte přehled znovu.",
			Findings: xsdFindings(invalid.Messages),
		})
	case errors.Is(err, xsdcheck.ErrXmllintMissing):
		respondError(w, http.StatusServiceUnavailable, "Kontrola XML proti schématu ČSSZ vyžaduje program xmllint (balík libxml2). Nainstalujte ho, nebo přehled podejte ručně. Nic nebylo odesláno.")
	case errors.Is(err, xsdcheck.ErrTimeout):
		respondError(w, http.StatusGatewayTimeout, "Kontrola XML proti schématu ČSSZ nedoběhla včas. Nic nebylo odesláno, zkuste to znovu.")
	case errors.Is(err, xsdcheck.ErrExec), errors.Is(err, xsdcheck.ErrInvalidConfig):
		slog.Error("XSD check could not run", "error", err)
		respondError(w, http.StatusInternalServerError, "Kontrolu XML proti schématu ČSSZ se nepodařilo provést. Nic nebylo odesláno.")

	case errors.Is(err, epo.ErrEPOTimeout):
		respondError(w, http.StatusGatewayTimeout, "EPO neodpovědělo včas, podání nelze ověřit. Nic nebylo odesláno, zkuste to znovu.")
	case errors.Is(err, epo.ErrEPOUnavailable), errors.Is(err, epo.ErrUnexpectedResponse):
		respondError(w, http.StatusBadGateway, "EPO je nedostupné, podání nelze ověřit. Nic nebylo odesláno, zkuste to později nebo podání podejte ručně.")
	case errors.Is(err, epo.ErrInvalidXML):
		respondError(w, http.StatusUnprocessableEntity, "XML podání nelze zkontrolovat v EPO. Vygenerujte ho znovu.")

	case errors.Is(err, domain.ErrConfirmTokenInvalid):
		respondError(w, http.StatusConflict, "Potvrzení vypršelo, bylo už použito, nebo se od náhledu změnilo podání, prostředí či příjemce. Zobrazte náhled znovu.")
	case errors.Is(err, domain.ErrWarningsNotAccepted):
		respondError(w, http.StatusUnprocessableEntity, "Kontrola podání vrátila upozornění. Odeslání vyžaduje jejich výslovné potvrzení.")
	case errors.Is(err, domain.ErrFilingAlreadySent):
		respondError(w, http.StatusConflict, msgFilingBlocked)
	case errors.Is(err, domain.ErrFilingAlreadyFiled):
		respondError(w, http.StatusConflict, "Podání je už označeno jako podané, nelze ho znovu připravit ani odeslat.")
	case errors.Is(err, domain.ErrSubmissionNotResolvable):
		respondError(w, http.StatusConflict, "Toto odeslání nečeká na potvrzení výsledku.")
	case errors.Is(err, domain.ErrFilingSendInProgress):
		respondError(w, http.StatusConflict, "Odesílání tohoto podání právě probíhá.")
	case errors.Is(err, domain.ErrFilingRecordFailed):
		slog.Error("filing sent but not recorded", "error", err)
		respondError(w, http.StatusInternalServerError, "Zpráva byla datovou schránkou přijata, ale záznam o odeslání se nepodařilo uložit. Neodesílejte podání znovu. Najděte zprávu mezi odeslanými v datové schránce a její číslo doplňte u odeslání (akce resolve), nebo podání podejte ručně přes EPO.")
	case errors.Is(err, domain.ErrXMLNotGenerated):
		respondError(w, http.StatusConflict, "Nejdřív vygenerujte XML.")
	case errors.Is(err, domain.ErrISDSCredentialsMissing):
		respondError(w, http.StatusConflict, "Pro vybrané prostředí nejsou uloženy přihlašovací údaje datové schránky. Uložte je v nastavení.")
	case errors.Is(err, domain.ErrKeyringLocked):
		respondError(w, http.StatusServiceUnavailable, msgISDSKeyringLocked)
	case errors.Is(err, domain.ErrKeyringUnavailable):
		respondError(w, http.StatusServiceUnavailable, msgISDSKeyring)
	case errors.Is(err, domain.ErrMissingSetting):
		if errors.As(err, &userMessage) {
			respondError(w, http.StatusConflict, userMessage.Message)
			return
		}
		respondError(w, http.StatusConflict, "Chybí nastavení potřebné pro odeslání, například testovací schránka příjemce v konfiguraci.")

	case errors.Is(err, isds.ErrUnauthorized):
		respondError(w, http.StatusUnprocessableEntity, "Datová schránka odmítla přihlašovací údaje. Zkontrolujte je v nastavení a otestujte přihlášení.")
	case errors.As(err, &statusErr):
		respondError(w, http.StatusUnprocessableEntity, fmt.Sprintf("Datová schránka požadavek odmítla (kód %s).", statusErr.Code))
	case errors.Is(err, isds.ErrSOAPFault) && op == opSend:
		respondError(w, http.StatusBadGateway, "Datová schránka zprávu odmítla, nic nebylo odesláno. Podání lze po chvíli odeslat znovu.")
	case errors.Is(err, isds.ErrISDSTimeout):
		respondError(w, http.StatusGatewayTimeout, unavailableMessage(op, "Datová schránka neodpověděla včas."))
	case errors.Is(err, isds.ErrISDSUnavailable), errors.Is(err, isds.ErrUnexpectedResponse),
		errors.Is(err, isds.ErrResponseTooLarge), errors.Is(err, isds.ErrSOAPFault):
		respondError(w, http.StatusBadGateway, unavailableMessage(op, "Datová schránka je nedostupná nebo vrátila nečekanou odpověď."))
	case errors.Is(err, isds.ErrInvalidInput):
		respondError(w, http.StatusUnprocessableEntity, "Zpráva nesplňuje limity datových schránek, nic nebylo odesláno.")
	case errors.Is(err, isds.ErrInvalidConfig):
		slog.Error("ISDS client misconfigured", "error", err)
		respondError(w, http.StatusInternalServerError, "Klient datové schránky je špatně nastaven.")

	case errors.Is(err, domain.ErrNotFound):
		respondError(w, http.StatusNotFound, "Podání nebylo nalezeno.")
	case errors.As(err, &userMessage):
		respondError(w, http.StatusBadRequest, userMessage.Message)
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusBadRequest, "Neplatný požadavek.")
	default:
		slog.Error("filing request failed", "operation", string(op), "error", err)
		respondError(w, http.StatusInternalServerError, "Při zpracování podání došlo k vnitřní chybě.")
	}
}

// unavailableMessage completes a transport failure with what it means for the
// action: a failed send may still have left.
func unavailableMessage(op filingOp, lead string) string {
	switch op {
	case opSend:
		return lead + " Odeslání se nepodařilo potvrdit, zpráva mohla odejít. " + msgFilingBlockedHint
	case opRefresh:
		return lead + " Stav doručení se nepodařilo zjistit, zkuste to později."
	case opLoginTest:
		return lead + " Přihlášení se nepodařilo otestovat."
	default:
		return lead
	}
}
