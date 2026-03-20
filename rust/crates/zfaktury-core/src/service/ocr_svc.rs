use std::sync::Arc;

use zfaktury_domain::{DomainError, OCRResult};

use super::document_svc::DocumentService;

/// Trait for OCR providers, abstracted for testability.
pub trait OCRProvider: Send + Sync {
    fn process_image(&self, data: &[u8], content_type: &str) -> Result<OCRResult, DomainError>;
}

/// Service for OCR processing of expense documents.
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

    /// Processes a document by ID through OCR.
    pub fn process_document(&self, document_id: i64) -> Result<OCRResult, DomainError> {
        if document_id == 0 {
            return Err(DomainError::InvalidInput);
        }
        let doc = self.documents.get_by_id(document_id)?;

        // Validate content type.
        let supported = ["image/jpeg", "image/png", "application/pdf"];
        if !supported.contains(&doc.content_type.as_str()) {
            return Err(DomainError::InvalidInput);
        }

        // In the full implementation, we would read the file from doc.storage_path
        // and pass it to the OCR provider. For now, we return an error indicating
        // that the file read must happen at a higher layer (handler/app).
        Err(DomainError::InvalidInput)
    }
}
