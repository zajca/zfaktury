use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::SequenceService;
use zfaktury_domain::InvoiceSequence;

use crate::theme::ZfColors;

/// Settings sequences view showing invoice number sequences.
pub struct SettingsSequencesView {
    service: Arc<SequenceService>,
    loading: bool,
    error: Option<String>,
    sequences: Vec<InvoiceSequence>,
}

impl SettingsSequencesView {
    pub fn new(service: Arc<SequenceService>, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            loading: true,
            error: None,
            sequences: Vec::new(),
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.list() })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(sequences) => this.sequences = sequences,
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani ciselnych rad: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }
}

impl Render for SettingsSequencesView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("settings-sequences-scroll")
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
                        .child("Ciselne rady"),
                )
                .child(
                    div()
                        .text_sm()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child(format!("({} celkem)", self.sequences.len())),
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
                .child(div().w(px(112.0)).child("Prefix"))
                .child(div().w_20().child("Rok"))
                .child(div().w(px(112.0)).child("Dalsi cislo"))
                .child(div().flex_1().child("Format"))
                .child(div().w(px(150.0)).child("Nahledy")),
        );

        if self.sequences.is_empty() {
            table = table.child(
                div()
                    .px_4()
                    .py_8()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadne ciselne rady."),
            );
        } else {
            for seq in &self.sequences {
                let preview = SequenceService::format_preview(seq);

                table = table.child(
                    div()
                        .flex()
                        .items_center()
                        .px_4()
                        .py_2()
                        .text_sm()
                        .border_t_1()
                        .border_color(rgb(ZfColors::BORDER_SUBTLE))
                        .hover(|s| s.bg(rgb(ZfColors::SURFACE_HOVER)))
                        .child(
                            div()
                                .w(px(112.0))
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(seq.prefix.clone()),
                        )
                        .child(
                            div()
                                .w_20()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(seq.year.to_string()),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(seq.next_number.to_string()),
                        )
                        .child(
                            div()
                                .flex_1()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(seq.format_pattern.clone()),
                        )
                        .child(
                            div()
                                .w(px(150.0))
                                .text_color(rgb(ZfColors::ACCENT))
                                .child(preview),
                        ),
                );
            }
        }

        content.child(table)
    }
}
