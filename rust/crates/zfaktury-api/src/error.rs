use thiserror::Error;

/// API client errors.
#[derive(Debug, Error)]
pub enum ApiError {
    #[error("not found")]
    NotFound,

    #[error("invalid input: {0}")]
    InvalidInput(String),

    #[error("rate limited")]
    RateLimited,

    #[error("service timeout")]
    Timeout,

    #[error("API error: HTTP {status}: {body}")]
    HttpError { status: u16, body: String },

    #[error("request failed: {0}")]
    RequestFailed(#[from] reqwest::Error),

    #[error("JSON parse error: {0}")]
    JsonError(#[from] serde_json::Error),

    #[error("parse error: {0}")]
    ParseError(String),

    #[error("SMTP error: {0}")]
    SmtpError(String),

    #[error("OAuth error: {0}")]
    OAuthError(String),

    #[error("unsupported content type: {0}")]
    UnsupportedContentType(String),
}

/// Convenience type alias.
pub type Result<T> = std::result::Result<T, ApiError>;
