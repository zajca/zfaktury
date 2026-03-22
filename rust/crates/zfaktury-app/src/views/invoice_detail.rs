use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::{InvoiceService, SettingsService};
use zfaktury_domain::{Invoice, InvoiceStatus, InvoiceType};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::confirm_dialog::{ConfirmDialog, ConfirmDialogResult};
use crate::components::status_badge::render_status_badge;
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;
use crate::util::format::{format_amount, format_date, format_number};

/// Invoice detail view displaying all invoice data with action buttons.
pub struct InvoiceDetailView {
    service: Arc<InvoiceService>,
    /// Settings service for supplier info + PDF settings.
    /// NOTE for lead: wire this from `services.settings.clone()` in root.rs when creating InvoiceDetailView.
    settings_service: Arc<SettingsService>,
    invoice_id: i64,
    loading: bool,
    error: Option<String>,
    success: Option<String>,
    invoice: Option<Invoice>,
    confirm_dialog: Option<Entity<ConfirmDialog>>,
    action_loading: bool,
    pdf_generating: bool,
    isdoc_generating: bool,
}

impl InvoiceDetailView {
    pub fn new(
        service: Arc<InvoiceService>,
        settings_service: Arc<SettingsService>,
        invoice_id: i64,
        cx: &mut Context<Self>,
    ) -> Self {
        let mut view = Self {
            service,
            settings_service,
            invoice_id,
            loading: true,
            error: None,
            success: None,
            invoice: None,
            confirm_dialog: None,
            action_loading: false,
            pdf_generating: false,
            isdoc_generating: false,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let id = self.invoice_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.get_by_id(id) })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(invoice) => this.invoice = Some(invoice),
                    Err(e) => this.error = Some(format!("Chyba pri nacitani faktury: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn handle_mark_sent(&mut self, cx: &mut Context<Self>) {
        self.action_loading = true;
        self.error = None;
        cx.notify();
        let service = self.service.clone();
        let id = self.invoice_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.mark_as_sent(id) })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(()) => this.load_data(cx),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn handle_mark_paid(&mut self, cx: &mut Context<Self>) {
        let total = match self.invoice {
            Some(ref inv) => inv.total_amount,
            None => return,
        };
        self.action_loading = true;
        self.error = None;
        cx.notify();
        let service = self.service.clone();
        let id = self.invoice_id;
        let now = chrono::Local::now().naive_local();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.mark_as_paid(id, total, now) })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(()) => this.load_data(cx),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn handle_duplicate(&mut self, cx: &mut Context<Self>) {
        self.action_loading = true;
        self.error = None;
        cx.notify();
        let service = self.service.clone();
        let id = self.invoice_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.duplicate(id) })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(new_inv) => cx.emit(NavigateEvent(Route::InvoiceDetail(new_inv.id))),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn handle_credit_note(&mut self, cx: &mut Context<Self>) {
        self.action_loading = true;
        self.error = None;
        cx.notify();
        let service = self.service.clone();
        let id = self.invoice_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.create_credit_note(id, None, "Dobropis") })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(cn) => cx.emit(NavigateEvent(Route::InvoiceDetail(cn.id))),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn handle_settle_proforma(&mut self, cx: &mut Context<Self>) {
        self.action_loading = true;
        self.error = None;
        cx.notify();
        let service = self.service.clone();
        let id = self.invoice_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.settle_proforma(id) })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(settlement) => {
                        cx.emit(NavigateEvent(Route::InvoiceDetail(settlement.id)));
                    }
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
        let id = self.invoice_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.delete(id) })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(()) => cx.emit(NavigateEvent(Route::InvoiceList)),
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
                "Smazat fakturu?",
                "Tato akce je nevratna. Faktura bude trvale smazana.",
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

    fn download_pdf(&mut self, cx: &mut Context<Self>) {
        let invoice = match self.invoice.clone() {
            Some(inv) => inv,
            None => return,
        };

        self.pdf_generating = true;
        self.error = None;
        self.success = None;
        cx.notify();

        let settings_service = self.settings_service.clone();
        let invoice_number = invoice.invoice_number.clone();

        cx.spawn(async move |this, cx| {
            let result: Result<String, String> = cx
                .background_executor()
                .spawn(async move {
                    // Load supplier info from settings.
                    let all_settings = settings_service
                        .get_all()
                        .map_err(|e| format!("Chyba pri nacitani nastaveni: {e}"))?;

                    let supplier = build_pdf_supplier_info(&all_settings);

                    // Load PDF settings.
                    let pdf_domain_settings = settings_service
                        .get_pdf_settings()
                        .map_err(|e| format!("Chyba pri nacitani PDF nastaveni: {e}"))?;
                    let render_settings = zfaktury_gen::pdf::PdfRenderSettings {
                        accent_color: pdf_domain_settings
                            .accent_color
                            .unwrap_or_else(|| "#2563eb".to_string()),
                        footer_text: pdf_domain_settings.footer_text.unwrap_or_default(),
                        show_qr: pdf_domain_settings.show_qr,
                        show_bank_details: pdf_domain_settings.show_bank_details,
                    };

                    // Generate PDF.
                    let pdf_bytes = zfaktury_gen::pdf::generate_invoice_pdf(
                        &invoice,
                        &supplier,
                        &render_settings,
                    )
                    .map_err(|e| format!("Chyba pri generovani PDF: {e}"))?;

                    // Save to temp file.
                    let tmp_path =
                        std::env::temp_dir().join(format!("faktura-{}.pdf", invoice_number));
                    std::fs::write(&tmp_path, &pdf_bytes)
                        .map_err(|e| format!("Chyba pri zapisu PDF: {e}"))?;

                    // Open with system viewer.
                    let _ = std::process::Command::new("xdg-open")
                        .arg(&tmp_path)
                        .spawn();

                    Ok(tmp_path.to_string_lossy().to_string())
                })
                .await;

            this.update(cx, |this, cx| {
                this.pdf_generating = false;
                match result {
                    Ok(_path) => {
                        this.success = Some("PDF vygenerovano a otevreno".to_string());
                    }
                    Err(e) => {
                        this.error = Some(e);
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn export_isdoc(&mut self, cx: &mut Context<Self>) {
        let invoice = match self.invoice.clone() {
            Some(inv) => inv,
            None => return,
        };

        self.isdoc_generating = true;
        self.error = None;
        self.success = None;
        cx.notify();

        let settings_service = self.settings_service.clone();
        let invoice_number = invoice.invoice_number.clone();

        cx.spawn(async move |this, cx| {
            let result: Result<String, String> = cx
                .background_executor()
                .spawn(async move {
                    // Load supplier info from settings.
                    let all_settings = settings_service
                        .get_all()
                        .map_err(|e| format!("Chyba pri nacitani nastaveni: {e}"))?;

                    let supplier = zfaktury_gen::isdoc::SupplierInfo {
                        company_name: all_settings
                            .get(zfaktury_domain::SETTING_COMPANY_NAME)
                            .cloned()
                            .unwrap_or_default(),
                        ico: all_settings
                            .get(zfaktury_domain::SETTING_ICO)
                            .cloned()
                            .unwrap_or_default(),
                        dic: all_settings
                            .get(zfaktury_domain::SETTING_DIC)
                            .cloned()
                            .unwrap_or_default(),
                        street: all_settings
                            .get(zfaktury_domain::SETTING_STREET)
                            .cloned()
                            .unwrap_or_default(),
                        city: all_settings
                            .get(zfaktury_domain::SETTING_CITY)
                            .cloned()
                            .unwrap_or_default(),
                        zip: all_settings
                            .get(zfaktury_domain::SETTING_ZIP)
                            .cloned()
                            .unwrap_or_default(),
                        email: all_settings
                            .get(zfaktury_domain::SETTING_EMAIL)
                            .cloned()
                            .unwrap_or_default(),
                        phone: all_settings
                            .get(zfaktury_domain::SETTING_PHONE)
                            .cloned()
                            .unwrap_or_default(),
                        bank_account: all_settings
                            .get(zfaktury_domain::SETTING_BANK_ACCOUNT)
                            .cloned()
                            .unwrap_or_default(),
                        bank_code: all_settings
                            .get(zfaktury_domain::SETTING_BANK_CODE)
                            .cloned()
                            .unwrap_or_default(),
                        iban: all_settings
                            .get(zfaktury_domain::SETTING_IBAN)
                            .cloned()
                            .unwrap_or_default(),
                        swift: all_settings
                            .get(zfaktury_domain::SETTING_SWIFT)
                            .cloned()
                            .unwrap_or_default(),
                    };

                    // Generate ISDOC XML.
                    let xml_bytes = zfaktury_gen::isdoc::generate_isdoc(&invoice, &supplier)
                        .map_err(|e| format!("Chyba pri generovani ISDOC: {e}"))?;

                    // Save to temp file.
                    let tmp_path =
                        std::env::temp_dir().join(format!("faktura-{}.isdoc", invoice_number));
                    std::fs::write(&tmp_path, &xml_bytes)
                        .map_err(|e| format!("Chyba pri zapisu ISDOC: {e}"))?;

                    // Open with system viewer.
                    let _ = std::process::Command::new("xdg-open")
                        .arg(&tmp_path)
                        .spawn();

                    Ok(tmp_path.to_string_lossy().to_string())
                })
                .await;

            this.update(cx, |this, cx| {
                this.isdoc_generating = false;
                match result {
                    Ok(_path) => {
                        this.success = Some("ISDOC vygenerovano a otevreno".to_string());
                    }
                    Err(e) => {
                        this.error = Some(e);
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn render_action_buttons(&self, inv: &Invoice, cx: &mut Context<Self>) -> Div {
        let mut bar = div().flex().items_center().gap_2().flex_wrap();
        let disabled = self.action_loading;

        // Back button (always)
        bar = bar.child(render_button(
            "btn-back",
            "Zpet",
            ButtonVariant::Secondary,
            disabled,
            false,
            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                cx.emit(NavigateEvent(Route::InvoiceList));
            }),
        ));

        match inv.status {
            InvoiceStatus::Draft => {
                // Edit
                let inv_id = inv.id;
                bar = bar.child(render_button(
                    "btn-edit",
                    "Upravit",
                    ButtonVariant::Secondary,
                    disabled,
                    false,
                    cx.listener(move |_this, _event: &ClickEvent, _window, cx| {
                        cx.emit(NavigateEvent(Route::InvoiceEdit(inv_id)));
                    }),
                ));
                // Send
                bar = bar.child(render_button(
                    "btn-send",
                    "Odeslat",
                    ButtonVariant::Primary,
                    disabled,
                    self.action_loading,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.handle_mark_sent(cx);
                    }),
                ));
                // Duplicate
                bar = bar.child(render_button(
                    "btn-duplicate",
                    "Duplikovat",
                    ButtonVariant::Secondary,
                    disabled,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.handle_duplicate(cx);
                    }),
                ));
                // Delete
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
            InvoiceStatus::Sent | InvoiceStatus::Overdue => {
                // Edit
                let inv_id = inv.id;
                bar = bar.child(render_button(
                    "btn-edit",
                    "Upravit",
                    ButtonVariant::Secondary,
                    disabled,
                    false,
                    cx.listener(move |_this, _event: &ClickEvent, _window, cx| {
                        cx.emit(NavigateEvent(Route::InvoiceEdit(inv_id)));
                    }),
                ));
                // Pay
                bar = bar.child(render_button(
                    "btn-pay",
                    "Uhradit",
                    ButtonVariant::Primary,
                    disabled,
                    self.action_loading,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.handle_mark_paid(cx);
                    }),
                ));
                // Credit note
                bar = bar.child(render_button(
                    "btn-credit-note",
                    "Dobropis",
                    ButtonVariant::Secondary,
                    disabled,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.handle_credit_note(cx);
                    }),
                ));
                // Duplicate
                bar = bar.child(render_button(
                    "btn-duplicate",
                    "Duplikovat",
                    ButtonVariant::Secondary,
                    disabled,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.handle_duplicate(cx);
                    }),
                ));
                // Delete
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
            InvoiceStatus::Paid => {
                // Duplicate
                bar = bar.child(render_button(
                    "btn-duplicate",
                    "Duplikovat",
                    ButtonVariant::Secondary,
                    disabled,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.handle_duplicate(cx);
                    }),
                ));
                // Settle proforma (only for paid proformas)
                if inv.invoice_type == InvoiceType::Proforma {
                    bar = bar.child(render_button(
                        "btn-settle",
                        "Vyuctovat",
                        ButtonVariant::Primary,
                        disabled,
                        self.action_loading,
                        cx.listener(|this, _event: &ClickEvent, _window, cx| {
                            this.handle_settle_proforma(cx);
                        }),
                    ));
                }
                // Credit note (for paid regular invoices)
                if inv.invoice_type == InvoiceType::Regular {
                    bar = bar.child(render_button(
                        "btn-credit-note",
                        "Dobropis",
                        ButtonVariant::Secondary,
                        disabled,
                        false,
                        cx.listener(|this, _event: &ClickEvent, _window, cx| {
                            this.handle_credit_note(cx);
                        }),
                    ));
                }
            }
            InvoiceStatus::Cancelled => {
                // Duplicate only
                bar = bar.child(render_button(
                    "btn-duplicate",
                    "Duplikovat",
                    ButtonVariant::Secondary,
                    disabled,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.handle_duplicate(cx);
                    }),
                ));
            }
        }

        // PDF download button (available for all statuses)
        bar = bar.child(render_button(
            "btn-download-pdf",
            "Stahnout PDF",
            ButtonVariant::Secondary,
            disabled || self.pdf_generating,
            self.pdf_generating,
            cx.listener(|this, _event: &ClickEvent, _window, cx| {
                this.download_pdf(cx);
            }),
        ));

        // ISDOC export button (available for all statuses)
        bar = bar.child(render_button(
            "btn-export-isdoc",
            "ISDOC",
            ButtonVariant::Secondary,
            disabled || self.isdoc_generating,
            self.isdoc_generating,
            cx.listener(|this, _event: &ClickEvent, _window, cx| {
                this.export_isdoc(cx);
            }),
        ));

        bar
    }

    fn render_field(&self, label: &str, value: String) -> Div {
        div()
            .flex()
            .flex_col()
            .gap(px(2.0))
            .child(
                div()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child(label.to_string()),
            )
            .child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(value),
            )
    }

    fn render_invoice_content(&self, inv: &Invoice, cx: &mut Context<Self>) -> Div {
        let customer_name = inv
            .customer
            .as_ref()
            .map(|c| c.name.clone())
            .unwrap_or_else(|| format!("ID {}", inv.customer_id));

        let mut content = div().flex().flex_col().gap_6();

        // Header with number and status
        content = content.child(
            div()
                .flex()
                .items_center()
                .justify_between()
                .child(
                    div()
                        .text_xl()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child(format!("Faktura {}", inv.invoice_number)),
                )
                .child(render_status_badge(&inv.status)),
        );

        // Action buttons
        content = content.child(self.render_action_buttons(inv, cx));

        // Success message
        if let Some(ref success) = self.success {
            content = content.child(
                div()
                    .px_4()
                    .py_3()
                    .bg(rgb(ZfColors::STATUS_GREEN_BG))
                    .rounded_md()
                    .text_sm()
                    .text_color(rgb(ZfColors::STATUS_GREEN))
                    .child(success.clone()),
            );
        }

        // Error message (if action failed)
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

        // Info grid
        content = content.child(
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
                        .flex()
                        .gap_8()
                        .child(self.render_field("Zakaznik", customer_name))
                        .child(self.render_field("Typ", inv.invoice_type.to_string()))
                        .child(self.render_field("Mena", inv.currency_code.clone())),
                )
                .child(
                    div()
                        .flex()
                        .gap_8()
                        .child(self.render_field("Datum vystaveni", format_date(inv.issue_date)))
                        .child(self.render_field("Datum splatnosti", format_date(inv.due_date)))
                        .child(self.render_field(
                            "Datum zdanitelneho plneni",
                            format_date(inv.delivery_date),
                        )),
                )
                .child(
                    div()
                        .flex()
                        .gap_8()
                        .child(self.render_field("Variabilni symbol", inv.variable_symbol.clone()))
                        .child(self.render_field("Zpusob platby", inv.payment_method.clone())),
                ),
        );

        // Items table
        if !inv.items.is_empty() {
            let mut items_table = div()
                .flex()
                .flex_col()
                .bg(rgb(ZfColors::SURFACE))
                .rounded_md()
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
                .overflow_hidden();

            items_table = items_table.child(
                div()
                    .px_4()
                    .py_3()
                    .border_b_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .text_sm()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child("Polozky"),
            );

            // Column headers
            items_table = items_table.child(
                div()
                    .flex()
                    .px_4()
                    .py_2()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .border_b_1()
                    .border_color(rgb(ZfColors::BORDER_SUBTLE))
                    .child(div().flex_1().child("Popis"))
                    .child(div().w_20().text_right().child("Mnozstvi"))
                    .child(div().w_16().text_right().child("Jednotka"))
                    .child(div().w_24().text_right().child("Cena/ks"))
                    .child(div().w_16().text_right().child("DPH %"))
                    .child(div().w_24().text_right().child("DPH"))
                    .child(div().w(px(112.0)).text_right().child("Celkem")),
            );

            for item in &inv.items {
                items_table = items_table.child(
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
                                .flex_1()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(item.description.clone()),
                        )
                        .child(
                            div()
                                .w_20()
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format_number(item.quantity)),
                        )
                        .child(
                            div()
                                .w_16()
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(item.unit.clone()),
                        )
                        .child(
                            div()
                                .w_24()
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format_amount(item.unit_price)),
                        )
                        .child(
                            div()
                                .w_16()
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format!("{}%", item.vat_rate_percent)),
                        )
                        .child(
                            div()
                                .w_24()
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format_amount(item.vat_amount)),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_right()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(item.total_amount)),
                        ),
                );
            }

            content = content.child(items_table);
        }

        // Totals
        content = content.child(
            div()
                .p_4()
                .bg(rgb(ZfColors::SURFACE))
                .rounded_md()
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
                .flex()
                .flex_col()
                .gap_2()
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child("Zaklad dane"),
                        )
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(inv.subtotal_amount)),
                        ),
                )
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(div().text_color(rgb(ZfColors::TEXT_SECONDARY)).child("DPH"))
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(inv.vat_amount)),
                        ),
                )
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
                                .child("Celkem"),
                        )
                        .child(
                            div()
                                .text_lg()
                                .font_weight(FontWeight::BOLD)
                                .text_color(rgb(ZfColors::ACCENT))
                                .child(format_amount(inv.total_amount)),
                        ),
                ),
        );

        // Notes
        if !inv.notes.is_empty() {
            content = content.child(
                div()
                    .p_4()
                    .bg(rgb(ZfColors::SURFACE))
                    .rounded_md()
                    .border_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .flex()
                    .flex_col()
                    .gap_1()
                    .child(
                        div()
                            .text_xs()
                            .text_color(rgb(ZfColors::TEXT_MUTED))
                            .child("Poznamky"),
                    )
                    .child(
                        div()
                            .text_sm()
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .child(inv.notes.clone()),
                    ),
            );
        }

        content
    }
}

/// Build PDF SupplierInfo from settings HashMap.
fn build_pdf_supplier_info(
    all_settings: &std::collections::HashMap<String, String>,
) -> zfaktury_gen::pdf::SupplierInfo {
    zfaktury_gen::pdf::SupplierInfo {
        name: all_settings
            .get(zfaktury_domain::SETTING_COMPANY_NAME)
            .cloned()
            .unwrap_or_default(),
        ico: all_settings
            .get(zfaktury_domain::SETTING_ICO)
            .cloned()
            .unwrap_or_default(),
        dic: all_settings
            .get(zfaktury_domain::SETTING_DIC)
            .cloned()
            .unwrap_or_default(),
        vat_registered: all_settings
            .get(zfaktury_domain::SETTING_VAT_REGISTERED)
            .map(|v| v == "true")
            .unwrap_or(false),
        street: all_settings
            .get(zfaktury_domain::SETTING_STREET)
            .cloned()
            .unwrap_or_default(),
        city: all_settings
            .get(zfaktury_domain::SETTING_CITY)
            .cloned()
            .unwrap_or_default(),
        zip: all_settings
            .get(zfaktury_domain::SETTING_ZIP)
            .cloned()
            .unwrap_or_default(),
        email: all_settings
            .get(zfaktury_domain::SETTING_EMAIL)
            .cloned()
            .unwrap_or_default(),
        phone: all_settings
            .get(zfaktury_domain::SETTING_PHONE)
            .cloned()
            .unwrap_or_default(),
        bank_account: all_settings
            .get(zfaktury_domain::SETTING_BANK_ACCOUNT)
            .cloned()
            .unwrap_or_default(),
        bank_code: all_settings
            .get(zfaktury_domain::SETTING_BANK_CODE)
            .cloned()
            .unwrap_or_default(),
        iban: all_settings
            .get(zfaktury_domain::SETTING_IBAN)
            .cloned()
            .unwrap_or_default(),
        swift: all_settings
            .get(zfaktury_domain::SETTING_SWIFT)
            .cloned()
            .unwrap_or_default(),
    }
}

impl EventEmitter<NavigateEvent> for InvoiceDetailView {}

impl Render for InvoiceDetailView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("invoice-detail-scroll")
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
                    .child("Nacitani faktury..."),
            );
        }

        if self.invoice.is_none()
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

        if let Some(ref inv) = self.invoice.clone() {
            outer = outer.child(self.render_invoice_content(inv, cx));
        }

        // Confirm dialog overlay
        if let Some(ref dialog) = self.confirm_dialog {
            outer = outer.child(dialog.clone());
        }

        outer
    }
}
