use std::sync::Mutex;

use chrono::Local;
use rusqlite::{Connection, Row, params};
use zfaktury_core::repository::traits::InvoiceRepo;
use zfaktury_domain::*;

use crate::helpers::*;

pub struct SqliteInvoiceRepo {
    conn: Mutex<Connection>,
}

impl SqliteInvoiceRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}

fn parse_invoice_type(s: &str) -> InvoiceType {
    match s {
        "proforma" => InvoiceType::Proforma,
        "credit_note" => InvoiceType::CreditNote,
        _ => InvoiceType::Regular,
    }
}

fn parse_invoice_status(s: &str) -> InvoiceStatus {
    match s {
        "sent" => InvoiceStatus::Sent,
        "paid" => InvoiceStatus::Paid,
        "overdue" => InvoiceStatus::Overdue,
        "cancelled" => InvoiceStatus::Cancelled,
        _ => InvoiceStatus::Draft,
    }
}

fn parse_relation_type(s: &str) -> RelationType {
    match s {
        "settlement" => RelationType::Settlement,
        "credit_note" => RelationType::CreditNote,
        _ => RelationType::None,
    }
}

fn scan_invoice_core(row: &Row<'_>) -> rusqlite::Result<Invoice> {
    let type_str: String = row.get("type")?;
    let status_str: String = row.get("status")?;
    let issue_date_str: String = row.get("issue_date")?;
    let due_date_str: String = row.get("due_date")?;
    let delivery_date_str: Option<String> = row.get("delivery_date")?;
    let created_at_str: String = row.get("created_at")?;
    let updated_at_str: String = row.get("updated_at")?;
    let deleted_at_str: Option<String> = row.get("deleted_at")?;
    let sent_at_str: Option<String> = row.get("sent_at")?;
    let paid_at_str: Option<String> = row.get("paid_at")?;
    let related_id: Option<i64> = row.get("related_invoice_id")?;
    let relation_type_str: String = row
        .get::<_, Option<String>>("relation_type")?
        .unwrap_or_default();
    let seq_id: Option<i64> = row.get("sequence_id")?;

    Ok(Invoice {
        id: row.get("id")?,
        sequence_id: seq_id.unwrap_or(0),
        invoice_number: row.get("invoice_number")?,
        invoice_type: parse_invoice_type(&type_str),
        status: parse_invoice_status(&status_str),
        issue_date: parse_date_or_default(&issue_date_str),
        due_date: parse_date_or_default(&due_date_str),
        delivery_date: parse_date_or_default(&delivery_date_str.unwrap_or_default()),
        variable_symbol: row
            .get::<_, Option<String>>("variable_symbol")?
            .unwrap_or_default(),
        constant_symbol: row
            .get::<_, Option<String>>("constant_symbol")?
            .unwrap_or_default(),
        customer_id: row.get("customer_id")?,
        customer: None,
        currency_code: row.get("currency_code")?,
        exchange_rate: Amount::from_halere(row.get::<_, i64>("exchange_rate")?),
        payment_method: row.get("payment_method")?,
        bank_account: row
            .get::<_, Option<String>>("bank_account")?
            .unwrap_or_default(),
        bank_code: row
            .get::<_, Option<String>>("bank_code")?
            .unwrap_or_default(),
        iban: row.get::<_, Option<String>>("iban")?.unwrap_or_default(),
        swift: row.get::<_, Option<String>>("swift")?.unwrap_or_default(),
        subtotal_amount: Amount::from_halere(row.get::<_, i64>("subtotal_amount")?),
        vat_amount: Amount::from_halere(row.get::<_, i64>("vat_amount")?),
        total_amount: Amount::from_halere(row.get::<_, i64>("total_amount")?),
        paid_amount: Amount::from_halere(row.get::<_, i64>("paid_amount")?),
        notes: row.get::<_, Option<String>>("notes")?.unwrap_or_default(),
        internal_notes: row
            .get::<_, Option<String>>("internal_notes")?
            .unwrap_or_default(),
        related_invoice_id: related_id,
        relation_type: parse_relation_type(&relation_type_str),
        sent_at: parse_datetime_optional(sent_at_str.as_deref()).unwrap_or(None),
        paid_at: parse_datetime_optional(paid_at_str.as_deref()).unwrap_or(None),
        items: Vec::new(),
        created_at: parse_datetime_or_default(&created_at_str),
        updated_at: parse_datetime_or_default(&updated_at_str),
        deleted_at: parse_datetime_optional(deleted_at_str.as_deref()).unwrap_or(None),
    })
}

fn scan_invoice_item(row: &Row<'_>) -> rusqlite::Result<InvoiceItem> {
    Ok(InvoiceItem {
        id: row.get("id")?,
        invoice_id: row.get("invoice_id")?,
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

const INV_COLS: &str = "id, sequence_id, invoice_number, type, status, \
    issue_date, due_date, delivery_date, variable_symbol, constant_symbol, \
    customer_id, currency_code, exchange_rate, \
    payment_method, bank_account, bank_code, iban, swift, \
    subtotal_amount, vat_amount, total_amount, paid_amount, \
    notes, internal_notes, sent_at, paid_at, \
    related_invoice_id, relation_type, \
    created_at, updated_at, deleted_at";

fn fetch_items(conn: &Connection, invoice_id: i64) -> Result<Vec<InvoiceItem>, DomainError> {
    let mut stmt = conn.prepare(
        "SELECT id, invoice_id, description, quantity, unit, unit_price, vat_rate_percent, vat_amount, total_amount, sort_order \
         FROM invoice_items WHERE invoice_id = ?1 ORDER BY sort_order ASC"
    ).map_err(|e| { log::error!("preparing invoice items: {e}"); DomainError::InvalidInput })?;
    let items = stmt
        .query_map(params![invoice_id], scan_invoice_item)
        .map_err(|e| {
            log::error!("querying invoice items: {e}");
            DomainError::InvalidInput
        })?
        .collect::<Result<Vec<_>, _>>()
        .map_err(|e| {
            log::error!("scanning invoice items: {e}");
            DomainError::InvalidInput
        })?;
    Ok(items)
}

fn insert_items(
    conn: &Connection,
    invoice_id: i64,
    items: &mut [InvoiceItem],
) -> Result<(), DomainError> {
    for (i, item) in items.iter_mut().enumerate() {
        item.invoice_id = invoice_id;
        conn.execute(
            "INSERT INTO invoice_items (invoice_id, description, quantity, unit, unit_price, vat_rate_percent, vat_amount, total_amount, sort_order) \
             VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9)",
            params![
                invoice_id, item.description, item.quantity.halere(), item.unit,
                item.unit_price.halere(), item.vat_rate_percent, item.vat_amount.halere(),
                item.total_amount.halere(), item.sort_order,
            ],
        ).map_err(|e| { log::error!("inserting invoice item {i}: {e}"); DomainError::InvalidInput })?;
        item.id = conn.last_insert_rowid();
    }
    Ok(())
}

impl InvoiceRepo for SqliteInvoiceRepo {
    fn create(&self, invoice: &mut Invoice) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = Local::now().naive_local();
        invoice.created_at = now;
        invoice.updated_at = now;
        let now_str = format_datetime(&now);

        let seq_id: Option<i64> = if invoice.sequence_id > 0 {
            Some(invoice.sequence_id)
        } else {
            None
        };
        let sent_at_str = format_datetime_opt(&invoice.sent_at);
        let paid_at_str = format_datetime_opt(&invoice.paid_at);

        let tx = conn.unchecked_transaction().map_err(|e| {
            log::error!("begin tx: {e}");
            DomainError::InvalidInput
        })?;

        tx.execute(
            &format!(
                "INSERT INTO invoices ({INV_COLS}) VALUES (\
                NULL, ?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13, ?14, ?15, ?16, ?17, \
                ?18, ?19, ?20, ?21, ?22, ?23, ?24, ?25, ?26, ?27, ?28, ?29, NULL)"
            ),
            params![
                seq_id,
                invoice.invoice_number,
                invoice.invoice_type.to_string(),
                invoice.status.to_string(),
                format_date(&invoice.issue_date),
                format_date(&invoice.due_date),
                format_date(&invoice.delivery_date),
                invoice.variable_symbol,
                invoice.constant_symbol,
                invoice.customer_id,
                invoice.currency_code,
                invoice.exchange_rate.halere(),
                invoice.payment_method,
                invoice.bank_account,
                invoice.bank_code,
                invoice.iban,
                invoice.swift,
                invoice.subtotal_amount.halere(),
                invoice.vat_amount.halere(),
                invoice.total_amount.halere(),
                invoice.paid_amount.halere(),
                invoice.notes,
                invoice.internal_notes,
                sent_at_str,
                paid_at_str,
                invoice.related_invoice_id,
                invoice.relation_type.to_string(),
                now_str,
                now_str,
            ],
        )
        .map_err(|e| {
            log::error!("inserting invoice: {e}");
            DomainError::InvalidInput
        })?;

        invoice.id = tx.last_insert_rowid();
        insert_items(&tx, invoice.id, &mut invoice.items)?;

        tx.commit().map_err(|e| {
            log::error!("committing invoice: {e}");
            DomainError::InvalidInput
        })?;
        Ok(())
    }

    fn update(&self, invoice: &mut Invoice) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = Local::now().naive_local();
        invoice.updated_at = now;
        let now_str = format_datetime(&now);

        let seq_id: Option<i64> = if invoice.sequence_id > 0 {
            Some(invoice.sequence_id)
        } else {
            None
        };
        let sent_at_str = format_datetime_opt(&invoice.sent_at);
        let paid_at_str = format_datetime_opt(&invoice.paid_at);

        let tx = conn.unchecked_transaction().map_err(|e| {
            log::error!("begin tx: {e}");
            DomainError::InvalidInput
        })?;

        tx.execute(
            "UPDATE invoices SET sequence_id = ?1, invoice_number = ?2, type = ?3, status = ?4, \
             issue_date = ?5, due_date = ?6, delivery_date = ?7, variable_symbol = ?8, constant_symbol = ?9, \
             customer_id = ?10, currency_code = ?11, exchange_rate = ?12, \
             payment_method = ?13, bank_account = ?14, bank_code = ?15, iban = ?16, swift = ?17, \
             subtotal_amount = ?18, vat_amount = ?19, total_amount = ?20, paid_amount = ?21, \
             notes = ?22, internal_notes = ?23, sent_at = ?24, paid_at = ?25, \
             related_invoice_id = ?26, relation_type = ?27, updated_at = ?28 \
             WHERE id = ?29 AND deleted_at IS NULL",
            params![
                seq_id, invoice.invoice_number, invoice.invoice_type.to_string(), invoice.status.to_string(),
                format_date(&invoice.issue_date), format_date(&invoice.due_date), format_date(&invoice.delivery_date),
                invoice.variable_symbol, invoice.constant_symbol, invoice.customer_id,
                invoice.currency_code, invoice.exchange_rate.halere(),
                invoice.payment_method, invoice.bank_account, invoice.bank_code, invoice.iban, invoice.swift,
                invoice.subtotal_amount.halere(), invoice.vat_amount.halere(), invoice.total_amount.halere(), invoice.paid_amount.halere(),
                invoice.notes, invoice.internal_notes, sent_at_str, paid_at_str,
                invoice.related_invoice_id, invoice.relation_type.to_string(), now_str,
                invoice.id,
            ],
        ).map_err(|e| { log::error!("updating invoice: {e}"); DomainError::InvalidInput })?;

        tx.execute(
            "DELETE FROM invoice_items WHERE invoice_id = ?1",
            params![invoice.id],
        )
        .map_err(|e| {
            log::error!("deleting old invoice items: {e}");
            DomainError::InvalidInput
        })?;
        insert_items(&tx, invoice.id, &mut invoice.items)?;

        tx.commit().map_err(|e| {
            log::error!("committing invoice update: {e}");
            DomainError::InvalidInput
        })?;
        Ok(())
    }

    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now_str = format_datetime(&Local::now().naive_local());
        let rows = conn.execute(
            "UPDATE invoices SET deleted_at = ?1, updated_at = ?1 WHERE id = ?2 AND deleted_at IS NULL",
            params![now_str, id],
        ).map_err(|e| { log::error!("deleting invoice: {e}"); DomainError::InvalidInput })?;
        if rows == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }

    fn get_by_id(&self, id: i64) -> Result<Invoice, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut inv = conn
            .query_row(
                &format!("SELECT {INV_COLS} FROM invoices WHERE id = ?1 AND deleted_at IS NULL"),
                params![id],
                scan_invoice_core,
            )
            .map_err(|e| match e {
                rusqlite::Error::QueryReturnedNoRows => DomainError::NotFound,
                _ => {
                    log::error!("querying invoice {id}: {e}");
                    DomainError::InvalidInput
                }
            })?;

        inv.items = fetch_items(&conn, id)?;
        Ok(inv)
    }

    fn list(&self, filter: &InvoiceFilter) -> Result<(Vec<Invoice>, i64), DomainError> {
        let conn = self.conn.lock().unwrap();

        let mut where_clause = String::from("i.deleted_at IS NULL");
        let mut param_values: Vec<Box<dyn rusqlite::types::ToSql>> = Vec::new();

        if let Some(ref status) = filter.status {
            let idx = param_values.len() + 1;
            where_clause.push_str(&format!(" AND i.status = ?{idx}"));
            param_values.push(Box::new(status.to_string()));
        }
        if let Some(ref inv_type) = filter.invoice_type {
            let idx = param_values.len() + 1;
            where_clause.push_str(&format!(" AND i.type = ?{idx}"));
            param_values.push(Box::new(inv_type.to_string()));
        }
        if let Some(cid) = filter.customer_id {
            let idx = param_values.len() + 1;
            where_clause.push_str(&format!(" AND i.customer_id = ?{idx}"));
            param_values.push(Box::new(cid));
        }
        if let Some(ref from) = filter.date_from {
            let idx = param_values.len() + 1;
            where_clause.push_str(&format!(" AND i.issue_date >= ?{idx}"));
            param_values.push(Box::new(format_date(from)));
        }
        if let Some(ref to) = filter.date_to {
            let idx = param_values.len() + 1;
            where_clause.push_str(&format!(" AND i.issue_date <= ?{idx}"));
            param_values.push(Box::new(format_date(to)));
        }
        if !filter.search.is_empty() {
            let idx = param_values.len() + 1;
            where_clause.push_str(&format!(
                " AND (i.invoice_number LIKE ?{idx} OR i.variable_symbol LIKE ?{idx})"
            ));
            param_values.push(Box::new(format!("%{}%", filter.search)));
        }

        let params_ref: Vec<&dyn rusqlite::types::ToSql> =
            param_values.iter().map(|p| p.as_ref()).collect();

        let count_query = format!("SELECT COUNT(*) FROM invoices i WHERE {where_clause}");
        let total: i64 = conn
            .query_row(&count_query, params_ref.as_slice(), |row| row.get(0))
            .map_err(|e| {
                log::error!("counting invoices: {e}");
                DomainError::InvalidInput
            })?;

        let inv_cols_prefixed = INV_COLS
            .split(", ")
            .map(|c| format!("i.{c}"))
            .collect::<Vec<_>>()
            .join(", ");
        let mut query = format!(
            "SELECT {inv_cols_prefixed} FROM invoices i WHERE {where_clause} ORDER BY i.issue_date DESC"
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
            log::error!("preparing invoice list: {e}");
            DomainError::InvalidInput
        })?;
        let invoices = stmt
            .query_map(params_ref2.as_slice(), scan_invoice_core)
            .map_err(|e| {
                log::error!("listing invoices: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("scanning invoices: {e}");
                DomainError::InvalidInput
            })?;

        Ok((invoices, total))
    }

    fn update_status(&self, id: i64, status: &str) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now_str = format_datetime(&Local::now().naive_local());
        let rows = conn.execute(
            "UPDATE invoices SET status = ?1, updated_at = ?2 WHERE id = ?3 AND deleted_at IS NULL",
            params![status, now_str, id],
        ).map_err(|e| { log::error!("updating status: {e}"); DomainError::InvalidInput })?;
        if rows == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }

    fn get_next_number(&self, sequence_id: i64) -> Result<String, DomainError> {
        let conn = self.conn.lock().unwrap();
        let tx = conn.unchecked_transaction().map_err(|e| {
            log::error!("begin tx: {e}");
            DomainError::InvalidInput
        })?;

        let (prefix, next_number, year): (String, i32, i32) = tx
            .query_row(
                "SELECT prefix, next_number, year FROM invoice_sequences WHERE id = ?1",
                params![sequence_id],
                |row| Ok((row.get(0)?, row.get(1)?, row.get(2)?)),
            )
            .map_err(|e| match e {
                rusqlite::Error::QueryReturnedNoRows => DomainError::NotFound,
                _ => {
                    log::error!("querying sequence: {e}");
                    DomainError::InvalidInput
                }
            })?;

        let number = format!("{prefix}{year}{next_number:04}");

        tx.execute(
            "UPDATE invoice_sequences SET next_number = next_number + 1 WHERE id = ?1",
            params![sequence_id],
        )
        .map_err(|e| {
            log::error!("incrementing sequence: {e}");
            DomainError::InvalidInput
        })?;

        tx.commit().map_err(|e| {
            log::error!("committing sequence: {e}");
            DomainError::InvalidInput
        })?;
        Ok(number)
    }

    fn get_related_invoices(&self, invoice_id: i64) -> Result<Vec<Invoice>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut stmt = conn.prepare(&format!(
            "SELECT {INV_COLS} FROM invoices WHERE related_invoice_id = ?1 AND deleted_at IS NULL ORDER BY created_at ASC"
        )).map_err(|e| { log::error!("preparing related invoices: {e}"); DomainError::InvalidInput })?;
        let invoices = stmt
            .query_map(params![invoice_id], scan_invoice_core)
            .map_err(|e| {
                log::error!("querying related invoices: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("scanning related invoices: {e}");
                DomainError::InvalidInput
            })?;
        Ok(invoices)
    }

    fn find_by_related_invoice(
        &self,
        related_id: i64,
        relation_type: &str,
    ) -> Result<Option<Invoice>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let result = conn.query_row(&format!(
            "SELECT {INV_COLS} FROM invoices WHERE related_invoice_id = ?1 AND relation_type = ?2 AND deleted_at IS NULL LIMIT 1"
        ), params![related_id, relation_type], scan_invoice_core);
        match result {
            Ok(inv) => Ok(Some(inv)),
            Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
            Err(e) => {
                log::error!("finding by related: {e}");
                Err(DomainError::InvalidInput)
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;
    use chrono::NaiveDate;

    fn create_test_contact(conn: &Connection) -> i64 {
        let now = format_datetime(&Local::now().naive_local());
        conn.execute(
            "INSERT INTO contacts (type, name, country, created_at, updated_at) VALUES ('company', 'Test Customer', 'CZ', ?1, ?1)",
            params![now],
        ).unwrap();
        conn.last_insert_rowid()
    }

    fn make_invoice(customer_id: i64) -> Invoice {
        Invoice {
            id: 0,
            sequence_id: 0,
            invoice_number: "FV20250001".to_string(),
            invoice_type: InvoiceType::Regular,
            status: InvoiceStatus::Draft,
            issue_date: NaiveDate::from_ymd_opt(2025, 1, 15).unwrap(),
            due_date: NaiveDate::from_ymd_opt(2025, 1, 29).unwrap(),
            delivery_date: NaiveDate::from_ymd_opt(2025, 1, 15).unwrap(),
            variable_symbol: "20250001".to_string(),
            constant_symbol: String::new(),
            customer_id,
            customer: None,
            currency_code: "CZK".to_string(),
            exchange_rate: Amount::from_halere(100),
            payment_method: "bank_transfer".to_string(),
            bank_account: String::new(),
            bank_code: String::new(),
            iban: String::new(),
            swift: String::new(),
            subtotal_amount: Amount::from_halere(10000),
            vat_amount: Amount::from_halere(2100),
            total_amount: Amount::from_halere(12100),
            paid_amount: Amount::ZERO,
            notes: String::new(),
            internal_notes: String::new(),
            related_invoice_id: None,
            relation_type: RelationType::None,
            sent_at: None,
            paid_at: None,
            items: vec![InvoiceItem {
                id: 0,
                invoice_id: 0,
                description: "Service".to_string(),
                quantity: Amount::from_halere(100),
                unit: "ks".to_string(),
                unit_price: Amount::from_halere(10000),
                vat_rate_percent: 21,
                vat_amount: Amount::from_halere(2100),
                total_amount: Amount::from_halere(12100),
                sort_order: 0,
            }],
            created_at: Default::default(),
            updated_at: Default::default(),
            deleted_at: None,
        }
    }

    #[test]
    fn test_create_and_get_by_id() {
        let conn = new_test_db();
        let cid = create_test_contact(&conn);
        let repo = SqliteInvoiceRepo::new(conn);

        let mut inv = make_invoice(cid);
        repo.create(&mut inv).unwrap();
        assert!(inv.id > 0);
        assert!(inv.items[0].id > 0);

        let fetched = repo.get_by_id(inv.id).unwrap();
        assert_eq!(fetched.invoice_number, "FV20250001");
        assert_eq!(fetched.items.len(), 1);
        assert_eq!(fetched.items[0].description, "Service");
        assert_eq!(fetched.total_amount, Amount::from_halere(12100));
    }

    #[test]
    fn test_update_invoice() {
        let conn = new_test_db();
        let cid = create_test_contact(&conn);
        let repo = SqliteInvoiceRepo::new(conn);

        let mut inv = make_invoice(cid);
        repo.create(&mut inv).unwrap();

        inv.notes = "Updated notes".to_string();
        inv.items = vec![InvoiceItem {
            id: 0,
            invoice_id: inv.id,
            description: "New item".to_string(),
            quantity: Amount::from_halere(200),
            unit: "ks".to_string(),
            unit_price: Amount::from_halere(5000),
            vat_rate_percent: 21,
            vat_amount: Amount::from_halere(2100),
            total_amount: Amount::from_halere(12100),
            sort_order: 0,
        }];
        repo.update(&mut inv).unwrap();

        let fetched = repo.get_by_id(inv.id).unwrap();
        assert_eq!(fetched.notes, "Updated notes");
        assert_eq!(fetched.items.len(), 1);
        assert_eq!(fetched.items[0].description, "New item");
    }

    #[test]
    fn test_soft_delete() {
        let conn = new_test_db();
        let cid = create_test_contact(&conn);
        let repo = SqliteInvoiceRepo::new(conn);

        let mut inv = make_invoice(cid);
        repo.create(&mut inv).unwrap();

        repo.delete(inv.id).unwrap();
        assert!(matches!(repo.get_by_id(inv.id), Err(DomainError::NotFound)));
    }

    #[test]
    fn test_list_with_filter() {
        let conn = new_test_db();
        let cid = create_test_contact(&conn);
        let repo = SqliteInvoiceRepo::new(conn);

        let mut inv1 = make_invoice(cid);
        inv1.invoice_number = "FV20250001".to_string();
        repo.create(&mut inv1).unwrap();

        let mut inv2 = make_invoice(cid);
        inv2.invoice_number = "FV20250002".to_string();
        inv2.status = InvoiceStatus::Paid;
        repo.create(&mut inv2).unwrap();

        // List all.
        let (all, total) = repo.list(&InvoiceFilter::default()).unwrap();
        assert_eq!(total, 2);
        assert_eq!(all.len(), 2);

        // Filter by status.
        let filter = InvoiceFilter {
            status: Some(InvoiceStatus::Paid),
            ..Default::default()
        };
        let (paid, total) = repo.list(&filter).unwrap();
        assert_eq!(total, 1);
        assert_eq!(paid[0].invoice_number, "FV20250002");
    }

    #[test]
    fn test_update_status() {
        let conn = new_test_db();
        let cid = create_test_contact(&conn);
        let repo = SqliteInvoiceRepo::new(conn);

        let mut inv = make_invoice(cid);
        repo.create(&mut inv).unwrap();

        repo.update_status(inv.id, "sent").unwrap();
        let fetched = repo.get_by_id(inv.id).unwrap();
        assert_eq!(fetched.status, InvoiceStatus::Sent);
    }

    #[test]
    fn test_get_next_number() {
        let conn = new_test_db();
        // Create a sequence.
        conn.execute(
            "INSERT INTO invoice_sequences (prefix, next_number, year, format_pattern) VALUES ('FV', 1, 2025, '')",
            [],
        ).unwrap();
        let seq_id = conn.last_insert_rowid();

        let repo = SqliteInvoiceRepo::new(conn);
        let num1 = repo.get_next_number(seq_id).unwrap();
        assert_eq!(num1, "FV20250001");

        let num2 = repo.get_next_number(seq_id).unwrap();
        assert_eq!(num2, "FV20250002");
    }
}
