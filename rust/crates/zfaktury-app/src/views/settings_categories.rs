use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::CategoryService;
use zfaktury_domain::ExpenseCategory;

use crate::theme::ZfColors;

/// Settings categories view showing expense categories.
pub struct SettingsCategoriesView {
    service: Arc<CategoryService>,
    loading: bool,
    error: Option<String>,
    categories: Vec<ExpenseCategory>,
}

impl SettingsCategoriesView {
    pub fn new(service: Arc<CategoryService>, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            loading: true,
            error: None,
            categories: Vec::new(),
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
}

impl Render for SettingsCategoriesView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("settings-categories-scroll")
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
                        .child("Kategorie nakladu"),
                )
                .child(
                    div()
                        .text_sm()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child(format!("({} celkem)", self.categories.len())),
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
                .child(div().w(px(112.0)).child("Klic"))
                .child(div().flex_1().child("Nazev (CZ)"))
                .child(div().flex_1().child("Nazev (EN)"))
                .child(div().w_20().child("Barva"))
                .child(div().w_20().child("Poradi"))
                .child(div().w_20().text_right().child("Vychozi")),
        );

        if self.categories.is_empty() {
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
                        ),
                );
            }
        }

        content.child(table)
    }
}
