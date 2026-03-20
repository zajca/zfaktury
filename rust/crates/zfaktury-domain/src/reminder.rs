use chrono::NaiveDateTime;

/// A payment reminder sent for an overdue invoice.
#[derive(Debug, Clone)]
pub struct PaymentReminder {
    pub id: i64,
    pub invoice_id: i64,
    pub reminder_number: i32,
    pub sent_at: NaiveDateTime,
    pub sent_to: String,
    pub subject: String,
    pub body_preview: String,
    pub created_at: NaiveDateTime,
}
