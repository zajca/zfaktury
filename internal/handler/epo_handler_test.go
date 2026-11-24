package handler

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/epo"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/testutil"
)

const (
	epoTestXML = `<?xml version="1.0" encoding="UTF-8"?><Pisemnost nazevSW="ZFaktury"><DPHDP3 verzePis="01.02.16"></DPHDP3></Pisemnost>`

	epoFakeFormURL = "https://mojedane.gov.cz/dpr/adis/idpr_epo/epo2/form/form_uvod.faces?form_id=FAKEFORMID0000000000000000000000"

	epoURLBody = `<?xml version="1.0" encoding="UTF-8"?><URL>` + epoFakeFormURL + `</URL>`

	epoCheckWarningBody = `<?xml version="1.0" encoding="UTF-8"?><Chyby>` +
		`<Chyba Typ="P" Zkr="PODANO_PO_LHUTE" Polozka="d_poddp" Oddil="VetaD" Radek="" DoplInfo=""><Text>Ověřte, zda není podání podáváno po lhůtě.</Text></Chyba>` +
		`<Chyba Typ="I" Zkr="TEST_REZIM" Polozka="" Oddil="" Radek="" DoplInfo=""><Text>Testovací režim.</Text></Chyba></Chyby>`

	epoCheckCriticalBody = `<?xml version="1.0" encoding="UTF-8"?><Chyby>` +
		`<Chyba Typ="K" Zkr="POVINNA_POLOZKA" Polozka="dic" Oddil="VetaP" Radek="3" DoplInfo="DIČ"><Text>Povinná položka není vyplněna.</Text></Chyba>` +
		`<Chyba Typ="I" Zkr="TEST_REZIM" Polozka="" Oddil="" Radek="" DoplInfo=""><Text>Testovací režim.</Text></Chyba></Chyby>`

	epoMaintenanceBody = `<!DOCTYPE html><html><body>Údržba</body></html>`
)

// epoFake is an httptest stand-in for Podatelna EPO. Each mode (test=1 and
// otevriFormular=1) answers with its configured status and body, after an
// optional delay.
type epoFake struct {
	mu         sync.Mutex
	checkBody  string
	openBody   string
	checkCode  int
	openCode   int
	checkDelay time.Duration
	received   map[string][][]byte
}

func newEPOFake(t *testing.T) (*epoFake, *httptest.Server) {
	t.Helper()
	f := &epoFake{
		checkBody: epoCheckWarningBody,
		openBody:  epoURLBody,
		checkCode: http.StatusOK,
		openCode:  http.StatusOK,
		received:  map[string][][]byte{},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mode := "open"
		code, resp := f.openCode, f.openBody
		if r.URL.Query().Get("test") == "1" {
			mode = "check"
			code, resp = f.checkCode, f.checkBody
			if f.checkDelay > 0 {
				select {
				case <-time.After(f.checkDelay):
				case <-r.Context().Done():
				}
			}
		}
		f.mu.Lock()
		f.received[mode] = append(f.received[mode], body)
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/xml;charset=UTF-8")
		w.WriteHeader(code)
		_, _ = w.Write([]byte(resp))
	}))
	t.Cleanup(srv.Close)
	return f, srv
}

func (f *epoFake) count(mode string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.received[mode])
}

type epoTestEnv struct {
	router *chi.Mux
	db     *sql.DB
	fake   *epoFake
	ids    map[service.EPOKind]int64
}

func setupEPOEnv(t *testing.T, clientOpts ...epo.Option) *epoTestEnv {
	t.Helper()
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	vatReturnSvc := service.NewVATReturnService(
		repository.NewVATReturnRepository(db),
		repository.NewInvoiceRepository(db),
		repository.NewExpenseRepository(db),
		repository.NewSettingsRepository(db),
		nil,
	)
	vatControlSvc := service.NewVATControlStatementService(
		repository.NewVATControlStatementRepository(db),
		repository.NewInvoiceRepository(db),
		repository.NewExpenseRepository(db),
		repository.NewContactRepository(db),
		repository.NewSettingsRepository(db),
		nil,
	)
	viesSvc := service.NewVIESSummaryService(
		repository.NewVIESSummaryRepository(db),
		repository.NewInvoiceRepository(db),
		repository.NewContactRepository(db),
		repository.NewSettingsRepository(db),
		nil,
	)
	incomeTaxSvc := service.NewIncomeTaxReturnService(
		repository.NewIncomeTaxReturnRepository(db),
		repository.NewInvoiceRepository(db),
		repository.NewExpenseRepository(db),
		repository.NewSettingsRepository(db),
		repository.NewTaxYearSettingsRepository(db),
		repository.NewTaxPrepaymentRepository(db),
		nil, nil,
	)

	vr := &domain.VATReturn{Period: domain.TaxPeriod{Year: 2025, Month: 1}, FilingType: domain.FilingTypeRegular}
	if err := vatReturnSvc.Create(ctx, vr); err != nil {
		t.Fatalf("creating VAT return: %v", err)
	}
	cs := &domain.VATControlStatement{Period: domain.TaxPeriod{Year: 2025, Month: 1}, FilingType: domain.FilingTypeRegular}
	if err := vatControlSvc.Create(ctx, cs); err != nil {
		t.Fatalf("creating control statement: %v", err)
	}
	vs := &domain.VIESSummary{Period: domain.TaxPeriod{Year: 2025, Quarter: 1}, FilingType: domain.FilingTypeRegular}
	if err := viesSvc.Create(ctx, vs); err != nil {
		t.Fatalf("creating VIES summary: %v", err)
	}
	itr := &domain.IncomeTaxReturn{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := incomeTaxSvc.Create(ctx, itr); err != nil {
		t.Fatalf("creating income tax return: %v", err)
	}

	fake, srv := newEPOFake(t)
	opts := append([]epo.Option{epo.WithBaseURL(srv.URL + "/dpr")}, clientOpts...)
	epoSvc := service.NewEPOService(map[service.EPOKind]service.EPOXMLSource{
		service.EPOKindVATReturn:        vatReturnSvc,
		service.EPOKindControlStatement: vatControlSvc,
		service.EPOKindVIESSummary:      viesSvc,
		service.EPOKindIncomeTax:        incomeTaxSvc,
	}, epo.NewClient(opts...))

	r := chi.NewRouter()
	r.Mount("/api/v1/epo", NewEPOHandler(epoSvc).Routes())

	return &epoTestEnv{
		router: r,
		db:     db,
		fake:   fake,
		ids: map[service.EPOKind]int64{
			service.EPOKindVATReturn:        vr.ID,
			service.EPOKindControlStatement: cs.ID,
			service.EPOKindVIESSummary:      vs.ID,
			service.EPOKindIncomeTax:        itr.ID,
		},
	}
}

var epoXMLTables = map[service.EPOKind]string{
	service.EPOKindVATReturn:        "vat_returns",
	service.EPOKindControlStatement: "vat_control_statements",
	service.EPOKindVIESSummary:      "vies_summaries",
	service.EPOKindIncomeTax:        "income_tax_returns",
}

func (e *epoTestEnv) storeXML(t *testing.T, kind service.EPOKind, xmlData string) {
	t.Helper()
	_, err := e.db.ExecContext(context.Background(),
		"UPDATE "+epoXMLTables[kind]+" SET xml_data = ? WHERE id = ?", []byte(xmlData), e.ids[kind])
	if err != nil {
		t.Fatalf("storing XML for %s: %v", kind, err)
	}
}

func (e *epoTestEnv) post(kind string, id string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/epo/"+kind+"/"+id+"/open-form", nil)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

func (e *epoTestEnv) postKind(kind service.EPOKind) *httptest.ResponseRecorder {
	return e.post(string(kind), strconv.FormatInt(e.ids[kind], 10))
}

func decodeEPOError(t *testing.T, w *httptest.ResponseRecorder) epoErrorResponse {
	t.Helper()
	var resp epoErrorResponse
	if err := json.NewDecoder(bytes.NewReader(w.Body.Bytes())).Decode(&resp); err != nil {
		t.Fatalf("decoding error body %q: %v", w.Body.String(), err)
	}
	return resp
}

func TestEPOOpenForm_AllKinds(t *testing.T) {
	for kind := range epoXMLTables {
		t.Run(string(kind), func(t *testing.T) {
			env := setupEPOEnv(t)
			env.storeXML(t, kind, epoTestXML)

			w := env.postKind(kind)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
			}

			var resp epoOpenFormResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("decoding: %v", err)
			}
			if resp.URL != epoFakeFormURL {
				t.Errorf("url = %q, want %q", resp.URL, epoFakeFormURL)
			}
			if len(resp.Warnings) != 1 {
				t.Fatalf("warnings = %d, want 1 (TEST_REZIM must be stripped)", len(resp.Warnings))
			}
			warn := resp.Warnings[0]
			if warn.Type != "P" || warn.Code != "PODANO_PO_LHUTE" || warn.Field != "d_poddp" || warn.Section != "VetaD" || warn.Text == "" {
				t.Errorf("warning = %+v", warn)
			}

			if env.fake.count("check") != 1 || env.fake.count("open") != 1 {
				t.Fatalf("EPO calls check/open = %d/%d, want 1/1", env.fake.count("check"), env.fake.count("open"))
			}
			for _, mode := range []string{"check", "open"} {
				got := env.fake.received[mode][0]
				if kind == service.EPOKindControlStatement {
					if xmlData := singleZIPEntry(t, got); xmlData != epoTestXML {
						t.Errorf("%s: archived XML differs from the stored XML", mode)
					}
					continue
				}
				if string(got) != epoTestXML {
					t.Errorf("%s: EPO did not receive the stored XML verbatim", mode)
				}
			}
		})
	}
}

// singleZIPEntry returns the content of the only entry of the ZIP archive body.
func singleZIPEntry(t *testing.T, body []byte) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("body is not a ZIP archive: %v", err)
	}
	if len(zr.File) != 1 {
		t.Fatalf("archive has %d entries, want 1", len(zr.File))
	}
	rc, err := zr.File[0].Open()
	if err != nil {
		t.Fatalf("opening ZIP entry: %v", err)
	}
	defer func() { _ = rc.Close() }()
	content, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading ZIP entry: %v", err)
	}
	return string(content)
}

func TestEPOOpenForm_NoWarningsIsEmptyArray(t *testing.T) {
	env := setupEPOEnv(t)
	env.fake.checkBody = `<Chyby><Chyba Typ="I" Zkr="TEST_REZIM" Polozka="" Oddil="" Radek="" DoplInfo=""><Text>x</Text></Chyba></Chyby>`
	env.storeXML(t, service.EPOKindVATReturn, epoTestXML)

	w := env.postKind(service.EPOKindVATReturn)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"warnings":[]`) {
		t.Errorf("body = %s, want warnings as an empty array", w.Body.String())
	}
}

func TestEPOOpenForm_StatusMapping(t *testing.T) {
	type tc struct {
		name       string
		setup      func(env *epoTestEnv)
		kind       string
		id         func(env *epoTestEnv) string
		wantStatus int
		wantOpens  int
	}
	vat := string(service.EPOKindVATReturn)
	vatID := func(env *epoTestEnv) string { return strconv.FormatInt(env.ids[service.EPOKindVATReturn], 10) }
	withXML := func(env *epoTestEnv) { env.storeXML(t, service.EPOKindVATReturn, epoTestXML) }

	cases := []tc{
		{
			name:       "unknown kind",
			setup:      withXML,
			kind:       "social-insurance",
			id:         vatID,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "non numeric id",
			setup:      withXML,
			kind:       vat,
			id:         func(*epoTestEnv) string { return "abc" },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero id",
			setup:      withXML,
			kind:       vat,
			id:         func(*epoTestEnv) string { return "0" },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found",
			setup:      withXML,
			kind:       vat,
			id:         func(*epoTestEnv) string { return "99999" },
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "xml not generated",
			setup:      func(*epoTestEnv) {},
			kind:       vat,
			id:         vatID,
			wantStatus: http.StatusConflict,
		},
		{
			name: "stored xml is not xml",
			setup: func(env *epoTestEnv) {
				env.storeXML(t, service.EPOKindVATReturn, "not xml at all")
			},
			kind:       vat,
			id:         vatID,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "EPO unavailable on check",
			setup: func(env *epoTestEnv) {
				withXML(env)
				env.fake.checkCode = http.StatusInternalServerError
			},
			kind:       vat,
			id:         vatID,
			wantStatus: http.StatusBadGateway,
		},
		{
			name: "EPO unavailable on open",
			setup: func(env *epoTestEnv) {
				withXML(env)
				env.fake.openCode = http.StatusServiceUnavailable
			},
			kind:       vat,
			id:         vatID,
			wantStatus: http.StatusBadGateway,
			wantOpens:  1,
		},
		{
			name: "EPO maintenance page",
			setup: func(env *epoTestEnv) {
				withXML(env)
				env.fake.checkBody = epoMaintenanceBody
			},
			kind:       vat,
			id:         vatID,
			wantStatus: http.StatusBadGateway,
		},
		{
			name: "EPO returns a foreign link",
			setup: func(env *epoTestEnv) {
				withXML(env)
				env.fake.openBody = `<URL>https://evil.example.com/dpr/form.faces</URL>`
			},
			kind:       vat,
			id:         vatID,
			wantStatus: http.StatusBadGateway,
			wantOpens:  1,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := setupEPOEnv(t)
			c.setup(env)

			w := env.post(c.kind, c.id(env))
			if w.Code != c.wantStatus {
				t.Fatalf("status = %d, want %d, body: %s", w.Code, c.wantStatus, w.Body.String())
			}
			if resp := decodeEPOError(t, w); resp.Error == "" {
				t.Errorf("error body is empty: %s", w.Body.String())
			}
			if strings.Contains(w.Body.String(), "FAKEFORMID") {
				t.Errorf("error body leaks a form link: %s", w.Body.String())
			}
			if got := env.fake.count("open"); got != c.wantOpens {
				t.Errorf("otevriFormular calls = %d, want %d", got, c.wantOpens)
			}
		})
	}
}

func TestEPOOpenForm_RejectedOnCheck(t *testing.T) {
	env := setupEPOEnv(t)
	env.fake.checkBody = epoCheckCriticalBody
	env.storeXML(t, service.EPOKindVATReturn, epoTestXML)

	w := env.postKind(service.EPOKindVATReturn)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422, body: %s", w.Code, w.Body.String())
	}
	resp := decodeEPOError(t, w)
	if len(resp.Findings) != 1 {
		t.Fatalf("findings = %d, want 1 (TEST_REZIM stripped)", len(resp.Findings))
	}
	want := epoFindingResponse{
		Type:    "K",
		Code:    "POVINNA_POLOZKA",
		Field:   "dic",
		Section: "VetaP",
		Row:     "3",
		Text:    "Povinná položka není vyplněna.",
	}
	if resp.Findings[0] != want {
		t.Errorf("finding = %+v, want %+v", resp.Findings[0], want)
	}
	if env.fake.count("open") != 0 {
		t.Errorf("otevriFormular must not be called after a blocking finding")
	}
}

func TestEPOOpenForm_RejectedOnOpen(t *testing.T) {
	env := setupEPOEnv(t)
	env.fake.openBody = epoCheckCriticalBody
	env.storeXML(t, service.EPOKindVATReturn, epoTestXML)
	env.fake.checkBody = epoCheckWarningBody

	w := env.postKind(service.EPOKindVATReturn)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422, body: %s", w.Code, w.Body.String())
	}
	if resp := decodeEPOError(t, w); len(resp.Findings) == 0 {
		t.Errorf("expected findings in the 422 body: %s", w.Body.String())
	}
}

func TestEPOOpenForm_Timeout(t *testing.T) {
	env := setupEPOEnv(t, epo.WithTimeout(50*time.Millisecond))
	env.fake.checkDelay = 2 * time.Second
	env.storeXML(t, service.EPOKindVATReturn, epoTestXML)

	w := env.postKind(service.EPOKindVATReturn)
	if w.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want 504, body: %s", w.Code, w.Body.String())
	}
	if env.fake.count("open") != 0 {
		t.Errorf("otevriFormular must not be called after a timed out check")
	}
}
