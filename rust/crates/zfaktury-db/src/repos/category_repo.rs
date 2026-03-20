use std::sync::Mutex;

use chrono::Local;
use rusqlite::{Connection, Row, params};
use zfaktury_core::repository::traits::CategoryRepo;
use zfaktury_domain::{DomainError, ExpenseCategory};

use crate::helpers::*;

pub struct SqliteCategoryRepo {
    conn: Mutex<Connection>,
}

impl SqliteCategoryRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}

fn scan_category(row: &Row<'_>) -> rusqlite::Result<ExpenseCategory> {
    let created_at_str: String = row.get("created_at")?;
    let deleted_at_str: Option<String> = row.get("deleted_at")?;
    Ok(ExpenseCategory {
        id: row.get("id")?,
        key: row.get("key")?,
        label_cs: row.get("label_cs")?,
        label_en: row.get("label_en")?,
        color: row.get("color")?,
        sort_order: row.get("sort_order")?,
        is_default: row.get::<_, i32>("is_default")? != 0,
        created_at: parse_datetime_or_default(&created_at_str),
        deleted_at: parse_datetime_optional(deleted_at_str.as_deref()).unwrap_or(None),
    })
}

const COLS: &str =
    "id, key, label_cs, label_en, color, sort_order, is_default, created_at, deleted_at";

impl CategoryRepo for SqliteCategoryRepo {
    fn create(&self, cat: &mut ExpenseCategory) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        conn.execute(
            "INSERT INTO expense_categories (key, label_cs, label_en, color, sort_order, is_default, created_at) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)",
            params![cat.key, cat.label_cs, cat.label_en, cat.color, cat.sort_order, cat.is_default as i32, now],
        ).map_err(|e| { log::error!("inserting category: {e}"); DomainError::InvalidInput })?;
        cat.id = conn.last_insert_rowid();
        Ok(())
    }

    fn update(&self, cat: &mut ExpenseCategory) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.execute(
            "UPDATE expense_categories SET key = ?1, label_cs = ?2, label_en = ?3, color = ?4, sort_order = ?5 WHERE id = ?6 AND deleted_at IS NULL",
            params![cat.key, cat.label_cs, cat.label_en, cat.color, cat.sort_order, cat.id],
        ).map_err(|e| { log::error!("updating category: {e}"); DomainError::InvalidInput })?;
        Ok(())
    }

    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        let rows = conn.execute(
            "UPDATE expense_categories SET deleted_at = ?1 WHERE id = ?2 AND deleted_at IS NULL",
            params![now, id],
        ).map_err(|e| { log::error!("deleting category: {e}"); DomainError::InvalidInput })?;
        if rows == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }

    fn get_by_id(&self, id: i64) -> Result<ExpenseCategory, DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.query_row(
            &format!("SELECT {COLS} FROM expense_categories WHERE id = ?1 AND deleted_at IS NULL"),
            params![id],
            scan_category,
        )
        .map_err(|e| match e {
            rusqlite::Error::QueryReturnedNoRows => DomainError::NotFound,
            _ => {
                log::error!("querying category: {e}");
                DomainError::InvalidInput
            }
        })
    }

    fn get_by_key(&self, key: &str) -> Result<ExpenseCategory, DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.query_row(
            &format!("SELECT {COLS} FROM expense_categories WHERE key = ?1 AND deleted_at IS NULL"),
            params![key],
            scan_category,
        )
        .map_err(|e| match e {
            rusqlite::Error::QueryReturnedNoRows => DomainError::NotFound,
            _ => {
                log::error!("querying category by key: {e}");
                DomainError::InvalidInput
            }
        })
    }

    fn list(&self) -> Result<Vec<ExpenseCategory>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut stmt = conn.prepare(&format!("SELECT {COLS} FROM expense_categories WHERE deleted_at IS NULL ORDER BY sort_order ASC"))
            .map_err(|e| { log::error!("preparing category list: {e}"); DomainError::InvalidInput })?;
        let rows = stmt.query_map([], scan_category).map_err(|e| {
            log::error!("listing categories: {e}");
            DomainError::InvalidInput
        })?;
        rows.collect::<Result<Vec<_>, _>>().map_err(|e| {
            log::error!("scanning categories: {e}");
            DomainError::InvalidInput
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;

    #[test]
    fn test_list_default_categories() {
        let conn = new_test_db();
        let repo = SqliteCategoryRepo::new(conn);
        let list = repo.list().unwrap();
        assert_eq!(list.len(), 16);
    }

    #[test]
    fn test_get_by_key() {
        let conn = new_test_db();
        let repo = SqliteCategoryRepo::new(conn);
        let cat = repo.get_by_key("software").unwrap();
        assert_eq!(cat.label_en, "Software & licenses");
    }

    #[test]
    fn test_create_and_delete() {
        let conn = new_test_db();
        let repo = SqliteCategoryRepo::new(conn);
        let mut cat = ExpenseCategory {
            id: 0,
            key: "test_cat".to_string(),
            label_cs: "Test".to_string(),
            label_en: "Test".to_string(),
            color: "#FF0000".to_string(),
            sort_order: 100,
            is_default: false,
            created_at: Default::default(),
            deleted_at: None,
        };
        repo.create(&mut cat).unwrap();
        assert!(cat.id > 0);

        repo.delete(cat.id).unwrap();
        assert!(matches!(repo.get_by_id(cat.id), Err(DomainError::NotFound)));
    }
}
