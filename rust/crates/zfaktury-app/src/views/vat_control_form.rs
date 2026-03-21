use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::VATControlStatementService;
use zfaktury_domain::{FilingStatus, FilingType, TaxPeriod, VATControlStatement};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::number_input::NumberInput;
use crate::components::select::{Select, SelectOption};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;

/// VAT control statement creation form.
pub struct VatControlFormView {
    service: Arc<VATControlStatementService>,
    saving: bool,
    error: Option<String>,
    year_input: Entity<NumberInput>,
    month_select: Entity<Select>,
    filing_type_select: Entity<Select>,
}

fn month_options() -> Vec<SelectOption> {
    let labels = [
        "Leden", "Unor", "Brezen", "Duben", "Kveten", "Cerven", "Cervenec", "Srpen", "Zari",
        "Rijen", "Listopad", "Prosinec",
    ];
    labels
        .iter()
        .enumerate()
        .map(|(i, label)| SelectOption {
            value: (i + 1).to_string(),
            label: label.to_string(),
        })
        .collect()
}

fn filing_type_options() -> Vec<SelectOption> {
    vec![
        SelectOption {
            value: "regular".to_string(),
            label: "Radne".to_string(),
        },
        SelectOption {
            value: "corrective".to_string(),
            label: "Opravne".to_string(),
        },
        SelectOption {
            value: "supplementary".to_string(),
            label: "Dodatecne".to_string(),
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

impl VatControlFormView {
    pub fn new(service: Arc<VATControlStatementService>, cx: &mut Context<Self>) -> Self {
        use chrono::Datelike;
        let current_year = chrono::Local::now().date_naive().year();

        let year_input = cx.new(|cx| {
            let mut input = NumberInput::new("year-input", "Rok", cx).integer_only();
            input.set_value(current_year.to_string(), cx);
            input
        });

        let month_select = cx.new(|cx| {
            let current_month = chrono::Local::now().date_naive().month();
            let mut s = Select::new("month-select", "Mesic", month_options());
            s.set_selected_value(&current_month.to_string(), cx);
            s
        });

        let filing_type_select = cx.new(|cx| {
            let mut s = Select::new("filing-type-select", "Typ podani", filing_type_options());
            s.set_selected_value("regular", cx);
            s
        });

        Self {
            service,
            saving: false,
            error: None,
            year_input,
            month_select,
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
                self.error = Some("Zadejte platny rok".into());
                cx.notify();
                return;
            }
        };

        let month: i32 = self
            .month_select
            .read(cx)
            .selected_value()
            .and_then(|v| v.parse().ok())
            .unwrap_or(1);

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
        let mut cs = VATControlStatement {
            id: 0,
            period: TaxPeriod {
                year,
                month,
                quarter: 0,
            },
            filing_type,
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
                    service.create(&mut cs)?;
                    Ok::<i64, zfaktury_domain::DomainError>(cs.id)
                })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(id) => cx.emit(NavigateEvent(Route::VATControlDetail(id))),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }
}

impl EventEmitter<NavigateEvent> for VatControlFormView {}

impl Render for VatControlFormView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("vat-control-form-scroll")
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
                .child("Nove kontrolni hlaseni"),
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
                        .child("Obdobi"),
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
                        .child(
                            div()
                                .w(px(220.0))
                                .child(render_labeled_field("Mesic", self.month_select.clone())),
                        )
                        .child(div().w(px(220.0)).child(render_labeled_field(
                            "Typ podani",
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
                .child("Kontrolni hlaseni se podava mesicne. Radky budou vygenerovany z faktur a nakladu."),
        );

        // Button bar
        let cancel_btn = render_button(
            "cancel-btn",
            "Zrusit",
            ButtonVariant::Secondary,
            self.saving,
            false,
            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                cx.emit(NavigateEvent(Route::VATOverview));
            }),
        );

        let save_btn = render_button(
            "save-btn",
            "Vytvorit",
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
