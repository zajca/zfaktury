use std::time::Duration;

use clap::Parser;
use gpui::*;

#[derive(Parser)]
#[command(name = "zfaktury", version)]
struct Cli {
    /// Initial route for testing (e.g., "/invoices", "/vat")
    #[arg(long)]
    route: Option<String>,

    /// Exit after N seconds (for headless screenshot testing)
    #[arg(long)]
    exit_after: Option<u64>,
}

struct RootView {
    route: Option<String>,
}

impl Render for RootView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let route_text = self.route.as_deref().unwrap_or("Dashboard");

        div()
            .flex()
            .flex_col()
            .size_full()
            .bg(rgb(0x1e1d21))
            .justify_center()
            .items_center()
            .gap_4()
            .child(
                div()
                    .text_xl()
                    .text_color(rgb(0xececef))
                    .child("ZFaktury"),
            )
            .child(
                div()
                    .text_color(rgb(0xa0a0a8))
                    .child(format!("Route: {route_text}")),
            )
            .child(
                div()
                    .px_4()
                    .py_2()
                    .bg(rgb(0x5e6ad2))
                    .rounded_md()
                    .text_color(rgb(0xffffff))
                    .cursor_pointer()
                    .child("Test Button"),
            )
    }
}

fn main() {
    let cli = Cli::parse();

    gpui_platform::application().run(move |cx: &mut App| {
        let bounds = Bounds::centered(None, size(px(800.0), px(600.0)), cx);
        let route = cli.route.clone();

        cx.open_window(
            WindowOptions {
                window_bounds: Some(WindowBounds::Windowed(bounds)),
                titlebar: Some(TitlebarOptions {
                    title: Some("ZFaktury".into()),
                    ..Default::default()
                }),
                ..Default::default()
            },
            |_window, cx| cx.new(|_cx| RootView { route }),
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
