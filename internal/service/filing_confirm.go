package service

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

const (
	// DefaultFilingConfirmTTL is how long a preview's confirmation token stays
	// valid.
	DefaultFilingConfirmTTL = 5 * time.Minute

	// maxFilingConfirmTokens bounds the in-memory token store; the oldest
	// token is dropped first when it is full.
	maxFilingConfirmTokens = 256

	// confirmTokenBytes is the entropy of a token (256 bits).
	confirmTokenBytes = 32
)

// filingBinding is everything a confirmation token is tied to. A token only
// authorizes sending exactly this: the same document, environment, recipient
// box and XML bytes the user saw in the preview.
type filingBinding struct {
	kind         domain.FilingKind
	documentID   int64
	environment  string
	recipientBox string
	xmlSHA256    string
}

type confirmEntry struct {
	binding   filingBinding
	expiresAt time.Time
	// hasWarnings is true when the pre-send check reported warnings, which
	// the sender must accept explicitly.
	hasWarnings bool
}

// confirmTokens is the in-memory store of one-time confirmation tokens. Tokens
// do not survive a restart, by design.
type confirmTokens struct {
	mu      sync.Mutex
	ttl     time.Duration
	now     func() time.Time
	entries map[string]confirmEntry
	// order lists tokens oldest first, for eviction.
	order []string
}

func newConfirmTokens(ttl time.Duration, now func() time.Time) *confirmTokens {
	return &confirmTokens{ttl: ttl, now: now, entries: make(map[string]confirmEntry)}
}

// issue creates a token for binding and returns it with its expiry.
func (t *confirmTokens) issue(binding filingBinding, hasWarnings bool) (string, time.Time, error) {
	raw := make([]byte, confirmTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, fmt.Errorf("generating confirmation token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)

	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	t.pruneLocked(now)
	for len(t.order) >= maxFilingConfirmTokens {
		delete(t.entries, t.order[0])
		t.order = t.order[1:]
	}
	expiresAt := now.Add(t.ttl)
	t.entries[token] = confirmEntry{binding: binding, expiresAt: expiresAt, hasWarnings: hasWarnings}
	t.order = append(t.order, token)
	return token, expiresAt, nil
}

// consume validates token against binding and, when valid, removes it so it
// cannot be used again. A token that exists but does not match binding is
// removed as well, so a mismatching request cannot be retried with the same
// token. When the check reported warnings and acceptWarnings is false, the
// token is kept and domain.ErrWarningsNotAccepted is returned, so the caller
// can repeat the request with the warnings accepted.
func (t *confirmTokens) consume(token string, binding filingBinding, acceptWarnings bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	t.pruneLocked(now)

	entry, ok := t.entries[token]
	if !ok {
		return fmt.Errorf("confirming send: %w", domain.ErrConfirmTokenInvalid)
	}
	if entry.binding != binding {
		t.removeLocked(token)
		return fmt.Errorf("confirming send: %w: the document, recipient or XML changed since the preview", domain.ErrConfirmTokenInvalid)
	}
	if entry.hasWarnings && !acceptWarnings {
		return fmt.Errorf("confirming send: %w", domain.ErrWarningsNotAccepted)
	}
	t.removeLocked(token)
	return nil
}

func (t *confirmTokens) pruneLocked(now time.Time) {
	kept := t.order[:0]
	for _, token := range t.order {
		if entry, ok := t.entries[token]; ok && now.Before(entry.expiresAt) {
			kept = append(kept, token)
			continue
		}
		delete(t.entries, token)
	}
	t.order = kept
}

func (t *confirmTokens) removeLocked(token string) {
	delete(t.entries, token)
	for i, existing := range t.order {
		if subtle.ConstantTimeCompare([]byte(existing), []byte(token)) == 1 {
			t.order = append(t.order[:i], t.order[i+1:]...)
			return
		}
	}
}
