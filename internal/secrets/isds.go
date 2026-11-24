// Package secrets keeps credentials in the operating system keyring. Nothing
// here ever falls back to plain storage: when the keyring cannot be reached
// the operations fail with domain.ErrKeyringUnavailable.
package secrets

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"unicode"

	"github.com/zalando/go-keyring"

	"github.com/zajca/zfaktury/internal/domain"
)

// ISDSEnvironment selects the ISDS (data box) environment a login belongs to.
// There is deliberately no default: callers always name the environment.
type ISDSEnvironment string

const (
	ISDSEnvironmentTest       ISDSEnvironment = "test"
	ISDSEnvironmentProduction ISDSEnvironment = "production"
)

// ISDSEnvironments lists every valid environment in display order.
var ISDSEnvironments = []ISDSEnvironment{ISDSEnvironmentTest, ISDSEnvironmentProduction}

const (
	// MaxISDSUsernameLen and MaxISDSPasswordLen bound what is handed to the
	// keyring; both stay far below the smallest backend limit (Windows caps a
	// credential blob at 2560 bytes).
	MaxISDSUsernameLen = 128
	MaxISDSPasswordLen = 256

	// keyringService is the keyring service name; the per-item key is
	// "isds/<env>/username" or "isds/<env>/password", i.e. the full key is
	// "zfaktury/isds/<env>/username|password".
	keyringService = "zfaktury"
)

// ParseISDSEnvironment validates a raw environment name.
func ParseISDSEnvironment(raw string) (ISDSEnvironment, error) {
	for _, env := range ISDSEnvironments {
		if string(env) == raw {
			return env, nil
		}
	}
	return "", &domain.UserMessageError{Message: "Neznámé prostředí datové schránky, použijte test nebo production."}
}

// ISDSCredentials is a data-box login.
type ISDSCredentials struct {
	Username string
	Password string
}

// ISDSStatus describes what is stored for one environment. It never carries
// the password.
type ISDSStatus struct {
	Configured bool
	Username   string
}

// ISDSStore reads and writes data-box credentials in the OS keyring.
type ISDSStore struct {
	// mu serializes the multi-step operations (save rollback, delete).
	mu sync.Mutex
}

// NewISDSStore creates a store over the OS keyring.
func NewISDSStore() *ISDSStore {
	return &ISDSStore{}
}

// setSecret writes one keyring item; tests replace it to fail a single write.
var setSecret = keyring.Set

// getSecret reads one keyring item; tests replace it to observe which items are read.
var getSecret = keyring.Get

// unlockFailedText is what go-keyring's Secret Service backend reports when
// the unlock prompt was dismissed or refused. The library exposes no sentinel
// for it, so the message text is the only signal.
const unlockFailedText = "failed to unlock correct collection"

func itemKey(env ISDSEnvironment, field string) string {
	return "isds/" + string(env) + "/" + field
}

// Validate checks the shape of a login without touching the keyring.
func (c ISDSCredentials) Validate() error {
	if strings.TrimSpace(c.Username) == "" {
		return &domain.UserMessageError{Message: "Zadejte uživatelské jméno."}
	}
	if c.Password == "" {
		return &domain.UserMessageError{Message: "Zadejte heslo."}
	}
	if len(c.Username) > MaxISDSUsernameLen {
		return &domain.UserMessageError{Message: "Uživatelské jméno je příliš dlouhé."}
	}
	if len(c.Password) > MaxISDSPasswordLen {
		return &domain.UserMessageError{Message: "Heslo je příliš dlouhé."}
	}
	if hasControlRune(c.Username) || hasControlRune(c.Password) {
		return &domain.UserMessageError{Message: "Jméno ani heslo nesmí obsahovat řídicí znaky."}
	}
	return nil
}

func hasControlRune(s string) bool {
	return strings.IndexFunc(s, unicode.IsControl) >= 0
}

// Normalize trims the username; surrounding whitespace is never part of a
// data-box login name.
func (c ISDSCredentials) Normalize() ISDSCredentials {
	c.Username = strings.TrimSpace(c.Username)
	return c
}

// Save stores the login for env. The password is written first and the
// username last: the username entry is the marker that a complete login
// exists, so Status never has to read the password. When writing the username
// fails, the password is removed again.
func (s *ISDSStore) Save(env ISDSEnvironment, creds ISDSCredentials) error {
	creds = creds.Normalize()
	if err := creds.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := setSecret(keyringService, itemKey(env, "password"), creds.Password); err != nil {
		return mapKeyringError("saving", env, err, creds)
	}
	if err := setSecret(keyringService, itemKey(env, "username"), creds.Username); err != nil {
		if rollbackErr := keyring.Delete(keyringService, itemKey(env, "password")); rollbackErr != nil && !errors.Is(rollbackErr, keyring.ErrNotFound) {
			slog.Warn("rolling back ISDS password failed", "environment", env)
		}
		return mapKeyringError("saving", env, err, creds)
	}
	return nil
}

// Load returns the stored login for env, or domain.ErrNotFound when either
// part is missing.
func (s *ISDSStore) Load(env ISDSEnvironment) (ISDSCredentials, error) {
	username, err := getSecret(keyringService, itemKey(env, "username"))
	if err != nil {
		return ISDSCredentials{}, mapKeyringError("reading", env, err, ISDSCredentials{})
	}
	password, err := getSecret(keyringService, itemKey(env, "password"))
	if err != nil {
		return ISDSCredentials{}, mapKeyringError("reading", env, err, ISDSCredentials{Username: username})
	}
	return ISDSCredentials{Username: username, Password: password}, nil
}

// Status reports whether a login is stored for env and its username. It
// reads only the username marker, never the password, so opening the settings
// page does not fetch the secret. A missing entry is not an error.
func (s *ISDSStore) Status(env ISDSEnvironment) (ISDSStatus, error) {
	username, err := getSecret(keyringService, itemKey(env, "username"))
	if errors.Is(err, keyring.ErrNotFound) {
		return ISDSStatus{}, nil
	}
	if err != nil {
		return ISDSStatus{}, mapKeyringError("reading", env, err, ISDSCredentials{})
	}
	return ISDSStatus{Configured: true, Username: username}, nil
}

// Delete removes both entries for env, the username marker first so an
// interrupted delete never leaves a login that reports as configured.
// Entries that are already absent are not an error.
func (s *ISDSStore) Delete(env ISDSEnvironment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var firstErr error
	for _, field := range []string{"username", "password"} {
		err := keyring.Delete(keyringService, itemKey(env, field))
		if err != nil && !errors.Is(err, keyring.ErrNotFound) && firstErr == nil {
			firstErr = mapKeyringError("deleting", env, err, ISDSCredentials{})
		}
	}
	return firstErr
}

// mapKeyringError converts a keyring error into a domain sentinel. The
// returned error text never contains the login; the redacted cause is logged.
func mapKeyringError(op string, env ISDSEnvironment, err error, creds ISDSCredentials) error {
	switch {
	case errors.Is(err, keyring.ErrNotFound):
		return fmt.Errorf("%s ISDS credentials for %s: %w", op, env, domain.ErrNotFound)
	case errors.Is(err, keyring.ErrSetDataTooBig):
		return fmt.Errorf("%s ISDS credentials for %s: %w", op, env,
			&domain.UserMessageError{Message: "Přihlašovací údaje jsou pro systémovou klíčenku příliš dlouhé."})
	case strings.Contains(err.Error(), unlockFailedText):
		slog.Warn("keyring unlock failed", "operation", op, "environment", env)
		return fmt.Errorf("%s ISDS credentials for %s: %w", op, env, domain.ErrKeyringLocked)
	default:
		slog.Warn("keyring operation failed",
			"operation", op, "environment", env, "error", redact(err.Error(), creds))
		return fmt.Errorf("%s ISDS credentials for %s: %w", op, env, domain.ErrKeyringUnavailable)
	}
}

// redact removes the login values from a message before it is logged.
func redact(msg string, creds ISDSCredentials) string {
	for _, secret := range []string{creds.Password, creds.Username} {
		if secret != "" {
			msg = strings.ReplaceAll(msg, secret, "[redacted]")
		}
	}
	return msg
}
