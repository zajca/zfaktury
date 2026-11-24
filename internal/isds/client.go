// Package isds is a client for the Czech data-box system (ISDS) web services
// v2.0 (https://www.datovka.gov.cz/static/wsdl/v20/). It supports the few
// operations ZFaktury needs to send a filing as a data message and to fetch
// its delivery evidence, over SOAP 1.1 with HTTP basic authentication.
//
// The client never logs and never embeds credentials in errors. The WSDL and
// XSD files the envelopes follow are kept in docs/isds/wsdl/.
package isds

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Base URLs of the two ISDS environments (without service path).
const (
	BaseURLProduction = "https://ws1.datovka.gov.cz"
	BaseURLTest       = "https://ws1.datovka-test.gov.cz"
)

// Environment names the ISDS environment a client talks to. The caller must
// choose one explicitly; there is no default.
type Environment string

// The supported environments.
const (
	EnvironmentProduction Environment = "production"
	EnvironmentTest       Environment = "test"
)

// ParseEnvironment converts a configuration value to an Environment.
func ParseEnvironment(s string) (Environment, error) {
	switch e := Environment(s); e {
	case EnvironmentProduction, EnvironmentTest:
		return e, nil
	default:
		return "", fmt.Errorf("%w: environment must be %q or %q", ErrInvalidConfig, EnvironmentProduction, EnvironmentTest)
	}
}

// BaseURL returns the service base URL of the environment.
func (e Environment) BaseURL() (string, error) {
	switch e {
	case EnvironmentProduction:
		return BaseURLProduction, nil
	case EnvironmentTest:
		return BaseURLTest, nil
	default:
		return "", fmt.Errorf("%w: unknown environment", ErrInvalidConfig)
	}
}

// Service paths of the web services the supported operations live on, taken
// from the soap:address of each WSDL in docs/isds/wsdl/.
const (
	pathMessageOperations = "/DS/dz"       // dm_operations.wsdl
	pathMessageInfo       = "/DS/dx"       // dm_info.wsdl
	pathAccessInfo        = "/DS/DsManage" // db_access.wsdl
)

// Package defaults, overridable through Option values.
const (
	// DefaultTimeout bounds each call, including a ZFO download.
	DefaultTimeout = 60 * time.Second
	// DefaultMaxResponseBytes bounds the response read from ISDS. It leaves
	// room for the base64 form of a 20 MiB message plus the SOAP wrapper.
	DefaultMaxResponseBytes int64 = 32 << 20
)

// Credentials are the ISDS login of the box (or delegated user). The value is
// redacted by every formatting and logging path.
type Credentials struct {
	Username string
	Password string
}

const redacted = "[REDACTED]"

// String implements fmt.Stringer and never reveals the secrets.
func (Credentials) String() string { return "isds.Credentials(" + redacted + ")" }

// GoString implements fmt.GoStringer and never reveals the secrets.
func (Credentials) GoString() string { return "isds.Credentials(" + redacted + ")" }

// LogValue implements slog.LogValuer and never reveals the secrets.
func (Credentials) LogValue() slog.Value { return slog.StringValue(redacted) }

// MarshalJSON never reveals the secrets.
func (Credentials) MarshalJSON() ([]byte, error) { return json.Marshal(redacted) }

// MarshalText never reveals the secrets.
func (Credentials) MarshalText() ([]byte, error) { return []byte(redacted), nil }

func (c Credentials) validate() error {
	switch {
	case c.Username == "":
		return fmt.Errorf("%w: username is required", ErrInvalidConfig)
	case c.Password == "":
		return fmt.Errorf("%w: password is required", ErrInvalidConfig)
	case strings.Contains(c.Username, ":"):
		return fmt.Errorf("%w: username must not contain a colon", ErrInvalidConfig)
	}
	return nil
}

// authenticator adds the login to a request. Basic auth is the only mode;
// another mode only needs another implementation.
type authenticator interface {
	apply(req *http.Request)
}

// basicAuth is used through a pointer: fmt prints a nested pointer as an
// address, so reflection-based printing of a Client (which cannot call methods
// on unexported fields) never reaches the credentials.
type basicAuth struct{ creds Credentials }

func (a *basicAuth) apply(req *http.Request) {
	req.SetBasicAuth(a.creds.Username, a.creds.Password)
}

func (*basicAuth) String() string   { return "basicAuth(" + redacted + ")" }
func (*basicAuth) GoString() string { return "basicAuth(" + redacted + ")" }

// Client talks to the ISDS web services of one environment as one login.
type Client struct {
	baseURL          string
	auth             authenticator
	maxResponseBytes int64
	httpClient       *http.Client
}

// String implements fmt.Stringer without exposing the login. The value
// receiver also covers a dereferenced Client.
func (c Client) String() string { return "isds.Client(" + c.baseURL + ")" }

// GoString implements fmt.GoStringer without exposing the login.
func (c Client) GoString() string { return c.String() }

// Option configures the Client.
type Option func(*Client)

// withBaseURL overrides the environment's base URL for tests against a fake.
// Only https URLs (or http to a loopback host) are accepted.
func withBaseURL(baseURL string) Option {
	return func(c *Client) {
		if baseURL != "" {
			c.baseURL = strings.TrimRight(baseURL, "/")
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

// WithMaxResponseBytes sets the largest response the client reads. A
// non-positive value keeps the default.
func WithMaxResponseBytes(n int64) Option {
	return func(c *Client) {
		if n > 0 {
			c.maxResponseBytes = n
		}
	}
}

// WithHTTPClient replaces the HTTP client. The given client is copied and the
// copy never follows redirects. Its transport must verify TLS certificates.
// A client without a timeout inherits the timeout configured so far.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			cp := *hc
			cp.CheckRedirect = noRedirect
			if cp.Timeout == 0 {
				cp.Timeout = c.httpClient.Timeout
			}
			c.httpClient = &cp
		}
	}
}

func noRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

// NewClient creates a client for env, logging in with creds. It fails when
// the environment or the credentials are missing; there are no defaults.
func NewClient(env Environment, creds Credentials, opts ...Option) (*Client, error) {
	base, err := env.BaseURL()
	if err != nil {
		return nil, err
	}
	if err := creds.validate(); err != nil {
		return nil, err
	}

	c := &Client{
		baseURL:          base,
		auth:             &basicAuth{creds: creds},
		maxResponseBytes: DefaultMaxResponseBytes,
		httpClient: &http.Client{
			Timeout:       DefaultTimeout,
			CheckRedirect: noRedirect,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	if err := validateBaseURL(c.baseURL); err != nil {
		return nil, err
	}
	return c, nil
}

func validateBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%w: unusable base URL", ErrInvalidConfig)
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("%w: base URL must not carry credentials, query or fragment", ErrInvalidConfig)
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		if ip := net.ParseIP(u.Hostname()); u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback()) {
			return nil
		}
	}
	return fmt.Errorf("%w: base URL must be https", ErrInvalidConfig)
}

// soapEnvelope wraps a request operation element.
type soapEnvelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Body    soapBody
}

type soapBody struct {
	XMLName xml.Name `xml:"Body"`
	Op      any
}

type soapFault struct {
	Code   string `xml:"faultcode"`
	String string `xml:"faultstring"`
}

// responseEnvelope decodes any SOAP answer: either a fault or the single
// operation response element.
type responseEnvelope[T any] struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		Fault    *soapFault `xml:"Fault"`
		Response *T         `xml:",any"`
	} `xml:"Body"`
}

// call sends op (a struct whose XMLName names the operation element in the
// ISDS namespace) to the service at path and decodes the response element.
func call[T any](ctx context.Context, c *Client, path, name string, op any) (*T, error) {
	payload, err := xml.Marshal(soapEnvelope{Body: soapBody{Op: op}})
	if err != nil {
		return nil, fmt.Errorf("encoding %s request: %w", name, err)
	}
	payload = append([]byte(xml.Header), payload...)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating %s request: %w", name, err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("Accept", "text/xml")
	req.Header.Set("SOAPAction", `""`)
	c.auth.apply(req)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, mapTransportError(name, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseBytes+1))
	if err != nil {
		return nil, mapTransportError(name, err)
	}
	slog.Debug("ISDS call",
		"operation", name,
		"status", resp.StatusCode,
		"request_bytes", len(payload),
		"response_bytes", len(body),
		"duration_ms", time.Since(start).Milliseconds(),
	)
	if int64(len(body)) > c.maxResponseBytes {
		return nil, fmt.Errorf("%s: %w", name, ErrResponseTooLarge)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%s: status %d: %w", name, resp.StatusCode, ErrUnauthorized)
	}

	var env responseEnvelope[T]
	decodeErr := xml.Unmarshal(body, &env)
	if decodeErr == nil && env.Body.Fault != nil {
		return nil, &FaultError{Code: truncate(env.Body.Fault.Code), Message: truncate(env.Body.Fault.String)}
	}

	switch {
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		return nil, fmt.Errorf("%s: redirect (status %d) refused: %w", name, resp.StatusCode, ErrUnexpectedResponse)
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return nil, fmt.Errorf("%s: status %d: %w", name, resp.StatusCode, ErrISDSUnavailable)
	case decodeErr != nil || env.Body.Response == nil:
		return nil, fmt.Errorf("%s: answer is not a SOAP response: %w", name, ErrUnexpectedResponse)
	}
	return env.Body.Response, nil
}

func mapTransportError(name string, err error) error {
	var netErr net.Error
	switch {
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("%s: %w", name, err)
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr) && netErr.Timeout():
		return fmt.Errorf("%s: %w: %w", name, ErrISDSTimeout, ErrISDSUnavailable)
	default:
		return fmt.Errorf("%s: %w: %w", name, ErrISDSUnavailable, err)
	}
}

// status codes
const statusOK = "0000"

func checkDMStatus(s *dmStatus) error {
	if s == nil {
		return fmt.Errorf("%w: missing dmStatus", ErrUnexpectedResponse)
	}
	if s.Code != statusOK {
		return &StatusError{Code: truncate(s.Code), Message: truncate(s.Message), RefNumber: truncate(s.RefNumber)}
	}
	return nil
}

func checkDBStatus(s *dbStatus) error {
	if s == nil {
		return fmt.Errorf("%w: missing dbStatus", ErrUnexpectedResponse)
	}
	if s.Code != statusOK {
		return &StatusError{Code: truncate(s.Code), Message: truncate(s.Message), RefNumber: truncate(s.RefNumber)}
	}
	return nil
}

type dmStatus struct {
	Code      string `xml:"dmStatusCode"`
	Message   string `xml:"dmStatusMessage"`
	RefNumber string `xml:"dmStatusRefNumber"`
}

type dbStatus struct {
	Code      string `xml:"dbStatusCode"`
	Message   string `xml:"dbStatusMessage"`
	RefNumber string `xml:"dbStatusRefNumber"`
}

func decodeBase64(s string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(s), ""))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64 content", ErrUnexpectedResponse)
	}
	return data, nil
}
