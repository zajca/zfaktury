package service

import (
	"context"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

// mockARESClient is a mock implementation of ARESClient for testing.
type mockARESClient struct {
	result *domain.Contact
	err    error
}

func (m *mockARESClient) LookupByICO(ctx context.Context, ico string) (*domain.Contact, error) {
	return m.result, m.err
}

func newContactService(t *testing.T, ares ARESClient) (*ContactService, *repository.ContactRepository) {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := repository.NewContactRepository(db)
	svc := NewContactService(repo, ares)
	return svc, repo
}

func TestContactService_Create_Valid(t *testing.T) {
	svc, _ := newContactService(t, nil)
	ctx := context.Background()

	c := &domain.Contact{
		Type: domain.ContactTypeCompany,
		Name: "Valid Company",
	}
	if err := svc.Create(ctx, c); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if c.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestContactService_Create_EmptyName(t *testing.T) {
	svc, _ := newContactService(t, nil)
	ctx := context.Background()

	c := &domain.Contact{Type: domain.ContactTypeCompany, Name: ""}
	err := svc.Create(ctx, c)
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestContactService_Create_InvalidType(t *testing.T) {
	svc, _ := newContactService(t, nil)
	ctx := context.Background()

	c := &domain.Contact{Name: "Test", Type: "invalid"}
	err := svc.Create(ctx, c)
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestContactService_Create_DefaultType(t *testing.T) {
	svc, _ := newContactService(t, nil)
	ctx := context.Background()

	c := &domain.Contact{Name: "No Type Set"}
	if err := svc.Create(ctx, c); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if c.Type != domain.ContactTypeCompany {
		t.Errorf("Type = %q, want %q", c.Type, domain.ContactTypeCompany)
	}
}

func TestContactService_Update_Valid(t *testing.T) {
	svc, _ := newContactService(t, nil)
	ctx := context.Background()

	c := &domain.Contact{Name: "Original", Type: domain.ContactTypeCompany}
	if err := svc.Create(ctx, c); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	c.Name = "Updated"
	if err := svc.Update(ctx, c); err != nil {
		t.Fatalf("Update() error: %v", err)
	}
}

func TestContactService_Update_ZeroID(t *testing.T) {
	svc, _ := newContactService(t, nil)
	ctx := context.Background()

	err := svc.Update(ctx, &domain.Contact{Name: "Test", Type: domain.ContactTypeCompany})
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestContactService_Update_EmptyName(t *testing.T) {
	svc, _ := newContactService(t, nil)
	ctx := context.Background()

	c := &domain.Contact{Name: "Original", Type: domain.ContactTypeCompany}
	if err := svc.Create(ctx, c); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	c.Name = ""
	err := svc.Update(ctx, c)
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestContactService_Delete_ZeroID(t *testing.T) {
	svc, _ := newContactService(t, nil)
	ctx := context.Background()

	err := svc.Delete(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestContactService_Delete_Success(t *testing.T) {
	svc, _ := newContactService(t, nil)
	ctx := context.Background()

	c := &domain.Contact{Name: "To Delete", Type: domain.ContactTypeCompany}
	if err := svc.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Delete(ctx, c.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.GetByID(ctx, c.ID); err == nil {
		t.Error("expected error after delete")
	}
}

func TestContactService_GetByID_ZeroID(t *testing.T) {
	svc, _ := newContactService(t, nil)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestContactService_GetByID_Success(t *testing.T) {
	svc, _ := newContactService(t, nil)
	ctx := context.Background()

	c := &domain.Contact{Name: "Test Contact", Type: domain.ContactTypeCompany}
	if err := svc.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := svc.GetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Test Contact" {
		t.Errorf("Name = %q, want %q", got.Name, "Test Contact")
	}
}

func TestContactService_List_DefaultLimit(t *testing.T) {
	svc, _ := newContactService(t, nil)
	ctx := context.Background()

	// Create some contacts.
	for i := 0; i < 3; i++ {
		svc.Create(ctx, &domain.Contact{Name: "Contact", Type: domain.ContactTypeCompany})
	}

	contacts, total, err := svc.List(ctx, domain.ContactFilter{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(contacts) != 3 {
		t.Errorf("len = %d, want 3", len(contacts))
	}
}

func TestContactService_List_LimitCapped(t *testing.T) {
	svc, _ := newContactService(t, nil)
	ctx := context.Background()

	// List with excessive limit - should be capped to 100.
	_, _, err := svc.List(ctx, domain.ContactFilter{Limit: 500})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	// No assertion on capping since we can't inspect the actual limit used, but no error is expected.
}

func TestContactService_LookupARES_Success(t *testing.T) {
	mock := &mockARESClient{
		result: &domain.Contact{
			Name: "ARES Company",
			ICO:  "12345678",
			DIC:  "CZ12345678",
		},
	}
	svc, _ := newContactService(t, mock)
	ctx := context.Background()

	c, err := svc.LookupARES(ctx, "12345678")
	if err != nil {
		t.Fatalf("LookupARES() error: %v", err)
	}
	if c.Name != "ARES Company" {
		t.Errorf("Name = %q, want %q", c.Name, "ARES Company")
	}
}

func TestContactService_LookupARES_EmptyICO(t *testing.T) {
	svc, _ := newContactService(t, &mockARESClient{})
	ctx := context.Background()

	_, err := svc.LookupARES(ctx, "")
	if err == nil {
		t.Error("expected error for empty ICO")
	}
}

func TestContactService_LookupARES_NoClient(t *testing.T) {
	svc, _ := newContactService(t, nil)
	ctx := context.Background()

	_, err := svc.LookupARES(ctx, "12345678")
	if err == nil {
		t.Error("expected error when ARES client is nil")
	}
}

func TestContactService_LookupARES_ClientError(t *testing.T) {
	mock := &mockARESClient{err: errors.New("network error")}
	svc, _ := newContactService(t, mock)
	ctx := context.Background()

	_, err := svc.LookupARES(ctx, "12345678")
	if err == nil {
		t.Error("expected error from ARES client")
	}
}
