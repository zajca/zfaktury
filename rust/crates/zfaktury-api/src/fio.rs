use chrono::NaiveDate;
use zfaktury_domain::BankTransaction;

use crate::error::Result;

/// Trait for bank transaction providers.
pub trait BankProvider: Send + Sync {
    /// Fetch transactions for the given date range.
    fn get_transactions(&self, from: NaiveDate, to: NaiveDate) -> Result<Vec<BankTransaction>>;
}

/// Client for the FIO Bank API.
///
/// This is a stub implementation; the actual API integration will be
/// implemented when the FIO Bank API specification is finalized.
pub struct FioClient {
    #[allow(dead_code)]
    api_token: String,
}

impl FioClient {
    /// Create a new FIO Bank client with the given API token.
    pub fn new(api_token: &str) -> Self {
        Self {
            api_token: api_token.to_string(),
        }
    }
}

impl BankProvider for FioClient {
    fn get_transactions(&self, _from: NaiveDate, _to: NaiveDate) -> Result<Vec<BankTransaction>> {
        // Stub: FIO Bank integration to be implemented.
        log::warn!("FIO Bank API not yet implemented, returning empty transaction list");
        Ok(Vec::new())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_fio_client_stub() {
        let client = FioClient::new("test-token");
        let from = NaiveDate::from_ymd_opt(2026, 1, 1).unwrap();
        let to = NaiveDate::from_ymd_opt(2026, 3, 31).unwrap();

        let result = client.get_transactions(from, to).unwrap();
        assert!(result.is_empty());
    }
}
