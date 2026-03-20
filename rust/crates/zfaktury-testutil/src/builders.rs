//! Builder pattern helpers for creating test domain entities.

use chrono::NaiveDate;
use zfaktury_domain::*;

// ---------------------------------------------------------------------------
// ContactBuilder
// ---------------------------------------------------------------------------

pub struct ContactBuilder {
    contact: Contact,
}

impl ContactBuilder {
    pub fn new() -> Self {
        Self {
            contact: Contact {
                id: 1,
                contact_type: ContactType::Company,
                name: "Test Company".to_string(),
                ico: "12345678".to_string(),
                dic: "CZ12345678".to_string(),
                street: "Test Street 1".to_string(),
                city: "Prague".to_string(),
                zip: "11000".to_string(),
                country: "CZ".to_string(),
                email: "test@example.com".to_string(),
                phone: "+420123456789".to_string(),
                web: String::new(),
                bank_account: "123456789".to_string(),
                bank_code: "0100".to_string(),
                iban: "CZ6508000000192000145399".to_string(),
                swift: "GIBACZPX".to_string(),
                payment_terms_days: 14,
                tags: String::new(),
                notes: String::new(),
                is_favorite: false,
                vat_unreliable_at: None,
                created_at: default_datetime(),
                updated_at: default_datetime(),
                deleted_at: None,
            },
        }
    }

    pub fn id(mut self, id: i64) -> Self {
        self.contact.id = id;
        self
    }

    pub fn name(mut self, name: &str) -> Self {
        self.contact.name = name.to_string();
        self
    }

    pub fn ico(mut self, ico: &str) -> Self {
        self.contact.ico = ico.to_string();
        self
    }

    pub fn dic(mut self, dic: &str) -> Self {
        self.contact.dic = dic.to_string();
        self
    }

    pub fn contact_type(mut self, ct: ContactType) -> Self {
        self.contact.contact_type = ct;
        self
    }

    pub fn country(mut self, country: &str) -> Self {
        self.contact.country = country.to_string();
        self
    }

    pub fn build(self) -> Contact {
        self.contact
    }
}

impl Default for ContactBuilder {
    fn default() -> Self {
        Self::new()
    }
}

// ---------------------------------------------------------------------------
// InvoiceBuilder
// ---------------------------------------------------------------------------

pub struct InvoiceBuilder {
    invoice: Invoice,
}

impl InvoiceBuilder {
    pub fn new() -> Self {
        Self {
            invoice: Invoice {
                id: 1,
                sequence_id: 1,
                invoice_number: "FV20240001".to_string(),
                invoice_type: InvoiceType::Regular,
                status: InvoiceStatus::Draft,
                issue_date: default_date(),
                due_date: default_date(),
                delivery_date: default_date(),
                variable_symbol: "20240001".to_string(),
                constant_symbol: String::new(),
                customer_id: 1,
                customer: None,
                currency_code: CURRENCY_CZK.to_string(),
                exchange_rate: Amount::new(1, 0),
                payment_method: "bank_transfer".to_string(),
                bank_account: String::new(),
                bank_code: String::new(),
                iban: String::new(),
                swift: String::new(),
                subtotal_amount: Amount::ZERO,
                vat_amount: Amount::ZERO,
                total_amount: Amount::ZERO,
                paid_amount: Amount::ZERO,
                notes: String::new(),
                internal_notes: String::new(),
                related_invoice_id: None,
                relation_type: RelationType::None,
                sent_at: None,
                paid_at: None,
                items: Vec::new(),
                created_at: default_datetime(),
                updated_at: default_datetime(),
                deleted_at: None,
            },
        }
    }

    pub fn id(mut self, id: i64) -> Self {
        self.invoice.id = id;
        self
    }

    pub fn invoice_type(mut self, t: InvoiceType) -> Self {
        self.invoice.invoice_type = t;
        self
    }

    pub fn status(mut self, s: InvoiceStatus) -> Self {
        self.invoice.status = s;
        self
    }

    pub fn customer_id(mut self, id: i64) -> Self {
        self.invoice.customer_id = id;
        self
    }

    pub fn issue_date(mut self, date: NaiveDate) -> Self {
        self.invoice.issue_date = date;
        self
    }

    pub fn due_date(mut self, date: NaiveDate) -> Self {
        self.invoice.due_date = date;
        self
    }

    pub fn with_items(mut self, items: Vec<InvoiceItem>) -> Self {
        self.invoice.items = items;
        self.invoice.calculate_totals();
        self
    }

    pub fn total_amount(mut self, amount: Amount) -> Self {
        self.invoice.total_amount = amount;
        self
    }

    pub fn build(self) -> Invoice {
        self.invoice
    }
}

impl Default for InvoiceBuilder {
    fn default() -> Self {
        Self::new()
    }
}

// ---------------------------------------------------------------------------
// InvoiceItemBuilder
// ---------------------------------------------------------------------------

pub struct InvoiceItemBuilder {
    item: InvoiceItem,
}

impl InvoiceItemBuilder {
    pub fn new() -> Self {
        Self {
            item: InvoiceItem {
                id: 0,
                invoice_id: 0,
                description: "Test Item".to_string(),
                quantity: Amount::new(1, 0),
                unit: "ks".to_string(),
                unit_price: Amount::new(1000, 0),
                vat_rate_percent: 21,
                vat_amount: Amount::ZERO,
                total_amount: Amount::ZERO,
                sort_order: 0,
            },
        }
    }

    pub fn description(mut self, desc: &str) -> Self {
        self.item.description = desc.to_string();
        self
    }

    pub fn quantity(mut self, qty: Amount) -> Self {
        self.item.quantity = qty;
        self
    }

    pub fn unit_price(mut self, price: Amount) -> Self {
        self.item.unit_price = price;
        self
    }

    pub fn vat_rate(mut self, rate: i32) -> Self {
        self.item.vat_rate_percent = rate;
        self
    }

    pub fn build(self) -> InvoiceItem {
        self.item
    }
}

impl Default for InvoiceItemBuilder {
    fn default() -> Self {
        Self::new()
    }
}

// ---------------------------------------------------------------------------
// ExpenseBuilder
// ---------------------------------------------------------------------------

pub struct ExpenseBuilder {
    expense: Expense,
}

impl ExpenseBuilder {
    pub fn new() -> Self {
        Self {
            expense: Expense {
                id: 1,
                vendor_id: None,
                vendor: None,
                expense_number: "N20240001".to_string(),
                category: "office".to_string(),
                description: "Test Expense".to_string(),
                issue_date: default_date(),
                amount: Amount::new(1000, 0),
                currency_code: CURRENCY_CZK.to_string(),
                exchange_rate: Amount::new(1, 0),
                vat_rate_percent: 21,
                vat_amount: Amount::new(210, 0),
                is_tax_deductible: true,
                business_percent: 100,
                payment_method: "bank_transfer".to_string(),
                document_path: String::new(),
                notes: String::new(),
                tax_reviewed_at: None,
                items: Vec::new(),
                created_at: default_datetime(),
                updated_at: default_datetime(),
                deleted_at: None,
            },
        }
    }

    pub fn id(mut self, id: i64) -> Self {
        self.expense.id = id;
        self
    }

    pub fn amount(mut self, amount: Amount) -> Self {
        self.expense.amount = amount;
        self
    }

    pub fn vat_amount(mut self, amount: Amount) -> Self {
        self.expense.vat_amount = amount;
        self
    }

    pub fn category(mut self, cat: &str) -> Self {
        self.expense.category = cat.to_string();
        self
    }

    pub fn business_percent(mut self, pct: i32) -> Self {
        self.expense.business_percent = pct;
        self
    }

    pub fn build(self) -> Expense {
        self.expense
    }
}

impl Default for ExpenseBuilder {
    fn default() -> Self {
        Self::new()
    }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

fn default_date() -> NaiveDate {
    NaiveDate::from_ymd_opt(2024, 1, 1).unwrap()
}

fn default_datetime() -> chrono::NaiveDateTime {
    default_date().and_hms_opt(0, 0, 0).unwrap()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_contact_builder() {
        let contact = ContactBuilder::new()
            .id(42)
            .name("Acme Corp")
            .ico("87654321")
            .build();
        assert_eq!(contact.id, 42);
        assert_eq!(contact.name, "Acme Corp");
        assert_eq!(contact.ico, "87654321");
    }

    #[test]
    fn test_invoice_builder_with_items() {
        let items = vec![
            InvoiceItemBuilder::new()
                .unit_price(Amount::new(1000, 0))
                .vat_rate(21)
                .build(),
        ];
        let invoice = InvoiceBuilder::new()
            .id(1)
            .status(InvoiceStatus::Sent)
            .with_items(items)
            .build();
        assert_eq!(invoice.status, InvoiceStatus::Sent);
        assert!(!invoice.subtotal_amount.is_zero());
    }

    #[test]
    fn test_expense_builder() {
        let expense = ExpenseBuilder::new()
            .amount(Amount::new(5000, 0))
            .category("travel")
            .build();
        assert_eq!(expense.amount, Amount::new(5000, 0));
        assert_eq!(expense.category, "travel");
    }
}
