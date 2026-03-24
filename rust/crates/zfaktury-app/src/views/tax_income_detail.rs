use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::IncomeTaxReturnService;
use zfaktury_domain::{FilingStatus, IncomeTaxReturn};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::confirm_dialog::{ConfirmDialog, ConfirmDialogResult};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;
use crate::util::format::format_amount;

/// Income tax return detail view.
pub struct TaxIncomeDetailView {
    service: Arc<IncomeTaxReturnService>,
    return_id: i64,
    loading: bool,
    error: Option<String>,
    tax_return: Option<IncomeTaxReturn>,
    confirm_dialog: Option<Entity<ConfirmDialog>>,
    action_loading: bool,
}

impl TaxIncomeDetailView {
    pub fn new(
        service: Arc<IncomeTaxReturnService>,
        return_id: i64,
        cx: &mut Context<Self>,
    ) -> Self {
        let mut view = Self {
            service,
            return_id,
            loading: true,
            error: None,
            tax_return: None,
            confirm_dialog: None,
            action_loading: false,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let id = self.return_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.get_by_id(id) })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(itr) => this.tax_return = Some(itr),
                    Err(e) => {
                        this.error = Some(format!("Chyba při načítání daňového přiznání: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn handle_mark_filed(&mut self, cx: &mut Context<Self>) {
        self.action_loading = true;
        self.error = None;
        cx.notify();
        let service = self.service.clone();
        let id = self.return_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.mark_filed(id) })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(_itr) => this.load_data(cx),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn handle_delete_confirmed(&mut self, cx: &mut Context<Self>) {
        self.confirm_dialog = None;
        self.action_loading = true;
        self.error = None;
        cx.notify();
        let service = self.service.clone();
        let id = self.return_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.delete(id) })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(()) => cx.emit(NavigateEvent(Route::TaxOverview)),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn show_delete_dialog(&mut self, cx: &mut Context<Self>) {
        let dialog = cx.new(|_cx| {
            ConfirmDialog::new(
                "Smazat daňové přiznání?",
                "Tato akce je nevratná. Daňové přiznání bude trvale smazáno.",
                "Smazat",
            )
        });
        cx.subscribe(
            &dialog,
            |this: &mut Self, _, result: &ConfirmDialogResult, cx| match result {
                ConfirmDialogResult::Confirmed => {
                    this.handle_delete_confirmed(cx);
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

    fn render_action_buttons(&self, itr: &IncomeTaxReturn, cx: &mut Context<Self>) -> Div {
        let mut bar = div().flex().items_center().gap_2().flex_wrap();
        let disabled = self.action_loading;

        bar = bar.child(render_button(
            "btn-back",
            "Zpět",
            ButtonVariant::Secondary,
            disabled,
            false,
            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                cx.emit(NavigateEvent(Route::TaxOverview));
            }),
        ));

        if itr.status != FilingStatus::Filed {
            bar = bar.child(render_button(
                "btn-mark-filed",
                "Označit jako podané",
                ButtonVariant::Primary,
                disabled,
                self.action_loading,
                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.handle_mark_filed(cx);
                }),
            ));

            bar = bar.child(render_button(
                "btn-delete",
                "Smazat",
                ButtonVariant::Danger,
                disabled,
                false,
                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.show_delete_dialog(cx);
                }),
            ));
        }

        bar
    }

    fn render_amount_row(&self, label: &str, value: zfaktury_domain::Amount) -> Div {
        div()
            .flex()
            .justify_between()
            .text_sm()
            .child(
                div()
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child(label.to_string()),
            )
            .child(
                div()
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(format_amount(value)),
            )
    }

    fn render_section_header(&self, title: &str) -> Div {
        div()
            .text_sm()
            .font_weight(FontWeight::SEMIBOLD)
            .text_color(rgb(ZfColors::TEXT_PRIMARY))
            .child(title.to_string())
    }

    fn render_card(&self, title: &str, rows: Vec<Div>) -> Div {
        let mut card = div()
            .p_4()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .flex()
            .flex_col()
            .gap_3()
            .child(self.render_section_header(title));

        for row in rows {
            card = card.child(row);
        }

        card
    }

    fn render_content(&self, itr: &IncomeTaxReturn, cx: &mut Context<Self>) -> Div {
        let status_text = itr.status.to_string();
        let status_color = match itr.status {
            FilingStatus::Draft => ZfColors::STATUS_GRAY,
            FilingStatus::Ready => ZfColors::STATUS_YELLOW,
            FilingStatus::Filed => ZfColors::STATUS_GREEN,
        };

        let mut content = div().flex().flex_col().gap_6();

        // Header
        content = content.child(
            div().flex().items_center().justify_between().child(
                div()
                    .flex()
                    .items_center()
                    .gap_3()
                    .child(
                        div()
                            .text_xl()
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child(format!("Daň z příjmů {}", itr.year)),
                    )
                    .child(
                        div()
                            .px_2()
                            .py(px(2.0))
                            .rounded(px(4.0))
                            .text_xs()
                            .text_color(rgb(status_color))
                            .child(status_text),
                    ),
            ),
        );

        // Action buttons
        content = content.child(self.render_action_buttons(itr, cx));

        // Error
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

        // Section 7 - Business income
        content = content.child(self.render_card(
            "Paragraf 7 - Příjmy z podnikání",
            vec![
                self.render_amount_row("Celkové příjmy", itr.total_revenue),
                self.render_amount_row("Skutečné výdaje", itr.actual_expenses),
                self.render_amount_row(
                    &format!("Paušální výdaje ({}%)", itr.flat_rate_percent),
                    itr.flat_rate_amount,
                ),
                self.render_amount_row("Uplatněné výdaje", itr.used_expenses),
            ],
        ));

        // Tax base
        content = content.child(self.render_card(
            "Základ daně",
            vec![
                self.render_amount_row("Základ daně", itr.tax_base),
                self.render_amount_row("Odpočty celkem", itr.total_deductions),
                self.render_amount_row("Zaokrouhlený základ", itr.tax_base_rounded),
            ],
        ));

        // Tax calculation
        content = content.child(self.render_card(
            "Výpočet daně",
            vec![
                self.render_amount_row("Daň 15%", itr.tax_at_15),
                self.render_amount_row("Daň 23%", itr.tax_at_23),
                self.render_amount_row("Celková daň", itr.total_tax),
            ],
        ));

        // Credits
        content = content.child(self.render_card(
            "Slevy na dani",
            vec![
                self.render_amount_row("Základní sleva", itr.credit_basic),
                self.render_amount_row("Sleva na manžela/ku", itr.credit_spouse),
                self.render_amount_row("Invalidita", itr.credit_disability),
                self.render_amount_row("Student", itr.credit_student),
                self.render_amount_row("Slevy celkem", itr.total_credits),
                div().h(px(1.0)).bg(rgb(ZfColors::BORDER)),
                self.render_amount_row("Daň po slevách", itr.tax_after_credits),
            ],
        ));

        // Child benefit
        content = content.child(self.render_card(
            "Daňové zvýhodnění na děti",
            vec![
                self.render_amount_row("Zvýhodnění", itr.child_benefit),
                self.render_amount_row("Daň po zvýhodnění", itr.tax_after_benefit),
            ],
        ));

        // Investment income
        if itr.capital_income_gross != zfaktury_domain::Amount::ZERO
            || itr.other_income_gross != zfaktury_domain::Amount::ZERO
        {
            content = content.child(self.render_card(
                "Investiční příjmy",
                vec![
                    self.render_amount_row(
                        "Kapitálové příjmy (ř.8) brutto",
                        itr.capital_income_gross,
                    ),
                    self.render_amount_row("Sražená daň (ř.8)", itr.capital_income_tax),
                    self.render_amount_row("Ostatní příjmy (ř.10) brutto", itr.other_income_gross),
                    self.render_amount_row("Výdaje (ř.10)", itr.other_income_expenses),
                    self.render_amount_row("Osvobozeno (ř.10)", itr.other_income_exempt),
                ],
            ));
        }

        // Result
        content = content.child(
            div()
                .p_4()
                .bg(rgb(ZfColors::SURFACE))
                .rounded_md()
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
                .flex()
                .flex_col()
                .gap_3()
                .child(self.render_section_header("Výsledek"))
                .child(self.render_amount_row("Zaplacené zálohy", itr.prepayments))
                .child(div().h(px(1.0)).bg(rgb(ZfColors::BORDER)))
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .child(
                            div()
                                .text_sm()
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child("Doplatek / Přeplatek"),
                        )
                        .child(
                            div()
                                .text_lg()
                                .font_weight(FontWeight::BOLD)
                                .text_color(rgb(ZfColors::ACCENT))
                                .child(format_amount(itr.tax_due)),
                        ),
                ),
        );

        content
    }
}

impl EventEmitter<NavigateEvent> for TaxIncomeDetailView {}

impl Render for TaxIncomeDetailView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("tax-income-detail-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .overflow_y_scroll()
            .relative();

        if self.loading {
            return outer.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Načítání daňového přiznání..."),
            );
        }

        if self.tax_return.is_none()
            && let Some(ref error) = self.error
        {
            return outer.child(
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

        if let Some(ref itr) = self.tax_return.clone() {
            outer = outer.child(self.render_content(itr, cx));
        }

        if let Some(ref dialog) = self.confirm_dialog {
            outer = outer.child(dialog.clone());
        }

        outer
    }
}
