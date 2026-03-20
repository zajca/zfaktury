use std::env;
use std::path::{Path, PathBuf};

use serde::Deserialize;
use thiserror::Error;

#[derive(Debug, Error)]
pub enum ConfigError {
    #[error("getting home directory")]
    HomeDir,

    #[error("reading config file {path}: {source}")]
    ReadFile {
        path: PathBuf,
        source: std::io::Error,
    },

    #[error("parsing config file {path}: {source}")]
    ParseFile {
        path: PathBuf,
        source: toml::de::Error,
    },
}

/// Application configuration loaded from config.toml.
#[derive(Debug, Clone, Deserialize)]
#[serde(default)]
pub struct Config {
    /// Override via `ZFAKTURY_DATA_DIR` env var.
    pub data_dir: Option<PathBuf>,
    pub database: DatabaseConfig,
    pub log: LogConfig,
    pub server: ServerConfig,
    pub backup: BackupConfig,
    pub smtp: Option<SmtpConfig>,
    pub fio: Option<FioConfig>,
    pub ocr: Option<OcrConfig>,
}

/// Database settings.
#[derive(Debug, Clone, Deserialize)]
#[serde(default)]
pub struct DatabaseConfig {
    /// Custom DB path. Defaults to `{data_dir}/zfaktury.db`.
    pub path: Option<PathBuf>,
    /// WAL mode (default) or "delete".
    pub journal_mode: String,
}

/// Logging settings.
#[derive(Debug, Clone, Deserialize)]
#[serde(default)]
pub struct LogConfig {
    /// Log file path. Empty/None = stderr only.
    pub path: Option<PathBuf>,
    /// Log level: debug, info, warn, error. Default: "info".
    pub level: String,
}

/// HTTP server settings.
#[derive(Debug, Clone, Deserialize)]
#[serde(default)]
pub struct ServerConfig {
    pub port: u16,
    pub dev: bool,
}

/// Backup settings.
#[derive(Debug, Clone, Deserialize)]
#[serde(default)]
pub struct BackupConfig {
    /// Backup directory. Defaults to `{data_dir}/backups`.
    pub destination: Option<PathBuf>,
    /// Cron expression, e.g. "0 2 * * *".
    pub schedule: Option<String>,
    /// Keep last N backups, 0 = keep all.
    pub retention_count: i32,
    /// S3-compatible storage configuration.
    pub s3: Option<S3Config>,
}

/// S3-compatible storage settings (AWS S3, MinIO, Backblaze B2, Cloudflare R2).
#[derive(Debug, Clone, Deserialize)]
pub struct S3Config {
    /// e.g. "s3.amazonaws.com" or "minio.example.com:9000"
    pub endpoint: String,
    /// e.g. "eu-central-1"
    pub region: Option<String>,
    /// Bucket name.
    pub bucket: String,
    pub access_key: String,
    pub secret_key: String,
    #[serde(default = "default_true")]
    pub use_ssl: bool,
}

/// SMTP email settings.
#[derive(Debug, Clone, Deserialize)]
pub struct SmtpConfig {
    pub host: String,
    #[serde(default = "default_smtp_port")]
    pub port: u16,
    pub username: String,
    pub password: String,
    pub from: String,
}

/// FIO Bank API settings.
#[derive(Debug, Clone, Deserialize)]
pub struct FioConfig {
    pub api_token: String,
}

/// OCR service settings.
#[derive(Debug, Clone, Deserialize)]
pub struct OcrConfig {
    pub provider: Option<String>,
    pub api_key: Option<String>,
    pub model: Option<String>,
    pub base_url: Option<String>,
}

impl S3Config {
    /// Returns true when both endpoint and bucket are non-empty.
    pub fn is_configured(&self) -> bool {
        !self.endpoint.is_empty() && !self.bucket.is_empty()
    }
}

// --- Defaults ---

fn default_true() -> bool {
    true
}

fn default_smtp_port() -> u16 {
    587
}

impl Default for Config {
    fn default() -> Self {
        Self {
            data_dir: None,
            database: DatabaseConfig::default(),
            log: LogConfig::default(),
            server: ServerConfig::default(),
            backup: BackupConfig::default(),
            smtp: None,
            fio: None,
            ocr: None,
        }
    }
}

impl Default for DatabaseConfig {
    fn default() -> Self {
        Self {
            path: None,
            journal_mode: "wal".to_string(),
        }
    }
}

impl Default for LogConfig {
    fn default() -> Self {
        Self {
            path: None,
            level: "info".to_string(),
        }
    }
}

impl Default for ServerConfig {
    fn default() -> Self {
        Self {
            port: 8080,
            dev: false,
        }
    }
}

impl Default for BackupConfig {
    fn default() -> Self {
        Self {
            destination: None,
            schedule: None,
            retention_count: 0,
            s3: None,
        }
    }
}

// --- Config resolution ---

/// Return the default data directory: `~/.zfaktury`.
fn default_data_dir() -> Result<PathBuf, ConfigError> {
    let home = dirs::home_dir().ok_or(ConfigError::HomeDir)?;
    Ok(home.join(".zfaktury"))
}

/// Replace a leading `~` with the user's home directory.
pub fn expand_home(path: &Path) -> PathBuf {
    let s = path.to_string_lossy();
    if !s.starts_with('~') {
        return path.to_path_buf();
    }
    match dirs::home_dir() {
        Some(home) => home.join(&s[2..].trim_start_matches('/')),
        None => path.to_path_buf(),
    }
}

impl Config {
    /// Load configuration using the default resolution logic:
    /// 1. Determine data_dir (env var `ZFAKTURY_DATA_DIR` or `~/.zfaktury`)
    /// 2. Config file: `{data_dir}/config.toml`
    /// 3. If config file doesn't exist, return default config
    /// 4. Parse TOML and apply defaults
    pub fn load() -> Result<Self, ConfigError> {
        let data_dir = match env::var("ZFAKTURY_DATA_DIR") {
            Ok(dir) if !dir.is_empty() => PathBuf::from(dir),
            _ => default_data_dir()?,
        };

        let config_path = data_dir.join("config.toml");

        let mut cfg = if config_path.exists() {
            Self::load_from(&config_path)?
        } else {
            Self::default()
        };

        // Env var always overrides data_dir from file.
        if let Ok(dir) = env::var("ZFAKTURY_DATA_DIR") {
            if !dir.is_empty() {
                cfg.data_dir = Some(PathBuf::from(dir));
            }
        }

        // If data_dir was not set at all, use the default.
        if cfg.data_dir.is_none() {
            cfg.data_dir = Some(data_dir);
        }

        Ok(cfg)
    }

    /// Load configuration from a specific TOML file.
    pub fn load_from(path: &Path) -> Result<Self, ConfigError> {
        let contents = std::fs::read_to_string(path).map_err(|e| ConfigError::ReadFile {
            path: path.to_path_buf(),
            source: e,
        })?;

        let mut cfg: Config = toml::from_str(&contents).map_err(|e| ConfigError::ParseFile {
            path: path.to_path_buf(),
            source: e,
        })?;

        // Expand ~ in paths.
        if let Some(ref dir) = cfg.data_dir {
            cfg.data_dir = Some(expand_home(dir));
        }
        if let Some(ref p) = cfg.database.path {
            cfg.database.path = Some(expand_home(p));
        }
        if let Some(ref p) = cfg.log.path {
            cfg.log.path = Some(expand_home(p));
        }
        if let Some(ref p) = cfg.backup.destination {
            cfg.backup.destination = Some(expand_home(p));
        }

        Ok(cfg)
    }

    /// Resolved data directory. Falls back to `~/.zfaktury` if not set.
    pub fn data_dir(&self) -> PathBuf {
        self.data_dir
            .clone()
            .unwrap_or_else(|| default_data_dir().unwrap_or_else(|_| PathBuf::from(".zfaktury")))
    }

    /// Resolved database path. Falls back to `{data_dir}/zfaktury.db`.
    pub fn database_path(&self) -> PathBuf {
        self.database
            .path
            .clone()
            .unwrap_or_else(|| self.data_dir().join("zfaktury.db"))
    }

    /// Resolved backup directory. Falls back to `{data_dir}/backups`.
    pub fn backup_dir(&self) -> PathBuf {
        self.backup
            .destination
            .clone()
            .unwrap_or_else(|| self.data_dir().join("backups"))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;

    #[test]
    fn test_default_config() {
        let cfg = Config::default();

        assert!(cfg.data_dir.is_none());
        assert!(cfg.database.path.is_none());
        assert_eq!(cfg.database.journal_mode, "wal");
        assert!(cfg.log.path.is_none());
        assert_eq!(cfg.log.level, "info");
        assert_eq!(cfg.server.port, 8080);
        assert!(!cfg.server.dev);
        assert!(cfg.backup.destination.is_none());
        assert!(cfg.backup.schedule.is_none());
        assert_eq!(cfg.backup.retention_count, 0);
        assert!(cfg.backup.s3.is_none());
        assert!(cfg.smtp.is_none());
        assert!(cfg.fio.is_none());
        assert!(cfg.ocr.is_none());
    }

    #[test]
    fn test_parse_full_toml() {
        let toml_content = r#"
data_dir = "/custom/data"

[database]
path = "/custom/db.sqlite"
journal_mode = "delete"

[log]
path = "/var/log/zfaktury.log"
level = "debug"

[server]
port = 9090
dev = true

[backup]
destination = "/backups/zfaktury"
schedule = "0 2 * * *"
retention_count = 5

[backup.s3]
endpoint = "s3.amazonaws.com"
region = "eu-central-1"
bucket = "my-bucket"
access_key = "AKIAIOSFODNN7EXAMPLE"
secret_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
use_ssl = true

[smtp]
host = "smtp.example.com"
port = 465
username = "user@example.com"
password = "secret"
from = "noreply@example.com"

[fio]
api_token = "test-token-123"

[ocr]
provider = "openai"
api_key = "sk-test-key"
model = "gpt-4o"
base_url = "https://api.openai.com"
"#;

        let cfg: Config = toml::from_str(toml_content).unwrap();

        assert_eq!(cfg.data_dir.unwrap(), PathBuf::from("/custom/data"));
        assert_eq!(
            cfg.database.path.unwrap(),
            PathBuf::from("/custom/db.sqlite")
        );
        assert_eq!(cfg.database.journal_mode, "delete");
        assert_eq!(
            cfg.log.path.unwrap(),
            PathBuf::from("/var/log/zfaktury.log")
        );
        assert_eq!(cfg.log.level, "debug");
        assert_eq!(cfg.server.port, 9090);
        assert!(cfg.server.dev);
        assert_eq!(
            cfg.backup.destination.unwrap(),
            PathBuf::from("/backups/zfaktury")
        );
        assert_eq!(cfg.backup.schedule.unwrap(), "0 2 * * *");
        assert_eq!(cfg.backup.retention_count, 5);

        let s3 = cfg.backup.s3.unwrap();
        assert_eq!(s3.endpoint, "s3.amazonaws.com");
        assert_eq!(s3.region.as_deref(), Some("eu-central-1"));
        assert_eq!(s3.bucket, "my-bucket");
        assert_eq!(s3.access_key, "AKIAIOSFODNN7EXAMPLE");
        assert!(s3.use_ssl);
        assert!(s3.is_configured());

        let smtp = cfg.smtp.unwrap();
        assert_eq!(smtp.host, "smtp.example.com");
        assert_eq!(smtp.port, 465);
        assert_eq!(smtp.username, "user@example.com");
        assert_eq!(smtp.from, "noreply@example.com");

        let fio = cfg.fio.unwrap();
        assert_eq!(fio.api_token, "test-token-123");

        let ocr = cfg.ocr.unwrap();
        assert_eq!(ocr.provider.unwrap(), "openai");
        assert_eq!(ocr.api_key.unwrap(), "sk-test-key");
        assert_eq!(ocr.model.unwrap(), "gpt-4o");
        assert_eq!(ocr.base_url.unwrap(), "https://api.openai.com");
    }

    #[test]
    fn test_parse_minimal_toml() {
        // Empty TOML should produce default config.
        let cfg: Config = toml::from_str("").unwrap();

        assert!(cfg.data_dir.is_none());
        assert!(cfg.database.path.is_none());
        assert_eq!(cfg.log.level, "info");
        assert_eq!(cfg.server.port, 8080);
        assert!(cfg.smtp.is_none());
        assert!(cfg.fio.is_none());
        assert!(cfg.ocr.is_none());
    }

    #[test]
    fn test_parse_partial_toml() {
        let toml_content = r#"
[log]
level = "warn"

[fio]
api_token = "my-token"
"#;

        let cfg: Config = toml::from_str(toml_content).unwrap();

        assert!(cfg.data_dir.is_none());
        assert_eq!(cfg.log.level, "warn");
        assert_eq!(cfg.server.port, 8080); // default
        let fio = cfg.fio.unwrap();
        assert_eq!(fio.api_token, "my-token");
        assert!(cfg.smtp.is_none());
    }

    #[test]
    fn test_env_var_override() {
        let tmp = tempfile::tempdir().unwrap();
        let config_path = tmp.path().join("config.toml");
        std::fs::write(&config_path, "").unwrap();

        // Set env var before loading.
        let env_dir = tmp.path().join("env-data");
        // SAFETY: This test is single-threaded and we restore the var at the end.
        unsafe {
            env::set_var("ZFAKTURY_DATA_DIR", &env_dir);
        }

        let cfg = Config::load_from(&config_path).unwrap();
        // load_from does not read env -- that's done by load().
        // So data_dir from file should be None.
        assert!(cfg.data_dir.is_none());

        // But data_dir() helper falls back to default.
        // Now test load() which does read the env var.
        // Create the config.toml in the env dir.
        std::fs::create_dir_all(&env_dir).unwrap();
        let env_config = env_dir.join("config.toml");
        std::fs::write(&env_config, "").unwrap();

        let cfg = Config::load().unwrap();
        assert_eq!(cfg.data_dir(), env_dir);

        // Clean up env.
        // SAFETY: Same as above -- single-threaded test cleanup.
        unsafe {
            env::remove_var("ZFAKTURY_DATA_DIR");
        }
    }

    #[test]
    fn test_database_path_resolution() {
        let cfg = Config {
            data_dir: Some(PathBuf::from("/data")),
            ..Config::default()
        };
        assert_eq!(cfg.database_path(), PathBuf::from("/data/zfaktury.db"));

        let cfg = Config {
            data_dir: Some(PathBuf::from("/data")),
            database: DatabaseConfig {
                path: Some(PathBuf::from("/custom/my.db")),
                ..DatabaseConfig::default()
            },
            ..Config::default()
        };
        assert_eq!(cfg.database_path(), PathBuf::from("/custom/my.db"));
    }

    #[test]
    fn test_backup_dir_resolution() {
        let cfg = Config {
            data_dir: Some(PathBuf::from("/data")),
            ..Config::default()
        };
        assert_eq!(cfg.backup_dir(), PathBuf::from("/data/backups"));

        let cfg = Config {
            data_dir: Some(PathBuf::from("/data")),
            backup: BackupConfig {
                destination: Some(PathBuf::from("/custom/backups")),
                ..BackupConfig::default()
            },
            ..Config::default()
        };
        assert_eq!(cfg.backup_dir(), PathBuf::from("/custom/backups"));
    }

    #[test]
    fn test_load_from_file() {
        let tmp = tempfile::tempdir().unwrap();
        let config_path = tmp.path().join("config.toml");

        let mut f = std::fs::File::create(&config_path).unwrap();
        writeln!(
            f,
            r#"
data_dir = "{}"

[database]
path = "{}"

[log]
level = "debug"
"#,
            tmp.path().join("mydata").display(),
            tmp.path().join("mydata/test.db").display(),
        )
        .unwrap();

        let cfg = Config::load_from(&config_path).unwrap();
        assert_eq!(cfg.data_dir(), tmp.path().join("mydata"));
        assert_eq!(cfg.database_path(), tmp.path().join("mydata/test.db"));
        assert_eq!(cfg.log.level, "debug");
    }

    #[test]
    fn test_load_nonexistent_file() {
        let result = Config::load_from(Path::new("/nonexistent/config.toml"));
        assert!(result.is_err());
        let err = result.unwrap_err();
        assert!(matches!(err, ConfigError::ReadFile { .. }));
    }

    #[test]
    fn test_load_invalid_toml() {
        let tmp = tempfile::tempdir().unwrap();
        let config_path = tmp.path().join("config.toml");
        std::fs::write(&config_path, "this is not valid toml {{{{").unwrap();

        let result = Config::load_from(&config_path);
        assert!(result.is_err());
        let err = result.unwrap_err();
        assert!(matches!(err, ConfigError::ParseFile { .. }));
    }

    #[test]
    fn test_expand_home() {
        let absolute = expand_home(Path::new("/absolute/path"));
        assert_eq!(absolute, PathBuf::from("/absolute/path"));

        let relative = expand_home(Path::new("relative/path"));
        assert_eq!(relative, PathBuf::from("relative/path"));

        // ~ expansion depends on the environment, but should not panic.
        let home_path = expand_home(Path::new("~/some/path"));
        // It should end with "some/path".
        assert!(
            home_path.ends_with("some/path"),
            "expanded path should end with some/path, got: {:?}",
            home_path
        );
        // It should not start with ~.
        assert!(
            !home_path.to_string_lossy().starts_with('~'),
            "expanded path should not start with ~, got: {:?}",
            home_path
        );
    }

    #[test]
    fn test_s3_is_configured() {
        let s3 = S3Config {
            endpoint: "s3.amazonaws.com".to_string(),
            region: None,
            bucket: "my-bucket".to_string(),
            access_key: "key".to_string(),
            secret_key: "secret".to_string(),
            use_ssl: true,
        };
        assert!(s3.is_configured());

        let s3_no_endpoint = S3Config {
            endpoint: String::new(),
            region: None,
            bucket: "my-bucket".to_string(),
            access_key: "key".to_string(),
            secret_key: "secret".to_string(),
            use_ssl: true,
        };
        assert!(!s3_no_endpoint.is_configured());

        let s3_no_bucket = S3Config {
            endpoint: "s3.amazonaws.com".to_string(),
            region: None,
            bucket: String::new(),
            access_key: "key".to_string(),
            secret_key: "secret".to_string(),
            use_ssl: true,
        };
        assert!(!s3_no_bucket.is_configured());
    }

    #[test]
    fn test_smtp_default_port() {
        let toml_content = r#"
[smtp]
host = "smtp.example.com"
username = "user"
password = "pass"
from = "noreply@example.com"
"#;
        let cfg: Config = toml::from_str(toml_content).unwrap();
        let smtp = cfg.smtp.unwrap();
        assert_eq!(smtp.port, 587);
    }

    #[test]
    fn test_s3_default_use_ssl() {
        let toml_content = r#"
[backup.s3]
endpoint = "minio.local:9000"
bucket = "test"
access_key = "key"
secret_key = "secret"
"#;
        let cfg: Config = toml::from_str(toml_content).unwrap();
        let s3 = cfg.backup.s3.unwrap();
        assert!(s3.use_ssl);
    }
}
