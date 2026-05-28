package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/zajca/zfaktury/internal/companyctx"
	"github.com/zajca/zfaktury/internal/domain"
)

// CompanyResolver is the minimum CompanyService surface the middleware needs.
type CompanyResolver interface {
	Get(ctx context.Context, id int64) (domain.Company, error)
}

// WithCompany resolves {companyID} from the URL, validates the company exists
// and is not soft-deleted, stores it in the request context, and sets the
// X-Company-Id response header for client-side race-condition detection.
func WithCompany(svc CompanyResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := chi.URLParam(r, "companyID")
			id, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || id <= 0 {
				http.Error(w, "invalid company id", http.StatusBadRequest)
				return
			}
			c, err := svc.Get(r.Context(), id)
			if err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					http.Error(w, "company not found", http.StatusNotFound)
					return
				}
				http.Error(w, "internal error resolving company", http.StatusInternalServerError)
				return
			}
			w.Header().Set("X-Company-Id", strconv.FormatInt(c.ID, 10))
			ctx := contextWithCompany(r.Context(), &c)
			ctx = companyctx.WithCompanyID(ctx, c.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
