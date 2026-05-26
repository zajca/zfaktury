// Package format renders and validates invoice-number templates such as
// "{prefix}-{yy}-{number:03d}". The same logic is reused by the service
// layer (for previews) and the repository layer (for GetNextNumber), and a
// TypeScript port at frontend/src/lib/utils/sequence-format.ts is kept in
// lockstep through the shared testdata/render_cases.json fixture.
package format

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zajca/zfaktury/internal/domain"
)

const (
	minNumberWidth = 1
	maxNumberWidth = 6
)

// Render evaluates pattern against (prefix, year, number) and returns the
// formatted invoice number. Callers must have validated the pattern with
// ValidatePattern before storing it; Render on an invalid pattern returns
// the raw pattern unchanged so a programming error stays visible rather
// than silently producing a half-rendered string.
func Render(pattern, prefix string, year, number int) string {
	if ValidatePattern(pattern) != nil {
		return pattern
	}
	var b strings.Builder
	b.Grow(len(pattern) + 8)
	i := 0
	for i < len(pattern) {
		if pattern[i] != '{' {
			b.WriteByte(pattern[i])
			i++
			continue
		}
		end := strings.IndexByte(pattern[i:], '}')
		// ValidatePattern already rejected unterminated braces, so end >= 0
		// and the token is one we know how to render.
		token := pattern[i+1 : i+end]
		kind, _ := classifyToken(token)
		writeTokenByKind(&b, kind, token, prefix, year, number)
		i += end + 1
	}
	return b.String()
}

// ValidatePattern reports whether pattern is a well-formed template that can
// safely drive GetNextNumber. Errors wrap domain.ErrInvalidInput so handlers
// surface them as 422.
func ValidatePattern(pattern string) error {
	if strings.TrimSpace(pattern) == "" {
		return fmt.Errorf("format pattern is empty: %w", domain.ErrInvalidInput)
	}
	numberTokens := 0
	i := 0
	for i < len(pattern) {
		if pattern[i] != '{' {
			i++
			continue
		}
		end := strings.IndexByte(pattern[i:], '}')
		if end < 0 {
			return fmt.Errorf("format pattern has unterminated %q: %w", "{", domain.ErrInvalidInput)
		}
		token := pattern[i+1 : i+end]
		kind, err := classifyToken(token)
		if err != nil {
			return err
		}
		if kind == tokenNumber {
			numberTokens++
		}
		i += end + 1
	}
	if numberTokens != 1 {
		return fmt.Errorf("format pattern must contain exactly one {number...} token, found %d: %w", numberTokens, domain.ErrInvalidInput)
	}
	return nil
}

type tokenKind int

const (
	tokenPrefix tokenKind = iota
	tokenYearFull
	tokenYearShort
	tokenNumber
)

func classifyToken(token string) (tokenKind, error) {
	switch token {
	case "prefix":
		return tokenPrefix, nil
	case "yyyy", "year":
		return tokenYearFull, nil
	case "yy":
		return tokenYearShort, nil
	case "number":
		return tokenNumber, nil
	}
	if strings.HasPrefix(token, "number:") {
		if _, err := parseNumberWidth(token[len("number:"):]); err != nil {
			return 0, err
		}
		return tokenNumber, nil
	}
	return 0, fmt.Errorf("unknown format token %q: %w", "{"+token+"}", domain.ErrInvalidInput)
}

// parseNumberWidth accepts strings shaped like "03d" or "4d" and returns the
// numeric width when it falls in [minNumberWidth, maxNumberWidth].
func parseNumberWidth(spec string) (int, error) {
	if !strings.HasSuffix(spec, "d") {
		return 0, fmt.Errorf("number width spec %q must end in 'd': %w", spec, domain.ErrInvalidInput)
	}
	digits := strings.TrimSuffix(spec, "d")
	if digits == "" {
		return 0, fmt.Errorf("number width spec is empty: %w", domain.ErrInvalidInput)
	}
	width, err := strconv.Atoi(digits)
	if err != nil {
		return 0, fmt.Errorf("number width %q is not numeric: %w", digits, domain.ErrInvalidInput)
	}
	if width < minNumberWidth || width > maxNumberWidth {
		return 0, fmt.Errorf("number width %d must be in %d..%d: %w", width, minNumberWidth, maxNumberWidth, domain.ErrInvalidInput)
	}
	return width, nil
}

func writeTokenByKind(b *strings.Builder, kind tokenKind, token, prefix string, year, number int) {
	switch kind {
	case tokenPrefix:
		b.WriteString(prefix)
	case tokenYearFull:
		fmt.Fprintf(b, "%04d", year)
	case tokenYearShort:
		fmt.Fprintf(b, "%02d", year%100)
	case tokenNumber:
		if token == "number" {
			fmt.Fprintf(b, "%d", number)
		} else {
			// Must be "number:Nd".
			width, _ := parseNumberWidth(token[len("number:"):])
			fmt.Fprintf(b, "%0*d", width, number)
		}
	}
}
