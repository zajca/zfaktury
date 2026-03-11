package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/testutil"
)

func TestSettingsRepository_GetAll_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSettingsRepository(db)
	ctx := context.Background()

	settings, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll on empty DB: %v", err)
	}
	if len(settings) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(settings))
	}
}

func TestSettingsRepository_SetAndGet(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSettingsRepository(db)
	ctx := context.Background()

	if err := repo.Set(ctx, "company_name", "Acme Corp"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, err := repo.Get(ctx, "company_name")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "Acme Corp" {
		t.Fatalf("expected %q, got %q", "Acme Corp", val)
	}
}

func TestSettingsRepository_Set_Upsert(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSettingsRepository(db)
	ctx := context.Background()

	if err := repo.Set(ctx, "theme", "light"); err != nil {
		t.Fatalf("Set first: %v", err)
	}
	if err := repo.Set(ctx, "theme", "dark"); err != nil {
		t.Fatalf("Set second: %v", err)
	}

	val, err := repo.Get(ctx, "theme")
	if err != nil {
		t.Fatalf("Get after upsert: %v", err)
	}
	if val != "dark" {
		t.Fatalf("expected %q after upsert, got %q", "dark", val)
	}
}

func TestSettingsRepository_SetBulk_GetAll(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSettingsRepository(db)
	ctx := context.Background()

	bulk := map[string]string{
		"key_a": "value_a",
		"key_b": "value_b",
		"key_c": "value_c",
	}
	if err := repo.SetBulk(ctx, bulk); err != nil {
		t.Fatalf("SetBulk: %v", err)
	}

	settings, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(settings) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(settings))
	}
	for k, want := range bulk {
		if got := settings[k]; got != want {
			t.Errorf("key %q: expected %q, got %q", k, want, got)
		}
	}
}

func TestSettingsRepository_SetBulk_Overwrites(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSettingsRepository(db)
	ctx := context.Background()

	if err := repo.Set(ctx, "existing", "old_value"); err != nil {
		t.Fatalf("Set initial: %v", err)
	}

	bulk := map[string]string{
		"existing": "new_value",
		"fresh":    "fresh_value",
	}
	if err := repo.SetBulk(ctx, bulk); err != nil {
		t.Fatalf("SetBulk: %v", err)
	}

	val, err := repo.Get(ctx, "existing")
	if err != nil {
		t.Fatalf("Get existing: %v", err)
	}
	if val != "new_value" {
		t.Fatalf("expected %q after bulk overwrite, got %q", "new_value", val)
	}

	val, err = repo.Get(ctx, "fresh")
	if err != nil {
		t.Fatalf("Get fresh: %v", err)
	}
	if val != "fresh_value" {
		t.Fatalf("expected %q, got %q", "fresh_value", val)
	}
}

func TestSettingsRepository_Get_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSettingsRepository(db)
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent key, got nil")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected error wrapping sql.ErrNoRows, got: %v", err)
	}
}
