use std::sync::Arc;

use chrono::{Local, NaiveDate};
use gpui::*;
use zfaktury_core::service::contact_svc::ContactService;
use zfaktury_core::service::invoice_svc::InvoiceService;
use zfaktury_domain::{
    Amount, CURRENCY_CZK, Contact, ContactFilter, Invoice, InvoiceStatus, InvoiceType, RelationType,
};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::date_input::DateInput;
use crate::components::invoice_items_editor::{InvoiceItemsEditor, ItemsChanged};
use crate::components::select::{Select, SelectOption};
use crate::components::text_area::TextArea;
use crate::components::text_input::{TextInput, render_form_field};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;

/// Invoice creation/edit form view.
#[allow(dead_code)]
pub struct InvoiceFormView {
    invoice_service: Arc<InvoiceService>,
    contact_service: Arc<ContactService>,
    is_edit: bool,
    invoice_id: Option<i64>,
    saving: bool,
    loading: bool,
    error: Option<String>,
    contacts: Vec<Contact>,

    // Form inputs
    customer_select: Entity<Select>,
    invoice_type_select: Entity<Select>,
    currency_select: Entity<Select>,
    payment_method_select: Entity<Select>,
    issue_date: Entity<DateInput>,
    due_date: Entity<DateInput>,
    delivery_date: Entity<DateInput>,
    variable_symbol: Entity<TextInput>,
    constant_symbol: Entity<TextInput>,
    bank_account: Entity<TextInput>,
    bank_code: Entity<TextInput>,
    iban: Entity<TextInput>,
    swift: Entity<TextInput>,
    notes: Entity<TextArea>,
    internal_notes: Entity<TextArea>,
    items_editor: Entity<InvoiceItemsEditor>,
}

impl EventEmitter<NavigateEvent> for InvoiceFormView {}

fn invoice_type_options() -> Vec<SelectOption> {
    vec![
        SelectOption {
            value: "regular".to_string(),
            label: "Faktura".to_string(),
        },
        SelectOption {
            value: "proforma".to_string(),
            label: "Zalohova faktura".to_string(),
        },
        SelectOption {
            value: "credit_note".to_string(),
            label: "Dobropis".to_string(),
        },
    ]
}

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

impl InvoiceFormView {
    /// Create a new invoice form (create mode).
    pub fn new_create(
        invoice_service: Arc<InvoiceService>,
        contact_service: Arc<ContactService>,
        cx: &mut Context<Self>,
    ) -> Self {
        let today = Local::now().date_naive();
        let due = today + chrono::Duration::days(14);

        let customer_select =
            cx.new(|_cx| Select::new("customer-select", "Vyberte zakaznika...", vec![]));

        let invoice_type_select = cx.new(|cx| {
            let mut s = Select::new("type-select", "Typ faktury", invoice_type_options());
            s.set_selected_value("regular", cx);
            s
        });

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

        let issue_date = cx.new(|cx| {
            let mut d = DateInput::new("issue-date", cx);
            d.set_iso_value(&format!("{}", today), cx);
            d
        });

        let due_date = cx.new(|cx| {
            let mut d = DateInput::new("due-date", cx);
            d.set_iso_value(&format!("{}", due), cx);
            d
        });

        let delivery_date = cx.new(|cx| {
            let mut d = DateInput::new("delivery-date", cx);
            d.set_iso_value(&format!("{}", today), cx);
            d
        });

        let variable_symbol = cx.new(|cx| TextInput::new("var-symbol", "Variabilni symbol", cx));
        let constant_symbol = cx.new(|cx| TextInput::new("const-symbol", "Konstantni symbol", cx));
        let bank_account = cx.new(|cx| TextInput::new("bank-account", "Cislo uctu", cx));
        let bank_code = cx.new(|cx| TextInput::new("bank-code", "Kod banky", cx));
        let iban = cx.new(|cx| TextInput::new("iban", "IBAN", cx));
        let swift = cx.new(|cx| TextInput::new("swift", "SWIFT/BIC", cx));
        let notes = cx.new(|cx| TextArea::new("notes", "Poznamky pro zakaznika...", cx));
        let internal_notes =
            cx.new(|cx| TextArea::new("internal-notes", "Interni poznamky (jen pro vas)...", cx));
        let items_editor = cx.new(InvoiceItemsEditor::new);

        // Subscribe to items changes to re-render totals
        cx.subscribe(
            &items_editor,
            |_this: &mut Self, _entity, _event: &ItemsChanged, cx| {
                cx.notify();
            },
        )
        .detach();

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
            invoice_service,
            contact_service,
            is_edit: false,
            invoice_id: None,
            saving: false,
            loading: false,
            error: None,
            contacts: Vec::new(),
            customer_select,
            invoice_type_select,
            currency_select,
            payment_method_select,
            issue_date,
            due_date,
            delivery_date,
            variable_symbol,
            constant_symbol,
            bank_account,
            bank_code,
            iban,
            swift,
            notes,
            internal_notes,
            items_editor,
        }
    }

    /// Create an invoice form in edit mode (loads existing invoice).
    pub fn new_edit(
        invoice_service: Arc<InvoiceService>,
        contact_service: Arc<ContactService>,
        id: i64,
        cx: &mut Context<Self>,
    ) -> Self {
        let customer_select =
            cx.new(|_cx| Select::new("customer-select", "Vyberte zakaznika...", vec![]));

        let invoice_type_select =
            cx.new(|_cx| Select::new("type-select", "Typ faktury", invoice_type_options()));

        let currency_select =
            cx.new(|_cx| Select::new("currency-select", "Mena", currency_options()));

        let payment_method_select = cx.new(|_cx| {
            Select::new(
                "payment-method-select",
                "Zpusob platby",
                payment_method_options(),
            )
        });

        let issue_date = cx.new(|cx| DateInput::new("issue-date", cx));
        let due_date = cx.new(|cx| DateInput::new("due-date", cx));
        let delivery_date = cx.new(|cx| DateInput::new("delivery-date", cx));

        let variable_symbol = cx.new(|cx| TextInput::new("var-symbol", "Variabilni symbol", cx));
        let constant_symbol = cx.new(|cx| TextInput::new("const-symbol", "Konstantni symbol", cx));
        let bank_account = cx.new(|cx| TextInput::new("bank-account", "Cislo uctu", cx));
        let bank_code = cx.new(|cx| TextInput::new("bank-code", "Kod banky", cx));
        let iban = cx.new(|cx| TextInput::new("iban", "IBAN", cx));
        let swift = cx.new(|cx| TextInput::new("swift", "SWIFT/BIC", cx));
        let notes = cx.new(|cx| TextArea::new("notes", "Poznamky pro zakaznika...", cx));
        let internal_notes =
            cx.new(|cx| TextArea::new("internal-notes", "Interni poznamky (jen pro vas)...", cx));
        let items_editor = cx.new(InvoiceItemsEditor::new);

        cx.subscribe(
            &items_editor,
            |_this: &mut Self, _entity, _event: &ItemsChanged, cx| {
                cx.notify();
            },
        )
        .detach();

        // Load contacts
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
                cx.notify();
            })
            .ok();
        })
        .detach();

        // Load the invoice data
        let inv_svc = invoice_service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { inv_svc.get_by_id(id) })
                .await;
            this.update(cx, |this, cx| {
                match result {
                    Ok(inv) => this.populate_from_invoice(&inv, cx),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                this.loading = false;
                cx.notify();
            })
            .ok();
        })
        .detach();

        Self {
            invoice_service,
            contact_service,
            is_edit: true,
            invoice_id: Some(id),
            saving: false,
            loading: true,
            error: None,
            contacts: Vec::new(),
            customer_select,
            invoice_type_select,
            currency_select,
            payment_method_select,
            issue_date,
            due_date,
            delivery_date,
            variable_symbol,
            constant_symbol,
            bank_account,
            bank_code,
            iban,
            swift,
            notes,
            internal_notes,
            items_editor,
        }
    }

    /// Populate all form fields from an existing Invoice.
    fn populate_from_invoice(&mut self, inv: &Invoice, cx: &mut Context<Self>) {
        self.customer_select.update(cx, |s, cx| {
            s.set_selected_value(&inv.customer_id.to_string(), cx);
        });
        self.invoice_type_select.update(cx, |s, cx| {
            s.set_selected_value(&inv.invoice_type.to_string(), cx);
        });
        self.currency_select.update(cx, |s, cx| {
            s.set_selected_value(&inv.currency_code, cx);
        });
        self.payment_method_select.update(cx, |s, cx| {
            s.set_selected_value(&inv.payment_method, cx);
        });
        self.issue_date.update(cx, |d, cx| {
            d.set_iso_value(&inv.issue_date.to_string(), cx);
        });
        self.due_date.update(cx, |d, cx| {
            d.set_iso_value(&inv.due_date.to_string(), cx);
        });
        self.delivery_date.update(cx, |d, cx| {
            d.set_iso_value(&inv.delivery_date.to_string(), cx);
        });
        self.variable_symbol.update(cx, |t, cx| {
            t.set_value(&inv.variable_symbol, cx);
        });
        self.constant_symbol.update(cx, |t, cx| {
            t.set_value(&inv.constant_symbol, cx);
        });
        self.bank_account.update(cx, |t, cx| {
            t.set_value(&inv.bank_account, cx);
        });
        self.bank_code.update(cx, |t, cx| {
            t.set_value(&inv.bank_code, cx);
        });
        self.iban.update(cx, |t, cx| {
            t.set_value(&inv.iban, cx);
        });
        self.swift.update(cx, |t, cx| {
            t.set_value(&inv.swift, cx);
        });
        self.notes.update(cx, |t, cx| {
            t.set_value(&inv.notes, cx);
        });
        self.internal_notes.update(cx, |t, cx| {
            t.set_value(&inv.internal_notes, cx);
        });
        self.items_editor.update(cx, |editor, cx| {
            editor.set_items(&inv.items, cx);
        });
    }

    /// Validate and save the invoice (create or update).
    fn save(&mut self, cx: &mut Context<Self>) {
        if self.saving {
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

        // Read dates
        let issue_date_str = self.issue_date.read(cx).iso_value().to_string();
        let due_date_str = self.due_date.read(cx).iso_value().to_string();
        let delivery_date_str = self.delivery_date.read(cx).iso_value().to_string();

        let issue_date = NaiveDate::parse_from_str(&issue_date_str, "%Y-%m-%d")
            .unwrap_or_else(|_| Local::now().date_naive());
        let due_date = NaiveDate::parse_from_str(&due_date_str, "%Y-%m-%d")
            .unwrap_or_else(|_| Local::now().date_naive() + chrono::Duration::days(14));
        let delivery_date =
            NaiveDate::parse_from_str(&delivery_date_str, "%Y-%m-%d").unwrap_or(issue_date);

        // Read items
        let items = self.items_editor.read(cx).to_invoice_items(cx);
        if items.is_empty() {
            self.error = Some("Pridejte alespon jednu polozku".into());
            cx.notify();
            return;
        }

        // Read other fields
        let invoice_type_str = self
            .invoice_type_select
            .read(cx)
            .selected_value()
            .unwrap_or("regular")
            .to_string();
        let invoice_type = match invoice_type_str.as_str() {
            "proforma" => InvoiceType::Proforma,
            "credit_note" => InvoiceType::CreditNote,
            _ => InvoiceType::Regular,
        };
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
        let variable_symbol = self.variable_symbol.read(cx).value().to_string();
        let constant_symbol = self.constant_symbol.read(cx).value().to_string();
        let bank_account = self.bank_account.read(cx).value().to_string();
        let bank_code = self.bank_code.read(cx).value().to_string();
        let iban = self.iban.read(cx).value().to_string();
        let swift = self.swift.read(cx).value().to_string();
        let notes = self.notes.read(cx).value().to_string();
        let internal_notes = self.internal_notes.read(cx).value().to_string();

        let now = Local::now().naive_local();
        let mut invoice = Invoice {
            id: self.invoice_id.unwrap_or(0),
            sequence_id: 0,
            invoice_number: String::new(),
            invoice_type,
            status: InvoiceStatus::Draft,
            issue_date,
            due_date,
            delivery_date,
            variable_symbol,
            constant_symbol,
            customer_id,
            customer: None,
            currency_code: currency,
            exchange_rate: Amount::new(1, 0),
            payment_method,
            bank_account,
            bank_code,
            iban,
            swift,
            subtotal_amount: Amount::ZERO,
            vat_amount: Amount::ZERO,
            total_amount: Amount::ZERO,
            paid_amount: Amount::ZERO,
            notes,
            internal_notes,
            related_invoice_id: None,
            relation_type: RelationType::None,
            sent_at: None,
            paid_at: None,
            items,
            created_at: now,
            updated_at: now,
            deleted_at: None,
        };

        // Recalculate totals from items
        invoice.calculate_totals();

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.invoice_service.clone();
        let is_edit = self.is_edit;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    if is_edit {
                        service.update(&mut invoice)?;
                    } else {
                        service.create(&mut invoice)?;
                    }
                    Ok::<i64, zfaktury_domain::DomainError>(invoice.id)
                })
                .await;
            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(id) => cx.emit(NavigateEvent(Route::InvoiceDetail(id))),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }
}

impl Render for InvoiceFormView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let title = if self.is_edit {
            format!("Upravit fakturu #{}", self.invoice_id.unwrap_or_default())
        } else {
            "Nova faktura".to_string()
        };

        let mut outer = div()
            .id("invoice-form-scroll")
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
                .child(title),
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
                    .child("Nacitani faktury..."),
            );
        }

        // Card 1: Basic info
        outer = outer.child(render_card(
            "Zakladni udaje",
            div()
                .flex()
                .flex_col()
                .gap_4()
                // Row 1: customer, type, currency
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(div().flex_1().child(render_labeled_field(
                            "Zakaznik",
                            self.customer_select.clone(),
                        )))
                        .child(div().w(px(192.0)).child(render_labeled_field(
                            "Typ",
                            self.invoice_type_select.clone(),
                        )))
                        .child(
                            div()
                                .w(px(128.0))
                                .child(render_labeled_field("Mena", self.currency_select.clone())),
                        ),
                )
                // Row 2: dates
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(render_labeled_field(
                            "Datum vystaveni",
                            self.issue_date.clone(),
                        ))
                        .child(render_labeled_field(
                            "Datum splatnosti",
                            self.due_date.clone(),
                        ))
                        .child(render_labeled_field(
                            "Datum zdanitelneho plneni",
                            self.delivery_date.clone(),
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
                // Row 1: payment method, variable symbol, constant symbol
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(div().flex_1().child(render_labeled_field(
                            "Zpusob platby",
                            self.payment_method_select.clone(),
                        )))
                        .child(render_form_field(
                            "Variabilni symbol",
                            self.variable_symbol.clone(),
                        ))
                        .child(render_form_field(
                            "Konstantni symbol",
                            self.constant_symbol.clone(),
                        )),
                )
                // Row 2: bank account, bank code
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(render_form_field("Cislo uctu", self.bank_account.clone()))
                        .child(render_form_field("Kod banky", self.bank_code.clone())),
                )
                // Row 3: iban, swift
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
            div()
                .flex()
                .flex_col()
                .gap_4()
                .child(render_labeled_field("Poznamky", self.notes.clone()))
                .child(render_labeled_field(
                    "Interni poznamky",
                    self.internal_notes.clone(),
                )),
        ));

        // Button bar
        let cancel_btn = render_button(
            "cancel-btn",
            "Zrusit",
            ButtonVariant::Secondary,
            self.saving,
            false,
            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                cx.emit(NavigateEvent(Route::InvoiceList));
            }),
        );

        let save_btn = render_button(
            "save-btn",
            "Ulozit fakturu",
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
