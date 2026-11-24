package isds

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// Distinctive fake login used to prove the secrets never leak.
const (
	testUsername = "fake-user-7Qe2"
	testPassword = "Fake-Passw0rd-Zq81xK"
)

var interTagSpace = regexp.MustCompile(`>\s+<`)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return data
}

// compactXML drops the indentation between tags so a pretty-printed golden
// file can be compared with the compact request body.
func compactXML(b []byte) string {
	return strings.TrimSpace(interTagSpace.ReplaceAllString(string(b), "><"))
}

// recorded holds the requests a fake received.
type recorded struct {
	mu       sync.Mutex
	requests []recordedRequest
}

type recordedRequest struct {
	method      string
	path        string
	contentType string
	soapAction  string
	user        string
	password    string
	hasAuth     bool
	body        []byte
}

func (r *recorded) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.requests)
}

func (r *recorded) last(t *testing.T) recordedRequest {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.requests) == 0 {
		t.Fatal("the fake received no request")
	}
	return r.requests[len(r.requests)-1]
}

func newFakeFunc(t *testing.T, h http.HandlerFunc) (*httptest.Server, *recorded) {
	t.Helper()
	rec := &recorded{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		user, pass, ok := r.BasicAuth()
		rec.mu.Lock()
		rec.requests = append(rec.requests, recordedRequest{
			method:      r.Method,
			path:        r.URL.Path,
			contentType: r.Header.Get("Content-Type"),
			soapAction:  r.Header.Get("SOAPAction"),
			user:        user,
			password:    pass,
			hasAuth:     ok,
			body:        body,
		})
		rec.mu.Unlock()
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

// newFake answers every request with status and body.
func newFake(t *testing.T, status int, body []byte) (*httptest.Server, *recorded) {
	t.Helper()
	return newFakeFunc(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/xml;charset=UTF-8")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	})
}

func newTestClient(t *testing.T, srv *httptest.Server, opts ...Option) *Client {
	t.Helper()
	all := append([]Option{WithBaseURL(srv.URL)}, opts...)
	c, err := NewClient(EnvironmentTest, Credentials{Username: testUsername, Password: testPassword}, all...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// assertRequest checks method, service path, headers, basic auth and the
// exact request body against a golden file.
func assertRequest(t *testing.T, rec *recorded, wantPath, golden string) {
	t.Helper()
	got := rec.last(t)
	if got.method != http.MethodPost {
		t.Errorf("method = %q, want POST", got.method)
	}
	if got.path != wantPath {
		t.Errorf("path = %q, want %q", got.path, wantPath)
	}
	if got.contentType != "text/xml; charset=utf-8" {
		t.Errorf("content type = %q", got.contentType)
	}
	if got.soapAction != `""` {
		t.Errorf("SOAPAction = %q, want an empty quoted string", got.soapAction)
	}
	if !got.hasAuth || got.user != testUsername || got.password != testPassword {
		t.Errorf("basic auth missing or wrong (present=%v)", got.hasAuth)
	}
	if want := compactXML(fixture(t, golden)); compactXML(got.body) != want {
		t.Errorf("request body mismatch\n got: %s\nwant: %s", compactXML(got.body), want)
	}
}

func TestNewClient_RequiresExplicitConfig(t *testing.T) {
	good := Credentials{Username: "u", Password: "p"}
	tests := []struct {
		name  string
		env   Environment
		creds Credentials
		opts  []Option
	}{
		{"no environment", "", good, nil},
		{"unknown environment", "staging", good, nil},
		{"no username", EnvironmentTest, Credentials{Password: "p"}, nil},
		{"no password", EnvironmentTest, Credentials{Username: "u"}, nil},
		{"colon in username", EnvironmentTest, Credentials{Username: "a:b", Password: "p"}, nil},
		{"plain http to a remote host", EnvironmentTest, good, []Option{WithBaseURL("http://ws1.example.test")}},
		{"credentials in the base URL", EnvironmentTest, good, []Option{WithBaseURL("https://user:pw@ws1.example.test")}},
		{"query in the base URL", EnvironmentTest, good, []Option{WithBaseURL("https://ws1.example.test?x=1")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewClient(tt.env, tt.creds, tt.opts...)
			if !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("err = %v (client %v), want ErrInvalidConfig", err, c)
			}
		})
	}
}

func TestNewClient_EnvironmentBaseURLs(t *testing.T) {
	for env, want := range map[Environment]string{
		EnvironmentProduction: BaseURLProduction,
		EnvironmentTest:       BaseURLTest,
	} {
		c, err := NewClient(env, Credentials{Username: "u", Password: "p"})
		if err != nil {
			t.Fatalf("NewClient(%s): %v", env, err)
		}
		if c.baseURL != want {
			t.Errorf("base URL for %s = %q, want %q", env, c.baseURL, want)
		}
	}
	if BaseURLProduction != "https://ws1.datovka.gov.cz" || BaseURLTest != "https://ws1.datovka-test.gov.cz" {
		t.Error("base URL constants changed")
	}
	if _, err := ParseEnvironment(""); err == nil {
		t.Error("ParseEnvironment accepted an empty value")
	}
	if e, err := ParseEnvironment("test"); err != nil || e != EnvironmentTest {
		t.Errorf("ParseEnvironment(test) = %q, %v", e, err)
	}
}

func TestErrors_Mapping(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   []byte
		check  func(t *testing.T, err error)
	}{
		{
			name: "401", status: http.StatusUnauthorized, body: []byte("<html>Unauthorized</html>"),
			check: func(t *testing.T, err error) { wantIs(t, err, ErrUnauthorized) },
		},
		{
			name: "403", status: http.StatusForbidden, body: nil,
			check: func(t *testing.T, err error) { wantIs(t, err, ErrUnauthorized) },
		},
		{
			name: "500 without fault", status: http.StatusInternalServerError, body: []byte("boom"),
			check: func(t *testing.T, err error) {
				wantIs(t, err, ErrISDSUnavailable)
				if !strings.Contains(err.Error(), "500") {
					t.Errorf("error %q does not name the status", err)
				}
			},
		},
		{
			name: "503", status: http.StatusServiceUnavailable, body: nil,
			check: func(t *testing.T, err error) { wantIs(t, err, ErrISDSUnavailable) },
		},
		{
			name: "SOAP fault on 500", status: http.StatusInternalServerError, body: fixture(t, "fault.xml"),
			check: func(t *testing.T, err error) {
				wantIs(t, err, ErrSOAPFault)
				var fe *FaultError
				if !errors.As(err, &fe) || fe.Code != "soap:Server" || fe.Message != "Fake internal failure" {
					t.Errorf("FaultError = %+v", fe)
				}
			},
		},
		{
			name: "SOAP fault on 200", status: http.StatusOK, body: fixture(t, "fault.xml"),
			check: func(t *testing.T, err error) { wantIs(t, err, ErrSOAPFault) },
		},
		{
			name: "HTML on 200", status: http.StatusOK, body: fixture(t, "maintenance.html"),
			check: func(t *testing.T, err error) { wantIs(t, err, ErrUnexpectedResponse) },
		},
		{
			name: "empty 200", status: http.StatusOK, body: nil,
			check: func(t *testing.T, err error) { wantIs(t, err, ErrUnexpectedResponse) },
		},
		{
			name: "dbStatus not 0000", status: http.StatusOK, body: fixture(t, "get_owner_info_response_status_error.xml"),
			check: func(t *testing.T, err error) {
				wantIs(t, err, ErrISDSRejected)
				var se *StatusError
				if !errors.As(err, &se) || se.Code != "1001" || se.Message != "Fake db status failure" {
					t.Errorf("StatusError = %+v", se)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := newFake(t, tt.status, tt.body)
			_, err := newTestClient(t, srv).GetOwnerInfo(context.Background())
			if err == nil {
				t.Fatal("expected an error")
			}
			tt.check(t, err)
		})
	}
}

func wantIs(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Errorf("err = %v, want it to match %v", err, target)
	}
}

func TestErrors_DMStatusRejected(t *testing.T) {
	srv, _ := newFake(t, http.StatusOK, fixture(t, "create_message_response_status_error.xml"))
	_, err := newTestClient(t, srv).CreateMessage(context.Background(), validMessage())
	wantIs(t, err, ErrISDSRejected)
	var se *StatusError
	if !errors.As(err, &se) {
		t.Fatalf("err = %v, want *StatusError", err)
	}
	if se.Code != "0004" || se.RefNumber != "REF-FAKE-1" || !strings.Contains(se.Message, "Fake rejection") {
		t.Errorf("StatusError = %+v", se)
	}
}

func TestErrors_OversizedResponse(t *testing.T) {
	big := append([]byte(`<Envelope><Body><X>`), bytes.Repeat([]byte("a"), 4096)...)
	srv, _ := newFake(t, http.StatusOK, big)
	_, err := newTestClient(t, srv, WithMaxResponseBytes(1024)).GetOwnerInfo(context.Background())
	wantIs(t, err, ErrResponseTooLarge)
}

func TestErrors_ResponseExactlyAtCapIsRead(t *testing.T) {
	body := fixture(t, "get_owner_info_response.xml")
	srv, _ := newFake(t, http.StatusOK, body)
	if _, err := newTestClient(t, srv, WithMaxResponseBytes(int64(len(body)))).GetOwnerInfo(context.Background()); err != nil {
		t.Fatalf("response at the cap must be accepted: %v", err)
	}
}

func TestErrors_Timeout(t *testing.T) {
	release := make(chan struct{})
	srv, _ := newFakeFunc(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	})
	t.Cleanup(func() { close(release) })

	_, err := newTestClient(t, srv, WithTimeout(50*time.Millisecond)).GetOwnerInfo(context.Background())
	wantIs(t, err, ErrISDSTimeout)
	wantIs(t, err, ErrISDSUnavailable)
}

func TestErrors_ContextCanceled(t *testing.T) {
	srv, _ := newFake(t, http.StatusOK, fixture(t, "get_owner_info_response.xml"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := newTestClient(t, srv).GetOwnerInfo(ctx)
	wantIs(t, err, context.Canceled)
}

func TestErrors_ConnectionRefused(t *testing.T) {
	srv, _ := newFake(t, http.StatusOK, nil)
	c := newTestClient(t, srv)
	srv.Close()
	_, err := c.GetOwnerInfo(context.Background())
	wantIs(t, err, ErrISDSUnavailable)
}

func TestRedirectIsRefused(t *testing.T) {
	target, targetRec := newFake(t, http.StatusOK, fixture(t, "get_owner_info_response.xml"))
	srv, _ := newFakeFunc(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/DS/DsManage", http.StatusFound)
	})

	_, err := newTestClient(t, srv).GetOwnerInfo(context.Background())
	wantIs(t, err, ErrUnexpectedResponse)
	if n := targetRec.count(); n != 0 {
		t.Errorf("the redirect target received %d request(s), want 0", n)
	}
}

func TestWithHTTPClient_RedirectsStayDisabledAndInputIsNotMutated(t *testing.T) {
	target, targetRec := newFake(t, http.StatusOK, fixture(t, "get_owner_info_response.xml"))
	srv, _ := newFakeFunc(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	})
	hc := &http.Client{Timeout: time.Second}

	_, err := newTestClient(t, srv, WithHTTPClient(hc)).GetOwnerInfo(context.Background())
	wantIs(t, err, ErrUnexpectedResponse)
	if targetRec.count() != 0 {
		t.Error("redirect was followed")
	}
	if hc.CheckRedirect != nil {
		t.Error("the caller's http.Client was modified")
	}
}

func TestResponseNamespacePrefixesAreTolerated(t *testing.T) {
	srv, _ := newFake(t, http.StatusOK, fixture(t, "get_owner_info_response_prefixed.xml"))
	info, err := newTestClient(t, srv).GetOwnerInfo(context.Background())
	if err != nil {
		t.Fatalf("GetOwnerInfo: %v", err)
	}
	if info.BoxID != "abc1234" || info.BoxType != "PFO" || info.FirmName != "Fake Firm" {
		t.Errorf("info = %+v", info)
	}
}

// captureLogs routes the default slog logger into a buffer at debug level.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func TestSecretsNeverLeak(t *testing.T) {
	logs := captureLogs(t)
	var seen []string

	record := func(err error) {
		if err != nil {
			seen = append(seen, err.Error(), fmt.Sprintf("%+v", err), fmt.Sprintf("%#v", err))
		}
	}

	scenarios := []struct {
		status int
		body   []byte
	}{
		{http.StatusOK, fixture(t, "get_owner_info_response.xml")},
		{http.StatusOK, fixture(t, "get_owner_info_response_status_error.xml")},
		{http.StatusOK, fixture(t, "maintenance.html")},
		{http.StatusOK, fixture(t, "fault.xml")},
		{http.StatusUnauthorized, nil},
		{http.StatusInternalServerError, []byte("boom")},
	}
	for _, s := range scenarios {
		srv, _ := newFake(t, s.status, s.body)
		c := newTestClient(t, srv)
		_, err := c.GetOwnerInfo(context.Background())
		record(err)
		_, err = c.CreateMessage(context.Background(), validMessage())
		record(err)
	}
	// Failures without a server: refused connection and invalid input.
	srv, _ := newFake(t, http.StatusOK, nil)
	c := newTestClient(t, srv)
	srv.Close()
	_, err := c.GetOwnerInfo(context.Background())
	record(err)
	_, err = c.CreateMessage(context.Background(), MessageInput{})
	record(err)

	creds := Credentials{Username: testUsername, Password: testPassword}
	asJSON, _ := json.Marshal(struct{ C Credentials }{creds})
	var structLog bytes.Buffer
	slog.New(slog.NewJSONHandler(&structLog, nil)).Info("login", "creds", creds)
	seen = append(seen,
		creds.String(), fmt.Sprintf("%v", creds), fmt.Sprintf("%+v", creds), fmt.Sprintf("%#v", creds),
		fmt.Sprintf("%v", c), fmt.Sprintf("%+v", c),
		string(asJSON), structLog.String(), logs.String(),
	)

	if logs.Len() == 0 {
		t.Fatal("no log output captured; the scan would prove nothing")
	}
	for i, s := range seen {
		if strings.Contains(s, testPassword) || strings.Contains(s, testUsername) {
			t.Errorf("output %d contains a credential: %q", i, s)
		}
	}
}
