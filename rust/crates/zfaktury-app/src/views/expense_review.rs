use std::sync::Arc;

use chrono::{Local, NaiveDate};
use gpui::*;
use zfaktury_core::service::expense_svc::ExpenseService;
use zfaktury_domain::{Amount, CURRENCY_CZK, Expense};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::confirm_dialog::{ConfirmDialog, ConfirmDialogResult};
use crate::components::date_input::DateInput;
use crate::components::number_input::NumberInput;
use crate::components::select::{Select, SelectOption};
use crate::components::text_input::{TextInput, render_form_field};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;

/// View for reviewing and editing expense data before final save.
/// Fields can be pre-populated from OCR results or left empty for manual entry.
pub struct ExpenseReviewView {
    expense_service: Arc<ExpenseService>,

    // Data
    expense_id: i64,
    loading: bool,

    // Editable form fields
    description: Entity<TextInput>,
    invoice_number: Entity<TextInput>,
    issue_date: Entity<DateInput>,
    total_amount: Entity<NumberInput>,
    vat_amount: Entity<NumberInput>,
    vat_rate: Entity<Select>,
    currency: Entity<Select>,

    // State
    saving: bool,
    error: Option<String>,
    confirm_dialog: Option<Entity<ConfirmDialog>>,
}

impl EventEmitter<NavigateEvent> for ExpenseReviewView {}

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

impl ExpenseReviewView {
    /// Create a new review view for the given expense ID.
    /// Loads expense data and pre-populates the form fields.
    pub fn new(
        expense_service: Arc<ExpenseService>,
        expense_id: i64,
        cx: &mut Context<Self>,
    ) -> Self {
        let description = cx.new(|cx| TextInput::new("review-description", "Popis nakladu...", cx));
        let invoice_number =
            cx.new(|cx| TextInput::new("review-invoice-number", "Cislo dokladu...", cx));
        let issue_date = cx.new(|cx| {
            let mut d = DateInput::new("review-issue-date", cx);
            let today = Local::now().date_naive();
            d.set_iso_value(&format!("{}", today), cx);
            d
        });
        let total_amount = cx.new(|cx| NumberInput::new("review-total-amount", "0,00", cx));
        let vat_amount = cx.new(|cx| NumberInput::new("review-vat-amount", "0,00", cx));
        let vat_rate = cx.new(|cx| {
            let mut s = Select::new("review-vat-rate", "DPH sazba", vat_rate_options());
            s.set_selected_value("21", cx);
            s
        });
        let currency = cx.new(|cx| {
            let mut s = Select::new("review-currency", "Mena", currency_options());
            s.set_selected_value(CURRENCY_CZK, cx);
            s
        });

        // Load existing expense data to pre-populate form
        let svc = expense_service.clone();
        let id = expense_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { svc.get_by_id(id) })
                .await;
            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(expense) => this.populate_from_expense(&expense, cx),
                    Err(e) => this.error = Some(format!("Chyba pri nacitani nakladu: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();

        Self {
            expense_service,
            expense_id,
            loading: true,
            description,
            invoice_number,
            issue_date,
            total_amount,
            vat_amount,
            vat_rate,
            currency,
            saving: false,
            error: None,
            confirm_dialog: None,
        }
    }

    /// Populate form fields from an existing expense.
    fn populate_from_expense(&mut self, exp: &Expense, cx: &mut Context<Self>) {
        self.description.update(cx, |t, cx| {
            t.set_value(&exp.description, cx);
        });
        self.invoice_number.update(cx, |t, cx| {
            t.set_value(&exp.expense_number, cx);
        });
        self.issue_date.update(cx, |d, cx| {
            d.set_iso_value(&exp.issue_date.to_string(), cx);
        });
        self.total_amount.update(cx, |n, cx| {
            n.set_amount(exp.amount, cx);
        });
        self.vat_amount.update(cx, |n, cx| {
            n.set_amount(exp.vat_amount, cx);
        });
        self.vat_rate.update(cx, |s, cx| {
            s.set_selected_value(&exp.vat_rate_percent.to_string(), cx);
        });
        self.currency.update(cx, |s, cx| {
            s.set_selected_value(&exp.currency_code, cx);
        });
    }

    /// Validate and save the reviewed expense data.
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

        // Read amount
        let amount_val = match self.total_amount.read(cx).to_amount() {
            Some(a) => a,
            None => {
                self.error = Some("Neplatna castka".into());
                cx.notify();
                return;
            }
        };

        // Read VAT amount
        let vat_amount_val = self.vat_amount.read(cx).to_amount().unwrap_or(Amount::ZERO);

        // Read invoice number
        let expense_number = self.invoice_number.read(cx).value().to_string();

        // Read issue date
        let issue_date_str = self.issue_date.read(cx).iso_value().to_string();
        let issue_date = NaiveDate::parse_from_str(&issue_date_str, "%Y-%m-%d")
            .unwrap_or_else(|_| Local::now().date_naive());

        // Read currency
        let currency = self
            .currency
            .read(cx)
            .selected_value()
            .unwrap_or(CURRENCY_CZK)
            .to_string();

        // Read VAT rate
        let vat_rate: i32 = self
            .vat_rate
            .read(cx)
            .selected_value()
            .and_then(|v| v.parse().ok())
            .unwrap_or(21);

        let now = chrono::Local::now().naive_local();
        let expense_id = self.expense_id;

        let mut expense = Expense {
            id: expense_id,
            vendor_id: None,
            vendor: None,
            expense_number,
            category: String::new(),
            description,
            issue_date,
            amount: amount_val,
            currency_code: currency,
            exchange_rate: Amount::new(1, 0),
            vat_rate_percent: vat_rate,
            vat_amount: vat_amount_val,
            is_tax_deductible: false,
            business_percent: 100,
            payment_method: "bank_transfer".to_string(),
            document_path: String::new(),
            notes: String::new(),
            tax_reviewed_at: None,
            items: Vec::new(),
            created_at: now,
            updated_at: now,
            deleted_at: None,
        };

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.expense_service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    service.update(&mut expense)?;
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

    /// Show delete confirmation dialog.
    fn show_discard_dialog(&mut self, cx: &mut Context<Self>) {
        let dialog = cx.new(|_cx| {
            ConfirmDialog::new(
                "Zahodit naklad?",
                "Naklad a pripadne nahrane dokumenty budou smazany.",
                "Zahodit",
            )
        });
        cx.subscribe(
            &dialog,
            |this: &mut Self, _, result: &ConfirmDialogResult, cx| match result {
                ConfirmDialogResult::Confirmed => {
                    let service = this.expense_service.clone();
                    let id = this.expense_id;
                    this.confirm_dialog = None;
                    this.saving = true;
                    this.error = None;
                    cx.notify();
                    cx.spawn(async move |this, cx| {
                        let result = cx
                            .background_executor()
                            .spawn(async move { service.delete(id) })
                            .await;
                        this.update(cx, |this, cx| {
                            this.saving = false;
                            match result {
                                Ok(()) => cx.emit(NavigateEvent(Route::ExpenseList)),
                                Err(e) => {
                                    this.error = Some(format!("{e}"));
                                    cx.notify();
                                }
                            }
                        })
                        .ok();
                    })
                    .detach();
                }
                ConfirmDialogResult::Cancelled => {
                    this.confirm_dialog = None;
                    cx.notify();
                }
            },
        )
        .detach();
        self.confirm_dialog = Some(dialog);
        cx.notify();
    }
}

impl Render for ExpenseReviewView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("expense-review-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll()
            .relative();

        // Header
        outer = outer.child(
            div()
                .text_xl()
                .font_weight(FontWeight::SEMIBOLD)
                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                .child("Kontrola udaju nakladu"),
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
                // Row 1: invoice number (w-48), description (flex-1)
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(div().w(px(192.0)).child(render_form_field(
                            "Cislo dokladu",
                            self.invoice_number.clone(),
                        )))
                        .child(
                            div()
                                .flex_1()
                                .child(render_form_field("Popis", self.description.clone())),
                        ),
                )
                // Row 2: issue date
                .child(div().flex().gap_4().child(render_labeled_field(
                    "Datum vystaveni",
                    self.issue_date.clone(),
                ))),
        ));

        // Card 2: Amount and VAT
        outer = outer.child(render_card(
            "Castka a DPH",
            div()
                .flex()
                .flex_col()
                .gap_4()
                // Row 1: total amount (flex-1), currency (w-32), vat_rate (w-32)
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(div().flex_1().child(render_labeled_field(
                            "Celkova castka",
                            self.total_amount.clone(),
                        )))
                        .child(
                            div()
                                .w(px(128.0))
                                .child(render_labeled_field("Mena", self.currency.clone())),
                        )
                        .child(
                            div()
                                .w(px(128.0))
                                .child(render_labeled_field("DPH sazba", self.vat_rate.clone())),
                        ),
                )
                // Row 2: vat amount
                .child(
                    div().flex().gap_4().child(
                        div()
                            .w(px(192.0))
                            .child(render_labeled_field("DPH castka", self.vat_amount.clone())),
                    ),
                ),
        ));

        // Action buttons
        let discard_btn = render_button(
            "discard-btn",
            "Zahodit",
            ButtonVariant::Danger,
            self.saving,
            false,
            cx.listener(|this, _event: &ClickEvent, _window, cx| {
                this.show_discard_dialog(cx);
            }),
        );

        let edit_btn = render_button(
            "edit-detail-btn",
            "Upravit podrobne",
            ButtonVariant::Secondary,
            self.saving,
            false,
            {
                let expense_id = self.expense_id;
                cx.listener(move |_this, _event: &ClickEvent, _window, cx| {
                    cx.emit(NavigateEvent(Route::ExpenseEdit(expense_id)));
                })
            },
        );

        let save_btn = render_button(
            "save-btn",
            "Ulozit",
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
                .child(discard_btn)
                .child(div().flex().gap_2().child(edit_btn).child(save_btn)),
        );

        // Confirm dialog overlay
        if let Some(ref dialog) = self.confirm_dialog {
            outer = outer.child(dialog.clone());
        }

        outer
    }
}
