use std::collections::HashMap;
use std::sync::Mutex;

use rusqlite::{Connection, params};

use chrono::Local;
use zfaktury_core::repository::traits::SettingsRepo;
use zfaktury_domain::DomainError;

use crate::helpers::format_datetime;

/// Returns true if the settings key might contain a secret value.
fn is_sensitive_key(key: &str) -> bool {
    let lower = key.to_ascii_lowercase();
    lower.contains("password")
        || lower.contains("secret")
        || lower.contains("token")
        || lower.contains("api_key")
}

/// Returns "<redacted>" for sensitive keys, otherwise the key itself.
fn redact_key<'a>(key: &'a str) -> &'a str {
    if is_sensitive_key(key) {
        "<redacted>"
    } else {
        key
    }
}

/// SQLite implementation of SettingsRepo.
pub struct SqliteSettingsRepo {
    conn: Mutex<Connection>,
}

impl SqliteSettingsRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}

impl SettingsRepo for SqliteSettingsRepo {
    fn get_all(&self) -> Result<HashMap<String, String>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut stmt = conn
            .prepare("SELECT key, value FROM settings")
            .map_err(|e| {
                log::error!("preparing get_all settings: {e}");
                DomainError::InvalidInput
            })?;

        let rows = stmt
            .query_map([], |row| {
                Ok((row.get::<_, String>(0)?, row.get::<_, String>(1)?))
            })
            .map_err(|e| {
                log::error!("querying all settings: {e}");
                DomainError::InvalidInput
            })?;

        let mut settings = HashMap::new();
        for row in rows {
            let (key, value) = row.map_err(|e| {
                log::error!("scanning setting row: {e}");
                DomainError::InvalidInput
            })?;
            settings.insert(key, value);
        }
        Ok(settings)
    }

    fn get(&self, key: &str) -> Result<String, DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.query_row(
            "SELECT value FROM settings WHERE key = ?1",
            params![key],
            |row| row.get(0),
        )
        .map_err(|e| match e {
            rusqlite::Error::QueryReturnedNoRows => DomainError::NotFound,
            _ => {
                log::error!("querying setting {}: {e}", redact_key(key));
                DomainError::InvalidInput
            }
        })
    }

    fn set(&self, key: &str, value: &str) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        conn.execute(
            "INSERT INTO settings (key, value, updated_at) VALUES (?1, ?2, ?3)
             ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at",
            params![key, value, now],
        )
        .map_err(|e| {
            log::error!("upserting setting {}: {e}", redact_key(key));
            DomainError::InvalidInput
        })?;
        Ok(())
    }

    fn set_bulk(&self, settings: &HashMap<String, String>) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());

        let tx = conn.unchecked_transaction().map_err(|e| {
            log::error!("beginning transaction for bulk settings: {e}");
            DomainError::InvalidInput
        })?;

        {
            let mut stmt = tx
                .prepare(
                    "INSERT INTO settings (key, value, updated_at) VALUES (?1, ?2, ?3)
                     ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at",
                )
                .map_err(|e| {
                    log::error!("preparing bulk settings statement: {e}");
                    DomainError::InvalidInput
                })?;

            for (key, value) in settings {
                stmt.execute(params![key, value, now]).map_err(|e| {
                    log::error!("upserting setting {} in bulk: {e}", redact_key(key));
                    DomainError::InvalidInput
                })?;
            }
        }

        tx.commit().map_err(|e| {
            log::error!("committing bulk settings: {e}");
            DomainError::InvalidInput
        })?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;

    #[test]
    fn test_set_and_get() {
        let conn = new_test_db();
        let repo = SqliteSettingsRepo::new(conn);

        repo.set("company_name", "My Company").unwrap();
        let val = repo.get("company_name").unwrap();
        assert_eq!(val, "My Company");
    }

    #[test]
    fn test_get_not_found() {
        let conn = new_test_db();
        let repo = SqliteSettingsRepo::new(conn);

        let result = repo.get("nonexistent_key");
        assert!(matches!(result, Err(DomainError::NotFound)));
    }

    #[test]
    fn test_upsert() {
        let conn = new_test_db();
        let repo = SqliteSettingsRepo::new(conn);

        repo.set("key1", "value1").unwrap();
        repo.set("key1", "value2").unwrap();
        assert_eq!(repo.get("key1").unwrap(), "value2");
    }

    #[test]
    fn test_get_all() {
        let conn = new_test_db();
        let repo = SqliteSettingsRepo::new(conn);

        repo.set("a", "1").unwrap();
        repo.set("b", "2").unwrap();

        let all = repo.get_all().unwrap();
        assert_eq!(all.get("a").unwrap(), "1");
        assert_eq!(all.get("b").unwrap(), "2");
    }

    #[test]
    fn test_redact_sensitive_keys() {
        assert_eq!(redact_key("company_name"), "company_name");
        assert_eq!(redact_key("smtp_password"), "<redacted>");
        assert_eq!(redact_key("fakturoid_client_secret"), "<redacted>");
        assert_eq!(redact_key("oauth_token"), "<redacted>");
        assert_eq!(redact_key("my_api_key"), "<redacted>");
        assert_eq!(redact_key("company_ico"), "company_ico");
    }

    #[test]
    fn test_set_bulk() {
        let conn = new_test_db();
        let repo = SqliteSettingsRepo::new(conn);

        let mut settings = HashMap::new();
        settings.insert("x".to_string(), "10".to_string());
        settings.insert("y".to_string(), "20".to_string());
        repo.set_bulk(&settings).unwrap();

        assert_eq!(repo.get("x").unwrap(), "10");
        assert_eq!(repo.get("y").unwrap(), "20");
    }
}
