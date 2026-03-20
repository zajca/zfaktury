use std::sync::Arc;

use chrono::{NaiveDate, NaiveDateTime};
use zfaktury_domain::{
    Amount, CURRENCY_CZK, DomainError, Invoice, InvoiceFilter, InvoiceItem, InvoiceStatus,
    InvoiceType, RelationType,
};

use super::audit_svc::AuditService;
use super::contact_svc::ContactService;
use super::sequence_svc::SequenceService;
use crate::repository::traits::InvoiceRepo;

/// Service for invoice management.
pub struct InvoiceService {
    repo: Arc<dyn InvoiceRepo + Send + Sync>,
    contacts: Arc<ContactService>,
    sequences: Option<Arc<SequenceService>>,
    audit: Option<Arc<AuditService>>,
}

impl InvoiceService {
    pub fn new(
        repo: Arc<dyn InvoiceRepo + Send + Sync>,
        contacts: Arc<ContactService>,
        sequences: Option<Arc<SequenceService>>,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self {
            repo,
            contacts,
            sequences,
            audit,
        }
    }

    /// Validates, calculates totals, assigns a number, and persists a new invoice.
    pub fn create(&self, invoice: &mut Invoice) -> Result<(), DomainError> {
        if invoice.customer_id == 0 {
            return Err(DomainError::InvalidInput);
        }
        if invoice.items.is_empty() {
            return Err(DomainError::NoItems);
        }
        if invoice.due_date == NaiveDate::default() {
            return Err(DomainError::InvalidInput);
        }

        // Verify customer exists.
        self.contacts.get_by_id(invoice.customer_id)?;

        // Set defaults.
        if invoice.status == InvoiceStatus::Draft {
            // already default
        }
        if invoice.invoice_type == InvoiceType::Regular {
            // already default
        }
        if invoice.currency_code.is_empty() {
            invoice.currency_code = CURRENCY_CZK.to_string();
        }
        if invoice.issue_date == NaiveDate::default() {
            invoice.issue_date = chrono::Local::now().date_naive();
        }
        if invoice.delivery_date == NaiveDate::default() {
            invoice.delivery_date = invoice.issue_date;
        }

        // Auto-assign sequence if none provided.
        if invoice.sequence_id == 0 && invoice.invoice_number.is_empty() {
            if let Some(ref sequences) = self.sequences {
                let prefix = match invoice.invoice_type {
                    InvoiceType::Proforma => "ZF",
                    InvoiceType::CreditNote => "DN",
                    _ => "FV",
                };
                let year = {
                    use chrono::Datelike;
                    invoice.issue_date.year()
                };
                let seq = sequences.get_or_create_for_year(prefix, year)?;
                invoice.sequence_id = seq.id;
            }
        }

        // Assign invoice number from sequence.
        if invoice.invoice_number.is_empty() && invoice.sequence_id > 0 {
            invoice.invoice_number = self.repo.get_next_number(invoice.sequence_id)?;
        }

        // Set variable symbol.
        if invoice.variable_symbol.is_empty() {
            invoice.variable_symbol = invoice.invoice_number.clone();
        }

        // Calculate totals.
        invoice.calculate_totals();

        self.repo.create(invoice)?;
        if let Some(ref audit) = self.audit {
            audit.log("invoice", invoice.id, "create", None, None);
        }
        Ok(())
    }

    /// Validates, recalculates totals, and updates an existing invoice.
    pub fn update(&self, invoice: &mut Invoice) -> Result<(), DomainError> {
        if invoice.id == 0 || invoice.customer_id == 0 {
            return Err(DomainError::InvalidInput);
        }
        if invoice.items.is_empty() {
            return Err(DomainError::NoItems);
        }
        if invoice.due_date == NaiveDate::default() {
            return Err(DomainError::InvalidInput);
        }

        let existing = self.repo.get_by_id(invoice.id)?;
        if existing.status == InvoiceStatus::Paid {
            return Err(DomainError::PaidInvoice);
        }

        // Preserve fields.
        if invoice.status == InvoiceStatus::Draft && existing.status != InvoiceStatus::Draft {
            invoice.status = existing.status.clone();
        }
        if invoice.invoice_number.is_empty() {
            invoice.invoice_number = existing.invoice_number;
        }
        if invoice.sequence_id == 0 {
            invoice.sequence_id = existing.sequence_id;
        }
        if invoice.variable_symbol.is_empty() {
            invoice.variable_symbol = existing.variable_symbol;
        }

        invoice.calculate_totals();

        self.repo.update(invoice)?;
        if let Some(ref audit) = self.audit {
            audit.log("invoice", invoice.id, "update", None, None);
        }
        Ok(())
    }

    /// Soft-deletes an invoice. Paid invoices cannot be deleted.
    pub fn delete(&self, id: i64) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        let existing = self.repo.get_by_id(id)?;
        if existing.status == InvoiceStatus::Paid {
            return Err(DomainError::PaidInvoice);
        }
        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("invoice", id, "delete", None, None);
        }
        Ok(())
    }

    pub fn get_by_id(&self, id: i64) -> Result<Invoice, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)
    }

    pub fn get_related_invoices(&self, invoice_id: i64) -> Result<Vec<Invoice>, DomainError> {
        self.repo.get_related_invoices(invoice_id)
    }

    pub fn list(&self, mut filter: InvoiceFilter) -> Result<(Vec<Invoice>, i64), DomainError> {
        if filter.limit <= 0 {
            filter.limit = 20;
        }
        if filter.limit > 100 {
            filter.limit = 100;
        }
        if filter.offset < 0 {
            filter.offset = 0;
        }
        self.repo.list(&filter)
    }

    /// Marks a draft invoice as sent.
    pub fn mark_as_sent(&self, id: i64) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        let mut invoice = self.repo.get_by_id(id)?;
        if invoice.status != InvoiceStatus::Draft {
            return Err(DomainError::InvalidInput);
        }
        invoice.status = InvoiceStatus::Sent;
        invoice.sent_at = Some(chrono::Local::now().naive_local());
        self.repo.update(&mut invoice)
    }

    /// Marks an invoice as paid.
    pub fn mark_as_paid(
        &self,
        id: i64,
        amount: Amount,
        paid_at: NaiveDateTime,
    ) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        let mut invoice = self.repo.get_by_id(id)?;
        if invoice.status == InvoiceStatus::Paid {
            return Err(DomainError::PaidInvoice);
        }
        if invoice.status == InvoiceStatus::Cancelled {
            return Err(DomainError::InvalidInput);
        }
        invoice.paid_amount = amount;
        invoice.paid_at = Some(paid_at);
        invoice.status = InvoiceStatus::Paid;
        self.repo.update(&mut invoice)
    }

    /// Creates a settlement invoice from a paid proforma.
    pub fn settle_proforma(&self, proforma_id: i64) -> Result<Invoice, DomainError> {
        if proforma_id == 0 {
            return Err(DomainError::InvalidInput);
        }
        let proforma = self.repo.get_by_id(proforma_id)?;
        if proforma.invoice_type != InvoiceType::Proforma {
            return Err(DomainError::InvalidInput);
        }
        if proforma.status != InvoiceStatus::Paid {
            return Err(DomainError::InvalidInput);
        }

        // Idempotency check.
        if let Ok(Some(existing)) = self
            .repo
            .find_by_related_invoice(proforma_id, &RelationType::Settlement.to_string())
        {
            return Ok(existing);
        }

        let today = chrono::Local::now().date_naive();
        let now = chrono::Local::now().naive_local();
        let mut settlement = Invoice {
            id: 0,
            sequence_id: 0,
            invoice_number: String::new(),
            invoice_type: InvoiceType::Regular,
            status: InvoiceStatus::Draft,
            issue_date: today,
            due_date: today,
            delivery_date: today,
            variable_symbol: String::new(),
            constant_symbol: proforma.constant_symbol,
            customer_id: proforma.customer_id,
            customer: None,
            currency_code: proforma.currency_code,
            exchange_rate: proforma.exchange_rate,
            payment_method: proforma.payment_method,
            bank_account: proforma.bank_account,
            bank_code: proforma.bank_code,
            iban: proforma.iban,
            swift: proforma.swift,
            subtotal_amount: Amount::ZERO,
            vat_amount: Amount::ZERO,
            total_amount: Amount::ZERO,
            paid_amount: Amount::ZERO,
            notes: proforma.notes,
            internal_notes: String::new(),
            related_invoice_id: Some(proforma.id),
            relation_type: RelationType::Settlement,
            sent_at: None,
            paid_at: None,
            items: proforma
                .items
                .iter()
                .map(|item| InvoiceItem {
                    id: 0,
                    invoice_id: 0,
                    description: item.description.clone(),
                    quantity: item.quantity,
                    unit: item.unit.clone(),
                    unit_price: item.unit_price,
                    vat_rate_percent: item.vat_rate_percent,
                    vat_amount: Amount::ZERO,
                    total_amount: Amount::ZERO,
                    sort_order: item.sort_order,
                })
                .collect(),
            created_at: now,
            updated_at: now,
            deleted_at: None,
        };

        self.create(&mut settlement)?;
        Ok(settlement)
    }

    /// Creates a credit note for a regular invoice.
    pub fn create_credit_note(
        &self,
        original_id: i64,
        items: Option<Vec<InvoiceItem>>,
        reason: &str,
    ) -> Result<Invoice, DomainError> {
        if original_id == 0 {
            return Err(DomainError::InvalidInput);
        }
        let original = self.repo.get_by_id(original_id)?;
        if original.invoice_type != InvoiceType::Regular {
            return Err(DomainError::InvalidInput);
        }
        if original.status != InvoiceStatus::Sent && original.status != InvoiceStatus::Paid {
            return Err(DomainError::InvalidInput);
        }

        let today = chrono::Local::now().date_naive();
        let now = chrono::Local::now().naive_local();
        let credit_items: Vec<InvoiceItem> = match items {
            Some(provided) if !provided.is_empty() => provided
                .iter()
                .map(|item| InvoiceItem {
                    id: 0,
                    invoice_id: 0,
                    description: item.description.clone(),
                    quantity: item.quantity,
                    unit: item.unit.clone(),
                    unit_price: -item.unit_price,
                    vat_rate_percent: item.vat_rate_percent,
                    vat_amount: Amount::ZERO,
                    total_amount: Amount::ZERO,
                    sort_order: item.sort_order,
                })
                .collect(),
            _ => original
                .items
                .iter()
                .map(|item| InvoiceItem {
                    id: 0,
                    invoice_id: 0,
                    description: item.description.clone(),
                    quantity: item.quantity,
                    unit: item.unit.clone(),
                    unit_price: -item.unit_price,
                    vat_rate_percent: item.vat_rate_percent,
                    vat_amount: Amount::ZERO,
                    total_amount: Amount::ZERO,
                    sort_order: item.sort_order,
                })
                .collect(),
        };

        let mut credit_note = Invoice {
            id: 0,
            sequence_id: 0,
            invoice_number: String::new(),
            invoice_type: InvoiceType::CreditNote,
            status: InvoiceStatus::Draft,
            issue_date: today,
            due_date: today,
            delivery_date: today,
            variable_symbol: String::new(),
            constant_symbol: original.constant_symbol,
            customer_id: original.customer_id,
            customer: None,
            currency_code: original.currency_code,
            exchange_rate: original.exchange_rate,
            payment_method: original.payment_method,
            bank_account: original.bank_account,
            bank_code: original.bank_code,
            iban: original.iban,
            swift: original.swift,
            subtotal_amount: Amount::ZERO,
            vat_amount: Amount::ZERO,
            total_amount: Amount::ZERO,
            paid_amount: Amount::ZERO,
            notes: reason.to_string(),
            internal_notes: String::new(),
            related_invoice_id: Some(original.id),
            relation_type: RelationType::CreditNote,
            sent_at: None,
            paid_at: None,
            items: credit_items,
            created_at: now,
            updated_at: now,
            deleted_at: None,
        };

        self.create(&mut credit_note)?;
        Ok(credit_note)
    }

    /// Creates a duplicate of an existing invoice as a new draft.
    pub fn duplicate(&self, id: i64) -> Result<Invoice, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        let original = self.repo.get_by_id(id)?;
        let today = chrono::Local::now().date_naive();
        let now = chrono::Local::now().naive_local();

        let mut dup = Invoice {
            id: 0,
            sequence_id: original.sequence_id,
            invoice_number: String::new(),
            invoice_type: original.invoice_type,
            status: InvoiceStatus::Draft,
            issue_date: today,
            due_date: today + chrono::Duration::days(14),
            delivery_date: today,
            variable_symbol: String::new(),
            constant_symbol: original.constant_symbol,
            customer_id: original.customer_id,
            customer: None,
            currency_code: original.currency_code,
            exchange_rate: original.exchange_rate,
            payment_method: original.payment_method,
            bank_account: original.bank_account,
            bank_code: original.bank_code,
            iban: original.iban,
            swift: original.swift,
            subtotal_amount: Amount::ZERO,
            vat_amount: Amount::ZERO,
            total_amount: Amount::ZERO,
            paid_amount: Amount::ZERO,
            notes: original.notes,
            internal_notes: String::new(),
            related_invoice_id: None,
            relation_type: RelationType::None,
            sent_at: None,
            paid_at: None,
            items: original
                .items
                .iter()
                .map(|item| InvoiceItem {
                    id: 0,
                    invoice_id: 0,
                    description: item.description.clone(),
                    quantity: item.quantity,
                    unit: item.unit.clone(),
                    unit_price: item.unit_price,
                    vat_rate_percent: item.vat_rate_percent,
                    vat_amount: Amount::ZERO,
                    total_amount: Amount::ZERO,
                    sort_order: item.sort_order,
                })
                .collect(),
            created_at: now,
            updated_at: now,
            deleted_at: None,
        };

        self.create(&mut dup)?;
        Ok(dup)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::repository::traits::ContactRepo;
    use std::sync::Mutex;
    use zfaktury_domain::{Contact, ContactFilter, ContactType};

    // Minimal mock repos for invoice tests.

    struct MockInvoiceRepo {
        invoices: Mutex<Vec<Invoice>>,
        next_id: Mutex<i64>,
    }

    impl MockInvoiceRepo {
        fn new() -> Self {
            Self {
                invoices: Mutex::new(Vec::new()),
                next_id: Mutex::new(1),
            }
        }
    }

    impl InvoiceRepo for MockInvoiceRepo {
        fn create(&self, inv: &mut Invoice) -> Result<(), DomainError> {
            let mut id = self.next_id.lock().unwrap();
            inv.id = *id;
            *id += 1;
            self.invoices.lock().unwrap().push(inv.clone());
            Ok(())
        }
        fn update(&self, inv: &mut Invoice) -> Result<(), DomainError> {
            let mut invoices = self.invoices.lock().unwrap();
            if let Some(i) = invoices.iter_mut().find(|i| i.id == inv.id) {
                *i = inv.clone();
                Ok(())
            } else {
                Err(DomainError::NotFound)
            }
        }
        fn delete(&self, _id: i64) -> Result<(), DomainError> {
            Ok(())
        }
        fn get_by_id(&self, id: i64) -> Result<Invoice, DomainError> {
            self.invoices
                .lock()
                .unwrap()
                .iter()
                .find(|i| i.id == id)
                .cloned()
                .ok_or(DomainError::NotFound)
        }
        fn list(&self, _filter: &InvoiceFilter) -> Result<(Vec<Invoice>, i64), DomainError> {
            let invoices = self.invoices.lock().unwrap();
            let count = invoices.len() as i64;
            Ok((invoices.clone(), count))
        }
        fn update_status(&self, _id: i64, _status: &str) -> Result<(), DomainError> {
            Ok(())
        }
        fn get_next_number(&self, _sequence_id: i64) -> Result<String, DomainError> {
            Ok("FV20240001".to_string())
        }
        fn get_related_invoices(&self, _invoice_id: i64) -> Result<Vec<Invoice>, DomainError> {
            Ok(Vec::new())
        }
        fn find_by_related_invoice(
            &self,
            _related_id: i64,
            _relation_type: &str,
        ) -> Result<Option<Invoice>, DomainError> {
            Ok(None)
        }
    }

    struct MockContactRepoForInvoice;

    impl ContactRepo for MockContactRepoForInvoice {
        fn create(&self, c: &mut Contact) -> Result<(), DomainError> {
            c.id = 1;
            Ok(())
        }
        fn update(&self, _c: &mut Contact) -> Result<(), DomainError> {
            Ok(())
        }
        fn delete(&self, _id: i64) -> Result<(), DomainError> {
            Ok(())
        }
        fn get_by_id(&self, _id: i64) -> Result<Contact, DomainError> {
            Ok(Contact {
                id: 1,
                contact_type: ContactType::Company,
                name: "Test Customer".to_string(),
                ico: String::new(),
                dic: String::new(),
                street: String::new(),
                city: String::new(),
                zip: String::new(),
                country: String::new(),
                email: String::new(),
                phone: String::new(),
                web: String::new(),
                bank_account: String::new(),
                bank_code: String::new(),
                iban: String::new(),
                swift: String::new(),
                payment_terms_days: 14,
                tags: String::new(),
                notes: String::new(),
                is_favorite: false,
                vat_unreliable_at: None,
                created_at: NaiveDateTime::default(),
                updated_at: NaiveDateTime::default(),
                deleted_at: None,
            })
        }
        fn list(&self, _f: &ContactFilter) -> Result<(Vec<Contact>, i64), DomainError> {
            Ok((Vec::new(), 0))
        }
        fn find_by_ico(&self, _ico: &str) -> Result<Contact, DomainError> {
            Err(DomainError::NotFound)
        }
    }

    fn make_invoice_svc() -> InvoiceService {
        let inv_repo = Arc::new(MockInvoiceRepo::new());
        let contact_repo = Arc::new(MockContactRepoForInvoice);
        let contact_svc = Arc::new(ContactService::new(contact_repo, None, None));
        InvoiceService::new(inv_repo, contact_svc, None, None)
    }

    fn make_test_invoice() -> Invoice {
        let today = chrono::Local::now().date_naive();
        let now = chrono::Local::now().naive_local();
        Invoice {
            id: 0,
            sequence_id: 0,
            invoice_number: "TEST001".to_string(),
            invoice_type: InvoiceType::Regular,
            status: InvoiceStatus::Draft,
            issue_date: today,
            due_date: today + chrono::Duration::days(14),
            delivery_date: today,
            variable_symbol: String::new(),
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
            items: vec![InvoiceItem {
                id: 0,
                invoice_id: 0,
                description: "Web development".to_string(),
                quantity: Amount::new(10, 0),
                unit: "hod".to_string(),
                unit_price: Amount::new(1500, 0),
                vat_rate_percent: 21,
                vat_amount: Amount::ZERO,
                total_amount: Amount::ZERO,
                sort_order: 1,
            }],
            created_at: now,
            updated_at: now,
            deleted_at: None,
        }
    }

    #[test]
    fn test_create_invoice() {
        let svc = make_invoice_svc();
        let mut inv = make_test_invoice();
        svc.create(&mut inv).unwrap();
        assert!(inv.id > 0);
        assert!(inv.total_amount > Amount::ZERO);
    }

    #[test]
    fn test_create_requires_items() {
        let svc = make_invoice_svc();
        let mut inv = make_test_invoice();
        inv.items.clear();
        assert!(matches!(svc.create(&mut inv), Err(DomainError::NoItems)));
    }

    #[test]
    fn test_create_requires_customer() {
        let svc = make_invoice_svc();
        let mut inv = make_test_invoice();
        inv.customer_id = 0;
        assert!(matches!(
            svc.create(&mut inv),
            Err(DomainError::InvalidInput)
        ));
    }

    #[test]
    fn test_cannot_delete_paid_invoice() {
        let svc = make_invoice_svc();
        let mut inv = make_test_invoice();
        inv.status = InvoiceStatus::Paid;
        inv.invoice_number = "PAID001".to_string();
        // Manually insert via repo to bypass create validation.
        let repo = &svc.repo;
        repo.create(&mut inv).unwrap();
        assert!(matches!(svc.delete(inv.id), Err(DomainError::PaidInvoice)));
    }

    #[test]
    fn test_cannot_update_paid_invoice() {
        let svc = make_invoice_svc();
        let mut inv = make_test_invoice();
        svc.create(&mut inv).unwrap();

        // Mark as paid via repo directly.
        inv.status = InvoiceStatus::Paid;
        svc.repo.update(&mut inv).unwrap();

        inv.notes = "updated".to_string();
        assert!(matches!(
            svc.update(&mut inv),
            Err(DomainError::PaidInvoice)
        ));
    }
}
