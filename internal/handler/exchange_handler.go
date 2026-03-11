package handler

import (
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service/cnb"
)

// ExchangeHandler handles exchange rate API endpoints.
type ExchangeHandler struct {
	cnbClient *cnb.Client
}

// NewExchangeHandler creates a new ExchangeHandler.
func NewExchangeHandler(cnbClient *cnb.Client) *ExchangeHandler {
	return &ExchangeHandler{cnbClient: cnbClient}
}

// Routes returns the chi.Router for exchange rate endpoints.
func (h *ExchangeHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.getExchangeRate)
	return r
}

// exchangeRateResponse is the JSON response for an exchange rate query.
type exchangeRateResponse struct {
	CurrencyCode string        `json:"currency_code"`
	Rate         domain.Amount `json:"rate"`
	Date         string        `json:"date"`
}

var currencyCodeRegex = regexp.MustCompile(`^[A-Z]{3}$`)

// getExchangeRate handles GET /api/v1/exchange-rate?currency=EUR&date=2026-03-10
func (h *ExchangeHandler) getExchangeRate(w http.ResponseWriter, r *http.Request) {
	currency := r.URL.Query().Get("currency")
	if currency == "" {
		respondError(w, http.StatusBadRequest, "currency parameter is required")
		return
	}
	if !currencyCodeRegex.MatchString(currency) {
		respondError(w, http.StatusBadRequest, "currency must be a 3-letter uppercase code")
		return
	}

	var date time.Time
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		date = time.Now()
	} else {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "date must be in YYYY-MM-DD format")
			return
		}
	}

	rate, err := h.cnbClient.GetRate(r.Context(), currency, date)
	if err != nil {
		slog.Warn("failed to fetch CNB exchange rate", "currency", currency, "date", date, "error", err)
		respondError(w, http.StatusBadGateway, "exchange rate unavailable, try again later")
		return
	}

	// Convert rate (CZK per 1 unit) to halere as domain.Amount
	rateAmount := domain.FromFloat(rate)

	respondJSON(w, http.StatusOK, exchangeRateResponse{
		CurrencyCode: currency,
		Rate:         rateAmount,
		Date:         date.Format("2006-01-02"),
	})
}
