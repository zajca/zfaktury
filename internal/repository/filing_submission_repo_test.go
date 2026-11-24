package repository

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

var _ FilingSubmissionRepo = (*FilingSubmissionRepository)(nil)

func newSubmission(sha string) *domain.FilingSubmission {
	return &domain.FilingSubmission{
		DocumentKind:   domain.FilingKindVATReturn,
		DocumentID:     7,
		Channel:        domain.FilingChannelISDS,
		Environment:    "test",
		RecipientBoxID: "abc1234",
		RecipientName:  "Fake tax office",
		DmAnnotation:   "Fake annotation",
		FileName:       "fake.xml",
		XMLSHA256:      sha,
		Status:         domain.SubmissionSending,
	}
}

func sha(c string) string { return strings.Repeat(c, 64) }

func TestFilingSubmissionRepository_CreateGetAndLifecycle(t *testing.T) {
	repo := NewFilingSubmissionRepository(testutil.NewTestDB(t))
	ctx := context.Background()

	sub := newSubmission(sha("a"))
	if err := repo.Create(ctx, sub); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sub.ID == 0 || sub.CreatedAt.IsZero() {
		t.Fatalf("Create did not set ID and CreatedAt: %+v", sub)
	}

	got, err := repo.GetByID(ctx, sub.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != domain.SubmissionSending || got.DmID != "" || got.SentAt != nil || got.HasReceipt || got.DmMessageStatus != nil {
		t.Errorf("fresh submission = %+v", got)
	}
	if got.DocumentKind != domain.FilingKindVATReturn || got.Environment != "test" || got.XMLSHA256 != sha("a") || got.RecipientBoxID != "abc1234" {
		t.Errorf("columns round-trip = %+v", got)
	}

	sentAt := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)
	if err := repo.MarkSent(ctx, sub.ID, "123456", sentAt); err != nil {
		t.Fatalf("MarkSent: %v", err)
	}
	got, _ = repo.GetByID(ctx, sub.ID)
	if got.Status != domain.SubmissionSent || got.DmID != "123456" || got.SentAt == nil || !got.SentAt.Equal(sentAt) {
		t.Errorf("after MarkSent = %+v", got)
	}

	deliveredAt := time.Date(2026, 11, 24, 10, 5, 0, 0, time.UTC)
	if err := repo.UpdateDelivery(ctx, sub.ID, domain.SubmissionDelivered, 4, &deliveredAt); err != nil {
		t.Fatalf("UpdateDelivery: %v", err)
	}
	got, _ = repo.GetByID(ctx, sub.ID)
	if got.Status != domain.SubmissionDelivered || got.DmMessageStatus == nil || *got.DmMessageStatus != 4 ||
		got.DeliveredAt == nil || !got.DeliveredAt.Equal(deliveredAt) {
		t.Errorf("after UpdateDelivery = %+v", got)
	}

	// A later refresh with another time keeps the first delivery time.
	later := deliveredAt.Add(time.Hour)
	if err := repo.UpdateDelivery(ctx, sub.ID, domain.SubmissionDelivered, 7, &later); err != nil {
		t.Fatalf("second UpdateDelivery: %v", err)
	}
	got, _ = repo.GetByID(ctx, sub.ID)
	if !got.DeliveredAt.Equal(deliveredAt) || *got.DmMessageStatus != 7 {
		t.Errorf("delivered_at overwritten or status not updated: %+v", got)
	}
}

func TestFilingSubmissionRepository_Receipt(t *testing.T) {
	repo := NewFilingSubmissionRepository(testutil.NewTestDB(t))
	ctx := context.Background()
	sub := newSubmission(sha("b"))
	if err := repo.Create(ctx, sub); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.GetReceipt(ctx, sub.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetReceipt without receipt = %v, want ErrNotFound", err)
	}
	if err := repo.SaveReceipt(ctx, sub.ID, nil); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("SaveReceipt(empty) = %v, want ErrInvalidInput", err)
	}

	blob := []byte{0x00, 0x01, 0xfe, 0xff, 'Z', 'F', 'O'}
	if err := repo.SaveReceipt(ctx, sub.ID, blob); err != nil {
		t.Fatalf("SaveReceipt: %v", err)
	}
	got, err := repo.GetReceipt(ctx, sub.ID)
	if err != nil || string(got) != string(blob) {
		t.Errorf("GetReceipt = %v, %v", got, err)
	}
	loaded, _ := repo.GetByID(ctx, sub.ID)
	if !loaded.HasReceipt {
		t.Error("HasReceipt is false although a receipt is stored")
	}
	if _, err := repo.GetReceipt(ctx, sub.ID+99); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetReceipt for a missing submission = %v", err)
	}
	if err := repo.SaveReceipt(ctx, sub.ID+99, blob); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("SaveReceipt for a missing submission = %v", err)
	}
}

func TestFilingSubmissionRepository_UniqueIndexBlocksResend(t *testing.T) {
	repo := NewFilingSubmissionRepository(testutil.NewTestDB(t))
	ctx := context.Background()

	first := newSubmission(sha("c"))
	if err := repo.Create(ctx, first); err != nil {
		t.Fatal(err)
	}

	// Same bytes, same environment: blocked while the first is sending.
	if err := repo.Create(ctx, newSubmission(sha("c"))); !errors.Is(err, domain.ErrFilingAlreadySent) {
		t.Fatalf("duplicate while sending = %v, want ErrFilingAlreadySent", err)
	}

	// Still blocked once sent, unknown and delivered.
	if err := repo.MarkSent(ctx, first.ID, "111", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, newSubmission(sha("c"))); !errors.Is(err, domain.ErrFilingAlreadySent) {
		t.Errorf("duplicate after sent = %v", err)
	}
	if err := repo.UpdateDelivery(ctx, first.ID, domain.SubmissionDelivered, 4, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, newSubmission(sha("c"))); !errors.Is(err, domain.ErrFilingAlreadySent) {
		t.Errorf("duplicate after delivered = %v", err)
	}

	// Other environment, other bytes, other document are independent.
	prod := newSubmission(sha("c"))
	prod.Environment = "production"
	if err := repo.Create(ctx, prod); err != nil {
		t.Errorf("same bytes in another environment: %v", err)
	}
	if err := repo.Create(ctx, newSubmission(sha("d"))); err != nil {
		t.Errorf("other bytes: %v", err)
	}
	otherDoc := newSubmission(sha("c"))
	otherDoc.DocumentID = 8
	if err := repo.Create(ctx, otherDoc); err != nil {
		t.Errorf("other document: %v", err)
	}
}

func TestFilingSubmissionRepository_FailedAndUndeliverableCanBeResent(t *testing.T) {
	repo := NewFilingSubmissionRepository(testutil.NewTestDB(t))
	ctx := context.Background()

	failed := newSubmission(sha("e"))
	if err := repo.Create(ctx, failed); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkFailed(ctx, failed.ID, domain.SubmissionFailed, "ISDS status 1234: fake"); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}
	got, _ := repo.GetByID(ctx, failed.ID)
	if got.Status != domain.SubmissionFailed || got.ErrorText != "ISDS status 1234: fake" {
		t.Errorf("after MarkFailed = %+v", got)
	}
	retry := newSubmission(sha("e"))
	if err := repo.Create(ctx, retry); err != nil {
		t.Fatalf("resend after a failed attempt: %v", err)
	}

	if err := repo.MarkSent(ctx, retry.ID, "222", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateDelivery(ctx, retry.ID, domain.SubmissionUndeliverable, 8, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, newSubmission(sha("e"))); err != nil {
		t.Errorf("resend after undeliverable: %v", err)
	}
}

func TestFilingSubmissionRepository_UnknownBlocksResend(t *testing.T) {
	repo := NewFilingSubmissionRepository(testutil.NewTestDB(t))
	ctx := context.Background()
	sub := newSubmission(sha("f"))
	if err := repo.Create(ctx, sub); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkFailed(ctx, sub.ID, domain.SubmissionUnknown, "ISDS request timed out"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, newSubmission(sha("f"))); !errors.Is(err, domain.ErrFilingAlreadySent) {
		t.Errorf("resend after an unknown outcome = %v, want ErrFilingAlreadySent", err)
	}
}

func TestFilingSubmissionRepository_StateGuards(t *testing.T) {
	repo := NewFilingSubmissionRepository(testutil.NewTestDB(t))
	ctx := context.Background()
	sub := newSubmission(sha("g"))
	if err := repo.Create(ctx, sub); err != nil {
		t.Fatal(err)
	}

	if err := repo.MarkFailed(ctx, sub.ID, domain.SubmissionSent, "x"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("MarkFailed with a sent status = %v", err)
	}
	if err := repo.UpdateDelivery(ctx, sub.ID, domain.SubmissionDelivered, 4, nil); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdateDelivery of a sending row = %v, want ErrNotFound", err)
	}
	if err := repo.UpdateDelivery(ctx, sub.ID, domain.SubmissionUnknown, 4, nil); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("UpdateDelivery with an unknown status = %v", err)
	}
	if err := repo.MarkSent(ctx, sub.ID, "333", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkSent(ctx, sub.ID, "444", time.Now()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("second MarkSent = %v, want ErrNotFound", err)
	}
	if err := repo.MarkFailed(ctx, sub.ID, domain.SubmissionFailed, "late"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("MarkFailed of a sent row = %v, want ErrNotFound", err)
	}
	if _, err := repo.GetByID(ctx, 9999); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetByID of a missing row = %v", err)
	}
}

func TestFilingSubmissionRepository_ListByDocument(t *testing.T) {
	repo := NewFilingSubmissionRepository(testutil.NewTestDB(t))
	ctx := context.Background()

	empty, err := repo.ListByDocument(ctx, domain.FilingKindVATReturn, 7)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty list = %v, %v; want a non-nil empty slice", empty, err)
	}

	a, b := newSubmission(sha("1")), newSubmission(sha("2"))
	other := newSubmission(sha("3"))
	other.DocumentKind = domain.FilingKindVIESSummary
	for _, s := range []*domain.FilingSubmission{a, b, other} {
		if err := repo.Create(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
	list, err := repo.ListByDocument(ctx, domain.FilingKindVATReturn, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != b.ID || list[1].ID != a.ID {
		t.Errorf("list = %+v, want the two vat-return submissions newest first", list)
	}
}

func TestFilingSubmissionRepository_ConstraintsRejectBadValues(t *testing.T) {
	repo := NewFilingSubmissionRepository(testutil.NewTestDB(t))
	ctx := context.Background()

	bad := newSubmission("short")
	if err := repo.Create(ctx, bad); err == nil {
		t.Error("a short sha256 was accepted")
	}
	bad = newSubmission(sha("a"))
	bad.Environment = "staging"
	if err := repo.Create(ctx, bad); err == nil {
		t.Error("an unknown environment was accepted")
	}
	bad = newSubmission(sha("a"))
	bad.DocumentKind = "health-insurance"
	if err := repo.Create(ctx, bad); err == nil {
		t.Error("an unsupported kind was accepted")
	}
}
