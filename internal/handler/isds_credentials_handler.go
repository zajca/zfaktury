package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/secrets"
	"github.com/zajca/zfaktury/internal/service"
)

const (
	// isdsCredentialsPath is the mount path of the credentials API. The dev
	// CORS middleware skips it so no foreign origin can reach the keyring.
	isdsCredentialsPath = "/api/v1/settings/isds-credentials"

	// isdsConfirmHeader must accompany every state-changing request. A custom
	// header forces a CORS preflight, which the production router never allows.
	isdsConfirmHeader = "X-ZFaktury-Confirm"

	// maxISDSCredentialsBody bounds the PUT body; the longest valid login is
	// well below it.
	maxISDSCredentialsBody = 4 << 10
)

const (
	msgISDSNotLoopback   = "Přihlašovací údaje datové schránky lze spravovat jen tehdy, když aplikace naslouchá na místní adrese (127.0.0.1)."
	msgISDSBadHost       = "Požadavek na přihlašovací údaje datové schránky musí mířit na místní adresu aplikace."
	msgISDSNoConfirm     = "Požadavek nemá potvrzovací hlavičku " + isdsConfirmHeader + "."
	msgISDSNotJSON       = "Požadavek musí mít Content-Type application/json."
	msgISDSBadBody       = "Neplatné tělo požadavku."
	msgISDSKeyringLocked = "Systémová klíčenka zůstala zamčená, odemknutí se nezdařilo nebo bylo zrušeno. Odemkněte klíčenku a zkuste to znovu. Údaje se nikdy neukládají mimo klíčenku."
	msgISDSKeyring       = "Systémová klíčenka není dostupná, proto se přihlašovací údaje nedají uložit ani načíst. Spusťte službu klíčenky (například GNOME Keyring nebo KeePassXC s podporou Secret Service) a zkuste to znovu. Údaje se nikdy neukládají mimo klíčenku."
)

// ISDSCredentialsHandler manages the data-box login stored in the OS keyring.
type ISDSCredentialsHandler struct {
	svc *service.ISDSCredentialsService
	// bindHost is the address the HTTP server listens on. The credentials API
	// refuses to work unless it is a loopback address.
	bindHost string
}

// NewISDSCredentialsHandler creates a new ISDSCredentialsHandler.
func NewISDSCredentialsHandler(svc *service.ISDSCredentialsService, bindHost string) *ISDSCredentialsHandler {
	return &ISDSCredentialsHandler{svc: svc, bindHost: bindHost}
}

// Routes returns a chi.Router with the credentials routes mounted.
func (h *ISDSCredentialsHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(h.requireLoopbackBind)
	r.Use(requireLocalHost)
	r.Get("/", h.Get)
	r.Group(func(w chi.Router) {
		w.Use(requireJSONContentType)
		w.Use(requireConfirmHeader)
		w.Put("/", h.Put)
		w.Delete("/", h.Delete)
	})
	return r
}

// isdsStatusResponse is the per-environment state. It deliberately has no
// password field.
type isdsStatusResponse struct {
	Configured bool   `json:"configured"`
	Username   string `json:"username"`
}

type isdsSaveRequest struct {
	Environment string `json:"environment"`
	Username    string `json:"username"`
	Password    string `json:"password"`
}

// Get handles GET /api/v1/settings/isds-credentials.
func (h *ISDSCredentialsHandler) Get(w http.ResponseWriter, r *http.Request) {
	statuses, err := h.svc.Status()
	if err != nil {
		h.respondError(w, err)
		return
	}
	resp := make(map[string]isdsStatusResponse, len(statuses))
	for env, status := range statuses {
		resp[string(env)] = isdsStatusResponse{Configured: status.Configured, Username: status.Username}
	}
	respondJSON(w, http.StatusOK, resp)
}

// Put handles PUT /api/v1/settings/isds-credentials.
func (h *ISDSCredentialsHandler) Put(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxISDSCredentialsBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req isdsSaveRequest
	if err := dec.Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, msgISDSBadBody)
		return
	}

	creds := secrets.ISDSCredentials{Username: req.Username, Password: req.Password}.Normalize()
	if err := h.svc.Save(req.Environment, creds); err != nil {
		h.respondError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, isdsStatusResponse{Configured: true, Username: creds.Username})
}

// Delete handles DELETE /api/v1/settings/isds-credentials?environment=.
func (h *ISDSCredentialsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.URL.Query().Get("environment")); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// respondError maps service errors to responses. Error text from the keyring
// layer is never forwarded.
func (h *ISDSCredentialsHandler) respondError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrKeyringLocked) {
		respondError(w, http.StatusServiceUnavailable, msgISDSKeyringLocked)
		return
	}
	if errors.Is(err, domain.ErrKeyringUnavailable) {
		respondError(w, http.StatusServiceUnavailable, msgISDSKeyring)
		return
	}
	var userMessage *domain.UserMessageError
	if errors.As(err, &userMessage) {
		respondError(w, http.StatusBadRequest, userMessage.Message)
		return
	}
	slog.Error("ISDS credentials request failed", "error", err)
	respondError(w, http.StatusInternalServerError, "internal server error")
}

// requireLoopbackBind refuses every request unless the server listens on a
// loopback address. It checks the configured listen address, never the
// request's RemoteAddr, which headers can influence.
func (h *ISDSCredentialsHandler) requireLoopbackBind(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackHost(h.bindHost) {
			respondError(w, http.StatusForbidden, msgISDSNotLoopback)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireLocalHost rejects requests whose Host header names anything but the
// local machine, which defeats DNS rebinding against the unauthenticated API.
func requireLocalHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLocalRequestHost(r.Host) {
			respondError(w, http.StatusForbidden, msgISDSBadHost)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireJSONContentType rejects bodies that are not declared as JSON, so a
// cross-site HTML form post cannot reach the handler.
func requireJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			respondError(w, http.StatusUnsupportedMediaType, msgISDSNotJSON)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireConfirmHeader rejects state-changing requests without the confirm header.
func requireConfirmHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(isdsConfirmHeader) != "1" {
			respondError(w, http.StatusForbidden, msgISDSNoConfirm)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isLoopbackHost reports whether host is a loopback IP or "localhost".
func isLoopbackHost(host string) bool {
	host = strings.Trim(strings.ToLower(host), "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// isLocalRequestHost reports whether a Host header (with optional port) names
// the local machine: a loopback address, localhost, or the Wails WebView host.
func isLocalRequestHost(hostHeader string) bool {
	host := hostHeader
	if h, _, err := net.SplitHostPort(hostHeader); err == nil {
		host = h
	}
	host = strings.Trim(strings.ToLower(host), "[]")
	return host == "wails.localhost" || isLoopbackHost(host)
}
