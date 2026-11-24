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

	at := sub.CreatedAt.Add(7 * time.Minute)
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
	if _, err := h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7, req); !errors.Is(err, domain.ErrFilingOutcomeUnknown) {
		t.Fatalf("send while unknown = %v, want ErrFilingOutcomeUnknown", err)
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

func TestFilingService_UnsettledProductionSendBlocksChangedBytes(t *testing.T) {
	ctx := context.Background()
	h := newFilingHarness(t)
	sendWithUnknownOutcome(t, h)

	// Changed data gives other bytes, which the unique index alone lets through.
	h.doc.setXML("<Pisemnost><DPHDP3>fake-body-2</DPHDP3></Pisemnost>")
	p, err := h.svc.Preview(ctx, domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatal(err)
	}
	created := len(h.client.created)
	_, err = h.svc.Send(ctx, domain.FilingKindVATReturn, 7, FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken})
	if !errors.Is(err, domain.ErrFilingOutcomeUnknown) || errors.Is(err, domain.ErrFilingAlreadySent) {
		t.Fatalf("Send = %v, want ErrFilingOutcomeUnknown only", err)
	}
	if len(h.client.created) != created {
		t.Errorf("CreateMessage was called %d more times", len(h.client.created)-created)
	}
}

func TestFilingService_SendingProductionRowBlocksChangedBytes(t *testing.T) {
	ctx := context.Background()
	h := newFilingHarness(t)
	// The row of an interrupted send that the startup sweep has not reached yet.
	if err := h.repo.Create(ctx, &domain.FilingSubmission{
		DocumentKind: domain.FilingKindVATReturn, DocumentID: 7, Channel: domain.FilingChannelISDS,
		Environment: "production", RecipientBoxID: "7nyn2d9", RecipientName: "x", DmAnnotation: "x",
		FileName: "x.xml", XMLSHA256: sha256Hex("older bytes"), Status: domain.SubmissionSending,
	}); err != nil {
		t.Fatal(err)
	}
	p, err := h.svc.Preview(ctx, domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.svc.Send(ctx, domain.FilingKindVATReturn, 7, FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken})
	if !errors.Is(err, domain.ErrFilingOutcomeUnknown) || len(h.client.created) != 0 {
		t.Errorf("Send = %v after %d CreateMessage calls, want ErrFilingOutcomeUnknown and none", err, len(h.client.created))
	}
}

func TestFilingService_NotSentReleasesTheDocumentForChangedBytes(t *testing.T) {
	ctx := context.Background()
	h := newFilingHarness(t)
	sub := sendWithUnknownOutcome(t, h)
	if _, err := h.svc.Resolve(ctx, sub.ID, FilingResolveRequest{Action: ResolveNotSent}); err != nil {
		t.Fatal(err)
	}
	h.doc.setXML("<Pisemnost><DPHDP3>fake-body-2</DPHDP3></Pisemnost>")
	h.previewSend(t, "production")
}

func TestFilingService_SameBytesOfAnUnsettledTestSendAreRefusedAsUnknown(t *testing.T) {
	ctx := context.Background()
	h := newFilingHarness(t)
	h.client.createErr = fmt.Errorf("CreateMessage: %w", isds.ErrISDSUnavailable)
	p, err := h.svc.Preview(ctx, domain.FilingKindVATReturn, 7, "test")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = h.svc.Send(ctx, domain.FilingKindVATReturn, 7, FilingSendRequest{Environment: "test", ConfirmToken: p.ConfirmToken})
	h.client.createErr = nil

	p, err = h.svc.Preview(ctx, domain.FilingKindVATReturn, 7, "test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.svc.Send(ctx, domain.FilingKindVATReturn, 7, FilingSendRequest{Environment: "test", ConfirmToken: p.ConfirmToken})
	if !errors.Is(err, domain.ErrFilingOutcomeUnknown) {
		t.Errorf("Send = %v, want ErrFilingOutcomeUnknown", err)
	}
}

func TestFilingService_ResolveAttachRefusesAMessageIDOfAnotherSubmission(t *testing.T) {
	ctx := context.Background()
	h := newFilingHarness(t)
	h.client.createID = "1234567890"
	h.previewSend(t, "production") // holds message id 1234567890

	other := &domain.FilingSubmission{
		DocumentKind: domain.FilingKindVATReturn, DocumentID: 8, Channel: domain.FilingChannelISDS,
		Environment: "production", RecipientBoxID: "7nyn2d9", RecipientName: "x", DmAnnotation: "x",
		FileName: "x.xml", XMLSHA256: sha256Hex("another"), Status: domain.SubmissionUnknown,
	}
	if err := h.repo.Create(ctx, other); err != nil {
		t.Fatal(err)
	}

	_, err := h.svc.Resolve(ctx, other.ID, FilingResolveRequest{Action: ResolveAttach, DmID: "1234567890"})
	if !errors.Is(err, domain.ErrMessageIDInUse) {
		t.Fatalf("Resolve = %v, want ErrMessageIDInUse", err)
	}
	if got, _ := h.repo.GetByID(ctx, other.ID); got.Status != domain.SubmissionUnknown || got.DmID != "" {
		t.Errorf("a refused attach changed the submission: %+v", got)
	}
	if _, err := h.svc.Resolve(ctx, other.ID, FilingResolveRequest{Action: ResolveAttach, DmID: "1234567891"}); err != nil {
		t.Errorf("Resolve with a free message id: %v", err)
	}
}

func TestFilingService_RevertedAttachCanBeDeclaredNotSent(t *testing.T) {
	ctx := context.Background()
	h := newFilingHarness(t)
	sub := sendWithUnknownOutcome(t, h)
	if _, err := h.svc.Resolve(ctx, sub.ID, FilingResolveRequest{Action: ResolveAttach, DmID: "987654321"}); err != nil {
		t.Fatal(err)
	}
	h.client.delivery = &isds.DeliveryInfo{Status: isds.StatusDelivered, RecipientBoxID: "zzzzzzz", Annotation: "x"}
	h.client.deliveryExact = true
	if _, err := h.svc.Refresh(ctx, sub.ID); !errors.Is(err, domain.ErrMessageMismatch) {
		t.Fatalf("Refresh = %v, want ErrMessageMismatch", err)
	}
	got, err := h.svc.Resolve(ctx, sub.ID, FilingResolveRequest{Action: ResolveNotSent})
	if err != nil || got.Status != domain.SubmissionFailed {
		t.Errorf("Resolve notSent = %+v, %v", got, err)
	}
}

func TestFilingService_MismatchOfOwnMessageIDKeepsTheSubmissionSent(t *testing.T) {
	ctx := context.Background()
	h := newFilingHarness(t)
	_, sub := h.previewSend(t, "production")
	h.client.delivery = &isds.DeliveryInfo{Status: isds.StatusDelivered, RecipientBoxID: "zzzzzzz", Annotation: "x"}
	h.client.deliveryExact = true

	if _, err := h.svc.Refresh(ctx, sub.ID); !errors.Is(err, domain.ErrMessageMismatch) {
		t.Fatalf("Refresh = %v, want ErrMessageMismatch", err)
	}
	got, _ := h.repo.GetByID(ctx, sub.ID)
	if got.Status != domain.SubmissionSent || got.DmID != sub.DmID || got.DmIDAttached {
		t.Errorf("submission = %+v, want it unchanged and sent", got)
	}
	if !strings.Contains(h.logs.String(), `"level":"ERROR","msg":"ISDS reports another recipient`) {
		t.Errorf("no error log for the ISDS inconsistency: %s", h.logs.String())
	}
	if actions := h.audit.actions(sub.ID); hasAuditAction(actions, "isds_attach_reverted") {
		t.Errorf("audit actions = %v, an own message id must not be reverted", actions)
	}
}

func hasAuditAction(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// attachAndDeliver attaches a user-typed message id to an unknown submission
// and makes ISDS report a delivery for it; the evidence names the message the
// fake created, so only its times can disqualify it.
func attachAndDeliver(t *testing.T, h *filingHarness, info *isds.DeliveryInfo) *domain.FilingSubmission {
	t.Helper()
	sub := sendWithUnknownOutcome(t, h)
	if _, err := h.svc.Resolve(context.Background(), sub.ID, FilingResolveRequest{Action: ResolveAttach, DmID: "987654321"}); err != nil {
		t.Fatal(err)
	}
	h.client.delivery = info
	h.client.zfo = []byte("zfo")
	return sub
}

func TestFilingService_RefreshRefusesAnAttachedMessageDeliveredBeforeTheSubmission(t *testing.T) {
	ctx := context.Background()
	for name, setTimes := range map[string]func(info *isds.DeliveryInfo, created time.Time){
		"delivery time":   func(i *isds.DeliveryInfo, c time.Time) { at := c.Add(-72 * time.Hour); i.DeliveryTime = &at },
		"acceptance time": func(i *isds.DeliveryInfo, c time.Time) { at := c.Add(-72 * time.Hour); i.AcceptanceTime = &at },
		"just past the allowance": func(i *isds.DeliveryInfo, c time.Time) {
			at := c.Add(-attachedClockSkew - 2*time.Minute)
			i.DeliveryTime = &at
		},
		"other offset": func(i *isds.DeliveryInfo, c time.Time) {
			at := c.Add(-48 * time.Hour).In(time.FixedZone("", 7200))
			i.DeliveryTime = &at
		},
	} {
		t.Run(name, func(t *testing.T) {
			h := newFilingHarness(t)
			info := &isds.DeliveryInfo{MessageID: "987654321", Status: isds.StatusDelivered}
			sub := sendWithUnknownOutcome(t, h)
			setTimes(info, sub.CreatedAt)
			if _, err := h.svc.Resolve(ctx, sub.ID, FilingResolveRequest{Action: ResolveAttach, DmID: "987654321"}); err != nil {
				t.Fatal(err)
			}
			h.client.delivery, h.client.zfo = info, []byte("zfo")

			if _, err := h.svc.Refresh(ctx, sub.ID); !errors.Is(err, domain.ErrMessageMismatch) {
				t.Fatalf("Refresh = %v, want ErrMessageMismatch", err)
			}
			if len(h.doc.markCalls) != 0 || h.client.zfoHit != 0 {
				t.Errorf("an older message filed the document (%v) or its receipt was downloaded (%d)", h.doc.markCalls, h.client.zfoHit)
			}
			got, _ := h.repo.GetByID(ctx, sub.ID)
			if got.Status != domain.SubmissionUnknown || got.DmID != "" || got.DmIDAttached || got.DeliveredAt != nil {
				t.Errorf("submission = %+v, want unknown without the attached id", got)
			}
			if actions := h.audit.actions(sub.ID); !hasAuditAction(actions, "isds_attach_reverted") {
				t.Errorf("audit actions = %v, want isds_attach_reverted", actions)
			}
		})
	}
}

func TestFilingService_RefreshAcceptsAnAttachedMessageWithinTheClockSkew(t *testing.T) {
	ctx := context.Background()
	h := newFilingHarness(t)
	sub := sendWithUnknownOutcome(t, h)
	at := sub.CreatedAt.Add(-10 * time.Minute)
	if _, err := h.svc.Resolve(ctx, sub.ID, FilingResolveRequest{Action: ResolveAttach, DmID: "987654321"}); err != nil {
		t.Fatal(err)
	}
	h.client.delivery = &isds.DeliveryInfo{MessageID: "987654321", Status: isds.StatusDelivered, DeliveryTime: &at, AcceptanceTime: &at}
	h.client.zfo = []byte("zfo")

	res, err := h.svc.Refresh(ctx, sub.ID)
	if err != nil || !res.DocumentFiled || res.Submission.Status != domain.SubmissionDelivered {
		t.Fatalf("Refresh = %+v, %v; want delivered and filed", res, err)
	}
	if actions := h.audit.actions(sub.ID); hasAuditAction(actions, "isds_attach_reverted") {
		t.Errorf("audit actions = %v, a message within the skew must not be reverted", actions)
	}
}

func TestFilingService_RefreshAcceptsAnAttachedMessageWithoutTimes(t *testing.T) {
	ctx := context.Background()
	h := newFilingHarness(t)
	sub := attachAndDeliver(t, h, &isds.DeliveryInfo{MessageID: "987654321", Status: isds.StatusStamped})

	res, err := h.svc.Refresh(ctx, sub.ID)
	if err != nil || res.Submission.Status != domain.SubmissionSent {
		t.Fatalf("Refresh = %+v, %v; want the evidence accepted as sent", res, err)
	}
}

func TestFilingService_RefreshIgnoresTheTimeOfAnOwnMessage(t *testing.T) {
	ctx := context.Background()
	h := newFilingHarness(t)
	_, sub := h.previewSend(t, "production")
	// Own ids are ours by construction; a skewed ISDS clock must not undo them.
	at := sub.CreatedAt.Add(-72 * time.Hour)
	h.client.delivery = deliveredInfo(isds.StatusDelivered, &at)
	h.client.zfo = []byte("zfo")

	res, err := h.svc.Refresh(ctx, sub.ID)
	if err != nil || res.Submission.Status != domain.SubmissionDelivered || !res.DocumentFiled {
		t.Fatalf("Refresh = %+v, %v; want delivered and filed", res, err)
	}
	if actions := h.audit.actions(sub.ID); hasAuditAction(actions, "isds_attach_reverted") {
		t.Errorf("audit actions = %v, an own message must not be reverted", actions)
	}
}

func TestFilingService_RefreshIgnoresAMessageOfAnotherRecipientOrSubject(t *testing.T) {
	ctx := context.Background()
	for name, mutate := range map[string]func(*isds.DeliveryInfo){
		"other recipient": func(i *isds.DeliveryInfo) { i.RecipientBoxID = "zzzzzzz" },
		"other subject":   func(i *isds.DeliveryInfo) { i.Annotation = "Jiná zpráva" },
		"no recipient":    func(i *isds.DeliveryInfo) { i.RecipientBoxID = "" },
		"no subject":      func(i *isds.DeliveryInfo) { i.Annotation = "" },
	} {
		t.Run(name, func(t *testing.T) {
			h := newFilingHarness(t)
			sub := sendWithUnknownOutcome(t, h)
			if _, err := h.svc.Resolve(ctx, sub.ID, FilingResolveRequest{Action: ResolveAttach, DmID: "987654321"}); err != nil {
				t.Fatal(err)
			}
			created := h.client.created[len(h.client.created)-1]
			at := sub.CreatedAt.Add(7 * time.Minute)
			info := deliveredInfo(isds.StatusDelivered, &at)
			info.RecipientBoxID, info.Annotation = created.RecipientBoxID, created.Annotation
			mutate(info)
			h.client.delivery, h.client.deliveryExact = info, true
			h.client.zfo = []byte("zfo")

			_, err := h.svc.Refresh(ctx, sub.ID)
			if !errors.Is(err, domain.ErrMessageMismatch) {
				t.Fatalf("Refresh = %v, want ErrMessageMismatch", err)
			}
			if len(h.doc.markCalls) != 0 || h.client.zfoHit != 0 {
				t.Errorf("a foreign message filed the document (%v) or its receipt was downloaded (%d)", h.doc.markCalls, h.client.zfoHit)
			}
			got, _ := h.repo.GetByID(ctx, sub.ID)
			// The id was typed by the user: it is dropped and the submission can be settled again.
			if got.Status != domain.SubmissionUnknown || got.DmID != "" || got.DmIDAttached || got.SentAt != nil ||
				got.DeliveredAt != nil || got.DmMessageStatus != nil || got.HasReceipt || got.ErrorText == "" {
				t.Errorf("after a mismatch the submission = %+v, want unknown without a message id", got)
			}
			if !strings.Contains(h.logs.String(), "returned to unknown") {
				t.Errorf("no log line for the reverted attach: %s", h.logs.String())
			}
			if actions := h.audit.actions(sub.ID); !hasAuditAction(actions, "isds_attach_reverted") {
				t.Errorf("audit actions = %v, want isds_attach_reverted", actions)
			}
			if _, err := h.svc.Resolve(ctx, sub.ID, FilingResolveRequest{Action: ResolveAttach, DmID: "987654322"}); err != nil {
				t.Errorf("attaching the right id after the revert: %v", err)
			}
		})
	}
}
