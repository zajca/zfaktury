use zfaktury_domain::DomainError;

/// Database-level error type.
#[derive(Debug, thiserror::Error)]
pub enum DbError {
    #[error("sqlite error: {0}")]
    Sqlite(#[from] rusqlite::Error),

    #[error("migration error: {0}")]
    Migration(String),

    #[error("date parse error: {0}")]
    DateParse(String),

    #[error("domain error: {0}")]
    Domain(#[from] DomainError),
}

impl From<DbError> for DomainError {
    fn from(err: DbError) -> Self {
        match err {
            DbError::Domain(e) => e,
            DbError::Sqlite(ref e) => {
                if let rusqlite::Error::QueryReturnedNoRows = e {
                    DomainError::NotFound
                } else {
                    // Log the underlying error for debugging; map to a generic error.
                    log::error!("database error: {err}");
                    DomainError::InvalidInput
                }
            }
            _ => {
                log::error!("database error: {err}");
                DomainError::InvalidInput
            }
        }
    }
}
