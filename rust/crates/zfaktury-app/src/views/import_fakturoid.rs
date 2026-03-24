use std::sync::Arc;

use gpui::prelude::FluentBuilder;
use gpui::*;
use zfaktury_core::service::FakturoidImportService;
use zfaktury_domain::FakturoidImportResult;

use crate::components::button::{ButtonVariant, render_button};
use crate::components::checkbox::Checkbox;
use crate::components::text_input::TextInput;
use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;

/// State of the import flow.
enum ImportState {
    Idle,
    Importing,
    Done(FakturoidImportResult),
}

/// Import from Fakturoid view.
pub struct ImportFakturoidView {
    service: Arc<FakturoidImportService>,

    state: ImportState,
    error: Option<String>,

    // Form inputs
    slug_input: Entity<TextInput>,
    email_input: Entity<TextInput>,
    client_id_input: Entity<TextInput>,
    client_secret_input: Entity<TextInput>,
    download_attachments: Entity<Checkbox>,
}

impl EventEmitter<NavigateEvent> for ImportFakturoidView {}

impl ImportFakturoidView {
    pub fn new(service: Arc<FakturoidImportService>, cx: &mut Context<Self>) -> Self {
        let slug_input = cx.new(|cx| TextInput::new("fak-slug", "napr. moje-firma", cx));
        let email_input = cx.new(|cx| TextInput::new("fak-email", "vas@email.cz", cx));
        let client_id_input = cx.new(|cx| TextInput::new("fak-client-id", "Client ID", cx));
        let client_secret_input =
            cx.new(|cx| TextInput::new("fak-client-secret", "Client Secret", cx));
        let download_attachments =
            cx.new(|_cx| Checkbox::new("fak-attachments", "Stahnout prilohy", false));

        Self {
            service,
            state: ImportState::Idle,
            error: None,
            slug_input,
            email_input,
            client_id_input,
            client_secret_input,
            download_attachments,
        }
    }

    fn start_import(&mut self, cx: &mut Context<Self>) {
        let slug = self.slug_input.read(cx).value().to_string();
        let email = self.email_input.read(cx).value().to_string();
        let client_id = self.client_id_input.read(cx).value().to_string();
        let client_secret = self.client_secret_input.read(cx).value().to_string();
        let download_attachments = self.download_attachments.read(cx).is_checked();

        // Validate
        if slug.is_empty() || email.is_empty() || client_id.is_empty() || client_secret.is_empty() {
            self.error = Some("Vyplnte vsechna pole".to_string());
            cx.notify();
            return;
        }

        self.state = ImportState::Importing;
        self.error = None;
        cx.notify();

        let service = self.service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    service.import_all(
                        &slug,
                        &email,
                        &client_id,
                        &client_secret,
                        download_attachments,
                    )
                })
                .await;

            this.update(cx, |this, cx| {
                match result {
                    Ok(import_result) => {
                        this.state = ImportState::Done(import_result);
                    }
                    Err(e) => {
                        this.state = ImportState::Idle;
                        this.error = Some(format!("Import selhal: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn reset(&mut self, cx: &mut Context<Self>) {
        self.state = ImportState::Idle;
        self.error = None;
        cx.notify();
    }

    fn render_form(&self, cx: &mut Context<Self>) -> Div {
        div()
            .flex()
            .flex_col()
            .gap_4()
            // Form card
            .child(
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
                            .child("Prihlasovaci udaje Fakturoid"),
                    )
                    .child(
                        div()
                            .text_xs()
                            .text_color(rgb(ZfColors::TEXT_MUTED))
                            .child("Udaje najdete v Nastaveni > API v uctu Fakturoid. Udaje se nikam neukladaji."),
                    )
                    // Slug
                    .child(self.render_field("Slug uctu", self.slug_input.clone()))
                    // Email
                    .child(self.render_field("Email", self.email_input.clone()))
                    // Client ID
                    .child(self.render_field("Client ID", self.client_id_input.clone()))
                    // Client Secret
                    .child(self.render_field("Client Secret", self.client_secret_input.clone()))
                    // Checkbox
                    .child(
                        div().pt_1().child(self.download_attachments.clone()),
                    ),
            )
            // Error
            .children(self.error.as_ref().map(|error| {
                div()
                    .px_4()
                    .py_3()
                    .bg(rgb(ZfColors::STATUS_RED_BG))
                    .rounded_md()
                    .text_sm()
                    .text_color(rgb(ZfColors::STATUS_RED))
                    .child(error.clone())
            }))
            // Button
            .child(
                div().flex().child(render_button(
                    "import-btn",
                    "Spustit import",
                    ButtonVariant::Primary,
                    false,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.start_import(cx);
                    }),
                )),
            )
    }

    fn render_importing(&self) -> Div {
        div().flex().flex_col().gap_4().child(
            div()
                .p_6()
                .bg(rgb(ZfColors::SURFACE))
                .rounded_md()
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
                .flex()
                .flex_col()
                .items_center()
                .gap_3()
                .child(
                    div()
                        .text_sm()
                        .font_weight(FontWeight::MEDIUM)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("Probiha import z Fakturoid..."),
                )
                .child(
                    div().text_xs().text_color(rgb(ZfColors::TEXT_MUTED)).child(
                        "Stahuji kontakty, faktury a naklady. Toto muze trvat nekolik minut.",
                    ),
                ),
        )
    }

    fn render_result(&self, result: &FakturoidImportResult, cx: &mut Context<Self>) -> Div {
        div()
            .flex()
            .flex_col()
            .gap_4()
            // Results card
            .child(
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
                            .child("Import dokoncen"),
                    )
                    // Stats grid
                    .child(
                        div()
                            .flex()
                            .flex_wrap()
                            .gap_6()
                            .child(self.render_stat("Kontakty vytvoreny", result.contacts_created))
                            .child(self.render_stat("Kontakty preskoceny", result.contacts_skipped))
                            .child(self.render_stat("Faktury vytvoreny", result.invoices_created))
                            .child(self.render_stat("Faktury preskoceny", result.invoices_skipped))
                            .child(self.render_stat("Naklady vytvoreny", result.expenses_created))
                            .child(self.render_stat("Naklady preskoceny", result.expenses_skipped))
                            .child(
                                self.render_stat("Prilohy stazeny", result.attachments_downloaded),
                            )
                            .child(
                                self.render_stat("Prilohy preskoceny", result.attachments_skipped),
                            ),
                    ),
            )
            // Errors
            .when(!result.errors.is_empty(), |this| {
                this.child(
                    div()
                        .p_4()
                        .bg(rgb(ZfColors::STATUS_RED_BG))
                        .rounded_md()
                        .border_1()
                        .border_color(rgb(ZfColors::STATUS_RED))
                        .flex()
                        .flex_col()
                        .gap_2()
                        .child(
                            div()
                                .text_sm()
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(rgb(ZfColors::STATUS_RED))
                                .child(format!("Chyby ({})", result.errors.len())),
                        )
                        .children(result.errors.iter().map(|e| {
                            div()
                                .text_xs()
                                .text_color(rgb(ZfColors::STATUS_RED))
                                .child(e.clone())
                        })),
                )
            })
            // Reset button
            .child(div().flex().child(render_button(
                "reset-btn",
                "Novy import",
                ButtonVariant::Secondary,
                false,
                false,
                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.reset(cx);
                }),
            )))
    }

    fn render_field(&self, label: &str, input: Entity<TextInput>) -> Div {
        div()
            .flex()
            .flex_col()
            .gap(px(4.0))
            .child(
                div()
                    .text_xs()
                    .font_weight(FontWeight::MEDIUM)
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child(label.to_string()),
            )
            .child(input)
    }

    fn render_stat(&self, label: &str, value: i32) -> Div {
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
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(value.to_string()),
            )
    }
}

impl Render for ImportFakturoidView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let content = match &self.state {
            ImportState::Idle => self.render_form(cx),
            ImportState::Importing => self.render_importing(),
            ImportState::Done(result) => {
                // Clone result to avoid borrow issues
                let result = result.clone();
                self.render_result(&result, cx)
            }
        };

        div()
            .id("import-fakturoid-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll()
            // Header
            .child(
                div()
                    .text_xl()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child("Import z Fakturoid"),
            )
            .child(content)
    }
}
