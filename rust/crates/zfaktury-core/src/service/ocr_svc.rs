use std::sync::Arc;

use zfaktury_domain::{DomainError, OCRResult};

use super::document_svc::DocumentService;

/// Trait for OCR providers, abstracted for testability.
pub trait OCRProvider: Send + Sync {
    fn process_image(&self, data: &[u8], content_type: &str) -> Result<OCRResult, DomainError>;
}

/// Service for OCR processing of expense documents.
#[allow(dead_code)]
pub struct OCRService {
    provider: Arc<dyn OCRProvider>,
    documents: Arc<DocumentService>,
}

impl OCRService {
    pub fn new(provider: Arc<dyn OCRProvider>, documents: Arc<DocumentService>) -> Self {
        Self {
            provider,
            documents,
        }
    }

    /// Process raw bytes through OCR without requiring a stored document.
    /// Used by views that have file bytes from rfd file picker.
    pub fn process_bytes(&self, data: &[u8], content_type: &str) -> Result<OCRResult, DomainError> {
        let supported = ["image/jpeg", "image/png", "application/pdf"];
        if !supported.contains(&content_type) {
            return Err(DomainError::InvalidInput);
        }
        if data.is_empty() {
            return Err(DomainError::InvalidInput);
        }
        self.provider.process_image(data, content_type)
    }

    /// Processes a stored document by ID through OCR.
    pub fn process_document(&self, document_id: i64) -> Result<OCRResult, DomainError> {
        if document_id == 0 {
            return Err(DomainError::InvalidInput);
        }
        let doc = self.documents.get_by_id(document_id)?;

        let supported = ["image/jpeg", "image/png", "application/pdf"];
        if !supported.contains(&doc.content_type.as_str()) {
            return Err(DomainError::InvalidInput);
        }

        let data = std::fs::read(&doc.storage_path).map_err(|e| {
            log::error!("reading document file {}: {e}", doc.storage_path);
            DomainError::NotFound
        })?;

        self.provider.process_image(&data, &doc.content_type)
    }
}
