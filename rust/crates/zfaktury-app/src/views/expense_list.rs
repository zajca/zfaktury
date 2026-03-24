use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::ExpenseService;
use zfaktury_domain::{Expense, ExpenseFilter};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::text_input::{TextChanged, TextInput};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;
use crate::util::format::{format_amount, format_date};

/// Expense list view with search, table, and pagination.
pub struct ExpenseListView {
    service: Arc<ExpenseService>,
    loading: bool,
    error: Option<String>,
    expenses: Vec<Expense>,
    total: i64,
    search_input: Entity<TextInput>,
    current_page: usize,
    page_size: i64,
}

impl ExpenseListView {
    pub fn new(service: Arc<ExpenseService>, cx: &mut Context<Self>) -> Self {
        let search_input = cx.new(|cx| TextInput::new("expense-search", "Hledat náklady...", cx));

        // Subscribe to search changes
        cx.subscribe(
            &search_input,
            |this: &mut Self, _input, _event: &TextChanged, cx| {
                this.current_page = 0;
                this.load_data(cx);
            },
        )
        .detach();

        let mut view = Self {
            service,
            loading: true,
            error: None,
            expenses: Vec::new(),
            total: 0,
            search_input,
            current_page: 0,
            page_size: 20,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let search = self.search_input.read(cx).value().to_string();
        let offset = (self.current_page as i32) * (self.page_size as i32);
        let limit = self.page_size as i32;

        self.loading = true;
        cx.notify();

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    service.list(ExpenseFilter {
                        search,
                        limit,
                        offset,
                        ..Default::default()
                    })
                })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok((expenses, total)) => {
                        this.expenses = expenses;
                        this.total = total;
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba při načítání nákladů: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn total_pages(&self) -> usize {
        if self.total == 0 {
            1
        } else {
            ((self.total as f64) / (self.page_size as f64)).ceil() as usize
        }
    }
}

impl EventEmitter<NavigateEvent> for ExpenseListView {}

impl Render for ExpenseListView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("expense-list-scroll")
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
                                .child("Náklady"),
                        )
                        .child(
                            div()
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format!("({} celkem)", self.total)),
                        ),
                )
                .child(
                    div()
                        .flex()
                        .gap_2()
                        .child(render_button(
                            "import-expense-btn",
                            "Import dokladu",
                            ButtonVariant::Secondary,
                            false,
                            false,
                            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                                cx.emit(NavigateEvent(Route::ExpenseImport));
                            }),
                        ))
                        .child(render_button(
                            "new-expense-btn",
                            "Nový náklad",
                            ButtonVariant::Primary,
                            false,
                            false,
                            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                                cx.emit(NavigateEvent(Route::ExpenseNew));
                            }),
                        )),
                ),
        );

        // Search row
        content = content.child(
            div()
                .flex()
                .items_center()
                .gap_3()
                .child(div().flex_1().child(self.search_input.clone())),
        );

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

        if self.loading {
            return content.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Načítání..."),
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
                .child(div().w_24().child("Číslo"))
                .child(div().flex_1().child("Popis"))
                .child(div().w(px(112.0)).child("Kategorie"))
                .child(div().w(px(112.0)).child("Datum"))
                .child(div().w_16().text_right().child("DPH %"))
                .child(div().w(px(112.0)).text_right().child("Částka")),
        );

        if self.expenses.is_empty() {
            table = table.child(
                div()
                    .px_4()
                    .py_8()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Žádné náklady. Vytvořte nový náklad."),
            );
        } else {
            for exp in &self.expenses {
                let exp_id = exp.id;
                table = table.child(
                    div()
                        .id(ElementId::Name(format!("exp-row-{exp_id}").into()))
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
                            cx.emit(NavigateEvent(Route::ExpenseDetail(exp_id)));
                        }))
                        .child(
                            div()
                                .w_24()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(exp.expense_number.clone()),
                        )
                        .child(
                            div()
                                .flex_1()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(exp.description.clone()),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(exp.category.clone()),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format_date(exp.issue_date)),
                        )
                        .child(
                            div()
                                .w_16()
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format!("{}%", exp.vat_rate_percent)),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_right()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(exp.amount)),
                        ),
                );
            }
        }

        content = content.child(table);

        // Pagination
        let total_pages = self.total_pages();
        if total_pages > 1 {
            let current = self.current_page;
            content = content.child(
                div()
                    .flex()
                    .items_center()
                    .justify_center()
                    .gap_3()
                    .py_2()
                    .child(render_button(
                        "btn-prev-page",
                        "Předchozí",
                        ButtonVariant::Secondary,
                        current == 0,
                        false,
                        cx.listener(|this, _event: &ClickEvent, _window, cx| {
                            if this.current_page > 0 {
                                this.current_page -= 1;
                                this.load_data(cx);
                            }
                        }),
                    ))
                    .child(
                        div()
                            .text_sm()
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .child(format!("Strana {} z {}", current + 1, total_pages)),
                    )
                    .child(render_button(
                        "btn-next-page",
                        "Další",
                        ButtonVariant::Secondary,
                        current >= total_pages - 1,
                        false,
                        cx.listener(move |this, _event: &ClickEvent, _window, cx| {
                            if this.current_page < total_pages - 1 {
                                this.current_page += 1;
                                this.load_data(cx);
                            }
                        }),
                    )),
            );
        }

        content
    }
}
