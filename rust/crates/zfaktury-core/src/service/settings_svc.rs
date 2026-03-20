use std::collections::{HashMap, HashSet};
use std::sync::Arc;

use zfaktury_domain::{
    DomainError, PDFSettings, SETTING_BANK_ACCOUNT, SETTING_BANK_CODE, SETTING_CITY,
    SETTING_COMPANY_NAME, SETTING_CSSZ_CODE, SETTING_DIC, SETTING_EMAIL,
    SETTING_EMAIL_ATTACH_ISDOC, SETTING_EMAIL_ATTACH_PDF, SETTING_EMAIL_BODY_TPL,
    SETTING_EMAIL_SUBJECT_TPL, SETTING_FINANCNI_URAD_CODE, SETTING_FIRST_NAME,
    SETTING_HEALTH_INS_CODE, SETTING_HOUSE_NUMBER, SETTING_IBAN, SETTING_ICO, SETTING_LAST_NAME,
    SETTING_OKEC, SETTING_PDF_ACCENT_COLOR, SETTING_PDF_FONT_SIZE, SETTING_PDF_FOOTER_TEXT,
    SETTING_PDF_LOGO_PATH, SETTING_PDF_SHOW_BANK_DETAILS, SETTING_PDF_SHOW_QR, SETTING_PHONE,
    SETTING_PRAC_UFO, SETTING_STREET, SETTING_SWIFT, SETTING_UFO_CODE, SETTING_VAT_REGISTERED,
    SETTING_ZIP,
};

use super::audit_svc::AuditService;
use crate::repository::traits::SettingsRepo;

/// All known setting keys.
fn known_keys() -> HashSet<&'static str> {
    [
        SETTING_COMPANY_NAME,
        SETTING_ICO,
        SETTING_DIC,
        SETTING_VAT_REGISTERED,
        SETTING_STREET,
        SETTING_CITY,
        SETTING_ZIP,
        SETTING_EMAIL,
        SETTING_PHONE,
        SETTING_BANK_ACCOUNT,
        SETTING_BANK_CODE,
        SETTING_IBAN,
        SETTING_SWIFT,
        SETTING_EMAIL_ATTACH_PDF,
        SETTING_EMAIL_ATTACH_ISDOC,
        SETTING_EMAIL_SUBJECT_TPL,
        SETTING_EMAIL_BODY_TPL,
        SETTING_FIRST_NAME,
        SETTING_LAST_NAME,
        SETTING_HOUSE_NUMBER,
        SETTING_HEALTH_INS_CODE,
        SETTING_FINANCNI_URAD_CODE,
        SETTING_CSSZ_CODE,
        SETTING_UFO_CODE,
        SETTING_PRAC_UFO,
        SETTING_OKEC,
        SETTING_PDF_LOGO_PATH,
        SETTING_PDF_ACCENT_COLOR,
        SETTING_PDF_FOOTER_TEXT,
        SETTING_PDF_SHOW_QR,
        SETTING_PDF_SHOW_BANK_DETAILS,
        SETTING_PDF_FONT_SIZE,
    ]
    .into_iter()
    .collect()
}

fn validate_key(key: &str) -> Result<(), DomainError> {
    if key.is_empty() {
        return Err(DomainError::InvalidInput);
    }
    if !known_keys().contains(key) {
        return Err(DomainError::InvalidInput);
    }
    Ok(())
}

/// Service for application settings (key-value store).
pub struct SettingsService {
    repo: Arc<dyn SettingsRepo + Send + Sync>,
    audit: Option<Arc<AuditService>>,
}

impl SettingsService {
    pub fn new(
        repo: Arc<dyn SettingsRepo + Send + Sync>,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self { repo, audit }
    }

    /// Retrieves all settings.
    pub fn get_all(&self) -> Result<HashMap<String, String>, DomainError> {
        self.repo.get_all()
    }

    /// Retrieves a single setting by key.
    pub fn get(&self, key: &str) -> Result<String, DomainError> {
        validate_key(key)?;
        self.repo.get(key)
    }

    /// Upserts a single setting.
    pub fn set(&self, key: &str, value: &str) -> Result<(), DomainError> {
        validate_key(key)?;
        let old_val = self.repo.get(key).unwrap_or_default();
        self.repo.set(key, value)?;
        if let Some(ref audit) = self.audit {
            let old_json = serde_json::json!({ key: old_val }).to_string();
            let new_json = serde_json::json!({ key: value }).to_string();
            audit.log("settings", 0, "set", Some(&old_json), Some(&new_json));
        }
        Ok(())
    }

    /// Upserts multiple settings at once.
    pub fn set_bulk(&self, settings: &HashMap<String, String>) -> Result<(), DomainError> {
        for key in settings.keys() {
            validate_key(key)?;
        }
        let old_settings = self.repo.get_all().unwrap_or_default();
        self.repo.set_bulk(settings)?;
        if let Some(ref audit) = self.audit {
            let mut changed = HashMap::new();
            let mut old = HashMap::new();
            for (k, v) in settings {
                let old_v = old_settings.get(k.as_str()).cloned().unwrap_or_default();
                if old_v != *v {
                    changed.insert(k.clone(), v.clone());
                    old.insert(k.clone(), old_v);
                }
            }
            if !changed.is_empty() {
                let old_json = serde_json::to_string(&old).unwrap_or_default();
                let new_json = serde_json::to_string(&changed).unwrap_or_default();
                audit.log("settings", 0, "set_bulk", Some(&old_json), Some(&new_json));
            }
        }
        Ok(())
    }

    /// Retrieves PDF template settings with defaults.
    pub fn get_pdf_settings(&self) -> Result<PDFSettings, DomainError> {
        let all = self.repo.get_all()?;

        let accent_color = all
            .get(SETTING_PDF_ACCENT_COLOR)
            .filter(|s| !s.is_empty())
            .cloned()
            .or_else(|| Some("#2563eb".to_string()));

        let font_size = all
            .get(SETTING_PDF_FONT_SIZE)
            .and_then(|s| s.parse::<f32>().ok());

        let footer_text = all.get(SETTING_PDF_FOOTER_TEXT).cloned();
        let logo_path = all.get(SETTING_PDF_LOGO_PATH).cloned();

        let show_qr = all
            .get(SETTING_PDF_SHOW_QR)
            .map(|v| v == "true")
            .unwrap_or(true);

        let show_bank_details = all
            .get(SETTING_PDF_SHOW_BANK_DETAILS)
            .map(|v| v == "true")
            .unwrap_or(true);

        Ok(PDFSettings {
            accent_color,
            font_size,
            footer_text,
            logo_path,
            show_qr,
            show_bank_details,
        })
    }

    /// Persists PDF template settings.
    pub fn save_pdf_settings(&self, ps: &PDFSettings) -> Result<(), DomainError> {
        let mut settings = HashMap::new();
        settings.insert(
            SETTING_PDF_LOGO_PATH.to_string(),
            ps.logo_path.clone().unwrap_or_default(),
        );
        settings.insert(
            SETTING_PDF_ACCENT_COLOR.to_string(),
            ps.accent_color.clone().unwrap_or_default(),
        );
        settings.insert(
            SETTING_PDF_FOOTER_TEXT.to_string(),
            ps.footer_text.clone().unwrap_or_default(),
        );
        settings.insert(SETTING_PDF_SHOW_QR.to_string(), ps.show_qr.to_string());
        settings.insert(
            SETTING_PDF_SHOW_BANK_DETAILS.to_string(),
            ps.show_bank_details.to_string(),
        );
        settings.insert(
            SETTING_PDF_FONT_SIZE.to_string(),
            ps.font_size.map(|f| f.to_string()).unwrap_or_default(),
        );

        self.repo.set_bulk(&settings)?;

        if let Some(ref audit) = self.audit {
            let new_json = serde_json::to_string(&settings).unwrap_or_default();
            audit.log("settings", 0, "set_pdf_settings", None, Some(&new_json));
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::Mutex;

    struct MockSettingsRepo {
        data: Mutex<HashMap<String, String>>,
    }

    impl MockSettingsRepo {
        fn new() -> Self {
            Self {
                data: Mutex::new(HashMap::new()),
            }
        }
    }

    impl SettingsRepo for MockSettingsRepo {
        fn get_all(&self) -> Result<HashMap<String, String>, DomainError> {
            Ok(self.data.lock().unwrap().clone())
        }

        fn get(&self, key: &str) -> Result<String, DomainError> {
            self.data
                .lock()
                .unwrap()
                .get(key)
                .cloned()
                .ok_or(DomainError::NotFound)
        }

        fn set(&self, key: &str, value: &str) -> Result<(), DomainError> {
            self.data
                .lock()
                .unwrap()
                .insert(key.to_string(), value.to_string());
            Ok(())
        }

        fn set_bulk(&self, settings: &HashMap<String, String>) -> Result<(), DomainError> {
            let mut data = self.data.lock().unwrap();
            for (k, v) in settings {
                data.insert(k.clone(), v.clone());
            }
            Ok(())
        }
    }

    #[test]
    fn test_set_and_get() {
        let repo = Arc::new(MockSettingsRepo::new());
        let svc = SettingsService::new(repo, None);

        svc.set(SETTING_COMPANY_NAME, "Test Company").unwrap();
        let val = svc.get(SETTING_COMPANY_NAME).unwrap();
        assert_eq!(val, "Test Company");
    }

    #[test]
    fn test_unknown_key_rejected() {
        let repo = Arc::new(MockSettingsRepo::new());
        let svc = SettingsService::new(repo, None);

        let result = svc.set("unknown_key", "value");
        assert!(result.is_err());
    }

    #[test]
    fn test_set_bulk() {
        let repo = Arc::new(MockSettingsRepo::new());
        let svc = SettingsService::new(repo, None);

        let mut settings = HashMap::new();
        settings.insert(SETTING_ICO.to_string(), "12345678".to_string());
        settings.insert(SETTING_DIC.to_string(), "CZ12345678".to_string());
        svc.set_bulk(&settings).unwrap();

        assert_eq!(svc.get(SETTING_ICO).unwrap(), "12345678");
        assert_eq!(svc.get(SETTING_DIC).unwrap(), "CZ12345678");
    }

    #[test]
    fn test_pdf_settings_defaults() {
        let repo = Arc::new(MockSettingsRepo::new());
        let svc = SettingsService::new(repo, None);

        let ps = svc.get_pdf_settings().unwrap();
        assert_eq!(ps.accent_color.as_deref(), Some("#2563eb"));
        assert!(ps.show_qr);
        assert!(ps.show_bank_details);
    }
}
