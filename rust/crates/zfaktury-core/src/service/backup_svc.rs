use std::sync::Arc;

use zfaktury_domain::{BackupRecord, DomainError};

use crate::repository::traits::BackupHistoryRepo;

/// Service for backup management.
pub struct BackupService {
    repo: Arc<dyn BackupHistoryRepo + Send + Sync>,
    data_dir: String,
}

impl BackupService {
    pub fn new(repo: Arc<dyn BackupHistoryRepo + Send + Sync>, data_dir: String) -> Self {
        Self { repo, data_dir }
    }

    pub fn data_dir(&self) -> &str {
        &self.data_dir
    }

    /// Lists all backup records.
    pub fn list_backups(&self) -> Result<Vec<BackupRecord>, DomainError> {
        self.repo.list()
    }

    /// Gets a single backup record by ID.
    pub fn get_backup(&self, id: i64) -> Result<BackupRecord, DomainError> {
        self.repo.get_by_id(id)
    }

    /// Deletes a backup record.
    pub fn delete_backup(&self, id: i64) -> Result<(), DomainError> {
        self.repo.get_by_id(id)?; // verify exists
        self.repo.delete(id)
    }
}
