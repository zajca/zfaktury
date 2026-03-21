use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::SequenceService;
use zfaktury_domain::InvoiceSequence;

use crate::components::button::{ButtonVariant, render_button};
use crate::components::confirm_dialog::{ConfirmDialog, ConfirmDialogResult};
use crate::components::number_input::NumberInput;
use crate::components::text_input::TextInput;
use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;

/// Row editing state for inline edit/create.
struct EditingRow {
    /// Sequence ID being edited (0 for new).
    id: i64,
    prefix: Entity<TextInput>,
    year: Entity<NumberInput>,
    next_number: Entity<NumberInput>,
    format_pattern: Entity<TextInput>,
}

/// Settings sequences view with inline CRUD.
pub struct SettingsSequencesView {
    service: Arc<SequenceService>,
    loading: bool,
    saving: bool,
    error: Option<String>,
    sequences: Vec<InvoiceSequence>,
    editing: Option<EditingRow>,
    confirm_dialog: Option<Entity<ConfirmDialog>>,
    delete_id: Option<i64>,
}

impl SettingsSequencesView {
    pub fn new(service: Arc<SequenceService>, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            loading: true,
            saving: false,
            error: None,
            sequences: Vec::new(),
            editing: None,
            confirm_dialog: None,
            delete_id: None,
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

    fn start_new(&mut self, cx: &mut Context<Self>) {
        let current_year = chrono::Local::now().format("%Y").to_string();
        let prefix = cx.new(|cx| TextInput::new("seq-new-prefix", "FV", cx));
        let year = cx.new(|cx| {
            NumberInput::new("seq-new-year", &current_year, cx)
                .integer_only()
                .with_value(&current_year)
        });
        let next_number = cx.new(|cx| {
            NumberInput::new("seq-new-next", "1", cx)
                .integer_only()
                .with_value("1")
        });
        let format_pattern =
            cx.new(|cx| TextInput::new("seq-new-format", "{prefix}{year}{number:04}", cx));

        self.editing = Some(EditingRow {
            id: 0,
            prefix,
            year,
            next_number,
            format_pattern,
        });
        self.error = None;
        cx.notify();
    }

    fn start_edit(&mut self, seq: &InvoiceSequence, cx: &mut Context<Self>) {
        let seq_id = seq.id;
        let prefix = cx.new(|cx| {
            let mut t = TextInput::new(
                SharedString::from(format!("seq-edit-prefix-{}", seq_id)),
                "FV",
                cx,
            );
            t.set_value(&seq.prefix, cx);
            t
        });
        let year = cx.new(|cx| {
            NumberInput::new(
                SharedString::from(format!("seq-edit-year-{}", seq_id)),
                "2024",
                cx,
            )
            .integer_only()
            .with_value(seq.year.to_string())
        });
        let next_number = cx.new(|cx| {
            NumberInput::new(
                SharedString::from(format!("seq-edit-next-{}", seq_id)),
                "1",
                cx,
            )
            .integer_only()
            .with_value(seq.next_number.to_string())
        });
        let format_pattern = cx.new(|cx| {
            let mut t = TextInput::new(
                SharedString::from(format!("seq-edit-format-{}", seq_id)),
                "{prefix}{year}{number:04}",
                cx,
            );
            t.set_value(&seq.format_pattern, cx);
            t
        });

        self.editing = Some(EditingRow {
            id: seq_id,
            prefix,
            year,
            next_number,
            format_pattern,
        });
        self.error = None;
        cx.notify();
    }

    fn cancel_edit(&mut self, cx: &mut Context<Self>) {
        self.editing = None;
        self.error = None;
        cx.notify();
    }

    fn save_row(&mut self, cx: &mut Context<Self>) {
        if self.saving {
            return;
        }
        let editing = match &self.editing {
            Some(e) => e,
            None => return,
        };

        let prefix_val = editing.prefix.read(cx).value().to_string();
        let year_val: i32 = editing.year.read(cx).value().parse().unwrap_or(0);
        let next_val: i32 = editing.next_number.read(cx).value().parse().unwrap_or(1);
        let format_val = editing.format_pattern.read(cx).value().to_string();
        let edit_id = editing.id;

        if prefix_val.trim().is_empty() {
            self.error = Some("Prefix je povinny.".into());
            cx.notify();
            return;
        }
        if year_val == 0 {
            self.error = Some("Rok je povinny.".into());
            cx.notify();
            return;
        }

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.service.clone();

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    let mut seq = InvoiceSequence {
                        id: edit_id,
                        prefix: prefix_val,
                        next_number: next_val,
                        year: year_val,
                        format_pattern: format_val,
                    };
                    if edit_id == 0 {
                        service.create(&mut seq)?;
                    } else {
                        service.update(&mut seq)?;
                    }
                    service.list()
                })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(sequences) => {
                        this.sequences = sequences;
                        this.editing = None;
                    }
                    Err(e) => this.error = Some(format!("Chyba pri ukladani: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn request_delete(&mut self, id: i64, cx: &mut Context<Self>) {
        self.delete_id = Some(id);
        let dialog = cx.new(|_cx| {
            ConfirmDialog::new(
                "Smazat ciselnou radu",
                "Opravdu chcete smazat tuto ciselnou radu?",
                "Smazat",
            )
        });
        cx.subscribe(&dialog, Self::on_confirm_result).detach();
        self.confirm_dialog = Some(dialog);
        cx.notify();
    }

    fn on_confirm_result(
        &mut self,
        _dialog: Entity<ConfirmDialog>,
        event: &ConfirmDialogResult,
        cx: &mut Context<Self>,
    ) {
        match event {
            ConfirmDialogResult::Confirmed => {
                if let Some(id) = self.delete_id.take() {
                    self.confirm_dialog = None;
                    self.do_delete(id, cx);
                }
            }
            ConfirmDialogResult::Cancelled => {
                self.delete_id = None;
                self.confirm_dialog = None;
                cx.notify();
            }
        }
    }

    fn do_delete(&mut self, id: i64, cx: &mut Context<Self>) {
        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    service.delete(id)?;
                    service.list()
                })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(sequences) => {
                        this.sequences = sequences;
                        if let Some(ref editing) = this.editing
                            && editing.id == id
                        {
                            this.editing = None;
                        }
                    }
                    Err(e) => this.error = Some(format!("Chyba pri mazani: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn render_editing_row(&self, editing: &EditingRow) -> Div {
        div()
            .flex()
            .items_center()
            .px_4()
            .py_2()
            .gap_2()
            .border_t_1()
            .border_color(rgb(ZfColors::BORDER_SUBTLE))
            .bg(rgb(ZfColors::SURFACE_HOVER))
            .child(div().w(px(112.0)).child(editing.prefix.clone()))
            .child(div().w_20().child(editing.year.clone()))
            .child(div().w(px(112.0)).child(editing.next_number.clone()))
            .child(div().flex_1().child(editing.format_pattern.clone()))
    }
}

impl EventEmitter<NavigateEvent> for SettingsSequencesView {}

impl Render for SettingsSequencesView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("settings-sequences-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_4()
            .overflow_y_scroll();

        let is_new_row = self.editing.as_ref().map(|e| e.id == 0).unwrap_or(false);

        // Title bar
        content = content.child(
            div()
                .flex()
                .items_center()
                .justify_between()
                .child(
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
                )
                .child(render_button(
                    "seq-new-btn",
                    "Nova ciselna rada",
                    ButtonVariant::Primary,
                    is_new_row || self.saving,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.start_new(cx);
                    }),
                )),
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
            content = content.child(
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
                .child(div().w(px(150.0)).child("Nahled"))
                .child(div().w(px(80.0)).text_right().child("Akce")),
        );

        // New row form at top (if adding).
        if let Some(ref editing) = self.editing
            && editing.id == 0
        {
            table = table.child(
                div()
                    .flex()
                    .flex_col()
                    .child(self.render_editing_row(editing))
                    .child(
                        div()
                            .flex()
                            .justify_end()
                            .gap_2()
                            .px_4()
                            .py_2()
                            .bg(rgb(ZfColors::SURFACE_HOVER))
                            .border_t_1()
                            .border_color(rgb(ZfColors::BORDER_SUBTLE))
                            .child(render_button(
                                "seq-new-cancel",
                                "Zrusit",
                                ButtonVariant::Secondary,
                                self.saving,
                                false,
                                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                    this.cancel_edit(cx);
                                }),
                            ))
                            .child(render_button(
                                "seq-new-save",
                                "Ulozit",
                                ButtonVariant::Primary,
                                false,
                                self.saving,
                                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                    this.save_row(cx);
                                }),
                            )),
                    ),
            );
        }

        if self.sequences.is_empty() && !is_new_row {
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
                let seq_id = seq.id;
                let is_editing_this = self
                    .editing
                    .as_ref()
                    .map(|e| e.id == seq_id)
                    .unwrap_or(false);

                if is_editing_this {
                    if let Some(ref editing) = self.editing {
                        table = table.child(
                            div()
                                .flex()
                                .flex_col()
                                .child(self.render_editing_row(editing))
                                .child(
                                    div()
                                        .flex()
                                        .justify_end()
                                        .gap_2()
                                        .px_4()
                                        .py_2()
                                        .bg(rgb(ZfColors::SURFACE_HOVER))
                                        .border_t_1()
                                        .border_color(rgb(ZfColors::BORDER_SUBTLE))
                                        .child(render_button(
                                            SharedString::from(format!(
                                                "seq-edit-cancel-{}",
                                                seq_id
                                            )),
                                            "Zrusit",
                                            ButtonVariant::Secondary,
                                            self.saving,
                                            false,
                                            cx.listener(
                                                |this, _event: &ClickEvent, _window, cx| {
                                                    this.cancel_edit(cx);
                                                },
                                            ),
                                        ))
                                        .child(render_button(
                                            SharedString::from(format!("seq-edit-save-{}", seq_id)),
                                            "Ulozit",
                                            ButtonVariant::Primary,
                                            false,
                                            self.saving,
                                            cx.listener(
                                                |this, _event: &ClickEvent, _window, cx| {
                                                    this.save_row(cx);
                                                },
                                            ),
                                        )),
                                ),
                        );
                    }
                } else {
                    let preview = SequenceService::format_preview(seq);
                    let has_editing = self.editing.is_some();

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
                            )
                            .child(
                                div()
                                    .w(px(80.0))
                                    .flex()
                                    .justify_end()
                                    .gap_1()
                                    .child(render_button(
                                        SharedString::from(format!("seq-edit-{}", seq_id)),
                                        "Upravit",
                                        ButtonVariant::Secondary,
                                        has_editing || self.saving,
                                        false,
                                        cx.listener(
                                            move |this, _event: &ClickEvent, _window, cx| {
                                                if let Some(seq) =
                                                    this.sequences.iter().find(|s| s.id == seq_id)
                                                {
                                                    let seq = seq.clone();
                                                    this.start_edit(&seq, cx);
                                                }
                                            },
                                        ),
                                    ))
                                    .child(render_button(
                                        SharedString::from(format!("seq-del-{}", seq_id)),
                                        "Smazat",
                                        ButtonVariant::Danger,
                                        has_editing || self.saving,
                                        false,
                                        cx.listener(
                                            move |this, _event: &ClickEvent, _window, cx| {
                                                this.request_delete(seq_id, cx);
                                            },
                                        ),
                                    )),
                            ),
                    );
                }
            }
        }

        content = content.child(table);

        // Confirm dialog overlay
        if let Some(ref dialog) = self.confirm_dialog {
            content = content.child(dialog.clone());
        }

        content
    }
}
