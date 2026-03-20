use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::SecurityTransactionRepo;
use zfaktury_domain::*;
pub struct SqliteSecurityTransactionRepo {
    conn: Mutex<Connection>,
}
impl SqliteSecurityTransactionRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
fn pat(s: &str) -> AssetType {
    match s {
        "stock" => AssetType::Stock,
        "etf" => AssetType::ETF,
        "bond" => AssetType::Bond,
        "fund" => AssetType::Fund,
        "crypto" => AssetType::Crypto,
        _ => AssetType::Other,
    }
}
fn ptt(s: &str) -> TransactionType {
    match s {
        "sell" => TransactionType::Sell,
        _ => TransactionType::Buy,
    }
}
fn scan(row: &Row<'_>) -> rusqlite::Result<SecurityTransaction> {
    let at: String = row.get("asset_type")?;
    let tt: String = row.get("transaction_type")?;
    let td: String = row.get("transaction_date")?;
    let c: String = row.get("created_at")?;
    let u: String = row.get("updated_at")?;
    Ok(SecurityTransaction {
        id: row.get("id")?,
        year: row.get("year")?,
        document_id: row.get("document_id")?,
        asset_type: pat(&at),
        asset_name: row.get("asset_name")?,
        isin: row.get("isin")?,
        transaction_type: ptt(&tt),
        transaction_date: parse_date_or_default(&td),
        quantity: row.get("quantity")?,
        unit_price: Amount::from_halere(row.get::<_, i64>("unit_price")?),
        total_amount: Amount::from_halere(row.get::<_, i64>("total_amount")?),
        fees: Amount::from_halere(row.get::<_, i64>("fees")?),
        currency_code: row.get("currency_code")?,
        exchange_rate: row.get("exchange_rate")?,
        cost_basis: Amount::from_halere(row.get::<_, i64>("cost_basis")?),
        computed_gain: Amount::from_halere(row.get::<_, i64>("computed_gain")?),
        time_test_exempt: row.get::<_, i32>("time_test_exempt")? != 0,
        exempt_amount: Amount::from_halere(row.get::<_, i64>("exempt_amount")?),
        created_at: parse_datetime_or_default(&c),
        updated_at: parse_datetime_or_default(&u),
    })
}
impl SecurityTransactionRepo for SqliteSecurityTransactionRepo {
    fn create(&self, tx: &mut SecurityTransaction) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("INSERT INTO security_transactions (year,document_id,asset_type,asset_name,isin,transaction_type,transaction_date,quantity,unit_price,total_amount,fees,currency_code,exchange_rate,cost_basis,computed_gain,time_test_exempt,exempt_amount,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,?12,?13,?14,?15,?16,?17,?18,?18)",params![tx.year,tx.document_id,tx.asset_type.to_string(),tx.asset_name,tx.isin,tx.transaction_type.to_string(),format_date(&tx.transaction_date),tx.quantity,tx.unit_price.halere(),tx.total_amount.halere(),tx.fees.halere(),tx.currency_code,tx.exchange_rate,tx.cost_basis.halere(),tx.computed_gain.halere(),tx.time_test_exempt as i32,tx.exempt_amount.halere(),n]).map_err(|e|{log::error!("ins: {e}");DomainError::InvalidInput})?;
        tx.id = c.last_insert_rowid();
        Ok(())
    }
    fn update(&self, tx: &mut SecurityTransaction) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("UPDATE security_transactions SET asset_type=?1,asset_name=?2,isin=?3,transaction_type=?4,transaction_date=?5,quantity=?6,unit_price=?7,total_amount=?8,fees=?9,currency_code=?10,exchange_rate=?11,cost_basis=?12,computed_gain=?13,time_test_exempt=?14,exempt_amount=?15,updated_at=?16 WHERE id=?17",params![tx.asset_type.to_string(),tx.asset_name,tx.isin,tx.transaction_type.to_string(),format_date(&tx.transaction_date),tx.quantity,tx.unit_price.halere(),tx.total_amount.halere(),tx.fees.halere(),tx.currency_code,tx.exchange_rate,tx.cost_basis.halere(),tx.computed_gain.halere(),tx.time_test_exempt as i32,tx.exempt_amount.halere(),n,tx.id]).map_err(|e|{log::error!("upd: {e}");DomainError::InvalidInput})?;
        Ok(())
    }
    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let r = c
            .execute("DELETE FROM security_transactions WHERE id=?1", params![id])
            .map_err(|e| {
                log::error!("del: {e}");
                DomainError::InvalidInput
            })?;
        if r == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }
    fn get_by_id(&self, id: i64) -> Result<SecurityTransaction, DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row(
            "SELECT * FROM security_transactions WHERE id=?1",
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
    fn list_by_year(&self, year: i32) -> Result<Vec<SecurityTransaction>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s = c
            .prepare("SELECT * FROM security_transactions WHERE year=?1 ORDER BY transaction_date")
            .map_err(|e| {
                log::error!("p: {e}");
                DomainError::InvalidInput
            })?;
        s.query_map(params![year], scan)
            .map_err(|e| {
                log::error!("l: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("s: {e}");
                DomainError::InvalidInput
            })
    }
    fn list_by_document_id(
        &self,
        document_id: i64,
    ) -> Result<Vec<SecurityTransaction>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s = c
            .prepare("SELECT * FROM security_transactions WHERE document_id=?1")
            .map_err(|e| {
                log::error!("p: {e}");
                DomainError::InvalidInput
            })?;
        s.query_map(params![document_id], scan)
            .map_err(|e| {
                log::error!("l: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("s: {e}");
                DomainError::InvalidInput
            })
    }
    fn list_buys_for_fifo(
        &self,
        asset_name: &str,
        asset_type: &str,
    ) -> Result<Vec<SecurityTransaction>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s=c.prepare("SELECT * FROM security_transactions WHERE asset_name=?1 AND asset_type=?2 AND transaction_type='buy' ORDER BY transaction_date").map_err(|e|{log::error!("p: {e}");DomainError::InvalidInput})?;
        s.query_map(params![asset_name, asset_type], scan)
            .map_err(|e| {
                log::error!("l: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("s: {e}");
                DomainError::InvalidInput
            })
    }
    fn list_sells_by_year(&self, year: i32) -> Result<Vec<SecurityTransaction>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s=c.prepare("SELECT * FROM security_transactions WHERE year=?1 AND transaction_type='sell' ORDER BY transaction_date").map_err(|e|{log::error!("p: {e}");DomainError::InvalidInput})?;
        s.query_map(params![year], scan)
            .map_err(|e| {
                log::error!("l: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("s: {e}");
                DomainError::InvalidInput
            })
    }
    fn update_fifo_results(
        &self,
        id: i64,
        cost_basis: Amount,
        computed_gain: Amount,
        exempt_amount: Amount,
        time_test_exempt: bool,
    ) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("UPDATE security_transactions SET cost_basis=?1,computed_gain=?2,exempt_amount=?3,time_test_exempt=?4,updated_at=?5 WHERE id=?6",params![cost_basis.halere(),computed_gain.halere(),exempt_amount.halere(),time_test_exempt as i32,n,id]).map_err(|e|{log::error!("upd fifo: {e}");DomainError::InvalidInput})?;
        Ok(())
    }
    fn delete_by_document_id(&self, document_id: i64) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        c.execute(
            "DELETE FROM security_transactions WHERE document_id=?1",
            params![document_id],
        )
        .map_err(|e| {
            log::error!("del: {e}");
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
    #[test]
    fn test_create() {
        let c = new_test_db();
        let r = SqliteSecurityTransactionRepo::new(c);
        let mut t = SecurityTransaction {
            id: 0,
            year: 2025,
            document_id: None,
            asset_type: AssetType::Stock,
            asset_name: "AAPL".into(),
            isin: "US0378331005".into(),
            transaction_type: TransactionType::Buy,
            transaction_date: NaiveDate::from_ymd_opt(2025, 1, 15).unwrap(),
            quantity: 100,
            unit_price: Amount::from_halere(15000),
            total_amount: Amount::from_halere(1500000),
            fees: Amount::from_halere(100),
            currency_code: "USD".into(),
            exchange_rate: 10000,
            cost_basis: Amount::ZERO,
            computed_gain: Amount::ZERO,
            time_test_exempt: false,
            exempt_amount: Amount::ZERO,
            created_at: Default::default(),
            updated_at: Default::default(),
        };
        r.create(&mut t).unwrap();
        assert!(t.id > 0);
    }
}
