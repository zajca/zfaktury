package calc

import (
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

func TestResolveUsedExpenses(t *testing.T) {
	caps := map[int]domain.Amount{
		60: domain.NewAmount(1_200_000, 0), // 1.2M CZK cap for 60%
		80: domain.NewAmount(1_600_000, 0), // 1.6M CZK cap for 80%
	}

	tests := []struct {
		name            string
		revenue         domain.Amount
		actualExpenses  domain.Amount
		flatRatePercent int
		caps            map[int]domain.Amount
		expected        domain.Amount
	}{
		{
			name:            "flat rate 60%, revenue 2M CZK, within cap",
			revenue:         domain.NewAmount(2_000_000, 0),
			actualExpenses:  domain.NewAmount(500_000, 0),
			flatRatePercent: 60,
			caps:            caps,
			expected:        domain.NewAmount(1_200_000, 0), // 2M * 0.6 = 1.2M (at cap)
		},
		{
			name:            "flat rate 60%, revenue 3M CZK, capped at 1.2M",
			revenue:         domain.NewAmount(3_000_000, 0),
			actualExpenses:  domain.NewAmount(500_000, 0),
			flatRatePercent: 60,
			caps:            caps,
			expected:        domain.NewAmount(1_200_000, 0), // 3M * 0.6 = 1.8M > 1.2M cap
		},
		{
			name:            "flat rate 80%, revenue 3M CZK, capped at 1.6M",
			revenue:         domain.NewAmount(3_000_000, 0),
			actualExpenses:  domain.NewAmount(500_000, 0),
			flatRatePercent: 80,
			caps:            caps,
			expected:        domain.NewAmount(1_600_000, 0), // 3M * 0.8 = 2.4M > 1.6M cap
		},
		{
			name:            "flat rate 0%, returns actual expenses",
			revenue:         domain.NewAmount(2_000_000, 0),
			actualExpenses:  domain.NewAmount(800_000, 0),
			flatRatePercent: 0,
			caps:            caps,
			expected:        domain.NewAmount(800_000, 0),
		},
		{
			name:            "flat rate with unknown percent, no cap applied",
			revenue:         domain.NewAmount(2_000_000, 0),
			actualExpenses:  domain.NewAmount(500_000, 0),
			flatRatePercent: 40,
			caps:            caps,
			expected:        domain.NewAmount(800_000, 0), // 2M * 0.4 = 800k, no cap for 40%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveUsedExpenses(tt.revenue, tt.actualExpenses, tt.flatRatePercent, tt.caps)
			if got != tt.expected {
				t.Errorf("ResolveUsedExpenses() = %d (%s), want %d (%s)",
					got, got.String(), tt.expected, tt.expected.String())
			}
		})
	}
}
