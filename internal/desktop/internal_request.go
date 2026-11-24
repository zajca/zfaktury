package desktop

import (
	"net/http"
	"net/http/httptest"
)

// internalRequestHost is the host the WebView uses for the app. The local-host
// guard on the filing routes accepts it, so an internal request must carry it.
const internalRequestHost = "wails.localhost"

// newInternalRequest builds the GET the download service serves through the
// router. httptest requests default to the host example.com, which the
// local-host guard of the filing routes refuses with 403.
func newInternalRequest(apiPath string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, apiPath, nil)
	req.Host = internalRequestHost
	return req
}
