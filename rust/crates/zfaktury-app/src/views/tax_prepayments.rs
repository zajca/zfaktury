use std::sync::Arc;

use chrono::Datelike;
use gpui::*;
use zfaktury_core::service::TaxYearSettingsService;
use zfaktury_domain::{Amount, TaxPrepayment};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::number_input::NumberInput;
use crate::components::select::{Select, SelectOption};
use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;
use crate::util::format::format_amount;

const CZECH_MONTHS: [&str; 12] = [
    "Leden", "Unor", "Brezen", "Duben", "Kveten", "Cerven", "Cervenec", "Srpen", "Zari", "Rijen",
    "Listopad", "Prosinec",
];

fn flat_rate_options() -> Vec<SelectOption> {
    vec![
        SelectOption {
            value: "0".into(),
            label: "Skutecne vydaje".into(),
        },
        SelectOption {
            value: "30".into(),
            label: "30%".into(),
        },
        SelectOption {
            value: "40".into(),
            label: "40%".into(),
        },
        SelectOption {
            value: "60".into(),
            label: "60%".into(),
        },
        SelectOption {
            value: "80".into(),
            label: "80%".into(),
        },
    ]
}

/// Helper struct holding three NumberInput entities for a single month row.
struct MonthRow {
    month: i32,
    tax_input: Entity<NumberInput>,
    social_input: Entity<NumberInput>,
    health_input: Entity<NumberInput>,
}

/// Tax prepayments view showing monthly prepayment schedule with edit support.
pub struct TaxPrepaymentsView {
    service: Arc<TaxYearSettingsService>,
    year: i32,
    loading: bool,
    saving: bool,
    editing: bool,
    error: Option<String>,
    success: Option<String>,

    // Read-only data (always loaded)
    prepayments: Vec<TaxPrepayment>,
    flat_rate_percent: i32,

    // Edit mode entities (created on enter edit, cleared on exit)
    flat_rate_select: Option<Entity<Select>>,
    month_rows: Vec<MonthRow>,
}

impl TaxPrepaymentsView {
    pub fn new(service: Arc<TaxYearSettingsService>, cx: &mut Context<Self>) -> Self {
        let year = chrono::Local::now().date_naive().year();
        let mut view = Self {
            service,
            year,
            loading: true,
            saving: false,
            editing: false,
            error: None,
            success: None,
            prepayments: Vec::new(),
            flat_rate_percent: 0,
            flat_rate_select: None,
            month_rows: Vec::new(),
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let year = self.year;

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    let prepayments = service.get_prepayments(year)?;
                    let settings = service.get_by_year(year)?;
                    Ok::<_, zfaktury_domain::DomainError>((prepayments, settings.flat_rate_percent))
                })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok((prepayments, flat_rate)) => {
                        this.prepayments = prepayments;
                        this.flat_rate_percent = flat_rate;
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani zaloh: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn change_year(&mut self, delta: i32, cx: &mut Context<Self>) {
        // Exit edit mode if active
        if self.editing {
            self.cancel_editing(cx);
        }

        self.year += delta;
        self.loading = true;
        self.error = None;
        self.success = None;
        cx.notify();
        self.load_data(cx);
    }

    fn start_editing(&mut self, cx: &mut Context<Self>) {
        self.editing = true;
        self.error = None;
        self.success = None;

        // Create flat rate select
        let flat_rate_percent = self.flat_rate_percent;
        let flat_rate_select = cx.new(|cx| {
            let mut s = Select::new("flat-rate-select", "Pausalni vydaje", flat_rate_options());
            s.set_selected_value(&flat_rate_percent.to_string(), cx);
            s
        });
        self.flat_rate_select = Some(flat_rate_select);

        // Create 12 month rows
        self.month_rows.clear();
        for i in 0..12 {
            let month = (i + 1) as i32;
            let prepayment = self.prepayments.get(i);

            let tax_input = cx.new(|cx| {
                let mut n =
                    NumberInput::new(SharedString::from(format!("tax-{month}")), "0,00", cx);
                if let Some(p) = prepayment {
                    n.set_amount(p.tax_amount, cx);
                }
                n
            });

            let social_input = cx.new(|cx| {
                let mut n =
                    NumberInput::new(SharedString::from(format!("social-{month}")), "0,00", cx);
                if let Some(p) = prepayment {
                    n.set_amount(p.social_amount, cx);
                }
                n
            });

            let health_input = cx.new(|cx| {
                let mut n =
                    NumberInput::new(SharedString::from(format!("health-{month}")), "0,00", cx);
                if let Some(p) = prepayment {
                    n.set_amount(p.health_amount, cx);
                }
                n
            });

            self.month_rows.push(MonthRow {
                month,
                tax_input,
                social_input,
                health_input,
            });
        }

        cx.notify();
    }

    fn cancel_editing(&mut self, cx: &mut Context<Self>) {
        self.editing = false;
        self.error = None;
        self.month_rows.clear();
        self.flat_rate_select = None;
        cx.notify();
    }

    fn fill_from_first_month(&mut self, cx: &mut Context<Self>) {
        if self.month_rows.is_empty() {
            return;
        }

        // Read values from month 1
        let tax_val = self.month_rows[0].tax_input.read(cx).value().to_string();
        let social_val = self.month_rows[0].social_input.read(cx).value().to_string();
        let health_val = self.month_rows[0].health_input.read(cx).value().to_string();

        // Apply to months 2-12
        for row in self.month_rows.iter().skip(1) {
            row.tax_input.update(cx, |n, cx| n.set_value(&tax_val, cx));
            row.social_input
                .update(cx, |n, cx| n.set_value(&social_val, cx));
            row.health_input
                .update(cx, |n, cx| n.set_value(&health_val, cx));
        }

        cx.notify();
    }

    fn save(&mut self, cx: &mut Context<Self>) {
        if self.saving {
            return;
        }

        // Read flat rate
        let flat_rate_percent: i32 = self
            .flat_rate_select
            .as_ref()
            .and_then(|s| s.read(cx).selected_value().map(|v| v.to_string()))
            .and_then(|v| v.parse().ok())
            .unwrap_or(0);

        // Collect all 12 prepayments
        let mut prepayments = Vec::with_capacity(12);
        for row in &self.month_rows {
            let tax = row.tax_input.read(cx).to_amount();
            let social = row.social_input.read(cx).to_amount();
            let health = row.health_input.read(cx).to_amount();

            // Validate: all must parse successfully
            if tax.is_none() || social.is_none() || health.is_none() {
                self.error = Some(format!(
                    "Neplatna castka v mesici {}",
                    CZECH_MONTHS[(row.month - 1) as usize]
                ));
                cx.notify();
                return;
            }

            prepayments.push(TaxPrepayment {
                year: self.year,
                month: row.month,
                tax_amount: tax.unwrap(),
                social_amount: social.unwrap(),
                health_amount: health.unwrap(),
            });
        }

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.service.clone();
        let year = self.year;
        let prepayments_for_update = prepayments.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.save(year, flat_rate_percent, &prepayments) })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(()) => {
                        // Update local state from saved data
                        this.prepayments = prepayments_for_update;
                        this.flat_rate_percent = flat_rate_percent;

                        // Exit edit mode
                        this.editing = false;
                        this.month_rows.clear();
                        this.flat_rate_select = None;

                        // Show success feedback
                        this.success = Some("Zalohy ulozeny".into());
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba pri ukladani zaloh: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    /// Compute running totals from NumberInput values in edit mode.
    fn compute_edit_totals(&self, cx: &App) -> (Amount, Amount, Amount) {
        let mut total_tax = Amount::ZERO;
        let mut total_social = Amount::ZERO;
        let mut total_health = Amount::ZERO;

        for row in &self.month_rows {
            total_tax += row.tax_input.read(cx).to_amount().unwrap_or(Amount::ZERO);
            total_social += row
                .social_input
                .read(cx)
                .to_amount()
                .unwrap_or(Amount::ZERO);
            total_health += row
                .health_input
                .read(cx)
                .to_amount()
                .unwrap_or(Amount::ZERO);
        }

        (total_tax, total_social, total_health)
    }

    /// Render the flat rate display label for view mode.
    fn flat_rate_label(&self) -> String {
        if self.flat_rate_percent == 0 {
            "Pausalni vydaje: Skutecne vydaje".to_string()
        } else {
            format!("Pausalni vydaje: {}%", self.flat_rate_percent)
        }
    }

    /// Render the table in read-only view mode.
    fn render_view_mode(&self, cx: &mut Context<Self>) -> Div {
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
                .child(div().w(px(120.0)).child("Mesic"))
                .child(div().flex_1().text_right().child("Dan z prijmu"))
                .child(div().flex_1().text_right().child("Socialni pojisteni"))
                .child(div().flex_1().text_right().child("Zdravotni pojisteni"))
                .child(div().flex_1().text_right().child("Celkem")),
        );

        let mut total_tax = Amount::ZERO;
        let mut total_social = Amount::ZERO;
        let mut total_health = Amount::ZERO;
        let mut total_all = Amount::ZERO;

        for (i, month) in CZECH_MONTHS.iter().enumerate() {
            let prepayment = self.prepayments.get(i);
            let tax_amount = prepayment.map_or(Amount::ZERO, |p| p.tax_amount);
            let social_amount = prepayment.map_or(Amount::ZERO, |p| p.social_amount);
            let health_amount = prepayment.map_or(Amount::ZERO, |p| p.health_amount);
            let row_total = tax_amount + social_amount + health_amount;

            total_tax += tax_amount;
            total_social += social_amount;
            total_health += health_amount;
            total_all += row_total;

            let has_data = tax_amount != Amount::ZERO
                || social_amount != Amount::ZERO
                || health_amount != Amount::ZERO;

            table = table.child(
                div()
                    .flex()
                    .items_center()
                    .px_4()
                    .py_2()
                    .text_sm()
                    .border_t_1()
                    .border_color(rgb(ZfColors::BORDER_SUBTLE))
                    .child(
                        div()
                            .w(px(120.0))
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child(format!("{}. {}", i + 1, month)),
                    )
                    .child(
                        div()
                            .flex_1()
                            .text_right()
                            .text_color(if has_data {
                                rgb(ZfColors::TEXT_PRIMARY)
                            } else {
                                rgb(ZfColors::TEXT_MUTED)
                            })
                            .child(if has_data {
                                format_amount(tax_amount)
                            } else {
                                "-".to_string()
                            }),
                    )
                    .child(
                        div()
                            .flex_1()
                            .text_right()
                            .text_color(if has_data {
                                rgb(ZfColors::TEXT_PRIMARY)
                            } else {
                                rgb(ZfColors::TEXT_MUTED)
                            })
                            .child(if has_data {
                                format_amount(social_amount)
                            } else {
                                "-".to_string()
                            }),
                    )
                    .child(
                        div()
                            .flex_1()
                            .text_right()
                            .text_color(if has_data {
                                rgb(ZfColors::TEXT_PRIMARY)
                            } else {
                                rgb(ZfColors::TEXT_MUTED)
                            })
                            .child(if has_data {
                                format_amount(health_amount)
                            } else {
                                "-".to_string()
                            }),
                    )
                    .child(
                        div()
                            .flex_1()
                            .text_right()
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(if has_data {
                                rgb(ZfColors::TEXT_PRIMARY)
                            } else {
                                rgb(ZfColors::TEXT_MUTED)
                            })
                            .child(if has_data {
                                format_amount(row_total)
                            } else {
                                "-".to_string()
                            }),
                    ),
            );
        }

        let has_totals = total_all != Amount::ZERO;

        // Totals row
        table = table.child(Self::render_totals_row(
            total_tax,
            total_social,
            total_health,
            total_all,
            has_totals,
        ));

        // Wrap in container with edit button
        div()
            .flex()
            .flex_col()
            .gap_4()
            .child(div().flex().justify_end().child(render_button(
                "btn-edit",
                "Upravit",
                ButtonVariant::Primary,
                false,
                false,
                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.start_editing(cx);
                }),
            )))
            .child(table)
    }

    /// Render the table in edit mode with NumberInput fields.
    fn render_edit_mode(&self, cx: &mut Context<Self>) -> Div {
        let mut edit_area = div().flex().flex_col().gap_4();

        // Flat rate selector + action buttons row
        let mut controls = div().flex().items_center().justify_between();

        // Left side: flat rate select
        if let Some(ref flat_rate_select) = self.flat_rate_select {
            controls = controls.child(
                div()
                    .flex()
                    .items_center()
                    .gap_3()
                    .child(
                        div()
                            .text_sm()
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .child("Pausalni vydaje:"),
                    )
                    .child(div().w(px(200.0)).child(flat_rate_select.clone())),
            );
        }

        // Right side: cancel + save
        controls = controls.child(
            div()
                .flex()
                .gap_2()
                .child(render_button(
                    "btn-cancel",
                    "Zrusit",
                    ButtonVariant::Secondary,
                    self.saving,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.cancel_editing(cx);
                    }),
                ))
                .child(render_button(
                    "btn-save",
                    "Ulozit",
                    ButtonVariant::Primary,
                    false,
                    self.saving,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.save(cx);
                    }),
                )),
        );

        edit_area = edit_area.child(controls);

        // Quick-fill button
        edit_area = edit_area.child(div().flex().child(render_button(
            "btn-fill",
            "Vyplnit z prvniho mesice",
            ButtonVariant::Secondary,
            self.saving,
            false,
            cx.listener(|this, _event: &ClickEvent, _window, cx| {
                this.fill_from_first_month(cx);
            }),
        )));

        // Table with inputs
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
                .child(div().w(px(120.0)).child("Mesic"))
                .child(div().flex_1().text_right().child("Dan z prijmu"))
                .child(div().flex_1().text_right().child("Socialni pojisteni"))
                .child(div().flex_1().text_right().child("Zdravotni pojisteni"))
                .child(div().flex_1().text_right().child("Celkem")),
        );

        // Compute totals from live input values
        let (total_tax, total_social, total_health) = self.compute_edit_totals(cx);
        let total_all = total_tax + total_social + total_health;

        // Month rows with NumberInputs
        for row in &self.month_rows {
            let month_idx = (row.month - 1) as usize;
            let month_name = CZECH_MONTHS[month_idx];

            // Compute row total from current input values
            let row_tax = row.tax_input.read(cx).to_amount().unwrap_or(Amount::ZERO);
            let row_social = row
                .social_input
                .read(cx)
                .to_amount()
                .unwrap_or(Amount::ZERO);
            let row_health = row
                .health_input
                .read(cx)
                .to_amount()
                .unwrap_or(Amount::ZERO);
            let row_total = row_tax + row_social + row_health;
            let has_data = row_total != Amount::ZERO;

            table = table.child(
                div()
                    .flex()
                    .items_center()
                    .px_4()
                    .py_2()
                    .text_sm()
                    .border_t_1()
                    .border_color(rgb(ZfColors::BORDER_SUBTLE))
                    .child(
                        div()
                            .w(px(120.0))
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child(format!("{}. {}", row.month, month_name)),
                    )
                    .child(div().flex_1().px_1().child(row.tax_input.clone()))
                    .child(div().flex_1().px_1().child(row.social_input.clone()))
                    .child(div().flex_1().px_1().child(row.health_input.clone()))
                    .child(
                        div()
                            .flex_1()
                            .text_right()
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(if has_data {
                                rgb(ZfColors::TEXT_PRIMARY)
                            } else {
                                rgb(ZfColors::TEXT_MUTED)
                            })
                            .child(if has_data {
                                format_amount(row_total)
                            } else {
                                "-".to_string()
                            }),
                    ),
            );
        }

        let has_totals = total_all != Amount::ZERO;

        // Totals row
        table = table.child(Self::render_totals_row(
            total_tax,
            total_social,
            total_health,
            total_all,
            has_totals,
        ));

        edit_area.child(table)
    }

    /// Render the totals footer row (shared between view and edit modes).
    fn render_totals_row(
        total_tax: Amount,
        total_social: Amount,
        total_health: Amount,
        total_all: Amount,
        has_totals: bool,
    ) -> Div {
        div()
            .flex()
            .items_center()
            .px_4()
            .py_3()
            .text_sm()
            .font_weight(FontWeight::SEMIBOLD)
            .border_t_1()
            .border_color(rgb(ZfColors::BORDER))
            .bg(rgb(ZfColors::SURFACE_HOVER))
            .child(
                div()
                    .w(px(120.0))
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child("Celkem"),
            )
            .child(
                div()
                    .flex_1()
                    .text_right()
                    .text_color(if has_totals {
                        rgb(ZfColors::TEXT_PRIMARY)
                    } else {
                        rgb(ZfColors::TEXT_MUTED)
                    })
                    .child(if has_totals {
                        format_amount(total_tax)
                    } else {
                        "-".to_string()
                    }),
            )
            .child(
                div()
                    .flex_1()
                    .text_right()
                    .text_color(if has_totals {
                        rgb(ZfColors::TEXT_PRIMARY)
                    } else {
                        rgb(ZfColors::TEXT_MUTED)
                    })
                    .child(if has_totals {
                        format_amount(total_social)
                    } else {
                        "-".to_string()
                    }),
            )
            .child(
                div()
                    .flex_1()
                    .text_right()
                    .text_color(if has_totals {
                        rgb(ZfColors::TEXT_PRIMARY)
                    } else {
                        rgb(ZfColors::TEXT_MUTED)
                    })
                    .child(if has_totals {
                        format_amount(total_health)
                    } else {
                        "-".to_string()
                    }),
            )
            .child(
                div()
                    .flex_1()
                    .text_right()
                    .text_color(if has_totals {
                        rgb(ZfColors::ACCENT)
                    } else {
                        rgb(ZfColors::TEXT_MUTED)
                    })
                    .child(if has_totals {
                        format_amount(total_all)
                    } else {
                        "-".to_string()
                    }),
            )
    }
}

impl EventEmitter<NavigateEvent> for TaxPrepaymentsView {}

impl Render for TaxPrepaymentsView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("tax-prepayments-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll();

        // Header with year selector and flat rate label (in view mode)
        let mut header = div()
            .flex()
            .items_center()
            .gap_3()
            .child(
                div()
                    .text_xl()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child("Zalohy"),
            )
            .child(render_button(
                "btn-year-prev",
                "<",
                ButtonVariant::Secondary,
                self.editing,
                false,
                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.change_year(-1, cx);
                }),
            ))
            .child(
                div()
                    .px_3()
                    .py_1()
                    .bg(rgb(ZfColors::SURFACE))
                    .border_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .rounded_md()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(self.year.to_string()),
            )
            .child(render_button(
                "btn-year-next",
                ">",
                ButtonVariant::Secondary,
                self.editing,
                false,
                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.change_year(1, cx);
                }),
            ));

        // Show flat rate label in view mode (not loading, not editing)
        if !self.loading && !self.editing {
            header = header.child(
                div()
                    .ml_4()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child(self.flat_rate_label()),
            );
        }

        content = content.child(header);

        if self.loading {
            return content.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Nacitani..."),
            );
        }

        // Success banner
        if let Some(ref msg) = self.success {
            content = content.child(
                div()
                    .px_4()
                    .py_3()
                    .bg(rgb(ZfColors::STATUS_GREEN_BG))
                    .rounded_md()
                    .text_sm()
                    .text_color(rgb(ZfColors::STATUS_GREEN))
                    .child(msg.clone()),
            );
        }

        // Error banner (non-blocking in edit mode)
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

        if self.editing {
            content = content.child(self.render_edit_mode(cx));
        } else {
            content = content.child(self.render_view_mode(cx));
        }

        content
    }
}
