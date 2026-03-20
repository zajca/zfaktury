use gpui::*;

use crate::theme::ZfColors;

/// Generic stub view that displays a route title and "Coming soon" text.
/// Used for routes that are not yet fully implemented.
pub struct StubView {
    title: String,
}

impl StubView {
    pub fn new(title: impl Into<String>) -> Self {
        Self {
            title: title.into(),
        }
    }
}

impl Render for StubView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        div()
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_3()
            .child(
                div()
                    .text_xl()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(self.title.clone()),
            )
            .child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Tato stranka bude brzy dostupna."),
            )
    }
}
