//! ISDOC 6.0.2 XML invoice generator.
//!
//! Produces an ISDOC 6.0.2 XML document from a domain Invoice and supplier info.

use std::collections::HashMap;
use std::io::Cursor;

use quick_xml::Writer;
use quick_xml::events::{BytesDecl, BytesEnd, BytesStart, BytesText, Event};

use zfaktury_domain::{Amount, CURRENCY_CZK, Invoice, InvoiceType};

use super::types::SupplierInfo;
use crate::Result;

/// ISDOC 6.0.2 namespace.
const ISDOC_NAMESPACE: &str = "urn:isdoc:invoice:6.0.2";

/// Generate ISDOC 6.0.2 XML bytes for the given invoice.
pub fn generate_isdoc(invoice: &Invoice, supplier: &SupplierInfo) -> Result<Vec<u8>> {
    let mut writer = Writer::new_with_indent(Cursor::new(Vec::new()), b' ', 2);

    // XML declaration.
    writer.write_event(Event::Decl(BytesDecl::new("1.0", Some("UTF-8"), None)))?;
    writer.write_event(Event::Text(BytesText::new("\n")))?;

    // <Invoice xmlns="urn:isdoc:invoice:6.0.2" version="6.0.2">
    let mut root = BytesStart::new("Invoice");
    root.push_attribute(("xmlns", ISDOC_NAMESPACE));
    root.push_attribute(("version", "6.0.2"));
    writer.write_event(Event::Start(root))?;

    // Document type: 1=invoice, 2=credit note, 4=proforma.
    let doc_type = match &invoice.invoice_type {
        InvoiceType::Regular => 1,
        InvoiceType::CreditNote => 2,
        InvoiceType::Proforma => 4,
    };
    write_text_elem(&mut writer, "DocumentType", &doc_type.to_string())?;
    write_text_elem(&mut writer, "ID", &invoice.invoice_number)?;
    write_text_elem(&mut writer, "UUID", &format!("zfaktury-{}", invoice.id))?;
    write_text_elem(&mut writer, "IssuingSystem", "ZFaktury")?;
    write_text_elem(
        &mut writer,
        "IssueDate",
        &invoice.issue_date.format("%Y-%m-%d").to_string(),
    )?;
    write_text_elem(
        &mut writer,
        "TaxPointDate",
        &invoice.delivery_date.format("%Y-%m-%d").to_string(),
    )?;

    // VATApplicable: true if any item has VAT.
    let vat_applicable = invoice.items.iter().any(|i| i.vat_rate_percent > 0);
    write_text_elem(
        &mut writer,
        "VATApplicable",
        if vat_applicable { "true" } else { "false" },
    )?;

    if !invoice.notes.is_empty() {
        write_text_elem(&mut writer, "Note", &invoice.notes)?;
    }

    write_text_elem(&mut writer, "LocalCurrencyCode", "CZK")?;

    // Foreign currency.
    if !invoice.currency_code.is_empty() && invoice.currency_code != CURRENCY_CZK {
        write_text_elem(&mut writer, "ForeignCurrencyCode", &invoice.currency_code)?;
        write_text_elem(&mut writer, "CurrRate", &invoice.exchange_rate.to_string())?;
        write_text_elem(&mut writer, "RefCurrRate", "1.00")?;
    }

    // Supplier party.
    write_party(&mut writer, "AccountingSupplierParty", supplier, None)?;

    // Customer party.
    write_customer_party(&mut writer, invoice)?;

    // Invoice lines.
    write_invoice_lines(&mut writer, invoice)?;

    // Tax total.
    write_tax_total(&mut writer, invoice)?;

    // Legal monetary total.
    write_legal_monetary_total(&mut writer, invoice)?;

    // Payment means.
    write_payment_means(&mut writer, invoice)?;

    // </Invoice>
    writer.write_event(Event::End(BytesEnd::new("Invoice")))?;

    Ok(writer.into_inner().into_inner())
}

fn write_party(
    writer: &mut Writer<Cursor<Vec<u8>>>,
    tag: &str,
    supplier: &SupplierInfo,
    _customer: Option<&zfaktury_domain::Contact>,
) -> crate::Result<()> {
    writer.write_event(Event::Start(BytesStart::new(tag)))?;
    writer.write_event(Event::Start(BytesStart::new("Party")))?;

    // PartyIdentification
    writer.write_event(Event::Start(BytesStart::new("PartyIdentification")))?;
    write_text_elem(writer, "UserID", &supplier.ico)?;
    write_text_elem(writer, "ID", &supplier.ico)?;
    writer.write_event(Event::End(BytesEnd::new("PartyIdentification")))?;

    // PartyName
    writer.write_event(Event::Start(BytesStart::new("PartyName")))?;
    write_text_elem(writer, "Name", &supplier.company_name)?;
    writer.write_event(Event::End(BytesEnd::new("PartyName")))?;

    // PostalAddress
    writer.write_event(Event::Start(BytesStart::new("PostalAddress")))?;
    write_text_elem(writer, "StreetName", &supplier.street)?;
    write_text_elem(writer, "CityName", &supplier.city)?;
    write_text_elem(writer, "PostalZone", &supplier.zip)?;
    writer.write_event(Event::Start(BytesStart::new("Country")))?;
    write_text_elem(writer, "IdentificationCode", "CZ")?;
    write_text_elem(writer, "Name", "Ceska republika")?;
    writer.write_event(Event::End(BytesEnd::new("Country")))?;
    writer.write_event(Event::End(BytesEnd::new("PostalAddress")))?;

    // PartyTaxScheme
    if !supplier.dic.is_empty() {
        writer.write_event(Event::Start(BytesStart::new("PartyTaxScheme")))?;
        write_text_elem(writer, "CompanyID", &supplier.dic)?;
        writer.write_event(Event::Start(BytesStart::new("TaxScheme")))?;
        write_text_elem(writer, "Name", "VAT")?;
        writer.write_event(Event::End(BytesEnd::new("TaxScheme")))?;
        writer.write_event(Event::End(BytesEnd::new("PartyTaxScheme")))?;
    }

    // Contact
    if !supplier.email.is_empty() || !supplier.phone.is_empty() {
        writer.write_event(Event::Start(BytesStart::new("Contact")))?;
        if !supplier.phone.is_empty() {
            write_text_elem(writer, "Telephone", &supplier.phone)?;
        }
        if !supplier.email.is_empty() {
            write_text_elem(writer, "ElectronicMail", &supplier.email)?;
        }
        writer.write_event(Event::End(BytesEnd::new("Contact")))?;
    }

    writer.write_event(Event::End(BytesEnd::new("Party")))?;
    writer.write_event(Event::End(BytesEnd::new(tag)))?;
    Ok(())
}

fn write_customer_party(
    writer: &mut Writer<Cursor<Vec<u8>>>,
    invoice: &Invoice,
) -> crate::Result<()> {
    writer.write_event(Event::Start(BytesStart::new("AccountingCustomerParty")))?;
    writer.write_event(Event::Start(BytesStart::new("Party")))?;

    if let Some(c) = &invoice.customer {
        // PartyIdentification
        writer.write_event(Event::Start(BytesStart::new("PartyIdentification")))?;
        write_text_elem(writer, "UserID", &c.ico)?;
        write_text_elem(writer, "ID", &c.ico)?;
        writer.write_event(Event::End(BytesEnd::new("PartyIdentification")))?;

        // PartyName
        writer.write_event(Event::Start(BytesStart::new("PartyName")))?;
        write_text_elem(writer, "Name", &c.name)?;
        writer.write_event(Event::End(BytesEnd::new("PartyName")))?;

        // PostalAddress
        writer.write_event(Event::Start(BytesStart::new("PostalAddress")))?;
        write_text_elem(writer, "StreetName", &c.street)?;
        write_text_elem(writer, "CityName", &c.city)?;
        write_text_elem(writer, "PostalZone", &c.zip)?;
        writer.write_event(Event::Start(BytesStart::new("Country")))?;
        let country_code = if c.country.is_empty() {
            "CZ"
        } else {
            &c.country
        };
        write_text_elem(writer, "IdentificationCode", country_code)?;
        write_text_elem(writer, "Name", &c.country)?;
        writer.write_event(Event::End(BytesEnd::new("Country")))?;
        writer.write_event(Event::End(BytesEnd::new("PostalAddress")))?;

        // PartyTaxScheme
        if !c.dic.is_empty() {
            writer.write_event(Event::Start(BytesStart::new("PartyTaxScheme")))?;
            write_text_elem(writer, "CompanyID", &c.dic)?;
            writer.write_event(Event::Start(BytesStart::new("TaxScheme")))?;
            write_text_elem(writer, "Name", "VAT")?;
            writer.write_event(Event::End(BytesEnd::new("TaxScheme")))?;
            writer.write_event(Event::End(BytesEnd::new("PartyTaxScheme")))?;
        }

        // Contact
        if !c.email.is_empty() || !c.phone.is_empty() {
            writer.write_event(Event::Start(BytesStart::new("Contact")))?;
            if !c.phone.is_empty() {
                write_text_elem(writer, "Telephone", &c.phone)?;
            }
            if !c.email.is_empty() {
                write_text_elem(writer, "ElectronicMail", &c.email)?;
            }
            writer.write_event(Event::End(BytesEnd::new("Contact")))?;
        }
    } else {
        // Minimal identification when customer is not loaded.
        writer.write_event(Event::Start(BytesStart::new("PartyIdentification")))?;
        let cust_id = format!("customer-{}", invoice.customer_id);
        write_text_elem(writer, "UserID", &cust_id)?;
        write_text_elem(writer, "ID", &cust_id)?;
        writer.write_event(Event::End(BytesEnd::new("PartyIdentification")))?;
    }

    writer.write_event(Event::End(BytesEnd::new("Party")))?;
    writer.write_event(Event::End(BytesEnd::new("AccountingCustomerParty")))?;
    Ok(())
}

fn write_invoice_lines(
    writer: &mut Writer<Cursor<Vec<u8>>>,
    invoice: &Invoice,
) -> crate::Result<()> {
    writer.write_event(Event::Start(BytesStart::new("InvoiceLines")))?;

    for (i, item) in invoice.items.iter().enumerate() {
        writer.write_event(Event::Start(BytesStart::new("InvoiceLine")))?;

        write_text_elem(writer, "ID", &(i + 1).to_string())?;

        // InvoicedQuantity with unitCode attribute.
        let mut qty_elem = BytesStart::new("InvoicedQuantity");
        if !item.unit.is_empty() {
            qty_elem.push_attribute(("unitCode", item.unit.as_str()));
        }
        writer.write_event(Event::Start(qty_elem))?;
        writer.write_event(Event::Text(BytesText::new(&item.quantity.to_string())))?;
        writer.write_event(Event::End(BytesEnd::new("InvoicedQuantity")))?;

        // Item subtotal before VAT = quantity * unit_price / 100.
        let item_subtotal =
            Amount::from_halere(item.quantity.halere() * item.unit_price.halere() / 100);
        let unit_price_vat = item
            .unit_price
            .multiply((100 + item.vat_rate_percent) as f64 / 100.0);

        write_text_elem(writer, "LineExtensionAmount", &item_subtotal.to_string())?;
        write_text_elem(
            writer,
            "LineExtensionAmountTaxInclusive",
            &item.total_amount.to_string(),
        )?;
        write_text_elem(
            writer,
            "LineExtensionTaxAmount",
            &item.vat_amount.to_string(),
        )?;
        write_text_elem(writer, "UnitPrice", &item.unit_price.to_string())?;
        write_text_elem(writer, "UnitPriceTaxInclusive", &unit_price_vat.to_string())?;

        // ClassifiedTaxCategory
        writer.write_event(Event::Start(BytesStart::new("ClassifiedTaxCategory")))?;
        write_text_elem(writer, "Percent", &item.vat_rate_percent.to_string())?;
        write_text_elem(writer, "VATCalculationMethod", "0")?;
        writer.write_event(Event::End(BytesEnd::new("ClassifiedTaxCategory")))?;

        // Item
        writer.write_event(Event::Start(BytesStart::new("Item")))?;
        write_text_elem(writer, "Description", &item.description)?;
        writer.write_event(Event::End(BytesEnd::new("Item")))?;

        writer.write_event(Event::End(BytesEnd::new("InvoiceLine")))?;
    }

    writer.write_event(Event::End(BytesEnd::new("InvoiceLines")))?;
    Ok(())
}

fn write_tax_total(writer: &mut Writer<Cursor<Vec<u8>>>, invoice: &Invoice) -> crate::Result<()> {
    // Group items by VAT rate.
    struct TaxGroup {
        taxable: Amount,
        tax: Amount,
        total: Amount,
    }
    let mut groups: HashMap<i32, TaxGroup> = HashMap::new();

    for item in &invoice.items {
        let item_subtotal =
            Amount::from_halere(item.quantity.halere() * item.unit_price.halere() / 100);
        let group = groups.entry(item.vat_rate_percent).or_insert(TaxGroup {
            taxable: Amount::ZERO,
            tax: Amount::ZERO,
            total: Amount::ZERO,
        });
        group.taxable = group.taxable + item_subtotal;
        group.tax = group.tax + item.vat_amount;
        group.total = group.total + item.total_amount;
    }

    writer.write_event(Event::Start(BytesStart::new("TaxTotal")))?;

    for (rate, group) in &groups {
        writer.write_event(Event::Start(BytesStart::new("TaxSubTotal")))?;
        write_text_elem(writer, "TaxableAmount", &group.taxable.to_string())?;
        write_text_elem(writer, "TaxAmount", &group.tax.to_string())?;
        write_text_elem(writer, "TaxInclusiveAmount", &group.total.to_string())?;
        write_text_elem(writer, "AlreadyClaimedTaxableAmount", "0.00")?;
        write_text_elem(writer, "AlreadyClaimedTaxAmount", "0.00")?;
        write_text_elem(writer, "AlreadyClaimedTaxInclusiveAmount", "0.00")?;
        write_text_elem(
            writer,
            "DifferenceTaxableAmount",
            &group.taxable.to_string(),
        )?;
        write_text_elem(writer, "DifferenceTaxAmount", &group.tax.to_string())?;
        write_text_elem(
            writer,
            "DifferenceTaxInclusiveAmount",
            &group.total.to_string(),
        )?;
        writer.write_event(Event::Start(BytesStart::new("TaxCategory")))?;
        write_text_elem(writer, "Percent", &rate.to_string())?;
        writer.write_event(Event::End(BytesEnd::new("TaxCategory")))?;
        writer.write_event(Event::End(BytesEnd::new("TaxSubTotal")))?;
    }

    write_text_elem(writer, "TaxAmount", &invoice.vat_amount.to_string())?;
    writer.write_event(Event::End(BytesEnd::new("TaxTotal")))?;
    Ok(())
}

fn write_legal_monetary_total(
    writer: &mut Writer<Cursor<Vec<u8>>>,
    invoice: &Invoice,
) -> crate::Result<()> {
    let payable = invoice.total_amount - invoice.paid_amount;

    writer.write_event(Event::Start(BytesStart::new("LegalMonetaryTotal")))?;
    write_text_elem(
        writer,
        "TaxExclusiveAmount",
        &invoice.subtotal_amount.to_string(),
    )?;
    write_text_elem(
        writer,
        "TaxInclusiveAmount",
        &invoice.total_amount.to_string(),
    )?;
    write_text_elem(writer, "AlreadyClaimedTaxExclusiveAmount", "0.00")?;
    write_text_elem(writer, "AlreadyClaimedTaxInclusiveAmount", "0.00")?;
    write_text_elem(
        writer,
        "DifferenceTaxExclusiveAmount",
        &invoice.subtotal_amount.to_string(),
    )?;
    write_text_elem(
        writer,
        "DifferenceTaxInclusiveAmount",
        &invoice.total_amount.to_string(),
    )?;
    write_text_elem(
        writer,
        "PaidDepositsAmount",
        &invoice.paid_amount.to_string(),
    )?;
    write_text_elem(writer, "PayableAmount", &payable.to_string())?;
    writer.write_event(Event::End(BytesEnd::new("LegalMonetaryTotal")))?;
    Ok(())
}

fn write_payment_means(
    writer: &mut Writer<Cursor<Vec<u8>>>,
    invoice: &Invoice,
) -> crate::Result<()> {
    // PaymentMeansCode: 42 = bank transfer, 10 = cash.
    let means_code = if invoice.payment_method == "cash" {
        10
    } else {
        42
    };

    writer.write_event(Event::Start(BytesStart::new("PaymentMeans")))?;

    // Payment
    writer.write_event(Event::Start(BytesStart::new("Payment")))?;
    write_text_elem(writer, "PaidAmount", &invoice.total_amount.to_string())?;
    write_text_elem(writer, "PaymentMeansCode", &means_code.to_string())?;

    // Details (if bank info available)
    if !invoice.bank_account.is_empty() || !invoice.iban.is_empty() {
        writer.write_event(Event::Start(BytesStart::new("Details")))?;
        write_text_elem(
            writer,
            "PaymentDueDate",
            &invoice.due_date.format("%Y-%m-%d").to_string(),
        )?;
        if !invoice.bank_account.is_empty() {
            write_text_elem(writer, "ID", &invoice.bank_account)?;
        }
        if !invoice.bank_code.is_empty() {
            write_text_elem(writer, "BankCode", &invoice.bank_code)?;
        }
        if !invoice.iban.is_empty() {
            write_text_elem(writer, "IBAN", &invoice.iban)?;
        }
        if !invoice.swift.is_empty() {
            write_text_elem(writer, "BIC", &invoice.swift)?;
        }
        if !invoice.variable_symbol.is_empty() {
            write_text_elem(writer, "VariableSymbol", &invoice.variable_symbol)?;
        }
        if !invoice.constant_symbol.is_empty() {
            write_text_elem(writer, "ConstantSymbol", &invoice.constant_symbol)?;
        }
        writer.write_event(Event::End(BytesEnd::new("Details")))?;
    }

    writer.write_event(Event::End(BytesEnd::new("Payment")))?;

    write_text_elem(
        writer,
        "PaymentDueDate",
        &invoice.due_date.format("%Y-%m-%d").to_string(),
    )?;

    writer.write_event(Event::End(BytesEnd::new("PaymentMeans")))?;
    Ok(())
}

fn write_text_elem(
    writer: &mut Writer<Cursor<Vec<u8>>>,
    tag: &str,
    text: &str,
) -> crate::Result<()> {
    writer.write_event(Event::Start(BytesStart::new(tag)))?;
    writer.write_event(Event::Text(BytesText::new(text)))?;
    writer.write_event(Event::End(BytesEnd::new(tag)))?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use zfaktury_testutil::builders::{InvoiceBuilder, InvoiceItemBuilder};

    fn test_supplier() -> SupplierInfo {
        SupplierInfo {
            company_name: "Jan Novak".to_string(),
            ico: "12345678".to_string(),
            dic: "CZ12345678".to_string(),
            street: "Hlavni 1".to_string(),
            city: "Praha".to_string(),
            zip: "11000".to_string(),
            email: "jan@example.com".to_string(),
            phone: "+420123456789".to_string(),
            bank_account: "123456789".to_string(),
            bank_code: "0100".to_string(),
            iban: "CZ6508000000192000145399".to_string(),
            swift: "GIBACZPX".to_string(),
        }
    }

    fn test_invoice_with_items() -> Invoice {
        InvoiceBuilder::new()
            .with_items(vec![
                InvoiceItemBuilder::new()
                    .description("Web Development")
                    .unit_price(Amount::new(10_000, 0))
                    .quantity(Amount::new(10, 0))
                    .vat_rate(21)
                    .build(),
                InvoiceItemBuilder::new()
                    .description("Hosting")
                    .unit_price(Amount::new(500, 0))
                    .quantity(Amount::new(12, 0))
                    .vat_rate(21)
                    .build(),
            ])
            .build()
    }

    #[test]
    fn test_generate_valid_xml() {
        let invoice = test_invoice_with_items();
        let supplier = test_supplier();
        let result = generate_isdoc(&invoice, &supplier).unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        // Verify valid XML.
        let mut reader = quick_xml::Reader::from_str(&xml_str);
        reader.config_mut().trim_text(true);
        let mut buf = Vec::new();
        loop {
            match reader.read_event_into(&mut buf) {
                Ok(quick_xml::events::Event::Eof) => break,
                Err(e) => panic!("XML parse error: {e}"),
                _ => {}
            }
            buf.clear();
        }

        assert!(xml_str.contains("<Invoice"));
        assert!(xml_str.contains("urn:isdoc:invoice:6.0.2"));
    }

    #[test]
    fn test_document_type_regular() {
        let invoice = test_invoice_with_items();
        let supplier = test_supplier();
        let result = generate_isdoc(&invoice, &supplier).unwrap();
        let xml_str = String::from_utf8(result).unwrap();
        assert!(xml_str.contains("<DocumentType>1</DocumentType>"));
    }

    #[test]
    fn test_document_type_credit_note() {
        let invoice = InvoiceBuilder::new()
            .invoice_type(InvoiceType::CreditNote)
            .with_items(vec![InvoiceItemBuilder::new().build()])
            .build();
        let supplier = test_supplier();
        let result = generate_isdoc(&invoice, &supplier).unwrap();
        let xml_str = String::from_utf8(result).unwrap();
        assert!(xml_str.contains("<DocumentType>2</DocumentType>"));
    }

    #[test]
    fn test_document_type_proforma() {
        let invoice = InvoiceBuilder::new()
            .invoice_type(InvoiceType::Proforma)
            .with_items(vec![InvoiceItemBuilder::new().build()])
            .build();
        let supplier = test_supplier();
        let result = generate_isdoc(&invoice, &supplier).unwrap();
        let xml_str = String::from_utf8(result).unwrap();
        assert!(xml_str.contains("<DocumentType>4</DocumentType>"));
    }

    #[test]
    fn test_amounts_as_decimal_strings() {
        let invoice = test_invoice_with_items();
        let supplier = test_supplier();
        let result = generate_isdoc(&invoice, &supplier).unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        // Amounts should have 2 decimal places (e.g. "10000.00").
        assert!(xml_str.contains("<UnitPrice>10000.00</UnitPrice>"));
        // Payable should also be decimal.
        assert!(xml_str.contains("<PayableAmount>"));
    }

    #[test]
    fn test_supplier_party() {
        let invoice = test_invoice_with_items();
        let supplier = test_supplier();
        let result = generate_isdoc(&invoice, &supplier).unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        assert!(xml_str.contains("<AccountingSupplierParty>"));
        assert!(xml_str.contains("<Name>Jan Novak</Name>"));
        assert!(xml_str.contains("<CompanyID>CZ12345678</CompanyID>"));
    }

    #[test]
    fn test_payment_means() {
        let invoice = test_invoice_with_items();
        let supplier = test_supplier();
        let result = generate_isdoc(&invoice, &supplier).unwrap();
        let xml_str = String::from_utf8(result).unwrap();

        assert!(xml_str.contains("<PaymentMeansCode>42</PaymentMeansCode>"));
    }
}
