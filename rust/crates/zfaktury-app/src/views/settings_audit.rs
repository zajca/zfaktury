use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::AuditService;
use zfaktury_domain::{AuditLogEntry, AuditLogFilter};

use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;

/// Audit log settings view.
pub struct SettingsAuditView {
    service: Arc<AuditService>,
    loading: bool,
    error: Option<String>,
    entries: Vec<AuditLogEntry>,
    total: i64,
}

impl SettingsAuditView {
    pub fn new(service: Arc<AuditService>, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            loading: true,
            error: None,
            entries: Vec::new(),
            total: 0,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    service.list(&AuditLogFilter {
                        entity_type: String::new(),
                        entity_id: None,
                        action: String::new(),
                        from: None,
                        to: None,
                        limit: 50,
                        offset: 0,
                    })
                })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok((entries, total)) => {
                        this.entries = entries;
                        this.total = total;
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani audit logu: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn format_datetime(dt: chrono::NaiveDateTime) -> String {
        dt.format("%-d. %-m. %Y %H:%M").to_string()
    }
}

impl EventEmitter<NavigateEvent> for SettingsAuditView {}

impl Render for SettingsAuditView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("settings-audit-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_4()
            .overflow_y_scroll();

        content = content.child(
            div()
                .flex()
                .items_center()
                .gap_3()
                .child(
                    div()
                        .text_xl()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("Audit log"),
                )
                .child(
                    div()
                        .text_sm()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child(format!("({} celkem)", self.total)),
                ),
        );

        if self.loading {
            return content.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Nacitani..."),
            );
        }

        if let Some(ref error) = self.error {
            return content.child(
                div()
                    .px_4()
                    .py_3()
                    .bg(rgb(ZfColors::STATUS_RED_BG))
                    .rounded_md()
                    .text_sm()
                    .text_color(rgb(ZfColors::STATUS_RED))
                    .child(error.clone()),
            );
        }

        // Table
        let mut table = div()
            .flex()
            .flex_col()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .overflow_hidden();

        // Column headers
        table = table.child(
            div()
                .flex()
                .px_4()
                .py_3()
                .text_xs()
                .font_weight(FontWeight::MEDIUM)
                .text_color(rgb(ZfColors::TEXT_MUTED))
                .border_b_1()
                .border_color(rgb(ZfColors::BORDER))
                .child(div().w(px(150.0)).child("Datum"))
                .child(div().w(px(112.0)).child("Entita"))
                .child(div().w_20().child("ID"))
                .child(div().w(px(112.0)).child("Akce"))
                .child(div().flex_1().child("Detail")),
        );

        if self.entries.is_empty() {
            table = table.child(
                div()
                    .px_4()
                    .py_8()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadne zaznamy."),
            );
        } else {
            for entry in &self.entries {
                let detail = if !entry.new_values.is_empty() {
                    entry.new_values.clone()
                } else if !entry.old_values.is_empty() {
                    entry.old_values.clone()
                } else {
                    "-".to_string()
                };

                // Truncate detail to reasonable length
                let detail_display = if detail.len() > 80 {
                    format!("{}...", &detail[..80])
                } else {
                    detail
                };

                table = table.child(
                    div()
                        .flex()
                        .items_center()
                        .px_4()
                        .py_2()
                        .text_sm()
                        .border_t_1()
                        .border_color(rgb(ZfColors::BORDER_SUBTLE))
                        .child(
                            div()
                                .w(px(150.0))
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(Self::format_datetime(entry.created_at)),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(entry.entity_type.clone()),
                        )
                        .child(
                            div()
                                .w_20()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(entry.entity_id.to_string()),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(entry.action.clone()),
                        )
                        .child(
                            div()
                                .flex_1()
                                .text_xs()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(detail_display),
                        ),
                );
            }
        }

        content.child(table)
    }
}
