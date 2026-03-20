use thiserror::Error;

/// Domain-level error variants mirroring the Go sentinel errors.
#[derive(Debug, Clone, PartialEq, Eq, Error)]
pub enum DomainError {
    #[error("not found")]
    NotFound,

    #[error("invalid input")]
    InvalidInput,

    #[error("invoice already paid")]
    PaidInvoice,

    #[error("no items")]
    NoItems,

    #[error("duplicate number")]
    DuplicateNumber,

    #[error("filing already exists for this period")]
    FilingAlreadyExists,

    #[error("filing already filed, cannot modify")]
    FilingAlreadyFiled,

    #[error("required setting not configured")]
    MissingSetting,

    #[error("invoice is not overdue")]
    InvoiceNotOverdue,

    #[error("customer has no email address")]
    NoCustomerEmail,
}
