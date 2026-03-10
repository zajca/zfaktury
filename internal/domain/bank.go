package domain

import "time"

// BankTransaction represents an imported bank transaction.
type BankTransaction struct {
	ID                  int64
	BankAccount         string
	TransactionDate     time.Time
	Amount              Amount
	Currency            string
	CounterpartyAccount string
	CounterpartyName    string
	VariableSymbol      string
	ConstantSymbol      string
	SpecificSymbol      string
	Message             string
	InvoiceID           *int64
	Matched             bool
	CreatedAt           time.Time
}
