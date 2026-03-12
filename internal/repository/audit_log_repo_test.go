package repository

import (
	"context"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestAuditLogRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	repo := NewAuditLogRepository(db)

	entry := &domain.AuditLogEntry{
		EntityType: "invoice",
		EntityID:   42,
		Action:     "create",
		NewValues:  `{"number":"FV-001"}`,
	}

	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if entry.ID == 0 {
		t.Fatal("expected non-zero ID after create")
	}
	if entry.CreatedAt.IsZero() {
		t.Fatal("expected non-zero CreatedAt after create")
	}
}

func TestAuditLogRepository_ListByEntity(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	repo := NewAuditLogRepository(db)

	// Create entries for two different entities.
	entry1 := &domain.AuditLogEntry{
		EntityType: "contact",
		EntityID:   1,
		Action:     "create",
		NewValues:  `{"name":"Alice"}`,
	}
	entry2 := &domain.AuditLogEntry{
		EntityType: "contact",
		EntityID:   1,
		Action:     "update",
		OldValues:  `{"name":"Alice"}`,
		NewValues:  `{"name":"Bob"}`,
	}
	entry3 := &domain.AuditLogEntry{
		EntityType: "contact",
		EntityID:   2,
		Action:     "create",
		NewValues:  `{"name":"Charlie"}`,
	}

	for _, e := range []*domain.AuditLogEntry{entry1, entry2, entry3} {
		if err := repo.Create(ctx, e); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	// List entries for entity (contact, 1).
	entries, err := repo.ListByEntity(ctx, "contact", 1)
	if err != nil {
		t.Fatalf("ListByEntity() error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Verify DESC ordering (newest first).
	if entries[0].Action != "update" {
		t.Errorf("first entry action = %q, want %q", entries[0].Action, "update")
	}
	if entries[1].Action != "create" {
		t.Errorf("second entry action = %q, want %q", entries[1].Action, "create")
	}

	// Verify old/new values are preserved.
	if entries[0].OldValues != `{"name":"Alice"}` {
		t.Errorf("OldValues = %q, want %q", entries[0].OldValues, `{"name":"Alice"}`)
	}
	if entries[0].NewValues != `{"name":"Bob"}` {
		t.Errorf("NewValues = %q, want %q", entries[0].NewValues, `{"name":"Bob"}`)
	}
}

func TestAuditLogRepository_ListByEntity_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	repo := NewAuditLogRepository(db)

	// Non-existent entity should return empty slice, not error.
	entries, err := repo.ListByEntity(ctx, "invoice", 9999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestAuditLogRepository_Create_NullableValues(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	repo := NewAuditLogRepository(db)

	// Create entry with empty old/new values.
	entry := &domain.AuditLogEntry{
		EntityType: "expense",
		EntityID:   10,
		Action:     "delete",
	}

	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	entries, err := repo.ListByEntity(ctx, "expense", 10)
	if err != nil {
		t.Fatalf("ListByEntity() error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].OldValues != "" {
		t.Errorf("OldValues = %q, want empty", entries[0].OldValues)
	}
	if entries[0].NewValues != "" {
		t.Errorf("NewValues = %q, want empty", entries[0].NewValues)
	}
}
