package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zalando/go-keyring"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/epo"
	"github.com/zajca/zfaktury/internal/isds"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/secrets"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/testutil"
	"github.com/zajca/zfaktury/internal/xsdcheck"
)

const filingTestXML = "<Pisemnost><DPHDP3>fake-body-1</DPHDP3></Pisemnost>"

type fakeFilingDoc struct {
	mu       sync.Mutex
	xml      []byte
	markedAt []time.Time
}

func (d *fakeFilingDoc) RegenerateXML(context.Context, int64) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.xml, nil
}

func (d *fakeFilingDoc) MarkFiledAt(_ context.Context, _ int64, at time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.markedAt = append(d.markedAt, at)
	return nil
}

func (d *fakeFilingDoc) setXML(x string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.xml = []byte(x)
}

type fakeFilingEPO struct {
	findings []epo.Finding
	err      error
}

func (f *fakeFilingEPO) Check(context.Context, []byte) ([]epo.Finding, error) {
	return f.findings, f.err
}

type fakeFilingXSD struct{ err error }

func (f *fakeFilingXSD) Check(context.Context, xsdcheck.Schema, []byte) error { return f.err }

type fakeFilingSettings struct{}

func (fakeFilingSettings) Get(_ context.Context, key string) (string, error) {
	if key == service.SettingUFOCode {
		return "451", nil
	}
	return "", sql.ErrNoRows
}

type fakeISDSClient struct {
	mu        sync.Mutex
	created   []isds.MessageInput
	createErr error
	delivery  *isds.DeliveryInfo
	zfo       []byte
	owner     *isds.OwnerInfo
	user      *isds.UserInfo
	ownerErr  error
}

func (c *fakeISDSClient) CreateMessage(_ context.Context, m isds.MessageInput) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.created = append(c.created, m)
	if c.createErr != nil {
		return "", c.createErr
	}
	return "1234567890", nil
}

func (c *fakeISDSClient) GetDeliveryInfo(context.Context, string) (*isds.DeliveryInfo, error) {
	return c.delivery, nil
}

func (c *fakeISDSClient) SignedSentMessageDownload(context.Context, string) ([]byte, error) {
	return c.zfo, nil
}

func (c *fakeISDSClient) GetOwnerInfo(context.Context) (*isds.OwnerInfo, error) {
	return c.owner, c.ownerErr
}

func (c *fakeISDSClient) GetUserInfo(context.Context) (*isds.UserInfo, error) { return c.user, nil }

type filingFixture struct {
	router *chi.Mux
	doc    *fakeFilingDoc
	epo    *fakeFilingEPO
	xsd    *fakeFilingXSD
	client *fakeISDSClient
	clock  *time.Time
	logs   *bytes.Buffer
	// responses collects every response body of the test, for the secret scan.
	responses []string
}

func newFilingFixture(t *testing.T, devMode bool, bindHost string) *filingFixture {
	t.Helper()
	keyring.MockInit()
	t.Cleanup(keyring.MockInit)
	store := secrets.NewISDSStore()
	for _, env := range secrets.ISDSEnvironments {
		if err := store.Save(env, secrets.ISDSCredentials{Username: testISDSUsername, Password: testISDSPassword}); err != nil {
			t.Fatal(err)
		}
	}
	logs := captureLogs(t)

	prod, err := isds.NewRecipients(isds.RecipientEnvProduction, isds.TestRecipientBoxes{})
	if err != nil {
		t.Fatal(err)
	}
	test, err := isds.NewRecipients(isds.RecipientEnvTest, isds.TestRecipientBoxes{CSSZ: "tst1234", TaxOffices: map[string]string{"451": "tst4511"}})
	if err != nil {
		t.Fatal(err)
	}

	clock := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	f := &filingFixture{
		doc:    &fakeFilingDoc{xml: []byte(filingTestXML)},
		epo:    &fakeFilingEPO{},
		xsd:    &fakeFilingXSD{},
		client: &fakeISDSClient{},
		clock:  &clock,
		logs:   logs,
	}
	svc, err := service.NewFilingService(service.FilingServiceConfig{
		Documents: map[domain.FilingKind]service.FilingDocument{
			domain.FilingKindVATReturn:       f.doc,
			domain.FilingKindSocialInsurance: f.doc,
		},
		Repo: repository.NewFilingSubmissionRepository(testutil.NewTestDB(t)),
		Recipients: map[isds.RecipientEnvironment]*isds.Recipients{
			isds.RecipientEnvProduction: prod,
			isds.RecipientEnvTest:       test,
		},
		Settings:    fakeFilingSettings{},
		EPO:         f.epo,
		XSD:         f.xsd,
		Credentials: store,
		NewClient: func(isds.Environment, isds.Credentials) (service.FilingISDSClient, error) {
			return f.client, nil
		},
		Now: func() time.Time { return clock },
	})
	if err != nil {
		t.Fatal(err)
	}
	f.router = NewRouter(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		service.NewISDSCredentialsService(store), svc,
		RouterConfig{DevMode: devMode, DataDir: t.TempDir(), BindHost: bindHost},
	)
	return f
}

// do runs q against the router and records the body for the secret scan.
func (f *filingFixture) do(q isdsReq) *httptest.ResponseRecorder {
	w := q.do(f.router)
	f.responses = append(f.responses, w.Body.String())
	return w
}

func filingPost(path string, body any) isdsReq {
	b, _ := json.Marshal(body)
	return isdsReq{method: http.MethodPost, path: path, body: string(b), contentType: "application/json", confirm: "1"}
}

const (
	vatPreviewPath = "/api/v1/filings/vat-return/7/isds/preview"
	vatSendPath    = "/api/v1/filings/vat-return/7/isds/send"
	vatListPath    = "/api/v1/filings/vat-return/7/isds/submissions"
)

func (f *filingFixture) preview(t *testing.T, env string) filingPreviewResponse {
	t.Helper()
	w := f.do(filingPost(vatPreviewPath, map[string]any{"environment": env}))
	if w.Code != http.StatusOK {
		t.Fatalf("preview status = %d, body: %s", w.Code, w.Body.String())
	}
	var p filingPreviewResponse
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func (f *filingFixture) send(token, env string, accept bool) *httptest.ResponseRecorder {
	return f.do(filingPost(vatSendPath, map[string]any{"environment": env, "confirmToken": token, "acceptWarnings": accept}))
}

func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("decoding %q: %v", w.Body.String(), err)
	}
	return v
}

func errorOf(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	return decode[map[string]any](t, w)["error"].(string)
}

func TestFilingHandler_PreviewAndSend(t *testing.T) {
	f := newFilingFixture(t, false, "127.0.0.1")
	p := f.preview(t, "production")
	if p.RecipientBoxID != "7nyn2d9" || !strings.Contains(p.RecipientName, "Prahu") || p.Environment != "production" ||
		p.FileName != "DPHDP3.xml" || p.SizeBytes != len(filingTestXML) || len(p.SHA256) != 64 || p.ConfirmToken == "" ||
		p.RequiresAcceptWarnings || p.Findings == nil || p.ExpiresAt == "" {
		t.Fatalf("preview = %+v", p)
	}
	if len(f.client.created) != 0 {
		t.Fatal("a preview sent a message")
	}

	w := f.send(p.ConfirmToken, "production", false)
	if w.Code != http.StatusCreated {
		t.Fatalf("send status = %d, body: %s", w.Code, w.Body.String())
	}
	sub := decode[filingSubmissionResponse](t, w)
	if sub.Status != "sent" || sub.DmID != "1234567890" || sub.RecipientBoxID != "7nyn2d9" || sub.XMLSHA256 != p.SHA256 ||
		sub.Kind != "vat-return" || sub.DocumentID != 7 || sub.SentAt == nil || sub.Environment != "production" {
		t.Errorf("submission = %+v", sub)
	}
	if len(f.client.created) != 1 || f.client.created[0].RecipientBoxID != "7nyn2d9" {
		t.Errorf("messages = %+v", f.client.created)
	}

	list := decode[[]filingSubmissionResponse](t, f.do(isdsReq{method: http.MethodGet, path: vatListPath}))
	if len(list) != 1 || list[0].ID != sub.ID {
		t.Errorf("list = %+v", list)
	}
}

func TestFilingHandler_TokenReuseExpiryAndDoubleSend(t *testing.T) {
	f := newFilingFixture(t, false, "127.0.0.1")
	p := f.preview(t, "production")
	if w := f.send(p.ConfirmToken, "production", false); w.Code != http.StatusCreated {
		t.Fatalf("first send = %d, %s", w.Code, w.Body.String())
	}

	// The same token again.
	w := f.send(p.ConfirmToken, "production", false)
	if w.Code != http.StatusConflict || !strings.Contains(errorOf(t, w), "Zobrazte náhled znovu") {
		t.Errorf("token reuse = %d, %s", w.Code, w.Body.String())
	}

	// A fresh preview of the same bytes: the document was already sent.
	p2 := f.preview(t, "production")
	w = f.send(p2.ConfirmToken, "production", false)
	if w.Code != http.StatusConflict || !strings.Contains(errorOf(t, w), "už bylo") {
		t.Errorf("double send = %d, %s", w.Code, w.Body.String())
	}
	if len(f.client.created) != 1 {
		t.Errorf("messages sent = %d, want 1", len(f.client.created))
	}

	// An expired token.
	p3 := f.preview(t, "test")
	*f.clock = f.clock.Add(service.DefaultFilingConfirmTTL + time.Second)
	w = f.send(p3.ConfirmToken, "test", false)
	if w.Code != http.StatusConflict || len(f.client.created) != 1 {
		t.Errorf("expired token = %d, messages %d", w.Code, len(f.client.created))
	}
}

func TestFilingHandler_ChangedXMLIsRefused(t *testing.T) {
	f := newFilingFixture(t, false, "127.0.0.1")
	p := f.preview(t, "production")
	f.doc.setXML(filingTestXML + "<!-- edited after the preview -->")

	w := f.send(p.ConfirmToken, "production", false)
	if w.Code != http.StatusConflict || len(f.client.created) != 0 {
		t.Errorf("send of changed XML = %d (messages %d), body: %s", w.Code, len(f.client.created), w.Body.String())
	}
}

func TestFilingHandler_RecipientIsNeverTakenFromTheBody(t *testing.T) {
	f := newFilingFixture(t, false, "127.0.0.1")
	p := f.preview(t, "production")

	for _, field := range []string{"recipientBoxId", "boxId", "dbIDRecipient", "recipient"} {
		w := f.do(filingPost(vatSendPath, map[string]any{
			"environment": "production", "confirmToken": p.ConfirmToken, field: "evil123",
		}))
		if w.Code != http.StatusBadRequest {
			t.Errorf("send with %q in the body = %d, want 400", field, w.Code)
		}
		w = f.do(filingPost(vatPreviewPath, map[string]any{"environment": "production", field: "evil123"}))
		if w.Code != http.StatusBadRequest {
			t.Errorf("preview with %q in the body = %d, want 400", field, w.Code)
		}
	}
	if len(f.client.created) != 0 {
		t.Fatal("a request with a recipient in its body sent a message")
	}

	// The refused requests did not spend the token, and the legitimate send
	// goes to the table recipient.
	if w := f.send(p.ConfirmToken, "production", false); w.Code != http.StatusCreated {
		t.Fatalf("send = %d, %s", w.Code, w.Body.String())
	}
	if got := f.client.created[0].RecipientBoxID; got != "7nyn2d9" {
		t.Errorf("recipient = %q, want the box from the table", got)
	}
}

func TestFilingHandler_FindingsFlow(t *testing.T) {
	t.Run("blocking findings", func(t *testing.T) {
		f := newFilingFixture(t, false, "127.0.0.1")
		f.epo.findings = []epo.Finding{{Type: epo.TypeCritical, Code: "FAKE_K", Field: "dic", Text: "fake critical"}}
		w := f.do(filingPost(vatPreviewPath, map[string]any{"environment": "production"}))
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
		}
		body := decode[epoErrorResponse](t, w)
		if len(body.Findings) != 1 || body.Findings[0].Code != "FAKE_K" || strings.Contains(w.Body.String(), "confirmToken") {
			t.Errorf("body = %s", w.Body.String())
		}
	})
	t.Run("warnings need acceptance", func(t *testing.T) {
		f := newFilingFixture(t, false, "127.0.0.1")
		f.epo.findings = []epo.Finding{{Type: epo.TypePassable, Code: "FAKE_P", Field: "d_poddp", Text: "fake warning"}}
		p := f.preview(t, "production")
		if !p.RequiresAcceptWarnings || len(p.Findings) != 1 || p.Findings[0].Code != "FAKE_P" {
			t.Fatalf("preview = %+v", p)
		}
		if w := f.send(p.ConfirmToken, "production", false); w.Code != http.StatusUnprocessableEntity || len(f.client.created) != 0 {
			t.Errorf("send without acceptance = %d (messages %d)", w.Code, len(f.client.created))
		}
		if w := f.send(p.ConfirmToken, "production", true); w.Code != http.StatusCreated {
			t.Errorf("send with acceptance = %d, %s", w.Code, w.Body.String())
		}
	})
	t.Run("EPO unreachable refuses", func(t *testing.T) {
		for err, want := range map[error]int{epo.ErrEPOUnavailable: http.StatusBadGateway, epo.ErrEPOTimeout: http.StatusGatewayTimeout} {
			f := newFilingFixture(t, false, "127.0.0.1")
			f.epo.err = fmt.Errorf("checking: %w", err)
			w := f.do(filingPost(vatPreviewPath, map[string]any{"environment": "production"}))
			if w.Code != want || !strings.Contains(errorOf(t, w), "Nic nebylo odesláno") {
				t.Errorf("%v: status %d, body %s", err, w.Code, w.Body.String())
			}
		}
	})
	t.Run("social insurance needs xmllint", func(t *testing.T) {
		f := newFilingFixture(t, false, "127.0.0.1")
		f.xsd.err = xsdcheck.ErrXmllintMissing
		w := f.do(filingPost("/api/v1/filings/social-insurance/3/isds/preview", map[string]any{"environment": "production"}))
		if w.Code != http.StatusServiceUnavailable || !strings.Contains(errorOf(t, w), "libxml2") {
			t.Errorf("status %d, body %s", w.Code, w.Body.String())
		}
	})
	t.Run("social insurance schema findings", func(t *testing.T) {
		f := newFilingFixture(t, false, "127.0.0.1")
		f.xsd.err = &xsdcheck.InvalidError{Messages: "element a: not expected\n\nelement b: missing"}
		w := f.do(filingPost("/api/v1/filings/social-insurance/3/isds/preview", map[string]any{"environment": "production"}))
		body := decode[epoErrorResponse](t, w)
		if w.Code != http.StatusUnprocessableEntity || len(body.Findings) != 2 {
			t.Errorf("status %d, body %s", w.Code, w.Body.String())
		}
	})
	t.Run("social insurance goes to CSSZ", func(t *testing.T) {
		f := newFilingFixture(t, false, "127.0.0.1")
		w := f.do(filingPost("/api/v1/filings/social-insurance/3/isds/preview", map[string]any{"environment": "production"}))
		p := decode[filingPreviewResponse](t, w)
		if w.Code != http.StatusOK || p.RecipientBoxID != isds.CSSZEpodaniBoxID || p.FileName != "OSVC.xml" {
			t.Errorf("status %d, preview %+v", w.Code, p)
		}
	})
}

func TestFilingHandler_RefreshAndReceipt(t *testing.T) {
	f := newFilingFixture(t, false, "127.0.0.1")
	p := f.preview(t, "production")
	sub := decode[filingSubmissionResponse](t, f.send(p.ConfirmToken, "production", false))

	receiptPath := fmt.Sprintf("/api/v1/filings/submissions/%d/receipt.zfo", sub.ID)
	if w := f.do(isdsReq{method: http.MethodGet, path: receiptPath}); w.Code != http.StatusNotFound {
		t.Errorf("receipt before delivery = %d", w.Code)
	}

	deliveredAt := time.Date(2026, 11, 24, 12, 7, 0, 0, time.UTC)
	f.client.delivery = &isds.DeliveryInfo{MessageID: "1234567890", Status: isds.StatusDelivered, DeliveryTime: &deliveredAt}
	f.client.zfo = []byte{0x00, 0xff, 'Z', 'F', 'O'}

	w := f.do(filingPost(fmt.Sprintf("/api/v1/filings/submissions/%d/refresh", sub.ID), map[string]any{}))
	if w.Code != http.StatusOK {
		t.Fatalf("refresh = %d, %s", w.Code, w.Body.String())
	}
	res := decode[filingRefreshResponse](t, w)
	if res.Submission.Status != "delivered" || !res.DocumentFiled || !res.Submission.HasReceipt ||
		res.Submission.DeliveredAt == nil || *res.Submission.DeliveredAt != "2026-11-24T12:07:00Z" {
		t.Errorf("refresh = %+v", res)
	}
	if len(f.doc.markedAt) != 1 || !f.doc.markedAt[0].Equal(deliveredAt) {
		t.Errorf("document marked filed at %v, want %v", f.doc.markedAt, deliveredAt)
	}

	w = f.do(isdsReq{method: http.MethodGet, path: receiptPath})
	if w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), f.client.zfo) {
		t.Fatalf("receipt = %d, %v", w.Code, w.Body.Bytes())
	}
	if w.Header().Get("Content-Type") != "application/octet-stream" ||
		w.Header().Get("Content-Disposition") != `attachment; filename=DZ_1234567890.zfo` ||
		w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("receipt headers = %v", w.Header())
	}
}

func TestFilingHandler_SendFailureMapping(t *testing.T) {
	tests := map[string]struct {
		err    error
		status int
		text   string
	}{
		"ISDS refused":  {&isds.StatusError{Code: "1234", Message: "internal ISDS text"}, http.StatusUnprocessableEntity, "kód 1234"},
		"bad login":     {fmt.Errorf("x: %w", isds.ErrUnauthorized), http.StatusUnprocessableEntity, "odmítla přihlašovací údaje"},
		"timeout":       {fmt.Errorf("x: %w: %w", isds.ErrISDSTimeout, isds.ErrISDSUnavailable), http.StatusGatewayTimeout, "zpráva mohla odejít"},
		"unavailable":   {fmt.Errorf("x: %w", isds.ErrISDSUnavailable), http.StatusBadGateway, "zpráva mohla odejít"},
		"unexpected":    {fmt.Errorf("x: %w", isds.ErrUnexpectedResponse), http.StatusBadGateway, "zpráva mohla odejít"},
		"fault":         {&isds.FaultError{Code: "S:Server", Message: "internal fault text"}, http.StatusBadGateway, "nečekanou odpověď"},
		"limit":         {fmt.Errorf("x: %w", isds.ErrInvalidInput), http.StatusUnprocessableEntity, "limity"},
		"response size": {fmt.Errorf("x: %w", isds.ErrResponseTooLarge), http.StatusBadGateway, "zpráva mohla odejít"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			f := newFilingFixture(t, false, "127.0.0.1")
			f.client.createErr = tc.err
			p := f.preview(t, "production")
			w := f.send(p.ConfirmToken, "production", false)
			if w.Code != tc.status || !strings.Contains(errorOf(t, w), tc.text) {
				t.Errorf("status %d body %s, want %d containing %q", w.Code, w.Body.String(), tc.status, tc.text)
			}
			for _, internal := range []string{"internal ISDS text", "internal fault text", "https://", "/DS/"} {
				if strings.Contains(w.Body.String(), internal) {
					t.Errorf("response leaks %q: %s", internal, w.Body.String())
				}
			}
		})
	}
}

func TestFilingHandler_ErrorMapping(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{domain.ErrNotFound, http.StatusNotFound},
		{domain.ErrXMLNotGenerated, http.StatusConflict},
		{domain.ErrISDSCredentialsMissing, http.StatusConflict},
		{domain.ErrFilingSendInProgress, http.StatusConflict},
		{domain.ErrFilingAlreadySent, http.StatusConflict},
		{domain.ErrConfirmTokenInvalid, http.StatusConflict},
		{domain.ErrWarningsNotAccepted, http.StatusUnprocessableEntity},
		{domain.ErrMissingSetting, http.StatusConflict},
		{domain.ErrKeyringLocked, http.StatusServiceUnavailable},
		{domain.ErrKeyringUnavailable, http.StatusServiceUnavailable},
		{domain.ErrFilingRecordFailed, http.StatusInternalServerError},
		{domain.ErrInvalidInput, http.StatusBadRequest},
		{&domain.UserMessageError{Message: "Zpráva pro uživatele."}, http.StatusBadRequest},
		{epo.ErrEPOTimeout, http.StatusGatewayTimeout},
		{epo.ErrEPOUnavailable, http.StatusBadGateway},
		{epo.ErrUnexpectedResponse, http.StatusBadGateway},
		{epo.ErrInvalidXML, http.StatusUnprocessableEntity},
		{&epo.RejectedError{Findings: []epo.Finding{{Type: "K"}}}, http.StatusUnprocessableEntity},
		{xsdcheck.ErrXmllintMissing, http.StatusServiceUnavailable},
		{xsdcheck.ErrTimeout, http.StatusGatewayTimeout},
		{xsdcheck.ErrExec, http.StatusInternalServerError},
		{&xsdcheck.InvalidError{Messages: "m"}, http.StatusUnprocessableEntity},
		{isds.ErrUnauthorized, http.StatusUnprocessableEntity},
		{&isds.StatusError{Code: "9999"}, http.StatusUnprocessableEntity},
		{isds.ErrISDSTimeout, http.StatusGatewayTimeout},
		{isds.ErrISDSUnavailable, http.StatusBadGateway},
		{isds.ErrSOAPFault, http.StatusBadGateway},
		{isds.ErrUnexpectedResponse, http.StatusBadGateway},
		{isds.ErrResponseTooLarge, http.StatusBadGateway},
		{isds.ErrInvalidInput, http.StatusUnprocessableEntity},
		{isds.ErrInvalidConfig, http.StatusInternalServerError},
		{errors.New("some /home/user/secret/path failure"), http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%T_%v", tc.err, tc.err), func(t *testing.T) {
			w := httptest.NewRecorder()
			respondFilingError(w, fmt.Errorf("wrapped: %w", tc.err), opSend)
			if w.Code != tc.status {
				t.Errorf("status = %d, want %d (body %s)", w.Code, tc.status, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "/home/") || strings.Contains(w.Body.String(), "wrapped") {
				t.Errorf("body leaks internal text: %s", w.Body.String())
			}
		})
	}
}

func TestFilingHandler_GuardsOnEveryRoute(t *testing.T) {
	routes := []struct {
		method string
		path   string
		body   string
		post   bool
	}{
		{http.MethodPost, vatPreviewPath, `{"environment":"test"}`, true},
		{http.MethodPost, vatSendPath, `{"environment":"test","confirmToken":"x"}`, true},
		{http.MethodPost, "/api/v1/filings/submissions/1/refresh", `{}`, true},
		{http.MethodGet, vatListPath, "", false},
		{http.MethodGet, "/api/v1/filings/submissions/1/receipt.zfo", "", false},
		{http.MethodPost, "/api/v1/settings/isds-credentials/test", `{"environment":"test"}`, true},
	}
	for _, rt := range routes {
		name := rt.method + " " + rt.path
		t.Run(name, func(t *testing.T) {
			f := newFilingFixture(t, false, "127.0.0.1")
			good := isdsReq{method: rt.method, path: rt.path, body: rt.body}
			if rt.post {
				good.contentType, good.confirm = "application/json", "1"
			}
			if w := good.do(f.router); w.Code == http.StatusForbidden || w.Code == http.StatusUnsupportedMediaType {
				t.Fatalf("a well-formed request was refused by a guard: %d %s", w.Code, w.Body.String())
			}

			bad := good
			bad.host = "evil.example.com"
			if w := bad.do(f.router); w.Code != http.StatusForbidden {
				t.Errorf("foreign Host = %d, want 403", w.Code)
			}
			bad = good
			bad.host = "127.0.0.1.evil.example.com:8080"
			if w := bad.do(f.router); w.Code != http.StatusForbidden {
				t.Errorf("rebinding-style Host = %d, want 403", w.Code)
			}

			if rt.post {
				bad = good
				bad.contentType = "text/plain"
				if w := bad.do(f.router); w.Code != http.StatusUnsupportedMediaType {
					t.Errorf("wrong Content-Type = %d, want 415", w.Code)
				}
				bad = good
				bad.contentType = "application/x-www-form-urlencoded"
				if w := bad.do(f.router); w.Code != http.StatusUnsupportedMediaType {
					t.Errorf("form Content-Type = %d, want 415", w.Code)
				}
				bad = good
				bad.confirm = ""
				if w := bad.do(f.router); w.Code != http.StatusForbidden {
					t.Errorf("missing confirm header = %d, want 403", w.Code)
				}
				bad = good
				bad.confirm = "0"
				if w := bad.do(f.router); w.Code != http.StatusForbidden {
					t.Errorf("wrong confirm header = %d, want 403", w.Code)
				}
			}

			lan := newFilingFixture(t, false, "0.0.0.0")
			if w := good.do(lan.router); w.Code != http.StatusForbidden {
				t.Errorf("non-loopback bind = %d, want 403", w.Code)
			}
			if len(lan.client.created) != 0 {
				t.Error("a request on a non-loopback bind reached ISDS")
			}
		})
	}
}

func TestFilingHandler_DevCORSDoesNotApply(t *testing.T) {
	f := newFilingFixture(t, true, "127.0.0.1")
	for _, path := range []string{vatSendPath, vatPreviewPath, vatListPath, "/api/v1/settings/isds-credentials/test"} {
		w := isdsReq{method: http.MethodOptions, path: path, origin: "https://evil.example.com"}.do(f.router)
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("OPTIONS %s got a CORS allow-origin header: %q", path, w.Header().Get("Access-Control-Allow-Origin"))
		}
		if w.Code == http.StatusNoContent {
			t.Errorf("OPTIONS %s was answered as a CORS preflight", path)
		}
		q := filingPost(path, map[string]any{"environment": "test"})
		q.origin = "https://evil.example.com"
		w = q.do(f.router)
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("POST %s got a CORS allow-origin header", path)
		}
	}
}

func TestFilingHandler_BadInput(t *testing.T) {
	f := newFilingFixture(t, false, "127.0.0.1")
	for name, q := range map[string]isdsReq{
		"unknown kind":      filingPost("/api/v1/filings/health-insurance/7/isds/preview", map[string]any{"environment": "test"}),
		"bad id":            filingPost("/api/v1/filings/vat-return/abc/isds/preview", map[string]any{"environment": "test"}),
		"zero id":           filingPost("/api/v1/filings/vat-return/0/isds/preview", map[string]any{"environment": "test"}),
		"missing env":       filingPost(vatPreviewPath, map[string]any{}),
		"unknown env":       filingPost(vatPreviewPath, map[string]any{"environment": "staging"}),
		"not json":          {method: http.MethodPost, path: vatPreviewPath, body: "nope", contentType: "application/json", confirm: "1"},
		"oversized body":    {method: http.MethodPost, path: vatPreviewPath, body: `{"environment":"` + strings.Repeat("a", 5000) + `"}`, contentType: "application/json", confirm: "1"},
		"refresh bad id":    filingPost("/api/v1/filings/submissions/abc/refresh", map[string]any{}),
		"login test no env": filingPost("/api/v1/settings/isds-credentials/test", map[string]any{}),
	} {
		t.Run(name, func(t *testing.T) {
			w := f.do(q)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, body: %s", w.Code, w.Body.String())
			}
		})
	}
	if w := f.do(filingPost("/api/v1/filings/submissions/999/refresh", map[string]any{})); w.Code != http.StatusNotFound {
		t.Errorf("refresh of a missing submission = %d", w.Code)
	}
}

func TestFilingHandler_LoginTest(t *testing.T) {
	testPath := testISDSPath + "/test"

	t.Run("sending only login", func(t *testing.T) {
		f := newFilingFixture(t, false, "127.0.0.1")
		f.client.owner = &isds.OwnerInfo{BoxID: "abc1234", BoxType: "FO", GivenNames: "Jan", LastName: "Testovaci"}
		f.client.user = &isds.UserInfo{Privileges: isds.PrivilegeCreateAndSend | isds.PrivilegeViewInfo}
		w := f.do(filingPost(testPath, map[string]any{"environment": "test"}))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
		}
		got := decode[map[string]any](t, w)
		want := map[string]any{"boxId": "abc1234", "boxType": "FO", "ownerName": "Jan Testovaci", "canCreateAndSend": true, "canReadMessages": false}
		if len(got) != len(want) {
			t.Errorf("fields = %v, want exactly %v", got, want)
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s = %v, want %v", k, got[k], v)
			}
		}
	})
	t.Run("reading login warns", func(t *testing.T) {
		f := newFilingFixture(t, false, "127.0.0.1")
		f.client.owner = &isds.OwnerInfo{BoxID: "abc1234", BoxType: "FO"}
		f.client.user = &isds.UserInfo{Privileges: isds.PrivilegeCreateAndSend | isds.PrivilegeReadAll}
		got := decode[isdsLoginTestResponse](t, f.do(filingPost(testPath, map[string]any{"environment": "test"})))
		if !got.CanReadMessages || !strings.Contains(got.Warning, "doručení přihlášením") {
			t.Errorf("response = %+v", got)
		}
	})
	t.Run("login that cannot send warns", func(t *testing.T) {
		f := newFilingFixture(t, false, "127.0.0.1")
		f.client.owner = &isds.OwnerInfo{BoxID: "abc1234", BoxType: "FO"}
		f.client.user = &isds.UserInfo{Privileges: isds.PrivilegeViewInfo}
		got := decode[isdsLoginTestResponse](t, f.do(filingPost(testPath, map[string]any{"environment": "test"})))
		if got.CanCreateAndSend || !strings.Contains(got.Warning, "nemá právo") {
			t.Errorf("response = %+v", got)
		}
	})
	t.Run("rejected login", func(t *testing.T) {
		f := newFilingFixture(t, false, "127.0.0.1")
		f.client.ownerErr = fmt.Errorf("x: %w", isds.ErrUnauthorized)
		w := f.do(filingPost(testPath, map[string]any{"environment": "test"}))
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("status = %d", w.Code)
		}
	})
	t.Run("no stored login", func(t *testing.T) {
		f := newFilingFixture(t, false, "127.0.0.1")
		if err := secrets.NewISDSStore().Delete(secrets.ISDSEnvironmentTest); err != nil {
			t.Fatal(err)
		}
		w := f.do(filingPost(testPath, map[string]any{"environment": "test"}))
		if w.Code != http.StatusConflict {
			t.Errorf("status = %d, body: %s", w.Code, w.Body.String())
		}
	})
	t.Run("not mounted without a tester", func(t *testing.T) {
		r := newISDSRouter(t, false, "127.0.0.1")
		w := isdsReq{method: http.MethodPost, path: testPath, body: `{"environment":"test"}`, contentType: "application/json", confirm: "1"}.do(r)
		if w.Code == http.StatusOK {
			t.Errorf("status = %d", w.Code)
		}
	})
}

func TestFilingHandler_NoSecretsInResponsesOrLogs(t *testing.T) {
	f := newFilingFixture(t, false, "127.0.0.1")
	f.epo.findings = []epo.Finding{{Type: epo.TypePassable, Code: "FAKE_P", Text: "fake warning"}}
	f.client.owner = &isds.OwnerInfo{BoxID: "abc1234", BoxType: "FO", GivenNames: "Jan", LastName: "Testovaci"}
	f.client.user = &isds.UserInfo{Privileges: isds.PrivilegeCreateAndSend}

	p := f.preview(t, "production")
	sub := decode[filingSubmissionResponse](t, f.send(p.ConfirmToken, "production", true))
	f.send(p.ConfirmToken, "production", true) // token reuse
	f.do(isdsReq{method: http.MethodGet, path: vatListPath})
	deliveredAt := time.Date(2026, 11, 24, 12, 7, 0, 0, time.UTC)
	f.client.delivery = &isds.DeliveryInfo{Status: isds.StatusDelivered, DeliveryTime: &deliveredAt}
	f.client.zfo = []byte("zfo")
	f.do(filingPost(fmt.Sprintf("/api/v1/filings/submissions/%d/refresh", sub.ID), map[string]any{}))
	f.do(filingPost(testISDSPath+"/test", map[string]any{"environment": "test"}))
	f.client.createErr = fmt.Errorf("CreateMessage: %w", isds.ErrISDSUnavailable)
	f.doc.setXML(filingTestXML + "<x/>")
	p2 := f.preview(t, "test")
	f.send(p2.ConfirmToken, "test", true)

	all := strings.Join(f.responses, "\n") + f.logs.String()
	for _, secret := range []string{testISDSUsername, testISDSPassword, "fake-body-1", "<Pisemnost>"} {
		if strings.Contains(all, secret) {
			t.Errorf("a response or log line contains %q", secret)
		}
	}
	// Tokens appear only in the preview responses, never in logs.
	for _, token := range []string{p.ConfirmToken, p2.ConfirmToken} {
		if strings.Contains(f.logs.String(), token) {
			t.Error("a confirmation token reached the logs")
		}
	}
	if !strings.Contains(f.logs.String(), "ISDS filing sent") {
		t.Error("the audit log line is missing")
	}
}
