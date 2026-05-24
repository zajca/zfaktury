package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/zajca/zfaktury/internal/domain"
)

type stubCompanyResolver struct {
	companies map[int64]domain.Company
}

func (s *stubCompanyResolver) Get(_ context.Context, id int64) (domain.Company, error) {
	c, ok := s.companies[id]
	if !ok {
		return domain.Company{}, domain.ErrNotFound
	}
	return c, nil
}

func mountWithCompany(t *testing.T, svc CompanyResolver) http.Handler {
	r := chi.NewRouter()
	r.Route("/api/v1/companies/{companyID}", func(r chi.Router) {
		r.Use(WithCompany(svc))
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			c, err := CompanyFromContext(r.Context())
			if err != nil {
				t.Errorf("CompanyFromContext: %v", err)
				w.WriteHeader(500)
				return
			}
			w.Header().Set("X-Returned-Company", strconv.FormatInt(c.ID, 10))
			w.WriteHeader(200)
		})
	})
	return r
}

func TestWithCompany_HappyPath(t *testing.T) {
	svc := &stubCompanyResolver{companies: map[int64]domain.Company{1: {ID: 1, Name: "A"}}}
	h := mountWithCompany(t, svc)

	req := httptest.NewRequest("GET", "/api/v1/companies/1/ping", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Errorf("code = %d, want 200", rr.Code)
	}
	if rr.Header().Get("X-Company-Id") != "1" {
		t.Errorf("X-Company-Id = %q, want 1", rr.Header().Get("X-Company-Id"))
	}
	if rr.Header().Get("X-Returned-Company") != "1" {
		t.Errorf("downstream handler did not receive company")
	}
}

func TestWithCompany_NotFound(t *testing.T) {
	svc := &stubCompanyResolver{companies: map[int64]domain.Company{}}
	h := mountWithCompany(t, svc)
	req := httptest.NewRequest("GET", "/api/v1/companies/42/ping", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 404 {
		t.Errorf("code = %d, want 404", rr.Code)
	}
}

func TestWithCompany_NonNumeric(t *testing.T) {
	svc := &stubCompanyResolver{companies: map[int64]domain.Company{}}
	h := mountWithCompany(t, svc)
	req := httptest.NewRequest("GET", "/api/v1/companies/abc/ping", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 400 {
		t.Errorf("code = %d, want 400", rr.Code)
	}
}
