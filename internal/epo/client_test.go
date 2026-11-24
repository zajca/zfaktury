package epo

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const sampleXML = `<?xml version="1.0" encoding="UTF-8"?>
<Pisemnost nazevSW="ZFaktury"><DPHDP3 verzePis="01.02.16"></DPHDP3></Pisemnost>`

const fakeFormURL = "https://mojedane.gov.cz/dpr/adis/idpr_epo/epo2/form/form_uvod.faces?form_id=FAKEFORMID0000000000000000000000"

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return data
}

// recorded is the last request the fake received.
type recorded struct {
	method      string
	path        string
	query       string
	contentType string
	accept      string
	body        []byte
}

// newFake starts an httptest server answering every request with status and
// body, recording the request it saw.
func newFake(t *testing.T, status int, body []byte) (*httptest.Server, *recorded) {
	t.Helper()
	rec := &recorded{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.method = r.Method
		rec.path = r.URL.Path
		rec.query = r.URL.RawQuery
		rec.contentType = r.Header.Get("Content-Type")
		rec.accept = r.Header.Get("Accept")
		rec.body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/xml;charset=UTF-8")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

func TestOpenForm_Success(t *testing.T) {
	for _, name := range []string{"open_form_url.xml", "open_form_url_mixed_case.xml"} {
		t.Run(name, func(t *testing.T) {
			srv, rec := newFake(t, http.StatusOK, fixture(t, name))
			c := NewClient(WithBaseURL(srv.URL + "/dpr"))

			got, err := c.OpenForm(context.Background(), []byte(sampleXML))
			if err != nil {
				t.Fatalf("OpenForm: %v", err)
			}
			if got != fakeFormURL {
				t.Errorf("url = %q, want %q", got, fakeFormURL)
			}
			if rec.method != http.MethodPost {
				t.Errorf("method = %q, want POST", rec.method)
			}
			if rec.path != "/dpr/epo_podani" {
				t.Errorf("path = %q, want /dpr/epo_podani", rec.path)
			}
			if rec.query != "otevriFormular=1" {
				t.Errorf("query = %q, want otevriFormular=1", rec.query)
			}
			if rec.contentType != "application/octet-stream" {
				t.Errorf("content type = %q", rec.contentType)
			}
			if rec.accept != "application/xml" {
				t.Errorf("accept = %q", rec.accept)
			}
			if !bytes.Equal(rec.body, []byte(sampleXML)) {
				t.Errorf("body bytes differ from the input XML")
			}
		})
	}
}

func TestOpenForm_RejectedWithFindings(t *testing.T) {
	srv, _ := newFake(t, http.StatusOK, fixture(t, "check_critical.xml"))
	c := NewClient(WithBaseURL(srv.URL))

	_, err := c.OpenForm(context.Background(), []byte(sampleXML))
	var rejected *RejectedError
	if !errors.As(err, &rejected) {
		t.Fatalf("err = %v, want *RejectedError", err)
	}
	if len(rejected.Findings) != 3 {
		t.Fatalf("findings = %d, want 3", len(rejected.Findings))
	}
	first := rejected.Findings[0]
	want := Finding{
		Type:    "K",
		Code:    "POVINNA_POLOZKA",
		Field:   "dic",
		Section: "VetaP",
		Row:     "3",
		Info:    "DIČ",
		Text:    "Povinná položka není vyplněna.",
	}
	if first != want {
		t.Errorf("first finding = %+v, want %+v", first, want)
	}
}

func TestOpenForm_RejectsBadLinks(t *testing.T) {
	cases := map[string]string{
		"foreign host":      "https://evil.example.com/dpr/adis/form.faces?form_id=x",
		"look-alike host":   "https://mojedane.gov.cz.evil.example.com/dpr/form.faces",
		"http scheme":       "http://mojedane.gov.cz/dpr/adis/form.faces?form_id=x",
		"javascript scheme": "javascript:alert(1)",
		"path outside dpr":  "https://mojedane.gov.cz/other/form.faces",
		"userinfo":          "https://mojedane.gov.cz@evil.example.com/dpr/form.faces",
		"relative":          "/dpr/adis/form.faces",
	}
	for name, link := range cases {
		t.Run(name, func(t *testing.T) {
			body := []byte(`<?xml version="1.0" encoding="UTF-8"?><URL>` + link + `</URL>`)
			srv, _ := newFake(t, http.StatusOK, body)
			c := NewClient(WithBaseURL(srv.URL))

			got, err := c.OpenForm(context.Background(), []byte(sampleXML))
			if err == nil {
				t.Fatalf("OpenForm returned %q, want an error", got)
			}
			if !errors.Is(err, ErrUnexpectedResponse) || !errors.Is(err, ErrInvalidFormURL) {
				t.Errorf("err = %v, want ErrUnexpectedResponse wrapping ErrInvalidFormURL", err)
			}
			if got != "" {
				t.Errorf("url = %q, want empty", got)
			}
		})
	}
}

func TestOpenForm_CustomAllowedHost(t *testing.T) {
	body := []byte(`<URL>https://epo.example.test/dpr/form.faces?form_id=x</URL>`)
	srv, _ := newFake(t, http.StatusOK, body)
	c := NewClient(WithBaseURL(srv.URL), WithAllowedHost("epo.example.test"))

	if _, err := c.OpenForm(context.Background(), []byte(sampleXML)); err != nil {
		t.Fatalf("OpenForm: %v", err)
	}
}

func TestCheck_HappyPathStripsTestMode(t *testing.T) {
	srv, rec := newFake(t, http.StatusOK, fixture(t, "check_warning.xml"))
	c := NewClient(WithBaseURL(srv.URL + "/dpr"))

	findings, err := c.Check(context.Background(), []byte(sampleXML))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if rec.query != "test=1" || rec.path != "/dpr/epo_podani" || rec.method != http.MethodPost {
		t.Errorf("request = %s %s?%s, want POST /dpr/epo_podani?test=1", rec.method, rec.path, rec.query)
	}
	if !bytes.Equal(rec.body, []byte(sampleXML)) {
		t.Errorf("body bytes differ from the input XML")
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1 (TEST_REZIM stripped)", len(findings))
	}
	if findings[0].Type != TypePassable || findings[0].Field != "d_poddp" || findings[0].IsBlocking() {
		t.Errorf("finding = %+v, want the passable d_poddp warning", findings[0])
	}
}

func TestCheck_ReportsBlockingFindings(t *testing.T) {
	srv, _ := newFake(t, http.StatusOK, fixture(t, "check_critical.xml"))
	c := NewClient(WithBaseURL(srv.URL))

	findings, err := c.Check(context.Background(), []byte(sampleXML))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("findings = %d, want 2", len(findings))
	}
	if !findings[0].IsBlocking() || findings[1].IsBlocking() {
		t.Errorf("blocking flags = %v/%v, want true/false", findings[0].IsBlocking(), findings[1].IsBlocking())
	}
}

func TestCheck_BlockingFindingWithoutTestMode(t *testing.T) {
	body := []byte(`<Chyby><Chyba Typ="S" Zkr="XSD" Polozka="" Oddil="" Radek="" DoplInfo=""><Text>Chybná struktura.</Text></Chyba></Chyby>`)
	srv, _ := newFake(t, http.StatusOK, body)
	c := NewClient(WithBaseURL(srv.URL))

	findings, err := c.Check(context.Background(), []byte(sampleXML))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(findings) != 1 || !findings[0].IsBlocking() {
		t.Errorf("findings = %+v, want one blocking finding", findings)
	}
}

func TestCheck_MissingTestModeIsAnError(t *testing.T) {
	srv, _ := newFake(t, http.StatusOK, fixture(t, "check_no_test_mode.xml"))
	c := NewClient(WithBaseURL(srv.URL))

	findings, err := c.Check(context.Background(), []byte(sampleXML))
	if !errors.Is(err, ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
	if findings != nil {
		t.Errorf("findings = %+v, want nil", findings)
	}
}

func TestCheck_URLAnswerIsAnError(t *testing.T) {
	srv, _ := newFake(t, http.StatusOK, fixture(t, "open_form_url.xml"))
	c := NewClient(WithBaseURL(srv.URL))

	if _, err := c.Check(context.Background(), []byte(sampleXML)); !errors.Is(err, ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

func TestUnavailableAndUnexpectedAnswers(t *testing.T) {
	t.Run("HTTP 500", func(t *testing.T) {
		srv, _ := newFake(t, http.StatusInternalServerError, []byte("boom"))
		c := NewClient(WithBaseURL(srv.URL))

		_, err := c.OpenForm(context.Background(), []byte(sampleXML))
		if !errors.Is(err, ErrEPOUnavailable) {
			t.Fatalf("err = %v, want ErrEPOUnavailable", err)
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("err = %q, want the status in the message", err)
		}
	})

	t.Run("redirect is not followed", func(t *testing.T) {
		srv, _ := newFake(t, http.StatusFound, nil)
		c := NewClient(WithBaseURL(srv.URL))

		if _, err := c.Check(context.Background(), []byte(sampleXML)); !errors.Is(err, ErrEPOUnavailable) {
			t.Fatalf("err = %v, want ErrEPOUnavailable", err)
		}
	})

	t.Run("HTML maintenance page with 200", func(t *testing.T) {
		srv, _ := newFake(t, http.StatusOK, fixture(t, "maintenance.html"))
		c := NewClient(WithBaseURL(srv.URL))

		if _, err := c.OpenForm(context.Background(), []byte(sampleXML)); !errors.Is(err, ErrUnexpectedResponse) {
			t.Fatalf("OpenForm err = %v, want ErrUnexpectedResponse", err)
		}
		if _, err := c.Check(context.Background(), []byte(sampleXML)); !errors.Is(err, ErrUnexpectedResponse) {
			t.Fatalf("Check err = %v, want ErrUnexpectedResponse", err)
		}
	})

	t.Run("empty body", func(t *testing.T) {
		srv, _ := newFake(t, http.StatusOK, nil)
		c := NewClient(WithBaseURL(srv.URL))

		if _, err := c.OpenForm(context.Background(), []byte(sampleXML)); !errors.Is(err, ErrUnexpectedResponse) {
			t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
		}
	})

	t.Run("empty findings list", func(t *testing.T) {
		srv, _ := newFake(t, http.StatusOK, []byte(`<Chyby></Chyby>`))
		c := NewClient(WithBaseURL(srv.URL))

		if _, err := c.OpenForm(context.Background(), []byte(sampleXML)); !errors.Is(err, ErrUnexpectedResponse) {
			t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
		}
	})
}

func TestTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(func() {
		close(release)
		srv.Close()
	})
	c := NewClient(WithBaseURL(srv.URL), WithTimeout(50*time.Millisecond))

	_, err := c.OpenForm(context.Background(), []byte(sampleXML))
	if !errors.Is(err, ErrEPOTimeout) {
		t.Fatalf("err = %v, want ErrEPOTimeout", err)
	}
}

func TestContextCancelIsNotATimeout(t *testing.T) {
	srv, _ := newFake(t, http.StatusOK, fixture(t, "open_form_url.xml"))
	c := NewClient(WithBaseURL(srv.URL))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.OpenForm(ctx, []byte(sampleXML))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if errors.Is(err, ErrEPOTimeout) {
		t.Errorf("a canceled context must not map to ErrEPOTimeout")
	}
}

func TestOversizedResponse(t *testing.T) {
	big := append([]byte(`<?xml version="1.0"?><URL>`), bytes.Repeat([]byte("a"), 2048)...)
	big = append(big, []byte(`</URL>`)...)
	srv, _ := newFake(t, http.StatusOK, big)
	c := NewClient(WithBaseURL(srv.URL), WithMaxResponseBytes(1024))

	_, err := c.OpenForm(context.Background(), []byte(sampleXML))
	if !errors.Is(err, ErrUnexpectedResponse) {
		t.Fatalf("err = %v, want ErrUnexpectedResponse", err)
	}
}

func TestInputValidation(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
	}))
	t.Cleanup(srv.Close)
	c := NewClient(WithBaseURL(srv.URL), WithMaxRequestBytes(64))

	cases := map[string][]byte{
		"empty":             nil,
		"not xml":           []byte("hello"),
		"html":              []byte("<html></html>"),
		"oversized":         append([]byte("<Pisemnost>"), bytes.Repeat([]byte("a"), 100)...),
		"json":              []byte(`{"a":1}`),
		"leading garbage":   []byte("x<?xml version=\"1.0\"?><Pisemnost/>"),
		"other root, no pi": []byte("<Other/>"),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := c.OpenForm(context.Background(), input); !errors.Is(err, ErrInvalidXML) {
				t.Errorf("OpenForm err = %v, want ErrInvalidXML", err)
			}
			if _, err := c.Check(context.Background(), input); !errors.Is(err, ErrInvalidXML) {
				t.Errorf("Check err = %v, want ErrInvalidXML", err)
			}
		})
	}
	if calls != 0 {
		t.Errorf("fake received %d request(s), want none for invalid input", calls)
	}
}

func TestInputAccepted(t *testing.T) {
	srv, _ := newFake(t, http.StatusOK, fixture(t, "open_form_url.xml"))
	c := NewClient(WithBaseURL(srv.URL))

	inputs := map[string]string{
		"declaration":    sampleXML,
		"bom":            "\xef\xbb\xbf" + sampleXML,
		"leading blanks": "\n  " + sampleXML,
		"bare root":      `<Pisemnost nazevSW="ZFaktury"></Pisemnost>`,
	}
	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			if _, err := c.OpenForm(context.Background(), []byte(input)); err != nil {
				t.Errorf("OpenForm: %v", err)
			}
		})
	}
}

func TestFindingClassification(t *testing.T) {
	cases := []struct {
		typ      string
		blocking bool
		warning  bool
	}{
		{"P", false, true},
		{"N", false, true},
		{"I", false, false},
		{"p", false, true},
		{" n ", false, true},
		{"K", true, false},
		{"k", true, false},
		{"S", true, false},
		{"E", true, false},
		{"", true, false},
		{"X", true, false},
		{"KRITICKA", true, false},
	}
	for _, c := range cases {
		f := Finding{Type: c.typ}
		if got := f.IsBlocking(); got != c.blocking {
			t.Errorf("Finding{Type: %q}.IsBlocking() = %v, want %v", c.typ, got, c.blocking)
		}
		if got := f.IsWarning(); got != c.warning {
			t.Errorf("Finding{Type: %q}.IsWarning() = %v, want %v", c.typ, got, c.warning)
		}
	}
}

func TestCheck_LowercaseAndUnknownTypesBlock(t *testing.T) {
	body := []byte(`<Chyby><Chyba Typ="k" Zkr="A"><Text>x</Text></Chyba><Chyba Typ="" Zkr="B"><Text>y</Text></Chyba></Chyby>`)
	srv, _ := newFake(t, http.StatusOK, body)
	c := NewClient(WithBaseURL(srv.URL))

	findings, err := c.Check(context.Background(), []byte(sampleXML))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(findings) != 2 || !findings[0].IsBlocking() || !findings[1].IsBlocking() {
		t.Errorf("findings = %+v, want two blocking findings", findings)
	}
}

func TestValidateFormURL(t *testing.T) {
	if err := ValidateFormURL(fakeFormURL, DefaultAllowedHost); err != nil {
		t.Errorf("valid link rejected: %v", err)
	}
	if err := ValidateFormURL("https://MOJEDANE.gov.cz/dpr/x", DefaultAllowedHost); err != nil {
		t.Errorf("host comparison must be case-insensitive: %v", err)
	}
	for _, bad := range []string{
		"",
		"http://mojedane.gov.cz/dpr/x",
		"https://mojedane.gov.cz:8443/dpr/x",
		"https://other.gov.cz/dpr/x",
		"https://mojedane.gov.cz/",
		"https://mojedane.gov.cz/dpr",
		"https://mojedane.gov.cz/dpr/../evil",
		"https://mojedane.gov.cz/dpr/%2e%2e/evil",
		"https://mojedane.gov.cz/dpr/a/./b",
		"https://mojedane.gov.cz/dpr//b",
		"file:///etc/passwd",
		"javascript:alert(1)",
		"https://user:pw@mojedane.gov.cz/dpr/x",
		"://broken",
	} {
		if err := ValidateFormURL(bad, DefaultAllowedHost); !errors.Is(err, ErrInvalidFormURL) {
			t.Errorf("ValidateFormURL(%q) = %v, want ErrInvalidFormURL", bad, err)
		}
	}
}

func TestNewClientDefaultsAndIgnoredZeroOptions(t *testing.T) {
	c := NewClient(WithBaseURL(""), WithAllowedHost(""), WithTimeout(0), WithMaxRequestBytes(0), WithMaxResponseBytes(-1))
	if c.baseURL != DefaultBaseURL {
		t.Errorf("baseURL = %q, want default", c.baseURL)
	}
	if c.AllowedHost() != DefaultAllowedHost {
		t.Errorf("allowedHost = %q, want default", c.AllowedHost())
	}
	if c.httpClient.Timeout != DefaultTimeout {
		t.Errorf("timeout = %v, want default", c.httpClient.Timeout)
	}
	if c.maxRequestBytes != DefaultMaxRequestBytes || c.maxResponseBytes != DefaultMaxResponseBytes {
		t.Errorf("size limits = %d/%d, want defaults", c.maxRequestBytes, c.maxResponseBytes)
	}
	if got := NewClient(WithBaseURL("http://x.test/dpr/")).baseURL; got != "http://x.test/dpr" {
		t.Errorf("trailing slash not trimmed: %q", got)
	}
}
