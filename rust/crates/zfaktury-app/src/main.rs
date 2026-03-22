mod app;
mod components;
mod navigation;
mod root;
mod sidebar;
mod theme;
mod util;
mod views;

use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::time::Duration;

use clap::{Parser, Subcommand};
use gpui::*;

use crate::app::AppServices;
use crate::navigation::Route;
use crate::root::RootView;

#[derive(Parser)]
#[command(name = "zfaktury", version)]
struct Cli {
    #[command(subcommand)]
    command: Option<Commands>,

    /// Initial route for testing (e.g., "/invoices", "/vat")
    #[arg(long)]
    route: Option<String>,

    /// Exit after N seconds (for headless screenshot testing)
    #[arg(long)]
    exit_after: Option<u64>,

    /// Custom database path (overrides config)
    #[arg(long)]
    db: Option<PathBuf>,
}

#[derive(Subcommand)]
enum Commands {
    /// Run database migrations
    Migrate {
        /// Show migration status only (do not run pending migrations)
        #[arg(long)]
        status: bool,
    },
    /// Show version information
    Version,
}

fn main() {
    let cli = Cli::parse();

    match cli.command {
        Some(Commands::Migrate { status }) => cmd_migrate(&cli, status),
        Some(Commands::Version) => cmd_version(),
        None => cmd_gui(cli),
    }
}

/// Run or show database migrations.
fn cmd_migrate(cli: &Cli, status_only: bool) {
    let db_path = resolve_db_path(cli);
    ensure_parent_dir(&db_path);

    let conn = match zfaktury_db::connection::open_connection(&db_path) {
        Ok(c) => c,
        Err(e) => {
            eprintln!("Error opening database: {e}");
            std::process::exit(1);
        }
    };

    if status_only {
        // Show current status without running migrations.
        match zfaktury_db::migrate::migration_status(&conn) {
            Ok(entries) => {
                let applied = entries.iter().filter(|(_, _, done)| *done).count();
                let total = entries.len();
                println!("Database: {}", db_path.display());
                println!("Migrations: {applied}/{total} applied\n");
                for (version, name, done) in &entries {
                    let marker = if *done { "[x]" } else { "[ ]" };
                    println!("  {marker} V{version}: {name}");
                }
                if applied < total {
                    println!("\nRun `zfaktury migrate` to apply pending migrations.");
                } else {
                    println!("\nAll migrations are up to date.");
                }
            }
            Err(e) => {
                eprintln!("Error reading migration status: {e}");
                std::process::exit(1);
            }
        }
    } else {
        // Run pending migrations.
        println!("Database: {}", db_path.display());
        println!("Running migrations...");
        match zfaktury_db::migrate::run_migrations(&conn) {
            Ok(()) => {
                let version = zfaktury_db::migrate::current_version(&conn).unwrap_or(0);
                let total = zfaktury_db::migrate::total_migrations();
                println!(
                    "Migrations complete. Current version: V{version} ({total} total migrations)."
                );
            }
            Err(e) => {
                eprintln!("Migration failed: {e}");
                std::process::exit(1);
            }
        }
    }
}

/// Print version and build information.
fn cmd_version() {
    let version = env!("CARGO_PKG_VERSION");
    let rust_version = option_env!("CARGO_PKG_RUST_VERSION").unwrap_or("unknown");
    println!("zfaktury {version}");
    println!("  edition: 2024");
    println!("  rust-version: {rust_version}");
    println!("  sqlite: bundled (rusqlite)");
    println!("  gui: GPUI (Zed)");
}

/// Launch the GPUI desktop application (default behavior).
fn cmd_gui(cli: Cli) {
    // Load config once.
    let config = match zfaktury_config::Config::load() {
        Ok(cfg) => cfg,
        Err(e) => {
            eprintln!("Error loading config: {e}");
            std::process::exit(1);
        }
    };

    // Resolve paths (CLI overrides config).
    let db_path = cli.db.clone().unwrap_or_else(|| config.database_path());
    let data_dir = if let Some(ref db) = cli.db {
        db.parent()
            .map(|p| p.to_path_buf())
            .unwrap_or_else(|| PathBuf::from("."))
    } else {
        config.data_dir()
    };

    ensure_parent_dir(&db_path);
    ensure_parent_dir_of(&data_dir);

    // Initialize services (now receives config for OCR, SMTP, etc.).
    let services = match AppServices::new(&db_path, &data_dir, &config) {
        Ok(s) => Arc::new(s),
        Err(e) => {
            eprintln!("Error initializing application: {e}");
            std::process::exit(1);
        }
    };

    // Parse initial route.
    let initial_route = cli
        .route
        .as_deref()
        .and_then(Route::from_path)
        .unwrap_or(Route::Dashboard);

    gpui_platform::application().run(move |cx: &mut App| {
        let bounds = Bounds::centered(None, size(px(1200.0), px(800.0)), cx);
        let services = services.clone();
        let initial_route = initial_route.clone();

        cx.open_window(
            WindowOptions {
                window_bounds: Some(WindowBounds::Windowed(bounds)),
                titlebar: Some(TitlebarOptions {
                    title: Some("ZFaktury".into()),
                    ..Default::default()
                }),
                ..Default::default()
            },
            |_window, cx| cx.new(|cx| RootView::new(services, initial_route, cx)),
        )
        .unwrap();

        if let Some(seconds) = cli.exit_after {
            cx.spawn(async move |cx| {
                cx.background_executor()
                    .timer(Duration::from_secs(seconds))
                    .await;
                cx.update(|cx| cx.quit());
            })
            .detach();
        }
    });
}

/// Resolve the database path from CLI args or config.
fn resolve_db_path(cli: &Cli) -> PathBuf {
    if let Some(ref db) = cli.db {
        db.clone()
    } else {
        match zfaktury_config::Config::load() {
            Ok(cfg) => cfg.database_path(),
            Err(e) => {
                eprintln!("Error loading config: {e}");
                std::process::exit(1);
            }
        }
    }
}

/// Ensure the parent directory of the given path exists.
fn ensure_parent_dir(path: &Path) {
    if let Some(parent) = path.parent()
        && !parent.exists()
        && let Err(e) = std::fs::create_dir_all(parent)
    {
        eprintln!("Error creating data directory: {e}");
        std::process::exit(1);
    }
}

/// Ensure the given directory exists.
fn ensure_parent_dir_of(path: &Path) {
    if !path.exists()
        && let Err(e) = std::fs::create_dir_all(path)
    {
        eprintln!("Error creating data directory: {e}");
        std::process::exit(1);
    }
}
