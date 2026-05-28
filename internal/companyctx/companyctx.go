// Package companyctx stashes the active company id on a request context.
// Both the handler middleware (which puts the full domain.Company in context
// via its own key) and downstream services (which only need the id, e.g. the
// audit logger) pass through this shared helper so the same value is
// available everywhere without import cycles.
package companyctx

import "context"

type contextKey struct{}

// WithCompanyID returns a new context carrying the given company id.
func WithCompanyID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// CompanyIDFromContext extracts the company id stored by WithCompanyID.
// Returns (0, false) when no id is present — services that record audit
// trails should treat that as a system-level event with no company.
func CompanyIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(contextKey{}).(int64)
	return id, ok
}
