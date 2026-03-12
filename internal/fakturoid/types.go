package fakturoid

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

// InvoiceLine represents a line item on a Fakturoid invoice.
type InvoiceLine struct {
	Name      string  `json:"name"`
	Quantity  float64 `json:"quantity"`
	UnitName  string  `json:"unit_name"`
	UnitPrice float64 `json:"unit_price"`
	VatRate   float64 `json:"vat_rate"`
}

// Payment represents a payment on a Fakturoid invoice.
type Payment struct {
	PaidOn string `json:"paid_on"` // "YYYY-MM-DD"
}

// Invoice represents a Fakturoid invoice.
type Invoice struct {
	ID                    int64         `json:"id"`
	Number                string        `json:"number"`
	DocumentType          string        `json:"document_type"` // "invoice", "proforma", "credit_note"
	Status                string        `json:"status"`        // "open", "sent", "overdue", "paid", "cancelled"
	IssuedOn              string        `json:"issued_on"`     // "YYYY-MM-DD"
	DueOn                 string        `json:"due_on"`
	TaxableFulfillmentDue string        `json:"taxable_fulfillment_due"`
	VariableSymbol        string        `json:"variable_symbol"`
	SubjectID             int64         `json:"subject_id"`
	Currency              string        `json:"currency"`
	ExchangeRate          float64       `json:"exchange_rate"`
	Subtotal              float64       `json:"subtotal"`
	Total                 float64       `json:"total"`
	Note                  string        `json:"note"`
	Lines                 []InvoiceLine `json:"lines"`
	Payments              []Payment     `json:"payments"`
}

// ExpenseLine represents a line item on a Fakturoid expense.
type ExpenseLine struct {
	Name      string  `json:"name"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	VatRate   float64 `json:"vat_rate"`
}

// Expense represents a Fakturoid expense.
type Expense struct {
	ID             int64         `json:"id"`
	OriginalNumber string        `json:"original_number"`
	IssuedOn       string        `json:"issued_on"`
	SubjectID      int64         `json:"subject_id"`
	Description    string        `json:"description"`
	Total          float64       `json:"total"`
	Currency       string        `json:"currency"`
	ExchangeRate   float64       `json:"exchange_rate"`
	PaymentMethod  string        `json:"payment_method"` // "bank", "cash", "card", etc.
	PrivateNote    string        `json:"private_note"`
	Lines          []ExpenseLine `json:"lines"`
}
