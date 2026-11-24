package service

import (
	"fmt"
	"log/slog"

	"github.com/zajca/zfaktury/internal/secrets"
)

// ISDSCredentialStore is the keyring-backed login store used by the service.
type ISDSCredentialStore interface {
	Save(env secrets.ISDSEnvironment, creds secrets.ISDSCredentials) error
	Status(env secrets.ISDSEnvironment) (secrets.ISDSStatus, error)
	Delete(env secrets.ISDSEnvironment) error
}

// ISDSCredentialsService manages the data-box login of the delegated user.
// The login lives only in the OS keyring; nothing here touches the database.
type ISDSCredentialsService struct {
	store ISDSCredentialStore
}

// NewISDSCredentialsService creates a new ISDSCredentialsService.
func NewISDSCredentialsService(store ISDSCredentialStore) *ISDSCredentialsService {
	return &ISDSCredentialsService{store: store}
}

// Status returns the stored state of every environment. It never returns a password.
func (s *ISDSCredentialsService) Status() (map[secrets.ISDSEnvironment]secrets.ISDSStatus, error) {
	out := make(map[secrets.ISDSEnvironment]secrets.ISDSStatus, len(secrets.ISDSEnvironments))
	for _, env := range secrets.ISDSEnvironments {
		status, err := s.store.Status(env)
		if err != nil {
			return nil, fmt.Errorf("fetching ISDS credentials status: %w", err)
		}
		out[env] = status
	}
	return out, nil
}

// Save validates and stores the login for the named environment.
func (s *ISDSCredentialsService) Save(rawEnv string, creds secrets.ISDSCredentials) error {
	env, err := secrets.ParseISDSEnvironment(rawEnv)
	if err != nil {
		return fmt.Errorf("saving ISDS credentials: %w", err)
	}
	creds = creds.Normalize()
	if err := creds.Validate(); err != nil {
		return fmt.Errorf("saving ISDS credentials: %w", err)
	}
	if err := s.store.Save(env, creds); err != nil {
		return fmt.Errorf("saving ISDS credentials: %w", err)
	}
	slog.Info("ISDS credentials saved", "environment", env)
	return nil
}

// Delete removes the login of the named environment.
func (s *ISDSCredentialsService) Delete(rawEnv string) error {
	env, err := secrets.ParseISDSEnvironment(rawEnv)
	if err != nil {
		return fmt.Errorf("deleting ISDS credentials: %w", err)
	}
	if err := s.store.Delete(env); err != nil {
		return fmt.Errorf("deleting ISDS credentials: %w", err)
	}
	slog.Info("ISDS credentials deleted", "environment", env)
	return nil
}
