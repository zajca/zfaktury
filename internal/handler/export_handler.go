package handler

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// ExportHandler handles CSV export HTTP requests.
type ExportHandler struct {
	invoiceSvc *service.InvoiceService
	expenseSvc *service.ExpenseService
}

// NewExportHandler creates a new ExportHandler.
func NewExportHandler(invoiceSvc *service.InvoiceService, expenseSvc *service.ExpenseService) *ExportHandler {
	return &ExportHandler{
		invoiceSvc: invoiceSvc,
		expenseSvc: expenseSvc,
	}
}

// ExportInvoices exports invoices for a given year as a CSV file.
func (h *ExportHandler) ExportInvoices(w http.ResponseWriter, r *http.Request) {
	year := time.Now().Year()
	if v := r.URL.Query().Get("year"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed < 2000 || parsed > 2100 {
			respondError(w, http.StatusBadRequest, "invalid year parameter")
			return
		}
		year = parsed
	}

	dateFrom := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)

	filter := domain.InvoiceFilter{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
		Limit:    10000,
		Offset:   0,
	}

	invoices, _, err := h.invoiceSvc.List(r.Context(), filter)
	if err != nil {
		slog.Error("exporting invoices", "year", year, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list invoices")
		return
	}

	filename := fmt.Sprintf("faktury_%d.csv", year)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)

	// Write UTF-8 BOM for Excel compatibility.
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	cw.Comma = ';'

	header := []string{
		"Cislo", "Typ", "Stav", "Odberatel", "Datum vystaveni",
		"Datum splatnosti", "DUZP", "Castka bez DPH", "DPH", "Celkem", "Mena",
	}
	_ = cw.Write(header)

	for _, inv := range invoices {
		customerName := ""
		if inv.Customer != nil {
			customerName = inv.Customer.Name
		}

		row := []string{
			inv.InvoiceNumber,
			inv.Type,
			inv.Status,
			customerName,
			formatCSVDate(inv.IssueDate),
			formatCSVDate(inv.DueDate),
			formatCSVDate(inv.DeliveryDate),
			formatCSVAmount(inv.SubtotalAmount),
			formatCSVAmount(inv.VATAmount),
			formatCSVAmount(inv.TotalAmount),
			inv.CurrencyCode,
		}
		_ = cw.Write(row)
	}

	cw.Flush()
}

// ExportExpenses exports expenses for a given year as a CSV file.
func (h *ExportHandler) ExportExpenses(w http.ResponseWriter, r *http.Request) {
	year := time.Now().Year()
	if v := r.URL.Query().Get("year"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed < 2000 || parsed > 2100 {
			respondError(w, http.StatusBadRequest, "invalid year parameter")
			return
		}
		year = parsed
	}

	dateFrom := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)

	filter := domain.ExpenseFilter{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
		Limit:    10000,
		Offset:   0,
	}

	expenses, _, err := h.expenseSvc.List(r.Context(), filter)
	if err != nil {
		slog.Error("exporting expenses", "year", year, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list expenses")
		return
	}

	filename := fmt.Sprintf("vydaje_%d.csv", year)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)

	// Write UTF-8 BOM for Excel compatibility.
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	cw.Comma = ';'

	header := []string{
		"Cislo", "Popis", "Kategorie", "Dodavatel", "Datum", "Castka", "DPH", "Mena",
	}
	_ = cw.Write(header)

	for _, exp := range expenses {
		vendorName := ""
		if exp.Vendor != nil {
			vendorName = exp.Vendor.Name
		}

		row := []string{
			exp.ExpenseNumber,
			exp.Description,
			exp.Category,
			vendorName,
			formatCSVDate(exp.IssueDate),
			formatCSVAmount(exp.Amount),
			formatCSVAmount(exp.VATAmount),
			exp.CurrencyCode,
		}
		_ = cw.Write(row)
	}

	cw.Flush()
}

// formatCSVAmount formats a domain.Amount for CSV output with comma as decimal separator.
func formatCSVAmount(a domain.Amount) string {
	s := fmt.Sprintf("%.2f", float64(a)/100.0)
	return strings.Replace(s, ".", ",", 1)
}

// formatCSVDate formats a time.Time as DD.MM.YYYY for Czech locale CSV output.
func formatCSVDate(t time.Time) string {
	return t.Format("02.01.2006")
}
