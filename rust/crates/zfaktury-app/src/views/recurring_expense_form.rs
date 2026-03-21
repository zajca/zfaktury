use std::sync::Arc;

use chrono::{Local, NaiveDate};
use gpui::*;
use zfaktury_core::service::RecurringExpenseService;
use zfaktury_core::service::category_svc::CategoryService;
use zfaktury_core::service::contact_svc::ContactService;
use zfaktury_domain::{
    Amount, CURRENCY_CZK, Contact, ContactFilter, ExpenseCategory, Frequency, RecurringExpense,
};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::date_input::DateInput;
use crate::components::number_input::NumberInput;
use crate::components::select::{Select, SelectOption};
use crate::components::text_area::TextArea;
use crate::components::text_input::{TextInput, render_form_field};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;

/// Recurring expense template creation form view.
#[allow(dead_code)]
pub struct RecurringExpenseFormView {
    service: Arc<RecurringExpenseService>,
    contact_service: Arc<ContactService>,
    category_service: Arc<CategoryService>,
    saving: bool,
    loading: bool,
    error: Option<String>,

    // Loaded reference data
    contacts: Vec<Contact>,
    categories: Vec<ExpenseCategory>,

    // Form fields
    name: Entity<TextInput>,
    description: Entity<TextInput>,
    category_select: Entity<Select>,
    vendor_select: Entity<Select>,
    amount_input: Entity<NumberInput>,
    currency_select: Entity<Select>,
    vat_rate_select: Entity<Select>,
    business_percent: Entity<NumberInput>,
    payment_method_select: Entity<Select>,
    frequency_select: Entity<Select>,
    start_date: Entity<DateInput>,
    end_date: Entity<DateInput>,
    notes: Entity<TextArea>,
    is_tax_deductible: bool,
}

impl EventEmitter<NavigateEvent> for RecurringExpenseFormView {}

fn currency_options() -> Vec<SelectOption> {
    vec![
        SelectOption {
            value: "CZK".to_string(),
            label: "CZK".to_string(),
        },
        SelectOption {
            value: "EUR".to_string(),
            label: "EUR".to_string(),
        },
        SelectOption {
            value: "USD".to_string(),
            label: "USD".to_string(),
        },
    ]
}

fn vat_rate_options() -> Vec<SelectOption> {
    vec![
        SelectOption {
            value: "0".to_string(),
            label: "0%".to_string(),
        },
        SelectOption {
            value: "12".to_string(),
            label: "12%".to_string(),
        },
        SelectOption {
            value: "21".to_string(),
            label: "21%".to_string(),
        },
    ]
}

fn payment_method_options() -> Vec<SelectOption> {
    vec![
        SelectOption {
            value: "bank_transfer".to_string(),
            label: "Bankovni prevod".to_string(),
        },
        SelectOption {
            value: "cash".to_string(),
            label: "Hotovost".to_string(),
        },
        SelectOption {
            value: "card".to_string(),
            label: "Kartou".to_string(),
        },
    ]
}

fn frequency_options() -> Vec<SelectOption> {
    vec![
        SelectOption {
            value: "weekly".to_string(),
            label: "Tydenni".to_string(),
        },
        SelectOption {
            value: "monthly".to_string(),
            label: "Mesicni".to_string(),
        },
        SelectOption {
            value: "quarterly".to_string(),
            label: "Ctvrtletni".to_string(),
        },
        SelectOption {
            value: "yearly".to_string(),
            label: "Rocni".to_string(),
        },
    ]
}

fn render_labeled_field(label: &str, child: impl IntoElement) -> Div {
    div()
        .flex()
        .flex_col()
        .gap_1()
        .child(
            div()
                .text_xs()
                .font_weight(FontWeight::MEDIUM)
                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                .child(label.to_string()),
        )
        .child(child)
}

fn render_card(title: &str, content: Div) -> Div {
    div()
        .p_4()
        .bg(rgb(ZfColors::SURFACE))
        .rounded_md()
        .border_1()
        .border_color(rgb(ZfColors::BORDER))
        .flex()
        .flex_col()
        .gap_4()
        .child(
            div()
                .text_sm()
                .font_weight(FontWeight::SEMIBOLD)
                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                .child(title.to_string()),
        )
        .child(content)
}

impl RecurringExpenseFormView {
    /// Create a new recurring expense template form.
    pub fn new_create(
        service: Arc<RecurringExpenseService>,
        contact_service: Arc<ContactService>,
        category_service: Arc<CategoryService>,
        cx: &mut Context<Self>,
    ) -> Self {
        let today = Local::now().date_naive();

        let name = cx.new(|cx| TextInput::new("re-name", "Nazev sablony...", cx));
        let description = cx.new(|cx| TextInput::new("re-description", "Popis nakladu...", cx));

        let category_select =
            cx.new(|_cx| Select::new("category-select", "Vyberte kategorii...", vec![]));

        let vendor_select =
            cx.new(|_cx| Select::new("vendor-select", "Vyberte dodavatele...", vec![]));

        let amount_input = cx.new(|cx| NumberInput::new("amount-input", "0,00", cx));

        let currency_select = cx.new(|cx| {
            let mut s = Select::new("currency-select", "Mena", currency_options());
            s.set_selected_value(CURRENCY_CZK, cx);
            s
        });

        let vat_rate_select = cx.new(|cx| {
            let mut s = Select::new("vat-rate-select", "DPH sazba", vat_rate_options());
            s.set_selected_value("21", cx);
            s
        });

        let business_percent = cx.new(|cx| {
            NumberInput::new("biz-percent", "100", cx)
                .integer_only()
                .with_value("100")
        });

        let payment_method_select = cx.new(|cx| {
            let mut s = Select::new(
                "payment-method-select",
                "Zpusob platby",
                payment_method_options(),
            );
            s.set_selected_value("bank_transfer", cx);
            s
        });

        let frequency_select = cx.new(|cx| {
            let mut s = Select::new("frequency-select", "Frekvence", frequency_options());
            s.set_selected_value("monthly", cx);
            s
        });

        let start_date = cx.new(|cx| {
            let mut d = DateInput::new("start-date", cx);
            d.set_iso_value(&format!("{}", today), cx);
            d
        });

        let end_date = cx.new(|cx| DateInput::new("end-date", cx));

        let notes = cx.new(|cx| TextArea::new("notes", "Volitelne poznamky...", cx));

        // Load contacts for vendor picker
        let con_svc = contact_service.clone();
        let vendor_sel = vendor_select.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    con_svc.list(ContactFilter {
                        limit: 200,
                        ..Default::default()
                    })
                })
                .await;
            this.update(cx, |this, cx| {
                if let Ok((contacts, _)) = result {
                    let options: Vec<SelectOption> = contacts
                        .iter()
                        .map(|c| SelectOption {
                            value: c.id.to_string(),
                            label: c.name.clone(),
                        })
                        .collect();
                    vendor_sel.update(cx, |sel, cx| sel.set_options(options, cx));
                    this.contacts = contacts;
                }
                cx.notify();
            })
            .ok();
        })
        .detach();

        // Load categories for category picker
        let cat_svc = category_service.clone();
        let cat_sel = category_select.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { cat_svc.list() })
                .await;
            this.update(cx, |this, cx| {
                if let Ok(categories) = result {
                    let options: Vec<SelectOption> = categories
                        .iter()
                        .map(|c| SelectOption {
                            value: c.key.clone(),
                            label: c.label_cs.clone(),
                        })
                        .collect();
                    cat_sel.update(cx, |sel, cx| sel.set_options(options, cx));
                    this.categories = categories;
                }
                cx.notify();
            })
            .ok();
        })
        .detach();

        Self {
            service,
            contact_service,
            category_service,
            saving: false,
            loading: false,
            error: None,
            contacts: Vec::new(),
            categories: Vec::new(),
            name,
            description,
            category_select,
            vendor_select,
            amount_input,
            currency_select,
            vat_rate_select,
            business_percent,
            payment_method_select,
            frequency_select,
            start_date,
            end_date,
            notes,
            is_tax_deductible: true,
        }
    }

    /// Validate and save the recurring expense template.
    fn save(&mut self, cx: &mut Context<Self>) {
        if self.saving {
            return;
        }

        // Read and validate name
        let name = self.name.read(cx).value().to_string();
        if name.trim().is_empty() {
            self.error = Some("Zadejte nazev sablony".into());
            cx.notify();
            return;
        }

        // Read and validate description
        let description = self.description.read(cx).value().to_string();
        if description.trim().is_empty() {
            self.error = Some("Zadejte popis nakladu".into());
            cx.notify();
            return;
        }

        // Read amount
        let amount_val = match self.amount_input.read(cx).to_amount() {
            Some(a) => a,
            None => {
                self.error = Some("Neplatna castka".into());
                cx.notify();
                return;
            }
        };
        if amount_val == Amount::ZERO {
            self.error = Some("Zadejte castku".into());
            cx.notify();
            return;
        }

        // Read category
        let category = self
            .category_select
            .read(cx)
            .selected_value()
            .unwrap_or("")
            .to_string();

        // Read vendor (optional)
        let vendor_id: Option<i64> = self
            .vendor_select
            .read(cx)
            .selected_value()
            .and_then(|v| v.parse().ok());

        // Read currency
        let currency = self
            .currency_select
            .read(cx)
            .selected_value()
            .unwrap_or(CURRENCY_CZK)
            .to_string();

        // Read VAT rate
        let vat_rate: i32 = self
            .vat_rate_select
            .read(cx)
            .selected_value()
            .and_then(|v| v.parse().ok())
            .unwrap_or(21);

        // Read business percent
        let biz_pct: i32 = self
            .business_percent
            .read(cx)
            .value()
            .parse()
            .unwrap_or(100);

        // Read payment method
        let payment_method = self
            .payment_method_select
            .read(cx)
            .selected_value()
            .unwrap_or("bank_transfer")
            .to_string();

        // Read frequency
        let freq_str = self
            .frequency_select
            .read(cx)
            .selected_value()
            .unwrap_or("monthly")
            .to_string();
        let frequency = match freq_str.as_str() {
            "weekly" => Frequency::Weekly,
            "quarterly" => Frequency::Quarterly,
            "yearly" => Frequency::Yearly,
            _ => Frequency::Monthly,
        };

        // Read dates
        let start_date_str = self.start_date.read(cx).iso_value().to_string();
        let next_issue_date = NaiveDate::parse_from_str(&start_date_str, "%Y-%m-%d")
            .unwrap_or_else(|_| Local::now().date_naive());

        let end_date_str = self.end_date.read(cx).iso_value().to_string();
        let end_date = NaiveDate::parse_from_str(&end_date_str, "%Y-%m-%d").ok();

        // Read notes
        let notes = self.notes.read(cx).value().to_string();

        let now = chrono::Local::now().naive_local();
        let mut re = RecurringExpense {
            id: 0,
            name,
            vendor_id,
            vendor: None,
            category,
            description,
            amount: amount_val,
            currency_code: currency,
            exchange_rate: Amount::new(1, 0),
            vat_rate_percent: vat_rate,
            vat_amount: Amount::ZERO,
            is_tax_deductible: self.is_tax_deductible,
            business_percent: biz_pct,
            payment_method,
            notes,
            frequency,
            next_issue_date,
            end_date,
            is_active: true,
            created_at: now,
            updated_at: now,
            deleted_at: None,
        };

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    service.create(&mut re)?;
                    Ok::<i64, zfaktury_domain::DomainError>(re.id)
                })
                .await;
            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(id) => cx.emit(NavigateEvent(Route::RecurringExpenseDetail(id))),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }
}

impl Render for RecurringExpenseFormView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("recurring-expense-form-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll();

        // Title
        outer = outer.child(
            div()
                .text_xl()
                .font_weight(FontWeight::SEMIBOLD)
                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                .child("Novy opakovany naklad"),
        );

        // Error message
        if let Some(ref error) = self.error {
            outer = outer.child(
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

        // Loading state
        if self.loading {
            return outer.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Nacitani..."),
            );
        }

        // Card 1: Basic info
        outer = outer.child(render_card(
            "Zakladni udaje",
            div()
                .flex()
                .flex_col()
                .gap_4()
                // Row 1: name, description
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(
                            div()
                                .w(px(240.0))
                                .child(render_form_field("Nazev sablony", self.name.clone())),
                        )
                        .child(
                            div()
                                .flex_1()
                                .child(render_form_field("Popis", self.description.clone())),
                        ),
                )
                // Row 2: category, vendor, frequency
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(div().flex_1().child(render_labeled_field(
                            "Kategorie",
                            self.category_select.clone(),
                        )))
                        .child(div().flex_1().child(render_labeled_field(
                            "Dodavatel",
                            self.vendor_select.clone(),
                        )))
                        .child(div().w(px(160.0)).child(render_labeled_field(
                            "Frekvence",
                            self.frequency_select.clone(),
                        ))),
                )
                // Row 3: dates
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(render_labeled_field(
                            "Datum zahajeni",
                            self.start_date.clone(),
                        ))
                        .child(render_labeled_field(
                            "Datum ukonceni (volitelne)",
                            self.end_date.clone(),
                        )),
                ),
        ));

        // Card 2: Amount and VAT
        let tax_label = if self.is_tax_deductible {
            "Danove uznatelne: Ano"
        } else {
            "Danove uznatelne: Ne"
        };

        let tax_bg = if self.is_tax_deductible {
            ZfColors::STATUS_GREEN_BG
        } else {
            ZfColors::SURFACE
        };

        outer = outer.child(render_card(
            "Castka a DPH",
            div()
                .flex()
                .flex_col()
                .gap_4()
                // Row 1: amount, currency, vat_rate
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(
                            div()
                                .flex_1()
                                .child(render_labeled_field("Castka", self.amount_input.clone())),
                        )
                        .child(
                            div()
                                .w(px(128.0))
                                .child(render_labeled_field("Mena", self.currency_select.clone())),
                        )
                        .child(div().w(px(128.0)).child(render_labeled_field(
                            "DPH sazba",
                            self.vat_rate_select.clone(),
                        ))),
                )
                // Row 2: business_percent, payment_method, tax deductible toggle
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .items_end()
                        .child(div().w(px(128.0)).child(render_labeled_field(
                            "Obchodni podil %",
                            self.business_percent.clone(),
                        )))
                        .child(div().flex_1().child(render_labeled_field(
                            "Zpusob platby",
                            self.payment_method_select.clone(),
                        )))
                        .child(
                            div()
                                .id("tax-toggle")
                                .cursor_pointer()
                                .px_3()
                                .py_2()
                                .bg(rgb(tax_bg))
                                .border_1()
                                .border_color(rgb(ZfColors::BORDER))
                                .rounded_md()
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .on_click(cx.listener(|this, _ev: &ClickEvent, _w, cx| {
                                    this.is_tax_deductible = !this.is_tax_deductible;
                                    cx.notify();
                                }))
                                .child(tax_label),
                        ),
                ),
        ));

        // Card 3: Notes
        outer = outer.child(render_card(
            "Poznamky",
            div().child(render_labeled_field("Poznamky", self.notes.clone())),
        ));

        // Button bar
        let cancel_btn = render_button(
            "cancel-btn",
            "Zrusit",
            ButtonVariant::Secondary,
            self.saving,
            false,
            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                cx.emit(NavigateEvent(Route::RecurringExpenseList));
            }),
        );

        let save_btn = render_button(
            "save-btn",
            "Ulozit sablonu",
            ButtonVariant::Primary,
            false,
            self.saving,
            cx.listener(|this, _event: &ClickEvent, _window, cx| {
                this.save(cx);
            }),
        );

        outer = outer.child(
            div()
                .flex()
                .justify_between()
                .child(cancel_btn)
                .child(save_btn),
        );

        outer
    }
}
