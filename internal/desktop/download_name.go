package desktop

import (
	"mime"
	"path/filepath"
)

// downloadName returns the filename announced by a Content-Disposition header,
// reduced to its base name, or fallback when the header carries none.
func downloadName(contentDisposition, fallback string) string {
	if contentDisposition == "" {
		return fallback
	}
	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil || params["filename"] == "" {
		return fallback
	}
	name := filepath.Base(params["filename"])
	if name == "." || name == string(filepath.Separator) {
		return fallback
	}
	return name
}
