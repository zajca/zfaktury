package ares

import (
	"fmt"
	"strconv"

	"github.com/zajca/zfaktury/internal/domain"
)

// aresResponse represents the top-level response from the ARES API
// endpoint /ekonomicke-subjekty/{ICO}.
type aresResponse struct {
	ICO           string    `json:"ico"`
	ObchodniJmeno string    `json:"obchodniJmeno"`
	DIC           string    `json:"dic"`
	Sidlo         aresSidlo `json:"sidlo"`
}

// aresSidlo represents the registered address in the ARES response.
type aresSidlo struct {
	TextovaAdresa   string `json:"textovaAdresa"`
	NazevObce       string `json:"nazevObce"`
	PSC             int    `json:"psc"`
	NazevUlice      string `json:"nazevUlice"`
	CisloDomovni    int    `json:"cisloDomovni"`
	CisloOrientacni int    `json:"cisloOrientacni"`
	NazevCastiObce  string `json:"nazevCastiObce"`
}

// toContact maps the ARES API response to a domain.Contact.
func (r *aresResponse) toContact() *domain.Contact {
	street := r.buildStreet()

	return &domain.Contact{
		Type:    domain.ContactTypeCompany,
		Name:    r.ObchodniJmeno,
		ICO:     r.ICO,
		DIC:     r.DIC,
		Street:  street,
		City:    r.Sidlo.NazevObce,
		ZIP:     r.formatZIP(),
		Country: "CZ",
	}
}

// buildStreet constructs the street line from ARES address components.
func (r *aresResponse) buildStreet() string {
	if r.Sidlo.NazevUlice == "" && r.Sidlo.CisloDomovni == 0 {
		return r.Sidlo.TextovaAdresa
	}

	street := r.Sidlo.NazevUlice
	if r.Sidlo.CisloDomovni > 0 {
		number := strconv.Itoa(r.Sidlo.CisloDomovni)
		if r.Sidlo.CisloOrientacni > 0 {
			number = fmt.Sprintf("%d/%d", r.Sidlo.CisloDomovni, r.Sidlo.CisloOrientacni)
		}
		if street != "" {
			street += " " + number
		} else {
			street = number
		}
	}

	return street
}

// formatZIP formats the PSC as a 5-digit string with leading zeros.
func (r *aresResponse) formatZIP() string {
	if r.Sidlo.PSC == 0 {
		return ""
	}
	return fmt.Sprintf("%05d", r.Sidlo.PSC)
}
