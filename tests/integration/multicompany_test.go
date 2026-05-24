//go:build integration

package integration

import (
	"testing"
)

// TestMultiCompanyEndToEnd exercises the full multi-company flow:
// create two companies, contacts, invoices, cross-company rejection,
// sequence collision, delete protection, soft-delete after cleanup.
// Populated in Phase 3 task 22 once the API surface is complete.
func TestMultiCompanyEndToEnd(t *testing.T) {
	t.Skip("populated in Phase 3 task 22")
}
