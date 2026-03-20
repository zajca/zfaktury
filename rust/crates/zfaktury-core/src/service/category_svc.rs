use std::sync::Arc;

use zfaktury_domain::{DomainError, ExpenseCategory};

use super::audit_svc::AuditService;
use crate::repository::traits::CategoryRepo;

/// Service for expense category management.
pub struct CategoryService {
    repo: Arc<dyn CategoryRepo + Send + Sync>,
    audit: Option<Arc<AuditService>>,
}

impl CategoryService {
    pub fn new(
        repo: Arc<dyn CategoryRepo + Send + Sync>,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self { repo, audit }
    }

    fn validate_category(cat: &mut ExpenseCategory) -> Result<(), DomainError> {
        cat.key = cat.key.trim().to_string();
        cat.label_cs = cat.label_cs.trim().to_string();
        cat.label_en = cat.label_en.trim().to_string();

        if cat.key.is_empty() || cat.label_cs.is_empty() || cat.label_en.is_empty() {
            return Err(DomainError::InvalidInput);
        }
        // Key must be lowercase alphanumeric with underscores.
        if !cat
            .key
            .chars()
            .all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || c == '_')
        {
            return Err(DomainError::InvalidInput);
        }
        Ok(())
    }

    pub fn create(&self, cat: &mut ExpenseCategory) -> Result<(), DomainError> {
        Self::validate_category(cat)?;

        // Check for duplicate key.
        if self.repo.get_by_key(&cat.key).is_ok() {
            return Err(DomainError::DuplicateNumber);
        }

        if cat.color.is_empty() {
            cat.color = "#6B7280".to_string();
        }

        self.repo.create(cat)?;
        if let Some(ref audit) = self.audit {
            audit.log("category", cat.id, "create", None, None);
        }
        Ok(())
    }

    pub fn update(&self, cat: &mut ExpenseCategory) -> Result<(), DomainError> {
        if cat.id == 0 {
            return Err(DomainError::InvalidInput);
        }
        Self::validate_category(cat)?;

        // Check for duplicate key (excluding self).
        if let Ok(existing) = self.repo.get_by_key(&cat.key)
            && existing.id != cat.id
        {
            return Err(DomainError::DuplicateNumber);
        }

        self.repo.get_by_id(cat.id)?; // verify exists
        self.repo.update(cat)?;
        if let Some(ref audit) = self.audit {
            audit.log("category", cat.id, "update", None, None);
        }
        Ok(())
    }

    pub fn delete(&self, id: i64) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        let cat = self.repo.get_by_id(id)?;
        if cat.is_default {
            return Err(DomainError::InvalidInput);
        }
        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("category", id, "delete", None, None);
        }
        Ok(())
    }

    pub fn get_by_id(&self, id: i64) -> Result<ExpenseCategory, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)
    }

    pub fn list(&self) -> Result<Vec<ExpenseCategory>, DomainError> {
        self.repo.list()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Mutex;

    fn make_category(key: &str, label_cs: &str, label_en: &str) -> ExpenseCategory {
        ExpenseCategory {
            id: 0,
            key: key.to_string(),
            label_cs: label_cs.to_string(),
            label_en: label_en.to_string(),
            color: String::new(),
            sort_order: 0,
            is_default: false,
            created_at: chrono::NaiveDateTime::default(),
            deleted_at: None,
        }
    }

    struct MockCategoryRepo {
        cats: Mutex<Vec<ExpenseCategory>>,
        next_id: Mutex<i64>,
    }

    impl MockCategoryRepo {
        fn new() -> Self {
            Self {
                cats: Mutex::new(Vec::new()),
                next_id: Mutex::new(1),
            }
        }
    }

    impl CategoryRepo for MockCategoryRepo {
        fn create(&self, cat: &mut ExpenseCategory) -> Result<(), DomainError> {
            let mut id = self.next_id.lock().unwrap();
            cat.id = *id;
            *id += 1;
            self.cats.lock().unwrap().push(cat.clone());
            Ok(())
        }
        fn update(&self, cat: &mut ExpenseCategory) -> Result<(), DomainError> {
            let mut cats = self.cats.lock().unwrap();
            if let Some(c) = cats.iter_mut().find(|c| c.id == cat.id) {
                *c = cat.clone();
                Ok(())
            } else {
                Err(DomainError::NotFound)
            }
        }
        fn delete(&self, id: i64) -> Result<(), DomainError> {
            let mut cats = self.cats.lock().unwrap();
            cats.retain(|c| c.id != id);
            Ok(())
        }
        fn get_by_id(&self, id: i64) -> Result<ExpenseCategory, DomainError> {
            self.cats
                .lock()
                .unwrap()
                .iter()
                .find(|c| c.id == id)
                .cloned()
                .ok_or(DomainError::NotFound)
        }
        fn get_by_key(&self, key: &str) -> Result<ExpenseCategory, DomainError> {
            self.cats
                .lock()
                .unwrap()
                .iter()
                .find(|c| c.key == key)
                .cloned()
                .ok_or(DomainError::NotFound)
        }
        fn list(&self) -> Result<Vec<ExpenseCategory>, DomainError> {
            Ok(self.cats.lock().unwrap().clone())
        }
    }

    #[test]
    fn test_create_category() {
        let repo = Arc::new(MockCategoryRepo::new());
        let svc = CategoryService::new(repo, None);

        let mut cat = make_category("office", "Kancelar", "Office");
        svc.create(&mut cat).unwrap();
        assert!(cat.id > 0);
        assert_eq!(cat.color, "#6B7280");
    }

    #[test]
    fn test_empty_key_rejected() {
        let repo = Arc::new(MockCategoryRepo::new());
        let svc = CategoryService::new(repo, None);

        let mut cat = make_category("", "Label", "Label");
        assert!(svc.create(&mut cat).is_err());
    }

    #[test]
    fn test_duplicate_key_rejected() {
        let repo = Arc::new(MockCategoryRepo::new());
        let svc = CategoryService::new(repo, None);

        let mut c1 = make_category("travel", "Cestovne", "Travel");
        svc.create(&mut c1).unwrap();

        let mut c2 = make_category("travel", "Jine", "Other");
        assert!(matches!(
            svc.create(&mut c2),
            Err(DomainError::DuplicateNumber)
        ));
    }

    #[test]
    fn test_cannot_delete_default() {
        let repo = Arc::new(MockCategoryRepo::new());
        let svc = CategoryService::new(repo, None);

        let mut cat = make_category("default_cat", "Vychozi", "Default");
        cat.is_default = true;
        svc.create(&mut cat).unwrap();

        assert!(svc.delete(cat.id).is_err());
    }
}
