use std::sync::Arc;

use zfaktury_domain::{AuditLogEntry, AuditLogFilter, DomainError};

use crate::repository::traits::AuditLogRepo;

/// Service for recording and querying audit log entries.
///
/// Audit failures are logged but never returned to callers so that audit
/// logging does not break main operations.
pub struct AuditService {
    repo: Arc<dyn AuditLogRepo + Send + Sync>,
}

impl AuditService {
    pub fn new(repo: Arc<dyn AuditLogRepo + Send + Sync>) -> Self {
        Self { repo }
    }

    /// Records an audit event. Errors are logged but never returned.
    pub fn log(
        &self,
        entity_type: &str,
        entity_id: i64,
        action: &str,
        old_values: Option<&str>,
        new_values: Option<&str>,
    ) {
        let mut entry = AuditLogEntry {
            id: 0,
            entity_type: entity_type.to_string(),
            entity_id,
            action: action.to_string(),
            old_values: old_values.unwrap_or_default().to_string(),
            new_values: new_values.unwrap_or_default().to_string(),
            created_at: chrono::Local::now().naive_local(),
        };

        if let Err(e) = self.repo.create(&mut entry) {
            log::error!(
                "creating audit log entry: {} entity_type={} entity_id={} action={}",
                e,
                entity_type,
                entity_id,
                action
            );
        }
    }

    /// Returns all audit log entries for a given entity.
    pub fn list_by_entity(
        &self,
        entity_type: &str,
        entity_id: i64,
    ) -> Result<Vec<AuditLogEntry>, DomainError> {
        self.repo.list_by_entity(entity_type, entity_id)
    }

    /// Returns audit log entries matching the given filter with total count.
    pub fn list(&self, filter: &AuditLogFilter) -> Result<(Vec<AuditLogEntry>, i64), DomainError> {
        self.repo.list(filter)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::NaiveDateTime;
    use std::sync::Mutex;

    struct MockAuditRepo {
        entries: Mutex<Vec<AuditLogEntry>>,
    }

    impl MockAuditRepo {
        fn new() -> Self {
            Self {
                entries: Mutex::new(Vec::new()),
            }
        }
    }

    impl AuditLogRepo for MockAuditRepo {
        fn create(&self, entry: &mut AuditLogEntry) -> Result<(), DomainError> {
            entry.id = 1;
            self.entries.lock().unwrap().push(entry.clone());
            Ok(())
        }

        fn list_by_entity(
            &self,
            entity_type: &str,
            entity_id: i64,
        ) -> Result<Vec<AuditLogEntry>, DomainError> {
            let entries = self.entries.lock().unwrap();
            Ok(entries
                .iter()
                .filter(|e| e.entity_type == entity_type && e.entity_id == entity_id)
                .cloned()
                .collect())
        }

        fn list(&self, _filter: &AuditLogFilter) -> Result<(Vec<AuditLogEntry>, i64), DomainError> {
            let entries = self.entries.lock().unwrap();
            let count = entries.len() as i64;
            Ok((entries.clone(), count))
        }
    }

    #[test]
    fn test_log_creates_entry() {
        let repo = Arc::new(MockAuditRepo::new());
        let svc = AuditService::new(repo.clone());

        svc.log("contact", 42, "create", None, Some(r#"{"name":"Test"}"#));

        let entries = repo.entries.lock().unwrap();
        assert_eq!(entries.len(), 1);
        assert_eq!(entries[0].entity_type, "contact");
        assert_eq!(entries[0].entity_id, 42);
        assert_eq!(entries[0].action, "create");
    }

    #[test]
    fn test_list_by_entity() {
        let repo = Arc::new(MockAuditRepo::new());
        let svc = AuditService::new(repo.clone());

        svc.log("invoice", 1, "create", None, None);
        svc.log("invoice", 1, "update", None, None);
        svc.log("contact", 2, "create", None, None);

        let result = svc.list_by_entity("invoice", 1).unwrap();
        assert_eq!(result.len(), 2);
    }
}
