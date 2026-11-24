package service

import (
	"context"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestBuildTaxpayerInfo_NormalisesDIC(t *testing.T) {
	tests := []struct {
		name string
		dic  string
		want string
	}{
		{"prefixed", "CZ12345678", "12345678"},
		{"lower case prefix", "cz12345678", "12345678"},
		{"surrounding spaces", "  CZ12345678 ", "12345678"},
		{"spaces and lower case", " cz12345678\t", "12345678"},
		{"no prefix", "12345678", "12345678"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testutil.NewTestDB(t)
			ctx := context.Background()
			if _, err := db.ExecContext(ctx,
				"INSERT INTO settings (key, value, updated_at) VALUES ('dic', ?, datetime('now'))", tt.dic); err != nil {
				t.Fatalf("inserting dic: %v", err)
			}

			info, err := buildTaxpayerInfo(ctx, repository.NewSettingsRepository(db))
			if err != nil {
				t.Fatalf("buildTaxpayerInfo() error: %v", err)
			}
			if info.DIC != tt.want {
				t.Errorf("DIC = %q, want %q", info.DIC, tt.want)
			}
		})
	}
}

func TestBuildTaxpayerInfo_MissingDIC(t *testing.T) {
	for _, dic := range []string{"", "   ", "CZ"} {
		db := testutil.NewTestDB(t)
		ctx := context.Background()
		if dic != "" {
			if _, err := db.ExecContext(ctx,
				"INSERT INTO settings (key, value, updated_at) VALUES ('dic', ?, datetime('now'))", dic); err != nil {
				t.Fatalf("inserting dic: %v", err)
			}
		}

		_, err := buildTaxpayerInfo(ctx, repository.NewSettingsRepository(db))
		if !errors.Is(err, domain.ErrMissingSetting) {
			t.Fatalf("dic %q: error = %v, want ErrMissingSetting", dic, err)
		}
		var userMessage *domain.UserMessageError
		if !errors.As(err, &userMessage) || userMessage.Message != "V nastavení chybí DIČ." {
			t.Errorf("dic %q: error should carry the missing-DIC user message, got %v", dic, err)
		}
	}
}
