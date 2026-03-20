use std::sync::Arc;

use zfaktury_domain::{DomainError, InvoiceSequence};

use super::audit_svc::AuditService;
use crate::repository::traits::InvoiceSequenceRepo;

/// Service for invoice numbering sequence management.
pub struct SequenceService {
    repo: Arc<dyn InvoiceSequenceRepo + Send + Sync>,
    audit: Option<Arc<AuditService>>,
}

impl SequenceService {
    pub fn new(
        repo: Arc<dyn InvoiceSequenceRepo + Send + Sync>,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self { repo, audit }
    }

    /// Creates a new invoice sequence. Validates prefix+year uniqueness.
    pub fn create(&self, seq: &mut InvoiceSequence) -> Result<(), DomainError> {
        if seq.prefix.is_empty() {
            return Err(DomainError::InvalidInput);
        }
        if seq.year == 0 {
            return Err(DomainError::InvalidInput);
        }
        if seq.next_number <= 0 {
            seq.next_number = 1;
        }
        if seq.format_pattern.is_empty() {
            seq.format_pattern = "{prefix}{year}{number:04d}".to_string();
        }

        // Check uniqueness of prefix+year.
        match self.repo.get_by_prefix_and_year(&seq.prefix, seq.year) {
            Ok(_) => return Err(DomainError::DuplicateNumber),
            Err(DomainError::NotFound) => {} // expected
            Err(e) => return Err(e),
        }

        self.repo.create(seq)?;
        if let Some(ref audit) = self.audit {
            audit.log("sequence", seq.id, "create", None, None);
        }
        Ok(())
    }

    /// Updates an existing sequence. Prevents lowering next_number below used numbers.
    pub fn update(&self, seq: &mut InvoiceSequence) -> Result<(), DomainError> {
        if seq.id == 0 || seq.prefix.is_empty() || seq.year == 0 || seq.next_number <= 0 {
            return Err(DomainError::InvalidInput);
        }

        self.repo.get_by_id(seq.id)?; // verify exists

        let max_used = self.repo.max_used_number(seq.id)?;
        if seq.next_number as i64 <= max_used {
            return Err(DomainError::InvalidInput);
        }

        // Check uniqueness of prefix+year (excluding self).
        if let Ok(dup) = self.repo.get_by_prefix_and_year(&seq.prefix, seq.year)
            && dup.id != seq.id
        {
            return Err(DomainError::DuplicateNumber);
        }

        self.repo.update(seq)?;
        if let Some(ref audit) = self.audit {
            audit.log("sequence", seq.id, "update", None, None);
        }
        Ok(())
    }

    /// Deletes a sequence if no invoices reference it.
    pub fn delete(&self, id: i64) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }

        let count = self.repo.count_invoices_by_sequence_id(id)?;
        if count > 0 {
            return Err(DomainError::InvalidInput);
        }

        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("sequence", id, "delete", None, None);
        }
        Ok(())
    }

    pub fn get_by_id(&self, id: i64) -> Result<InvoiceSequence, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)
    }

    pub fn list(&self) -> Result<Vec<InvoiceSequence>, DomainError> {
        self.repo.list()
    }

    /// Retrieves or creates a sequence for the given prefix and year.
    pub fn get_or_create_for_year(
        &self,
        prefix: &str,
        year: i32,
    ) -> Result<InvoiceSequence, DomainError> {
        if prefix.is_empty() || year == 0 {
            return Err(DomainError::InvalidInput);
        }

        match self.repo.get_by_prefix_and_year(prefix, year) {
            Ok(seq) => return Ok(seq),
            Err(DomainError::NotFound) => {}
            Err(e) => return Err(e),
        }

        let mut new_seq = InvoiceSequence {
            id: 0,
            prefix: prefix.to_string(),
            next_number: 1,
            year,
            format_pattern: "{prefix}{year}{number:04d}".to_string(),
        };

        match self.repo.create(&mut new_seq) {
            Ok(()) => Ok(new_seq),
            Err(_) => {
                // Race condition: retry lookup.
                self.repo.get_by_prefix_and_year(prefix, year)
            }
        }
    }

    /// Returns a preview of the next formatted invoice number.
    pub fn format_preview(seq: &InvoiceSequence) -> String {
        format!("{}{}{:04}", seq.prefix, seq.year, seq.next_number)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Mutex;

    struct MockSeqRepo {
        seqs: Mutex<Vec<InvoiceSequence>>,
        next_id: Mutex<i64>,
    }

    impl MockSeqRepo {
        fn new() -> Self {
            Self {
                seqs: Mutex::new(Vec::new()),
                next_id: Mutex::new(1),
            }
        }
    }

    impl InvoiceSequenceRepo for MockSeqRepo {
        fn create(&self, seq: &mut InvoiceSequence) -> Result<(), DomainError> {
            let mut id = self.next_id.lock().unwrap();
            seq.id = *id;
            *id += 1;
            self.seqs.lock().unwrap().push(seq.clone());
            Ok(())
        }

        fn update(&self, seq: &mut InvoiceSequence) -> Result<(), DomainError> {
            let mut seqs = self.seqs.lock().unwrap();
            if let Some(s) = seqs.iter_mut().find(|s| s.id == seq.id) {
                *s = seq.clone();
                Ok(())
            } else {
                Err(DomainError::NotFound)
            }
        }

        fn delete(&self, id: i64) -> Result<(), DomainError> {
            let mut seqs = self.seqs.lock().unwrap();
            seqs.retain(|s| s.id != id);
            Ok(())
        }

        fn get_by_id(&self, id: i64) -> Result<InvoiceSequence, DomainError> {
            self.seqs
                .lock()
                .unwrap()
                .iter()
                .find(|s| s.id == id)
                .cloned()
                .ok_or(DomainError::NotFound)
        }

        fn list(&self) -> Result<Vec<InvoiceSequence>, DomainError> {
            Ok(self.seqs.lock().unwrap().clone())
        }

        fn get_by_prefix_and_year(
            &self,
            prefix: &str,
            year: i32,
        ) -> Result<InvoiceSequence, DomainError> {
            self.seqs
                .lock()
                .unwrap()
                .iter()
                .find(|s| s.prefix == prefix && s.year == year)
                .cloned()
                .ok_or(DomainError::NotFound)
        }

        fn count_invoices_by_sequence_id(&self, _sequence_id: i64) -> Result<i64, DomainError> {
            Ok(0)
        }

        fn max_used_number(&self, _sequence_id: i64) -> Result<i64, DomainError> {
            Ok(0)
        }
    }

    #[test]
    fn test_create_sequence() {
        let repo = Arc::new(MockSeqRepo::new());
        let svc = SequenceService::new(repo, None);

        let mut seq = InvoiceSequence {
            id: 0,
            prefix: "FV".to_string(),
            next_number: 1,
            year: 2024,
            format_pattern: String::new(),
        };
        svc.create(&mut seq).unwrap();
        assert!(seq.id > 0);
        assert!(!seq.format_pattern.is_empty());
    }

    #[test]
    fn test_duplicate_prefix_year_rejected() {
        let repo = Arc::new(MockSeqRepo::new());
        let svc = SequenceService::new(repo, None);

        let mut s1 = InvoiceSequence {
            id: 0,
            prefix: "FV".to_string(),
            next_number: 1,
            year: 2024,
            format_pattern: String::new(),
        };
        svc.create(&mut s1).unwrap();

        let mut s2 = InvoiceSequence {
            id: 0,
            prefix: "FV".to_string(),
            next_number: 1,
            year: 2024,
            format_pattern: String::new(),
        };
        assert!(matches!(
            svc.create(&mut s2),
            Err(DomainError::DuplicateNumber)
        ));
    }

    #[test]
    fn test_get_or_create() {
        let repo = Arc::new(MockSeqRepo::new());
        let svc = SequenceService::new(repo, None);

        let seq1 = svc.get_or_create_for_year("FV", 2024).unwrap();
        let seq2 = svc.get_or_create_for_year("FV", 2024).unwrap();
        assert_eq!(seq1.id, seq2.id);
    }

    #[test]
    fn test_format_preview() {
        let seq = InvoiceSequence {
            id: 1,
            prefix: "FV".to_string(),
            next_number: 5,
            year: 2024,
            format_pattern: String::new(),
        };
        assert_eq!(SequenceService::format_preview(&seq), "FV20240005");
    }
}
