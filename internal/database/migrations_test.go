package database

import (
	"testing"
)

// TestMultiCompanyMigration validates that migration 025 transforms
// a single-company v024 database into a multi-company v025 database
// with no data loss. Populated incrementally across Phase 2 tasks.
func TestMultiCompanyMigration(t *testing.T) {
	t.Skip("populated in Phase 2 tasks 5-15")
}

// TestMultiCompanyMigrationProductionSized runs migration 025 against
// a synthetic ~5k-invoice fixture and asserts completion under 30s.
// Gated behind -tags integration so it doesn't run on every CI build.
func TestMultiCompanyMigrationProductionSized(t *testing.T) {
	t.Skip("populated in Phase 2 task 15")
}
