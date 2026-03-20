use chrono::{NaiveDate, NaiveDateTime};

use crate::error::DbError;

const DATE_FORMAT: &str = "%Y-%m-%d";
const DATETIME_FORMAT: &str = "%Y-%m-%dT%H:%M:%SZ";
// SQLite sometimes produces fractional seconds
const DATETIME_FORMAT_FRAC: &str = "%Y-%m-%dT%H:%M:%S%.fZ";

/// Parse a date string (YYYY-MM-DD) into NaiveDate.
pub fn parse_date(value: &str) -> Result<NaiveDate, DbError> {
    NaiveDate::parse_from_str(value, DATE_FORMAT)
        .map_err(|e| DbError::DateParse(format!("parsing date '{value}': {e}")))
}

/// Parse a datetime string (RFC3339-like) into NaiveDateTime.
pub fn parse_datetime(value: &str) -> Result<NaiveDateTime, DbError> {
    NaiveDateTime::parse_from_str(value, DATETIME_FORMAT)
        .or_else(|_| NaiveDateTime::parse_from_str(value, DATETIME_FORMAT_FRAC))
        .or_else(|_| NaiveDateTime::parse_from_str(value, "%Y-%m-%d %H:%M:%S"))
        .map_err(|e| DbError::DateParse(format!("parsing datetime '{value}': {e}")))
}

/// Parse an optional date string. Returns None if value is empty or None-equivalent.
pub fn parse_date_optional(value: Option<&str>) -> Result<Option<NaiveDate>, DbError> {
    match value {
        Some(v) if !v.is_empty() => Ok(Some(parse_date(v)?)),
        _ => Ok(None),
    }
}

/// Parse an optional datetime string. Returns None if value is empty or None-equivalent.
pub fn parse_datetime_optional(value: Option<&str>) -> Result<Option<NaiveDateTime>, DbError> {
    match value {
        Some(v) if !v.is_empty() => Ok(Some(parse_datetime(v)?)),
        _ => Ok(None),
    }
}

/// Format a NaiveDate as YYYY-MM-DD for SQLite storage.
pub fn format_date(date: &NaiveDate) -> String {
    date.format(DATE_FORMAT).to_string()
}

/// Format a NaiveDateTime as RFC3339-like for SQLite storage.
pub fn format_datetime(dt: &NaiveDateTime) -> String {
    dt.format(DATETIME_FORMAT).to_string()
}

/// Parse a date string with a lenient fallback: logs a warning and returns the default
/// value when parsing fails. Use this instead of `parse_date(...).unwrap_or_default()`.
pub fn parse_date_or_default(value: &str) -> NaiveDate {
    parse_date(value).unwrap_or_else(|e| {
        log::warn!("Failed to parse date '{}': {}", value, e);
        NaiveDate::default()
    })
}

/// Parse a datetime string with a lenient fallback: logs a warning and returns the default
/// value when parsing fails. Use this instead of `parse_datetime(...).unwrap_or_default()`.
pub fn parse_datetime_or_default(value: &str) -> NaiveDateTime {
    parse_datetime(value).unwrap_or_else(|e| {
        log::warn!("Failed to parse datetime '{}': {}", value, e);
        NaiveDateTime::default()
    })
}

/// Format an optional NaiveDateTime. Returns None if the value is None.
pub fn format_datetime_opt(dt: &Option<NaiveDateTime>) -> Option<String> {
    dt.as_ref().map(format_datetime)
}
