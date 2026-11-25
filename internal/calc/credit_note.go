package calc

import "github.com/zajca/zfaktury/internal/domain"

// CreditNoteAmount returns the contribution of a credit note as a negative
// amount, computed as -|a| so the stored sign does not matter.
//
// The app stores credit notes with negated unit prices, so their subtotal, item
// bases and item VAT are already negative. An imported credit note (Fakturoid
// "correction") may arrive with either sign. A credit note must always reduce
// the totals it belongs to, so callers must not subtract the stored amount
// again; they add CreditNoteAmount instead.
func CreditNoteAmount(a domain.Amount) domain.Amount {
	if a > 0 {
		return -a
	}
	return a
}

// creditNoteDocumentFactor returns the factor to apply to every base and VAT
// amount of one credit note: -1 when the credit note's item bases sum to a
// positive amount (a whole credit note stored with the opposite sign), +1
// otherwise.
//
// The factor is decided once per document from the sum of its item bases, not
// per line, so the relative signs of the lines (a discount line, for example)
// are preserved. Flipping each line on its own would break a credit note that
// contains mixed-sign lines and make it disagree with the control statement.
func creditNoteDocumentFactor(items []VATItemInput) domain.Amount {
	var sum domain.Amount
	for _, item := range items {
		sum += item.Base
	}
	if sum > 0 {
		return -1
	}
	return 1
}
