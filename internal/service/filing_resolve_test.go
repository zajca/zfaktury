package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/isds"
)

// sendWithUnknownOutcome leaves a production submission in unknown: the
// transport failed while CreateMessage was in flight.
func sendWithUnknownOutcome(t *testing.T, h *filingHarness) *domain.FilingSubmission {
	t.Helper()
	h.client.createErr = fmt.Errorf("CreateMessage: %w", isds.ErrISDSUnavailable)
	p, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
		FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken})
	h.client.createErr = nil
	subs, err := h.repo.ListByDocument(context.Background(), domain.FilingKindVATReturn, 7)
	if err != nil || len(subs) != 1 || subs[0].Status != domain.SubmissionUnknown {
		t.Fatalf("setup: submissions = %+v, %v; want one unknown", subs, err)
	}
	return &subs[0]
}

func TestFilingService_ResolveAttachMovesToSentAndRefreshFilesOnDelivery(t *testing.T) {
	h := newFilingHarness(t)
	sub := sendWithUnknownOutcome(t, h)

	got, err := h.svc.Resolve(context.Background(), sub.ID, FilingResolveRequest{Action: ResolveAttach, DmID: " 987654321 "})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Status != domain.SubmissionSent || got.DmID != "987654321" || got.SentAt == nil {
		t.Fatalf("resolved submission = %+v", got)
	}
	if len(h.doc.markCalls) != 0 {
		t.Fatal("attaching a message id marked the document filed")
	}
	if !strings.Contains(h.logs.String(), `"resolution":"attach"`) {
		t.Errorf("no log line for the resolution: %s", h.logs.String())
	}

	at := time.Date(2026, 11, 24, 12, 7, 0, 0, time.UTC)
	h.client.delivery = deliveredInfo(isds.StatusDelivered, &at)
	h.client.zfo = []byte("zfo")
	res, err := h.svc.Refresh(context.Background(), sub.ID)
	if err != nil || !res.DocumentFiled || len(h.doc.markCalls) != 1 || !h.doc.markCalls[0].Equal(at) {
		t.Fatalf("Refresh after attach = %+v, %v (marks %v)", res, err, h.doc.markCalls)
	}
}

func TestFilingService_ResolveNotSentReleasesTheDocument(t *testing.T) {
	h := newFilingHarness(t)
	sub := sendWithUnknownOutcome(t, h)

	// Blocked until the user settles it.
	p, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatal(err)
	}
	req := FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken}
	if _, err := h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7, req); !errors.Is(err, domain.ErrFilingAlreadySent) {
		t.Fatalf("send while unknown = %v, want ErrFilingAlreadySent", err)
	}

	got, err := h.svc.Resolve(context.Background(), sub.ID, FilingResolveRequest{Action: ResolveNotSent})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Status != domain.SubmissionFailed || got.ErrorText == "" || got.DmID != "" {
		t.Fatalf("resolved submission = %+v", got)
	}
	if !strings.Contains(h.logs.String(), `"resolution":"notSent"`) {
		t.Errorf("no log line for the resolution: %s", h.logs.String())
	}

	// The same bytes can be previewed and sent again.
	h.previewSend(t, "production")
	if len(h.client.created) != 2 {
		t.Errorf("CreateMessage calls = %d, want the failed attempt and the new send", len(h.client.created))
	}
}

func TestFilingService_ResolveWorksOnAStuckSendingRow(t *testing.T) {
	h := newFilingHarness(t)
	p, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatal(err)
	}
	stuck := &domain.FilingSubmission{
		DocumentKind: domain.FilingKindVATReturn, DocumentID: 7, Channel: domain.FilingChannelISDS,
		Environment: "production", RecipientBoxID: p.RecipientBoxID, RecipientName: p.RecipientName,
		FileName: p.FileName, XMLSHA256: p.SHA256, Status: domain.SubmissionSending,
	}
	if err := h.repo.Create(context.Background(), stuck); err != nil {
		t.Fatal(err)
	}
	got, err := h.svc.Resolve(context.Background(), stuck.ID, FilingResolveRequest{Action: ResolveAttach, DmID: "55"})
	if err != nil || got.Status != domain.SubmissionSent {
		t.Fatalf("Resolve of a sending row = %+v, %v", got, err)
	}
}

func TestFilingService_ResolveRefusals(t *testing.T) {
	ctx := context.Background()

	t.Run("a submission in another status", func(t *testing.T) {
		h := newFilingHarness(t)
		_, sent := h.previewSend(t, "production")
		for _, req := range []FilingResolveRequest{
			{Action: ResolveNotSent},
			{Action: ResolveAttach, DmID: "123"},
		} {
			if _, err := h.svc.Resolve(ctx, sent.ID, req); !errors.Is(err, domain.ErrSubmissionNotResolvable) {
				t.Errorf("Resolve(%s) of a sent submission = %v, want ErrSubmissionNotResolvable", req.Action, err)
			}
		}
		got, _ := h.repo.GetByID(ctx, sent.ID)
		if got.Status != domain.SubmissionSent || got.DmID != "1234567890" {
			t.Errorf("a refused resolve changed the submission: %+v", got)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		h := newFilingHarness(t)
		sub := sendWithUnknownOutcome(t, h)
		for name, req := range map[string]FilingResolveRequest{
			"unknown action":      {Action: "retry"},
			"empty action":        {},
			"attach without id":   {Action: ResolveAttach},
			"attach with letters": {Action: ResolveAttach, DmID: "12ab"},
			"attach too long":     {Action: ResolveAttach, DmID: strings.Repeat("1", 21)},
			"attach with spaces":  {Action: ResolveAttach, DmID: "12 34"},
			"notSent with an id":  {Action: ResolveNotSent, DmID: "123"},
		} {
			var userMessage *domain.UserMessageError
			if _, err := h.svc.Resolve(ctx, sub.ID, req); !errors.As(err, &userMessage) {
				t.Errorf("%s: Resolve = %v, want a user message error", name, err)
			}
		}
		if got, _ := h.repo.GetByID(ctx, sub.ID); got.Status != domain.SubmissionUnknown {
			t.Errorf("invalid input changed the submission to %s", got.Status)
		}
	})

	t.Run("missing submission", func(t *testing.T) {
		h := newFilingHarness(t)
		if _, err := h.svc.Resolve(ctx, 999, FilingResolveRequest{Action: ResolveNotSent}); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("Resolve = %v, want ErrNotFound", err)
		}
	})

	t.Run("a send of the document is running", func(t *testing.T) {
		h := newFilingHarness(t)
		sub := sendWithUnknownOutcome(t, h)
		if !h.svc.acquire(domain.FilingKindVATReturn, 7) {
			t.Fatal("could not take the in-flight lock")
		}
		defer h.svc.release(domain.FilingKindVATReturn, 7)
		if _, err := h.svc.Resolve(ctx, sub.ID, FilingResolveRequest{Action: ResolveNotSent}); !errors.Is(err, domain.ErrFilingSendInProgress) {
			t.Errorf("Resolve during a send = %v, want ErrFilingSendInProgress", err)
		}
	})
}

func TestFilingService_RecoverInterruptedMarksOnlyStaleSendingRows(t *testing.T) {
	ctx := context.Background()
	h := newFilingHarness(t)
	p, err := h.svc.Preview(ctx, domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatal(err)
	}
	newRow := func(status domain.FilingSubmissionStatus, env string, sha string) *domain.FilingSubmission {
		sub := &domain.FilingSubmission{
			DocumentKind: domain.FilingKindVATReturn, DocumentID: 7, Channel: domain.FilingChannelISDS,
			Environment: env, RecipientBoxID: p.RecipientBoxID, RecipientName: p.RecipientName,
			FileName: p.FileName, XMLSHA256: sha, Status: status,
		}
		if err := h.repo.Create(ctx, sub); err != nil {
			t.Fatal(err)
		}
		return sub
	}
	sending := newRow(domain.SubmissionSending, "production", p.SHA256)
	other := newRow(domain.SubmissionSent, "test", p.SHA256)

	// A send younger than the request timeout may still be running.
	*h.clock = time.Now().Add(isds.DefaultTimeout / 2)
	if n, err := h.svc.RecoverInterrupted(ctx); err != nil || n != 0 {
		t.Fatalf("RecoverInterrupted of a fresh row = %d, %v", n, err)
	}
	if got, _ := h.repo.GetByID(ctx, sending.ID); got.Status != domain.SubmissionSending {
		t.Fatalf("a fresh sending row became %s", got.Status)
	}

	*h.clock = time.Now().Add(isds.DefaultTimeout + 2*time.Minute)
	n, err := h.svc.RecoverInterrupted(ctx)
	if err != nil || n != 1 {
		t.Fatalf("RecoverInterrupted = %d, %v, want 1", n, err)
	}
	got, _ := h.repo.GetByID(ctx, sending.ID)
	if got.Status != domain.SubmissionUnknown || got.ErrorText == "" {
		t.Errorf("stale sending row = %+v, want unknown with a reason", got)
	}
	if got, _ := h.repo.GetByID(ctx, other.ID); got.Status != domain.SubmissionSent {
		t.Errorf("a sent row became %s", got.Status)
	}
	if !strings.Contains(h.logs.String(), "interrupted ISDS submissions marked unknown") {
		t.Errorf("no log line for the recovery: %s", h.logs.String())
	}

	// The recovered row is now settled through Resolve.
	if _, err := h.svc.Resolve(ctx, sending.ID, FilingResolveRequest{Action: ResolveNotSent}); err != nil {
		t.Errorf("Resolve of a recovered row: %v", err)
	}
}

func TestFilingService_FiledDocumentIsNeitherPreviewedNorSent(t *testing.T) {
	ctx := context.Background()
	for _, env := range []string{"production", "test"} {
		t.Run(env, func(t *testing.T) {
			h := newFilingHarness(t)
			p, err := h.svc.Preview(ctx, domain.FilingKindVATReturn, 7, env)
			if err != nil {
				t.Fatal(err)
			}
			h.doc.filed = true
			regen := h.doc.regenCalls

			if _, err := h.svc.Preview(ctx, domain.FilingKindVATReturn, 7, env); !errors.Is(err, domain.ErrFilingAlreadyFiled) {
				t.Errorf("Preview of a filed document = %v, want ErrFilingAlreadyFiled", err)
			}
			_, err = h.svc.Send(ctx, domain.FilingKindVATReturn, 7, FilingSendRequest{Environment: env, ConfirmToken: p.ConfirmToken})
			if !errors.Is(err, domain.ErrFilingAlreadyFiled) {
				t.Errorf("Send of a filed document = %v, want ErrFilingAlreadyFiled", err)
			}
			if h.doc.regenCalls != regen {
				t.Error("a filed document's XML was regenerated")
			}
			if len(h.client.created) != 0 || len(h.factory) != 0 {
				t.Error("a filed document reached ISDS")
			}
			if subs, _ := h.repo.ListByDocument(ctx, domain.FilingKindVATReturn, 7); len(subs) != 0 {
				t.Errorf("a refused send stored %d submissions", len(subs))
			}
		})
	}

	t.Run("state cannot be read", func(t *testing.T) {
		h := newFilingHarness(t)
		h.doc.filedErr = errors.New("database is locked")
		if _, err := h.svc.Preview(ctx, domain.FilingKindVATReturn, 7, "production"); !errors.Is(err, h.doc.filedErr) {
			t.Errorf("Preview = %v, want the read error", err)
		}
		if h.doc.regenCalls != 0 {
			t.Error("the XML was regenerated although the state is unknown")
		}
	})
}

func TestFilingService_SendRefusedAfterASentOrDeliveredProductionSubmission(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		deliver bool
	}{{"sent", false}, {"delivered", true}} {
		t.Run(tc.name, func(t *testing.T) {
			h := newFilingHarness(t)
			_, sub := h.previewSend(t, "production")
			if tc.deliver {
				at := time.Date(2026, 11, 24, 12, 7, 0, 0, time.UTC)
				h.client.delivery = deliveredInfo(isds.StatusDelivered, &at)
				h.client.zfo = []byte("zfo")
				h.doc.markErr = errors.New("database is locked") // not marked filed
				if _, err := h.svc.Refresh(ctx, sub.ID); err != nil {
					t.Fatal(err)
				}
			}

			// Changed data gives other bytes, which the unique index alone lets through.
			h.doc.setXML("<Pisemnost><DPHDP3>fake-body-2</DPHDP3></Pisemnost>")
			p, err := h.svc.Preview(ctx, domain.FilingKindVATReturn, 7, "production")
			if err != nil {
				t.Fatal(err)
			}
			created := len(h.client.created)
			_, err = h.svc.Send(ctx, domain.FilingKindVATReturn, 7, FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken})
			if !errors.Is(err, domain.ErrFilingAlreadySent) {
				t.Errorf("Send = %v, want ErrFilingAlreadySent", err)
			}
			if len(h.client.created) != created {
				t.Error("a second production message was sent")
			}

			// The test box is refused too.
			pt, err := h.svc.Preview(ctx, domain.FilingKindVATReturn, 7, "test")
			if err != nil {
				t.Fatal(err)
			}
			_, err = h.svc.Send(ctx, domain.FilingKindVATReturn, 7, FilingSendRequest{Environment: "test", ConfirmToken: pt.ConfirmToken})
			if !errors.Is(err, domain.ErrFilingAlreadySent) || len(h.client.created) != created {
				t.Errorf("test send after a production send = %v, want ErrFilingAlreadySent", err)
			}
		})
	}
}

func TestFilingService_DeliveredWithoutATimeIsNotMarkedFiledUntilATimeIsKnown(t *testing.T) {
	ctx := context.Background()
	h := newFilingHarness(t)
	_, sub := h.previewSend(t, "production")
	h.client.delivery = deliveredInfo(isds.StatusDelivered, nil) // no delivery or acceptance time
	h.client.zfo = []byte("zfo")

	res, err := h.svc.Refresh(ctx, sub.ID)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if res.Submission.Status != domain.SubmissionDelivered || res.Submission.DeliveredAt != nil || res.DocumentFiled {
		t.Fatalf("result = %+v (filed %v), want delivered without a time and not filed", res.Submission, res.DocumentFiled)
	}
	if len(h.doc.markCalls) != 0 {
		t.Errorf("the document was marked filed at %v without a delivery time", h.doc.markCalls)
	}
	if !strings.Contains(h.logs.String(), `"level":"WARN","msg":"delivered submission has no delivery time`) {
		t.Errorf("no warning for the missing delivery time: %s", h.logs.String())
	}

	// A later refresh learns the time and files the document at it.
	at := time.Date(2026, 11, 24, 12, 7, 0, 0, time.UTC)
	h.client.delivery = deliveredInfo(isds.StatusDelivered, &at)
	res, err = h.svc.Refresh(ctx, sub.ID)
	if err != nil || !res.DocumentFiled || len(h.doc.markCalls) != 1 || !h.doc.markCalls[0].Equal(at) {
		t.Fatalf("second refresh = %+v, %v (marks %v)", res, err, h.doc.markCalls)
	}
}

func TestFilingService_VirusStatusReleasesTheDocumentForANewSend(t *testing.T) {
	ctx := context.Background()
	h := newFilingHarness(t)
	_, sub := h.previewSend(t, "production")
	h.client.delivery = &isds.DeliveryInfo{MessageID: "1234567890", Status: isds.StatusRejectedMalicious}

	res, err := h.svc.Refresh(ctx, sub.ID)
	if err != nil || res.Submission.Status != domain.SubmissionFailed || res.DocumentFiled {
		t.Fatalf("Refresh = %+v, %v, want a failed submission", res, err)
	}
	h.previewSend(t, "production")
}
