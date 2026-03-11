package domain

import "testing"

func TestContact_DICCountryCode(t *testing.T) {
	tests := []struct {
		name string
		dic  string
		want string
	}{
		{"empty DIC", "", ""},
		{"single char DIC", "C", ""},
		{"CZ DIC", "CZ12345678", "CZ"},
		{"DE DIC", "DE123456789", "DE"},
		{"US DIC", "US12345", "US"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Contact{DIC: tt.dic}
			got := c.DICCountryCode()
			if got != tt.want {
				t.Errorf("DICCountryCode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContact_IsEUPartner(t *testing.T) {
	tests := []struct {
		name string
		dic  string
		want bool
	}{
		{"empty DIC", "", false},
		{"CZ DIC", "CZ12345678", false},
		{"DE DIC", "DE123456789", true},
		{"US DIC", "US12345", false},
		{"SK DIC", "SK2020123456", true},
		{"EL DIC (Greece)", "EL123456789", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Contact{DIC: tt.dic}
			got := c.IsEUPartner()
			if got != tt.want {
				t.Errorf("IsEUPartner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContact_HasCZDIC(t *testing.T) {
	tests := []struct {
		name string
		dic  string
		want bool
	}{
		{"CZ DIC", "CZ12345678", true},
		{"DE DIC", "DE123456789", false},
		{"empty DIC", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Contact{DIC: tt.dic}
			got := c.HasCZDIC()
			if got != tt.want {
				t.Errorf("HasCZDIC() = %v, want %v", got, tt.want)
			}
		})
	}
}
