use crate::helpers::*;
use chrono::{Local, NaiveDate};
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::RecurringExpenseRepo;
use zfaktury_domain::*;

pub struct SqliteRecurringExpenseRepo {
    conn: Mutex<Connection>,
}
impl SqliteRecurringExpenseRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}

fn parse_freq(s: &str) -> Frequency {
    match s {
        "weekly" => Frequency::Weekly,
        "quarterly" => Frequency::Quarterly,
        "yearly" => Frequency::Yearly,
        _ => Frequency::Monthly,
    }
}

fn scan(row: &Row<'_>) -> rusqlite::Result<RecurringExpense> {
    let f: String = row.get("frequency")?;
    let n: String = row.get("next_issue_date")?;
    let e: Option<String> = row.get("end_date")?;
    let c: String = row.get("created_at")?;
    let u: String = row.get("updated_at")?;
    let d: Option<String> = row.get("deleted_at")?;
    Ok(RecurringExpense {
        id: row.get("id")?,
        name: row.get("name")?,
        vendor_id: row.get("vendor_id")?,
        vendor: None,
        category: row.get("category")?,
        description: row.get("description")?,
        amount: Amount::from_halere(row.get::<_, i64>("amount")?),
        currency_code: row.get("currency_code")?,
        exchange_rate: Amount::from_halere(row.get::<_, i64>("exchange_rate")?),
        vat_rate_percent: row.get("vat_rate_percent")?,
        vat_amount: Amount::from_halere(row.get::<_, i64>("vat_amount")?),
        is_tax_deductible: row.get::<_, i32>("is_tax_deductible")? != 0,
        business_percent: row.get("business_percent")?,
        payment_method: row.get("payment_method")?,
        notes: row.get("notes")?,
        frequency: parse_freq(&f),
        next_issue_date: parse_date_or_default(&n),
        end_date: parse_date_optional(e.as_deref()).unwrap_or(None),
        is_active: row.get::<_, i32>("is_active")? != 0,
        created_at: parse_datetime_or_default(&c),
        updated_at: parse_datetime_or_default(&u),
        deleted_at: parse_datetime_optional(d.as_deref()).unwrap_or(None),
    })
}

const COLS: &str = "id,name,vendor_id,category,description,amount,currency_code,exchange_rate,vat_rate_percent,vat_amount,is_tax_deductible,business_percent,payment_method,notes,frequency,next_issue_date,end_date,is_active,created_at,updated_at,deleted_at";

impl RecurringExpenseRepo for SqliteRecurringExpenseRepo {
    fn create(&self, re: &mut RecurringExpense) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        let end = re.end_date.as_ref().map(format_date);
        conn.execute("INSERT INTO recurring_expenses (name,vendor_id,category,description,amount,currency_code,exchange_rate,vat_rate_percent,vat_amount,is_tax_deductible,business_percent,payment_method,notes,frequency,next_issue_date,end_date,is_active,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,?12,?13,?14,?15,?16,?17,?18,?18)",
            params![re.name,re.vendor_id,re.category,re.description,re.amount.halere(),re.currency_code,re.exchange_rate.halere(),re.vat_rate_percent,re.vat_amount.halere(),re.is_tax_deductible as i32,re.business_percent,re.payment_method,re.notes,re.frequency.to_string(),format_date(&re.next_issue_date),end,re.is_active as i32,now])
            .map_err(|e|{log::error!("insert re: {e}");DomainError::InvalidInput})?;
        re.id = conn.last_insert_rowid();
        Ok(())
    }
    fn update(&self, re: &mut RecurringExpense) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        let end = re.end_date.as_ref().map(format_date);
        conn.execute("UPDATE recurring_expenses SET name=?1,vendor_id=?2,category=?3,description=?4,amount=?5,currency_code=?6,exchange_rate=?7,vat_rate_percent=?8,vat_amount=?9,is_tax_deductible=?10,business_percent=?11,payment_method=?12,notes=?13,frequency=?14,next_issue_date=?15,end_date=?16,is_active=?17,updated_at=?18 WHERE id=?19 AND deleted_at IS NULL",
            params![re.name,re.vendor_id,re.category,re.description,re.amount.halere(),re.currency_code,re.exchange_rate.halere(),re.vat_rate_percent,re.vat_amount.halere(),re.is_tax_deductible as i32,re.business_percent,re.payment_method,re.notes,re.frequency.to_string(),format_date(&re.next_issue_date),end,re.is_active as i32,now,re.id])
            .map_err(|e|{log::error!("update re: {e}");DomainError::InvalidInput})?;
        Ok(())
    }
    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        let r = conn.execute("UPDATE recurring_expenses SET deleted_at=?1,updated_at=?1 WHERE id=?2 AND deleted_at IS NULL",params![now,id]).map_err(|e|{log::error!("del re: {e}");DomainError::InvalidInput})?;
        if r == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }
    fn get_by_id(&self, id: i64) -> Result<RecurringExpense, DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.query_row(
            &format!("SELECT {COLS} FROM recurring_expenses WHERE id=?1 AND deleted_at IS NULL"),
            params![id],
            scan,
        )
        .map_err(|e| match e {
            rusqlite::Error::QueryReturnedNoRows => DomainError::NotFound,
            _ => {
                log::error!("q: {e}");
                DomainError::InvalidInput
            }
        })
    }
    fn list(&self, limit: i32, offset: i32) -> Result<(Vec<RecurringExpense>, i64), DomainError> {
        let conn = self.conn.lock().unwrap();
        let total: i64 = conn
            .query_row(
                "SELECT COUNT(*) FROM recurring_expenses WHERE deleted_at IS NULL",
                [],
                |r| r.get(0),
            )
            .map_err(|e| {
                log::error!("cnt: {e}");
                DomainError::InvalidInput
            })?;
        let mut q =
            format!("SELECT {COLS} FROM recurring_expenses WHERE deleted_at IS NULL ORDER BY name");
        let mut param_values: Vec<Box<dyn rusqlite::types::ToSql>> = Vec::new();
        if limit > 0 {
            q.push_str(" LIMIT ?1 OFFSET ?2");
            param_values.push(Box::new(limit as i64));
            param_values.push(Box::new(offset as i64));
        }
        let params_ref: Vec<&dyn rusqlite::types::ToSql> =
            param_values.iter().map(|p| p.as_ref()).collect();
        let mut s = conn.prepare(&q).map_err(|e| {
            log::error!("prep: {e}");
            DomainError::InvalidInput
        })?;
        let list = s
            .query_map(params_ref.as_slice(), scan)
            .map_err(|e| {
                log::error!("list: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("scan: {e}");
                DomainError::InvalidInput
            })?;
        Ok((list, total))
    }
    fn list_active(&self) -> Result<Vec<RecurringExpense>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut s = conn
            .prepare(&format!(
                "SELECT {COLS} FROM recurring_expenses WHERE deleted_at IS NULL AND is_active=1"
            ))
            .map_err(|e| {
                log::error!("prep: {e}");
                DomainError::InvalidInput
            })?;
        s.query_map([], scan)
            .map_err(|e| {
                log::error!("list: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("scan: {e}");
                DomainError::InvalidInput
            })
    }
    fn list_due(&self, as_of_date: NaiveDate) -> Result<Vec<RecurringExpense>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut s = conn.prepare(&format!("SELECT {COLS} FROM recurring_expenses WHERE deleted_at IS NULL AND is_active=1 AND next_issue_date<=?1")).map_err(|e|{log::error!("prep: {e}");DomainError::InvalidInput})?;
        s.query_map(params![format_date(&as_of_date)], scan)
            .map_err(|e| {
                log::error!("list: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("scan: {e}");
                DomainError::InvalidInput
            })
    }
    fn deactivate(&self, id: i64) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        conn.execute(
            "UPDATE recurring_expenses SET is_active=0,updated_at=?1 WHERE id=?2",
            params![now, id],
        )
        .map_err(|e| {
            log::error!("deact: {e}");
            DomainError::InvalidInput
        })?;
        Ok(())
    }
    fn activate(&self, id: i64) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        conn.execute(
            "UPDATE recurring_expenses SET is_active=1,updated_at=?1 WHERE id=?2",
            params![now, id],
        )
        .map_err(|e| {
            log::error!("act: {e}");
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
    fn test_create_and_list() {
        let conn = new_test_db();
        let repo = SqliteRecurringExpenseRepo::new(conn);
        let mut re = RecurringExpense {
            id: 0,
            name: "Monthly hosting".into(),
            vendor_id: None,
            vendor: None,
            category: "software".into(),
            description: "Server".into(),
            amount: Amount::from_halere(50000),
            currency_code: "CZK".into(),
            exchange_rate: Amount::from_halere(100),
            vat_rate_percent: 21,
            vat_amount: Amount::from_halere(10500),
            is_tax_deductible: true,
            business_percent: 100,
            payment_method: "bank_transfer".into(),
            notes: String::new(),
            frequency: Frequency::Monthly,
            next_issue_date: NaiveDate::from_ymd_opt(2025, 2, 1).unwrap(),
            end_date: None,
            is_active: true,
            created_at: Default::default(),
            updated_at: Default::default(),
            deleted_at: None,
        };
        repo.create(&mut re).unwrap();
        let (list, total) = repo.list(0, 0).unwrap();
        assert_eq!(total, 1);
        assert_eq!(list[0].name, "Monthly hosting");
    }
}
