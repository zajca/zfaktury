//go:build !server

package desktop

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/zajca/zfaktury/internal/epo"
)

// BrowserService opens EPO form links in the system browser. The WebView can
// call any bound method, so the service validates the link itself instead of
// trusting the caller.
type BrowserService struct {
	// AllowedHost is the only host a link may point to.
	AllowedHost string
}

// OpenExternal opens rawURL in the default browser when it is an https link to
// the allowed host under the EPO form path.
func (s *BrowserService) OpenExternal(rawURL string) error {
	if err := epo.ValidateFormURL(rawURL, s.AllowedHost); err != nil {
		return fmt.Errorf("refusing to open link: %w", err)
	}
	if err := application.Get().Browser.OpenURL(rawURL); err != nil {
		return fmt.Errorf("opening link in the system browser: %w", err)
	}
	return nil
}
