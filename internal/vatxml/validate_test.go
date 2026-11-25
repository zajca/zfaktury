package vatxml

import (
	"errors"
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

func validInfo() TaxpayerInfo {
	info := testTaxpayerInfo()
	info.OKEC = "62010" // testTaxpayerInfo uses a 6-digit value not accepted by the validator
	return info
}

func TestValidateTaxpayerInfo_Valid(t *testing.T) {
	if err := ValidateTaxpayerInfo(validInfo()); err != nil {
		t.Errorf("expected nil error for valid info, got: %v", err)
	}
}

func TestValidateTaxpayerInfo_MissingFields(t *testing.T) {
	info := TaxpayerInfo{} // everything empty
	err := ValidateTaxpayerInfo(info)
	if err == nil {
		t.Fatal("expected error for empty info")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
	for _, label := range []string{"DIČ", "Jméno", "Příjmení", "Ulice", "Obec", "PSČ", "Číslo popisné", "Kód FÚ", "Kód pracoviště FÚ", "NACE"} {
		if !strings.Contains(err.Error(), label) {
			t.Errorf("expected error to mention missing %q, got: %v", label, err)
		}
	}
}

func TestValidateTaxpayerInfo_FormatErrors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*TaxpayerInfo)
		wantSub string
	}{
		{"bad DIC", func(i *TaxpayerInfo) { i.DIC = "123" }, "DIČ musí být"},
		{"bad OKEC", func(i *TaxpayerInfo) { i.OKEC = "abc" }, "NACE kód"},
		{"bad ZIP", func(i *TaxpayerInfo) { i.ZIP = "abcde" }, "PSČ"},
		{"bad UFO", func(i *TaxpayerInfo) { i.UFOCode = "12" }, "Kód FÚ"},
		{"bad PracUFO", func(i *TaxpayerInfo) { i.PracUFO = "1" }, "pracoviště FÚ"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := validInfo()
			tt.mutate(&info)
			err := ValidateTaxpayerInfo(info)
			if err == nil || !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("expected error containing %q, got: %v", tt.wantSub, err)
			}
		})
	}
}

func TestValidateTaxpayerInfo_ZIPWithSpace(t *testing.T) {
	info := validInfo()
	info.ZIP = "763 41" // spaces stripped before validation
	if err := ValidateTaxpayerInfo(info); err != nil {
		t.Errorf("expected ZIP with space to be accepted, got: %v", err)
	}
}

func TestValidateTaxpayer_HouseNum(t *testing.T) {
	tests := []struct {
		name     string
		houseNum string
		wantErr  bool
	}{
		{"slash is refused", "123/4", true},
		{"seven digits are refused", "1234567", true},
		{"letters are refused", "12a", true},
		{"four digits are accepted", "1234", false},
		{"six digits are accepted", "123456", false},
		{"one digit is accepted", "1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := validInfo()
			info.HouseNum = tt.houseNum
			err := ValidateTaxpayerInfo(info)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "Číslo popisné") {
					t.Errorf("expected a Číslo popisné error for %q, got: %v", tt.houseNum, err)
				}
				return
			}
			if err != nil {
				t.Errorf("expected %q to be accepted, got: %v", tt.houseNum, err)
			}
		})
	}
}

func TestValidateTaxpayer_Phone(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		wantErr bool
	}{
		{"14 characters are accepted", strings.Repeat("9", 14), false},
		{"15 characters are refused", strings.Repeat("9", 15), true},
		{"a short number is accepted", "600000000", false},
		{"an empty number is accepted", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := validInfo()
			info.Phone = tt.phone
			err := ValidateTaxpayerInfo(info)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "Telefon") {
					t.Errorf("expected a Telefon error for %q, got: %v", tt.phone, err)
				}
				return
			}
			if err != nil {
				t.Errorf("expected %q to be accepted, got: %v", tt.phone, err)
			}
		})
	}
}

// The control statement and VIES summary share ValidateTaxpayerIdentity, which
// has no NACE code; the new address and phone rules must apply there too.
func TestValidateTaxpayerIdentity_AppliesSharedRules(t *testing.T) {
	info := validInfo()
	info.OKEC = "" // not part of the shared identity block
	info.HouseNum = "123/4"
	if err := ValidateTaxpayerIdentity(info, TaxpayerDocControlStatement); err == nil {
		t.Error("expected a house-number error from ValidateTaxpayerIdentity")
	}
}
