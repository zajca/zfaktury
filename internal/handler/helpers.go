package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
)

// respondJSON writes a JSON response with the given status code.
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("failed to encode JSON response", "error", err)
		}
	}
}

// respondError writes a JSON error response.
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, errorResponse{Error: message})
}

// parseID extracts an int64 ID from the chi URL parameter "id".
func parseID(r *http.Request) (int64, error) {
	idStr := chi.URLParam(r, "id")
	return strconv.ParseInt(idStr, 10, 64)
}

const maxPaginationLimit = 500

// parsePagination extracts limit and offset from query parameters with defaults.
func parsePagination(r *http.Request) (limit, offset int) {
	limit = 20
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	if limit > maxPaginationLimit {
		limit = maxPaginationLimit
	}
	if limit < 1 {
		limit = 1
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// parseOptionalInt64 parses an optional int64 query parameter.
// Returns nil if the parameter is empty or not present.
func parseOptionalInt64(r *http.Request, key string) *int64 {
	v := r.URL.Query().Get(key)
	if v == "" {
		return nil
	}
	parsed, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

// parseOptionalTime parses an optional time query parameter in YYYY-MM-DD format.
// Returns nil if the parameter is empty or not present.
func parseOptionalTime(r *http.Request, key string) *time.Time {
	v := r.URL.Query().Get(key)
	if v == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil
	}
	return &t
}

// parseOptionalBool parses an optional bool query parameter.
// Returns nil if the parameter is empty or not present.
func parseOptionalBool(r *http.Request, key string) *bool {
	v := r.URL.Query().Get(key)
	if v == "" {
		return nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return nil
	}
	return &b
}

// --- Shared Response DTOs ---

// errorResponse is a standard error response.
type errorResponse struct {
	Error string `json:"error"`
}

// listResponse wraps a paginated list response.
type listResponse[T any] struct {
	Data   []T `json:"data"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// formatOptionalTime formats a nullable time pointer as an RFC3339 string pointer.
func formatOptionalTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

// --- Contact DTOs ---

// contactRequest is the JSON request body for creating/updating a contact.
type contactRequest struct {
	Type             string `json:"type"`
	Name             string `json:"name"`
	ICO              string `json:"ico"`
	DIC              string `json:"dic"`
	Street           string `json:"street"`
	City             string `json:"city"`
	ZIP              string `json:"zip"`
	Country          string `json:"country"`
	Email            string `json:"email"`
	Phone            string `json:"phone"`
	Web              string `json:"web"`
	BankAccount      string `json:"bank_account"`
	BankCode         string `json:"bank_code"`
	IBAN             string `json:"iban"`
	SWIFT            string `json:"swift"`
	PaymentTermsDays int    `json:"payment_terms_days"`
	Tags             string `json:"tags"`
	Notes            string `json:"notes"`
	IsFavorite       bool   `json:"is_favorite"`
}

// toDomain converts a contactRequest to a domain.Contact.
func (r *contactRequest) toDomain() *domain.Contact {
	return &domain.Contact{
		Type:             r.Type,
		Name:             r.Name,
		ICO:              r.ICO,
		DIC:              r.DIC,
		Street:           r.Street,
		City:             r.City,
		ZIP:              r.ZIP,
		Country:          r.Country,
		Email:            r.Email,
		Phone:            r.Phone,
		Web:              r.Web,
		BankAccount:      r.BankAccount,
		BankCode:         r.BankCode,
		IBAN:             r.IBAN,
		SWIFT:            r.SWIFT,
		PaymentTermsDays: r.PaymentTermsDays,
		Tags:             r.Tags,
		Notes:            r.Notes,
		IsFavorite:       r.IsFavorite,
	}
}

// contactResponse is the JSON response for a contact.
type contactResponse struct {
	ID               int64   `json:"id"`
	Type             string  `json:"type"`
	Name             string  `json:"name"`
	ICO              string  `json:"ico"`
	DIC              string  `json:"dic"`
	Street           string  `json:"street"`
	City             string  `json:"city"`
	ZIP              string  `json:"zip"`
	Country          string  `json:"country"`
	Email            string  `json:"email"`
	Phone            string  `json:"phone"`
	Web              string  `json:"web"`
	BankAccount      string  `json:"bank_account"`
	BankCode         string  `json:"bank_code"`
	IBAN             string  `json:"iban"`
	SWIFT            string  `json:"swift"`
	PaymentTermsDays int     `json:"payment_terms_days"`
	Tags             string  `json:"tags"`
	Notes            string  `json:"notes"`
	IsFavorite       bool    `json:"is_favorite"`
	VATUnreliableAt  *string `json:"vat_unreliable_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// contactFromDomain converts a domain.Contact to a contactResponse.
func contactFromDomain(c *domain.Contact) contactResponse {
	return contactResponse{
		ID:               c.ID,
		Type:             c.Type,
		Name:             c.Name,
		ICO:              c.ICO,
		DIC:              c.DIC,
		Street:           c.Street,
		City:             c.City,
		ZIP:              c.ZIP,
		Country:          c.Country,
		Email:            c.Email,
		Phone:            c.Phone,
		Web:              c.Web,
		BankAccount:      c.BankAccount,
		BankCode:         c.BankCode,
		IBAN:             c.IBAN,
		SWIFT:            c.SWIFT,
		PaymentTermsDays: c.PaymentTermsDays,
		Tags:             c.Tags,
		Notes:            c.Notes,
		IsFavorite:       c.IsFavorite,
		VATUnreliableAt:  formatOptionalTime(c.VATUnreliableAt),
		CreatedAt:        c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        c.UpdatedAt.Format(time.RFC3339),
	}
}

// --- Invoice DTOs ---

// invoiceItemRequest is the JSON request body for an invoice line item.
type invoiceItemRequest struct {
	Description    string `json:"description"`
	Quantity       int64  `json:"quantity"` // in smallest unit (cents), e.g. 250 = 2.50
	Unit           string `json:"unit"`
	UnitPrice      int64  `json:"unit_price"` // in halere
	VATRatePercent int    `json:"vat_rate_percent"`
	SortOrder      int    `json:"sort_order"`
}

// invoiceRequest is the JSON request body for creating/updating an invoice.
type invoiceRequest struct {
	SequenceID     int64                `json:"sequence_id"`
	InvoiceNumber  string               `json:"invoice_number"`
	Type           string               `json:"type"`
	CustomerID     int64                `json:"customer_id"`
	IssueDate      string               `json:"issue_date"`
	DueDate        string               `json:"due_date"`
	DeliveryDate   string               `json:"delivery_date"`
	VariableSymbol string               `json:"variable_symbol"`
	ConstantSymbol string               `json:"constant_symbol"`
	CurrencyCode   string               `json:"currency_code"`
	ExchangeRate   int64                `json:"exchange_rate"`
	PaymentMethod  string               `json:"payment_method"`
	BankAccount    string               `json:"bank_account"`
	BankCode       string               `json:"bank_code"`
	IBAN           string               `json:"iban"`
	SWIFT          string               `json:"swift"`
	Notes          string               `json:"notes"`
	InternalNotes  string               `json:"internal_notes"`
	Items          []invoiceItemRequest `json:"items"`
}

// toDomain converts an invoiceRequest to a domain.Invoice.
func (r *invoiceRequest) toDomain() (*domain.Invoice, error) {
	if r.IssueDate == "" {
		return nil, fmt.Errorf("issue_date is required: %w", domain.ErrInvalidInput)
	}
	if r.DueDate == "" {
		return nil, fmt.Errorf("due_date is required: %w", domain.ErrInvalidInput)
	}

	inv := &domain.Invoice{
		SequenceID:     r.SequenceID,
		InvoiceNumber:  r.InvoiceNumber,
		Type:           r.Type,
		CustomerID:     r.CustomerID,
		VariableSymbol: r.VariableSymbol,
		ConstantSymbol: r.ConstantSymbol,
		CurrencyCode:   r.CurrencyCode,
		ExchangeRate:   domain.Amount(r.ExchangeRate),
		PaymentMethod:  r.PaymentMethod,
		BankAccount:    r.BankAccount,
		BankCode:       r.BankCode,
		IBAN:           r.IBAN,
		SWIFT:          r.SWIFT,
		Notes:          r.Notes,
		InternalNotes:  r.InternalNotes,
	}

	issueDate, err := time.Parse("2006-01-02", r.IssueDate)
	if err != nil {
		return nil, fmt.Errorf("invalid issue_date format, expected YYYY-MM-DD: %w", domain.ErrInvalidInput)
	}
	inv.IssueDate = issueDate

	dueDate, err := time.Parse("2006-01-02", r.DueDate)
	if err != nil {
		return nil, fmt.Errorf("invalid due_date format, expected YYYY-MM-DD: %w", domain.ErrInvalidInput)
	}
	inv.DueDate = dueDate

	if r.DeliveryDate != "" {
		t, err := time.Parse("2006-01-02", r.DeliveryDate)
		if err != nil {
			return nil, err
		}
		inv.DeliveryDate = t
	}

	for _, item := range r.Items {
		inv.Items = append(inv.Items, domain.InvoiceItem{
			Description:    item.Description,
			Quantity:       domain.Amount(item.Quantity),
			Unit:           item.Unit,
			UnitPrice:      domain.Amount(item.UnitPrice),
			VATRatePercent: item.VATRatePercent,
			SortOrder:      item.SortOrder,
		})
	}

	return inv, nil
}

// invoiceItemResponse is the JSON response for an invoice line item.
type invoiceItemResponse struct {
	ID             int64  `json:"id"`
	InvoiceID      int64  `json:"invoice_id"`
	Description    string `json:"description"`
	Quantity       int64  `json:"quantity"`
	Unit           string `json:"unit"`
	UnitPrice      int64  `json:"unit_price"`
	VATRatePercent int    `json:"vat_rate_percent"`
	VATAmount      int64  `json:"vat_amount"`
	TotalAmount    int64  `json:"total_amount"`
	SortOrder      int    `json:"sort_order"`
}

// relatedInvoiceResponse is a compact reference to a linked invoice.
type relatedInvoiceResponse struct {
	ID            int64  `json:"id"`
	InvoiceNumber string `json:"invoice_number"`
	Type          string `json:"type"`
	RelationType  string `json:"relation_type"`
}

// invoiceResponse is the JSON response for an invoice.
type invoiceResponse struct {
	ID               int64                    `json:"id"`
	SequenceID       int64                    `json:"sequence_id"`
	InvoiceNumber    string                   `json:"invoice_number"`
	Type             string                   `json:"type"`
	Status           string                   `json:"status"`
	CustomerID       int64                    `json:"customer_id"`
	IssueDate        string                   `json:"issue_date"`
	DueDate          string                   `json:"due_date"`
	DeliveryDate     string                   `json:"delivery_date"`
	VariableSymbol   string                   `json:"variable_symbol"`
	ConstantSymbol   string                   `json:"constant_symbol"`
	CurrencyCode     string                   `json:"currency_code"`
	ExchangeRate     int64                    `json:"exchange_rate"`
	PaymentMethod    string                   `json:"payment_method"`
	BankAccount      string                   `json:"bank_account"`
	BankCode         string                   `json:"bank_code"`
	IBAN             string                   `json:"iban"`
	SWIFT            string                   `json:"swift"`
	SubtotalAmount   int64                    `json:"subtotal_amount"`
	VATAmount        int64                    `json:"vat_amount"`
	TotalAmount      int64                    `json:"total_amount"`
	PaidAmount       int64                    `json:"paid_amount"`
	Notes            string                   `json:"notes"`
	InternalNotes    string                   `json:"internal_notes"`
	RelatedInvoiceID *int64                   `json:"related_invoice_id,omitempty"`
	RelationType     string                   `json:"relation_type,omitempty"`
	RelatedInvoices  []relatedInvoiceResponse `json:"related_invoices,omitempty"`
	SentAt           *string                  `json:"sent_at,omitempty"`
	PaidAt           *string                  `json:"paid_at,omitempty"`
	Items            []invoiceItemResponse    `json:"items"`
	Customer         *contactResponse         `json:"customer,omitempty"`
	CreatedAt        string                   `json:"created_at"`
	UpdatedAt        string                   `json:"updated_at"`
}

// invoiceFromDomain converts a domain.Invoice to an invoiceResponse.
func invoiceFromDomain(inv *domain.Invoice) invoiceResponse {
	resp := invoiceResponse{
		ID:               inv.ID,
		SequenceID:       inv.SequenceID,
		InvoiceNumber:    inv.InvoiceNumber,
		Type:             inv.Type,
		Status:           inv.Status,
		CustomerID:       inv.CustomerID,
		IssueDate:        inv.IssueDate.Format("2006-01-02"),
		DueDate:          inv.DueDate.Format("2006-01-02"),
		DeliveryDate:     inv.DeliveryDate.Format("2006-01-02"),
		VariableSymbol:   inv.VariableSymbol,
		ConstantSymbol:   inv.ConstantSymbol,
		CurrencyCode:     inv.CurrencyCode,
		ExchangeRate:     int64(inv.ExchangeRate),
		PaymentMethod:    inv.PaymentMethod,
		BankAccount:      inv.BankAccount,
		BankCode:         inv.BankCode,
		IBAN:             inv.IBAN,
		SWIFT:            inv.SWIFT,
		SubtotalAmount:   int64(inv.SubtotalAmount),
		VATAmount:        int64(inv.VATAmount),
		TotalAmount:      int64(inv.TotalAmount),
		PaidAmount:       int64(inv.PaidAmount),
		Notes:            inv.Notes,
		InternalNotes:    inv.InternalNotes,
		RelatedInvoiceID: inv.RelatedInvoiceID,
		RelationType:     inv.RelationType,
		CreatedAt:        inv.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        inv.UpdatedAt.Format(time.RFC3339),
	}

	if inv.SentAt != nil {
		s := inv.SentAt.Format(time.RFC3339)
		resp.SentAt = &s
	}
	if inv.PaidAt != nil {
		s := inv.PaidAt.Format(time.RFC3339)
		resp.PaidAt = &s
	}

	if inv.Customer != nil {
		c := contactFromDomain(inv.Customer)
		resp.Customer = &c
	}

	for _, item := range inv.Items {
		resp.Items = append(resp.Items, invoiceItemResponse{
			ID:             item.ID,
			InvoiceID:      item.InvoiceID,
			Description:    item.Description,
			Quantity:       int64(item.Quantity),
			Unit:           item.Unit,
			UnitPrice:      int64(item.UnitPrice),
			VATRatePercent: item.VATRatePercent,
			VATAmount:      int64(item.VATAmount),
			TotalAmount:    int64(item.TotalAmount),
			SortOrder:      item.SortOrder,
		})
	}

	return resp
}

// --- Invoice Action DTOs ---

// markPaidRequest is the JSON request body for marking an invoice as paid.
type markPaidRequest struct {
	Amount int64  `json:"amount"`  // in halere
	PaidAt string `json:"paid_at"` // YYYY-MM-DD or RFC3339
}

// --- Expense DTOs ---

// expenseItemRequest is the JSON request body for an expense line item.
type expenseItemRequest struct {
	Description    string `json:"description"`
	Quantity       int64  `json:"quantity"` // in smallest unit (cents), e.g. 250 = 2.50
	Unit           string `json:"unit"`
	UnitPrice      int64  `json:"unit_price"` // in halere
	VATRatePercent int    `json:"vat_rate_percent"`
	SortOrder      int    `json:"sort_order"`
}

// expenseItemResponse is the JSON response for an expense line item.
type expenseItemResponse struct {
	ID             int64  `json:"id"`
	ExpenseID      int64  `json:"expense_id"`
	Description    string `json:"description"`
	Quantity       int64  `json:"quantity"`
	Unit           string `json:"unit"`
	UnitPrice      int64  `json:"unit_price"`
	VATRatePercent int    `json:"vat_rate_percent"`
	VATAmount      int64  `json:"vat_amount"`
	TotalAmount    int64  `json:"total_amount"`
	SortOrder      int    `json:"sort_order"`
}

// expenseRequest is the JSON request body for creating/updating an expense.
type expenseRequest struct {
	VendorID        *int64               `json:"vendor_id"`
	ExpenseNumber   string               `json:"expense_number"`
	Category        string               `json:"category"`
	Description     string               `json:"description"`
	IssueDate       string               `json:"issue_date"`
	Amount          int64                `json:"amount"`
	CurrencyCode    string               `json:"currency_code"`
	ExchangeRate    int64                `json:"exchange_rate"`
	VATRatePercent  int                  `json:"vat_rate_percent"`
	VATAmount       int64                `json:"vat_amount"`
	IsTaxDeductible bool                 `json:"is_tax_deductible"`
	BusinessPercent int                  `json:"business_percent"`
	PaymentMethod   string               `json:"payment_method"`
	DocumentPath    string               `json:"document_path"`
	Notes           string               `json:"notes"`
	Items           []expenseItemRequest `json:"items"`
}

// toDomain converts an expenseRequest to a domain.Expense.
func (r *expenseRequest) toDomain() (*domain.Expense, error) {
	if r.IssueDate == "" {
		return nil, fmt.Errorf("issue_date is required: %w", domain.ErrInvalidInput)
	}

	exp := &domain.Expense{
		VendorID:        r.VendorID,
		ExpenseNumber:   r.ExpenseNumber,
		Category:        r.Category,
		Description:     r.Description,
		Amount:          domain.Amount(r.Amount),
		CurrencyCode:    r.CurrencyCode,
		ExchangeRate:    domain.Amount(r.ExchangeRate),
		VATRatePercent:  r.VATRatePercent,
		VATAmount:       domain.Amount(r.VATAmount),
		IsTaxDeductible: r.IsTaxDeductible,
		BusinessPercent: r.BusinessPercent,
		PaymentMethod:   r.PaymentMethod,
		DocumentPath:    r.DocumentPath,
		Notes:           r.Notes,
	}

	issueDate, err := time.Parse("2006-01-02", r.IssueDate)
	if err != nil {
		return nil, fmt.Errorf("invalid issue_date format, expected YYYY-MM-DD: %w", domain.ErrInvalidInput)
	}
	exp.IssueDate = issueDate

	for _, item := range r.Items {
		exp.Items = append(exp.Items, domain.ExpenseItem{
			Description:    item.Description,
			Quantity:       domain.Amount(item.Quantity),
			Unit:           item.Unit,
			UnitPrice:      domain.Amount(item.UnitPrice),
			VATRatePercent: item.VATRatePercent,
			SortOrder:      item.SortOrder,
		})
	}

	return exp, nil
}

// expenseResponse is the JSON response for an expense.
type expenseResponse struct {
	ID              int64                 `json:"id"`
	VendorID        *int64                `json:"vendor_id,omitempty"`
	Vendor          *contactResponse      `json:"vendor,omitempty"`
	ExpenseNumber   string                `json:"expense_number"`
	Category        string                `json:"category"`
	Description     string                `json:"description"`
	IssueDate       string                `json:"issue_date"`
	Amount          int64                 `json:"amount"`
	CurrencyCode    string                `json:"currency_code"`
	ExchangeRate    int64                 `json:"exchange_rate"`
	VATRatePercent  int                   `json:"vat_rate_percent"`
	VATAmount       int64                 `json:"vat_amount"`
	IsTaxDeductible bool                  `json:"is_tax_deductible"`
	BusinessPercent int                   `json:"business_percent"`
	PaymentMethod   string                `json:"payment_method"`
	DocumentPath    string                `json:"document_path,omitempty"`
	Notes           string                `json:"notes"`
	TaxReviewedAt   *string               `json:"tax_reviewed_at,omitempty"`
	Items           []expenseItemResponse `json:"items"`
	CreatedAt       string                `json:"created_at"`
	UpdatedAt       string                `json:"updated_at"`
}

// expenseFromDomain converts a domain.Expense to an expenseResponse.
func expenseFromDomain(e *domain.Expense) expenseResponse {
	resp := expenseResponse{
		ID:              e.ID,
		VendorID:        e.VendorID,
		ExpenseNumber:   e.ExpenseNumber,
		Category:        e.Category,
		Description:     e.Description,
		IssueDate:       e.IssueDate.Format("2006-01-02"),
		Amount:          int64(e.Amount),
		CurrencyCode:    e.CurrencyCode,
		ExchangeRate:    int64(e.ExchangeRate),
		VATRatePercent:  e.VATRatePercent,
		VATAmount:       int64(e.VATAmount),
		IsTaxDeductible: e.IsTaxDeductible,
		BusinessPercent: e.BusinessPercent,
		PaymentMethod:   e.PaymentMethod,
		DocumentPath:    e.DocumentPath,
		Notes:           e.Notes,
		TaxReviewedAt:   formatOptionalTime(e.TaxReviewedAt),
		CreatedAt:       e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       e.UpdatedAt.Format(time.RFC3339),
	}
	if e.Vendor != nil {
		v := contactFromDomain(e.Vendor)
		resp.Vendor = &v
	}
	for _, item := range e.Items {
		resp.Items = append(resp.Items, expenseItemResponse{
			ID:             item.ID,
			ExpenseID:      item.ExpenseID,
			Description:    item.Description,
			Quantity:       int64(item.Quantity),
			Unit:           item.Unit,
			UnitPrice:      int64(item.UnitPrice),
			VATRatePercent: item.VATRatePercent,
			VATAmount:      int64(item.VATAmount),
			TotalAmount:    int64(item.TotalAmount),
			SortOrder:      item.SortOrder,
		})
	}
	return resp
}

// --- Tax Period DTO ---

// taxPeriodResponse is the JSON response for a tax period.
type taxPeriodResponse struct {
	Year    int `json:"year"`
	Month   int `json:"month"`
	Quarter int `json:"quarter"`
}

// --- VAT Return DTOs ---

// vatReturnRequest is the JSON request body for creating a VAT return.
type vatReturnRequest struct {
	Year       int    `json:"year"`
	Month      int    `json:"month"`
	Quarter    int    `json:"quarter"`
	FilingType string `json:"filing_type"`
}

// vatReturnResponse is the JSON response for a VAT return.
type vatReturnResponse struct {
	ID         int64             `json:"id"`
	Period     taxPeriodResponse `json:"period"`
	FilingType string            `json:"filing_type"`

	OutputVATBase21   int64 `json:"output_vat_base_21"`
	OutputVATAmount21 int64 `json:"output_vat_amount_21"`
	OutputVATBase12   int64 `json:"output_vat_base_12"`
	OutputVATAmount12 int64 `json:"output_vat_amount_12"`
	OutputVATBase0    int64 `json:"output_vat_base_0"`

	ReverseChargeBase21   int64 `json:"reverse_charge_base_21"`
	ReverseChargeAmount21 int64 `json:"reverse_charge_amount_21"`
	ReverseChargeBase12   int64 `json:"reverse_charge_base_12"`
	ReverseChargeAmount12 int64 `json:"reverse_charge_amount_12"`

	InputVATBase21   int64 `json:"input_vat_base_21"`
	InputVATAmount21 int64 `json:"input_vat_amount_21"`
	InputVATBase12   int64 `json:"input_vat_base_12"`
	InputVATAmount12 int64 `json:"input_vat_amount_12"`

	TotalOutputVAT int64 `json:"total_output_vat"`
	TotalInputVAT  int64 `json:"total_input_vat"`
	NetVAT         int64 `json:"net_vat"`

	HasXML    bool    `json:"has_xml"`
	Status    string  `json:"status"`
	FiledAt   *string `json:"filed_at,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// vatReturnFromDomain converts a domain.VATReturn to a vatReturnResponse.
func vatReturnFromDomain(vr *domain.VATReturn) vatReturnResponse {
	return vatReturnResponse{
		ID: vr.ID,
		Period: taxPeriodResponse{
			Year:    vr.Period.Year,
			Month:   vr.Period.Month,
			Quarter: vr.Period.Quarter,
		},
		FilingType: vr.FilingType,

		OutputVATBase21:   int64(vr.OutputVATBase21),
		OutputVATAmount21: int64(vr.OutputVATAmount21),
		OutputVATBase12:   int64(vr.OutputVATBase12),
		OutputVATAmount12: int64(vr.OutputVATAmount12),
		OutputVATBase0:    int64(vr.OutputVATBase0),

		ReverseChargeBase21:   int64(vr.ReverseChargeBase21),
		ReverseChargeAmount21: int64(vr.ReverseChargeAmount21),
		ReverseChargeBase12:   int64(vr.ReverseChargeBase12),
		ReverseChargeAmount12: int64(vr.ReverseChargeAmount12),

		InputVATBase21:   int64(vr.InputVATBase21),
		InputVATAmount21: int64(vr.InputVATAmount21),
		InputVATBase12:   int64(vr.InputVATBase12),
		InputVATAmount12: int64(vr.InputVATAmount12),

		TotalOutputVAT: int64(vr.TotalOutputVAT),
		TotalInputVAT:  int64(vr.TotalInputVAT),
		NetVAT:         int64(vr.NetVAT),

		HasXML:    len(vr.XMLData) > 0,
		Status:    vr.Status,
		FiledAt:   formatOptionalTime(vr.FiledAt),
		CreatedAt: vr.CreatedAt.Format(time.RFC3339),
		UpdatedAt: vr.UpdatedAt.Format(time.RFC3339),
	}
}

// --- VAT Control Statement DTOs ---

// controlStatementRequest is the JSON request body for creating a control statement.
type controlStatementRequest struct {
	Year       int    `json:"year"`
	Month      int    `json:"month"`
	FilingType string `json:"filing_type"`
}

// controlStatementLineResponse is the JSON response for a control statement line.
type controlStatementLineResponse struct {
	ID             int64  `json:"id"`
	Section        string `json:"section"`
	PartnerDIC     string `json:"partner_dic"`
	DocumentNumber string `json:"document_number"`
	DPPD           string `json:"dppd"`
	Base           int64  `json:"base"`
	VAT            int64  `json:"vat"`
	VATRatePercent int    `json:"vat_rate_percent"`
	InvoiceID      *int64 `json:"invoice_id,omitempty"`
	ExpenseID      *int64 `json:"expense_id,omitempty"`
}

// controlStatementResponse is the JSON response for a control statement.
type controlStatementResponse struct {
	ID         int64                          `json:"id"`
	Period     taxPeriodResponse              `json:"period"`
	FilingType string                         `json:"filing_type"`
	Lines      []controlStatementLineResponse `json:"lines,omitempty"`
	HasXML     bool                           `json:"has_xml"`
	Status     string                         `json:"status"`
	FiledAt    *string                        `json:"filed_at,omitempty"`
	CreatedAt  string                         `json:"created_at"`
	UpdatedAt  string                         `json:"updated_at"`
}

// controlStatementFromDomain converts a domain.VATControlStatement to a controlStatementResponse.
func controlStatementFromDomain(cs *domain.VATControlStatement, lines []domain.VATControlStatementLine) controlStatementResponse {
	resp := controlStatementResponse{
		ID: cs.ID,
		Period: taxPeriodResponse{
			Year:    cs.Period.Year,
			Month:   cs.Period.Month,
			Quarter: cs.Period.Quarter,
		},
		FilingType: cs.FilingType,
		HasXML:     len(cs.XMLData) > 0,
		Status:     cs.Status,
		FiledAt:    formatOptionalTime(cs.FiledAt),
		CreatedAt:  cs.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  cs.UpdatedAt.Format(time.RFC3339),
	}
	for _, l := range lines {
		resp.Lines = append(resp.Lines, controlStatementLineResponse{
			ID:             l.ID,
			Section:        l.Section,
			PartnerDIC:     l.PartnerDIC,
			DocumentNumber: l.DocumentNumber,
			DPPD:           l.DPPD,
			Base:           int64(l.Base),
			VAT:            int64(l.VAT),
			VATRatePercent: l.VATRatePercent,
			InvoiceID:      l.InvoiceID,
			ExpenseID:      l.ExpenseID,
		})
	}
	return resp
}

// --- VIES Summary DTOs ---

// viesSummaryRequest is the JSON request body for creating a VIES summary.
type viesSummaryRequest struct {
	Year       int    `json:"year"`
	Quarter    int    `json:"quarter"`
	FilingType string `json:"filing_type"`
}

// viesSummaryLineResponse is the JSON response for a VIES summary line.
type viesSummaryLineResponse struct {
	ID          int64  `json:"id"`
	PartnerDIC  string `json:"partner_dic"`
	CountryCode string `json:"country_code"`
	TotalAmount int64  `json:"total_amount"`
	ServiceCode string `json:"service_code"`
}

// viesSummaryResponse is the JSON response for a VIES summary.
type viesSummaryResponse struct {
	ID         int64                     `json:"id"`
	Period     taxPeriodResponse         `json:"period"`
	FilingType string                    `json:"filing_type"`
	Lines      []viesSummaryLineResponse `json:"lines,omitempty"`
	HasXML     bool                      `json:"has_xml"`
	Status     string                    `json:"status"`
	FiledAt    *string                   `json:"filed_at,omitempty"`
	CreatedAt  string                    `json:"created_at"`
	UpdatedAt  string                    `json:"updated_at"`
}

// viesSummaryFromDomain converts a domain.VIESSummary to a viesSummaryResponse.
func viesSummaryFromDomain(vs *domain.VIESSummary, lines []domain.VIESSummaryLine) viesSummaryResponse {
	resp := viesSummaryResponse{
		ID: vs.ID,
		Period: taxPeriodResponse{
			Year:    vs.Period.Year,
			Month:   vs.Period.Month,
			Quarter: vs.Period.Quarter,
		},
		FilingType: vs.FilingType,
		HasXML:     len(vs.XMLData) > 0,
		Status:     vs.Status,
		FiledAt:    formatOptionalTime(vs.FiledAt),
		CreatedAt:  vs.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  vs.UpdatedAt.Format(time.RFC3339),
	}
	for _, l := range lines {
		resp.Lines = append(resp.Lines, viesSummaryLineResponse{
			ID:          l.ID,
			PartnerDIC:  l.PartnerDIC,
			CountryCode: l.CountryCode,
			TotalAmount: int64(l.TotalAmount),
			ServiceCode: l.ServiceCode,
		})
	}
	return resp
}
