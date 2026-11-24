package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestFilingSubmissionRepository_OnlyAUniqueViolationMeansAlreadySent(t *testing.T) {
	repo := NewFilingSubmissionRepository(testutil.NewTestDB(t))
	ctx := context.Background()

	if err := repo.Create(ctx, newSubmission(sha("a"))); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, newSubmission(sha("a"))); !errors.Is(err, domain.ErrFilingAlreadySent) {
		t.Errorf("duplicate = %v, want ErrFilingAlreadySent", err)
	}

	// Other constraint failures are plain errors, whatever their text says.
	for name, mutate := range map[string]func(*domain.FilingSubmission){
		"check on sha256":      func(s *domain.FilingSubmission) { s.XMLSHA256 = "short" },
		"check on environment": func(s *domain.FilingSubmission) { s.Environment = "staging" },
		"check on status":      func(s *domain.FilingSubmission) { s.Status = "lost" },
	} {
		sub := newSubmission(sha("b"))
		mutate(sub)
		err := repo.Create(ctx, sub)
		if err == nil || errors.Is(err, domain.ErrFilingAlreadySent) {
			t.Errorf("%s: Create = %v, want a plain error", name, err)
		}
	}
}

func TestFilingSubmissionRepository_ResolveSentAndNotSent(t *testing.T) {
	repo := NewFilingSubmissionRepository(testutil.NewTestDB(t))
	ctx := context.Background()
	at := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)

	for i, from := range []domain.FilingSubmissionStatus{domain.SubmissionSending, domain.SubmissionUnknown} {
		// Each pass uses its own bytes: a sent row blocks its own sha256.
		sentSHA, unsentSHA := sha(string(rune('c'+2*i))), sha(string(rune('d'+2*i)))
		t.Run("sent from "+string(from), func(t *testing.T) {
			sub := newSubmission(sentSHA)
			sub.Status = from
			if err := repo.Create(ctx, sub); err != nil {
				t.Fatal(err)
			}
			dmID := fmt.Sprintf("42%d", i)
			if err := repo.ResolveSent(ctx, sub.ID, dmID, at); err != nil {
				t.Fatalf("ResolveSent: %v", err)
			}
			got, _ := repo.GetByID(ctx, sub.ID)
			if got.Status != domain.SubmissionSent || got.DmID != dmID || got.SentAt == nil || !got.SentAt.Equal(at) || got.ErrorText != "" {
				t.Errorf("resolved = %+v", got)
			}
			if err := repo.ResolveSent(ctx, sub.ID, "4243", at); !errors.Is(err, domain.ErrSubmissionNotResolvable) {
				t.Errorf("second resolve = %v, want ErrSubmissionNotResolvable", err)
			}
			if got, _ := repo.GetByID(ctx, sub.ID); got.DmID != dmID {
				t.Errorf("a refused resolve changed the message id to %q", got.DmID)
			}
		})

		t.Run("not sent from "+string(from), func(t *testing.T) {
			sub := newSubmission(unsentSHA)
			sub.Status = from
			if err := repo.Create(ctx, sub); err != nil {
				t.Fatal(err)
			}
			if err := repo.ResolveNotSent(ctx, sub.ID, "user says no"); err != nil {
				t.Fatalf("ResolveNotSent: %v", err)
			}
			got, _ := repo.GetByID(ctx, sub.ID)
			if got.Status != domain.SubmissionFailed || got.ErrorText != "user says no" {
				t.Errorf("resolved = %+v", got)
			}
			// failed leaves the unique index, so the same bytes can go again.
			if err := repo.Create(ctx, newSubmission(unsentSHA)); err != nil {
				t.Errorf("Create after ResolveNotSent: %v", err)
			}
		})
	}

	t.Run("other states and missing rows", func(t *testing.T) {
		sub := newSubmission(sha("9"))
		if err := repo.Create(ctx, sub); err != nil {
			t.Fatal(err)
		}
		if err := repo.MarkSent(ctx, sub.ID, "1", at); err != nil {
			t.Fatal(err)
		}
		if err := repo.ResolveNotSent(ctx, sub.ID, "x"); !errors.Is(err, domain.ErrSubmissionNotResolvable) {
			t.Errorf("ResolveNotSent of a sent row = %v, want ErrSubmissionNotResolvable", err)
		}
		if err := repo.ResolveSent(ctx, 9999, "1", at); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("ResolveSent of a missing row = %v, want ErrNotFound", err)
		}
	})
}

func TestFilingSubmissionRepository_MarkStaleSendingUnknown(t *testing.T) {
	repo := NewFilingSubmissionRepository(testutil.NewTestDB(t))
	ctx := context.Background()

	sending := newSubmission(sha("a"))
	if err := repo.Create(ctx, sending); err != nil {
		t.Fatal(err)
	}
	sent := newSubmission(sha("b"))
	if err := repo.Create(ctx, sent); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkSent(ctx, sent.ID, "1", time.Now()); err != nil {
		t.Fatal(err)
	}

	// Rows created at or after the cutoff are left alone.
	n, err := repo.MarkStaleSendingUnknown(ctx, time.Now().Add(-time.Hour), "interrupted")
	if err != nil || n != 0 {
		t.Fatalf("with an old cutoff: %d, %v", n, err)
	}
	n, err = repo.MarkStaleSendingUnknown(ctx, time.Now().Add(time.Hour), "interrupted")
	if err != nil || n != 1 {
		t.Fatalf("with a future cutoff: %d, %v, want 1", n, err)
	}
	if got, _ := repo.GetByID(ctx, sending.ID); got.Status != domain.SubmissionUnknown || got.ErrorText != "interrupted" {
		t.Errorf("sending row = %+v", got)
	}
	if got, _ := repo.GetByID(ctx, sent.ID); got.Status != domain.SubmissionSent {
		t.Errorf("sent row became %s", got.Status)
	}
	// Unknown still blocks the same bytes.
	if err := repo.Create(ctx, newSubmission(sha("a"))); !errors.Is(err, domain.ErrFilingAlreadySent) {
		t.Errorf("Create over an unknown row = %v, want ErrFilingAlreadySent", err)
	}
}

func TestFilingSubmissionRepository_ResolveSentRefusesAMessageIDOfAnotherSubmission(t *testing.T) {
	repo := NewFilingSubmissionRepository(testutil.NewTestDB(t))
	ctx := context.Background()
	at := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)

	first := newSubmission(sha("a"))
	if err := repo.Create(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := repo.ResolveSent(ctx, first.ID, "777", at); err != nil {
		t.Fatal(err)
	}

	second := newSubmission(sha("b"))
	if err := repo.Create(ctx, second); err != nil {
		t.Fatal(err)
	}
	if err := repo.ResolveSent(ctx, second.ID, "777", at); !errors.Is(err, domain.ErrMessageIDInUse) {
		t.Fatalf("same id, same environment = %v, want ErrMessageIDInUse", err)
	}
	if got, _ := repo.GetByID(ctx, second.ID); got.Status != domain.SubmissionSending || got.DmID != "" {
		t.Errorf("a refused attach changed the row: %+v", got)
	}

	// Message ids of the test and production services are separate spaces.
	prod := newSubmission(sha("c"))
	prod.Environment = "production"
	if err := repo.Create(ctx, prod); err != nil {
		t.Fatal(err)
	}
	if err := repo.ResolveSent(ctx, prod.ID, "777", at); err != nil {
		t.Errorf("same id, other environment: %v", err)
	}
}

func TestFilingSubmissionRepository_RevertAttachedOnlyForAnAttachedSentRow(t *testing.T) {
	repo := NewFilingSubmissionRepository(testutil.NewTestDB(t))
	ctx := context.Background()
	at := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)

	attached := newSubmission(sha("a"))
	own := newSubmission(sha("b"))
	for _, s := range []*domain.FilingSubmission{attached, own} {
		if err := repo.Create(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.ResolveSent(ctx, attached.ID, "501", at); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkSent(ctx, own.ID, "502", at); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.GetByID(ctx, attached.ID); !got.DmIDAttached {
		t.Fatal("an attached id is not flagged")
	}
	if got, _ := repo.GetByID(ctx, own.ID); got.DmIDAttached {
		t.Fatal("an id from CreateMessage is flagged as attached")
	}

	if err := repo.RevertAttached(ctx, own.ID, "502", "x"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("revert of an own id = %v, want ErrNotFound", err)
	}
	if err := repo.RevertAttached(ctx, attached.ID, "999", "x"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("revert with another id = %v, want ErrNotFound", err)
	}
	if err := repo.RevertAttached(ctx, attached.ID, "501", "wrong message"); err != nil {
		t.Fatalf("RevertAttached: %v", err)
	}
	got, _ := repo.GetByID(ctx, attached.ID)
	if got.Status != domain.SubmissionUnknown || got.DmID != "" || got.DmIDAttached || got.SentAt != nil || got.ErrorText != "wrong message" {
		t.Errorf("reverted row = %+v", got)
	}
	if err := repo.RevertAttached(ctx, attached.ID, "501", "x"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("second revert = %v, want ErrNotFound", err)
	}
	if err := repo.ResolveSent(ctx, attached.ID, "501", at); err != nil {
		t.Errorf("the freed id cannot be attached again: %v", err)
	}
}

func TestFilingSubmissionRepository_UpdateDeliveryCanFailASentSubmission(t *testing.T) {
	repo := NewFilingSubmissionRepository(testutil.NewTestDB(t))
	ctx := context.Background()
	sub := newSubmission(sha("a"))
	if err := repo.Create(ctx, sub); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkSent(ctx, sub.ID, "1", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateDelivery(ctx, sub.ID, domain.SubmissionFailed, 3, nil); err != nil {
		t.Fatalf("UpdateDelivery to failed: %v", err)
	}
	if got, _ := repo.GetByID(ctx, sub.ID); got.Status != domain.SubmissionFailed || got.DmMessageStatus == nil || *got.DmMessageStatus != 3 {
		t.Errorf("submission = %+v", got)
	}
	if err := repo.Create(ctx, newSubmission(sha("a"))); err != nil {
		t.Errorf("Create after a failed delivery: %v", err)
	}
}
