package ares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const validAresResponse = `{
	"ico": "27082440",
	"obchodniJmeno": "Alza.cz a.s.",
	"dic": "CZ27082440",
	"sidlo": {
		"textovaAdresa": "Jankovcova 1522/53, 17000 Praha 7",
		"nazevObce": "Praha",
		"psc": 17000,
		"nazevUlice": "Jankovcova",
		"cisloDomovni": 1522,
		"cisloOrientacni": 53,
		"nazevCastiObce": "Praha 7"
	}
}`

func TestLookupByICO_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ekonomicke-subjekty/27082440" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if accept := r.Header.Get("Accept"); accept != "application/json" {
			t.Errorf("expected Accept: application/json, got %s", accept)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validAresResponse))
	}))
	defer srv.Close()

	client := NewClient(WithBaseURL(srv.URL))
	contact, err := client.LookupByICO(context.Background(), "27082440")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if contact.Name != "Alza.cz a.s." {
		t.Errorf("expected Name 'Alza.cz a.s.', got %q", contact.Name)
	}
	if contact.ICO != "27082440" {
		t.Errorf("expected ICO '27082440', got %q", contact.ICO)
	}
	if contact.DIC != "CZ27082440" {
		t.Errorf("expected DIC 'CZ27082440', got %q", contact.DIC)
	}
	if contact.Street != "Jankovcova 1522/53" {
		t.Errorf("expected Street 'Jankovcova 1522/53', got %q", contact.Street)
	}
	if contact.City != "Praha" {
		t.Errorf("expected City 'Praha', got %q", contact.City)
	}
	if contact.ZIP != "17000" {
		t.Errorf("expected ZIP '17000', got %q", contact.ZIP)
	}
	if contact.Country != "CZ" {
		t.Errorf("expected Country 'CZ', got %q", contact.Country)
	}
	if contact.Type != "company" {
		t.Errorf("expected Type 'company', got %q", contact.Type)
	}
}

func TestLookupByICO_InvalidFormat(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name string
		ico  string
	}{
		{"too short", "1234567"},
		{"too long", "123456789"},
		{"letters", "abcdefgh"},
		{"empty", ""},
		{"with spaces", "1234 678"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.LookupByICO(context.Background(), tt.ico)
			if err == nil {
				t.Fatal("expected error for invalid ICO format")
			}
			if !strings.Contains(err.Error(), "invalid ICO format") {
				t.Errorf("expected 'invalid ICO format' error, got: %v", err)
			}
		})
	}
}

func TestLookupByICO_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewClient(WithBaseURL(srv.URL))
	_, err := client.LookupByICO(context.Background(), "99999999")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "subject not found") {
		t.Errorf("expected 'subject not found' error, got: %v", err)
	}
}

func TestLookupByICO_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewClient(WithBaseURL(srv.URL))
	_, err := client.LookupByICO(context.Background(), "12345678")
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !strings.Contains(err.Error(), "ARES service error") {
		t.Errorf("expected 'ARES service error', got: %v", err)
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status code 500 in error, got: %v", err)
	}
}

func TestLookupByICO_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer srv.Close()

	client := NewClient(WithBaseURL(srv.URL))
	_, err := client.LookupByICO(context.Background(), "12345678")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "parsing ARES response") {
		t.Errorf("expected 'parsing ARES response' error, got: %v", err)
	}
}

func TestLookupByICO_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(
		WithBaseURL(srv.URL),
		WithTimeout(50*time.Millisecond),
	)
	_, err := client.LookupByICO(context.Background(), "12345678")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "Timeout") {
		t.Errorf("expected timeout-related error, got: %v", err)
	}
}

func TestLookupByICO_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client := NewClient(WithBaseURL(srv.URL))
	_, err := client.LookupByICO(context.Background(), "12345678")
	if err == nil {
		t.Fatal("expected error for 429")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("expected 'rate limited' error, got: %v", err)
	}
}

func TestToContact_StreetWithoutOrientacni(t *testing.T) {
	resp := &aresResponse{
		ICO:           "12345678",
		ObchodniJmeno: "Test s.r.o.",
		Sidlo: aresSidlo{
			NazevUlice:   "Hlavni",
			CisloDomovni: 42,
			NazevObce:    "Brno",
			PSC:          60200,
		},
	}

	contact := resp.toContact()
	if contact.Street != "Hlavni 42" {
		t.Errorf("expected 'Hlavni 42', got %q", contact.Street)
	}
}

func TestToContact_FallbackToTextovaAdresa(t *testing.T) {
	resp := &aresResponse{
		ICO:           "12345678",
		ObchodniJmeno: "Test s.r.o.",
		Sidlo: aresSidlo{
			TextovaAdresa: "Some full address text",
			NazevObce:     "Brno",
			PSC:           60200,
		},
	}

	contact := resp.toContact()
	if contact.Street != "Some full address text" {
		t.Errorf("expected fallback to textovaAdresa, got %q", contact.Street)
	}
}
