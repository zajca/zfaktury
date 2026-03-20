mod app;
mod components;
mod navigation;
mod root;
mod sidebar;
mod theme;
mod util;
mod views;

use std::sync::Arc;
use std::time::Duration;

use clap::Parser;
use gpui::*;

use crate::app::AppServices;
use crate::navigation::Route;
use crate::root::RootView;

#[derive(Parser)]
#[command(name = "zfaktury", version)]
struct Cli {
    /// Initial route for testing (e.g., "/invoices", "/vat")
    #[arg(long)]
    route: Option<String>,

    /// Exit after N seconds (for headless screenshot testing)
    #[arg(long)]
    exit_after: Option<u64>,

    /// Custom database path (overrides config)
    #[arg(long)]
    db: Option<String>,
}

fn main() {
    let cli = Cli::parse();

    // Determine database path.
    let db_path = if let Some(ref db) = cli.db {
        std::path::PathBuf::from(db)
    } else {
        match zfaktury_config::Config::load() {
            Ok(cfg) => cfg.database_path(),
            Err(e) => {
                eprintln!("Error loading config: {e}");
                std::process::exit(1);
            }
        }
    };

    // Ensure parent directory exists.
    if let Some(parent) = db_path.parent() {
        if !parent.exists() {
            if let Err(e) = std::fs::create_dir_all(parent) {
                eprintln!("Error creating data directory: {e}");
                std::process::exit(1);
            }
        }
    }

    // Initialize services.
    let services = match AppServices::new(&db_path) {
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
