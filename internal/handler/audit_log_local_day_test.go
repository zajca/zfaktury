package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/testutil"
)

// The from/to day bounds are local days: an entry at 00:30 local time belongs
// to that local day even though it is still the previous day in UTC.
func TestAuditLogHandler_List_DayBoundsAreLocalDays(t *testing.T) {
	origLocal := time.Local
	time.Local = time.FixedZone("CET", 3600)
	t.Cleanup(func() { time.Local = origLocal })

	db := testutil.NewTestDB(t)
	h := NewAuditLogHandler(service.NewAuditService(repository.NewAuditLogRepository(db)))

	entries := []struct {
		id        int
		createdAt string
	}{
		{1, "2026-02-28T23:30:00+01:00"}, // previous local day
		{2, "2026-03-01T00:30:00+01:00"}, // first minutes of the local day, previous day in UTC
		{3, "2026-03-01T23:30:00+01:00"}, // last minutes of the local day
		{4, "2026-03-02T00:30:00+01:00"}, // next local day
	}
	for _, e := range entries {
		if _, err := db.ExecContext(context.Background(), `
			INSERT INTO audit_log (entity_type, entity_id, action, old_values, new_values, created_at)
			VALUES ('invoice', ?, 'create', '', '', ?)`, e.id, e.createdAt); err != nil {
			t.Fatalf("inserting audit entry %d: %v", e.id, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/audit-log?from=2026-03-01&to=2026-03-01", nil)
	w := serveAuditLog(h, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp auditLogListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	got := map[int64]bool{}
	for _, item := range resp.Items {
		got[item.EntityID] = true
	}
	if len(resp.Items) != 2 || !got[2] || !got[3] {
		t.Errorf("entity IDs listed = %v, want exactly 2 and 3", got)
	}
}
