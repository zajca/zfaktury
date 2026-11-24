package desktop

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewInternalRequestCarriesTheWebViewHost(t *testing.T) {
	req := newInternalRequest("/api/v1/filings/submissions/3/receipt.zfo")

	if req.Method != http.MethodGet || req.URL.Path != "/api/v1/filings/submissions/3/receipt.zfo" {
		t.Fatalf("request = %s %s", req.Method, req.URL.Path)
	}
	if req.Host != "wails.localhost" {
		t.Fatalf("Host = %q, want wails.localhost", req.Host)
	}

	// A host guard like the one on the filing routes lets it through.
	guard := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Host, "wails.localhost") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	guard.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}
