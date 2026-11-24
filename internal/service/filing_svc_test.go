package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/epo"
	"github.com/zajca/zfaktury/internal/isds"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/secrets"
	"github.com/zajca/zfaktury/internal/testutil"
	"github.com/zajca/zfaktury/internal/xsdcheck"
)

const (
	filingTestUsername = "dsuser-fake-9"
	filingTestPassword = "Fake-Pass-Word-77!"
	filingTestXML      = "<Pisemnost><DPHDP3>fake-body-1</DPHDP3></Pisemnost>"
)

type fakeFilingDoc struct {
	mu          sync.Mutex
	xml         []byte
	regenErr    error
	regenCalls  int
	markErr     error
	markCalls   []time.Time
	alreadyDone bool
}

func (d *fakeFilingDoc) RegenerateXML(_ context.Context, _ int64) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.regenCalls++
	return d.xml, d.regenErr
}

func (d *fakeFilingDoc) MarkFiledAt(_ context.Context, _ int64, at time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.markCalls = append(d.markCalls, at)
	if d.markErr != nil {
		return d.markErr
	}
	if d.alreadyDone {
		return domain.ErrFilingAlreadyFiled
	}
	d.alreadyDone = true
	return nil
}

func (d *fakeFilingDoc) setXML(x string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.xml = []byte(x)
}

type fakeFilingEPO struct {
	findings []epo.Finding
	err      error
	calls    int
	got      []byte
}

func (f *fakeFilingEPO) Check(_ context.Context, xmlData []byte) ([]epo.Finding, error) {
	f.calls++
	f.got = xmlData
	return f.findings, f.err
}

type fakeFilingXSD struct {
	err    error
	calls  int
	schema xsdcheck.Schema
}

func (f *fakeFilingXSD) Check(_ context.Context, schema xsdcheck.Schema, _ []byte) error {
	f.calls++
	f.schema = schema
	return f.err
}

type fakeFilingSettings struct{ values map[string]string }

func (f *fakeFilingSettings) Get(_ context.Context, key string) (string, error) {
	if v, ok := f.values[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("setting %q not found: %w", key, sql.ErrNoRows)
}

type fakeISDSClient struct {
	mu          sync.Mutex
	created     []isds.MessageInput
	createErr   error
	createID    string
	createGate  chan struct{} // when set, CreateMessage blocks until it is closed
	createEnter chan struct{} // receives once CreateMessage is entered
	delivery    *isds.DeliveryInfo
	deliveryErr error
	deliveryHit int
	zfo         []byte
	zfoErr      error
	zfoHit      int
	owner       *isds.OwnerInfo
	user        *isds.UserInfo
	ownerErr    error
}

func (c *fakeISDSClient) CreateMessage(_ context.Context, m isds.MessageInput) (string, error) {
	if c.createEnter != nil {
		c.createEnter <- struct{}{}
	}
	if c.createGate != nil {
		<-c.createGate
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.created = append(c.created, m)
	if c.createErr != nil {
		return "", c.createErr
	}
	return c.createID, nil
}

func (c *fakeISDSClient) GetDeliveryInfo(_ context.Context, _ string) (*isds.DeliveryInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deliveryHit++
	return c.delivery, c.deliveryErr
}

func (c *fakeISDSClient) SignedSentMessageDownload(_ context.Context, _ string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.zfoHit++
	return c.zfo, c.zfoErr
}

func (c *fakeISDSClient) GetOwnerInfo(_ context.Context) (*isds.OwnerInfo, error) {
	return c.owner, c.ownerErr
}

func (c *fakeISDSClient) GetUserInfo(_ context.Context) (*isds.UserInfo, error) {
	return c.user, nil
}

type filingHarness struct {
	svc      *FilingService
	repo     *repository.FilingSubmissionRepository
	doc      *fakeFilingDoc
	epo      *fakeFilingEPO
	xsd      *fakeFilingXSD
	client   *fakeISDSClient
	settings *fakeFilingSettings
	clock    *time.Time
	logs     *bytes.Buffer
	factory  []isds.Environment
}

func newFilingHarness(t *testing.T) *filingHarness {
	t.Helper()
	keyring.MockInit()
	t.Cleanup(keyring.MockInit)
	if err := secrets.NewISDSStore().Save(secrets.ISDSEnvironmentTest,
		secrets.ISDSCredentials{Username: filingTestUsername, Password: filingTestPassword}); err != nil {
		t.Fatalf("saving the fake login: %v", err)
	}
	if err := secrets.NewISDSStore().Save(secrets.ISDSEnvironmentProduction,
		secrets.ISDSCredentials{Username: filingTestUsername, Password: filingTestPassword}); err != nil {
		t.Fatalf("saving the fake production login: %v", err)
	}

	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	prod, err := isds.NewRecipients(isds.RecipientEnvProduction, isds.TestRecipientBoxes{})
	if err != nil {
		t.Fatal(err)
	}
	test, err := isds.NewRecipients(isds.RecipientEnvTest, isds.TestRecipientBoxes{
		CSSZ:       "tst1234",
		TaxOffices: map[string]string{"451": "tst4511"},
	})
	if err != nil {
		t.Fatal(err)
	}

	clock := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	h := &filingHarness{
		repo:     repository.NewFilingSubmissionRepository(testutil.NewTestDB(t)),
		doc:      &fakeFilingDoc{xml: []byte(filingTestXML)},
		epo:      &fakeFilingEPO{},
		xsd:      &fakeFilingXSD{},
		client:   &fakeISDSClient{createID: "1234567890"},
		settings: &fakeFilingSettings{values: map[string]string{SettingUFOCode: "451"}},
		clock:    &clock,
		logs:     &logs,
	}
	svc, err := NewFilingService(FilingServiceConfig{
		Documents: map[domain.FilingKind]FilingDocument{
			domain.FilingKindVATReturn:       h.doc,
			domain.FilingKindSocialInsurance: h.doc,
		},
		Repo: h.repo,
		Recipients: map[isds.RecipientEnvironment]*isds.Recipients{
			isds.RecipientEnvProduction: prod,
			isds.RecipientEnvTest:       test,
		},
		Settings:    h.settings,
		EPO:         h.epo,
		XSD:         h.xsd,
		Credentials: secrets.NewISDSStore(),
		NewClient: func(env isds.Environment, creds isds.Credentials) (FilingISDSClient, error) {
			h.factory = append(h.factory, env)
			if creds.Username != filingTestUsername || creds.Password != filingTestPassword {
				return nil, errors.New("fake factory got another login")
			}
			return h.client, nil
		},
		Now: func() time.Time { return clock },
	})
	if err != nil {
		t.Fatalf("NewFilingService: %v", err)
	}
	h.svc = svc
	return h
}

func (h *filingHarness) advance(d time.Duration) { *h.clock = h.clock.Add(d) }

func (h *filingHarness) previewSend(t *testing.T, env string) (*FilingPreview, *domain.FilingSubmission) {
	t.Helper()
	p, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, env)
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	sub, err := h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
		FilingSendRequest{Environment: env, ConfirmToken: p.ConfirmToken})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	return p, sub
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestFilingService_PreviewTaxOffice(t *testing.T) {
	h := newFilingHarness(t)
	p, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if p.RecipientBoxID != "7nyn2d9" || !strings.Contains(p.RecipientName, "Prahu") {
		t.Errorf("recipient = %s %q, want the Praha FÚ box for c_ufo 451", p.RecipientBoxID, p.RecipientName)
	}
	if p.Environment != "production" || p.FileName != "DPHDP3.xml" || p.SizeBytes != len(filingTestXML) || p.SHA256 != sha256Hex(filingTestXML) {
		t.Errorf("preview = %+v", p)
	}
	if p.ConfirmToken == "" || !p.ExpiresAt.Equal(h.clock.Add(DefaultFilingConfirmTTL)) {
		t.Errorf("token %q expires %v", p.ConfirmToken, p.ExpiresAt)
	}
	if string(h.epo.got) != filingTestXML || h.epo.calls != 1 || h.xsd.calls != 0 {
		t.Errorf("the check did not run on the regenerated XML (epo %d, xsd %d)", h.epo.calls, h.xsd.calls)
	}
	if h.doc.regenCalls != 1 {
		t.Errorf("regenerated %d times, want 1", h.doc.regenCalls)
	}
	if len(h.factory) != 0 || len(h.client.created) != 0 {
		t.Error("a preview touched ISDS")
	}
	if subs, _ := h.repo.ListByDocument(context.Background(), domain.FilingKindVATReturn, 7); len(subs) != 0 {
		t.Errorf("a preview stored %d submissions", len(subs))
	}
}

func TestFilingService_PreviewSocialInsuranceUsesCSSZAndXSD(t *testing.T) {
	h := newFilingHarness(t)
	p, err := h.svc.Preview(context.Background(), domain.FilingKindSocialInsurance, 3, "production")
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if p.RecipientBoxID != isds.CSSZEpodaniBoxID || p.FileName != "OSVC.xml" {
		t.Errorf("preview = %+v", p)
	}
	if h.xsd.calls != 1 || h.xsd.schema != xsdcheck.SchemaOSVC25 || h.epo.calls != 0 {
		t.Errorf("checks: xsd %d (%v), epo %d", h.xsd.calls, h.xsd.schema, h.epo.calls)
	}
}

func TestFilingService_PreviewRefusesWhenTheCheckCannotRun(t *testing.T) {
	tests := map[string]struct {
		kind  domain.FilingKind
		setup func(*filingHarness)
		is    error
	}{
		"xmllint missing": {domain.FilingKindSocialInsurance, func(h *filingHarness) { h.xsd.err = xsdcheck.ErrXmllintMissing }, xsdcheck.ErrXmllintMissing},
		"xsd invalid": {domain.FilingKindSocialInsurance, func(h *filingHarness) {
			h.xsd.err = &xsdcheck.InvalidError{Messages: "fake schema message"}
		}, xsdcheck.ErrInvalid},
		"EPO timeout":     {domain.FilingKindVATReturn, func(h *filingHarness) { h.epo.err = epo.ErrEPOTimeout }, epo.ErrEPOTimeout},
		"EPO unavailable": {domain.FilingKindVATReturn, func(h *filingHarness) { h.epo.err = epo.ErrEPOUnavailable }, epo.ErrEPOUnavailable},
		"EPO blocking finding": {domain.FilingKindVATReturn, func(h *filingHarness) {
			h.epo.findings = []epo.Finding{{Type: epo.TypeCritical, Code: "FAKE", Text: "fake critical"}}
		}, nil},
		"XML not generated": {domain.FilingKindVATReturn, func(h *filingHarness) { h.doc.xml = nil }, domain.ErrXMLNotGenerated},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			h := newFilingHarness(t)
			tc.setup(h)
			p, err := h.svc.Preview(context.Background(), tc.kind, 7, "production")
			if err == nil || p != nil {
				t.Fatalf("Preview = %v, %v; want an error and no token", p, err)
			}
			if tc.is != nil && !errors.Is(err, tc.is) {
				t.Errorf("error = %v, want %v", err, tc.is)
			}
			if name == "EPO blocking finding" {
				var rejected *epo.RejectedError
				if !errors.As(err, &rejected) || len(rejected.Findings) != 1 {
					t.Errorf("error = %v, want *epo.RejectedError with the finding", err)
				}
			}
		})
	}
}

func TestFilingService_PreviewRecipientSettings(t *testing.T) {
	t.Run("missing c_ufo", func(t *testing.T) {
		h := newFilingHarness(t)
		delete(h.settings.values, SettingUFOCode)
		_, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
		if !errors.Is(err, domain.ErrMissingSetting) {
			t.Errorf("error = %v, want ErrMissingSetting", err)
		}
	})
	t.Run("unknown c_ufo", func(t *testing.T) {
		h := newFilingHarness(t)
		h.settings.values[SettingUFOCode] = "999"
		_, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("error = %v, want ErrInvalidInput", err)
		}
	})
	t.Run("test environment uses the configured test box", func(t *testing.T) {
		h := newFilingHarness(t)
		p, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "test")
		if err != nil {
			t.Fatal(err)
		}
		if p.RecipientBoxID != "tst4511" {
			t.Errorf("test recipient = %s, want the configured test box", p.RecipientBoxID)
		}
	})
	t.Run("test environment without a test box", func(t *testing.T) {
		h := newFilingHarness(t)
		h.settings.values[SettingUFOCode] = "452"
		_, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "test")
		if !errors.Is(err, domain.ErrMissingSetting) {
			t.Errorf("error = %v, want ErrMissingSetting and no production fallback", err)
		}
	})
	t.Run("unknown environment", func(t *testing.T) {
		h := newFilingHarness(t)
		for _, env := range []string{"", "staging", "PRODUCTION"} {
			if _, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, env); !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("environment %q: error = %v, want ErrInvalidInput", env, err)
			}
		}
	})
	t.Run("unsupported kind and id", func(t *testing.T) {
		h := newFilingHarness(t)
		if _, err := h.svc.Preview(context.Background(), domain.FilingKindVIESSummary, 7, "test"); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("unwired kind: %v", err)
		}
		if _, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 0, "test"); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("zero id: %v", err)
		}
	})
}

func TestFilingService_SendHappyPath(t *testing.T) {
	h := newFilingHarness(t)
	p, sub := h.previewSend(t, "production")

	if len(h.client.created) != 1 {
		t.Fatalf("CreateMessage calls = %d, want 1", len(h.client.created))
	}
	msg := h.client.created[0]
	if msg.RecipientBoxID != "7nyn2d9" || msg.Annotation != p.Annotation || len(msg.Files) != 1 {
		t.Errorf("message = %+v", msg)
	}
	f := msg.Files[0]
	if f.Name != "DPHDP3.xml" || f.MIMEType != isds.MIMETypeXML || f.MetaType != isds.FileMain || string(f.Content) != filingTestXML {
		t.Errorf("file = %+v", f)
	}
	if err := msg.Validate(); err != nil {
		t.Errorf("the message violates an ISDS limit: %v", err)
	}
	if len(h.factory) != 1 || h.factory[0] != isds.EnvironmentProduction {
		t.Errorf("client environments = %v", h.factory)
	}

	if sub.Status != domain.SubmissionSent || sub.DmID != "1234567890" || sub.SentAt == nil || !sub.SentAt.Equal(*h.clock) {
		t.Errorf("submission = %+v", sub)
	}
	if sub.RecipientBoxID != "7nyn2d9" || sub.XMLSHA256 != p.SHA256 || sub.Environment != "production" || sub.DmAnnotation != p.Annotation {
		t.Errorf("stored columns = %+v", sub)
	}
}

func TestFilingService_TokenIsSingleUse(t *testing.T) {
	h := newFilingHarness(t)
	p, _ := h.previewSend(t, "production")

	_, err := h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
		FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken})
	if !errors.Is(err, domain.ErrConfirmTokenInvalid) {
		t.Errorf("reusing the token = %v, want ErrConfirmTokenInvalid", err)
	}
	if len(h.client.created) != 1 {
		t.Errorf("CreateMessage calls = %d, want 1", len(h.client.created))
	}
}

func TestFilingService_SameBytesAreNotSentTwiceToOneEnvironment(t *testing.T) {
	h := newFilingHarness(t)
	h.previewSend(t, "production")

	p2, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
		FilingSendRequest{Environment: "production", ConfirmToken: p2.ConfirmToken})
	if !errors.Is(err, domain.ErrFilingAlreadySent) {
		t.Fatalf("second send = %v, want ErrFilingAlreadySent", err)
	}
	if len(h.client.created) != 1 {
		t.Errorf("CreateMessage calls = %d, want 1", len(h.client.created))
	}

	// The test environment is independent of the production one.
	h.previewSend(t, "test")
	if len(h.client.created) != 2 {
		t.Errorf("a send to the test environment was blocked by the production send")
	}
}

func TestFilingService_TokenExpires(t *testing.T) {
	h := newFilingHarness(t)
	p, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatal(err)
	}
	h.advance(DefaultFilingConfirmTTL + time.Second)
	_, err = h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
		FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken})
	if !errors.Is(err, domain.ErrConfirmTokenInvalid) {
		t.Errorf("expired token = %v, want ErrConfirmTokenInvalid", err)
	}
	if len(h.client.created) != 0 {
		t.Error("an expired token sent a message")
	}
}

func TestFilingService_TokenStillValidJustBeforeExpiry(t *testing.T) {
	h := newFilingHarness(t)
	p, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatal(err)
	}
	h.advance(DefaultFilingConfirmTTL - time.Second)
	if _, err := h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
		FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken}); err != nil {
		t.Errorf("token rejected before its expiry: %v", err)
	}
}

func TestFilingService_ChangedXMLInvalidatesToken(t *testing.T) {
	h := newFilingHarness(t)
	p, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatal(err)
	}
	h.doc.setXML(filingTestXML + "<!-- changed -->")

	_, err = h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
		FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken})
	if !errors.Is(err, domain.ErrConfirmTokenInvalid) {
		t.Fatalf("send after the XML changed = %v, want ErrConfirmTokenInvalid", err)
	}
	if len(h.client.created) != 0 {
		t.Error("changed bytes were sent under an old confirmation")
	}
	// The mismatch spends the token.
	h.doc.setXML(filingTestXML)
	_, err = h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
		FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken})
	if !errors.Is(err, domain.ErrConfirmTokenInvalid) {
		t.Errorf("a mismatching request left the token usable: %v", err)
	}
}

func TestFilingService_TokenIsBoundToEnvironmentRecipientAndDocument(t *testing.T) {
	t.Run("environment", func(t *testing.T) {
		h := newFilingHarness(t)
		p, _ := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "test")
		_, err := h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
			FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken})
		if !errors.Is(err, domain.ErrConfirmTokenInvalid) || len(h.client.created) != 0 {
			t.Errorf("send to another environment = %v (messages %d)", err, len(h.client.created))
		}
	})
	t.Run("recipient", func(t *testing.T) {
		h := newFilingHarness(t)
		p, _ := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
		h.settings.values[SettingUFOCode] = "452" // the tax office changed after the preview
		_, err := h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
			FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken})
		if !errors.Is(err, domain.ErrConfirmTokenInvalid) || len(h.client.created) != 0 {
			t.Errorf("send to another recipient = %v (messages %d)", err, len(h.client.created))
		}
	})
	t.Run("document", func(t *testing.T) {
		h := newFilingHarness(t)
		p, _ := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
		_, err := h.svc.Send(context.Background(), domain.FilingKindVATReturn, 8,
			FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken})
		if !errors.Is(err, domain.ErrConfirmTokenInvalid) || len(h.client.created) != 0 {
			t.Errorf("send of another document = %v (messages %d)", err, len(h.client.created))
		}
	})
	t.Run("kind", func(t *testing.T) {
		h := newFilingHarness(t)
		p, _ := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
		_, err := h.svc.Send(context.Background(), domain.FilingKindSocialInsurance, 7,
			FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken})
		if !errors.Is(err, domain.ErrConfirmTokenInvalid) || len(h.client.created) != 0 {
			t.Errorf("send of another kind = %v (messages %d)", err, len(h.client.created))
		}
	})
	t.Run("garbage token", func(t *testing.T) {
		h := newFilingHarness(t)
		for _, tok := range []string{"", "nope", strings.Repeat("A", 43)} {
			_, err := h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
				FilingSendRequest{Environment: "production", ConfirmToken: tok})
			if !errors.Is(err, domain.ErrConfirmTokenInvalid) {
				t.Errorf("token %q: %v", tok, err)
			}
		}
		if len(h.client.created) != 0 {
			t.Error("a garbage token sent a message")
		}
	})
}

func TestFilingService_WarningsNeedAcceptance(t *testing.T) {
	h := newFilingHarness(t)
	h.epo.findings = []epo.Finding{
		{Type: epo.TypeInfo, Code: "TEST", Text: "info only"},
		{Type: epo.TypePassable, Code: "PODANO_PO_LHUTE", Field: "d_poddp", Text: "fake warning"},
	}
	p, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatal(err)
	}
	if !p.RequiresAcceptWarnings || len(p.Findings) != 1 || p.Findings[0].Code != "PODANO_PO_LHUTE" {
		t.Fatalf("preview findings = %+v (requires %v)", p.Findings, p.RequiresAcceptWarnings)
	}

	req := FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken}
	if _, err := h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7, req); !errors.Is(err, domain.ErrWarningsNotAccepted) {
		t.Fatalf("send without acceptance = %v, want ErrWarningsNotAccepted", err)
	}
	if len(h.client.created) != 0 {
		t.Fatal("a send with unaccepted warnings went out")
	}
	// The token survives the refusal, so the user can accept and repeat.
	req.AcceptWarnings = true
	if _, err := h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7, req); err != nil {
		t.Errorf("send with acceptance: %v", err)
	}
}

func TestFilingService_InFlightLock(t *testing.T) {
	h := newFilingHarness(t)
	h.client.createGate = make(chan struct{})
	h.client.createEnter = make(chan struct{}, 1)

	p, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
			FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken})
		done <- err
	}()
	<-h.client.createEnter // the first send is inside CreateMessage

	p2, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
		FilingSendRequest{Environment: "production", ConfirmToken: p2.ConfirmToken})
	if !errors.Is(err, domain.ErrFilingSendInProgress) {
		t.Errorf("concurrent send = %v, want ErrFilingSendInProgress", err)
	}

	close(h.client.createGate)
	if err := <-done; err != nil {
		t.Errorf("first send: %v", err)
	}
	if len(h.client.created) != 1 {
		t.Errorf("CreateMessage calls = %d, want 1", len(h.client.created))
	}
}

func TestFilingService_SendFailureOutcomes(t *testing.T) {
	tests := map[string]struct {
		err        error
		wantStatus domain.FilingSubmissionStatus
		canResend  bool
	}{
		"ISDS refused":       {&isds.StatusError{Code: "1234", Message: "fake refusal"}, domain.SubmissionFailed, true},
		"bad credentials":    {fmt.Errorf("CreateMessage: %w", isds.ErrUnauthorized), domain.SubmissionFailed, true},
		"SOAP fault":         {&isds.FaultError{Code: "S:Server", Message: "fake"}, domain.SubmissionFailed, true},
		"invalid input":      {fmt.Errorf("%w: too big", isds.ErrInvalidInput), domain.SubmissionFailed, true},
		"timeout":            {fmt.Errorf("CreateMessage: %w: %w", isds.ErrISDSTimeout, isds.ErrISDSUnavailable), domain.SubmissionUnknown, false},
		"unavailable":        {fmt.Errorf("CreateMessage: %w", isds.ErrISDSUnavailable), domain.SubmissionUnknown, false},
		"unexpected content": {fmt.Errorf("CreateMessage: %w", isds.ErrUnexpectedResponse), domain.SubmissionUnknown, false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			h := newFilingHarness(t)
			h.client.createErr = tc.err
			p, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
			if err != nil {
				t.Fatal(err)
			}
			_, err = h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
				FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken})
			if !errors.Is(err, tc.err) {
				t.Fatalf("Send error = %v, want it to wrap %v", err, tc.err)
			}
			subs, _ := h.repo.ListByDocument(context.Background(), domain.FilingKindVATReturn, 7)
			if len(subs) != 1 || subs[0].Status != tc.wantStatus || subs[0].ErrorText == "" {
				t.Fatalf("stored submissions = %+v, want one %s with a reason", subs, tc.wantStatus)
			}

			h.client.createErr = nil
			p2, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
			if err != nil {
				t.Fatal(err)
			}
			_, err = h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
				FilingSendRequest{Environment: "production", ConfirmToken: p2.ConfirmToken})
			if tc.canResend && err != nil {
				t.Errorf("resend after a definite failure: %v", err)
			}
			if !tc.canResend && !errors.Is(err, domain.ErrFilingAlreadySent) {
				t.Errorf("resend after an unknown outcome = %v, want ErrFilingAlreadySent", err)
			}
		})
	}
}

func TestFilingService_MissingLoginKeepsTheToken(t *testing.T) {
	h := newFilingHarness(t)
	p, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatal(err)
	}
	if err := secrets.NewISDSStore().Delete(secrets.ISDSEnvironmentProduction); err != nil {
		t.Fatal(err)
	}
	req := FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken}
	if _, err := h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7, req); !errors.Is(err, domain.ErrISDSCredentialsMissing) {
		t.Fatalf("send without a login = %v, want ErrISDSCredentialsMissing", err)
	}
	if subs, _ := h.repo.ListByDocument(context.Background(), domain.FilingKindVATReturn, 7); len(subs) != 0 {
		t.Errorf("a refused send stored %d submissions", len(subs))
	}

	if err := secrets.NewISDSStore().Save(secrets.ISDSEnvironmentProduction,
		secrets.ISDSCredentials{Username: filingTestUsername, Password: filingTestPassword}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7, req); err != nil {
		t.Errorf("send after saving the login with the same token: %v", err)
	}
}

func TestFilingService_KeyringErrorsPassThrough(t *testing.T) {
	h := newFilingHarness(t)
	p, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatal(err)
	}
	keyring.MockInitWithError(errors.New("no secret service"))
	t.Cleanup(keyring.MockInit)
	_, err = h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
		FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken})
	if !errors.Is(err, domain.ErrKeyringUnavailable) {
		t.Errorf("send with a broken keyring = %v, want ErrKeyringUnavailable", err)
	}
}

func deliveredInfo(status isds.MessageStatus, at *time.Time) *isds.DeliveryInfo {
	return &isds.DeliveryInfo{MessageID: "1234567890", Status: status, DeliveryTime: at}
}

func TestFilingService_RefreshDeliveredMarksDocumentFiledAtDeliveryTime(t *testing.T) {
	h := newFilingHarness(t)
	_, sub := h.previewSend(t, "production")

	deliveredAt := time.Date(2026, 11, 24, 12, 7, 0, 0, time.FixedZone("", 3600))
	h.client.delivery = deliveredInfo(isds.StatusDelivered, &deliveredAt)
	h.client.zfo = []byte{0x01, 0x02, 'Z', 'F', 'O'}
	h.advance(time.Hour) // the click happens well after the delivery

	res, err := h.svc.Refresh(context.Background(), sub.ID)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	got := res.Submission
	if got.Status != domain.SubmissionDelivered || got.DmMessageStatus == nil || *got.DmMessageStatus != 4 {
		t.Errorf("submission = %+v", got)
	}
	if got.DeliveredAt == nil || !got.DeliveredAt.Equal(deliveredAt) || !got.HasReceipt {
		t.Errorf("delivered_at/receipt = %v / %v", got.DeliveredAt, got.HasReceipt)
	}
	if len(h.doc.markCalls) != 1 || !h.doc.markCalls[0].Equal(deliveredAt) || !res.DocumentFiled {
		t.Errorf("document marked filed at %v (filed %v), want the delivery time %v", h.doc.markCalls, res.DocumentFiled, deliveredAt)
	}
	zfo, name, err := h.svc.Receipt(context.Background(), sub.ID)
	if err != nil || !bytes.Equal(zfo, h.client.zfo) || name != "DZ_1234567890.zfo" {
		t.Errorf("Receipt = %v, %q, %v", zfo, name, err)
	}

	// A second refresh needs no ISDS call.
	hits := h.client.deliveryHit
	if _, err := h.svc.Refresh(context.Background(), sub.ID); err != nil {
		t.Fatal(err)
	}
	if h.client.deliveryHit != hits || h.client.zfoHit != 1 {
		t.Errorf("a refresh of a finished submission called ISDS again (delivery %d, zfo %d)", h.client.deliveryHit, h.client.zfoHit)
	}
}

func TestFilingService_RefreshStatuses(t *testing.T) {
	tests := map[string]struct {
		status      isds.MessageStatus
		wantStatus  domain.FilingSubmissionStatus
		wantFiled   bool
		wantReceipt bool
	}{
		"submitted":         {isds.StatusSubmitted, domain.SubmissionSent, false, false},
		"stamped":           {isds.StatusStamped, domain.SubmissionSent, false, false},
		"malicious":         {isds.StatusRejectedMalicious, domain.SubmissionSent, false, false},
		"delivered":         {isds.StatusDelivered, domain.SubmissionDelivered, true, true},
		"delivered fiction": {isds.StatusDeliveredByFiction, domain.SubmissionDelivered, true, true},
		"delivered login":   {isds.StatusDeliveredByLogin, domain.SubmissionDelivered, true, true},
		"read":              {isds.StatusRead, domain.SubmissionDelivered, true, true},
		"undeliverable":     {isds.StatusUndeliverable, domain.SubmissionUndeliverable, false, false},
		"unknown value":     {isds.MessageStatus(10), domain.SubmissionSent, false, false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			h := newFilingHarness(t)
			_, sub := h.previewSend(t, "production")
			at := time.Date(2026, 11, 24, 12, 7, 0, 0, time.UTC)
			h.client.delivery = deliveredInfo(tc.status, &at)
			h.client.zfo = []byte("zfo")

			res, err := h.svc.Refresh(context.Background(), sub.ID)
			if err != nil {
				t.Fatalf("Refresh: %v", err)
			}
			if res.Submission.Status != tc.wantStatus || *res.Submission.DmMessageStatus != int(tc.status) {
				t.Errorf("status = %s (%v), want %s with the raw value %d", res.Submission.Status, res.Submission.DmMessageStatus, tc.wantStatus, tc.status)
			}
			if (len(h.doc.markCalls) == 1) != tc.wantFiled {
				t.Errorf("document marked filed %d times, want filed=%v", len(h.doc.markCalls), tc.wantFiled)
			}
			if res.Submission.HasReceipt != tc.wantReceipt {
				t.Errorf("HasReceipt = %v, want %v", res.Submission.HasReceipt, tc.wantReceipt)
			}
			if !tc.wantFiled && res.Submission.DeliveredAt != nil {
				t.Errorf("delivered_at set for %s", tc.wantStatus)
			}
		})
	}
}

func TestFilingService_RefreshRetriesMarkingFiledAndToleratesAFiledDocument(t *testing.T) {
	h := newFilingHarness(t)
	_, sub := h.previewSend(t, "production")
	at := time.Date(2026, 11, 24, 12, 7, 0, 0, time.UTC)
	h.client.delivery = deliveredInfo(isds.StatusDelivered, &at)
	h.client.zfo = []byte("zfo")
	h.doc.markErr = errors.New("database is locked")

	res, err := h.svc.Refresh(context.Background(), sub.ID)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if res.DocumentFiled || res.Submission.Status != domain.SubmissionDelivered {
		t.Fatalf("result = %+v; the delivery must be stored even when marking filed fails", res)
	}

	h.doc.markErr = nil
	res, err = h.svc.Refresh(context.Background(), sub.ID)
	if err != nil || !res.DocumentFiled {
		t.Fatalf("retry = %+v, %v", res, err)
	}
	if got := h.doc.markCalls[len(h.doc.markCalls)-1]; !got.Equal(at) {
		t.Errorf("retry marked filed at %v, want the delivery time %v", got, at)
	}

	// A document the user already marked filed by hand is not an error.
	res, err = h.svc.Refresh(context.Background(), sub.ID)
	if err != nil || !res.DocumentFiled {
		t.Errorf("refresh with an already filed document = %+v, %v", res, err)
	}
}

func TestFilingService_RefreshReceiptFailureIsRetriedNextTime(t *testing.T) {
	h := newFilingHarness(t)
	_, sub := h.previewSend(t, "production")
	at := time.Date(2026, 11, 24, 12, 7, 0, 0, time.UTC)
	h.client.delivery = deliveredInfo(isds.StatusDelivered, &at)
	h.client.zfoErr = fmt.Errorf("SignedSentMessageDownload: %w", isds.ErrISDSUnavailable)

	res, err := h.svc.Refresh(context.Background(), sub.ID)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if res.Submission.Status != domain.SubmissionDelivered || res.Submission.HasReceipt {
		t.Errorf("submission = %+v", res.Submission)
	}
	if _, _, err := h.svc.Receipt(context.Background(), sub.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Receipt without a stored ZFO = %v, want ErrNotFound", err)
	}

	h.client.zfoErr = nil
	h.client.zfo = []byte("zfo")
	res, err = h.svc.Refresh(context.Background(), sub.ID)
	if err != nil || !res.Submission.HasReceipt {
		t.Errorf("second refresh = %+v, %v", res, err)
	}
	if got := h.doc.markCalls[0]; !got.Equal(at) {
		t.Errorf("filed at %v, want the delivery time %v", got, at)
	}
}

func TestFilingService_RefreshErrors(t *testing.T) {
	t.Run("ISDS failure leaves the stored state", func(t *testing.T) {
		h := newFilingHarness(t)
		_, sub := h.previewSend(t, "production")
		h.client.deliveryErr = fmt.Errorf("GetDeliveryInfo: %w", isds.ErrISDSTimeout)
		if _, err := h.svc.Refresh(context.Background(), sub.ID); !errors.Is(err, isds.ErrISDSTimeout) {
			t.Errorf("Refresh = %v", err)
		}
		got, _ := h.repo.GetByID(context.Background(), sub.ID)
		if got.Status != domain.SubmissionSent {
			t.Errorf("status = %s after a failed refresh", got.Status)
		}
	})
	t.Run("a failed or unknown submission has nothing to refresh", func(t *testing.T) {
		h := newFilingHarness(t)
		h.client.createErr = fmt.Errorf("CreateMessage: %w", isds.ErrISDSUnavailable)
		p, _ := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
		_, _ = h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
			FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken})
		subs, _ := h.repo.ListByDocument(context.Background(), domain.FilingKindVATReturn, 7)
		if _, err := h.svc.Refresh(context.Background(), subs[0].ID); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("Refresh of an unknown submission = %v, want ErrInvalidInput", err)
		}
		if h.client.deliveryHit != 0 {
			t.Error("ISDS was called for a submission without a message id")
		}
	})
	t.Run("missing submission", func(t *testing.T) {
		h := newFilingHarness(t)
		if _, err := h.svc.Refresh(context.Background(), 999); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("Refresh = %v, want ErrNotFound", err)
		}
		if _, _, err := h.svc.Receipt(context.Background(), 999); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("Receipt = %v, want ErrNotFound", err)
		}
	})
}

func TestFilingService_List(t *testing.T) {
	h := newFilingHarness(t)
	h.previewSend(t, "test")
	h.previewSend(t, "production")
	subs, err := h.svc.List(context.Background(), domain.FilingKindVATReturn, 7)
	if err != nil || len(subs) != 2 || subs[0].Environment != "production" {
		t.Errorf("List = %+v, %v", subs, err)
	}
	if _, err := h.svc.List(context.Background(), domain.FilingKindVIESSummary, 7); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("List of an unwired kind = %v", err)
	}
}

func TestFilingService_TestLogin(t *testing.T) {
	h := newFilingHarness(t)
	h.client.owner = &isds.OwnerInfo{BoxID: "abc1234", BoxType: "FO", GivenNames: "Jan", LastName: "Testovaci"}
	h.client.user = &isds.UserInfo{UserType: "PRIMARY_USER", Privileges: isds.PrivilegeCreateAndSend | isds.PrivilegeViewInfo}

	res, err := h.svc.TestLogin(context.Background(), "test")
	if err != nil {
		t.Fatalf("TestLogin: %v", err)
	}
	if res.BoxID != "abc1234" || res.BoxType != "FO" || res.OwnerName != "Jan Testovaci" || !res.CanCreateAndSend || res.CanReadMessages {
		t.Errorf("result = %+v", res)
	}
	if len(h.factory) != 1 || h.factory[0] != isds.EnvironmentTest {
		t.Errorf("client environments = %v", h.factory)
	}

	h.client.user = &isds.UserInfo{Privileges: isds.PrivilegeCreateAndSend | isds.PrivilegeReadAll}
	h.client.owner = &isds.OwnerInfo{BoxID: "abc1234", BoxType: "PFO", FirmName: "Fake Firm"}
	res, err = h.svc.TestLogin(context.Background(), "test")
	if err != nil || !res.CanReadMessages || res.OwnerName != "Fake Firm" {
		t.Errorf("reading login = %+v, %v", res, err)
	}

	h.client.ownerErr = fmt.Errorf("GetOwnerInfoFromLogin2: %w", isds.ErrUnauthorized)
	if _, err := h.svc.TestLogin(context.Background(), "test"); !errors.Is(err, isds.ErrUnauthorized) {
		t.Errorf("rejected login = %v", err)
	}
	if err := secrets.NewISDSStore().Delete(secrets.ISDSEnvironmentTest); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.TestLogin(context.Background(), "test"); !errors.Is(err, domain.ErrISDSCredentialsMissing) {
		t.Errorf("login test without a login = %v", err)
	}
	if _, err := h.svc.TestLogin(context.Background(), "bogus"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("login test of an unknown environment = %v", err)
	}
}

func TestFilingService_LogsAndErrorsNeverCarrySecretsOrXML(t *testing.T) {
	h := newFilingHarness(t)
	h.epo.findings = []epo.Finding{{Type: epo.TypePassable, Code: "W", Text: "fake"}}
	p, err := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "production")
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
		FilingSendRequest{Environment: "production", ConfirmToken: p.ConfirmToken, AcceptWarnings: true})
	if err != nil {
		t.Fatal(err)
	}
	h.client.createErr = fmt.Errorf("CreateMessage: %w", isds.ErrISDSUnavailable)
	h.doc.setXML(filingTestXML + "<x/>")
	p2, _ := h.svc.Preview(context.Background(), domain.FilingKindVATReturn, 7, "test")
	_, sendErr := h.svc.Send(context.Background(), domain.FilingKindVATReturn, 7,
		FilingSendRequest{Environment: "test", ConfirmToken: p2.ConfirmToken, AcceptWarnings: true})
	if sendErr == nil {
		t.Fatal("expected the failing send to error")
	}

	all := h.logs.String() + sendErr.Error()
	subs, _ := h.repo.ListByDocument(context.Background(), domain.FilingKindVATReturn, 7)
	for _, s := range subs {
		all += s.ErrorText
	}
	for _, secret := range []string{filingTestUsername, filingTestPassword, "fake-body-1", "<Pisemnost>", p.ConfirmToken, p2.ConfirmToken} {
		if strings.Contains(all, secret) {
			t.Errorf("logs or errors contain %q", secret)
		}
	}
	for _, want := range []string{"7nyn2d9", "1234567890", p.SHA256, "ISDS filing sent"} {
		if !strings.Contains(h.logs.String(), want) {
			t.Errorf("the audit log line lacks %q", want)
		}
	}
}

func TestNewFilingService_RejectsIncompleteConfig(t *testing.T) {
	if _, err := NewFilingService(FilingServiceConfig{}); err == nil {
		t.Error("an empty config was accepted")
	}
	h := newFilingHarness(t)
	_ = h
	cfg := FilingServiceConfig{
		Documents: map[domain.FilingKind]FilingDocument{"health-insurance": &fakeFilingDoc{}},
	}
	if _, err := NewFilingService(cfg); err == nil {
		t.Error("an unsupported kind was accepted")
	}
}
