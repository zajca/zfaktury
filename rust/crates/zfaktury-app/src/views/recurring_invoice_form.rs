use std::sync::Arc;

use chrono::{Local, NaiveDate};
use gpui::*;
use zfaktury_core::service::RecurringInvoiceService;
use zfaktury_core::service::contact_svc::ContactService;
use zfaktury_domain::{
    Amount, CURRENCY_CZK, Contact, ContactFilter, Frequency, RecurringInvoice, RecurringInvoiceItem,
};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::date_input::DateInput;
use crate::components::invoice_items_editor::{InvoiceItemsEditor, ItemsChanged};
use crate::components::select::{Select, SelectOption};
use crate::components::text_area::TextArea;
use crate::components::text_input::{TextInput, render_form_field};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;

/// Recurring invoice template creation form view.
#[allow(dead_code)]
pub struct RecurringInvoiceFormView {
    service: Arc<RecurringInvoiceService>,
    contact_service: Arc<ContactService>,
    saving: bool,
    loading: bool,
    error: Option<String>,
    contacts: Vec<Contact>,

    // Form inputs
    name: Entity<TextInput>,
    customer_select: Entity<Select>,
    currency_select: Entity<Select>,
    payment_method_select: Entity<Select>,
    frequency_select: Entity<Select>,
    start_date: Entity<DateInput>,
    end_date: Entity<DateInput>,
    bank_account: Entity<TextInput>,
    bank_code: Entity<TextInput>,
    iban: Entity<TextInput>,
    swift: Entity<TextInput>,
    constant_symbol: Entity<TextInput>,
    notes: Entity<TextArea>,
    items_editor: Entity<InvoiceItemsEditor>,
}

impl EventEmitter<NavigateEvent> for RecurringInvoiceFormView {}

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

impl RecurringInvoiceFormView {
    /// Create a new recurring invoice template form.
    pub fn new_create(
        service: Arc<RecurringInvoiceService>,
        contact_service: Arc<ContactService>,
        cx: &mut Context<Self>,
    ) -> Self {
        let today = Local::now().date_naive();

        let name = cx.new(|cx| TextInput::new("ri-name", "Nazev sablony...", cx));

        let customer_select =
            cx.new(|_cx| Select::new("customer-select", "Vyberte zakaznika...", vec![]));

        let currency_select = cx.new(|cx| {
            let mut s = Select::new("currency-select", "Mena", currency_options());
            s.set_selected_value(CURRENCY_CZK, cx);
            s
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

        let bank_account = cx.new(|cx| TextInput::new("bank-account", "Cislo uctu", cx));
        let bank_code = cx.new(|cx| TextInput::new("bank-code", "Kod banky", cx));
        let iban = cx.new(|cx| TextInput::new("iban", "IBAN", cx));
        let swift = cx.new(|cx| TextInput::new("swift", "SWIFT/BIC", cx));
        let constant_symbol = cx.new(|cx| TextInput::new("const-symbol", "Konstantni symbol", cx));
        let notes = cx.new(|cx| TextArea::new("notes", "Poznamky pro zakaznika...", cx));
        let items_editor = cx.new(InvoiceItemsEditor::new);

        // Subscribe to items changes to re-render totals
        cx.subscribe(
            &items_editor,
            |_this: &mut Self, _entity, _event: &ItemsChanged, cx| {
                cx.notify();
            },
        )
        .detach();

        // Load contacts for customer picker
        let con_svc = contact_service.clone();
        let customer_sel = customer_select.clone();
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
                    customer_sel.update(cx, |sel, cx| sel.set_options(options, cx));
                    this.contacts = contacts;
                }
                this.loading = false;
                cx.notify();
            })
            .ok();
        })
        .detach();

        Self {
            service,
            contact_service,
            saving: false,
            loading: false,
            error: None,
            contacts: Vec::new(),
            name,
            customer_select,
            currency_select,
            payment_method_select,
            frequency_select,
            start_date,
            end_date,
            bank_account,
            bank_code,
            iban,
            swift,
            constant_symbol,
            notes,
            items_editor,
        }
    }

    /// Validate and save the recurring invoice template.
    fn save(&mut self, cx: &mut Context<Self>) {
        if self.saving {
            return;
        }

        // Read name
        let name = self.name.read(cx).value().to_string();
        if name.trim().is_empty() {
            self.error = Some("Zadejte nazev sablony".into());
            cx.notify();
            return;
        }

        // Read customer_id
        let customer_id: i64 = self
            .customer_select
            .read(cx)
            .selected_value()
            .and_then(|v| v.parse().ok())
            .unwrap_or(0);
        if customer_id == 0 {
            self.error = Some("Vyberte zakaznika".into());
            cx.notify();
            return;
        }

        // Read items from the items editor and convert to RecurringInvoiceItem
        let invoice_items = self.items_editor.read(cx).to_invoice_items(cx);
        if invoice_items.is_empty() {
            self.error = Some("Pridejte alespon jednu polozku".into());
            cx.notify();
            return;
        }

        let items: Vec<RecurringInvoiceItem> = invoice_items
            .into_iter()
            .map(|ii| RecurringInvoiceItem {
                id: 0,
                recurring_invoice_id: 0,
                description: ii.description,
                quantity: ii.quantity,
                unit: ii.unit,
                unit_price: ii.unit_price,
                vat_rate_percent: ii.vat_rate_percent,
                sort_order: ii.sort_order,
            })
            .collect();

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

        // Read other fields
        let currency = self
            .currency_select
            .read(cx)
            .selected_value()
            .unwrap_or(CURRENCY_CZK)
            .to_string();
        let payment_method = self
            .payment_method_select
            .read(cx)
            .selected_value()
            .unwrap_or("bank_transfer")
            .to_string();
        let bank_account = self.bank_account.read(cx).value().to_string();
        let bank_code = self.bank_code.read(cx).value().to_string();
        let iban = self.iban.read(cx).value().to_string();
        let swift = self.swift.read(cx).value().to_string();
        let constant_symbol = self.constant_symbol.read(cx).value().to_string();
        let notes = self.notes.read(cx).value().to_string();

        let now = Local::now().naive_local();
        let mut ri = RecurringInvoice {
            id: 0,
            name,
            customer_id,
            customer: None,
            frequency,
            next_issue_date,
            end_date,
            currency_code: currency,
            exchange_rate: Amount::new(1, 0),
            payment_method,
            bank_account,
            bank_code,
            iban,
            swift,
            constant_symbol,
            notes,
            is_active: true,
            items,
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
                    service.create(&mut ri)?;
                    Ok::<i64, zfaktury_domain::DomainError>(ri.id)
                })
                .await;
            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(id) => cx.emit(NavigateEvent(Route::RecurringInvoiceDetail(id))),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }
}

impl Render for RecurringInvoiceFormView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("recurring-invoice-form-scroll")
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
                .child("Nova sablona faktury"),
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
                // Row 1: name, customer
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(
                            div()
                                .flex_1()
                                .child(render_form_field("Nazev sablony", self.name.clone())),
                        )
                        .child(div().flex_1().child(render_labeled_field(
                            "Zakaznik",
                            self.customer_select.clone(),
                        ))),
                )
                // Row 2: currency, payment method, frequency
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(
                            div()
                                .w(px(128.0))
                                .child(render_labeled_field("Mena", self.currency_select.clone())),
                        )
                        .child(div().flex_1().child(render_labeled_field(
                            "Zpusob platby",
                            self.payment_method_select.clone(),
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

        // Card 2: Payment info
        outer = outer.child(render_card(
            "Platebni udaje",
            div()
                .flex()
                .flex_col()
                .gap_4()
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(render_form_field("Cislo uctu", self.bank_account.clone()))
                        .child(render_form_field("Kod banky", self.bank_code.clone()))
                        .child(render_form_field(
                            "Konstantni symbol",
                            self.constant_symbol.clone(),
                        )),
                )
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(render_form_field("IBAN", self.iban.clone()))
                        .child(render_form_field("SWIFT/BIC", self.swift.clone())),
                ),
        ));

        // Card 3: Invoice items
        outer = outer.child(render_card(
            "Polozky faktury",
            div().child(self.items_editor.clone()),
        ));

        // Card 4: Notes
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
                cx.emit(NavigateEvent(Route::RecurringInvoiceList));
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
