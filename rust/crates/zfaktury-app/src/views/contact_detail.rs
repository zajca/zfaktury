use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::ContactService;
use zfaktury_domain::Contact;

use crate::components::button::{ButtonVariant, render_button};
use crate::components::confirm_dialog::{ConfirmDialog, ConfirmDialogResult};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;

/// Contact detail view displaying all contact data with action buttons.
pub struct ContactDetailView {
    service: Arc<ContactService>,
    contact_id: i64,
    loading: bool,
    error: Option<String>,
    contact: Option<Contact>,
    confirm_dialog: Option<Entity<ConfirmDialog>>,
    action_loading: bool,
}

impl ContactDetailView {
    pub fn new(service: Arc<ContactService>, contact_id: i64, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            contact_id,
            loading: true,
            error: None,
            contact: None,
            confirm_dialog: None,
            action_loading: false,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let id = self.contact_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.get_by_id(id) })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(contact) => this.contact = Some(contact),
                    Err(e) => this.error = Some(format!("Chyba pri nacitani kontaktu: {e}")),
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
                "Smazat kontakt?",
                "Tato akce je nevratna. Kontakt bude trvale smazan.",
                "Smazat",
            )
        });
        cx.subscribe(
            &dialog,
            |this: &mut Self, _, result: &ConfirmDialogResult, cx| match result {
                ConfirmDialogResult::Confirmed => {
                    let service = this.service.clone();
                    let id = this.contact_id;
                    this.confirm_dialog = None;
                    this.action_loading = true;
                    this.error = None;
                    cx.notify();
                    cx.spawn(async move |this, cx| {
                        let result = cx
                            .background_executor()
                            .spawn(async move { service.delete(id) })
                            .await;
                        this.update(cx, |this, cx| {
                            this.action_loading = false;
                            match result {
                                Ok(()) => cx.emit(NavigateEvent(Route::ContactList)),
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

    fn render_action_buttons(&self, cx: &mut Context<Self>) -> Div {
        let mut bar = div().flex().items_center().gap_2().flex_wrap();
        let disabled = self.action_loading;
        let contact_id = self.contact_id;

        // Back button
        bar = bar.child(render_button(
            "btn-back",
            "Zpet na seznam",
            ButtonVariant::Secondary,
            disabled,
            false,
            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                cx.emit(NavigateEvent(Route::ContactList));
            }),
        ));

        // Edit button
        bar = bar.child(render_button(
            "btn-edit",
            "Upravit",
            ButtonVariant::Secondary,
            disabled,
            false,
            cx.listener(move |_this, _event: &ClickEvent, _window, cx| {
                cx.emit(NavigateEvent(Route::ContactEdit(contact_id)));
            }),
        ));

        // Delete button
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

        bar
    }

    fn render_field(&self, label: &str, value: &str) -> Div {
        div()
            .flex()
            .items_center()
            .gap_4()
            .child(
                div()
                    .w_40()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child(label.to_string()),
            )
            .child(
                div()
                    .flex_1()
                    .text_sm()
                    .text_color(if value.is_empty() {
                        rgb(ZfColors::TEXT_MUTED)
                    } else {
                        rgb(ZfColors::TEXT_PRIMARY)
                    })
                    .child(if value.is_empty() {
                        "-".to_string()
                    } else {
                        value.to_string()
                    }),
            )
    }

    fn render_section(&self, title: &str, fields: Vec<(&str, String)>) -> Div {
        let mut section = div()
            .p_4()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .flex()
            .flex_col()
            .gap_3()
            .child(
                div()
                    .text_sm()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(title.to_string()),
            );

        for (label, value) in fields {
            section = section.child(self.render_field(label, &value));
        }

        section
    }

    fn render_contact_content(&self, c: &Contact, cx: &mut Context<Self>) -> Div {
        let mut content = div().flex().flex_col().gap_6();

        // Header
        content = content.child(
            div()
                .text_xl()
                .font_weight(FontWeight::SEMIBOLD)
                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                .child(c.name.clone()),
        );

        // Action buttons
        content = content.child(self.render_action_buttons(cx));

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

        // Basic info section
        content = content.child(self.render_section(
            "Zakladni udaje",
            vec![
                ("Nazev", c.name.clone()),
                ("Typ", c.contact_type.to_string()),
                ("ICO", c.ico.clone()),
                ("DIC", c.dic.clone()),
                ("Ulice", c.street.clone()),
                ("Mesto", c.city.clone()),
                ("PSC", c.zip.clone()),
                ("Zeme", c.country.clone()),
                ("Email", c.email.clone()),
                ("Telefon", c.phone.clone()),
                ("Web", c.web.clone()),
            ],
        ));

        // Bank details section
        content = content.child(self.render_section(
            "Bankovni udaje",
            vec![
                ("Cislo uctu", c.bank_account.clone()),
                ("Kod banky", c.bank_code.clone()),
                ("IBAN", c.iban.clone()),
                ("SWIFT/BIC", c.swift.clone()),
                ("Splatnost (dny)", c.payment_terms_days.to_string()),
            ],
        ));

        // Notes
        if !c.notes.is_empty() {
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
                            .child(c.notes.clone()),
                    ),
            );
        }

        content
    }
}

impl EventEmitter<NavigateEvent> for ContactDetailView {}

impl Render for ContactDetailView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("contact-detail-scroll")
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
                    .child("Nacitani kontaktu..."),
            );
        }

        if self.contact.is_none()
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

        if let Some(ref contact) = self.contact.clone() {
            outer = outer.child(self.render_contact_content(contact, cx));
        }

        // Confirm dialog overlay
        if let Some(ref dialog) = self.confirm_dialog {
            outer = outer.child(dialog.clone());
        }

        outer
    }
}
