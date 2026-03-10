package ares

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

const (
	defaultBaseURL = "https://ares.gov.cz/ekonomicke-subjekty-v-be/rest"
	defaultTimeout = 10 * time.Second
)

var icoRegexp = regexp.MustCompile(`^\d{8}$`)

// Client is an HTTP client for the Czech ARES business registry API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Option configures the ARES Client.
type Option func(*Client)

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// WithBaseURL overrides the default ARES API base URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// NewClient creates a new ARES API client.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// LookupByICO looks up a company by its ICO (identification number) in the ARES registry.
func (c *Client) LookupByICO(ctx context.Context, ico string) (*domain.Contact, error) {
	if !icoRegexp.MatchString(ico) {
		return nil, errors.New("invalid ICO format: must be exactly 8 digits")
	}

	url := fmt.Sprintf("%s/ekonomicke-subjekty/%s", c.baseURL, ico)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating ARES request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, errors.New("ARES service timeout")
		}
		if isTimeoutError(err) {
			return nil, errors.New("ARES service timeout")
		}
		return nil, fmt.Errorf("ARES service error: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// continue below
	case http.StatusNotFound:
		return nil, errors.New("subject not found")
	case http.StatusTooManyRequests:
		return nil, errors.New("rate limited")
	default:
		return nil, fmt.Errorf("ARES service error: HTTP %d", resp.StatusCode)
	}

	// Limit response body to 1 MB to prevent memory exhaustion.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading ARES response: %w", err)
	}

	var aresResp aresResponse
	if err := json.Unmarshal(body, &aresResp); err != nil {
		return nil, fmt.Errorf("parsing ARES response: %w", err)
	}

	return aresResp.toContact(), nil
}

// isTimeoutError checks if the error is a timeout (net.Error with Timeout()).
func isTimeoutError(err error) bool {
	type timeouter interface {
		Timeout() bool
	}
	var t timeouter
	if errors.As(err, &t) {
		return t.Timeout()
	}
	return false
}
