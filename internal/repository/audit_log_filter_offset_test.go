package repository

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

// An entry written at 2026-03-01T00:30:00+02:00 happened at 22:30 UTC on 28 February.
func TestAuditLogRepository_List_ComparesAcrossOffsets(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO audit_log (entity_type, entity_id, action, old_values, new_values, created_at)
		VALUES ('invoice', 1, 'create', '', '', '2026-03-01T00:30:00+02:00')`); err != nil {
		t.Fatalf("inserting audit entry: %v", err)
	}

	tests := []struct {
		name   string
		filter domain.AuditLogFilter
		want   int
	}{
		{
			name:   "upper bound after the entry in UTC",
			filter: domain.AuditLogFilter{To: time.Date(2026, 2, 28, 23, 30, 0, 0, time.UTC)},
			want:   1,
		},
		{
			name:   "upper bound before the entry in UTC",
			filter: domain.AuditLogFilter{To: time.Date(2026, 2, 28, 22, 0, 0, 0, time.UTC)},
			want:   0,
		},
		{
			name:   "lower bound after the entry in UTC",
			filter: domain.AuditLogFilter{From: time.Date(2026, 2, 28, 23, 0, 0, 0, time.UTC)},
			want:   0,
		},
		{
			name:   "lower bound before the entry in UTC",
			filter: domain.AuditLogFilter{From: time.Date(2026, 2, 28, 22, 0, 0, 0, time.UTC)},
			want:   1,
		},
		{
			name: "bound given in a non-UTC zone",
			filter: domain.AuditLogFilter{
				From: time.Date(2026, 3, 1, 0, 0, 0, 0, time.FixedZone("CEST", 2*3600)),
				To:   time.Date(2026, 3, 1, 1, 0, 0, 0, time.FixedZone("CEST", 2*3600)),
			},
			want: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entries, total, err := repo.List(ctx, tc.filter)
			if err != nil {
				t.Fatalf("List() error: %v", err)
			}
			if len(entries) != tc.want || total != tc.want {
				t.Errorf("got %d entries (total %d), want %d", len(entries), total, tc.want)
			}
		})
	}
}
