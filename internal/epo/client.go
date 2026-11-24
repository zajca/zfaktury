// Package epo is a client for the Czech tax portal's "Podatelna EPO" endpoint
// (https://mojedane.gov.cz/dpr/epo_podani), which accepts the XML files
// ZFaktury generates and either validates them (test=1) or turns them into a
// prefilled online form (otevriFormular=1).
package epo

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// Package defaults, overridable through Option values (and in the application
// through the [epo] config section).
const (
	// DefaultBaseURL is the Podatelna EPO base; the submission endpoint is
	// {base}/epo_podani.
	DefaultBaseURL = "https://mojedane.gov.cz/dpr"
	// DefaultAllowedHost is the only host a returned form link may point to.
	DefaultAllowedHost = "mojedane.gov.cz"
	// DefaultTimeout bounds each upstream call.
	DefaultTimeout = 10 * time.Second
	// DefaultMaxRequestBytes bounds the XML sent upstream.
	DefaultMaxRequestBytes int64 = 5 << 20
	// DefaultMaxResponseBytes bounds the response read from upstream.
	DefaultMaxResponseBytes int64 = 1 << 20

	// FormPathPrefix is the path every form link must live under.
	FormPathPrefix = "/dpr/"

	submissionPath = "/epo_podani"
	testModeCode   = "TEST_REZIM"
	debugBodyBytes = 200
)

// Finding types reported by EPO (Podatelna EPO v1.11, Priloha 3).
const (
	TypeInfo        = "I"
	TypeStructure   = "S"
	TypeCritical    = "K"
	TypeNonCritical = "N"
	TypePassable    = "P"
	TypeException   = "E"
)

var (
	// ErrEPOTimeout is returned when EPO does not answer in time.
	ErrEPOTimeout = errors.New("EPO request timed out")
	// ErrEPOUnavailable is returned for transport failures and non-2xx answers.
	ErrEPOUnavailable = errors.New("EPO unavailable")
	// ErrUnexpectedResponse is returned when EPO answers 2xx with something
	// that is neither a form link nor a list of findings.
	ErrUnexpectedResponse = errors.New("unexpected EPO response")
	// ErrInvalidFormURL is returned when a link is not an https link to the
	// allowed host under FormPathPrefix.
	ErrInvalidFormURL = errors.New("invalid EPO form URL")
	// ErrInvalidXML is returned when the input is not a ZFaktury-style XML
	// document or exceeds the configured size.
	ErrInvalidXML = errors.New("XML is not acceptable for EPO")
)

// Finding is one item of EPO's <Chyby> list.
type Finding struct {
	Type    string
	Code    string
	Field   string
	Section string
	Row     string
	Info    string
	Text    string
}

func (f Finding) normalizedType() string {
	return strings.ToUpper(strings.TrimSpace(f.Type))
}

// IsBlocking reports whether the finding means EPO cannot process the file.
// Only passable, non-critical and info findings do not block; an empty or
// unknown type blocks, so a new EPO finding type is never read as harmless.
func (f Finding) IsBlocking() bool {
	switch f.normalizedType() {
	case TypePassable, TypeNonCritical, TypeInfo:
		return false
	default:
		return true
	}
}

// IsWarning reports whether the finding is a passable or non-critical one the
// user should see before submitting.
func (f Finding) IsWarning() bool {
	switch f.normalizedType() {
	case TypePassable, TypeNonCritical:
		return true
	default:
		return false
	}
}

// RejectedError is returned by OpenForm when EPO answers with findings instead
// of a form link.
type RejectedError struct {
	Findings []Finding
}

func (e *RejectedError) Error() string {
	return fmt.Sprintf("EPO rejected the submission with %d finding(s)", len(e.Findings))
}

// Client talks to Podatelna EPO.
type Client struct {
	baseURL          string
	allowedHost      string
	maxRequestBytes  int64
	maxResponseBytes int64
	httpClient       *http.Client
}

// Option configures the Client.
type Option func(*Client)

// WithBaseURL overrides the default base URL (useful for testing). An empty
// value keeps the default.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		if baseURL != "" {
			c.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

// WithAllowedHost overrides the host a returned form link must point to. An
// empty value keeps the default.
func WithAllowedHost(host string) Option {
	return func(c *Client) {
		if host != "" {
			c.allowedHost = host
		}
	}
}

// WithTimeout sets the per-call HTTP timeout. A non-positive value keeps the
// default.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.httpClient.Timeout = d
		}
	}
}

// WithMaxRequestBytes sets the largest XML the client sends. A non-positive
// value keeps the default.
func WithMaxRequestBytes(n int64) Option {
	return func(c *Client) {
		if n > 0 {
			c.maxRequestBytes = n
		}
	}
}

// WithMaxResponseBytes sets the largest response the client reads. A
// non-positive value keeps the default.
func WithMaxResponseBytes(n int64) Option {
	return func(c *Client) {
		if n > 0 {
			c.maxResponseBytes = n
		}
	}
}

// WithHTTPClient replaces the HTTP client. Redirects stay disabled.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
			c.httpClient.CheckRedirect = noRedirect
		}
	}
}

func noRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

// NewClient creates a Podatelna EPO client.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL:          DefaultBaseURL,
		allowedHost:      DefaultAllowedHost,
		maxRequestBytes:  DefaultMaxRequestBytes,
		maxResponseBytes: DefaultMaxResponseBytes,
		httpClient: &http.Client{
			Timeout:       DefaultTimeout,
			CheckRedirect: noRedirect,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// AllowedHost returns the host a form link must point to.
func (c *Client) AllowedHost() string { return c.allowedHost }

// OpenForm uploads the XML with otevriFormular=1 and returns the validated
// link to the prefilled EPO form. EPO answering with findings yields a
// *RejectedError.
func (c *Client) OpenForm(ctx context.Context, xmlData []byte) (string, error) {
	res, err := c.post(ctx, "otevriFormular", xmlData)
	if err != nil {
		return "", err
	}
	switch {
	case res.link != "":
		if err := ValidateFormURL(res.link, c.allowedHost); err != nil {
			slog.Warn("EPO returned a form link that failed validation", "error", err)
			return "", fmt.Errorf("%w: %w", ErrUnexpectedResponse, err)
		}
		return res.link, nil
	case len(res.findings) > 0:
		return "", &RejectedError{Findings: res.findings}
	default:
		return "", c.unexpected(res)
	}
}

// Check uploads the XML with test=1, which validates it without filing. The
// TEST_REZIM marker is removed from the returned findings. A response that
// carries neither the marker nor a blocking finding is an error, so an
// unexpected answer is never read as success.
func (c *Client) Check(ctx context.Context, xmlData []byte) ([]Finding, error) {
	res, err := c.post(ctx, "test", xmlData)
	if err != nil {
		return nil, err
	}
	if res.findings == nil {
		return nil, c.unexpected(res)
	}

	findings := make([]Finding, 0, len(res.findings))
	sawTestMode := false
	sawBlocking := false
	for _, f := range res.findings {
		if f.Code == testModeCode {
			sawTestMode = true
			continue
		}
		if f.IsBlocking() {
			sawBlocking = true
		}
		findings = append(findings, f)
	}
	if !sawTestMode && !sawBlocking {
		return nil, c.unexpected(res)
	}
	return findings, nil
}

// ValidateFormURL checks that rawURL is an https link without credentials to
// allowedHost under FormPathPrefix.
func ValidateFormURL(rawURL, allowedHost string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: unparsable", ErrInvalidFormURL)
	}
	cleaned := path.Clean(u.Path)
	switch {
	case u.Scheme != "https":
		return fmt.Errorf("%w: scheme %q is not https", ErrInvalidFormURL, u.Scheme)
	case u.User != nil:
		return fmt.Errorf("%w: credentials in URL", ErrInvalidFormURL)
	case !strings.EqualFold(u.Host, allowedHost):
		return fmt.Errorf("%w: host %q is not allowed", ErrInvalidFormURL, u.Host)
	case cleaned != u.Path:
		return fmt.Errorf("%w: path contains dot segments", ErrInvalidFormURL)
	case !strings.HasPrefix(cleaned, FormPathPrefix):
		return fmt.Errorf("%w: path is outside %s", ErrInvalidFormURL, FormPathPrefix)
	}
	return nil
}

// parsed is the decoded content of an EPO answer.
type parsed struct {
	status   int
	body     []byte
	link     string
	findings []Finding
}

func (c *Client) unexpected(res *parsed) error {
	n := min(len(res.body), debugBodyBytes)
	slog.Debug("unexpected EPO response", "status", res.status, "body_prefix", string(res.body[:n]))
	return ErrUnexpectedResponse
}

func (c *Client) post(ctx context.Context, mode string, xmlData []byte) (*parsed, error) {
	if err := c.validateInput(xmlData); err != nil {
		return nil, err
	}

	endpoint := c.baseURL + submissionPath + "?" + mode + "=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(xmlData))
	if err != nil {
		return nil, fmt.Errorf("creating EPO request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Accept", "application/xml")

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, mapTransportError(err)
	}
	defer func() { _ = resp.Body.Close() }()

	slog.Debug("EPO call",
		"mode", mode,
		"status", resp.StatusCode,
		"request_bytes", len(xmlData),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("%w: HTTP %d", ErrEPOUnavailable, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseBytes+1))
	if err != nil {
		return nil, mapTransportError(err)
	}
	if int64(len(body)) > c.maxResponseBytes {
		return nil, fmt.Errorf("%w: response exceeds %d bytes", ErrUnexpectedResponse, c.maxResponseBytes)
	}

	res := &parsed{status: resp.StatusCode, body: body}
	parseBody(res)
	return res, nil
}

func (c *Client) validateInput(xmlData []byte) error {
	if int64(len(xmlData)) > c.maxRequestBytes {
		return fmt.Errorf("%w: %d bytes exceed the %d byte limit", ErrInvalidXML, len(xmlData), c.maxRequestBytes)
	}
	trimmed := bytes.TrimPrefix(xmlData, []byte("\xef\xbb\xbf"))
	trimmed = bytes.TrimLeft(trimmed, " \t\r\n")
	if !bytes.HasPrefix(trimmed, []byte("<?xml")) && !bytes.HasPrefix(trimmed, []byte("<Pisemnost")) {
		return fmt.Errorf("%w: missing XML declaration or <Pisemnost> root", ErrInvalidXML)
	}
	return nil
}

func mapTransportError(err error) error {
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("EPO request canceled: %w", err)
	}
	if errors.Is(err, context.DeadlineExceeded) || isTimeout(err) {
		return fmt.Errorf("%w: %w", ErrEPOTimeout, err)
	}
	return fmt.Errorf("%w: %w", ErrEPOUnavailable, err)
}

func isTimeout(err error) bool {
	var t interface{ Timeout() bool }
	return errors.As(err, &t) && t.Timeout()
}

// xmlChyba mirrors <Chyba Typ=".." Zkr=".." Polozka=".." Oddil=".." Radek=".." DoplInfo=".."><Text/></Chyba>.
type xmlChyba struct {
	Typ     string `xml:"Typ,attr"`
	Zkr     string `xml:"Zkr,attr"`
	Polozka string `xml:"Polozka,attr"`
	Oddil   string `xml:"Oddil,attr"`
	Radek   string `xml:"Radek,attr"`
	Info    string `xml:"DoplInfo,attr"`
	Text    string `xml:"Text"`
}

type xmlChyby struct {
	Chyby []xmlChyba `xml:"Chyba"`
}

// parseBody fills link or findings from the first element of the document.
// Documents with any other root, or malformed ones, leave both empty.
func parseBody(res *parsed) {
	dec := xml.NewDecoder(bytes.NewReader(res.body))
	for {
		tok, err := dec.Token()
		if err != nil {
			return
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "URL", "Url":
			var link string
			if err := dec.DecodeElement(&link, &start); err == nil {
				res.link = strings.TrimSpace(link)
			}
		case "Chyby":
			var list xmlChyby
			if err := dec.DecodeElement(&list, &start); err != nil {
				return
			}
			res.findings = make([]Finding, 0, len(list.Chyby))
			for _, ch := range list.Chyby {
				res.findings = append(res.findings, Finding{
					Type:    ch.Typ,
					Code:    ch.Zkr,
					Field:   ch.Polozka,
					Section: ch.Oddil,
					Row:     ch.Radek,
					Info:    ch.Info,
					Text:    strings.TrimSpace(ch.Text),
				})
			}
		}
		return
	}
}
