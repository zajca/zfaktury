package service

import (
	"context"
	"testing"

	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func newSettingsService(t *testing.T) *SettingsService {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := repository.NewSettingsRepository(db)
	return NewSettingsService(repo)
}

func TestSettings_GetAll_EmptyDB(t *testing.T) {
	svc := newSettingsService(t)
	ctx := context.Background()

	settings, err := svc.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() error: %v", err)
	}
	if len(settings) != 0 {
		t.Errorf("expected empty map, got %d entries", len(settings))
	}
}

func TestSettings_Set_And_Get(t *testing.T) {
	svc := newSettingsService(t)
	ctx := context.Background()

	if err := svc.Set(ctx, "company_name", "Test s.r.o."); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	val, err := svc.Get(ctx, "company_name")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if val != "Test s.r.o." {
		t.Errorf("Get() = %q, want %q", val, "Test s.r.o.")
	}
}

func TestSettings_Set_EmptyKey(t *testing.T) {
	svc := newSettingsService(t)
	ctx := context.Background()

	err := svc.Set(ctx, "", "value")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestSettings_Set_UnknownKey(t *testing.T) {
	svc := newSettingsService(t)
	ctx := context.Background()

	err := svc.Set(ctx, "nonexistent_key", "value")
	if err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestSettings_Get_UnknownKey(t *testing.T) {
	svc := newSettingsService(t)
	ctx := context.Background()

	_, err := svc.Get(ctx, "nonexistent_key")
	if err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestSettings_SetBulk_ValidKeys(t *testing.T) {
	svc := newSettingsService(t)
	ctx := context.Background()

	bulk := map[string]string{
		"company_name": "Firma s.r.o.",
		"ico":          "12345678",
		"city":         "Praha",
	}

	if err := svc.SetBulk(ctx, bulk); err != nil {
		t.Fatalf("SetBulk() error: %v", err)
	}

	all, err := svc.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() error: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 settings, got %d", len(all))
	}
	for k, want := range bulk {
		if got := all[k]; got != want {
			t.Errorf("setting %q = %q, want %q", k, got, want)
		}
	}
}

func TestSettings_SetBulk_InvalidKey_NoChanges(t *testing.T) {
	svc := newSettingsService(t)
	ctx := context.Background()

	// First set a valid key so we can verify it's unchanged after failed bulk.
	if err := svc.Set(ctx, "email", "before@test.cz"); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	bulk := map[string]string{
		"email":       "after@test.cz",
		"invalid_key": "bad",
	}

	err := svc.SetBulk(ctx, bulk)
	if err == nil {
		t.Fatal("expected error for invalid key in bulk")
	}

	// Verify the existing value was not changed (validation happens before DB call).
	val, err := svc.Get(ctx, "email")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if val != "before@test.cz" {
		t.Errorf("email = %q, want %q (should be unchanged after failed bulk)", val, "before@test.cz")
	}
}
