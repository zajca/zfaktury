use std::sync::Arc;

use zfaktury_domain::{Contact, ContactFilter, ContactType, DomainError};

use super::audit_svc::AuditService;
use crate::repository::traits::ContactRepo;

/// External ARES lookup client trait.
pub trait ARESClient: Send + Sync {
    fn lookup_by_ico(&self, ico: &str) -> Result<Contact, DomainError>;
}

/// Service for contact management.
pub struct ContactService {
    repo: Arc<dyn ContactRepo + Send + Sync>,
    ares: Option<Arc<dyn ARESClient>>,
    audit: Option<Arc<AuditService>>,
}

impl ContactService {
    pub fn new(
        repo: Arc<dyn ContactRepo + Send + Sync>,
        ares: Option<Arc<dyn ARESClient>>,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self { repo, ares, audit }
    }

    /// Validates and persists a new contact.
    pub fn create(&self, contact: &mut Contact) -> Result<(), DomainError> {
        if contact.name.is_empty() {
            return Err(DomainError::InvalidInput);
        }
        if contact.contact_type != ContactType::Company
            && contact.contact_type != ContactType::Individual
        {
            contact.contact_type = ContactType::Company;
        }
        self.repo.create(contact)?;
        if let Some(ref audit) = self.audit {
            audit.log("contact", contact.id, "create", None, None);
        }
        Ok(())
    }

    /// Validates and updates an existing contact.
    pub fn update(&self, contact: &mut Contact) -> Result<(), DomainError> {
        if contact.id == 0 {
            return Err(DomainError::InvalidInput);
        }
        if contact.name.is_empty() {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(contact.id)?; // verify exists
        self.repo.update(contact)?;
        if let Some(ref audit) = self.audit {
            audit.log("contact", contact.id, "update", None, None);
        }
        Ok(())
    }

    /// Soft-deletes a contact by ID.
    pub fn delete(&self, id: i64) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("contact", id, "delete", None, None);
        }
        Ok(())
    }

    /// Retrieves a contact by its ID.
    pub fn get_by_id(&self, id: i64) -> Result<Contact, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)
    }

    /// Retrieves contacts matching the given filter.
    pub fn list(&self, mut filter: ContactFilter) -> Result<(Vec<Contact>, i64), DomainError> {
        if filter.limit <= 0 {
            filter.limit = 20;
        }
        if filter.limit > 100 {
            filter.limit = 100;
        }
        if filter.offset < 0 {
            filter.offset = 0;
        }
        self.repo.list(&filter)
    }

    /// Looks up a company by ICO using the ARES registry.
    pub fn lookup_ares(&self, ico: &str) -> Result<Contact, DomainError> {
        if ico.is_empty() {
            return Err(DomainError::InvalidInput);
        }
        match &self.ares {
            Some(ares) => ares.lookup_by_ico(ico),
            None => Err(DomainError::InvalidInput),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::NaiveDateTime;
    use std::sync::Mutex;

    fn make_contact(name: &str) -> Contact {
        Contact {
            id: 0,
            contact_type: ContactType::Company,
            name: name.to_string(),
            ico: String::new(),
            dic: String::new(),
            street: String::new(),
            city: String::new(),
            zip: String::new(),
            country: String::new(),
            email: String::new(),
            phone: String::new(),
            web: String::new(),
            bank_account: String::new(),
            bank_code: String::new(),
            iban: String::new(),
            swift: String::new(),
            payment_terms_days: 14,
            tags: String::new(),
            notes: String::new(),
            is_favorite: false,
            vat_unreliable_at: None,
            created_at: NaiveDateTime::default(),
            updated_at: NaiveDateTime::default(),
            deleted_at: None,
        }
    }

    struct MockContactRepo {
        contacts: Mutex<Vec<Contact>>,
        next_id: Mutex<i64>,
    }

    impl MockContactRepo {
        fn new() -> Self {
            Self {
                contacts: Mutex::new(Vec::new()),
                next_id: Mutex::new(1),
            }
        }
    }

    impl ContactRepo for MockContactRepo {
        fn create(&self, contact: &mut Contact) -> Result<(), DomainError> {
            let mut id = self.next_id.lock().unwrap();
            contact.id = *id;
            *id += 1;
            self.contacts.lock().unwrap().push(contact.clone());
            Ok(())
        }

        fn update(&self, contact: &mut Contact) -> Result<(), DomainError> {
            let mut contacts = self.contacts.lock().unwrap();
            if let Some(c) = contacts.iter_mut().find(|c| c.id == contact.id) {
                *c = contact.clone();
                Ok(())
            } else {
                Err(DomainError::NotFound)
            }
        }

        fn delete(&self, id: i64) -> Result<(), DomainError> {
            let contacts = self.contacts.lock().unwrap();
            if contacts.iter().any(|c| c.id == id) {
                Ok(())
            } else {
                Err(DomainError::NotFound)
            }
        }

        fn get_by_id(&self, id: i64) -> Result<Contact, DomainError> {
            let contacts = self.contacts.lock().unwrap();
            contacts
                .iter()
                .find(|c| c.id == id)
                .cloned()
                .ok_or(DomainError::NotFound)
        }

        fn list(&self, _filter: &ContactFilter) -> Result<(Vec<Contact>, i64), DomainError> {
            let contacts = self.contacts.lock().unwrap();
            let count = contacts.len() as i64;
            Ok((contacts.clone(), count))
        }

        fn find_by_ico(&self, ico: &str) -> Result<Contact, DomainError> {
            let contacts = self.contacts.lock().unwrap();
            contacts
                .iter()
                .find(|c| c.ico == ico)
                .cloned()
                .ok_or(DomainError::NotFound)
        }
    }

    #[test]
    fn test_create_contact() {
        let repo = Arc::new(MockContactRepo::new());
        let svc = ContactService::new(repo, None, None);

        let mut contact = make_contact("Test Corp");
        svc.create(&mut contact).unwrap();
        assert_eq!(contact.id, 1);
    }

    #[test]
    fn test_create_requires_name() {
        let repo = Arc::new(MockContactRepo::new());
        let svc = ContactService::new(repo, None, None);

        let mut contact = make_contact("");
        let result = svc.create(&mut contact);
        assert!(matches!(result, Err(DomainError::InvalidInput)));
    }

    #[test]
    fn test_get_by_id() {
        let repo = Arc::new(MockContactRepo::new());
        let svc = ContactService::new(repo, None, None);

        let mut contact = make_contact("Test");
        svc.create(&mut contact).unwrap();

        let fetched = svc.get_by_id(contact.id).unwrap();
        assert_eq!(fetched.name, "Test");
    }

    #[test]
    fn test_update_contact() {
        let repo = Arc::new(MockContactRepo::new());
        let svc = ContactService::new(repo, None, None);

        let mut contact = make_contact("Original");
        svc.create(&mut contact).unwrap();

        contact.name = "Updated".to_string();
        svc.update(&mut contact).unwrap();

        let fetched = svc.get_by_id(contact.id).unwrap();
        assert_eq!(fetched.name, "Updated");
    }

    #[test]
    fn test_delete_contact() {
        let repo = Arc::new(MockContactRepo::new());
        let svc = ContactService::new(repo, None, None);

        let mut contact = make_contact("ToDelete");
        svc.create(&mut contact).unwrap();

        svc.delete(contact.id).unwrap();
    }

    #[test]
    fn test_list_with_defaults() {
        let repo = Arc::new(MockContactRepo::new());
        let svc = ContactService::new(repo, None, None);

        let mut c = make_contact("A");
        svc.create(&mut c).unwrap();

        let (contacts, count) = svc.list(ContactFilter::default()).unwrap();
        assert_eq!(count, 1);
        assert_eq!(contacts.len(), 1);
    }
}
