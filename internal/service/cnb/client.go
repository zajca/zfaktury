package cnb

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	baseURL       = "https://www.cnb.cz/cs/financni-trhy/devizovy-trh/kurzy-devizoveho-trhu/kurzy-devizoveho-trhu/denni_kurz.txt"
	cacheTTL      = 1 * time.Hour
	maxFallback   = 5 // max days to try backwards for weekends/holidays
)

// cacheEntry holds cached exchange rates for a specific date.
type cacheEntry struct {
	rates     map[string]ExchangeRate // keyed by currency code
	fetchedAt time.Time
}

// Client fetches and caches exchange rates from the Czech National Bank.
type Client struct {
	httpClient *http.Client
	baseURL    string
	mu         sync.RWMutex
	cache      map[string]cacheEntry // keyed by date string "DD.MM.YYYY"
}

// NewClient creates a new CNB exchange rate client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    baseURL,
		cache:      make(map[string]cacheEntry),
	}
}

// SetBaseURL overrides the CNB API base URL. Intended for testing.
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// GetRate returns the CZK exchange rate per 1 unit of the given foreign currency
// for the specified date. If no rates are available for the exact date (weekends,
// holidays), it tries up to 5 previous days.
func (c *Client) GetRate(ctx context.Context, currencyCode string, date time.Time) (float64, error) {
	currencyCode = strings.ToUpper(currencyCode)
	if len(currencyCode) != 3 {
		return 0, fmt.Errorf("invalid currency code: %s", currencyCode)
	}

	for i := 0; i < maxFallback; i++ {
		d := date.AddDate(0, 0, -i)
		rates, err := c.getRatesForDate(ctx, d)
		if err != nil {
			continue // try previous day
		}
		rate, ok := rates[currencyCode]
		if !ok {
			continue // currency may appear on a different trading day
		}
		// Rate is CZK per Amount units, convert to CZK per 1 unit
		return rate.Rate / float64(rate.Amount), nil
	}

	return 0, fmt.Errorf("no exchange rates available for %s within %d days of %s", currencyCode, maxFallback, date.Format("2006-01-02"))
}

// getRatesForDate fetches rates for a specific date, using cache if available.
func (c *Client) getRatesForDate(ctx context.Context, date time.Time) (map[string]ExchangeRate, error) {
	dateKey := date.Format("02.01.2006")

	// Check cache with read lock
	c.mu.RLock()
	entry, ok := c.cache[dateKey]
	c.mu.RUnlock()

	if ok && time.Since(entry.fetchedAt) < cacheTTL {
		return entry.rates, nil
	}

	// Fetch fresh data
	rates, err := c.fetchRates(ctx, dateKey)
	if err != nil {
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.cache[dateKey] = cacheEntry{
		rates:     rates,
		fetchedAt: time.Now(),
	}
	c.mu.Unlock()

	return rates, nil
}

// fetchRates downloads and parses the CNB exchange rate sheet for the given date.
func (c *Client) fetchRates(ctx context.Context, dateKey string) (map[string]ExchangeRate, error) {
	url := fmt.Sprintf("%s?date=%s", c.baseURL, dateKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching CNB rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CNB returned status %d", resp.StatusCode)
	}

	const maxCNBResponseBytes = 256 * 1024 // 256 KB is generous for ~35 currency lines
	limited := io.LimitReader(resp.Body, maxCNBResponseBytes)
	return parseRates(limited)
}

// parseRates parses the pipe-delimited CNB exchange rate format.
// Format:
//
//	Line 1: date + sequence number (e.g., "10.03.2026 #049")
//	Line 2: column headers (země|měna|množství|kód|kurz)
//	Lines 3+: data rows (country|currency|amount|code|rate)
//
// Rate uses comma as decimal separator (e.g., "25,340").
func parseRates(r io.Reader) (map[string]ExchangeRate, error) {
	scanner := bufio.NewScanner(r)
	rates := make(map[string]ExchangeRate)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip the first two header lines
		if lineNum <= 2 {
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) != 5 {
			continue
		}

		amount, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil {
			continue
		}

		// Rate uses comma as decimal separator
		rateStr := strings.TrimSpace(parts[4])
		rateStr = strings.ReplaceAll(rateStr, ",", ".")
		rate, err := strconv.ParseFloat(rateStr, 64)
		if err != nil {
			continue
		}

		code := strings.TrimSpace(parts[3])
		rates[code] = ExchangeRate{
			Country:  strings.TrimSpace(parts[0]),
			Currency: strings.TrimSpace(parts[1]),
			Amount:   amount,
			Code:     code,
			Rate:     rate,
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading CNB response: %w", err)
	}

	if len(rates) == 0 {
		return nil, fmt.Errorf("no rates parsed from CNB response")
	}

	return rates, nil
}
