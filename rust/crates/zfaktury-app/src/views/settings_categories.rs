use std::sync::Arc;

use chrono::Local;
use gpui::*;
use zfaktury_core::service::CategoryService;
use zfaktury_domain::ExpenseCategory;

use crate::components::button::{ButtonVariant, render_button};
use crate::components::confirm_dialog::{ConfirmDialog, ConfirmDialogResult};
use crate::components::number_input::NumberInput;
use crate::components::text_input::TextInput;
use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;

/// Row editing state for inline edit/create.
struct EditingRow {
    /// Category ID being edited (0 for new).
    id: i64,
    key: Entity<TextInput>,
    label_cs: Entity<TextInput>,
    label_en: Entity<TextInput>,
    color: Entity<TextInput>,
    sort_order: Entity<NumberInput>,
}

/// Settings categories view with inline CRUD.
pub struct SettingsCategoriesView {
    service: Arc<CategoryService>,
    loading: bool,
    saving: bool,
    error: Option<String>,
    categories: Vec<ExpenseCategory>,
    /// Currently edited row (None = no editing, Some with id=0 = new row).
    editing: Option<EditingRow>,
    /// Confirm dialog for delete.
    confirm_dialog: Option<Entity<ConfirmDialog>>,
    delete_id: Option<i64>,
}

impl SettingsCategoriesView {
    pub fn new(service: Arc<CategoryService>, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            loading: true,
            saving: false,
            error: None,
            categories: Vec::new(),
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
                    Ok(categories) => this.categories = categories,
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani kategorii: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn start_new(&mut self, cx: &mut Context<Self>) {
        let key = cx.new(|cx| TextInput::new("cat-new-key", "Klic...", cx));
        let label_cs = cx.new(|cx| TextInput::new("cat-new-label-cs", "Nazev CZ...", cx));
        let label_en = cx.new(|cx| TextInput::new("cat-new-label-en", "Nazev EN...", cx));
        let color = cx.new(|cx| {
            let mut t = TextInput::new("cat-new-color", "#000000", cx);
            t.set_value("#3b82f6", cx);
            t
        });
        let sort_order = cx.new(|cx| {
            NumberInput::new("cat-new-sort", "0", cx)
                .integer_only()
                .with_value("0")
        });
        self.editing = Some(EditingRow {
            id: 0,
            key,
            label_cs,
            label_en,
            color,
            sort_order,
        });
        self.error = None;
        cx.notify();
    }

    fn start_edit(&mut self, cat: &ExpenseCategory, cx: &mut Context<Self>) {
        let cat_id = cat.id;
        let key = cx.new(|cx| {
            let mut t = TextInput::new(
                SharedString::from(format!("cat-edit-key-{}", cat_id)),
                "Klic...",
                cx,
            );
            t.set_value(&cat.key, cx);
            t
        });
        let label_cs = cx.new(|cx| {
            let mut t = TextInput::new(
                SharedString::from(format!("cat-edit-cs-{}", cat_id)),
                "Nazev CZ...",
                cx,
            );
            t.set_value(&cat.label_cs, cx);
            t
        });
        let label_en = cx.new(|cx| {
            let mut t = TextInput::new(
                SharedString::from(format!("cat-edit-en-{}", cat_id)),
                "Nazev EN...",
                cx,
            );
            t.set_value(&cat.label_en, cx);
            t
        });
        let color = cx.new(|cx| {
            let mut t = TextInput::new(
                SharedString::from(format!("cat-edit-color-{}", cat_id)),
                "#000000",
                cx,
            );
            t.set_value(&cat.color, cx);
            t
        });
        let sort_order = cx.new(|cx| {
            NumberInput::new(
                SharedString::from(format!("cat-edit-sort-{}", cat_id)),
                "0",
                cx,
            )
            .integer_only()
            .with_value(cat.sort_order.to_string())
        });
        self.editing = Some(EditingRow {
            id: cat_id,
            key,
            label_cs,
            label_en,
            color,
            sort_order,
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

        let key_val = editing.key.read(cx).value().to_string();
        let label_cs_val = editing.label_cs.read(cx).value().to_string();
        let label_en_val = editing.label_en.read(cx).value().to_string();
        let color_val = editing.color.read(cx).value().to_string();
        let sort_val: i32 = editing.sort_order.read(cx).value().parse().unwrap_or(0);
        let edit_id = editing.id;

        if key_val.trim().is_empty() || label_cs_val.trim().is_empty() {
            self.error = Some("Klic a nazev CZ jsou povinne.".into());
            cx.notify();
            return;
        }

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.service.clone();
        let now = Local::now().naive_local();

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    let mut cat = ExpenseCategory {
                        id: edit_id,
                        key: key_val,
                        label_cs: label_cs_val,
                        label_en: label_en_val,
                        color: color_val,
                        sort_order: sort_val,
                        is_default: false,
                        created_at: now,
                        deleted_at: None,
                    };
                    if edit_id == 0 {
                        service.create(&mut cat)?;
                    } else {
                        service.update(&mut cat)?;
                    }
                    service.list()
                })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(categories) => {
                        this.categories = categories;
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
                "Smazat kategorii",
                "Opravdu chcete smazat tuto kategorii?",
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
                    Ok(categories) => {
                        this.categories = categories;
                        // Clear editing if we deleted the row being edited.
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
            .child(div().w(px(112.0)).child(editing.key.clone()))
            .child(div().flex_1().child(editing.label_cs.clone()))
            .child(div().flex_1().child(editing.label_en.clone()))
            .child(div().w_20().child(editing.color.clone()))
            .child(div().w_20().child(editing.sort_order.clone()))
    }
}

impl EventEmitter<NavigateEvent> for SettingsCategoriesView {}

impl Render for SettingsCategoriesView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("settings-categories-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_4()
            .overflow_y_scroll();

        // Title bar
        let is_new_row = self.editing.as_ref().map(|e| e.id == 0).unwrap_or(false);

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
                                .child("Kategorie nakladu"),
                        )
                        .child(
                            div()
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format!("({} celkem)", self.categories.len())),
                        ),
                )
                .child(render_button(
                    "cat-new-btn",
                    "Nova kategorie",
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
                .child(div().w(px(112.0)).child("Klic"))
                .child(div().flex_1().child("Nazev (CZ)"))
                .child(div().flex_1().child("Nazev (EN)"))
                .child(div().w_20().child("Barva"))
                .child(div().w_20().child("Poradi"))
                .child(div().w_20().text_right().child("Vychozi"))
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
                                "cat-new-cancel",
                                "Zrusit",
                                ButtonVariant::Secondary,
                                self.saving,
                                false,
                                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                    this.cancel_edit(cx);
                                }),
                            ))
                            .child(render_button(
                                "cat-new-save",
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

        if self.categories.is_empty() && !is_new_row {
            table = table.child(
                div()
                    .px_4()
                    .py_8()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadne kategorie."),
            );
        } else {
            for cat in &self.categories {
                let cat_id = cat.id;
                let is_editing_this = self
                    .editing
                    .as_ref()
                    .map(|e| e.id == cat_id)
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
                                                "cat-edit-cancel-{}",
                                                cat_id
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
                                            SharedString::from(format!("cat-edit-save-{}", cat_id)),
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
                                    .child(cat.key.clone()),
                            )
                            .child(
                                div()
                                    .flex_1()
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child(cat.label_cs.clone()),
                            )
                            .child(
                                div()
                                    .flex_1()
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(cat.label_en.clone()),
                            )
                            .child(
                                div()
                                    .w_20()
                                    .flex()
                                    .items_center()
                                    .gap_2()
                                    .child(
                                        div()
                                            .w_4()
                                            .h_4()
                                            .rounded(px(2.0))
                                            .bg(rgb(ZfColors::TEXT_MUTED)),
                                    )
                                    .child(
                                        div()
                                            .text_xs()
                                            .text_color(rgb(ZfColors::TEXT_MUTED))
                                            .child(cat.color.clone()),
                                    ),
                            )
                            .child(
                                div()
                                    .w_20()
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(cat.sort_order.to_string()),
                            )
                            .child(
                                div()
                                    .w_20()
                                    .text_right()
                                    .text_color(if cat.is_default {
                                        rgb(ZfColors::STATUS_GREEN)
                                    } else {
                                        rgb(ZfColors::TEXT_MUTED)
                                    })
                                    .child(if cat.is_default { "Ano" } else { "-" }),
                            )
                            .child(
                                div()
                                    .w(px(80.0))
                                    .flex()
                                    .justify_end()
                                    .gap_1()
                                    .child(render_button(
                                        SharedString::from(format!("cat-edit-{}", cat_id)),
                                        "Upravit",
                                        ButtonVariant::Secondary,
                                        has_editing || self.saving,
                                        false,
                                        cx.listener(
                                            move |this, _event: &ClickEvent, _window, cx| {
                                                if let Some(cat) =
                                                    this.categories.iter().find(|c| c.id == cat_id)
                                                {
                                                    let cat = cat.clone();
                                                    this.start_edit(&cat, cx);
                                                }
                                            },
                                        ),
                                    ))
                                    .child(render_button(
                                        SharedString::from(format!("cat-del-{}", cat_id)),
                                        "Smazat",
                                        ButtonVariant::Danger,
                                        has_editing || self.saving,
                                        false,
                                        cx.listener(
                                            move |this, _event: &ClickEvent, _window, cx| {
                                                this.request_delete(cat_id, cx);
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
