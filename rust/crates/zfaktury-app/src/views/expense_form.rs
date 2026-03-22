use std::sync::Arc;

use chrono::{Local, NaiveDate};
use gpui::*;
use zfaktury_core::service::category_svc::CategoryService;
use zfaktury_core::service::contact_svc::ContactService;
use zfaktury_core::service::expense_svc::ExpenseService;
use zfaktury_domain::{Amount, CURRENCY_CZK, Contact, ContactFilter, Expense, ExpenseCategory};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::date_input::DateInput;
use crate::components::expense_items_editor::{ExpenseItemsChanged, ExpenseItemsEditor};
use crate::components::number_input::NumberInput;
use crate::components::select::{Select, SelectOption};
use crate::components::text_area::TextArea;
use crate::components::text_input::{TextInput, render_form_field};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;

/// Expense creation/edit form view.
#[allow(dead_code)]
pub struct ExpenseFormView {
    expense_service: Arc<ExpenseService>,
    contact_service: Arc<ContactService>,
    category_service: Arc<CategoryService>,
    is_edit: bool,
    expense_id: Option<i64>,
    saving: bool,
    loading: bool,
    error: Option<String>,

    // Loaded reference data
    contacts: Vec<Contact>,
    categories: Vec<ExpenseCategory>,

    // Form fields
    expense_number: Entity<TextInput>,
    description: Entity<TextInput>,
    category_select: Entity<Select>,
    vendor_select: Entity<Select>,
    issue_date: Entity<DateInput>,
    amount_input: Entity<NumberInput>,
    currency_select: Entity<Select>,
    vat_rate_select: Entity<Select>,
    business_percent: Entity<NumberInput>,
    payment_method_select: Entity<Select>,
    notes: Entity<TextArea>,
    is_tax_deductible: bool,

    // Items editor
    items_editor: Entity<ExpenseItemsEditor>,
    use_items: bool,
}

impl EventEmitter<NavigateEvent> for ExpenseFormView {}

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

impl ExpenseFormView {
    /// Create a new expense form (create mode).
    pub fn new_create(
        expense_service: Arc<ExpenseService>,
        contact_service: Arc<ContactService>,
        category_service: Arc<CategoryService>,
        cx: &mut Context<Self>,
    ) -> Self {
        let today = Local::now().date_naive();

        let expense_number = cx.new(|cx| TextInput::new("expense-number", "Napr. N2024001", cx));
        let description =
            cx.new(|cx| TextInput::new("expense-description", "Popis nakladu...", cx));

        let category_select =
            cx.new(|_cx| Select::new("category-select", "Vyberte kategorii...", vec![]));

        let vendor_select =
            cx.new(|_cx| Select::new("vendor-select", "Vyberte dodavatele...", vec![]));

        let issue_date = cx.new(|cx| {
            let mut d = DateInput::new("issue-date", cx);
            d.set_iso_value(&format!("{}", today), cx);
            d
        });

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

        let notes = cx.new(|cx| TextArea::new("notes", "Volitelne poznamky...", cx));

        let items_editor = cx.new(ExpenseItemsEditor::new);
        cx.subscribe(
            &items_editor,
            |_this: &mut Self, _, _: &ExpenseItemsChanged, cx| {
                cx.notify();
            },
        )
        .detach();

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
            expense_service,
            contact_service,
            category_service,
            is_edit: false,
            expense_id: None,
            saving: false,
            loading: false,
            error: None,
            contacts: Vec::new(),
            categories: Vec::new(),
            expense_number,
            description,
            category_select,
            vendor_select,
            issue_date,
            amount_input,
            currency_select,
            vat_rate_select,
            business_percent,
            payment_method_select,
            notes,
            is_tax_deductible: true,
            items_editor,
            use_items: false,
        }
    }

    /// Create an expense form in edit mode (loads existing expense).
    pub fn new_edit(
        expense_service: Arc<ExpenseService>,
        contact_service: Arc<ContactService>,
        category_service: Arc<CategoryService>,
        id: i64,
        cx: &mut Context<Self>,
    ) -> Self {
        let expense_number = cx.new(|cx| TextInput::new("expense-number", "Napr. N2024001", cx));
        let description =
            cx.new(|cx| TextInput::new("expense-description", "Popis nakladu...", cx));

        let category_select =
            cx.new(|_cx| Select::new("category-select", "Vyberte kategorii...", vec![]));

        let vendor_select =
            cx.new(|_cx| Select::new("vendor-select", "Vyberte dodavatele...", vec![]));

        let issue_date = cx.new(|cx| DateInput::new("issue-date", cx));

        let amount_input = cx.new(|cx| NumberInput::new("amount-input", "0,00", cx));

        let currency_select =
            cx.new(|_cx| Select::new("currency-select", "Mena", currency_options()));

        let vat_rate_select =
            cx.new(|_cx| Select::new("vat-rate-select", "DPH sazba", vat_rate_options()));

        let business_percent =
            cx.new(|cx| NumberInput::new("biz-percent", "100", cx).integer_only());

        let payment_method_select = cx.new(|_cx| {
            Select::new(
                "payment-method-select",
                "Zpusob platby",
                payment_method_options(),
            )
        });

        let notes = cx.new(|cx| TextArea::new("notes", "Volitelne poznamky...", cx));

        let items_editor = cx.new(ExpenseItemsEditor::new);
        cx.subscribe(
            &items_editor,
            |_this: &mut Self, _, _: &ExpenseItemsChanged, cx| {
                cx.notify();
            },
        )
        .detach();

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

        // Load the expense data
        let exp_svc = expense_service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { exp_svc.get_by_id(id) })
                .await;
            this.update(cx, |this, cx| {
                match result {
                    Ok(exp) => this.populate_from_expense(&exp, cx),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                this.loading = false;
                cx.notify();
            })
            .ok();
        })
        .detach();

        Self {
            expense_service,
            contact_service,
            category_service,
            is_edit: true,
            expense_id: Some(id),
            saving: false,
            loading: true,
            error: None,
            contacts: Vec::new(),
            categories: Vec::new(),
            expense_number,
            description,
            category_select,
            vendor_select,
            issue_date,
            amount_input,
            currency_select,
            vat_rate_select,
            business_percent,
            payment_method_select,
            notes,
            is_tax_deductible: true,
            items_editor,
            use_items: false,
        }
    }

    /// Populate all form fields from an existing Expense.
    fn populate_from_expense(&mut self, exp: &Expense, cx: &mut Context<Self>) {
        self.expense_number.update(cx, |t, cx| {
            t.set_value(&exp.expense_number, cx);
        });
        self.description.update(cx, |t, cx| {
            t.set_value(&exp.description, cx);
        });
        self.category_select.update(cx, |s, cx| {
            s.set_selected_value(&exp.category, cx);
        });
        if let Some(vendor_id) = exp.vendor_id {
            self.vendor_select.update(cx, |s, cx| {
                s.set_selected_value(&vendor_id.to_string(), cx);
            });
        }
        self.issue_date.update(cx, |d, cx| {
            d.set_iso_value(&exp.issue_date.to_string(), cx);
        });
        self.amount_input.update(cx, |n, cx| {
            n.set_amount(exp.amount, cx);
        });
        self.currency_select.update(cx, |s, cx| {
            s.set_selected_value(&exp.currency_code, cx);
        });
        self.vat_rate_select.update(cx, |s, cx| {
            s.set_selected_value(&exp.vat_rate_percent.to_string(), cx);
        });
        self.business_percent.update(cx, |n, cx| {
            n.set_value(exp.business_percent.to_string(), cx);
        });
        self.payment_method_select.update(cx, |s, cx| {
            s.set_selected_value(&exp.payment_method, cx);
        });
        self.notes.update(cx, |t, cx| {
            t.set_value(&exp.notes, cx);
        });
        self.is_tax_deductible = exp.is_tax_deductible;
        if !exp.items.is_empty() {
            self.use_items = true;
            self.items_editor
                .update(cx, |editor, cx| editor.set_items(&exp.items, cx));
        }
    }

    /// Validate and save the expense (create or update).
    fn save(&mut self, cx: &mut Context<Self>) {
        if self.saving {
            return;
        }

        // Read and validate description
        let description = self.description.read(cx).value().to_string();
        if description.trim().is_empty() {
            self.error = Some("Zadejte popis nakladu".into());
            cx.notify();
            return;
        }

        // Read amount (skip zero check when using items -- service calculates from items)
        let amount_val = match self.amount_input.read(cx).to_amount() {
            Some(a) => a,
            None => {
                if !self.use_items {
                    self.error = Some("Neplatna castka".into());
                    cx.notify();
                    return;
                }
                Amount::ZERO
            }
        };
        if !self.use_items && amount_val == Amount::ZERO {
            self.error = Some("Zadejte castku".into());
            cx.notify();
            return;
        }

        // Read expense number
        let expense_number = self.expense_number.read(cx).value().to_string();

        // Read category
        let category_key = self
            .category_select
            .read(cx)
            .selected_value()
            .unwrap_or("")
            .to_string();

        // Read vendor (optional)
        let vendor_id_opt: Option<i64> = self
            .vendor_select
            .read(cx)
            .selected_value()
            .and_then(|v| v.parse().ok());

        // Read issue date
        let issue_date_str = self.issue_date.read(cx).iso_value().to_string();
        let issue_date = NaiveDate::parse_from_str(&issue_date_str, "%Y-%m-%d")
            .unwrap_or_else(|_| Local::now().date_naive());

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

        // Read notes
        let notes = self.notes.read(cx).value().to_string();

        // Read items if in items mode
        let items = if self.use_items {
            let editor_items = self.items_editor.read(cx).to_expense_items(cx);
            if editor_items.is_empty() {
                self.error = Some("Pridejte alespon jednu polozku".into());
                cx.notify();
                return;
            }
            editor_items
        } else {
            Vec::new()
        };

        let now = chrono::Local::now().naive_local();
        let mut expense = Expense {
            id: self.expense_id.unwrap_or(0),
            vendor_id: vendor_id_opt,
            vendor: None,
            expense_number,
            category: category_key,
            description,
            issue_date,
            amount: amount_val,
            currency_code: currency,
            exchange_rate: Amount::new(1, 0),
            vat_rate_percent: vat_rate,
            vat_amount: Amount::ZERO,
            is_tax_deductible: self.is_tax_deductible,
            business_percent: biz_pct,
            payment_method,
            document_path: String::new(),
            notes,
            tax_reviewed_at: None,
            items,
            created_at: now,
            updated_at: now,
            deleted_at: None,
        };

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.expense_service.clone();
        let is_edit = self.is_edit;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    if is_edit {
                        service.update(&mut expense)?;
                    } else {
                        service.create(&mut expense)?;
                    }
                    Ok::<i64, zfaktury_domain::DomainError>(expense.id)
                })
                .await;
            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(id) => cx.emit(NavigateEvent(Route::ExpenseDetail(id))),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }
}

impl Render for ExpenseFormView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let title = if self.is_edit {
            format!("Upravit naklad #{}", self.expense_id.unwrap_or_default())
        } else {
            "Novy naklad".to_string()
        };

        let mut outer = div()
            .id("expense-form-scroll")
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
                    .child("Nacitani nakladu..."),
            );
        }

        // Card 1: Basic info
        outer = outer.child(render_card(
            "Zakladni udaje",
            div()
                .flex()
                .flex_col()
                .gap_4()
                // Row 1: expense_number (w-48), description (flex-1)
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(div().w(px(192.0)).child(render_form_field(
                            "Cislo dokladu",
                            self.expense_number.clone(),
                        )))
                        .child(
                            div()
                                .flex_1()
                                .child(render_form_field("Popis", self.description.clone())),
                        ),
                )
                // Row 2: category, issue_date, vendor
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(div().flex_1().child(render_labeled_field(
                            "Kategorie",
                            self.category_select.clone(),
                        )))
                        .child(render_labeled_field("Datum", self.issue_date.clone()))
                        .child(div().flex_1().child(render_labeled_field(
                            "Dodavatel",
                            self.vendor_select.clone(),
                        ))),
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
                // Row 1: amount (flex-1), currency (w-32), vat_rate (w-32)
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
                // Row 2: business_percent (w-32), payment_method (flex-1), tax deductible toggle
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

        // Items toggle button
        let toggle_label = if self.use_items {
            "Prepnout na jednoduchou castku"
        } else {
            "Pridat polozky"
        };
        outer = outer.child(
            div().flex().child(
                div()
                    .id("toggle-items-mode")
                    .cursor_pointer()
                    .px_3()
                    .py_2()
                    .bg(rgb(ZfColors::SURFACE))
                    .border_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .rounded_md()
                    .text_sm()
                    .text_color(rgb(ZfColors::ACCENT))
                    .on_click(cx.listener(|this, _ev: &ClickEvent, _w, cx| {
                        this.use_items = !this.use_items;
                        cx.notify();
                    }))
                    .child(toggle_label),
            ),
        );

        // Items editor (shown only in items mode)
        if self.use_items {
            outer = outer.child(render_card(
                "Polozky nakladu",
                div().child(self.items_editor.clone()),
            ));
        }

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
                cx.emit(NavigateEvent(Route::ExpenseList));
            }),
        );

        let save_btn = render_button(
            "save-btn",
            "Ulozit naklad",
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
