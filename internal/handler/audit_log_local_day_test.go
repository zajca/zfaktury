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
	t.Run("fixed zone", func(t *testing.T) {
		origLocal := time.Local
		time.Local = time.FixedZone("CET", 3600)
		t.Cleanup(func() { time.Local = origLocal })

		entries := []struct {
			id        int
			createdAt string
		}{
			{1, "2026-02-28T23:30:00+01:00"}, // previous local day
			{2, "2026-03-01T00:30:00+01:00"}, // first minutes of the local day, previous day in UTC
			{3, "2026-03-01T23:30:00+01:00"}, // last minutes of the local day
			{4, "2026-03-02T00:30:00+01:00"}, // next local day
		}
		got := listAuditLogDay(t, "2026-03-01", "2026-03-01", entries)
		if len(got) != 2 || !got[2] || !got[3] {
			t.Errorf("entity IDs listed = %v, want exactly 2 and 3", got)
		}
	})

	// 2026-03-29 is the Europe/Prague DST change day: at 02:00 the clock
	// jumps to 03:00, so the local day is 23 hours long. A flat 24-hour
	// "next local midnight" bound would leak the first hour of the next
	// local day into the result.
	t.Run("Europe/Prague on the DST change day", func(t *testing.T) {
		prague, err := time.LoadLocation("Europe/Prague")
		if err != nil {
			t.Fatalf("loading Europe/Prague: %v", err)
		}
		origLocal := time.Local
		time.Local = prague
		t.Cleanup(func() { time.Local = origLocal })

		entries := []struct {
			id        int
			createdAt string
		}{
			{1, "2026-03-28T23:30:00+01:00"}, // previous local day
			{2, "2026-03-29T00:30:00+01:00"}, // first minutes of the local day, previous day in UTC
			{3, "2026-03-29T03:30:00+02:00"}, // after the 02:00 -> 03:00 jump
			{4, "2026-03-29T23:30:00+02:00"}, // last minutes of the 23-hour day
			{5, "2026-03-30T00:30:00+02:00"}, // next local day; a flat 24-hour bound would wrongly include it
		}
		got := listAuditLogDay(t, "2026-03-29", "2026-03-29", entries)
		if len(got) != 3 || !got[2] || !got[3] || !got[4] {
			t.Errorf("entity IDs listed = %v, want exactly 2, 3 and 4", got)
		}
	})

	// 2026-10-25 is the other Europe/Prague DST change day: at 03:00 the
	// clock falls back to 02:00, so the local day is 25 hours long. A flat
	// 24-hour "next local midnight" bound would end at 23:00 local and
	// wrongly drop the last hour of the local day from the result.
	t.Run("Europe/Prague on the autumn DST change day", func(t *testing.T) {
		prague, err := time.LoadLocation("Europe/Prague")
		if err != nil {
			t.Fatalf("loading Europe/Prague: %v", err)
		}
		origLocal := time.Local
		time.Local = prague
		t.Cleanup(func() { time.Local = origLocal })

		entries := []struct {
			id        int
			createdAt string
		}{
			{1, "2026-10-24T23:30:00+02:00"}, // previous local day
			{2, "2026-10-25T00:30:00+02:00"}, // first minutes of the local day, previous day in UTC
			{3, "2026-10-25T02:30:00+02:00"}, // before the 03:00 -> 02:00 fallback
			{4, "2026-10-25T02:30:00+01:00"}, // the repeated hour after the fallback
			{5, "2026-10-25T23:30:00+01:00"}, // last minutes of the 25-hour day; a flat 24-hour bound would wrongly exclude it
			{6, "2026-10-26T00:30:00+01:00"}, // next local day
		}
		got := listAuditLogDay(t, "2026-10-25", "2026-10-25", entries)
		if len(got) != 4 || !got[2] || !got[3] || !got[4] || !got[5] {
			t.Errorf("entity IDs listed = %v, want exactly 2, 3, 4 and 5", got)
		}
	})
}

// listAuditLogDay seeds the given audit entries and returns the entity IDs the
// handler lists for the from/to day.
func listAuditLogDay(t *testing.T, from, to string, entries []struct {
	id        int
	createdAt string
}) map[int64]bool {
	t.Helper()

	db := testutil.NewTestDB(t)
	h := NewAuditLogHandler(service.NewAuditService(repository.NewAuditLogRepository(db)))

	for _, e := range entries {
		if _, err := db.ExecContext(context.Background(), `
			INSERT INTO audit_log (entity_type, entity_id, action, old_values, new_values, created_at)
			VALUES ('invoice', ?, 'create', '', '', ?)`, e.id, e.createdAt); err != nil {
			t.Fatalf("inserting audit entry %d: %v", e.id, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/audit-log?from="+from+"&to="+to, nil)
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
	return got
}
