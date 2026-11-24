package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/taxrules"
)

// noRuleSetMessage is the user-facing (Czech) text for a year without tax rules.
func noRuleSetMessage(year int) string {
	return fmt.Sprintf("Pro rok %d zatím nejsou daňová pravidla.", year)
}

// mapDomainError translates domain errors to HTTP status codes.
func mapDomainError(w http.ResponseWriter, err error) {
	var noRules *taxrules.NoRuleSetError
	var userMessage *domain.UserMessageError
	switch {
	case errors.Is(err, domain.ErrMissingSetting) && errors.As(err, &userMessage):
		respondError(w, http.StatusUnprocessableEntity, userMessage.Message)
	case errors.As(err, &userMessage):
		respondError(w, http.StatusBadRequest, userMessage.Message)
	case errors.As(err, &noRules):
		respondError(w, http.StatusBadRequest, noRuleSetMessage(noRules.Year))
	case errors.Is(err, domain.ErrNotFound):
		respondError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrNoItems):
		respondError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrPaidInvoice):
		respondError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrDuplicateNumber):
		respondError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrFilingAlreadyExists):
		respondError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrFilingAlreadyFiled):
		respondError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrMissingSetting):
		respondError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrInvoiceNotOverdue):
		respondError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrNoCustomerEmail):
		respondError(w, http.StatusUnprocessableEntity, err.Error())
	default:
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}
