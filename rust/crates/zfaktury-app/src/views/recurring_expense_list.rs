use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::RecurringExpenseService;
use zfaktury_domain::RecurringExpense;

use crate::components::button::{ButtonVariant, render_button};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;
use crate::util::format::{format_amount, format_date};

/// Recurring expense list view.
pub struct RecurringExpenseListView {
    service: Arc<RecurringExpenseService>,
    loading: bool,
    error: Option<String>,
    items: Vec<RecurringExpense>,
    total: i64,
}

impl RecurringExpenseListView {
    pub fn new(service: Arc<RecurringExpenseService>, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            loading: true,
            error: None,
            items: Vec::new(),
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
                .spawn(async move { service.list(50, 0) })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok((items, total)) => {
                        this.items = items;
                        this.total = total;
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani opakovanych nakladu: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }
}

impl EventEmitter<NavigateEvent> for RecurringExpenseListView {}

impl Render for RecurringExpenseListView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("recurring-expense-list-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_4()
            .overflow_y_scroll();

        // Header with title and New button
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
                                .child("Opakovane naklady"),
                        )
                        .child(
                            div()
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format!("({} celkem)", self.total)),
                        ),
                )
                .child(render_button(
                    "new-recurring-expense-btn",
                    "Novy opakovany naklad",
                    ButtonVariant::Primary,
                    false,
                    false,
                    cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                        cx.emit(NavigateEvent(Route::RecurringExpenseNew));
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
                .child(div().flex_1().child("Popis"))
                .child(div().w(px(112.0)).child("Kategorie"))
                .child(div().w(px(112.0)).text_right().child("Castka"))
                .child(div().w_24().child("Frekvence"))
                .child(div().w(px(112.0)).child("Dalsi datum"))
                .child(div().w_20().text_right().child("Stav")),
        );

        if self.items.is_empty() {
            table = table.child(
                div()
                    .px_4()
                    .py_8()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadne opakovane naklady. Vytvorte novy opakovany naklad."),
            );
        } else {
            for re in &self.items {
                let status_text = if re.is_active { "Aktivni" } else { "Neaktivni" };
                let status_color = if re.is_active {
                    ZfColors::STATUS_GREEN
                } else {
                    ZfColors::STATUS_GRAY
                };

                let re_id = re.id;
                table = table.child(
                    div()
                        .id(ElementId::Name(format!("re-row-{re_id}").into()))
                        .flex()
                        .items_center()
                        .px_4()
                        .py_2()
                        .text_sm()
                        .border_t_1()
                        .border_color(rgb(ZfColors::BORDER_SUBTLE))
                        .cursor_pointer()
                        .hover(|s| s.bg(rgb(ZfColors::SURFACE_HOVER)))
                        .on_click(cx.listener(move |_this, _event: &ClickEvent, _window, cx| {
                            cx.emit(NavigateEvent(Route::RecurringExpenseDetail(re_id)));
                        }))
                        .child(
                            div()
                                .flex_1()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(re.description.clone()),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(re.category.clone()),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_right()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(re.amount)),
                        )
                        .child(
                            div()
                                .w_24()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(re.frequency.to_string()),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format_date(re.next_issue_date)),
                        )
                        .child(
                            div().w_20().flex().justify_end().child(
                                div()
                                    .px_2()
                                    .py(px(2.0))
                                    .rounded(px(4.0))
                                    .text_xs()
                                    .text_color(rgb(status_color))
                                    .child(status_text),
                            ),
                        ),
                );
            }
        }

        content.child(table)
    }
}
