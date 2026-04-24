package handler

import (
	"fmt"
	"log/slog"
	"mime"
	"net/http"
)

// DownloadBundle handles GET /api/v1/income-tax-returns/{id}/bundle.
//
// The response is a ZIP archive containing the generated DPFO XML plus all
// mandatory tax deduction proof documents (prilohy per §15). Investment
// statements are included when the bundle service is configured with an
// investment document service.
//
// If the bundle service is not configured on the handler, the endpoint
// responds with 501 Not Implemented (mirroring the pattern used by the OCR
// extraction endpoints).
func (h *IncomeTaxHandler) DownloadBundle(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid income tax return ID")
		return
	}

	if h.bundleSvc == nil {
		respondError(w, http.StatusNotImplemented, "income tax bundle generation is not configured")
		return
	}

	data, filename, err := h.bundleSvc.GenerateBundle(r.Context(), id)
	if err != nil {
		slog.Error("failed to generate income tax return bundle", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
