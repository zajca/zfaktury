package calc

import "github.com/zajca/zfaktury/internal/domain"

// VATInvoiceInput represents an invoice for VAT return calculation.
type VATInvoiceInput struct {
	Type  string // domain.InvoiceTypeCreditNote etc.
	Items []VATItemInput
}

// VATItemInput represents a single invoice item for VAT calculation.
type VATItemInput struct {
	Base           domain.Amount // pre-computed: quantity * unitPrice / 100
	VATAmount      domain.Amount
	VATRatePercent int
}

// VATExpenseInput represents an expense for VAT return calculation.
type VATExpenseInput struct {
	Amount          domain.Amount
	VATAmount       domain.Amount
	VATRatePercent  int
	BusinessPercent int // 0 means 100
}

// VATResult holds the calculated VAT return values.
type VATResult struct {
	OutputVATBase21   domain.Amount
	OutputVATAmount21 domain.Amount
	OutputVATBase12   domain.Amount
	OutputVATAmount12 domain.Amount
	InputVATBase21    domain.Amount
	InputVATAmount21  domain.Amount
	InputVATBase12    domain.Amount
	InputVATAmount12  domain.Amount
	TotalOutputVAT    domain.Amount
	TotalInputVAT     domain.Amount
	NetVAT            domain.Amount
}

// CalculateVATReturn computes VAT return totals from invoices and expenses.
func CalculateVATReturn(invoices []VATInvoiceInput, expenses []VATExpenseInput) VATResult {
	var r VATResult

	for _, inv := range invoices {
		sign := domain.Amount(1)
		if inv.Type == "credit_note" {
			sign = -1
		}

		for _, item := range inv.Items {
			base := item.Base * sign
			vat := item.VATAmount * sign

			switch item.VATRatePercent {
			case 21:
				r.OutputVATBase21 += base
				r.OutputVATAmount21 += vat
			case 12:
				r.OutputVATBase12 += base
				r.OutputVATAmount12 += vat
			}
		}
	}

	for _, exp := range expenses {
		businessPct := exp.BusinessPercent
		if businessPct == 0 {
			businessPct = 100
		}

		factor := float64(businessPct) / 100.0
		inputVAT := exp.VATAmount.Multiply(factor)
		inputBase := (exp.Amount - exp.VATAmount).Multiply(factor)

		switch exp.VATRatePercent {
		case 21:
			r.InputVATBase21 += inputBase
			r.InputVATAmount21 += inputVAT
		case 12:
			r.InputVATBase12 += inputBase
			r.InputVATAmount12 += inputVAT
		}
	}

	r.TotalOutputVAT = r.OutputVATAmount21 + r.OutputVATAmount12
	r.TotalInputVAT = r.InputVATAmount21 + r.InputVATAmount12
	r.NetVAT = r.TotalOutputVAT - r.TotalInputVAT

	return r
}
