package domain

import "time"

// BankTransaction represents an imported bank transaction.
type BankTransaction struct {
	ID                  int64     `json:"id"`
	BankAccount         string    `json:"bank_account"`
	TransactionDate     time.Time `json:"transaction_date"`
	Amount              Amount    `json:"amount"`
	Currency            string    `json:"currency"`
	CounterpartyAccount string    `json:"counterparty_account"`
	CounterpartyName    string    `json:"counterparty_name"`
	VariableSymbol      string    `json:"variable_symbol"`
	ConstantSymbol      string    `json:"constant_symbol"`
	SpecificSymbol      string    `json:"specific_symbol"`
	Message             string    `json:"message"`
	InvoiceID           *int64    `json:"invoice_id,omitempty"`
	Matched             bool      `json:"matched"`
	CreatedAt           time.Time `json:"created_at"`
}
