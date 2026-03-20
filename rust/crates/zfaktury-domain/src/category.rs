use chrono::NaiveDateTime;

/// A category for classifying expenses.
#[derive(Debug, Clone)]
pub struct ExpenseCategory {
    pub id: i64,
    pub key: String,
    pub label_cs: String,
    pub label_en: String,
    pub color: String,
    pub sort_order: i32,
    pub is_default: bool,
    pub created_at: NaiveDateTime,
    pub deleted_at: Option<NaiveDateTime>,
}
