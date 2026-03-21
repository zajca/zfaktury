use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::InvoiceService;
use zfaktury_domain::{Invoice, InvoiceFilter, InvoiceStatus, InvoiceType};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::select::{Select, SelectOption, SelectionChanged};
use crate::components::status_badge::render_status_badge;
use crate::components::text_input::{TextChanged, TextInput};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;
use crate::util::format::{format_amount, format_date};

/// Invoice list view with table, search, filters, and pagination.
pub struct InvoiceListView {
    service: Arc<InvoiceService>,
    loading: bool,
    error: Option<String>,
    invoices: Vec<Invoice>,
    total: i64,
    search_input: Entity<TextInput>,
    status_filter: Entity<Select>,
    type_filter: Entity<Select>,
    current_page: usize,
    page_size: i64,
}

impl InvoiceListView {
    pub fn new(service: Arc<InvoiceService>, cx: &mut Context<Self>) -> Self {
        let search_input = cx.new(|cx| TextInput::new("invoice-search", "Hledat faktury...", cx));

        let status_filter = cx.new(|_cx| {
            Select::new(
                "invoice-status-filter",
                "Vsechny stavy",
                vec![
                    SelectOption {
                        value: "".to_string(),
                        label: "Vsechny stavy".to_string(),
                    },
                    SelectOption {
                        value: "draft".to_string(),
                        label: "Koncept".to_string(),
                    },
                    SelectOption {
                        value: "sent".to_string(),
                        label: "Odeslana".to_string(),
                    },
                    SelectOption {
                        value: "paid".to_string(),
                        label: "Uhrazena".to_string(),
                    },
                    SelectOption {
                        value: "overdue".to_string(),
                        label: "Po splatnosti".to_string(),
                    },
                    SelectOption {
                        value: "cancelled".to_string(),
                        label: "Zrusena".to_string(),
                    },
                ],
            )
        });

        let type_filter = cx.new(|_cx| {
            Select::new(
                "invoice-type-filter",
                "Vse",
                vec![
                    SelectOption {
                        value: "".to_string(),
                        label: "Vse".to_string(),
                    },
                    SelectOption {
                        value: "regular".to_string(),
                        label: "Faktura".to_string(),
                    },
                    SelectOption {
                        value: "proforma".to_string(),
                        label: "Zalohova".to_string(),
                    },
                    SelectOption {
                        value: "credit_note".to_string(),
                        label: "Dobropis".to_string(),
                    },
                ],
            )
        });

        // Subscribe to search changes
        cx.subscribe(
            &search_input,
            |this: &mut Self, _input, _event: &TextChanged, cx| {
                this.current_page = 0;
                this.load_data(cx);
            },
        )
        .detach();

        // Subscribe to status filter changes
        cx.subscribe(
            &status_filter,
            |this: &mut Self, _select, _event: &SelectionChanged, cx| {
                this.current_page = 0;
                this.load_data(cx);
            },
        )
        .detach();

        // Subscribe to type filter changes
        cx.subscribe(
            &type_filter,
            |this: &mut Self, _select, _event: &SelectionChanged, cx| {
                this.current_page = 0;
                this.load_data(cx);
            },
        )
        .detach();

        let mut view = Self {
            service,
            loading: true,
            error: None,
            invoices: Vec::new(),
            total: 0,
            search_input,
            status_filter,
            type_filter,
            current_page: 0,
            page_size: 20,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let search = self.search_input.read(cx).value().to_string();
        let status_str = self
            .status_filter
            .read(cx)
            .selected_value()
            .unwrap_or("")
            .to_string();
        let type_str = self
            .type_filter
            .read(cx)
            .selected_value()
            .unwrap_or("")
            .to_string();
        let offset = (self.current_page as i32) * (self.page_size as i32);
        let limit = self.page_size as i32;

        let status = match status_str.as_str() {
            "draft" => Some(InvoiceStatus::Draft),
            "sent" => Some(InvoiceStatus::Sent),
            "paid" => Some(InvoiceStatus::Paid),
            "overdue" => Some(InvoiceStatus::Overdue),
            "cancelled" => Some(InvoiceStatus::Cancelled),
            _ => None,
        };
        let invoice_type = match type_str.as_str() {
            "regular" => Some(InvoiceType::Regular),
            "proforma" => Some(InvoiceType::Proforma),
            "credit_note" => Some(InvoiceType::CreditNote),
            _ => None,
        };

        self.loading = true;
        cx.notify();

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    service.list(InvoiceFilter {
                        search,
                        status,
                        invoice_type,
                        limit,
                        offset,
                        ..Default::default()
                    })
                })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok((invoices, total)) => {
                        this.invoices = invoices;
                        this.total = total;
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani faktur: {e}"));
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

impl EventEmitter<NavigateEvent> for InvoiceListView {}

impl Render for InvoiceListView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("invoice-list-scroll")
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
                                .child("Faktury"),
                        )
                        .child(
                            div()
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format!("({} celkem)", self.total)),
                        ),
                )
                .child(render_button(
                    "new-invoice-btn",
                    "Nova faktura",
                    ButtonVariant::Primary,
                    false,
                    false,
                    cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                        cx.emit(NavigateEvent(Route::InvoiceNew));
                    }),
                )),
        );

        // Search and filters row
        content = content.child(
            div()
                .flex()
                .items_center()
                .gap_3()
                .child(div().flex_1().child(self.search_input.clone()))
                .child(div().w(px(160.0)).child(self.status_filter.clone()))
                .child(div().w(px(140.0)).child(self.type_filter.clone())),
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
                    .child("Nacitani..."),
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
                .child(div().w(px(112.0)).child("Cislo"))
                .child(div().flex_1().child("Zakaznik"))
                .child(div().w(px(112.0)).child("Datum vystaveni"))
                .child(div().w(px(112.0)).child("Splatnost"))
                .child(div().w(px(112.0)).text_right().child("Castka"))
                .child(div().w(px(112.0)).text_right().child("Stav")),
        );

        if self.invoices.is_empty() {
            table = table.child(
                div()
                    .px_4()
                    .py_8()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadne faktury. Vytvorte novou fakturu."),
            );
        } else {
            for inv in &self.invoices {
                let customer_name = inv
                    .customer
                    .as_ref()
                    .map(|c| c.name.clone())
                    .unwrap_or_else(|| format!("ID {}", inv.customer_id));

                let inv_id = inv.id;
                table = table.child(
                    div()
                        .id(ElementId::Name(format!("inv-row-{inv_id}").into()))
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
                            cx.emit(NavigateEvent(Route::InvoiceDetail(inv_id)));
                        }))
                        .child(
                            div()
                                .w(px(112.0))
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(inv.invoice_number.clone()),
                        )
                        .child(
                            div()
                                .flex_1()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(customer_name),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format_date(inv.issue_date)),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format_date(inv.due_date)),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_right()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(inv.total_amount)),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .flex()
                                .justify_end()
                                .child(render_status_badge(&inv.status)),
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
                        "Predchozi",
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
                        "Dalsi",
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
