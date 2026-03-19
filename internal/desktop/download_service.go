//go:build !server

package desktop

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// DownloadService handles file downloads in the native desktop window.
// WebKitGTK in Wails v3 silently ignores blob: URL downloads and <a download> links,
// so this service performs the HTTP request internally and uses a native GTK save dialog.
type DownloadService struct {
	Handler http.Handler
}

// SaveToFile fetches apiPath via the internal router, opens a native save dialog
// with suggestedName, and writes the response body to the selected path.
// Returns nil if the user cancels the dialog.
func (s *DownloadService) SaveToFile(apiPath, suggestedName string) error {
	req := httptest.NewRequest(http.MethodGet, apiPath, nil)
	rec := httptest.NewRecorder()
	s.Handler.ServeHTTP(rec, req)

	result := rec.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", result.StatusCode)
	}

	body := rec.Body.Bytes()

	dialog := application.Get().Dialog.SaveFile()
	dialog.SetFilename(suggestedName)

	ext := strings.TrimPrefix(filepath.Ext(suggestedName), ".")
	if ext != "" {
		dialog.AddFilter(strings.ToUpper(ext)+" files", "*."+ext)
	}
	dialog.AddFilter("All files", "*.*")

	selectedPath, err := dialog.PromptForSingleSelection()
	if err != nil {
		return fmt.Errorf("save dialog: %w", err)
	}
	if selectedPath == "" {
		return nil // user cancelled
	}

	if err := os.WriteFile(selectedPath, body, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}
