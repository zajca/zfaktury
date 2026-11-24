package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zalando/go-keyring"

	"github.com/zajca/zfaktury/internal/secrets"
	"github.com/zajca/zfaktury/internal/service"
)

const (
	testISDSUsername = "dsuser-qq42"
	testISDSPassword = "Zx9-Distinct-Pw-7q!"
	testISDSPath     = "/api/v1/settings/isds-credentials"
)

// TestMain installs the in-memory keyring so no handler test can reach the OS keyring.
func TestMain(m *testing.M) {
	keyring.MockInit()
	m.Run()
}

// newISDSRouter builds the real application router with only the credentials
// service wired, so the global middleware (logging, dev CORS) is in the path.
func newISDSRouter(t *testing.T, devMode bool, bindHost string) *chi.Mux {
	t.Helper()
	keyring.MockInit()
	t.Cleanup(keyring.MockInit)

	svc := service.NewISDSCredentialsService(secrets.NewISDSStore())
	return NewRouter(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		svc, nil,
		RouterConfig{DevMode: devMode, DataDir: t.TempDir(), BindHost: bindHost},
	)
}

// captureLogs routes slog output at debug level into a buffer for the test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

type isdsReq struct {
	method      string
	path        string
	body        string
	contentType string
	confirm     string
	host        string
	origin      string
}

func (q isdsReq) do(r http.Handler) *httptest.ResponseRecorder {
	path := q.path
	if path == "" {
		path = testISDSPath
	}
	req := httptest.NewRequest(q.method, path, strings.NewReader(q.body))
	req.Host = "127.0.0.1:8080"
	if q.host != "" {
		req.Host = q.host
	}
	if q.contentType != "" {
		req.Header.Set("Content-Type", q.contentType)
	}
	if q.confirm != "" {
		req.Header.Set("X-ZFaktury-Confirm", q.confirm)
	}
	if q.origin != "" {
		req.Header.Set("Origin", q.origin)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func putBody(env, username, password string) string {
	b, _ := json.Marshal(map[string]string{"environment": env, "username": username, "password": password})
	return string(b)
}

func validPut(env string) isdsReq {
	return isdsReq{
		method:      http.MethodPut,
		body:        putBody(env, testISDSUsername, testISDSPassword),
		contentType: "application/json",
		confirm:     "1",
	}
}

func assertNoSecret(t *testing.T, what, text string) {
	t.Helper()
	if strings.Contains(text, testISDSPassword) {
		t.Errorf("%s contains the password: %s", what, text)
	}
}

// assertNoLoginInLogs fails when captured log output carries the password or
// the username; responses may legitimately echo the username, logs may not.
func assertNoLoginInLogs(t *testing.T, logs string) {
	t.Helper()
	assertNoSecret(t, "log output", logs)
	if strings.Contains(logs, testISDSUsername) {
		t.Errorf("log output contains the username: %s", logs)
	}
}

func TestISDSCredentials_Lifecycle(t *testing.T) {
	logs := captureLogs(t)
	r := newISDSRouter(t, false, "127.0.0.1")

	// Nothing stored yet.
	w := isdsReq{method: http.MethodGet}.do(r)
	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d, body: %s", w.Code, w.Body.String())
	}
	var status map[string]isdsStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("decoding GET: %v", err)
	}
	if len(status) != 2 || status["test"].Configured || status["production"].Configured {
		t.Fatalf("initial status = %+v", status)
	}

	// Save for test.
	w = validPut("test").do(r)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body: %s", w.Code, w.Body.String())
	}
	assertNoSecret(t, "PUT response", w.Body.String())

	// The login is in the keyring under the documented keys.
	if got, err := keyring.Get("zfaktury", "isds/test/password"); err != nil || got != testISDSPassword {
		t.Errorf("keyring password = %q, %v", got, err)
	}

	w = isdsReq{method: http.MethodGet}.do(r)
	assertNoSecret(t, "GET response", w.Body.String())
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("decoding GET: %v", err)
	}
	if !status["test"].Configured || status["test"].Username != testISDSUsername {
		t.Errorf("test status = %+v", status["test"])
	}
	if status["production"].Configured {
		t.Errorf("production must stay unconfigured: %+v", status["production"])
	}

	// The GET body has exactly the two documented fields per environment.
	var raw map[string]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	for env, fields := range raw {
		if len(fields) != 2 {
			t.Errorf("%s has fields %v, want only configured and username", env, fields)
		}
	}

	// Delete removes both entries.
	w = isdsReq{method: http.MethodDelete, path: testISDSPath + "?environment=test", contentType: "application/json", confirm: "1"}.do(r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE status = %d, body: %s", w.Code, w.Body.String())
	}
	for _, field := range []string{"username", "password"} {
		if _, err := keyring.Get("zfaktury", "isds/test/"+field); !errors.Is(err, keyring.ErrNotFound) {
			t.Errorf("%s still in keyring after delete: %v", field, err)
		}
	}

	// Deleting again is not an error.
	w = isdsReq{method: http.MethodDelete, path: testISDSPath + "?environment=test", contentType: "application/json", confirm: "1"}.do(r)
	if w.Code != http.StatusNoContent {
		t.Errorf("second DELETE status = %d", w.Code)
	}

	// Neither the password nor the username appears in any log line.
	assertNoLoginInLogs(t, logs.String())
}

func TestISDSCredentials_PutReturnsTrimmedUsername(t *testing.T) {
	r := newISDSRouter(t, false, "127.0.0.1")
	q := validPut("test")
	q.body = putBody("test", "  "+testISDSUsername+" \t", testISDSPassword)

	w := q.do(r)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body: %s", w.Code, w.Body.String())
	}
	var got isdsStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Username != testISDSUsername {
		t.Errorf("PUT username = %q, want the trimmed %q", got.Username, testISDSUsername)
	}
	if stored, err := keyring.Get("zfaktury", "isds/test/username"); err != nil || stored != got.Username {
		t.Errorf("stored username = %q, %v; the response must match it", stored, err)
	}
}

func TestISDSCredentials_ChangingStateRequiresContentTypeAndConfirmHeader(t *testing.T) {
	r := newISDSRouter(t, false, "127.0.0.1")

	tests := []struct {
		name   string
		req    isdsReq
		status int
	}{
		{"PUT without confirm header", withConfirm(validPut("test"), ""), http.StatusForbidden},
		{"PUT with wrong confirm value", withConfirm(validPut("test"), "yes"), http.StatusForbidden},
		{"PUT without content type", withContentType(validPut("test"), ""), http.StatusUnsupportedMediaType},
		{"PUT as form post", withContentType(validPut("test"), "application/x-www-form-urlencoded"), http.StatusUnsupportedMediaType},
		{"PUT as text/plain", withContentType(validPut("test"), "text/plain"), http.StatusUnsupportedMediaType},
		{"DELETE without confirm header", isdsReq{method: http.MethodDelete, path: testISDSPath + "?environment=test", contentType: "application/json"}, http.StatusForbidden},
		{"DELETE without content type", isdsReq{method: http.MethodDelete, path: testISDSPath + "?environment=test", confirm: "1"}, http.StatusUnsupportedMediaType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := tt.req.do(r)
			if w.Code != tt.status {
				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.status, w.Body.String())
			}
			assertNoSecret(t, "response", w.Body.String())
		})
	}

	if _, err := keyring.Get("zfaktury", "isds/test/username"); !errors.Is(err, keyring.ErrNotFound) {
		t.Errorf("a refused request still stored a login: %v", err)
	}

	// A JSON content type with a charset parameter is accepted.
	ok := withContentType(validPut("test"), "application/json; charset=utf-8").do(r)
	if ok.Code != http.StatusOK {
		t.Errorf("PUT with charset status = %d, body: %s", ok.Code, ok.Body.String())
	}
}

func withConfirm(q isdsReq, v string) isdsReq     { q.confirm = v; return q }
func withContentType(q isdsReq, v string) isdsReq { q.contentType = v; return q }

func TestISDSCredentials_RefusesNonLoopbackBind(t *testing.T) {
	for _, bind := range []string{"0.0.0.0", "192.168.1.20", "::", "example.com", ""} {
		t.Run("bind "+bind, func(t *testing.T) {
			r := newISDSRouter(t, false, bind)
			for _, q := range []isdsReq{
				{method: http.MethodGet},
				validPut("test"),
				{method: http.MethodDelete, path: testISDSPath + "?environment=test", contentType: "application/json", confirm: "1"},
			} {
				w := q.do(r)
				if w.Code != http.StatusForbidden {
					t.Errorf("%s status = %d, want 403, body: %s", q.method, w.Code, w.Body.String())
				}
				if !strings.Contains(w.Body.String(), "127.0.0.1") {
					t.Errorf("%s: expected the Czech loopback explanation, got %s", q.method, w.Body.String())
				}
			}
			if _, err := keyring.Get("zfaktury", "isds/test/username"); !errors.Is(err, keyring.ErrNotFound) {
				t.Errorf("a refused request stored a login: %v", err)
			}
		})
	}

	for _, bind := range []string{"127.0.0.1", "::1", "localhost"} {
		r := newISDSRouter(t, false, bind)
		if w := (isdsReq{method: http.MethodGet}).do(r); w.Code != http.StatusOK {
			t.Errorf("bind %s: GET status = %d", bind, w.Code)
		}
	}
}

func TestISDSCredentials_RefusesForeignHostHeader(t *testing.T) {
	r := newISDSRouter(t, false, "127.0.0.1")
	for _, host := range []string{"evil.example", "127.0.0.1.evil.example", "192.168.1.20:8080", ""} {
		q := validPut("test")
		q.host = host
		if host == "" {
			// httptest fills an empty Host with example.com, which is foreign too.
			q.host = "example.com"
		}
		if w := q.do(r); w.Code != http.StatusForbidden {
			t.Errorf("Host %q: status = %d, want 403", host, w.Code)
		}
	}
	for _, host := range []string{"127.0.0.1:8080", "localhost:5173", "localhost", "[::1]:8080", "wails.localhost"} {
		q := validPut("test")
		q.host = host
		if w := q.do(r); w.Code != http.StatusOK {
			t.Errorf("Host %q: status = %d, want 200, body: %s", host, w.Code, w.Body.String())
		}
	}
}

func TestISDSCredentials_KeyringUnavailable(t *testing.T) {
	logs := captureLogs(t)
	r := newISDSRouter(t, false, "127.0.0.1")
	keyring.MockInitWithError(errors.New("dial unix: no secret service"))

	for _, q := range []isdsReq{
		{method: http.MethodGet},
		validPut("test"),
		{method: http.MethodDelete, path: testISDSPath + "?environment=test", contentType: "application/json", confirm: "1"},
	} {
		w := q.do(r)
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s status = %d, want 503, body: %s", q.method, w.Code, w.Body.String())
		}
		var body errorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decoding error body: %v", err)
		}
		if !strings.Contains(body.Error, "klíčenka") {
			t.Errorf("%s: expected the Czech keyring message, got %q", q.method, body.Error)
		}
		if strings.Contains(body.Error, "dial unix") {
			t.Errorf("%s: backend error text leaked: %q", q.method, body.Error)
		}
		assertNoSecret(t, q.method+" response", w.Body.String())
	}
	assertNoLoginInLogs(t, logs.String())
}

func TestISDSCredentials_KeyringLocked(t *testing.T) {
	logs := captureLogs(t)
	r := newISDSRouter(t, false, "127.0.0.1")
	keyring.MockInitWithError(errors.New("failed to unlock correct collection '/org/freedesktop/secrets/aliases/login'"))

	for _, q := range []isdsReq{{method: http.MethodGet}, validPut("test")} {
		w := q.do(r)
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("%s status = %d, want 503, body: %s", q.method, w.Code, w.Body.String())
		}
		var body errorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decoding error body: %v", err)
		}
		if !strings.Contains(body.Error, "zamčená") {
			t.Errorf("%s: expected the locked-keyring message, got %q", q.method, body.Error)
		}
		if strings.Contains(body.Error, "Spusťte službu") || strings.Contains(body.Error, "failed to unlock") {
			t.Errorf("%s: wrong or leaking message: %q", q.method, body.Error)
		}
	}
	assertNoLoginInLogs(t, logs.String())
}

func TestISDSCredentials_Validation(t *testing.T) {
	r := newISDSRouter(t, false, "127.0.0.1")

	tests := []struct {
		name string
		body string
	}{
		{"empty username", putBody("test", "", testISDSPassword)},
		{"empty password", putBody("test", testISDSUsername, "")},
		{"missing environment", putBody("", testISDSUsername, testISDSPassword)},
		{"unknown environment", putBody("staging", testISDSUsername, testISDSPassword)},
		{"username too long", putBody("test", strings.Repeat("u", secrets.MaxISDSUsernameLen+1), testISDSPassword)},
		{"password too long", putBody("test", testISDSUsername, strings.Repeat("p", secrets.MaxISDSPasswordLen+1))},
		{"unknown field", `{"environment":"test","username":"u","password":"p","extra":1}`},
		{"not JSON", "environment=test"},
		{"oversized body", `{"environment":"test","username":"u","password":"` + strings.Repeat("p", 8192) + `"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := validPut("test")
			q.body = tt.body
			w := q.do(r)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400, body: %s", w.Code, w.Body.String())
			}
			assertNoSecret(t, "response", w.Body.String())
		})
	}

	if w := (isdsReq{method: http.MethodDelete, contentType: "application/json", confirm: "1"}).do(r); w.Code != http.StatusBadRequest {
		t.Errorf("DELETE without environment status = %d, want 400", w.Code)
	}
	if w := (isdsReq{method: http.MethodDelete, path: testISDSPath + "?environment=staging", contentType: "application/json", confirm: "1"}).do(r); w.Code != http.StatusBadRequest {
		t.Errorf("DELETE with unknown environment status = %d, want 400", w.Code)
	}
}

func TestISDSCredentials_DevCORSDoesNotApply(t *testing.T) {
	r := newISDSRouter(t, true, "127.0.0.1")

	// Control: dev CORS still answers on an ordinary API route.
	control := isdsReq{method: http.MethodOptions, path: "/api/v1/settings", origin: "http://localhost:5173"}.do(r)
	if control.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("dev CORS control failed: status %d, headers %v", control.Code, control.Header())
	}

	preflight := isdsReq{method: http.MethodOptions, origin: "http://evil.example"}.do(r)
	if preflight.Code == http.StatusNoContent {
		t.Errorf("preflight was answered with 204")
	}
	for _, h := range []string{"Access-Control-Allow-Origin", "Access-Control-Allow-Methods", "Access-Control-Allow-Headers"} {
		if v := preflight.Header().Get(h); v != "" {
			t.Errorf("preflight carries %s: %q", h, v)
		}
	}

	for _, q := range []isdsReq{{method: http.MethodGet, origin: "http://evil.example"}, withOrigin(validPut("test"), "http://evil.example")} {
		w := q.do(r)
		if v := w.Header().Get("Access-Control-Allow-Origin"); v != "" {
			t.Errorf("%s carries Access-Control-Allow-Origin: %q", q.method, v)
		}
	}
}

func withOrigin(q isdsReq, v string) isdsReq { q.origin = v; return q }

func TestIsLocalRequestHost(t *testing.T) {
	for host, want := range map[string]bool{
		"127.0.0.1":         true,
		"127.0.0.1:5173":    true,
		"localhost":         true,
		"LOCALHOST:8080":    true,
		"[::1]:8080":        true,
		"::1":               true,
		"wails.localhost":   true,
		"127.0.0.2":         true,
		"example.com":       false,
		"localhost.evil.io": false,
		"0.0.0.0":           false,
		"10.0.0.1:80":       false,
		"":                  false,
	} {
		if got := isLocalRequestHost(host); got != want {
			t.Errorf("isLocalRequestHost(%q) = %v, want %v", host, got, want)
		}
	}
}
