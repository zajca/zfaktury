use gpui::*;

use crate::components::button::{ButtonVariant, render_button};
use crate::theme::ZfColors;

/// Event emitted when the user confirms or cancels the dialog.
pub enum ConfirmDialogResult {
    Confirmed,
    Cancelled,
}

/// Modal confirmation dialog overlay.
pub struct ConfirmDialog {
    title: SharedString,
    message: SharedString,
    confirm_label: SharedString,
}

impl EventEmitter<ConfirmDialogResult> for ConfirmDialog {}

impl ConfirmDialog {
    pub fn new(
        title: impl Into<SharedString>,
        message: impl Into<SharedString>,
        confirm_label: impl Into<SharedString>,
    ) -> Self {
        Self {
            title: title.into(),
            message: message.into(),
            confirm_label: confirm_label.into(),
        }
    }
}

impl Render for ConfirmDialog {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let confirm_label = self.confirm_label.clone();

        // Full-screen overlay
        div()
            .id("confirm-dialog-overlay")
            .absolute()
            .inset_0()
            .flex()
            .items_center()
            .justify_center()
            .bg(rgba(0x000000aa))
            // Backdrop click = cancel
            .on_click(cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                cx.emit(ConfirmDialogResult::Cancelled);
            }))
            .child(
                // Dialog card
                div()
                    .id("confirm-dialog-card")
                    .w(px(400.0))
                    .bg(rgb(ZfColors::SURFACE))
                    .rounded_lg()
                    .border_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .shadow_lg()
                    .p_6()
                    .flex()
                    .flex_col()
                    .gap_4()
                    // Prevent click-through to overlay
                    .on_click(|_event: &ClickEvent, _window, _cx| {})
                    // Title
                    .child(
                        div()
                            .text_base()
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child(self.title.clone()),
                    )
                    // Message
                    .child(
                        div()
                            .text_sm()
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .child(self.message.clone()),
                    )
                    // Buttons
                    .child(
                        div()
                            .flex()
                            .justify_end()
                            .gap_3()
                            .child(render_button(
                                "confirm-cancel",
                                "Zrusit",
                                ButtonVariant::Secondary,
                                false,
                                false,
                                cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                                    cx.emit(ConfirmDialogResult::Cancelled);
                                }),
                            ))
                            .child(render_button(
                                "confirm-ok",
                                &confirm_label,
                                ButtonVariant::Danger,
                                false,
                                false,
                                cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                                    cx.emit(ConfirmDialogResult::Confirmed);
                                }),
                            )),
                    ),
            )
    }
}
