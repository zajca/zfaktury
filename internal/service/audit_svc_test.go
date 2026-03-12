package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

// mockAuditLogRepo is a test double for AuditLogRepo.
type mockAuditLogRepo struct {
	entries   []domain.AuditLogEntry
	createErr error
}

func (m *mockAuditLogRepo) Create(_ context.Context, entry *domain.AuditLogEntry) error {
	if m.createErr != nil {
		return m.createErr
	}
	entry.ID = int64(len(m.entries) + 1)
	m.entries = append(m.entries, *entry)
	return nil
}

func (m *mockAuditLogRepo) ListByEntity(_ context.Context, entityType string, entityID int64) ([]domain.AuditLogEntry, error) {
	var result []domain.AuditLogEntry
	for _, e := range m.entries {
		if e.EntityType == entityType && e.EntityID == entityID {
			result = append(result, e)
		}
	}
	return result, nil
}

func TestAuditService_Log_Success(t *testing.T) {
	repo := &mockAuditLogRepo{}
	svc := NewAuditService(repo)

	type testData struct {
		Name string `json:"name"`
	}

	svc.Log(context.Background(), "contact", 1, "create", nil, testData{Name: "Alice"})

	if len(repo.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(repo.entries))
	}

	entry := repo.entries[0]
	if entry.EntityType != "contact" {
		t.Errorf("EntityType = %q, want %q", entry.EntityType, "contact")
	}
	if entry.EntityID != 1 {
		t.Errorf("EntityID = %d, want %d", entry.EntityID, 1)
	}
	if entry.Action != "create" {
		t.Errorf("Action = %q, want %q", entry.Action, "create")
	}
	if entry.OldValues != "" {
		t.Errorf("OldValues = %q, want empty", entry.OldValues)
	}

	// Verify new values are valid JSON.
	var parsed testData
	if err := json.Unmarshal([]byte(entry.NewValues), &parsed); err != nil {
		t.Fatalf("NewValues is not valid JSON: %v", err)
	}
	if parsed.Name != "Alice" {
		t.Errorf("parsed name = %q, want %q", parsed.Name, "Alice")
	}
}

func TestAuditService_Log_WithOldAndNewValues(t *testing.T) {
	repo := &mockAuditLogRepo{}
	svc := NewAuditService(repo)

	oldVal := map[string]string{"name": "Alice"}
	newVal := map[string]string{"name": "Bob"}

	svc.Log(context.Background(), "contact", 1, "update", oldVal, newVal)

	if len(repo.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(repo.entries))
	}

	entry := repo.entries[0]
	if entry.OldValues == "" {
		t.Error("OldValues should not be empty")
	}
	if entry.NewValues == "" {
		t.Error("NewValues should not be empty")
	}
}

func TestAuditService_Log_NilValues(t *testing.T) {
	repo := &mockAuditLogRepo{}
	svc := NewAuditService(repo)

	// Should not panic with nil old and new values.
	svc.Log(context.Background(), "invoice", 5, "delete", nil, nil)

	if len(repo.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(repo.entries))
	}
	if repo.entries[0].OldValues != "" {
		t.Errorf("OldValues = %q, want empty", repo.entries[0].OldValues)
	}
	if repo.entries[0].NewValues != "" {
		t.Errorf("NewValues = %q, want empty", repo.entries[0].NewValues)
	}
}

func TestAuditService_Log_RepoError_DoesNotPanic(t *testing.T) {
	repo := &mockAuditLogRepo{createErr: errors.New("database error")}
	svc := NewAuditService(repo)

	// Should not panic even when repo returns an error.
	svc.Log(context.Background(), "contact", 1, "create", nil, map[string]string{"name": "test"})

	// Entry should not be stored since repo returned error.
	if len(repo.entries) != 0 {
		t.Errorf("expected 0 entries (repo failed), got %d", len(repo.entries))
	}
}
