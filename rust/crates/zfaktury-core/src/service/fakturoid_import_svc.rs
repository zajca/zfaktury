use std::collections::HashMap;
use std::sync::Arc;
use std::thread;
use std::time::Duration;

use chrono::NaiveDate;
use log::{info, warn};
use zfaktury_api::fakturoid::{self, FakturoidClient};
use zfaktury_domain::{
    Amount, ContactType, DomainError, Expense, ExpenseDocument, ExpenseItem, FakturoidImportLog,
    FakturoidImportResult, InvoiceDocument, InvoiceItem, InvoiceStatus, InvoiceType,
};

use super::contact_svc::ContactService;
use super::document_svc::DocumentService;
use super::expense_svc::ExpenseService;
use super::invoice_document_svc::InvoiceDocumentService;
use super::invoice_svc::InvoiceService;
use crate::repository::traits::{ContactRepo, ExpenseRepo, FakturoidImportLogRepo, InvoiceRepo};

const ATTACHMENT_DELAY: Duration = Duration::from_millis(700);

/// Dependencies for the Fakturoid import service.
pub struct FakturoidImportDeps {
    pub import_repo: Arc<dyn FakturoidImportLogRepo + Send + Sync>,
    pub contact_repo: Arc<dyn ContactRepo + Send + Sync>,
    pub invoice_repo: Arc<dyn InvoiceRepo + Send + Sync>,
    pub expense_repo: Arc<dyn ExpenseRepo + Send + Sync>,
    pub contact_svc: Arc<ContactService>,
    pub invoice_svc: Arc<InvoiceService>,
    pub expense_svc: Arc<ExpenseService>,
    pub document_svc: Arc<DocumentService>,
    pub invoice_document_svc: Arc<InvoiceDocumentService>,
}

/// Service for importing data from Fakturoid.
pub struct FakturoidImportService {
    import_repo: Arc<dyn FakturoidImportLogRepo + Send + Sync>,
    contact_repo: Arc<dyn ContactRepo + Send + Sync>,
    invoice_repo: Arc<dyn InvoiceRepo + Send + Sync>,
    expense_repo: Arc<dyn ExpenseRepo + Send + Sync>,
    contact_svc: Arc<ContactService>,
    invoice_svc: Arc<InvoiceService>,
    expense_svc: Arc<ExpenseService>,
    document_svc: Arc<DocumentService>,
    invoice_document_svc: Arc<InvoiceDocumentService>,
}

impl FakturoidImportService {
    pub fn new(deps: FakturoidImportDeps) -> Self {
        Self {
            import_repo: deps.import_repo,
            contact_repo: deps.contact_repo,
            invoice_repo: deps.invoice_repo,
            expense_repo: deps.expense_repo,
            contact_svc: deps.contact_svc,
            invoice_svc: deps.invoice_svc,
            expense_svc: deps.expense_svc,
            document_svc: deps.document_svc,
            invoice_document_svc: deps.invoice_document_svc,
        }
    }

    /// Import all data from Fakturoid using the given credentials.
    /// Credentials are provided at call time and not stored anywhere.
    pub fn import_all(
        &self,
        slug: &str,
        email: &str,
        client_id: &str,
        client_secret: &str,
        download_attachments: bool,
    ) -> Result<FakturoidImportResult, DomainError> {
        // Authenticate
        let mut client = FakturoidClient::new(slug, email, client_id, client_secret);
        client.authenticate().map_err(|e| {
            warn!("Fakturoid authentication failed: {e}");
            DomainError::InvalidInput
        })?;

        // Fetch all data
        info!("Fetching subjects from Fakturoid...");
        let subjects = client.list_subjects().map_err(|e| {
            warn!("Fetching Fakturoid subjects failed: {e}");
            DomainError::InvalidInput
        })?;

        info!("Fetching invoices from Fakturoid...");
        let invoices = client.list_invoices().map_err(|e| {
            warn!("Fetching Fakturoid invoices failed: {e}");
            DomainError::InvalidInput
        })?;

        info!("Fetching expenses from Fakturoid...");
        let expenses = client.list_expenses().map_err(|e| {
            warn!("Fetching Fakturoid expenses failed: {e}");
            DomainError::InvalidInput
        })?;

        // Build lookup maps for attachments
        let invoice_by_fak_id: HashMap<i64, &fakturoid::Invoice> =
            invoices.iter().map(|inv| (inv.id, inv)).collect();
        let expense_by_fak_id: HashMap<i64, &fakturoid::Expense> =
            expenses.iter().map(|exp| (exp.id, exp)).collect();

        let mut result = FakturoidImportResult {
            contacts_created: 0,
            contacts_skipped: 0,
            invoices_created: 0,
            invoices_skipped: 0,
            expenses_created: 0,
            expenses_skipped: 0,
            attachments_downloaded: 0,
            attachments_skipped: 0,
            errors: Vec::new(),
        };

        // --- Import contacts first (invoices/expenses depend on them) ---
        let mut subject_map: HashMap<i64, i64> = HashMap::new(); // fakturoid_id -> local_id

        for subj in &subjects {
            let preview = self.preview_contact(subj);
            match preview.status.as_str() {
                "duplicate" => {
                    if let Some(existing_id) = preview.existing_id {
                        subject_map.insert(subj.id, existing_id);
                    }
                    result.contacts_skipped += 1;
                }
                "new" => {
                    let mut contact = map_subject_to_contact(subj);
                    match self.contact_svc.create(&mut contact) {
                        Ok(()) => {
                            subject_map.insert(subj.id, contact.id);
                            self.log_import("subject", subj.id, "contact", contact.id);
                            result.contacts_created += 1;
                        }
                        Err(e) => {
                            result.errors.push(format!("contact {}: {}", subj.id, e));
                        }
                    }
                }
                _ => {
                    result.contacts_skipped += 1;
                }
            }
        }

        // --- Import invoices ---
        for inv in &invoices {
            let preview = self.preview_invoice(inv, &subject_map);
            if preview.status != "new" {
                result.invoices_skipped += 1;
                continue;
            }

            let mut invoice = map_fakturoid_invoice(inv, &subject_map);
            if invoice.customer_id == 0 {
                result
                    .errors
                    .push(format!("invoice {}: customer not resolved", inv.id));
                continue;
            }
            // Resolve customer via subject_map
            if let Some(&local_id) = subject_map.get(&invoice.customer_id) {
                invoice.customer_id = local_id;
            }

            match self.invoice_svc.create(&mut invoice) {
                Ok(()) => {
                    // Update status if not Draft
                    if invoice.status != InvoiceStatus::Draft {
                        let _ = self
                            .invoice_repo
                            .update_status(invoice.id, &invoice.status.to_string());
                    }
                    self.log_import("invoice", inv.id, "invoice", invoice.id);
                    result.invoices_created += 1;

                    // Download attachments
                    if download_attachments && let Some(fak_inv) = invoice_by_fak_id.get(&inv.id) {
                        self.download_invoice_attachments(
                            &client,
                            invoice.id,
                            &fak_inv.attachments,
                            &mut result,
                        );
                    }
                }
                Err(e) => {
                    result.errors.push(format!("invoice {}: {}", inv.id, e));
                }
            }
        }

        // --- Import expenses ---
        for exp in &expenses {
            let preview = self.preview_expense(exp, &subject_map);
            if preview.status != "new" {
                result.expenses_skipped += 1;
                continue;
            }

            let mut expense = map_fakturoid_expense(exp, &subject_map);
            // Resolve vendor via subject_map
            if let Some(vendor_id) = expense.vendor_id
                && let Some(&local_id) = subject_map.get(&vendor_id)
            {
                expense.vendor_id = Some(local_id);
            }

            match self.expense_svc.create(&mut expense) {
                Ok(()) => {
                    self.log_import("expense", exp.id, "expense", expense.id);
                    result.expenses_created += 1;

                    // Download attachments
                    if download_attachments && let Some(fak_exp) = expense_by_fak_id.get(&exp.id) {
                        self.download_expense_attachments(
                            &client,
                            expense.id,
                            &fak_exp.attachments,
                            &mut result,
                        );
                    }
                }
                Err(e) => {
                    result.errors.push(format!("expense {}: {}", exp.id, e));
                }
            }
        }

        info!(
            "Fakturoid import complete: {} contacts, {} invoices, {} expenses created",
            result.contacts_created, result.invoices_created, result.expenses_created
        );

        Ok(result)
    }

    fn preview_contact(&self, subj: &fakturoid::Subject) -> PreviewItem {
        // Check import log
        if let Ok(Some(log_entry)) = self.import_repo.find_by_fakturoid_id("subject", subj.id) {
            return PreviewItem {
                status: "duplicate".to_string(),
                existing_id: Some(log_entry.local_id),
            };
        }

        // Check by ICO
        if !subj.registration_no.is_empty()
            && let Ok(existing) = self.contact_repo.find_by_ico(&subj.registration_no)
        {
            return PreviewItem {
                status: "duplicate".to_string(),
                existing_id: Some(existing.id),
            };
        }

        // Check by exact name
        if !subj.name.is_empty() {
            let filter = zfaktury_domain::ContactFilter {
                search: subj.name.clone(),
                limit: 1,
                ..Default::default()
            };
            if let Ok((contacts, _)) = self.contact_repo.list(&filter)
                && !contacts.is_empty()
                && contacts[0].name == subj.name
            {
                return PreviewItem {
                    status: "duplicate".to_string(),
                    existing_id: Some(contacts[0].id),
                };
            }
        }

        PreviewItem {
            status: "new".to_string(),
            existing_id: None,
        }
    }

    fn preview_invoice(
        &self,
        inv: &fakturoid::Invoice,
        _subject_map: &HashMap<i64, i64>,
    ) -> PreviewItem {
        // Check import log
        if let Ok(Some(log_entry)) = self.import_repo.find_by_fakturoid_id("invoice", inv.id) {
            return PreviewItem {
                status: "duplicate".to_string(),
                existing_id: Some(log_entry.local_id),
            };
        }

        // Check by invoice number
        if !inv.number.is_empty() {
            let filter = zfaktury_domain::InvoiceFilter {
                search: inv.number.clone(),
                limit: 1,
                ..Default::default()
            };
            if let Ok((invoices, _)) = self.invoice_repo.list(&filter)
                && !invoices.is_empty()
                && invoices[0].invoice_number == inv.number
            {
                return PreviewItem {
                    status: "duplicate".to_string(),
                    existing_id: Some(invoices[0].id),
                };
            }
        }

        PreviewItem {
            status: "new".to_string(),
            existing_id: None,
        }
    }

    fn preview_expense(
        &self,
        exp: &fakturoid::Expense,
        subject_map: &HashMap<i64, i64>,
    ) -> PreviewItem {
        // Check import log
        if let Ok(Some(log_entry)) = self.import_repo.find_by_fakturoid_id("expense", exp.id) {
            return PreviewItem {
                status: "duplicate".to_string(),
                existing_id: Some(log_entry.local_id),
            };
        }

        // Check by expense number + vendor + date
        if !exp.original_number.is_empty() {
            let issue_date = parse_date(&exp.issued_on);
            let vendor_id = if exp.subject_id > 0 {
                subject_map
                    .get(&exp.subject_id)
                    .copied()
                    .or(Some(exp.subject_id))
            } else {
                None
            };

            let filter = zfaktury_domain::ExpenseFilter {
                search: exp.original_number.clone(),
                vendor_id,
                date_from: issue_date,
                date_to: issue_date,
                limit: 1,
                ..Default::default()
            };
            if let Ok((expenses, _)) = self.expense_repo.list(&filter)
                && !expenses.is_empty()
                && expenses[0].expense_number == exp.original_number
            {
                return PreviewItem {
                    status: "duplicate".to_string(),
                    existing_id: Some(expenses[0].id),
                };
            }
        }

        PreviewItem {
            status: "new".to_string(),
            existing_id: None,
        }
    }

    fn log_import(
        &self,
        fakturoid_entity_type: &str,
        fakturoid_id: i64,
        local_entity_type: &str,
        local_id: i64,
    ) {
        let mut entry = FakturoidImportLog {
            id: 0,
            fakturoid_entity_type: fakturoid_entity_type.to_string(),
            fakturoid_id,
            local_entity_type: local_entity_type.to_string(),
            local_id,
            imported_at: chrono::Local::now().naive_local(),
        };
        if let Err(e) = self.import_repo.create(&mut entry) {
            warn!(
                "Failed to log import: {fakturoid_entity_type} {fakturoid_id} -> {local_entity_type} {local_id}: {e}"
            );
        }
    }

    fn download_invoice_attachments(
        &self,
        client: &FakturoidClient,
        invoice_id: i64,
        attachments: &[fakturoid::Attachment],
        result: &mut FakturoidImportResult,
    ) {
        for att in attachments {
            match client.download_attachment(&att.download_url) {
                Ok((data, content_type)) => {
                    let filename = if att.filename.is_empty() {
                        format!("attachment_{}", att.id)
                    } else {
                        att.filename.clone()
                    };
                    let ct =
                        if content_type.is_empty() || content_type == "application/octet-stream" {
                            att.content_type.clone()
                        } else {
                            content_type
                        };

                    // Write file to disk
                    let doc_dir = format!(
                        "{}/invoice_documents/{}",
                        self.invoice_document_svc.data_dir(),
                        invoice_id
                    );
                    if std::fs::create_dir_all(&doc_dir).is_err() {
                        warn!("Failed to create directory: {doc_dir}");
                        result.attachments_skipped += 1;
                        continue;
                    }
                    let file_path = format!("{}/{}", doc_dir, filename);
                    if std::fs::write(&file_path, &data).is_err() {
                        warn!("Failed to write file: {file_path}");
                        result.attachments_skipped += 1;
                        continue;
                    }

                    let now = chrono::Local::now().naive_local();
                    let mut doc = InvoiceDocument {
                        id: 0,
                        invoice_id,
                        filename,
                        content_type: ct,
                        storage_path: file_path,
                        size: data.len() as i64,
                        created_at: now,
                        deleted_at: None,
                    };
                    if let Err(e) = self.invoice_document_svc.create_record(&mut doc) {
                        warn!("Failed to store invoice attachment record: {e}");
                        result.attachments_skipped += 1;
                        continue;
                    }
                    result.attachments_downloaded += 1;
                }
                Err(e) => {
                    warn!("Failed to download invoice attachment {}: {e}", att.id);
                    result.attachments_skipped += 1;
                }
            }
            thread::sleep(ATTACHMENT_DELAY);
        }
    }

    fn download_expense_attachments(
        &self,
        client: &FakturoidClient,
        expense_id: i64,
        attachments: &[fakturoid::Attachment],
        result: &mut FakturoidImportResult,
    ) {
        for att in attachments {
            match client.download_attachment(&att.download_url) {
                Ok((data, content_type)) => {
                    let filename = if att.filename.is_empty() {
                        format!("attachment_{}", att.id)
                    } else {
                        att.filename.clone()
                    };
                    let ct =
                        if content_type.is_empty() || content_type == "application/octet-stream" {
                            att.content_type.clone()
                        } else {
                            content_type
                        };

                    // Write file to disk
                    let doc_dir = format!(
                        "{}/expense_documents/{}",
                        self.document_svc.data_dir(),
                        expense_id
                    );
                    if std::fs::create_dir_all(&doc_dir).is_err() {
                        warn!("Failed to create directory: {doc_dir}");
                        result.attachments_skipped += 1;
                        continue;
                    }
                    let file_path = format!("{}/{}", doc_dir, filename);
                    if std::fs::write(&file_path, &data).is_err() {
                        warn!("Failed to write file: {file_path}");
                        result.attachments_skipped += 1;
                        continue;
                    }

                    let now = chrono::Local::now().naive_local();
                    let mut doc = ExpenseDocument {
                        id: 0,
                        expense_id,
                        filename,
                        content_type: ct,
                        storage_path: file_path,
                        size: data.len() as i64,
                        created_at: now,
                        deleted_at: None,
                    };
                    if let Err(e) = self.document_svc.create_record(&mut doc) {
                        warn!("Failed to store expense attachment record: {e}");
                        result.attachments_skipped += 1;
                        continue;
                    }
                    result.attachments_downloaded += 1;
                }
                Err(e) => {
                    warn!("Failed to download expense attachment {}: {e}", att.id);
                    result.attachments_skipped += 1;
                }
            }
            thread::sleep(ATTACHMENT_DELAY);
        }
    }
}

/// Internal preview item (not exposed to domain).
struct PreviewItem {
    status: String,
    existing_id: Option<i64>,
}

fn parse_date(s: &str) -> Option<NaiveDate> {
    NaiveDate::parse_from_str(s, "%Y-%m-%d").ok()
}

fn map_subject_to_contact(subj: &fakturoid::Subject) -> zfaktury_domain::Contact {
    let now = chrono::Local::now().naive_local();

    let (bank_account, bank_code) = if !subj.bank_account.is_empty() {
        let parts: Vec<&str> = subj.bank_account.splitn(2, '/').collect();
        if parts.len() == 2 {
            (parts[0].to_string(), parts[1].to_string())
        } else {
            (parts[0].to_string(), String::new())
        }
    } else {
        (String::new(), String::new())
    };

    zfaktury_domain::Contact {
        id: 0,
        contact_type: ContactType::Company,
        name: subj.name.clone(),
        ico: subj.registration_no.clone(),
        dic: subj.vat_no.clone(),
        street: subj.street.clone(),
        city: subj.city.clone(),
        zip: subj.zip.clone(),
        country: subj.country.clone(),
        email: subj.email.clone(),
        phone: subj.phone.clone(),
        web: subj.web.clone(),
        bank_account,
        bank_code,
        iban: subj.iban.clone(),
        swift: String::new(),
        payment_terms_days: subj.due,
        tags: String::new(),
        notes: String::new(),
        is_favorite: false,
        vat_unreliable_at: None,
        created_at: now,
        updated_at: now,
        deleted_at: None,
    }
}

fn map_fakturoid_invoice(
    inv: &fakturoid::Invoice,
    subject_map: &HashMap<i64, i64>,
) -> zfaktury_domain::Invoice {
    let now = chrono::Local::now().naive_local();
    let today = chrono::Local::now().date_naive();

    let invoice_type = match inv.document_type.as_str() {
        "proforma" => InvoiceType::Proforma,
        "correction" => InvoiceType::CreditNote,
        _ => InvoiceType::Regular,
    };

    let status = match inv.status.as_str() {
        "paid" => InvoiceStatus::Paid,
        "overdue" => InvoiceStatus::Overdue,
        "cancelled" | "uncollectible" => InvoiceStatus::Cancelled,
        _ => InvoiceStatus::Sent,
    };

    let payment_method = match inv.payment_method.as_str() {
        "cash" => "cash",
        "card" => "card",
        "cod" | "paypal" | "custom" => "other",
        _ => "bank_transfer",
    }
    .to_string();

    let issue_date = parse_date(&inv.issued_on).unwrap_or(today);
    let due_date = parse_date(&inv.due_on).unwrap_or(today);
    let delivery_date = parse_date(&inv.taxable_fulfillment_due).unwrap_or(issue_date);

    // Resolve customer
    let customer_id = if let Some(&local_id) = subject_map.get(&inv.subject_id) {
        local_id
    } else {
        inv.subject_id
    };

    // Paid date from first payment
    let (paid_at, paid_amount) = if !inv.payments.is_empty() && !inv.payments[0].paid_on.is_empty()
    {
        if let Some(paid_date) = parse_date(&inv.payments[0].paid_on) {
            (
                Some(paid_date.and_hms_opt(0, 0, 0).unwrap()),
                Amount::from_float(inv.total.0),
            )
        } else {
            (None, Amount::ZERO)
        }
    } else {
        (None, Amount::ZERO)
    };

    // Map line items
    let items: Vec<InvoiceItem> = inv
        .lines
        .iter()
        .enumerate()
        .map(|(i, line)| InvoiceItem {
            id: 0,
            invoice_id: 0,
            description: line.name.clone(),
            quantity: Amount::from_float(line.quantity.0),
            unit: line.unit_name.clone(),
            unit_price: Amount::from_float(line.unit_price.0),
            vat_rate_percent: line.vat_rate.0 as i32,
            vat_amount: Amount::ZERO,
            total_amount: Amount::ZERO,
            sort_order: (i + 1) as i32,
        })
        .collect();

    let mut invoice = zfaktury_domain::Invoice {
        id: 0,
        sequence_id: 0,
        invoice_number: inv.number.clone(),
        invoice_type,
        status,
        issue_date,
        due_date,
        delivery_date,
        variable_symbol: inv.variable_symbol.clone(),
        constant_symbol: String::new(),
        customer_id,
        customer: None,
        currency_code: inv.currency.clone(),
        exchange_rate: Amount::from_float(inv.exchange_rate.0),
        payment_method,
        bank_account: String::new(),
        bank_code: String::new(),
        iban: String::new(),
        swift: String::new(),
        subtotal_amount: Amount::from_float(inv.subtotal.0),
        vat_amount: Amount::ZERO,
        total_amount: Amount::from_float(inv.total.0),
        paid_amount,
        notes: inv.note.clone(),
        internal_notes: String::new(),
        related_invoice_id: None,
        relation_type: zfaktury_domain::RelationType::None,
        sent_at: None,
        paid_at,
        items,
        created_at: now,
        updated_at: now,
        deleted_at: None,
    };

    invoice.calculate_totals();
    invoice
}

fn map_fakturoid_expense(
    exp: &fakturoid::Expense,
    subject_map: &HashMap<i64, i64>,
) -> zfaktury_domain::Expense {
    let now = chrono::Local::now().naive_local();
    let today = chrono::Local::now().date_naive();

    let issue_date = parse_date(&exp.issued_on).unwrap_or(today);

    let payment_method = match exp.payment_method.as_str() {
        "cash" => "cash",
        _ => "bank_transfer",
    }
    .to_string();

    // Description: prefer description field, fallback to first line name
    let description = if !exp.description.is_empty() {
        exp.description.clone()
    } else if !exp.lines.is_empty() && !exp.lines[0].name.is_empty() {
        exp.lines[0].name.clone()
    } else {
        "Import z Fakturoidu".to_string()
    };

    // Resolve vendor
    let vendor_id = if exp.subject_id > 0 {
        if let Some(&local_id) = subject_map.get(&exp.subject_id) {
            Some(local_id)
        } else {
            Some(exp.subject_id)
        }
    } else {
        None
    };

    // Map line items
    let items: Vec<ExpenseItem> = exp
        .lines
        .iter()
        .enumerate()
        .map(|(i, line)| ExpenseItem {
            id: 0,
            expense_id: 0,
            description: line.name.clone(),
            quantity: Amount::from_float(line.quantity.0),
            unit: "ks".to_string(),
            unit_price: Amount::from_float(line.unit_price.0),
            vat_rate_percent: line.vat_rate.0 as i32,
            vat_amount: Amount::ZERO,
            total_amount: Amount::ZERO,
            sort_order: (i + 1) as i32,
        })
        .collect();

    let mut expense = Expense {
        id: 0,
        vendor_id,
        vendor: None,
        expense_number: exp.original_number.clone(),
        category: String::new(),
        description,
        issue_date,
        amount: Amount::from_float(exp.total.0),
        currency_code: exp.currency.clone(),
        exchange_rate: Amount::from_float(exp.exchange_rate.0),
        vat_rate_percent: 0,
        vat_amount: Amount::ZERO,
        is_tax_deductible: true,
        business_percent: 100,
        payment_method,
        document_path: String::new(),
        notes: exp.private_note.clone(),
        tax_reviewed_at: None,
        items,
        created_at: now,
        updated_at: now,
        deleted_at: None,
    };

    expense.calculate_totals();
    expense
}
