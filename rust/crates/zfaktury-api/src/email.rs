use lettre::message::header::ContentType;
use lettre::message::{Attachment, MultiPart, SinglePart};
use lettre::transport::smtp::authentication::Credentials;
use lettre::transport::smtp::client::{Tls, TlsParameters};
use lettre::{Message, SmtpTransport, Transport};

use crate::error::{ApiError, Result};

/// SMTP server configuration.
#[derive(Debug, Clone)]
pub struct SmtpConfig {
    pub host: String,
    pub port: u16,
    pub username: String,
    pub password: String,
    pub from: String,
}

/// A file attached to an email message.
#[derive(Debug, Clone)]
pub struct EmailAttachment {
    pub filename: String,
    pub content_type: String,
    pub data: Vec<u8>,
}

/// All data needed to send an email.
#[derive(Debug, Clone)]
pub struct EmailMessage {
    pub to: Vec<String>,
    pub cc: Vec<String>,
    pub bcc: Vec<String>,
    pub subject: String,
    pub body_html: String,
    pub body_text: String,
    pub attachments: Vec<EmailAttachment>,
}

/// Sends emails via SMTP using lettre.
pub struct EmailSender {
    config: SmtpConfig,
}

impl EmailSender {
    /// Create a new email sender with the given SMTP configuration.
    pub fn new(config: SmtpConfig) -> Self {
        Self { config }
    }

    /// Returns true if the SMTP host is configured.
    pub fn is_configured(&self) -> bool {
        !self.config.host.is_empty()
    }

    /// Send an email message.
    pub fn send(&self, message: EmailMessage) -> Result<()> {
        if !self.is_configured() {
            return Err(ApiError::SmtpError(
                "SMTP not configured (host is empty)".to_string(),
            ));
        }

        let email = self.build_message(&message)?;
        let transport = self.build_transport()?;

        transport
            .send(&email)
            .map_err(|e| ApiError::SmtpError(format!("sending email: {}", e)))?;

        log::info!(
            "Email sent successfully to {} recipients, subject: {}",
            message.to.len(),
            message.subject
        );

        Ok(())
    }

    /// Build a lettre Message from our EmailMessage.
    fn build_message(&self, msg: &EmailMessage) -> Result<Message> {
        let from = self
            .config
            .from
            .parse()
            .map_err(|e| ApiError::SmtpError(format!("invalid from address: {}", e)))?;

        let mut builder = Message::builder().from(from).subject(&msg.subject);

        for to in &msg.to {
            let addr = to
                .parse()
                .map_err(|e| ApiError::SmtpError(format!("invalid to address '{}': {}", to, e)))?;
            builder = builder.to(addr);
        }

        for cc in &msg.cc {
            let addr = cc
                .parse()
                .map_err(|e| ApiError::SmtpError(format!("invalid cc address '{}': {}", cc, e)))?;
            builder = builder.cc(addr);
        }

        for bcc in &msg.bcc {
            let addr = bcc.parse().map_err(|e| {
                ApiError::SmtpError(format!("invalid bcc address '{}': {}", bcc, e))
            })?;
            builder = builder.bcc(addr);
        }

        // Build multipart body.
        let alternative = MultiPart::alternative()
            .singlepart(
                SinglePart::builder()
                    .header(ContentType::TEXT_PLAIN)
                    .body(msg.body_text.clone()),
            )
            .singlepart(
                SinglePart::builder()
                    .header(ContentType::TEXT_HTML)
                    .body(msg.body_html.clone()),
            );

        let body = if msg.attachments.is_empty() {
            builder
                .multipart(alternative)
                .map_err(|e| ApiError::SmtpError(format!("building message: {}", e)))?
        } else {
            let mut mixed = MultiPart::mixed().multipart(alternative);

            for att in &msg.attachments {
                let ct = att
                    .content_type
                    .parse::<ContentType>()
                    .unwrap_or(ContentType::parse("application/octet-stream").unwrap());
                let attachment = Attachment::new(att.filename.clone()).body(att.data.clone(), ct);
                mixed = mixed.singlepart(attachment);
            }

            builder
                .multipart(mixed)
                .map_err(|e| ApiError::SmtpError(format!("building message: {}", e)))?
        };

        Ok(body)
    }

    /// Build the SMTP transport based on port configuration.
    fn build_transport(&self) -> Result<SmtpTransport> {
        let creds = if self.config.username.is_empty() {
            None
        } else {
            Some(Credentials::new(
                self.config.username.clone(),
                self.config.password.clone(),
            ))
        };

        let tls_params = TlsParameters::new(self.config.host.clone())
            .map_err(|e| ApiError::SmtpError(format!("TLS parameters: {}", e)))?;

        let transport = match self.config.port {
            465 => {
                // Implicit TLS.
                let mut builder = SmtpTransport::relay(&self.config.host)
                    .map_err(|e| ApiError::SmtpError(format!("relay setup: {}", e)))?
                    .port(465)
                    .tls(Tls::Wrapper(tls_params));
                if let Some(c) = creds {
                    builder = builder.credentials(c);
                }
                builder.build()
            }
            587 => {
                // STARTTLS.
                let mut builder = SmtpTransport::starttls_relay(&self.config.host)
                    .map_err(|e| ApiError::SmtpError(format!("STARTTLS relay setup: {}", e)))?
                    .port(587)
                    .tls(Tls::Required(tls_params));
                if let Some(c) = creds {
                    builder = builder.credentials(c);
                }
                builder.build()
            }
            port => {
                // Plain SMTP (e.g., localhost relay).
                let mut builder = SmtpTransport::builder_dangerous(&self.config.host).port(port);
                if let Some(c) = creds {
                    builder = builder.credentials(c);
                }
                builder.build()
            }
        };

        Ok(transport)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_config() -> SmtpConfig {
        SmtpConfig {
            host: "smtp.example.com".to_string(),
            port: 587,
            username: "user@example.com".to_string(),
            password: "password".to_string(),
            from: "sender@example.com".to_string(),
        }
    }

    #[test]
    fn test_is_configured() {
        let sender = EmailSender::new(test_config());
        assert!(sender.is_configured());

        let sender = EmailSender::new(SmtpConfig {
            host: String::new(),
            port: 587,
            username: String::new(),
            password: String::new(),
            from: String::new(),
        });
        assert!(!sender.is_configured());
    }

    #[test]
    fn test_send_not_configured() {
        let sender = EmailSender::new(SmtpConfig {
            host: String::new(),
            port: 587,
            username: String::new(),
            password: String::new(),
            from: String::new(),
        });

        let msg = EmailMessage {
            to: vec!["test@example.com".to_string()],
            cc: vec![],
            bcc: vec![],
            subject: "Test".to_string(),
            body_html: "<p>Hello</p>".to_string(),
            body_text: "Hello".to_string(),
            attachments: vec![],
        };

        let result = sender.send(msg);
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ApiError::SmtpError(_)));
    }

    #[test]
    fn test_build_message_simple() {
        let sender = EmailSender::new(test_config());

        let msg = EmailMessage {
            to: vec!["test@example.com".to_string()],
            cc: vec![],
            bcc: vec![],
            subject: "Test Subject".to_string(),
            body_html: "<p>Hello World</p>".to_string(),
            body_text: "Hello World".to_string(),
            attachments: vec![],
        };

        let email = sender.build_message(&msg).unwrap();
        let formatted = email.formatted();
        let formatted_str = String::from_utf8_lossy(&formatted);

        assert!(formatted_str.contains("To:"));
        assert!(formatted_str.contains("test@example.com"));
        assert!(formatted_str.contains("Subject: Test Subject"));
        assert!(formatted_str.contains("text/plain"));
        assert!(formatted_str.contains("text/html"));
    }

    #[test]
    fn test_build_message_with_attachments() {
        let sender = EmailSender::new(test_config());

        let msg = EmailMessage {
            to: vec!["test@example.com".to_string()],
            cc: vec!["cc@example.com".to_string()],
            bcc: vec!["bcc@example.com".to_string()],
            subject: "Invoice PDF".to_string(),
            body_html: "<p>Please find attached.</p>".to_string(),
            body_text: "Please find attached.".to_string(),
            attachments: vec![EmailAttachment {
                filename: "invoice.pdf".to_string(),
                content_type: "application/pdf".to_string(),
                data: vec![0x25, 0x50, 0x44, 0x46], // %PDF
            }],
        };

        let email = sender.build_message(&msg).unwrap();
        let formatted = email.formatted();
        let formatted_str = String::from_utf8_lossy(&formatted);

        assert!(formatted_str.contains("multipart/mixed"));
        assert!(formatted_str.contains("invoice.pdf"));
        assert!(formatted_str.contains("Cc:"));
    }

    #[test]
    fn test_build_message_unicode_subject() {
        let sender = EmailSender::new(test_config());

        let msg = EmailMessage {
            to: vec!["test@example.com".to_string()],
            cc: vec![],
            bcc: vec![],
            subject: "Faktura c. 2026-001".to_string(),
            body_html: "<p>Dobrý den</p>".to_string(),
            body_text: "Dobrý den".to_string(),
            attachments: vec![],
        };

        // Should not panic on non-ASCII.
        let _email = sender.build_message(&msg).unwrap();
    }

    #[test]
    fn test_build_transport_port_465() {
        let mut config = test_config();
        config.port = 465;
        let sender = EmailSender::new(config);
        let result = sender.build_transport();
        assert!(result.is_ok());
    }

    #[test]
    fn test_build_transport_port_587() {
        let sender = EmailSender::new(test_config());
        let result = sender.build_transport();
        assert!(result.is_ok());
    }

    #[test]
    fn test_build_transport_plain() {
        let mut config = test_config();
        config.port = 25;
        let sender = EmailSender::new(config);
        let result = sender.build_transport();
        assert!(result.is_ok());
    }
}
