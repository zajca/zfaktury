package fakturoid

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	pageDelay      = 700 * time.Millisecond // ~85 req/min, under 100 limit
)

// Client is an HTTP client for the Fakturoid API v3.
type Client struct {
	baseURL    string
	email      string
	apiToken   string
	httpClient *http.Client
}

// Option configures the Fakturoid Client.
type Option func(*Client)

// WithBaseURL overrides the default Fakturoid API base URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// NewClient creates a new Fakturoid API client.
// slug is the Fakturoid account slug, email is the user's email, apiToken is the API token.
func NewClient(slug, email, apiToken string, opts ...Option) *Client {
	c := &Client{
		baseURL:  fmt.Sprintf("https://app.fakturoid.cz/api/v3/accounts/%s", slug),
		email:    email,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// listPaginated fetches all pages of a paginated Fakturoid API endpoint.
// It returns the raw JSON messages for each item across all pages.
func (c *Client) listPaginated(ctx context.Context, path string) ([]json.RawMessage, error) {
	var all []json.RawMessage
	page := 1
	for {
		url := fmt.Sprintf("%s/%s?page=%d", c.baseURL, path, page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.SetBasicAuth(c.email, c.apiToken)
		req.Header.Set("User-Agent", fmt.Sprintf("ZFaktury (%s)", c.email))
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching %s page %d: %w", path, page, err)
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fakturoid API error: HTTP %d for %s", resp.StatusCode, path)
		}

		var items []json.RawMessage
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, fmt.Errorf("parsing response: %w", err)
		}

		if len(items) == 0 {
			break
		}
		all = append(all, items...)
		page++

		// Rate limiting delay between pages.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pageDelay):
		}
	}
	return all, nil
}

// ListSubjects returns all subjects (contacts) from the Fakturoid account.
func (c *Client) ListSubjects(ctx context.Context) ([]Subject, error) {
	raw, err := c.listPaginated(ctx, "subjects.json")
	if err != nil {
		return nil, fmt.Errorf("listing subjects: %w", err)
	}
	subjects := make([]Subject, 0, len(raw))
	for _, r := range raw {
		var s Subject
		if err := json.Unmarshal(r, &s); err != nil {
			return nil, fmt.Errorf("parsing subject: %w", err)
		}
		subjects = append(subjects, s)
	}
	return subjects, nil
}

// ListInvoices returns all invoices from the Fakturoid account.
func (c *Client) ListInvoices(ctx context.Context) ([]Invoice, error) {
	raw, err := c.listPaginated(ctx, "invoices.json")
	if err != nil {
		return nil, fmt.Errorf("listing invoices: %w", err)
	}
	invoices := make([]Invoice, 0, len(raw))
	for _, r := range raw {
		var inv Invoice
		if err := json.Unmarshal(r, &inv); err != nil {
			return nil, fmt.Errorf("parsing invoice: %w", err)
		}
		invoices = append(invoices, inv)
	}
	return invoices, nil
}

// ListExpenses returns all expenses from the Fakturoid account.
func (c *Client) ListExpenses(ctx context.Context) ([]Expense, error) {
	raw, err := c.listPaginated(ctx, "expenses.json")
	if err != nil {
		return nil, fmt.Errorf("listing expenses: %w", err)
	}
	expenses := make([]Expense, 0, len(raw))
	for _, r := range raw {
		var exp Expense
		if err := json.Unmarshal(r, &exp); err != nil {
			return nil, fmt.Errorf("parsing expense: %w", err)
		}
		expenses = append(expenses, exp)
	}
	return expenses, nil
}
