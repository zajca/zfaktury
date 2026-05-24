package repository

import (
	"testing"
)

// TestCrossCompanyLeakDetection exhaustively tries every per-company
// repository's Get/List/Update/Delete with a wrong companyID and
// asserts ErrNotFound (or empty list). Populated incrementally
// across Phase 3 tasks as each repo gains the companyID parameter.
func TestCrossCompanyLeakDetection(t *testing.T) {
	t.Skip("populated as repos gain companyID parameter in Phase 3")
}
