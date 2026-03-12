package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/zajca/zfaktury/internal/service"
)

// TaxCalendarHandler handles tax calendar HTTP requests.
type TaxCalendarHandler struct {
	svc *service.TaxCalendarService
}

// NewTaxCalendarHandler creates a new TaxCalendarHandler.
func NewTaxCalendarHandler(svc *service.TaxCalendarService) *TaxCalendarHandler {
	return &TaxCalendarHandler{svc: svc}
}

// taxDeadlineResponse is the JSON response for a single tax deadline.
type taxDeadlineResponse struct {
	Name        string `json:"name"`
	Date        string `json:"date"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// taxCalendarResponse is the JSON response for the tax calendar.
type taxCalendarResponse struct {
	Year      int                   `json:"year"`
	Deadlines []taxDeadlineResponse `json:"deadlines"`
}

// GetCalendar returns all tax deadlines for a given year.
func (h *TaxCalendarHandler) GetCalendar(w http.ResponseWriter, r *http.Request) {
	year := time.Now().Year()
	if v := r.URL.Query().Get("year"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed < 2000 || parsed > 2100 {
			respondError(w, http.StatusBadRequest, "invalid year parameter")
			return
		}
		year = parsed
	}

	deadlines := h.svc.GetDeadlines(year)

	resp := taxCalendarResponse{
		Year:      year,
		Deadlines: make([]taxDeadlineResponse, 0, len(deadlines)),
	}
	for _, d := range deadlines {
		resp.Deadlines = append(resp.Deadlines, taxDeadlineResponse{
			Name:        d.Name,
			Date:        d.Date.Format("2006-01-02"),
			Type:        d.Type,
			Description: d.Description,
		})
	}

	respondJSON(w, http.StatusOK, resp)
}
