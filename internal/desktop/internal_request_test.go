package desktop

import (
	"net/http"
	"testing"

	"github.com/zajca/zfaktury/internal/handler"
)

func TestNewInternalRequestPassesTheLocalHostGuard(t *testing.T) {
	req := newInternalRequest("/api/v1/filings/submissions/3/receipt.zfo")

	if req.Method != http.MethodGet || req.URL.Path != "/api/v1/filings/submissions/3/receipt.zfo" {
		t.Fatalf("request = %s %s", req.Method, req.URL.Path)
	}
	if !handler.IsLocalRequestHost(req.Host) {
		t.Errorf("Host %q is refused by the local-host guard", req.Host)
	}
}
