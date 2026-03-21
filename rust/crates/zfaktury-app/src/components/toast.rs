use gpui::*;

use crate::theme::ZfColors;

/// Toast notification severity level.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ToastLevel {
    Success,
    Error,
    Info,
}

/// A single toast notification.
#[derive(Debug, Clone)]
pub struct ToastMessage {
    pub message: String,
    pub level: ToastLevel,
}

/// Event to dismiss a toast.
pub struct ToastDismissed;

/// Toast notification view, rendered at the top-right of the screen.
pub struct ToastView {
    toast: Option<ToastMessage>,
}

impl EventEmitter<ToastDismissed> for ToastView {}

impl ToastView {
    pub fn new() -> Self {
        Self { toast: None }
    }

    pub fn show(&mut self, message: impl Into<String>, level: ToastLevel, cx: &mut Context<Self>) {
        self.toast = Some(ToastMessage {
            message: message.into(),
            level,
        });
        cx.notify();

        // Auto-dismiss after 4 seconds.
        cx.spawn(async move |this, cx| {
            cx.background_executor()
                .timer(std::time::Duration::from_secs(4))
                .await;
            this.update(cx, |this, cx| {
                this.toast = None;
                cx.emit(ToastDismissed);
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    pub fn dismiss(&mut self, cx: &mut Context<Self>) {
        self.toast = None;
        cx.notify();
    }

    pub fn has_toast(&self) -> bool {
        self.toast.is_some()
    }
}

impl Render for ToastView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let Some(ref toast) = self.toast else {
            return div().into_any_element();
        };

        let (bg, text_color, border) = match toast.level {
            ToastLevel::Success => (
                ZfColors::STATUS_GREEN_BG,
                ZfColors::STATUS_GREEN,
                ZfColors::STATUS_GREEN,
            ),
            ToastLevel::Error => (
                ZfColors::STATUS_RED_BG,
                ZfColors::STATUS_RED,
                ZfColors::STATUS_RED,
            ),
            ToastLevel::Info => (
                ZfColors::STATUS_BLUE_BG,
                ZfColors::STATUS_BLUE,
                ZfColors::STATUS_BLUE,
            ),
        };

        div()
            .id("toast-container")
            .absolute()
            .top_4()
            .right_4()
            .w(px(320.0))
            .child(
                div()
                    .id("toast-message")
                    .px_4()
                    .py_3()
                    .bg(rgb(bg))
                    .border_1()
                    .border_color(rgb(border))
                    .rounded_md()
                    .shadow_md()
                    .flex()
                    .items_center()
                    .justify_between()
                    .child(
                        div()
                            .text_sm()
                            .text_color(rgb(text_color))
                            .child(toast.message.clone()),
                    )
                    .child(
                        div()
                            .id("toast-dismiss")
                            .text_xs()
                            .text_color(rgb(text_color))
                            .cursor_pointer()
                            .on_click(cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                this.dismiss(cx);
                            }))
                            .child("✕"),
                    ),
            )
            .into_any_element()
    }
}
