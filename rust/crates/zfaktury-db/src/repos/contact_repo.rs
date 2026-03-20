use rusqlite::{Connection, Row, params};
use std::sync::Mutex;

use chrono::Local;
use zfaktury_core::repository::traits::ContactRepo;
use zfaktury_domain::{Contact, ContactFilter, ContactType, DomainError};

use crate::helpers::*;

/// SQLite implementation of ContactRepo.
pub struct SqliteContactRepo {
    conn: Mutex<Connection>,
}

impl SqliteContactRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}

fn parse_contact_type(s: &str) -> ContactType {
    match s {
        "individual" => ContactType::Individual,
        _ => ContactType::Company,
    }
}

fn scan_contact(row: &Row<'_>) -> rusqlite::Result<Contact> {
    let type_str: String = row.get("type")?;
    let created_at_str: String = row.get("created_at")?;
    let updated_at_str: String = row.get("updated_at")?;
    let deleted_at_str: Option<String> = row.get("deleted_at")?;
    let vat_unreliable_at_str: Option<String> = row.get("vat_unreliable_at")?;

    Ok(Contact {
        id: row.get("id")?,
        contact_type: parse_contact_type(&type_str),
        name: row.get("name")?,
        ico: row.get::<_, Option<String>>("ico")?.unwrap_or_default(),
        dic: row.get::<_, Option<String>>("dic")?.unwrap_or_default(),
        street: row.get::<_, Option<String>>("street")?.unwrap_or_default(),
        city: row.get::<_, Option<String>>("city")?.unwrap_or_default(),
        zip: row.get::<_, Option<String>>("zip")?.unwrap_or_default(),
        country: row.get("country")?,
        email: row.get::<_, Option<String>>("email")?.unwrap_or_default(),
        phone: row.get::<_, Option<String>>("phone")?.unwrap_or_default(),
        web: row.get::<_, Option<String>>("web")?.unwrap_or_default(),
        bank_account: row
            .get::<_, Option<String>>("bank_account")?
            .unwrap_or_default(),
        bank_code: row
            .get::<_, Option<String>>("bank_code")?
            .unwrap_or_default(),
        iban: row.get::<_, Option<String>>("iban")?.unwrap_or_default(),
        swift: row.get::<_, Option<String>>("swift")?.unwrap_or_default(),
        payment_terms_days: row.get("payment_terms_days")?,
        tags: row.get::<_, Option<String>>("tags")?.unwrap_or_default(),
        notes: row.get::<_, Option<String>>("notes")?.unwrap_or_default(),
        is_favorite: row.get::<_, i32>("is_favorite")? != 0,
        vat_unreliable_at: parse_datetime_optional(vat_unreliable_at_str.as_deref())
            .unwrap_or(None),
        created_at: parse_datetime(&created_at_str).unwrap_or_default(),
        updated_at: parse_datetime(&updated_at_str).unwrap_or_default(),
        deleted_at: parse_datetime_optional(deleted_at_str.as_deref()).unwrap_or(None),
    })
}

const SELECT_COLUMNS: &str = "id, type, name, ico, dic, street, city, zip, country, \
     email, phone, web, bank_account, bank_code, iban, swift, \
     payment_terms_days, tags, notes, is_favorite, vat_unreliable_at, \
     created_at, updated_at, deleted_at";

impl ContactRepo for SqliteContactRepo {
    fn create(&self, contact: &mut Contact) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = Local::now().naive_local();
        contact.created_at = now;
        contact.updated_at = now;

        let now_str = format_datetime(&now);
        let vat_unreliable_str = format_datetime_opt(&contact.vat_unreliable_at);

        conn.execute(
            "INSERT INTO contacts (
                type, name, ico, dic, street, city, zip, country,
                email, phone, web, bank_account, bank_code, iban, swift,
                payment_terms_days, tags, notes, is_favorite, vat_unreliable_at,
                created_at, updated_at
            ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13, ?14, ?15, ?16, ?17, ?18, ?19, ?20, ?21, ?22)",
            params![
                contact.contact_type.to_string(),
                contact.name,
                contact.ico,
                contact.dic,
                contact.street,
                contact.city,
                contact.zip,
                contact.country,
                contact.email,
                contact.phone,
                contact.web,
                contact.bank_account,
                contact.bank_code,
                contact.iban,
                contact.swift,
                contact.payment_terms_days,
                contact.tags,
                contact.notes,
                contact.is_favorite as i32,
                vat_unreliable_str,
                now_str,
                now_str,
            ],
        ).map_err(|e| {
            log::error!("inserting contact: {e}");
            DomainError::InvalidInput
        })?;

        contact.id = conn.last_insert_rowid();
        Ok(())
    }

    fn update(&self, contact: &mut Contact) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = Local::now().naive_local();
        contact.updated_at = now;

        let now_str = format_datetime(&now);
        let vat_unreliable_str = format_datetime_opt(&contact.vat_unreliable_at);

        conn.execute(
            "UPDATE contacts SET
                type = ?1, name = ?2, ico = ?3, dic = ?4, street = ?5, city = ?6, zip = ?7, country = ?8,
                email = ?9, phone = ?10, web = ?11, bank_account = ?12, bank_code = ?13, iban = ?14, swift = ?15,
                payment_terms_days = ?16, tags = ?17, notes = ?18, is_favorite = ?19, vat_unreliable_at = ?20,
                updated_at = ?21
            WHERE id = ?22 AND deleted_at IS NULL",
            params![
                contact.contact_type.to_string(),
                contact.name,
                contact.ico,
                contact.dic,
                contact.street,
                contact.city,
                contact.zip,
                contact.country,
                contact.email,
                contact.phone,
                contact.web,
                contact.bank_account,
                contact.bank_code,
                contact.iban,
                contact.swift,
                contact.payment_terms_days,
                contact.tags,
                contact.notes,
                contact.is_favorite as i32,
                vat_unreliable_str,
                now_str,
                contact.id,
            ],
        ).map_err(|e| {
            log::error!("updating contact {}: {e}", contact.id);
            DomainError::InvalidInput
        })?;

        Ok(())
    }

    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now_str = format_datetime(&Local::now().naive_local());

        let rows = conn.execute(
            "UPDATE contacts SET deleted_at = ?1, updated_at = ?1 WHERE id = ?2 AND deleted_at IS NULL",
            params![now_str, id],
        ).map_err(|e| {
            log::error!("soft-deleting contact {id}: {e}");
            DomainError::InvalidInput
        })?;

        if rows == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }

    fn get_by_id(&self, id: i64) -> Result<Contact, DomainError> {
        let conn = self.conn.lock().unwrap();
        let query =
            format!("SELECT {SELECT_COLUMNS} FROM contacts WHERE id = ?1 AND deleted_at IS NULL");
        conn.query_row(&query, params![id], scan_contact)
            .map_err(|e| match e {
                rusqlite::Error::QueryReturnedNoRows => DomainError::NotFound,
                _ => {
                    log::error!("querying contact {id}: {e}");
                    DomainError::InvalidInput
                }
            })
    }

    fn list(&self, filter: &ContactFilter) -> Result<(Vec<Contact>, i64), DomainError> {
        let conn = self.conn.lock().unwrap();

        let mut where_clause = String::from("deleted_at IS NULL");
        let mut param_values: Vec<Box<dyn rusqlite::types::ToSql>> = Vec::new();

        if !filter.search.is_empty() {
            where_clause.push_str(" AND (name LIKE ?1 OR ico LIKE ?1 OR email LIKE ?1)");
            let search = format!("%{}%", filter.search);
            param_values.push(Box::new(search));
        }
        if let Some(ref ct) = filter.contact_type {
            let idx = param_values.len() + 1;
            where_clause.push_str(&format!(" AND type = ?{idx}"));
            param_values.push(Box::new(ct.to_string()));
        }
        if let Some(fav) = filter.favorite {
            let idx = param_values.len() + 1;
            where_clause.push_str(&format!(" AND is_favorite = ?{idx}"));
            param_values.push(Box::new(fav as i32));
        }

        let params_ref: Vec<&dyn rusqlite::types::ToSql> =
            param_values.iter().map(|p| p.as_ref()).collect();

        // Count total matching rows.
        let count_query = format!("SELECT COUNT(*) FROM contacts WHERE {where_clause}");
        let total: i64 = conn
            .query_row(&count_query, params_ref.as_slice(), |row| row.get(0))
            .map_err(|e| {
                log::error!("counting contacts: {e}");
                DomainError::InvalidInput
            })?;

        // Fetch page.
        let mut query =
            format!("SELECT {SELECT_COLUMNS} FROM contacts WHERE {where_clause} ORDER BY name ASC");
        if filter.limit > 0 {
            query.push_str(&format!(" LIMIT {} OFFSET {}", filter.limit, filter.offset));
        }

        let params_ref2: Vec<&dyn rusqlite::types::ToSql> =
            param_values.iter().map(|p| p.as_ref()).collect();

        let mut stmt = conn.prepare(&query).map_err(|e| {
            log::error!("preparing contact list query: {e}");
            DomainError::InvalidInput
        })?;

        let contacts = stmt
            .query_map(params_ref2.as_slice(), scan_contact)
            .map_err(|e| {
                log::error!("listing contacts: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("scanning contact rows: {e}");
                DomainError::InvalidInput
            })?;

        Ok((contacts, total))
    }

    fn find_by_ico(&self, ico: &str) -> Result<Contact, DomainError> {
        let conn = self.conn.lock().unwrap();
        let query =
            format!("SELECT {SELECT_COLUMNS} FROM contacts WHERE ico = ?1 AND deleted_at IS NULL");
        conn.query_row(&query, params![ico], scan_contact)
            .map_err(|e| match e {
                rusqlite::Error::QueryReturnedNoRows => DomainError::NotFound,
                _ => {
                    log::error!("querying contact by ICO {ico}: {e}");
                    DomainError::InvalidInput
                }
            })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;

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
            country: "CZ".to_string(),
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
            created_at: Default::default(),
            updated_at: Default::default(),
            deleted_at: None,
        }
    }

    #[test]
    fn test_create_and_get_by_id() {
        let conn = new_test_db();
        let repo = SqliteContactRepo::new(conn);

        let mut contact = make_contact("Test Company");
        contact.ico = "12345678".to_string();
        contact.email = "test@example.com".to_string();

        repo.create(&mut contact).unwrap();
        assert!(contact.id > 0);

        let fetched = repo.get_by_id(contact.id).unwrap();
        assert_eq!(fetched.name, "Test Company");
        assert_eq!(fetched.ico, "12345678");
        assert_eq!(fetched.email, "test@example.com");
        assert_eq!(fetched.country, "CZ");
    }

    #[test]
    fn test_update() {
        let conn = new_test_db();
        let repo = SqliteContactRepo::new(conn);

        let mut contact = make_contact("Original Name");
        repo.create(&mut contact).unwrap();

        contact.name = "Updated Name".to_string();
        contact.city = "Prague".to_string();
        repo.update(&mut contact).unwrap();

        let fetched = repo.get_by_id(contact.id).unwrap();
        assert_eq!(fetched.name, "Updated Name");
        assert_eq!(fetched.city, "Prague");
    }

    #[test]
    fn test_soft_delete() {
        let conn = new_test_db();
        let repo = SqliteContactRepo::new(conn);

        let mut contact = make_contact("To Delete");
        repo.create(&mut contact).unwrap();

        repo.delete(contact.id).unwrap();

        let result = repo.get_by_id(contact.id);
        assert!(matches!(result, Err(DomainError::NotFound)));
    }

    #[test]
    fn test_list_with_search() {
        let conn = new_test_db();
        let repo = SqliteContactRepo::new(conn);

        let mut c1 = make_contact("Alpha Corp");
        let mut c2 = make_contact("Beta LLC");
        let mut c3 = make_contact("Gamma Inc");
        repo.create(&mut c1).unwrap();
        repo.create(&mut c2).unwrap();
        repo.create(&mut c3).unwrap();

        // List all.
        let filter = ContactFilter::default();
        let (all, total) = repo.list(&filter).unwrap();
        assert_eq!(total, 3);
        assert_eq!(all.len(), 3);

        // Search for "Beta".
        let filter = ContactFilter {
            search: "Beta".to_string(),
            ..Default::default()
        };
        let (found, total) = repo.list(&filter).unwrap();
        assert_eq!(total, 1);
        assert_eq!(found[0].name, "Beta LLC");
    }

    #[test]
    fn test_list_with_pagination() {
        let conn = new_test_db();
        let repo = SqliteContactRepo::new(conn);

        for i in 0..5 {
            let mut c = make_contact(&format!("Company {i:02}"));
            repo.create(&mut c).unwrap();
        }

        let filter = ContactFilter {
            limit: 2,
            offset: 0,
            ..Default::default()
        };
        let (page1, total) = repo.list(&filter).unwrap();
        assert_eq!(total, 5);
        assert_eq!(page1.len(), 2);

        let filter2 = ContactFilter {
            limit: 2,
            offset: 2,
            ..Default::default()
        };
        let (page2, _) = repo.list(&filter2).unwrap();
        assert_eq!(page2.len(), 2);
        assert_ne!(page1[0].id, page2[0].id);
    }

    #[test]
    fn test_find_by_ico() {
        let conn = new_test_db();
        let repo = SqliteContactRepo::new(conn);

        let mut contact = make_contact("ICO Company");
        contact.ico = "87654321".to_string();
        repo.create(&mut contact).unwrap();

        let found = repo.find_by_ico("87654321").unwrap();
        assert_eq!(found.name, "ICO Company");

        let not_found = repo.find_by_ico("99999999");
        assert!(matches!(not_found, Err(DomainError::NotFound)));
    }

    #[test]
    fn test_delete_not_found() {
        let conn = new_test_db();
        let repo = SqliteContactRepo::new(conn);

        let result = repo.delete(99999);
        assert!(matches!(result, Err(DomainError::NotFound)));
    }

    #[test]
    fn test_deleted_hidden_from_list() {
        let conn = new_test_db();
        let repo = SqliteContactRepo::new(conn);

        let mut c1 = make_contact("Visible");
        let mut c2 = make_contact("Deleted");
        repo.create(&mut c1).unwrap();
        repo.create(&mut c2).unwrap();

        repo.delete(c2.id).unwrap();

        let (list, total) = repo.list(&ContactFilter::default()).unwrap();
        assert_eq!(total, 1);
        assert_eq!(list[0].name, "Visible");
    }
}
