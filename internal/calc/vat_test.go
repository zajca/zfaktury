package calc

import (
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

func TestCalculateVATReturn(t *testing.T) {
	tests := []struct {
		name     string
		invoices []VATInvoiceInput
		expenses []VATExpenseInput
		want     VATResult
	}{
		{
			name:     "zero invoices and zero expenses",
			invoices: nil,
			expenses: nil,
			want:     VATResult{},
		},
		{
			name: "single invoice 21% VAT",
			invoices: []VATInvoiceInput{
				{
					Type: "invoice",
					Items: []VATItemInput{
						{Base: domain.NewAmount(1000, 0), VATAmount: domain.NewAmount(210, 0), VATRatePercent: 21},
					},
				},
			},
			want: VATResult{
				OutputVATBase21:   domain.NewAmount(1000, 0),
				OutputVATAmount21: domain.NewAmount(210, 0),
				TotalOutputVAT:    domain.NewAmount(210, 0),
				NetVAT:            domain.NewAmount(210, 0),
			},
		},
		{
			name: "single invoice 12% VAT",
			invoices: []VATInvoiceInput{
				{
					Type: "invoice",
					Items: []VATItemInput{
						{Base: domain.NewAmount(500, 0), VATAmount: domain.NewAmount(60, 0), VATRatePercent: 12},
					},
				},
			},
			want: VATResult{
				OutputVATBase12:   domain.NewAmount(500, 0),
				OutputVATAmount12: domain.NewAmount(60, 0),
				TotalOutputVAT:    domain.NewAmount(60, 0),
				NetVAT:            domain.NewAmount(60, 0),
			},
		},
		{
			name: "multiple invoices mixed rates 21% and 12%",
			invoices: []VATInvoiceInput{
				{
					Type: "invoice",
					Items: []VATItemInput{
						{Base: domain.NewAmount(1000, 0), VATAmount: domain.NewAmount(210, 0), VATRatePercent: 21},
					},
				},
				{
					Type: "invoice",
					Items: []VATItemInput{
						{Base: domain.NewAmount(500, 0), VATAmount: domain.NewAmount(60, 0), VATRatePercent: 12},
					},
				},
			},
			want: VATResult{
				OutputVATBase21:   domain.NewAmount(1000, 0),
				OutputVATAmount21: domain.NewAmount(210, 0),
				OutputVATBase12:   domain.NewAmount(500, 0),
				OutputVATAmount12: domain.NewAmount(60, 0),
				TotalOutputVAT:    domain.NewAmount(270, 0),
				NetVAT:            domain.NewAmount(270, 0),
			},
		},
		{
			name: "credit note reverses sign",
			invoices: []VATInvoiceInput{
				{
					Type: "credit_note",
					Items: []VATItemInput{
						{Base: domain.NewAmount(200, 0), VATAmount: domain.NewAmount(42, 0), VATRatePercent: 21},
					},
				},
			},
			want: VATResult{
				OutputVATBase21:   -domain.NewAmount(200, 0),
				OutputVATAmount21: -domain.NewAmount(42, 0),
				TotalOutputVAT:    -domain.NewAmount(42, 0),
				NetVAT:            -domain.NewAmount(42, 0),
			},
		},
		{
			name: "invoice with 0% VAT rate excluded",
			invoices: []VATInvoiceInput{
				{
					Type: "invoice",
					Items: []VATItemInput{
						{Base: domain.NewAmount(1000, 0), VATAmount: 0, VATRatePercent: 0},
					},
				},
			},
			want: VATResult{},
		},
		{
			name: "single expense 21% rate business percent 100",
			expenses: []VATExpenseInput{
				{
					Amount:          domain.NewAmount(1210, 0),
					VATAmount:       domain.NewAmount(210, 0),
					VATRatePercent:  21,
					BusinessPercent: 100,
				},
			},
			want: VATResult{
				InputVATBase21:   domain.NewAmount(1000, 0),
				InputVATAmount21: domain.NewAmount(210, 0),
				TotalInputVAT:    domain.NewAmount(210, 0),
				NetVAT:           -domain.NewAmount(210, 0),
			},
		},
		{
			name: "single expense 12% rate business percent 50",
			expenses: []VATExpenseInput{
				{
					Amount:          domain.NewAmount(560, 0),
					VATAmount:       domain.NewAmount(60, 0),
					VATRatePercent:  12,
					BusinessPercent: 50,
				},
			},
			want: VATResult{
				InputVATBase12:   domain.NewAmount(250, 0),
				InputVATAmount12: domain.NewAmount(30, 0),
				TotalInputVAT:    domain.NewAmount(30, 0),
				NetVAT:           -domain.NewAmount(30, 0),
			},
		},
		{
			name: "expense with business percent 0 treated as 100",
			expenses: []VATExpenseInput{
				{
					Amount:          domain.NewAmount(1210, 0),
					VATAmount:       domain.NewAmount(210, 0),
					VATRatePercent:  21,
					BusinessPercent: 0,
				},
			},
			want: VATResult{
				InputVATBase21:   domain.NewAmount(1000, 0),
				InputVATAmount21: domain.NewAmount(210, 0),
				TotalInputVAT:    domain.NewAmount(210, 0),
				NetVAT:           -domain.NewAmount(210, 0),
			},
		},
		{
			name: "mixed invoices and expenses",
			invoices: []VATInvoiceInput{
				{
					Type: "invoice",
					Items: []VATItemInput{
						{Base: domain.NewAmount(1000, 0), VATAmount: domain.NewAmount(210, 0), VATRatePercent: 21},
						{Base: domain.NewAmount(500, 0), VATAmount: domain.NewAmount(60, 0), VATRatePercent: 12},
					},
				},
			},
			expenses: []VATExpenseInput{
				{
					Amount:          domain.NewAmount(605, 0),
					VATAmount:       domain.NewAmount(105, 0),
					VATRatePercent:  21,
					BusinessPercent: 100,
				},
			},
			want: VATResult{
				OutputVATBase21:   domain.NewAmount(1000, 0),
				OutputVATAmount21: domain.NewAmount(210, 0),
				OutputVATBase12:   domain.NewAmount(500, 0),
				OutputVATAmount12: domain.NewAmount(60, 0),
				InputVATBase21:    domain.NewAmount(500, 0),
				InputVATAmount21:  domain.NewAmount(105, 0),
				TotalOutputVAT:    domain.NewAmount(270, 0),
				TotalInputVAT:     domain.NewAmount(105, 0),
				NetVAT:            domain.NewAmount(165, 0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateVATReturn(tt.invoices, tt.expenses)

			if got != tt.want {
				t.Errorf("CalculateVATReturn() =\n  %+v\nwant\n  %+v", got, tt.want)
			}

			// Verify derived field invariants
			if got.TotalOutputVAT != got.OutputVATAmount21+got.OutputVATAmount12 {
				t.Errorf("TotalOutputVAT (%d) != OutputVATAmount21 (%d) + OutputVATAmount12 (%d)",
					got.TotalOutputVAT, got.OutputVATAmount21, got.OutputVATAmount12)
			}
			if got.TotalInputVAT != got.InputVATAmount21+got.InputVATAmount12 {
				t.Errorf("TotalInputVAT (%d) != InputVATAmount21 (%d) + InputVATAmount12 (%d)",
					got.TotalInputVAT, got.InputVATAmount21, got.InputVATAmount12)
			}
			if got.NetVAT != got.TotalOutputVAT-got.TotalInputVAT {
				t.Errorf("NetVAT (%d) != TotalOutputVAT (%d) - TotalInputVAT (%d)",
					got.NetVAT, got.TotalOutputVAT, got.TotalInputVAT)
			}
		})
	}
}
