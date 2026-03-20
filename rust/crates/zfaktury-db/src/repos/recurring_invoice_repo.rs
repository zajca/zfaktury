use crate::helpers::*;
use chrono::{Local, NaiveDate};
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::RecurringInvoiceRepo;
use zfaktury_domain::*;

pub struct SqliteRecurringInvoiceRepo {
    conn: Mutex<Connection>,
}
impl SqliteRecurringInvoiceRepo {
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

fn scan_ri(row: &Row<'_>) -> rusqlite::Result<RecurringInvoice> {
    let freq: String = row.get("frequency")?;
    let next: String = row.get("next_issue_date")?;
    let end: Option<String> = row.get("end_date")?;
    let c: String = row.get("created_at")?;
    let u: String = row.get("updated_at")?;
    let d: Option<String> = row.get("deleted_at")?;
    Ok(RecurringInvoice {
        id: row.get("id")?,
        name: row.get("name")?,
        customer_id: row.get("customer_id")?,
        customer: None,
        frequency: parse_freq(&freq),
        next_issue_date: parse_date(&next).unwrap_or_default(),
        end_date: parse_date_optional(end.as_deref()).unwrap_or(None),
        currency_code: row.get("currency_code")?,
        exchange_rate: Amount::from_halere(row.get::<_, i64>("exchange_rate")?),
        payment_method: row.get("payment_method")?,
        bank_account: row.get("bank_account")?,
        bank_code: row.get("bank_code")?,
        iban: row.get("iban")?,
        swift: row.get("swift")?,
        constant_symbol: row.get("constant_symbol")?,
        notes: row.get("notes")?,
        is_active: row.get::<_, i32>("is_active")? != 0,
        items: Vec::new(),
        created_at: parse_datetime(&c).unwrap_or_default(),
        updated_at: parse_datetime(&u).unwrap_or_default(),
        deleted_at: parse_datetime_optional(d.as_deref()).unwrap_or(None),
    })
}

const COLS: &str = "id,name,customer_id,frequency,next_issue_date,end_date,currency_code,exchange_rate,payment_method,bank_account,bank_code,iban,swift,constant_symbol,notes,is_active,created_at,updated_at,deleted_at";

impl RecurringInvoiceRepo for SqliteRecurringInvoiceRepo {
    fn create(&self, ri: &mut RecurringInvoice) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        let end = ri.end_date.as_ref().map(format_date);
        conn.execute(&"INSERT INTO recurring_invoices (name,customer_id,frequency,next_issue_date,end_date,currency_code,exchange_rate,payment_method,bank_account,bank_code,iban,swift,constant_symbol,notes,is_active,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,?12,?13,?14,?15,?16,?16)".to_string(),
            params![ri.name, ri.customer_id, ri.frequency.to_string(), format_date(&ri.next_issue_date), end, ri.currency_code, ri.exchange_rate.halere(), ri.payment_method, ri.bank_account, ri.bank_code, ri.iban, ri.swift, ri.constant_symbol, ri.notes, ri.is_active as i32, now])
            .map_err(|e|{log::error!("insert ri: {e}");DomainError::InvalidInput})?;
        ri.id = conn.last_insert_rowid();
        for item in &mut ri.items {
            item.recurring_invoice_id = ri.id;
            conn.execute("INSERT INTO recurring_invoice_items (recurring_invoice_id,description,quantity,unit,unit_price,vat_rate_percent,sort_order) VALUES (?1,?2,?3,?4,?5,?6,?7)",
                params![ri.id, item.description, item.quantity.halere(), item.unit, item.unit_price.halere(), item.vat_rate_percent, item.sort_order])
                .map_err(|e|{log::error!("insert ri item: {e}");DomainError::InvalidInput})?;
            item.id = conn.last_insert_rowid();
        }
        Ok(())
    }
    fn update(&self, ri: &mut RecurringInvoice) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        let end = ri.end_date.as_ref().map(format_date);
        conn.execute("UPDATE recurring_invoices SET name=?1,customer_id=?2,frequency=?3,next_issue_date=?4,end_date=?5,currency_code=?6,exchange_rate=?7,payment_method=?8,bank_account=?9,bank_code=?10,iban=?11,swift=?12,constant_symbol=?13,notes=?14,is_active=?15,updated_at=?16 WHERE id=?17 AND deleted_at IS NULL",
            params![ri.name, ri.customer_id, ri.frequency.to_string(), format_date(&ri.next_issue_date), end, ri.currency_code, ri.exchange_rate.halere(), ri.payment_method, ri.bank_account, ri.bank_code, ri.iban, ri.swift, ri.constant_symbol, ri.notes, ri.is_active as i32, now, ri.id])
            .map_err(|e|{log::error!("update ri: {e}");DomainError::InvalidInput})?;
        conn.execute(
            "DELETE FROM recurring_invoice_items WHERE recurring_invoice_id=?1",
            params![ri.id],
        )
        .map_err(|e| {
            log::error!("del ri items: {e}");
            DomainError::InvalidInput
        })?;
        for item in &mut ri.items {
            item.recurring_invoice_id = ri.id;
            conn.execute("INSERT INTO recurring_invoice_items (recurring_invoice_id,description,quantity,unit,unit_price,vat_rate_percent,sort_order) VALUES (?1,?2,?3,?4,?5,?6,?7)",
                params![ri.id, item.description, item.quantity.halere(), item.unit, item.unit_price.halere(), item.vat_rate_percent, item.sort_order])
                .map_err(|e|{log::error!("insert ri item: {e}");DomainError::InvalidInput})?;
            item.id = conn.last_insert_rowid();
        }
        Ok(())
    }
    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        let r = conn.execute("UPDATE recurring_invoices SET deleted_at=?1,updated_at=?1 WHERE id=?2 AND deleted_at IS NULL",params![now,id]).map_err(|e|{log::error!("del ri: {e}");DomainError::InvalidInput})?;
        if r == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }
    fn get_by_id(&self, id: i64) -> Result<RecurringInvoice, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut ri = conn
            .query_row(
                &format!(
                    "SELECT {COLS} FROM recurring_invoices WHERE id=?1 AND deleted_at IS NULL"
                ),
                params![id],
                scan_ri,
            )
            .map_err(|e| match e {
                rusqlite::Error::QueryReturnedNoRows => DomainError::NotFound,
                _ => {
                    log::error!("q ri: {e}");
                    DomainError::InvalidInput
                }
            })?;
        let mut s = conn.prepare("SELECT id,recurring_invoice_id,description,quantity,unit,unit_price,vat_rate_percent,sort_order FROM recurring_invoice_items WHERE recurring_invoice_id=?1 ORDER BY sort_order")
            .map_err(|e|{log::error!("prep ri items: {e}");DomainError::InvalidInput})?;
        ri.items = s
            .query_map(params![id], |row| {
                Ok(RecurringInvoiceItem {
                    id: row.get("id")?,
                    recurring_invoice_id: row.get("recurring_invoice_id")?,
                    description: row.get("description")?,
                    quantity: Amount::from_halere(row.get::<_, i64>("quantity")?),
                    unit: row.get("unit")?,
                    unit_price: Amount::from_halere(row.get::<_, i64>("unit_price")?),
                    vat_rate_percent: row.get("vat_rate_percent")?,
                    sort_order: row.get("sort_order")?,
                })
            })
            .map_err(|e| {
                log::error!("q ri items: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("scan ri items: {e}");
                DomainError::InvalidInput
            })?;
        Ok(ri)
    }
    fn list(&self) -> Result<Vec<RecurringInvoice>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut s = conn
            .prepare(&format!(
                "SELECT {COLS} FROM recurring_invoices WHERE deleted_at IS NULL ORDER BY name"
            ))
            .map_err(|e| {
                log::error!("prep ri list: {e}");
                DomainError::InvalidInput
            })?;
        s.query_map([], scan_ri)
            .map_err(|e| {
                log::error!("list ri: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("scan: {e}");
                DomainError::InvalidInput
            })
    }
    fn list_due(&self, date: NaiveDate) -> Result<Vec<RecurringInvoice>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut s = conn.prepare(&format!("SELECT {COLS} FROM recurring_invoices WHERE deleted_at IS NULL AND is_active=1 AND next_issue_date<=?1")).map_err(|e|{log::error!("prep: {e}");DomainError::InvalidInput})?;
        s.query_map(params![format_date(&date)], scan_ri)
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
        conn.execute("UPDATE recurring_invoices SET is_active=0,updated_at=?1 WHERE id=?2 AND deleted_at IS NULL",params![now,id]).map_err(|e|{log::error!("deactivate: {e}");DomainError::InvalidInput})?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;
    #[test]
    fn test_create_and_get() {
        let conn = new_test_db();
        let now = format_datetime(&Local::now().naive_local());
        conn.execute("INSERT INTO contacts (type,name,country,created_at,updated_at) VALUES ('company','C','CZ',?1,?1)",params![now]).unwrap();
        let cid = conn.last_insert_rowid();
        let repo = SqliteRecurringInvoiceRepo::new(conn);
        let mut ri = RecurringInvoice {
            id: 0,
            name: "Monthly".into(),
            customer_id: cid,
            customer: None,
            frequency: Frequency::Monthly,
            next_issue_date: NaiveDate::from_ymd_opt(2025, 2, 1).unwrap(),
            end_date: None,
            currency_code: "CZK".into(),
            exchange_rate: Amount::from_halere(100),
            payment_method: "bank_transfer".into(),
            bank_account: String::new(),
            bank_code: String::new(),
            iban: String::new(),
            swift: String::new(),
            constant_symbol: String::new(),
            notes: String::new(),
            is_active: true,
            items: Vec::new(),
            created_at: Default::default(),
            updated_at: Default::default(),
            deleted_at: None,
        };
        repo.create(&mut ri).unwrap();
        assert!(ri.id > 0);
        let f = repo.get_by_id(ri.id).unwrap();
        assert_eq!(f.name, "Monthly");
    }
}
