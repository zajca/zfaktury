package fakturoid

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// FlexFloat64 is a float64 that can be unmarshaled from both JSON numbers and strings.
// Fakturoid API returns some numeric fields (e.g., exchange_rate) as strings.
type FlexFloat64 float64

// UnmarshalJSON implements json.Unmarshaler for FlexFloat64.
func (f *FlexFloat64) UnmarshalJSON(data []byte) error {
	// Try number first.
	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		*f = FlexFloat64(n)
		return nil
	}

	// Try string.
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("fakturoid: cannot parse %q as float64: %w", s, err)
		}
		*f = FlexFloat64(n)
		return nil
	}

	return fmt.Errorf("fakturoid: cannot unmarshal %s into float64", string(data))
}

// Subject represents a Fakturoid subject (contact).
type Subject struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	RegistrationNo string `json:"registration_no"` // ICO
	VatNo          string `json:"vat_no"`          // DIC
	Street         string `json:"street"`
	City           string `json:"city"`
	Zip            string `json:"zip"`
	Country        string `json:"country"`
	BankAccount    string `json:"bank_account"` // "cislo/kod" format
	IBAN           string `json:"iban"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	Web            string `json:"web"`
	Type           string `json:"type"` // "customer", "supplier", "both"
	Due            int    `json:"due"`  // payment terms days
}

// Attachment represents a file attachment on a Fakturoid entity.
type Attachment struct {
	ID          int64  `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	DownloadURL string `json:"download_url"`
}

// InvoiceLine represents a line item on a Fakturoid invoice.
type InvoiceLine struct {
	Name      string      `json:"name"`
	Quantity  FlexFloat64 `json:"quantity"`
	UnitName  string      `json:"unit_name"`
	UnitPrice FlexFloat64 `json:"unit_price"`
	VatRate   FlexFloat64 `json:"vat_rate"`
}

// Payment represents a payment on a Fakturoid invoice.
type Payment struct {
	PaidOn string `json:"paid_on"` // "YYYY-MM-DD"
}

// Invoice represents a Fakturoid invoice.
type Invoice struct {
	ID                    int64         `json:"id"`
	Number                string        `json:"number"`
	DocumentType          string        `json:"document_type"` // "invoice", "proforma", "partial_proforma", "correction", "tax_document", "final_invoice"
	Status                string        `json:"status"`        // "open", "sent", "overdue", "paid", "cancelled", "uncollectible"
	IssuedOn              string        `json:"issued_on"`     // "YYYY-MM-DD"
	DueOn                 string        `json:"due_on"`
	TaxableFulfillmentDue string        `json:"taxable_fulfillment_due"`
	VariableSymbol        string        `json:"variable_symbol"`
	SubjectID             int64         `json:"subject_id"`
	Currency              string        `json:"currency"`
	ExchangeRate          FlexFloat64   `json:"exchange_rate"`
	Subtotal              FlexFloat64   `json:"subtotal"`
	Total                 FlexFloat64   `json:"total"`
	Note                  string        `json:"note"`
	Lines                 []InvoiceLine `json:"lines"`
	Payments              []Payment     `json:"payments"`
	Attachments           []Attachment  `json:"attachments"`
}

// ExpenseLine represents a line item on a Fakturoid expense.
type ExpenseLine struct {
	Name      string      `json:"name"`
	Quantity  FlexFloat64 `json:"quantity"`
	UnitPrice FlexFloat64 `json:"unit_price"`
	VatRate   FlexFloat64 `json:"vat_rate"`
}

// Expense represents a Fakturoid expense.
type Expense struct {
	ID             int64         `json:"id"`
	OriginalNumber string        `json:"original_number"`
	IssuedOn       string        `json:"issued_on"`
	SubjectID      int64         `json:"subject_id"`
	Description    string        `json:"description"`
	Total          FlexFloat64   `json:"total"`
	Currency       string        `json:"currency"`
	ExchangeRate   FlexFloat64   `json:"exchange_rate"`
	PaymentMethod  string        `json:"payment_method"` // "bank", "cash", "card", etc.
	PrivateNote    string        `json:"private_note"`
	Lines          []ExpenseLine `json:"lines"`
	Attachments    []Attachment  `json:"attachments"`
}
