package secrets

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/zajca/zfaktury/internal/domain"
)

// TestMain installs the in-memory keyring so no test can reach the OS keyring.
func TestMain(m *testing.M) {
	keyring.MockInit()
	m.Run()
}

func useMock(t *testing.T) {
	t.Helper()
	keyring.MockInit()
	t.Cleanup(keyring.MockInit)
}

func useBrokenKeyring(t *testing.T, err error) {
	t.Helper()
	keyring.MockInitWithError(err)
	t.Cleanup(keyring.MockInit)
}

func TestParseISDSEnvironment(t *testing.T) {
	for _, raw := range []string{"test", "production"} {
		env, err := ParseISDSEnvironment(raw)
		if err != nil || string(env) != raw {
			t.Errorf("ParseISDSEnvironment(%q) = %q, %v", raw, env, err)
		}
	}
	for _, raw := range []string{"", "prod", "Test", "TEST", " test"} {
		if _, err := ParseISDSEnvironment(raw); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("ParseISDSEnvironment(%q) error = %v, want ErrInvalidInput", raw, err)
		}
	}
}

func TestISDSStore_SaveLoadDelete(t *testing.T) {
	useMock(t)
	store := NewISDSStore()
	creds := ISDSCredentials{Username: "sender-user", Password: "Sample-Pass-1"}

	if err := store.Save(ISDSEnvironmentTest, creds); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// The keys follow zfaktury/isds/<env>/username|password.
	if got, err := keyring.Get("zfaktury", "isds/test/username"); err != nil || got != creds.Username {
		t.Errorf("username key = %q, %v", got, err)
	}
	if got, err := keyring.Get("zfaktury", "isds/test/password"); err != nil || got != creds.Password {
		t.Errorf("password key = %q, %v", got, err)
	}

	loaded, err := store.Load(ISDSEnvironmentTest)
	if err != nil || loaded != creds {
		t.Fatalf("Load = %+v, %v", loaded, err)
	}

	status, err := store.Status(ISDSEnvironmentTest)
	if err != nil || !status.Configured || status.Username != creds.Username {
		t.Fatalf("Status = %+v, %v", status, err)
	}

	if err := store.Delete(ISDSEnvironmentTest); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Load(ISDSEnvironmentTest); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Load after delete error = %v, want ErrNotFound", err)
	}
	if status, err := store.Status(ISDSEnvironmentTest); err != nil || status.Configured || status.Username != "" {
		t.Errorf("Status after delete = %+v, %v", status, err)
	}
}

func TestISDSStore_EnvironmentsAreIndependent(t *testing.T) {
	useMock(t)
	store := NewISDSStore()
	if err := store.Save(ISDSEnvironmentTest, ISDSCredentials{Username: "u-test", Password: "p-test"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ISDSEnvironmentProduction, ISDSCredentials{Username: "u-prod", Password: "p-prod"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ISDSEnvironmentTest); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load(ISDSEnvironmentProduction)
	if err != nil || got.Username != "u-prod" {
		t.Errorf("production login after deleting test = %+v, %v", got, err)
	}
}

func TestISDSStore_MissingEntry(t *testing.T) {
	useMock(t)
	store := NewISDSStore()

	if _, err := store.Load(ISDSEnvironmentProduction); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Load error = %v, want ErrNotFound", err)
	}
	status, err := store.Status(ISDSEnvironmentProduction)
	if err != nil || status.Configured {
		t.Errorf("Status = %+v, %v", status, err)
	}
	if err := store.Delete(ISDSEnvironmentProduction); err != nil {
		t.Errorf("Delete of absent entries: %v", err)
	}
}

func TestISDSStore_StatusWithOnlyUsername(t *testing.T) {
	useMock(t)
	if err := keyring.Set("zfaktury", "isds/test/username", "orphan"); err != nil {
		t.Fatal(err)
	}
	status, err := NewISDSStore().Status(ISDSEnvironmentTest)
	if err != nil || status.Configured || status.Username != "orphan" {
		t.Errorf("Status = %+v, %v; want unconfigured with username", status, err)
	}
	if _, err := NewISDSStore().Load(ISDSEnvironmentTest); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Load error = %v, want ErrNotFound", err)
	}
}

func TestISDSStore_KeyringUnavailable(t *testing.T) {
	useBrokenKeyring(t, errors.New("dial unix: no secret service"))
	store := NewISDSStore()

	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	creds := ISDSCredentials{Username: "sender-user", Password: "Sample-Pass-1"}
	checks := map[string]error{
		"Save":   store.Save(ISDSEnvironmentTest, creds),
		"Delete": store.Delete(ISDSEnvironmentTest),
	}
	_, checks["Load"] = store.Load(ISDSEnvironmentTest)
	_, checks["Status"] = store.Status(ISDSEnvironmentTest)

	for name, err := range checks {
		if !errors.Is(err, domain.ErrKeyringUnavailable) {
			t.Errorf("%s error = %v, want ErrKeyringUnavailable", name, err)
		}
		if err != nil && (strings.Contains(err.Error(), creds.Password) || strings.Contains(err.Error(), creds.Username)) {
			t.Errorf("%s error leaks the login: %v", name, err)
		}
	}
	if strings.Contains(logs.String(), creds.Password) {
		t.Errorf("log output leaks the password: %s", logs.String())
	}
}

func TestRedact(t *testing.T) {
	creds := ISDSCredentials{Username: "sender-user", Password: "Sample-Pass-1"}
	got := redact("backend said sender-user / Sample-Pass-1 failed", creds)
	if strings.Contains(got, creds.Username) || strings.Contains(got, creds.Password) {
		t.Errorf("redact left the login in place: %q", got)
	}
	if got := redact("no secrets here", ISDSCredentials{}); got != "no secrets here" {
		t.Errorf("redact changed a clean message: %q", got)
	}
}

func TestISDSStore_SaveRollsBackUsername(t *testing.T) {
	useMock(t)
	// Only the password write fails, after the username write succeeded.
	setSecret = func(service, user, password string) error {
		if strings.HasSuffix(user, "/password") {
			return keyring.ErrSetDataTooBig
		}
		return keyring.Set(service, user, password)
	}
	t.Cleanup(func() { setSecret = keyring.Set })

	err := NewISDSStore().Save(ISDSEnvironmentTest, ISDSCredentials{Username: "u", Password: "p"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("Save error = %v, want ErrInvalidInput", err)
	}
	if status, _ := NewISDSStore().Status(ISDSEnvironmentTest); status.Configured || status.Username != "" {
		t.Errorf("a failed save left state behind: %+v", status)
	}
}

func TestISDSCredentials_Validate(t *testing.T) {
	long := func(n int) string { return strings.Repeat("a", n) }
	tests := []struct {
		name  string
		creds ISDSCredentials
		ok    bool
	}{
		{"valid", ISDSCredentials{Username: "user", Password: "pass"}, true},
		{"empty username", ISDSCredentials{Username: "", Password: "pass"}, false},
		{"blank username", ISDSCredentials{Username: "   ", Password: "pass"}, false},
		{"empty password", ISDSCredentials{Username: "user"}, false},
		{"username at limit", ISDSCredentials{Username: long(MaxISDSUsernameLen), Password: "p"}, true},
		{"username too long", ISDSCredentials{Username: long(MaxISDSUsernameLen + 1), Password: "p"}, false},
		{"password at limit", ISDSCredentials{Username: "u", Password: long(MaxISDSPasswordLen)}, true},
		{"password too long", ISDSCredentials{Username: "u", Password: long(MaxISDSPasswordLen + 1)}, false},
		{"control char in username", ISDSCredentials{Username: "us\ner", Password: "p"}, false},
		{"control char in password", ISDSCredentials{Username: "u", Password: "p\x00"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.creds.Validate()
			if tt.ok && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.ok && !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestISDSStore_SaveRejectsInvalidBeforeTouchingKeyring(t *testing.T) {
	useMock(t)
	if err := NewISDSStore().Save(ISDSEnvironmentTest, ISDSCredentials{Username: "u"}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
	if _, err := keyring.Get("zfaktury", "isds/test/username"); !errors.Is(err, keyring.ErrNotFound) {
		t.Errorf("invalid input reached the keyring: %v", err)
	}
}
