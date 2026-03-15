package handler

import (
	"errors"
	"net/http"

	"github.com/zajca/zfaktury/internal/domain"
)

// mapDomainError translates domain errors to HTTP status codes.
func mapDomainError(w http.ResponseWriter, err error) {
	switch {
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
