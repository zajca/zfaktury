package domain

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrInvalidInput    = errors.New("invalid input")
	ErrPaidInvoice     = errors.New("invoice already paid")
	ErrNoItems         = errors.New("no items")
	ErrDuplicateNumber = errors.New("duplicate number")
)
