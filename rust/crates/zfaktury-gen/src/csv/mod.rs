//! CSV export for invoices and expenses.
//!
//! Uses BOM + semicolon separator (Czech convention).

use csv::WriterBuilder;

use zfaktury_domain::{Expense, Invoice};

use crate::Result;

/// UTF-8 BOM bytes.
const UTF8_BOM: &[u8] = &[0xEF, 0xBB, 0xBF];

/// Export invoices to CSV format.
///
/// Columns: Číslo;Typ;Stav;Zákazník;Datum vystavení;Splatnost;Základ;DPH;Celkem;Měna
pub fn export_invoices_csv(invoices: &[Invoice]) -> Result<Vec<u8>> {
    let mut output = Vec::new();
    output.extend_from_slice(UTF8_BOM);

    let mut wtr = WriterBuilder::new()
        .delimiter(b';')
        .from_writer(&mut output);

    // Header row.
    wtr.write_record([
        "Číslo",
        "Typ",
        "Stav",
        "Zákazník",
        "Datum vystavení",
        "Splatnost",
        "Základ",
        "DPH",
        "Celkem",
        "Měna",
    ])?;

    for inv in invoices {
        let customer_name = inv.customer.as_ref().map(|c| c.name.as_str()).unwrap_or("");

        wtr.write_record([
            &inv.invoice_number,
            &inv.invoice_type.to_string(),
            &inv.status.to_string(),
            customer_name,
            &inv.issue_date.format("%Y-%m-%d").to_string(),
            &inv.due_date.format("%Y-%m-%d").to_string(),
            &inv.subtotal_amount.to_string(),
            &inv.vat_amount.to_string(),
            &inv.total_amount.to_string(),
            &inv.currency_code,
        ])?;
    }

    wtr.flush()?;
    drop(wtr);

    Ok(output)
}

/// Export expenses to CSV format.
///
/// Columns: Číslo;Kategorie;Popis;Dodavatel;Datum;Částka;DPH;Měna
pub fn export_expenses_csv(expenses: &[Expense]) -> Result<Vec<u8>> {
    let mut output = Vec::new();
    output.extend_from_slice(UTF8_BOM);

    let mut wtr = WriterBuilder::new()
        .delimiter(b';')
        .from_writer(&mut output);

    // Header row.
    wtr.write_record([
        "Číslo",
        "Kategorie",
        "Popis",
        "Dodavatel",
        "Datum",
        "Částka",
        "DPH",
        "Měna",
    ])?;

    for exp in expenses {
        let vendor_name = exp.vendor.as_ref().map(|v| v.name.as_str()).unwrap_or("");

        wtr.write_record([
            &exp.expense_number,
            &exp.category,
            &exp.description,
            vendor_name,
            &exp.issue_date.format("%Y-%m-%d").to_string(),
            &exp.amount.to_string(),
            &exp.vat_amount.to_string(),
            &exp.currency_code,
        ])?;
    }

    wtr.flush()?;
    drop(wtr);

    Ok(output)
}

#[cfg(test)]
mod tests {
    use super::*;
    use zfaktury_domain::Amount;
    use zfaktury_testutil::builders::{ExpenseBuilder, InvoiceBuilder, InvoiceItemBuilder};

    #[test]
    fn test_invoices_csv_bom() {
        let invoices = vec![
            InvoiceBuilder::new()
                .with_items(vec![InvoiceItemBuilder::new().build()])
                .build(),
        ];
        let result = export_invoices_csv(&invoices).unwrap();

        // Starts with UTF-8 BOM.
        assert_eq!(&result[0..3], UTF8_BOM);
    }

    #[test]
    fn test_invoices_csv_headers() {
        let invoices = vec![
            InvoiceBuilder::new()
                .with_items(vec![InvoiceItemBuilder::new().build()])
                .build(),
        ];
        let result = export_invoices_csv(&invoices).unwrap();
        let csv_str = String::from_utf8(result[3..].to_vec()).unwrap();

        let first_line = csv_str.lines().next().unwrap();
        assert_eq!(
            first_line,
            "Číslo;Typ;Stav;Zákazník;Datum vystavení;Splatnost;Základ;DPH;Celkem;Měna"
        );
    }

    #[test]
    fn test_invoices_csv_column_count() {
        let invoices = vec![
            InvoiceBuilder::new()
                .with_items(vec![InvoiceItemBuilder::new().build()])
                .build(),
        ];
        let result = export_invoices_csv(&invoices).unwrap();
        let csv_str = String::from_utf8(result[3..].to_vec()).unwrap();

        // Parse back.
        let mut rdr = csv::ReaderBuilder::new()
            .delimiter(b';')
            .from_reader(csv_str.as_bytes());

        let headers = rdr.headers().unwrap();
        assert_eq!(headers.len(), 10);

        let records: Vec<_> = rdr.records().collect();
        assert_eq!(records.len(), 1);
        let record = records[0].as_ref().unwrap();
        assert_eq!(record.len(), 10);
    }

    #[test]
    fn test_invoices_csv_data() {
        let invoices = vec![
            InvoiceBuilder::new()
                .with_items(vec![
                    InvoiceItemBuilder::new()
                        .unit_price(Amount::new(1000, 0))
                        .vat_rate(21)
                        .build(),
                ])
                .build(),
        ];
        let result = export_invoices_csv(&invoices).unwrap();
        let csv_str = String::from_utf8(result[3..].to_vec()).unwrap();

        let mut rdr = csv::ReaderBuilder::new()
            .delimiter(b';')
            .from_reader(csv_str.as_bytes());
        let record = rdr.records().next().unwrap().unwrap();

        // First column should be invoice number.
        assert_eq!(&record[0], "FV20240001");
        // Type should be "regular".
        assert_eq!(&record[1], "regular");
        // Currency code.
        assert_eq!(&record[9], "CZK");
    }

    #[test]
    fn test_expenses_csv_bom() {
        let expenses = vec![ExpenseBuilder::new().build()];
        let result = export_expenses_csv(&expenses).unwrap();
        assert_eq!(&result[0..3], UTF8_BOM);
    }

    #[test]
    fn test_expenses_csv_headers() {
        let expenses = vec![ExpenseBuilder::new().build()];
        let result = export_expenses_csv(&expenses).unwrap();
        let csv_str = String::from_utf8(result[3..].to_vec()).unwrap();

        let first_line = csv_str.lines().next().unwrap();
        assert_eq!(
            first_line,
            "Číslo;Kategorie;Popis;Dodavatel;Datum;Částka;DPH;Měna"
        );
    }

    #[test]
    fn test_expenses_csv_column_count() {
        let expenses = vec![ExpenseBuilder::new().build()];
        let result = export_expenses_csv(&expenses).unwrap();
        let csv_str = String::from_utf8(result[3..].to_vec()).unwrap();

        let mut rdr = csv::ReaderBuilder::new()
            .delimiter(b';')
            .from_reader(csv_str.as_bytes());
        let headers = rdr.headers().unwrap();
        assert_eq!(headers.len(), 8);

        let records: Vec<_> = rdr.records().collect();
        assert_eq!(records.len(), 1);
        let record = records[0].as_ref().unwrap();
        assert_eq!(record.len(), 8);
    }

    #[test]
    fn test_empty_invoices_csv() {
        let result = export_invoices_csv(&[]).unwrap();
        let csv_str = String::from_utf8(result[3..].to_vec()).unwrap();

        // Should only have header line.
        let lines: Vec<_> = csv_str.lines().collect();
        assert_eq!(lines.len(), 1);
    }

    #[test]
    fn test_empty_expenses_csv() {
        let result = export_expenses_csv(&[]).unwrap();
        let csv_str = String::from_utf8(result[3..].to_vec()).unwrap();

        let lines: Vec<_> = csv_str.lines().collect();
        assert_eq!(lines.len(), 1);
    }
}
