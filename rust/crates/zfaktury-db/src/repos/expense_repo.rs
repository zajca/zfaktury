use std::sync::Mutex;

use chrono::Local;
use rusqlite::{Connection, Row, params};
use zfaktury_core::repository::traits::ExpenseRepo;
use zfaktury_domain::*;

use crate::helpers::*;

pub struct SqliteExpenseRepo {
    conn: Mutex<Connection>,
}

impl SqliteExpenseRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}

fn scan_expense_core(row: &Row<'_>) -> rusqlite::Result<Expense> {
    let issue_date_str: String = row.get("issue_date")?;
    let created_at_str: String = row.get("created_at")?;
    let updated_at_str: String = row.get("updated_at")?;
    let deleted_at_str: Option<String> = row.get("deleted_at")?;
    let tax_reviewed_str: Option<String> = row.get("tax_reviewed_at")?;
    let vendor_id: Option<i64> = row.get("vendor_id")?;

    Ok(Expense {
        id: row.get("id")?,
        vendor_id,
        vendor: None,
        expense_number: row
            .get::<_, Option<String>>("expense_number")?
            .unwrap_or_default(),
        category: row
            .get::<_, Option<String>>("category")?
            .unwrap_or_default(),
        description: row.get("description")?,
        issue_date: parse_date_or_default(&issue_date_str),
        amount: Amount::from_halere(row.get::<_, i64>("amount")?),
        currency_code: row.get("currency_code")?,
        exchange_rate: Amount::from_halere(row.get::<_, i64>("exchange_rate")?),
        vat_rate_percent: row.get("vat_rate_percent")?,
        vat_amount: Amount::from_halere(row.get::<_, i64>("vat_amount")?),
        is_tax_deductible: row.get::<_, i32>("is_tax_deductible")? != 0,
        business_percent: row.get("business_percent")?,
        payment_method: row.get("payment_method")?,
        document_path: row
            .get::<_, Option<String>>("document_path")?
            .unwrap_or_default(),
        notes: row.get::<_, Option<String>>("notes")?.unwrap_or_default(),
        tax_reviewed_at: parse_datetime_optional(tax_reviewed_str.as_deref()).unwrap_or(None),
        items: Vec::new(),
        created_at: parse_datetime_or_default(&created_at_str),
        updated_at: parse_datetime_or_default(&updated_at_str),
        deleted_at: parse_datetime_optional(deleted_at_str.as_deref()).unwrap_or(None),
    })
}

fn scan_expense_item(row: &Row<'_>) -> rusqlite::Result<ExpenseItem> {
    Ok(ExpenseItem {
        id: row.get("id")?,
        expense_id: row.get("expense_id")?,
        description: row.get("description")?,
        quantity: Amount::from_halere(row.get::<_, i64>("quantity")?),
        unit: row.get("unit")?,
        unit_price: Amount::from_halere(row.get::<_, i64>("unit_price")?),
        vat_rate_percent: row.get("vat_rate_percent")?,
        vat_amount: Amount::from_halere(row.get::<_, i64>("vat_amount")?),
        total_amount: Amount::from_halere(row.get::<_, i64>("total_amount")?),
        sort_order: row.get("sort_order")?,
    })
}

const EXP_COLS: &str = "id, vendor_id, expense_number, category, description, \
    issue_date, amount, currency_code, exchange_rate, \
    vat_rate_percent, vat_amount, \
    is_tax_deductible, business_percent, payment_method, \
    document_path, notes, tax_reviewed_at, \
    created_at, updated_at, deleted_at";

fn fetch_expense_items(
    conn: &Connection,
    expense_id: i64,
) -> Result<Vec<ExpenseItem>, DomainError> {
    let mut stmt = conn.prepare(
        "SELECT id, expense_id, description, quantity, unit, unit_price, vat_rate_percent, vat_amount, total_amount, sort_order \
         FROM expense_items WHERE expense_id = ?1 ORDER BY sort_order ASC"
    ).map_err(|e| { log::error!("preparing expense items: {e}"); DomainError::InvalidInput })?;
    let items = stmt
        .query_map(params![expense_id], scan_expense_item)
        .map_err(|e| {
            log::error!("querying expense items: {e}");
            DomainError::InvalidInput
        })?
        .collect::<Result<Vec<_>, _>>()
        .map_err(|e| {
            log::error!("scanning expense items: {e}");
            DomainError::InvalidInput
        })?;
    Ok(items)
}

fn insert_expense_items(
    conn: &Connection,
    expense_id: i64,
    items: &mut [ExpenseItem],
) -> Result<(), DomainError> {
    for (i, item) in items.iter_mut().enumerate() {
        item.expense_id = expense_id;
        conn.execute(
            "INSERT INTO expense_items (expense_id, description, quantity, unit, unit_price, vat_rate_percent, vat_amount, total_amount, sort_order) \
             VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9)",
            params![expense_id, item.description, item.quantity.halere(), item.unit, item.unit_price.halere(), item.vat_rate_percent, item.vat_amount.halere(), item.total_amount.halere(), item.sort_order],
        ).map_err(|e| { log::error!("inserting expense item {i}: {e}"); DomainError::InvalidInput })?;
        item.id = conn.last_insert_rowid();
    }
    Ok(())
}

impl ExpenseRepo for SqliteExpenseRepo {
    fn create(&self, expense: &mut Expense) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = Local::now().naive_local();
        expense.created_at = now;
        expense.updated_at = now;
        let now_str = format_datetime(&now);

        let tx = conn.unchecked_transaction().map_err(|e| {
            log::error!("begin tx: {e}");
            DomainError::InvalidInput
        })?;

        tx.execute(
            "INSERT INTO expenses (vendor_id, expense_number, category, description, \
             issue_date, amount, currency_code, exchange_rate, \
             vat_rate_percent, vat_amount, is_tax_deductible, business_percent, payment_method, \
             document_path, notes, tax_reviewed_at, created_at, updated_at) \
             VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13, ?14, ?15, NULL, ?16, ?16)",
            params![
                expense.vendor_id, expense.expense_number, expense.category, expense.description,
                format_date(&expense.issue_date), expense.amount.halere(), expense.currency_code, expense.exchange_rate.halere(),
                expense.vat_rate_percent, expense.vat_amount.halere(), expense.is_tax_deductible as i32, expense.business_percent, expense.payment_method,
                expense.document_path, expense.notes, now_str,
            ],
        ).map_err(|e| { log::error!("inserting expense: {e}"); DomainError::InvalidInput })?;

        expense.id = tx.last_insert_rowid();
        insert_expense_items(&tx, expense.id, &mut expense.items)?;

        tx.commit().map_err(|e| {
            log::error!("committing expense: {e}");
            DomainError::InvalidInput
        })?;
        Ok(())
    }

    fn update(&self, expense: &mut Expense) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = Local::now().naive_local();
        expense.updated_at = now;
        let now_str = format_datetime(&now);
        let tax_reviewed_str = format_datetime_opt(&expense.tax_reviewed_at);

        let tx = conn.unchecked_transaction().map_err(|e| {
            log::error!("begin tx: {e}");
            DomainError::InvalidInput
        })?;

        tx.execute(
            "UPDATE expenses SET vendor_id = ?1, expense_number = ?2, category = ?3, description = ?4, \
             issue_date = ?5, amount = ?6, currency_code = ?7, exchange_rate = ?8, \
             vat_rate_percent = ?9, vat_amount = ?10, is_tax_deductible = ?11, business_percent = ?12, payment_method = ?13, \
             document_path = ?14, notes = ?15, tax_reviewed_at = ?16, updated_at = ?17 \
             WHERE id = ?18 AND deleted_at IS NULL",
            params![
                expense.vendor_id, expense.expense_number, expense.category, expense.description,
                format_date(&expense.issue_date), expense.amount.halere(), expense.currency_code, expense.exchange_rate.halere(),
                expense.vat_rate_percent, expense.vat_amount.halere(), expense.is_tax_deductible as i32, expense.business_percent, expense.payment_method,
                expense.document_path, expense.notes, tax_reviewed_str, now_str, expense.id,
            ],
        ).map_err(|e| { log::error!("updating expense: {e}"); DomainError::InvalidInput })?;

        tx.execute(
            "DELETE FROM expense_items WHERE expense_id = ?1",
            params![expense.id],
        )
        .map_err(|e| {
            log::error!("deleting old expense items: {e}");
            DomainError::InvalidInput
        })?;
        insert_expense_items(&tx, expense.id, &mut expense.items)?;

        tx.commit().map_err(|e| {
            log::error!("committing expense update: {e}");
            DomainError::InvalidInput
        })?;
        Ok(())
    }

    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now_str = format_datetime(&Local::now().naive_local());
        let rows = conn.execute(
            "UPDATE expenses SET deleted_at = ?1, updated_at = ?1 WHERE id = ?2 AND deleted_at IS NULL",
            params![now_str, id],
        ).map_err(|e| { log::error!("deleting expense: {e}"); DomainError::InvalidInput })?;
        if rows == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }

    fn get_by_id(&self, id: i64) -> Result<Expense, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut exp = conn
            .query_row(
                &format!("SELECT {EXP_COLS} FROM expenses WHERE id = ?1 AND deleted_at IS NULL"),
                params![id],
                scan_expense_core,
            )
            .map_err(|e| match e {
                rusqlite::Error::QueryReturnedNoRows => DomainError::NotFound,
                _ => {
                    log::error!("querying expense {id}: {e}");
                    DomainError::InvalidInput
                }
            })?;
        exp.items = fetch_expense_items(&conn, id)?;
        Ok(exp)
    }

    fn list(&self, filter: &ExpenseFilter) -> Result<(Vec<Expense>, i64), DomainError> {
        let conn = self.conn.lock().unwrap();

        let mut where_clause = String::from("e.deleted_at IS NULL");
        let mut param_values: Vec<Box<dyn rusqlite::types::ToSql>> = Vec::new();

        if !filter.category.is_empty() {
            let idx = param_values.len() + 1;
            where_clause.push_str(&format!(" AND e.category = ?{idx}"));
            param_values.push(Box::new(filter.category.clone()));
        }
        if let Some(vid) = filter.vendor_id {
            let idx = param_values.len() + 1;
            where_clause.push_str(&format!(" AND e.vendor_id = ?{idx}"));
            param_values.push(Box::new(vid));
        }
        if let Some(ref from) = filter.date_from {
            let idx = param_values.len() + 1;
            where_clause.push_str(&format!(" AND e.issue_date >= ?{idx}"));
            param_values.push(Box::new(format_date(from)));
        }
        if let Some(ref to) = filter.date_to {
            let idx = param_values.len() + 1;
            where_clause.push_str(&format!(" AND e.issue_date <= ?{idx}"));
            param_values.push(Box::new(format_date(to)));
        }
        if !filter.search.is_empty() {
            let idx = param_values.len() + 1;
            where_clause.push_str(&format!(
                " AND (e.expense_number LIKE ?{idx} OR e.description LIKE ?{idx})"
            ));
            param_values.push(Box::new(format!("%{}%", filter.search)));
        }
        if let Some(reviewed) = filter.tax_reviewed {
            if reviewed {
                where_clause.push_str(" AND e.tax_reviewed_at IS NOT NULL");
            } else {
                where_clause.push_str(" AND e.tax_reviewed_at IS NULL");
            }
        }

        let params_ref: Vec<&dyn rusqlite::types::ToSql> =
            param_values.iter().map(|p| p.as_ref()).collect();

        let count_query = format!("SELECT COUNT(*) FROM expenses e WHERE {where_clause}");
        let total: i64 = conn
            .query_row(&count_query, params_ref.as_slice(), |row| row.get(0))
            .map_err(|e| {
                log::error!("counting expenses: {e}");
                DomainError::InvalidInput
            })?;

        let exp_cols_prefixed = EXP_COLS
            .split(", ")
            .map(|c| format!("e.{c}"))
            .collect::<Vec<_>>()
            .join(", ");
        let mut query = format!(
            "SELECT {exp_cols_prefixed} FROM expenses e WHERE {where_clause} ORDER BY e.issue_date DESC"
        );
        if filter.limit > 0 {
            let next = param_values.len() + 1;
            query.push_str(&format!(" LIMIT ?{} OFFSET ?{}", next, next + 1));
            param_values.push(Box::new(filter.limit as i64));
            param_values.push(Box::new(filter.offset as i64));
        }

        let params_ref2: Vec<&dyn rusqlite::types::ToSql> =
            param_values.iter().map(|p| p.as_ref()).collect();
        let mut stmt = conn.prepare(&query).map_err(|e| {
            log::error!("preparing expense list: {e}");
            DomainError::InvalidInput
        })?;
        let expenses = stmt
            .query_map(params_ref2.as_slice(), scan_expense_core)
            .map_err(|e| {
                log::error!("listing expenses: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("scanning expenses: {e}");
                DomainError::InvalidInput
            })?;

        Ok((expenses, total))
    }

    fn mark_tax_reviewed(&self, ids: &[i64]) -> Result<(), DomainError> {
        if ids.is_empty() {
            return Ok(());
        }
        let conn = self.conn.lock().unwrap();
        let now_str = format_datetime(&Local::now().naive_local());
        let placeholders: String = (0..ids.len())
            .map(|i| format!("?{}", i + 2))
            .collect::<Vec<_>>()
            .join(",");
        let sql = format!(
            "UPDATE expenses SET tax_reviewed_at = ?1 WHERE id IN ({}) AND deleted_at IS NULL",
            placeholders
        );
        let mut params: Vec<Box<dyn rusqlite::types::ToSql>> = Vec::new();
        params.push(Box::new(now_str));
        for id in ids {
            params.push(Box::new(*id));
        }
        let param_refs: Vec<&dyn rusqlite::types::ToSql> =
            params.iter().map(|p| p.as_ref()).collect();
        conn.execute(&sql, param_refs.as_slice()).map_err(|e| {
            log::error!("marking tax reviewed: {e}");
            DomainError::InvalidInput
        })?;
        Ok(())
    }

    fn unmark_tax_reviewed(&self, ids: &[i64]) -> Result<(), DomainError> {
        if ids.is_empty() {
            return Ok(());
        }
        let conn = self.conn.lock().unwrap();
        let placeholders: String = (0..ids.len())
            .map(|i| format!("?{}", i + 1))
            .collect::<Vec<_>>()
            .join(",");
        let sql = format!(
            "UPDATE expenses SET tax_reviewed_at = NULL WHERE id IN ({}) AND deleted_at IS NULL",
            placeholders
        );
        let mut params: Vec<Box<dyn rusqlite::types::ToSql>> = Vec::new();
        for id in ids {
            params.push(Box::new(*id));
        }
        let param_refs: Vec<&dyn rusqlite::types::ToSql> =
            params.iter().map(|p| p.as_ref()).collect();
        conn.execute(&sql, param_refs.as_slice()).map_err(|e| {
            log::error!("unmarking tax reviewed: {e}");
            DomainError::InvalidInput
        })?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;
    use chrono::NaiveDate;

    fn make_expense() -> Expense {
        Expense {
            id: 0,
            vendor_id: None,
            vendor: None,
            expense_number: "EXP001".to_string(),
            category: "software".to_string(),
            description: "IDE License".to_string(),
            issue_date: NaiveDate::from_ymd_opt(2025, 3, 1).unwrap(),
            amount: Amount::from_halere(5000),
            currency_code: "CZK".to_string(),
            exchange_rate: Amount::from_halere(100),
            vat_rate_percent: 21,
            vat_amount: Amount::from_halere(1050),
            is_tax_deductible: true,
            business_percent: 100,
            payment_method: "bank_transfer".to_string(),
            document_path: String::new(),
            notes: String::new(),
            tax_reviewed_at: None,
            items: Vec::new(),
            created_at: Default::default(),
            updated_at: Default::default(),
            deleted_at: None,
        }
    }

    #[test]
    fn test_create_and_get() {
        let conn = new_test_db();
        let repo = SqliteExpenseRepo::new(conn);
        let mut exp = make_expense();
        repo.create(&mut exp).unwrap();
        assert!(exp.id > 0);

        let fetched = repo.get_by_id(exp.id).unwrap();
        assert_eq!(fetched.description, "IDE License");
        assert_eq!(fetched.amount, Amount::from_halere(5000));
    }

    #[test]
    fn test_update() {
        let conn = new_test_db();
        let repo = SqliteExpenseRepo::new(conn);
        let mut exp = make_expense();
        repo.create(&mut exp).unwrap();

        exp.description = "Updated description".to_string();
        repo.update(&mut exp).unwrap();

        let fetched = repo.get_by_id(exp.id).unwrap();
        assert_eq!(fetched.description, "Updated description");
    }

    #[test]
    fn test_soft_delete() {
        let conn = new_test_db();
        let repo = SqliteExpenseRepo::new(conn);
        let mut exp = make_expense();
        repo.create(&mut exp).unwrap();

        repo.delete(exp.id).unwrap();
        assert!(matches!(repo.get_by_id(exp.id), Err(DomainError::NotFound)));
    }

    #[test]
    fn test_list_with_filter() {
        let conn = new_test_db();
        let repo = SqliteExpenseRepo::new(conn);

        let mut e1 = make_expense();
        e1.expense_number = "E001".to_string();
        e1.category = "software".to_string();
        repo.create(&mut e1).unwrap();

        let mut e2 = make_expense();
        e2.expense_number = "E002".to_string();
        e2.category = "hardware".to_string();
        repo.create(&mut e2).unwrap();

        let filter = ExpenseFilter {
            category: "software".to_string(),
            ..Default::default()
        };
        let (found, total) = repo.list(&filter).unwrap();
        assert_eq!(total, 1);
        assert_eq!(found[0].expense_number, "E001");
    }

    #[test]
    fn test_mark_tax_reviewed() {
        let conn = new_test_db();
        let repo = SqliteExpenseRepo::new(conn);

        let mut exp = make_expense();
        repo.create(&mut exp).unwrap();

        assert!(repo.get_by_id(exp.id).unwrap().tax_reviewed_at.is_none());

        repo.mark_tax_reviewed(&[exp.id]).unwrap();
        assert!(repo.get_by_id(exp.id).unwrap().tax_reviewed_at.is_some());

        repo.unmark_tax_reviewed(&[exp.id]).unwrap();
        assert!(repo.get_by_id(exp.id).unwrap().tax_reviewed_at.is_none());
    }

    #[test]
    fn test_with_items() {
        let conn = new_test_db();
        let repo = SqliteExpenseRepo::new(conn);

        let mut exp = make_expense();
        exp.items = vec![ExpenseItem {
            id: 0,
            expense_id: 0,
            description: "Item 1".to_string(),
            quantity: Amount::from_halere(100),
            unit: "ks".to_string(),
            unit_price: Amount::from_halere(2500),
            vat_rate_percent: 21,
            vat_amount: Amount::from_halere(525),
            total_amount: Amount::from_halere(3025),
            sort_order: 0,
        }];
        repo.create(&mut exp).unwrap();

        let fetched = repo.get_by_id(exp.id).unwrap();
        assert_eq!(fetched.items.len(), 1);
        assert_eq!(fetched.items[0].description, "Item 1");
    }
}
