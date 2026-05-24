package handler

import (
	"context"
	"errors"

	"github.com/zajca/zfaktury/internal/domain"
)

type companyCtxKey struct{}

// CompanyFromContext retrieves the active company loaded by WithCompany.
// Returns an error only if called from a handler not mounted under
// the per-company subrouter — that's a programming error.
func CompanyFromContext(ctx context.Context) (*domain.Company, error) {
	c, ok := ctx.Value(companyCtxKey{}).(*domain.Company)
	if !ok {
		return nil, errors.New("no company in context (handler not mounted under WithCompany?)")
	}
	return c, nil
}

func contextWithCompany(ctx context.Context, c *domain.Company) context.Context {
	return context.WithValue(ctx, companyCtxKey{}, c)
}
