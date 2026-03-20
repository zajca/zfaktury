use chrono::NaiveDateTime;
use std::fmt;

/// A single audit trail record for entity changes.
#[derive(Debug, Clone)]
pub struct AuditLogEntry {
    pub id: i64,
    pub entity_type: String,
    pub entity_id: i64,
    pub action: String,
    pub old_values: String,
    pub new_values: String,
    pub created_at: NaiveDateTime,
}

/// Filtering options for listing audit log entries.
#[derive(Debug, Clone)]
pub struct AuditLogFilter {
    pub entity_type: String,
    pub entity_id: Option<i64>,
    pub action: String,
    pub from: Option<NaiveDateTime>,
    pub to: Option<NaiveDateTime>,
    pub limit: i32,
    pub offset: i32,
}

/// Backup trigger type.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum BackupTrigger {
    Manual,
    Scheduled,
    CLI,
}

impl fmt::Display for BackupTrigger {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            BackupTrigger::Manual => write!(f, "manual"),
            BackupTrigger::Scheduled => write!(f, "scheduled"),
            BackupTrigger::CLI => write!(f, "cli"),
        }
    }
}

/// Backup status.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum BackupStatus {
    Running,
    Completed,
    Failed,
}

impl fmt::Display for BackupStatus {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            BackupStatus::Running => write!(f, "running"),
            BackupStatus::Completed => write!(f, "completed"),
            BackupStatus::Failed => write!(f, "failed"),
        }
    }
}

/// A backup operation record.
#[derive(Debug, Clone)]
pub struct BackupRecord {
    pub id: i64,
    pub filename: String,
    pub status: BackupStatus,
    pub trigger: BackupTrigger,
    pub destination: String,
    pub size_bytes: i64,
    pub file_count: i32,
    pub db_migration_version: i64,
    pub duration_ms: i64,
    pub error_message: String,
    pub created_at: NaiveDateTime,
    pub completed_at: Option<NaiveDateTime>,
}
