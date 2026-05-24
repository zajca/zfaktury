package domain

import "testing"

func TestCompany_Validate_requiresName(t *testing.T) {
	c := Company{LegalName: "X", ICO: "12345678"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestCompany_Validate_requiresICO(t *testing.T) {
	c := Company{Name: "X", LegalName: "X"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for missing ICO")
	}
}

func TestCompany_Validate_VATRegisteredRequiresDIC(t *testing.T) {
	c := Company{Name: "X", LegalName: "X", ICO: "12345678", VATRegistered: true}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error: VAT-registered without DIC")
	}
}

func TestCompany_Validate_DICFormat(t *testing.T) {
	c := Company{Name: "X", LegalName: "X", ICO: "12345678", VATRegistered: true, DIC: "notvalid"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error: invalid DIC format")
	}
}

func TestCompany_Validate_acceptsValidVATPayer(t *testing.T) {
	c := Company{Name: "M OSVČ", LegalName: "Manas s.r.o.", ICO: "12345678", VATRegistered: true, DIC: "CZ12345678"}
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompany_Validate_acceptsNonVATPayer(t *testing.T) {
	c := Company{Name: "M OSVČ", LegalName: "Manas OSVČ", ICO: "12345678", VATRegistered: false}
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
