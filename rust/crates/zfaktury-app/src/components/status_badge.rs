use gpui::*;
use zfaktury_domain::InvoiceStatus;

use crate::theme::ZfColors;

/// Returns (text_color, bg_color, label) for an invoice status badge.
pub fn status_badge_style(status: &InvoiceStatus) -> (u32, u32, &'static str) {
    match status {
        InvoiceStatus::Draft => (ZfColors::STATUS_GRAY, ZfColors::STATUS_GRAY_BG, "Koncept"),
        InvoiceStatus::Sent => (ZfColors::STATUS_BLUE, ZfColors::STATUS_BLUE_BG, "Odeslaná"),
        InvoiceStatus::Paid => (
            ZfColors::STATUS_GREEN,
            ZfColors::STATUS_GREEN_BG,
            "Uhrazená",
        ),
        InvoiceStatus::Overdue => (
            ZfColors::STATUS_RED,
            ZfColors::STATUS_RED_BG,
            "Po splatnosti",
        ),
        InvoiceStatus::Cancelled => (ZfColors::TEXT_MUTED, ZfColors::SURFACE_HOVER, "Stornovaná"),
    }
}

/// Render an invoice status badge element.
pub fn render_status_badge(status: &InvoiceStatus) -> Div {
    let (text_color, bg_color, label) = status_badge_style(status);

    div()
        .px_2()
        .py(px(2.0))
        .rounded_sm()
        .bg(rgb(bg_color))
        .text_xs()
        .text_color(rgb(text_color))
        .child(label.to_string())
}
