use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::HealthInsuranceService;
use zfaktury_domain::{Amount, FilingStatus, FilingType, HealthInsuranceOverview};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::number_input::NumberInput;
use crate::components::select::{Select, SelectOption};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;

/// Health insurance overview creation form.
pub struct TaxHealthFormView {
    service: Arc<HealthInsuranceService>,
    saving: bool,
    error: Option<String>,
    year_input: Entity<NumberInput>,
    filing_type_select: Entity<Select>,
}

fn filing_type_options() -> Vec<SelectOption> {
    vec![
        SelectOption {
            value: "regular".to_string(),
            label: "Řádné".to_string(),
        },
        SelectOption {
            value: "corrective".to_string(),
            label: "Opravné".to_string(),
        },
        SelectOption {
            value: "supplementary".to_string(),
            label: "Dodatečné".to_string(),
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

impl TaxHealthFormView {
    pub fn new(service: Arc<HealthInsuranceService>, cx: &mut Context<Self>) -> Self {
        use chrono::Datelike;
        let current_year = chrono::Local::now().date_naive().year();
        let default_year = current_year - 1;

        let year_input = cx.new(|cx| {
            let mut input = NumberInput::new("year-input", "Rok", cx).integer_only();
            input.set_value(default_year.to_string(), cx);
            input
        });

        let filing_type_select = cx.new(|cx| {
            let mut s = Select::new("filing-type-select", "Typ podání", filing_type_options());
            s.set_selected_value("regular", cx);
            s
        });

        Self {
            service,
            saving: false,
            error: None,
            year_input,
            filing_type_select,
        }
    }

    fn save(&mut self, cx: &mut Context<Self>) {
        if self.saving {
            return;
        }

        let year_str = self.year_input.read(cx).value().to_string();
        let year: i32 = match year_str.parse() {
            Ok(y) => y,
            Err(_) => {
                self.error = Some("Zadejte platný rok".into());
                cx.notify();
                return;
            }
        };

        let filing_type_str = self
            .filing_type_select
            .read(cx)
            .selected_value()
            .unwrap_or("regular")
            .to_string();
        let filing_type = match filing_type_str.as_str() {
            "corrective" => FilingType::Corrective,
            "supplementary" => FilingType::Supplementary,
            _ => FilingType::Regular,
        };

        let now = chrono::Local::now().naive_local();
        let mut hi = HealthInsuranceOverview {
            id: 0,
            year,
            filing_type,
            total_revenue: Amount::ZERO,
            total_expenses: Amount::ZERO,
            tax_base: Amount::ZERO,
            assessment_base: Amount::ZERO,
            min_assessment_base: Amount::ZERO,
            final_assessment_base: Amount::ZERO,
            insurance_rate: 0,
            total_insurance: Amount::ZERO,
            prepayments: Amount::ZERO,
            difference: Amount::ZERO,
            new_monthly_prepay: Amount::ZERO,
            xml_data: Vec::new(),
            status: FilingStatus::Draft,
            filed_at: None,
            created_at: now,
            updated_at: now,
        };

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    service.create(&mut hi)?;
                    Ok::<i64, zfaktury_domain::DomainError>(hi.id)
                })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(id) => cx.emit(NavigateEvent(Route::TaxHealthDetail(id))),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }
}

impl EventEmitter<NavigateEvent> for TaxHealthFormView {}

impl Render for TaxHealthFormView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("tax-health-form-scroll")
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
                .child("Nový přehled ZP"),
        );

        // Error
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

        // Form card
        outer = outer.child(
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
                        .child("Přehled OSVČ pro zdravotní pojišťovnu"),
                )
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(
                            div()
                                .w(px(120.0))
                                .child(render_labeled_field("Rok", self.year_input.clone())),
                        )
                        .child(div().w(px(220.0)).child(render_labeled_field(
                            "Typ podání",
                            self.filing_type_select.clone(),
                        ))),
                ),
        );

        // Info
        outer = outer.child(
            div()
                .p_4()
                .bg(rgb(ZfColors::SURFACE))
                .rounded_md()
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
                .text_sm()
                .text_color(rgb(ZfColors::TEXT_MUTED))
                .child("Přehled bude automaticky vypočítán z vašich příjmů a výdajů za daný rok."),
        );

        // Button bar
        let cancel_btn = render_button(
            "cancel-btn",
            "Zrušit",
            ButtonVariant::Secondary,
            self.saving,
            false,
            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                cx.emit(NavigateEvent(Route::TaxOverview));
            }),
        );

        let save_btn = render_button(
            "save-btn",
            "Vytvořit",
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
