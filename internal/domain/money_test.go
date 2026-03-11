package domain

import (
	"testing"
)

func TestNewAmount(t *testing.T) {
	tests := []struct {
		name     string
		whole    int64
		fraction int64
		want     Amount
	}{
		{"zero", 0, 0, 0},
		{"whole only", 100, 0, 10000},
		{"with halere", 100, 50, 10050},
		{"one haler", 0, 1, 1},
		{"negative whole", -50, 0, -5000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewAmount(tt.whole, tt.fraction)
			if got != tt.want {
				t.Errorf("NewAmount(%d, %d) = %d, want %d", tt.whole, tt.fraction, got, tt.want)
			}
		})
	}
}

func TestFromFloat(t *testing.T) {
	tests := []struct {
		name string
		f    float64
		want Amount
	}{
		{"zero", 0.0, 0},
		{"whole", 100.0, 10000},
		{"with cents", 99.99, 9999},
		{"rounding up", 10.005, 1001},
		{"rounding down", 10.004, 1000},
		{"negative", -25.50, -2550},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromFloat(tt.f)
			if got != tt.want {
				t.Errorf("FromFloat(%f) = %d, want %d", tt.f, got, tt.want)
			}
		})
	}
}

func TestAmount_ToCZK(t *testing.T) {
	tests := []struct {
		name string
		a    Amount
		want float64
	}{
		{"zero", 0, 0.0},
		{"100 CZK", 10000, 100.0},
		{"99.99 CZK", 9999, 99.99},
		{"negative", -2550, -25.50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.ToCZK()
			if got != tt.want {
				t.Errorf("Amount(%d).ToCZK() = %f, want %f", tt.a, got, tt.want)
			}
		})
	}
}

func TestAmount_String(t *testing.T) {
	tests := []struct {
		name string
		a    Amount
		want string
	}{
		{"zero", 0, "0.00"},
		{"whole", 10000, "100.00"},
		{"with halere", 10050, "100.50"},
		{"single digit halere", 10005, "100.05"},
		{"negative", -2550, "-25.50"},
		{"negative with fraction", -99, "-0.99"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.String()
			if got != tt.want {
				t.Errorf("Amount(%d).String() = %q, want %q", tt.a, got, tt.want)
			}
		})
	}
}

func TestAmount_Add(t *testing.T) {
	tests := []struct {
		name string
		a, b Amount
		want Amount
	}{
		{"both zero", 0, 0, 0},
		{"add to zero", 0, 100, 100},
		{"positive", 1000, 500, 1500},
		{"negative result", 100, -200, -100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Add(tt.b)
			if got != tt.want {
				t.Errorf("%d.Add(%d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestAmount_Sub(t *testing.T) {
	tests := []struct {
		name string
		a, b Amount
		want Amount
	}{
		{"both zero", 0, 0, 0},
		{"positive result", 1500, 500, 1000},
		{"negative result", 500, 1500, -1000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Sub(tt.b)
			if got != tt.want {
				t.Errorf("%d.Sub(%d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestAmount_Multiply(t *testing.T) {
	tests := []struct {
		name   string
		a      Amount
		factor float64
		want   Amount
	}{
		{"by zero", 1000, 0, 0},
		{"by one", 1000, 1.0, 1000},
		{"by two", 1000, 2.0, 2000},
		{"by half", 1000, 0.5, 500},
		{"21% VAT", 10000, 0.21, 2100},
		{"rounding", 333, 0.5, 167}, // 166.5 -> 167
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Multiply(tt.factor)
			if got != tt.want {
				t.Errorf("%d.Multiply(%f) = %d, want %d", tt.a, tt.factor, got, tt.want)
			}
		})
	}
}

func TestAmount_IsZero(t *testing.T) {
	if !Amount(0).IsZero() {
		t.Error("Amount(0).IsZero() should be true")
	}
	if Amount(1).IsZero() {
		t.Error("Amount(1).IsZero() should be false")
	}
	if Amount(-1).IsZero() {
		t.Error("Amount(-1).IsZero() should be false")
	}
}

func TestAmount_IsNegative(t *testing.T) {
	if Amount(0).IsNegative() {
		t.Error("Amount(0).IsNegative() should be false")
	}
	if Amount(1).IsNegative() {
		t.Error("Amount(1).IsNegative() should be false")
	}
	if !Amount(-1).IsNegative() {
		t.Error("Amount(-1).IsNegative() should be true")
	}
}
