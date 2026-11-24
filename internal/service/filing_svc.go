package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/epo"
	"github.com/zajca/zfaktury/internal/isds"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/secrets"
	"github.com/zajca/zfaktury/internal/xsdcheck"
)

// maxStoredErrorRunes bounds the failure reason kept with a submission.
const maxStoredErrorRunes = 500

// FilingDocument is one kind of generated filing the service can send.
type FilingDocument interface {
	// RegenerateXML regenerates the filing XML from the current data, stores
	// it with the document and returns it. An empty result is an error.
	RegenerateXML(ctx context.Context, id int64) ([]byte, error)
	// MarkFiledAt marks the document as filed at the given time. It returns
	// domain.ErrFilingAlreadyFiled when the document is already filed.
	MarkFiledAt(ctx context.Context, id int64, at time.Time) error
}

// FilingEPOChecker is the part of the EPO client the pre-send check uses.
type FilingEPOChecker interface {
	Check(ctx context.Context, xmlData []byte) ([]epo.Finding, error)
}

// FilingXSDChecker validates XML against an embedded schema.
type FilingXSDChecker interface {
	Check(ctx context.Context, schema xsdcheck.Schema, xml []byte) error
}

// FilingSettings reads single settings (the tax office code).
type FilingSettings interface {
	Get(ctx context.Context, key string) (string, error)
}

// FilingCredentialStore loads the data-box login from the OS keyring.
type FilingCredentialStore interface {
	Load(env secrets.ISDSEnvironment) (secrets.ISDSCredentials, error)
}

// FilingISDSClient is the part of the ISDS client the service uses.
type FilingISDSClient interface {
	CreateMessage(ctx context.Context, m isds.MessageInput) (string, error)
	GetDeliveryInfo(ctx context.Context, dmID string) (*isds.DeliveryInfo, error)
	SignedSentMessageDownload(ctx context.Context, dmID string) ([]byte, error)
	GetOwnerInfo(ctx context.Context) (*isds.OwnerInfo, error)
	GetUserInfo(ctx context.Context) (*isds.UserInfo, error)
}

// FilingISDSClientFactory builds an ISDS client for one login. The service
// builds a client per request, so a login is read from the keyring only when
// the user asks for an action.
type FilingISDSClientFactory func(env isds.Environment, creds isds.Credentials) (FilingISDSClient, error)

// FilingServiceConfig carries the dependencies of FilingService.
type FilingServiceConfig struct {
	Documents   map[domain.FilingKind]FilingDocument
	Repo        repository.FilingSubmissionRepo
	Recipients  map[isds.RecipientEnvironment]*isds.Recipients
	Settings    FilingSettings
	EPO         FilingEPOChecker
	XSD         FilingXSDChecker
	Credentials FilingCredentialStore
	NewClient   FilingISDSClientFactory
	Audit       *AuditService
	// ConfirmTTL is the lifetime of a confirmation token; zero keeps
	// DefaultFilingConfirmTTL.
	ConfirmTTL time.Duration
	// Now returns the current time; nil keeps time.Now.
	Now func() time.Time
}

// FilingService sends generated filings through a data box and tracks their
// delivery. Every ISDS call happens on an explicit request; nothing runs in
// the background.
type FilingService struct {
	docs        map[domain.FilingKind]FilingDocument
	repo        repository.FilingSubmissionRepo
	recipients  map[isds.RecipientEnvironment]*isds.Recipients
	settings    FilingSettings
	epo         FilingEPOChecker
	xsd         FilingXSDChecker
	credentials FilingCredentialStore
	newClient   FilingISDSClientFactory
	audit       *AuditService
	now         func() time.Time
	tokens      *confirmTokens

	mu       sync.Mutex
	inflight map[string]struct{}
}

// NewFilingService validates the dependencies and creates a FilingService.
func NewFilingService(cfg FilingServiceConfig) (*FilingService, error) {
	switch {
	case len(cfg.Documents) == 0:
		return nil, errors.New("filing service: no document kinds configured")
	case cfg.Repo == nil, cfg.Settings == nil, cfg.EPO == nil, cfg.XSD == nil, cfg.Credentials == nil, cfg.NewClient == nil:
		return nil, errors.New("filing service: a dependency is missing")
	case cfg.Recipients[isds.RecipientEnvTest] == nil || cfg.Recipients[isds.RecipientEnvProduction] == nil:
		return nil, errors.New("filing service: recipients are required for the test and production environments")
	case cfg.ConfirmTTL < 0:
		return nil, errors.New("filing service: confirmation TTL must not be negative")
	}
	for kind := range cfg.Documents {
		if _, ok := filingKinds[kind]; !ok {
			return nil, fmt.Errorf("filing service: unsupported kind %q", kind)
		}
	}
	if cfg.ConfirmTTL == 0 {
		cfg.ConfirmTTL = DefaultFilingConfirmTTL
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &FilingService{
		docs:        cfg.Documents,
		repo:        cfg.Repo,
		recipients:  cfg.Recipients,
		settings:    cfg.Settings,
		epo:         cfg.EPO,
		xsd:         cfg.XSD,
		credentials: cfg.Credentials,
		newClient:   cfg.NewClient,
		audit:       cfg.Audit,
		now:         cfg.Now,
		tokens:      newConfirmTokens(cfg.ConfirmTTL, cfg.Now),
		inflight:    make(map[string]struct{}),
	}, nil
}

// filingKindInfo describes how a kind is sent: where it goes, how it is
// checked before sending, and the subject and file name of the data message.
// The ISDS side prescribes neither a subject nor a file name for tax filings,
// so these are plain Czech descriptions with ASCII file names.
type filingKindInfo struct {
	recipient  isds.RecipientKind
	annotation string
	fileName   string
}

var filingKinds = map[domain.FilingKind]filingKindInfo{
	domain.FilingKindVATReturn: {
		recipient:  isds.RecipientKindTaxOffice,
		annotation: "Přiznání k dani z přidané hodnoty (DPHDP3)",
		fileName:   "DPHDP3.xml",
	},
	domain.FilingKindControlStatement: {
		recipient:  isds.RecipientKindTaxOffice,
		annotation: "Kontrolní hlášení k dani z přidané hodnoty (DPHKH1)",
		fileName:   "DPHKH1.xml",
	},
	domain.FilingKindVIESSummary: {
		recipient:  isds.RecipientKindTaxOffice,
		annotation: "Souhrnné hlášení k dani z přidané hodnoty (DPHSHV)",
		fileName:   "DPHSHV.xml",
	},
	domain.FilingKindIncomeTax: {
		recipient:  isds.RecipientKindTaxOffice,
		annotation: "Přiznání k dani z příjmů fyzických osob (DPFDP)",
		fileName:   "DPFDP.xml",
	},
	domain.FilingKindSocialInsurance: {
		recipient:  isds.RecipientKindCSSZ,
		annotation: "Přehled o příjmech a výdajích osoby samostatně výdělečně činné (e-Podání)",
		fileName:   "OSVC.xml",
	},
}

// ParseFilingKind converts an API path value to a FilingKind.
func ParseFilingKind(raw string) (domain.FilingKind, error) {
	kind := domain.FilingKind(raw)
	if _, ok := filingKinds[kind]; !ok {
		return "", &domain.UserMessageError{Message: "Neznámý typ podání."}
	}
	return kind, nil
}

// filingEnv is one environment named in the three vocabularies the ISDS
// packages use for it.
type filingEnv struct {
	name      string
	client    isds.Environment
	recipient isds.RecipientEnvironment
	keyring   secrets.ISDSEnvironment
}

// parseFilingEnv validates a request's environment. There is no default.
func parseFilingEnv(raw string) (filingEnv, error) {
	env, err := secrets.ParseISDSEnvironment(raw)
	if err != nil {
		return filingEnv{}, err
	}
	return filingEnv{
		name:      string(env),
		client:    isds.Environment(env),
		recipient: isds.RecipientEnvironment(env),
		keyring:   env,
	}, nil
}

// FilingPreview is what the user confirms before a send.
type FilingPreview struct {
	Kind           domain.FilingKind
	DocumentID     int64
	Environment    string
	RecipientBoxID string
	RecipientName  string
	FileName       string
	Annotation     string
	SizeBytes      int
	SHA256         string
	// Findings are the passable and non-critical findings of the check.
	Findings []epo.Finding
	// RequiresAcceptWarnings is true when Findings is not empty: the send then
	// needs an explicit acceptance.
	RequiresAcceptWarnings bool
	ConfirmToken           string
	ExpiresAt              time.Time
}

// FilingSendRequest is the confirmed send.
type FilingSendRequest struct {
	Environment    string
	ConfirmToken   string
	AcceptWarnings bool
}

// preparedFiling is the XML and the resolved recipient of one document, as of
// now.
type preparedFiling struct {
	xml       []byte
	sha256    string
	recipient isds.Recipient
	info      filingKindInfo
}

func (p *preparedFiling) binding(kind domain.FilingKind, id int64, env filingEnv) filingBinding {
	return filingBinding{
		kind:         kind,
		documentID:   id,
		environment:  env.name,
		recipientBox: p.recipient.BoxID,
		xmlSHA256:    p.sha256,
	}
}

func (s *FilingService) document(kind domain.FilingKind, id int64) (FilingDocument, error) {
	doc, ok := s.docs[kind]
	if !ok {
		return nil, fmt.Errorf("sending filing: %w: unsupported kind %q", domain.ErrInvalidInput, kind)
	}
	if id <= 0 {
		return nil, fmt.Errorf("sending filing: %w: document ID is required", domain.ErrInvalidInput)
	}
	return doc, nil
}

// prepare regenerates the XML and resolves the recipient. The recipient comes
// only from the embedded table and the settings, never from a request.
func (s *FilingService) prepare(ctx context.Context, kind domain.FilingKind, id int64, env filingEnv) (*preparedFiling, error) {
	doc, err := s.document(kind, id)
	if err != nil {
		return nil, err
	}
	xmlData, err := doc.RegenerateXML(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("regenerating XML of %s %d: %w", kind, id, err)
	}
	if len(xmlData) == 0 {
		return nil, fmt.Errorf("regenerating XML of %s %d: %w", kind, id, domain.ErrXMLNotGenerated)
	}

	info := filingKinds[kind]
	cUfo := 0
	if info.recipient == isds.RecipientKindTaxOffice {
		if cUfo, err = s.taxOfficeCode(ctx); err != nil {
			return nil, err
		}
	}
	recipient, err := s.recipients[env.recipient].Lookup(info.recipient, cUfo)
	if err != nil {
		return nil, fmt.Errorf("resolving the recipient of %s %d: %w", kind, id, err)
	}

	sum := sha256.Sum256(xmlData)
	return &preparedFiling{xml: xmlData, sha256: hex.EncodeToString(sum[:]), recipient: recipient, info: info}, nil
}

// taxOfficeCode reads the c_ufo setting that selects the regional tax office.
func (s *FilingService) taxOfficeCode(ctx context.Context) (int, error) {
	raw, err := s.settings.Get(ctx, SettingUFOCode)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("reading the tax office code: %w", err)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("resolving the tax office: %w: %w", domain.ErrMissingSetting,
			&domain.UserMessageError{Message: "V nastavení chybí kód finančního úřadu (c_ufo)."})
	}
	code, err := strconv.Atoi(raw)
	if err != nil {
		return 0, &domain.UserMessageError{Message: "Kód finančního úřadu (c_ufo) v nastavení není číslo."}
	}
	return code, nil
}

// check runs the pre-send check of a kind. It returns the warnings to show,
// or an error: *epo.RejectedError for blocking findings, a wrapped
// xsdcheck/epo error when the check could not run or the XML does not conform.
// A check that cannot run never lets a send through.
func (s *FilingService) check(ctx context.Context, kind domain.FilingKind, p *preparedFiling) ([]epo.Finding, error) {
	if p.info.recipient == isds.RecipientKindCSSZ {
		if err := s.xsd.Check(ctx, xsdcheck.SchemaOSVC25, p.xml); err != nil {
			return nil, fmt.Errorf("checking %s against the schema: %w", kind, err)
		}
		return nil, nil
	}

	findings, err := s.epo.Check(ctx, p.xml)
	if err != nil {
		return nil, fmt.Errorf("checking %s with EPO: %w", kind, err)
	}
	var warnings []epo.Finding
	for _, f := range findings {
		if f.IsBlocking() {
			return nil, fmt.Errorf("checking %s with EPO: %w", kind, &epo.RejectedError{Findings: findings})
		}
		if f.IsWarning() {
			warnings = append(warnings, f)
		}
	}
	return warnings, nil
}

// Preview regenerates the XML, runs the pre-send check, resolves the recipient
// and issues a one-time confirmation token bound to all of it. Nothing is sent
// and ISDS is not contacted.
func (s *FilingService) Preview(ctx context.Context, kind domain.FilingKind, id int64, rawEnv string) (*FilingPreview, error) {
	env, err := parseFilingEnv(rawEnv)
	if err != nil {
		return nil, fmt.Errorf("previewing filing: %w", err)
	}
	p, err := s.prepare(ctx, kind, id, env)
	if err != nil {
		return nil, err
	}
	warnings, err := s.check(ctx, kind, p)
	if err != nil {
		return nil, err
	}
	token, expiresAt, err := s.tokens.issue(p.binding(kind, id, env), len(warnings) > 0)
	if err != nil {
		return nil, err
	}

	slog.Info("ISDS filing previewed",
		"kind", string(kind), "id", id, "environment", env.name,
		"recipient_box", p.recipient.BoxID, "xml_sha256", p.sha256,
		"xml_bytes", len(p.xml), "warnings", len(warnings))
	return &FilingPreview{
		Kind:                   kind,
		DocumentID:             id,
		Environment:            env.name,
		RecipientBoxID:         p.recipient.BoxID,
		RecipientName:          p.recipient.Name,
		FileName:               p.info.fileName,
		Annotation:             p.info.annotation,
		SizeBytes:              len(p.xml),
		SHA256:                 p.sha256,
		Findings:               warnings,
		RequiresAcceptWarnings: len(warnings) > 0,
		ConfirmToken:           token,
		ExpiresAt:              expiresAt,
	}, nil
}

func (s *FilingService) acquire(kind domain.FilingKind, id int64) bool {
	key := string(kind) + ":" + strconv.FormatInt(id, 10)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, busy := s.inflight[key]; busy {
		return false
	}
	s.inflight[key] = struct{}{}
	return true
}

func (s *FilingService) release(kind domain.FilingKind, id int64) {
	key := string(kind) + ":" + strconv.FormatInt(id, 10)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.inflight, key)
}

// client loads the login of env from the keyring and builds an ISDS client.
func (s *FilingService) client(env filingEnv) (FilingISDSClient, error) {
	creds, err := s.credentials.Load(env.keyring)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("loading the data box login for %s: %w", env.name, domain.ErrISDSCredentialsMissing)
		}
		return nil, fmt.Errorf("loading the data box login for %s: %w", env.name, err)
	}
	client, err := s.newClient(env.client, isds.Credentials{Username: creds.Username, Password: creds.Password})
	if err != nil {
		return nil, fmt.Errorf("creating the ISDS client for %s: %w", env.name, err)
	}
	return client, nil
}

// Send sends a previewed filing. It regenerates the XML and refuses unless the
// bytes, the document, the environment and the recipient still match the
// confirmation token. The same bytes are never sent twice to one environment.
func (s *FilingService) Send(ctx context.Context, kind domain.FilingKind, id int64, req FilingSendRequest) (*domain.FilingSubmission, error) {
	env, err := parseFilingEnv(req.Environment)
	if err != nil {
		return nil, fmt.Errorf("sending filing: %w", err)
	}
	if _, err := s.document(kind, id); err != nil {
		return nil, err
	}
	if !s.acquire(kind, id) {
		return nil, fmt.Errorf("sending %s %d: %w", kind, id, domain.ErrFilingSendInProgress)
	}
	defer s.release(kind, id)

	p, err := s.prepare(ctx, kind, id, env)
	if err != nil {
		return nil, err
	}
	// The login is loaded before the token is spent, so a missing login does
	// not cost the user a new preview.
	client, err := s.client(env)
	if err != nil {
		return nil, err
	}
	if err := s.tokens.consume(req.ConfirmToken, p.binding(kind, id, env), req.AcceptWarnings); err != nil {
		return nil, fmt.Errorf("sending %s %d: %w", kind, id, err)
	}

	// Past this point the message may leave: a closed client connection must
	// neither abort it nor stop its record from being written.
	ctx = context.WithoutCancel(ctx)

	sub := &domain.FilingSubmission{
		DocumentKind:   kind,
		DocumentID:     id,
		Channel:        domain.FilingChannelISDS,
		Environment:    env.name,
		RecipientBoxID: p.recipient.BoxID,
		RecipientName:  p.recipient.Name,
		DmAnnotation:   p.info.annotation,
		FileName:       p.info.fileName,
		XMLSHA256:      p.sha256,
		Status:         domain.SubmissionSending,
	}
	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("recording the send of %s %d: %w", kind, id, err)
	}

	dmID, err := client.CreateMessage(ctx, isds.MessageInput{
		RecipientBoxID: p.recipient.BoxID,
		Annotation:     p.info.annotation,
		Files: []isds.File{{
			Name:     p.info.fileName,
			MIMEType: isds.MIMETypeXML,
			MetaType: isds.FileMain,
			Content:  p.xml,
		}},
	})
	if err != nil {
		status := classifySendError(err)
		reason := describeSendError(err)
		if markErr := s.repo.MarkFailed(ctx, sub.ID, status, reason); markErr != nil {
			slog.Error("storing the failed ISDS send failed", "submission_id", sub.ID, "error", markErr)
		}
		slog.Warn("ISDS filing send failed",
			"kind", string(kind), "id", id, "environment", env.name,
			"recipient_box", p.recipient.BoxID, "xml_sha256", p.sha256,
			"submission_id", sub.ID, "outcome", string(status), "error", reason)
		return nil, fmt.Errorf("sending %s %d: %w", kind, id, err)
	}

	sentAt := s.now()
	slog.Info("ISDS filing sent",
		"kind", string(kind), "id", id, "environment", env.name,
		"recipient_box", p.recipient.BoxID, "dm_id", dmID,
		"xml_sha256", p.sha256, "submission_id", sub.ID)
	if s.audit != nil {
		s.audit.Log(ctx, "filing_submission", sub.ID, "isds_send", nil, nil)
	}
	if err := s.repo.MarkSent(ctx, sub.ID, dmID, sentAt); err != nil {
		slog.Error("ISDS message accepted but its record could not be stored",
			"submission_id", sub.ID, "dm_id", dmID, "error", err)
		return nil, fmt.Errorf("recording the sent %s %d: %w: %w", kind, id, domain.ErrFilingRecordFailed, err)
	}
	stored, err := s.repo.GetByID(ctx, sub.ID)
	if err != nil {
		return nil, fmt.Errorf("fetching the sent submission: %w", err)
	}
	return stored, nil
}

// classifySendError decides whether a failed CreateMessage definitively did
// not send (failed, may be retried) or may have left (unknown, blocks a
// resend until the user checks the data box).
func classifySendError(err error) domain.FilingSubmissionStatus {
	switch {
	case errors.Is(err, isds.ErrInvalidInput),
		errors.Is(err, isds.ErrInvalidConfig),
		errors.Is(err, isds.ErrUnauthorized),
		errors.Is(err, isds.ErrISDSRejected),
		errors.Is(err, isds.ErrSOAPFault):
		return domain.SubmissionFailed
	default:
		return domain.SubmissionUnknown
	}
}

// describeSendError is the reason kept with a failed submission. The isds
// package never puts credentials into its errors.
func describeSendError(err error) string {
	text := err.Error()
	if r := []rune(text); len(r) > maxStoredErrorRunes {
		text = string(r[:maxStoredErrorRunes])
	}
	return text
}

// FilingRefreshResult is the outcome of a delivery refresh.
type FilingRefreshResult struct {
	Submission *domain.FilingSubmission
	// DocumentFiled is false when the message is delivered but the document
	// could not be marked filed; refreshing again retries it.
	DocumentFiled bool
}

// Refresh asks ISDS for the delivery state of a submission, stores the signed
// message once it is delivered and marks the document filed with the delivery
// time. It runs only when the user asks for it.
func (s *FilingService) Refresh(ctx context.Context, submissionID int64) (*FilingRefreshResult, error) {
	sub, err := s.repo.GetByID(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("fetching submission %d for refresh: %w", submissionID, err)
	}
	switch sub.Status {
	case domain.SubmissionSent, domain.SubmissionDelivered, domain.SubmissionUndeliverable:
	default:
		return nil, fmt.Errorf("refreshing submission %d: %w", submissionID,
			&domain.UserMessageError{Message: "U tohoto odeslání není číslo zprávy, stav doručení nelze zjistit."})
	}
	env, err := parseFilingEnv(sub.Environment)
	if err != nil {
		return nil, fmt.Errorf("refreshing submission %d: %w", submissionID, err)
	}

	// A delivered submission with its receipt needs no further ISDS call;
	// refreshing it only retries marking the document filed.
	if sub.Status == domain.SubmissionDelivered && sub.HasReceipt && sub.DeliveredAt != nil {
		return &FilingRefreshResult{Submission: sub, DocumentFiled: s.markFiled(ctx, sub)}, nil
	}

	client, err := s.client(env)
	if err != nil {
		return nil, err
	}
	info, err := client.GetDeliveryInfo(ctx, sub.DmID)
	if err != nil {
		return nil, fmt.Errorf("fetching the delivery state of submission %d: %w", submissionID, err)
	}

	status, deliveredAt := s.deliveryState(sub, info)
	if err := s.repo.UpdateDelivery(ctx, sub.ID, status, int(info.Status), deliveredAt); err != nil {
		return nil, fmt.Errorf("storing the delivery state of submission %d: %w", submissionID, err)
	}
	slog.Info("ISDS delivery state refreshed",
		"submission_id", sub.ID, "kind", string(sub.DocumentKind), "id", sub.DocumentID,
		"environment", sub.Environment, "dm_id", sub.DmID,
		"dm_message_status", int(info.Status), "status", string(status))

	if status == domain.SubmissionDelivered && !sub.HasReceipt {
		s.storeReceipt(ctx, client, sub)
	}

	updated, err := s.repo.GetByID(ctx, sub.ID)
	if err != nil {
		return nil, fmt.Errorf("fetching refreshed submission %d: %w", submissionID, err)
	}
	result := &FilingRefreshResult{Submission: updated, DocumentFiled: true}
	if updated.Status == domain.SubmissionDelivered {
		result.DocumentFiled = s.markFiled(ctx, updated)
	}
	return result, nil
}

// deliveryState maps ISDS delivery evidence to a submission status and the
// delivery time. A status the client does not know counts as not delivered.
func (s *FilingService) deliveryState(sub *domain.FilingSubmission, info *isds.DeliveryInfo) (domain.FilingSubmissionStatus, *time.Time) {
	switch {
	case info.Status.Delivered():
		switch {
		case sub.DeliveredAt != nil:
			return domain.SubmissionDelivered, sub.DeliveredAt
		case info.DeliveryTime != nil:
			return domain.SubmissionDelivered, info.DeliveryTime
		case info.AcceptanceTime != nil:
			return domain.SubmissionDelivered, info.AcceptanceTime
		default:
			now := s.now()
			return domain.SubmissionDelivered, &now
		}
	case info.Status == isds.StatusUndeliverable:
		return domain.SubmissionUndeliverable, nil
	default:
		return domain.SubmissionSent, nil
	}
}

// storeReceipt downloads and stores the signed message. A failure is logged
// and left for the next refresh: the delivery state is already stored.
func (s *FilingService) storeReceipt(ctx context.Context, client FilingISDSClient, sub *domain.FilingSubmission) {
	zfo, err := client.SignedSentMessageDownload(ctx, sub.DmID)
	if err != nil {
		slog.Warn("downloading the signed ISDS message failed", "submission_id", sub.ID, "error", describeSendError(err))
		return
	}
	if err := s.repo.SaveReceipt(ctx, sub.ID, zfo); err != nil {
		slog.Error("storing the signed ISDS message failed", "submission_id", sub.ID, "error", err)
	}
}

// markFiled marks the document of a delivered submission filed at the delivery
// time. An already filed document counts as done.
func (s *FilingService) markFiled(ctx context.Context, sub *domain.FilingSubmission) bool {
	doc, ok := s.docs[sub.DocumentKind]
	if !ok || sub.DeliveredAt == nil {
		return false
	}
	err := doc.MarkFiledAt(ctx, sub.DocumentID, *sub.DeliveredAt)
	if err != nil && !errors.Is(err, domain.ErrFilingAlreadyFiled) {
		slog.Error("marking the delivered document filed failed",
			"submission_id", sub.ID, "kind", string(sub.DocumentKind), "id", sub.DocumentID, "error", err)
		return false
	}
	return true
}

// List returns the submissions of one document, newest first.
func (s *FilingService) List(ctx context.Context, kind domain.FilingKind, id int64) ([]domain.FilingSubmission, error) {
	if _, err := s.document(kind, id); err != nil {
		return nil, err
	}
	subs, err := s.repo.ListByDocument(ctx, kind, id)
	if err != nil {
		return nil, fmt.Errorf("listing submissions of %s %d: %w", kind, id, err)
	}
	return subs, nil
}

// Receipt returns the stored signed message (ZFO) of a submission and a safe
// download file name derived from the message id.
func (s *FilingService) Receipt(ctx context.Context, submissionID int64) ([]byte, string, error) {
	sub, err := s.repo.GetByID(ctx, submissionID)
	if err != nil {
		return nil, "", fmt.Errorf("fetching submission %d for its receipt: %w", submissionID, err)
	}
	zfo, err := s.repo.GetReceipt(ctx, submissionID)
	if err != nil {
		return nil, "", fmt.Errorf("fetching the receipt of submission %d: %w", submissionID, err)
	}
	return zfo, "DZ_" + sub.DmID + ".zfo", nil
}

// LoginTestResult is what a successful login test reports. It never carries
// the username or the password.
type LoginTestResult struct {
	BoxID            string
	BoxType          string
	OwnerName        string
	CanCreateAndSend bool
	// CanReadMessages is true when the login may read delivered messages,
	// which makes every login deliver them (delivery by login).
	CanReadMessages bool
}

// TestLogin checks the stored login of an environment against ISDS. It runs
// only when the user asks for it.
func (s *FilingService) TestLogin(ctx context.Context, rawEnv string) (*LoginTestResult, error) {
	env, err := parseFilingEnv(rawEnv)
	if err != nil {
		return nil, fmt.Errorf("testing the data box login: %w", err)
	}
	client, err := s.client(env)
	if err != nil {
		return nil, err
	}
	owner, err := client.GetOwnerInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("testing the data box login for %s: %w", env.name, err)
	}
	user, err := client.GetUserInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("testing the data box login for %s: %w", env.name, err)
	}

	slog.Info("ISDS login tested", "environment", env.name, "box_type", owner.BoxType,
		"can_send", user.Privileges.CanCreateAndSend(), "can_read", user.Privileges.CanReadMessages())
	return &LoginTestResult{
		BoxID:            owner.BoxID,
		BoxType:          owner.BoxType,
		OwnerName:        ownerDisplayName(owner),
		CanCreateAndSend: user.Privileges.CanCreateAndSend(),
		CanReadMessages:  user.Privileges.CanReadMessages(),
	}, nil
}

func ownerDisplayName(o *isds.OwnerInfo) string {
	if o.FirmName != "" {
		return o.FirmName
	}
	return strings.TrimSpace(o.GivenNames + " " + o.LastName)
}
