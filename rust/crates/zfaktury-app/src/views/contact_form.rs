use std::sync::Arc;

use chrono::Local;
use gpui::*;
use zfaktury_core::service::contact_svc::ContactService;
use zfaktury_domain::{Contact, ContactType, DomainError};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::checkbox::Checkbox;
use crate::components::number_input::NumberInput;
use crate::components::select::{Select, SelectOption};
use crate::components::text_area::TextArea;
use crate::components::text_input::{TextInput, render_form_field};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;

/// Contact creation/edit form view.
#[allow(dead_code)]
pub struct ContactFormView {
    service: Arc<ContactService>,
    is_edit: bool,
    contact_id: Option<i64>,
    saving: bool,
    loading: bool,
    error: Option<String>,

    // Form fields
    name: Entity<TextInput>,
    contact_type: Entity<Select>,
    ico: Entity<TextInput>,
    dic: Entity<TextInput>,
    street: Entity<TextInput>,
    city: Entity<TextInput>,
    zip: Entity<TextInput>,
    country: Entity<TextInput>,
    email: Entity<TextInput>,
    phone: Entity<TextInput>,
    web: Entity<TextInput>,
    bank_account: Entity<TextInput>,
    bank_code: Entity<TextInput>,
    iban: Entity<TextInput>,
    swift: Entity<TextInput>,
    payment_terms: Entity<NumberInput>,
    notes: Entity<TextArea>,
    is_favorite: Entity<Checkbox>,
}

impl EventEmitter<NavigateEvent> for ContactFormView {}

fn contact_type_options() -> Vec<SelectOption> {
    vec![
        SelectOption {
            value: "company".to_string(),
            label: "Firma".to_string(),
        },
        SelectOption {
            value: "individual".to_string(),
            label: "Fyzická osoba".to_string(),
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

impl ContactFormView {
    /// Create a new contact form (create mode).
    pub fn new_create(service: Arc<ContactService>, cx: &mut Context<Self>) -> Self {
        let name = cx.new(|cx| TextInput::new("name", "Název kontaktu...", cx));
        let contact_type = cx.new(|cx| {
            let mut s = Select::new("contact-type", "Typ kontaktu", contact_type_options());
            s.set_selected_value("company", cx);
            s
        });
        let ico = cx.new(|cx| TextInput::new("ico", "IČO...", cx));
        let dic = cx.new(|cx| TextInput::new("dic", "DIČ...", cx));
        let street = cx.new(|cx| TextInput::new("street", "Ulice a číslo popisné...", cx));
        let city = cx.new(|cx| TextInput::new("city", "Město...", cx));
        let zip = cx.new(|cx| TextInput::new("zip", "PSČ...", cx));
        let country = cx.new(|cx| {
            let mut t = TextInput::new("country", "Země...", cx);
            t.set_value("CZ", cx);
            t
        });
        let email = cx.new(|cx| TextInput::new("email", "Email...", cx));
        let phone = cx.new(|cx| TextInput::new("phone", "Telefon...", cx));
        let web = cx.new(|cx| TextInput::new("web", "Web...", cx));
        let bank_account = cx.new(|cx| TextInput::new("bank-account", "Číslo účtu...", cx));
        let bank_code = cx.new(|cx| TextInput::new("bank-code", "Kód banky...", cx));
        let iban = cx.new(|cx| TextInput::new("iban", "IBAN...", cx));
        let swift = cx.new(|cx| TextInput::new("swift", "SWIFT/BIC...", cx));
        let payment_terms = cx.new(|cx| {
            NumberInput::new("payment-terms", "14", cx)
                .integer_only()
                .with_value("14")
        });
        let notes = cx.new(|cx| TextArea::new("notes", "Poznámky...", cx).with_rows(4));
        let is_favorite = cx.new(|_cx| Checkbox::new("is-favorite", "Oblíbený kontakt", false));

        Self {
            service,
            is_edit: false,
            contact_id: None,
            saving: false,
            loading: false,
            error: None,
            name,
            contact_type,
            ico,
            dic,
            street,
            city,
            zip,
            country,
            email,
            phone,
            web,
            bank_account,
            bank_code,
            iban,
            swift,
            payment_terms,
            notes,
            is_favorite,
        }
    }

    /// Create a contact form in edit mode (loads existing contact by ID).
    pub fn new_edit(service: Arc<ContactService>, id: i64, cx: &mut Context<Self>) -> Self {
        let name = cx.new(|cx| TextInput::new("name", "Název kontaktu...", cx));
        let contact_type =
            cx.new(|_cx| Select::new("contact-type", "Typ kontaktu", contact_type_options()));
        let ico = cx.new(|cx| TextInput::new("ico", "IČO...", cx));
        let dic = cx.new(|cx| TextInput::new("dic", "DIČ...", cx));
        let street = cx.new(|cx| TextInput::new("street", "Ulice a číslo popisné...", cx));
        let city = cx.new(|cx| TextInput::new("city", "Město...", cx));
        let zip = cx.new(|cx| TextInput::new("zip", "PSČ...", cx));
        let country = cx.new(|cx| TextInput::new("country", "Země...", cx));
        let email = cx.new(|cx| TextInput::new("email", "Email...", cx));
        let phone = cx.new(|cx| TextInput::new("phone", "Telefon...", cx));
        let web = cx.new(|cx| TextInput::new("web", "Web...", cx));
        let bank_account = cx.new(|cx| TextInput::new("bank-account", "Číslo účtu...", cx));
        let bank_code = cx.new(|cx| TextInput::new("bank-code", "Kód banky...", cx));
        let iban = cx.new(|cx| TextInput::new("iban", "IBAN...", cx));
        let swift = cx.new(|cx| TextInput::new("swift", "SWIFT/BIC...", cx));
        let payment_terms = cx.new(|cx| NumberInput::new("payment-terms", "14", cx).integer_only());
        let notes = cx.new(|cx| TextArea::new("notes", "Poznámky...", cx).with_rows(4));
        let is_favorite = cx.new(|_cx| Checkbox::new("is-favorite", "Oblíbený kontakt", false));

        // Load the contact data
        let svc = service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { svc.get_by_id(id) })
                .await;
            this.update(cx, |this, cx| {
                match result {
                    Ok(contact) => this.populate_from_contact(&contact, cx),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                this.loading = false;
                cx.notify();
            })
            .ok();
        })
        .detach();

        Self {
            service,
            is_edit: true,
            contact_id: Some(id),
            saving: false,
            loading: true,
            error: None,
            name,
            contact_type,
            ico,
            dic,
            street,
            city,
            zip,
            country,
            email,
            phone,
            web,
            bank_account,
            bank_code,
            iban,
            swift,
            payment_terms,
            notes,
            is_favorite,
        }
    }

    /// Populate all form fields from an existing Contact.
    fn populate_from_contact(&mut self, contact: &Contact, cx: &mut Context<Self>) {
        self.name.update(cx, |t, cx| t.set_value(&contact.name, cx));
        self.contact_type.update(cx, |s, cx| {
            s.set_selected_value(&contact.contact_type.to_string(), cx);
        });
        self.ico.update(cx, |t, cx| t.set_value(&contact.ico, cx));
        self.dic.update(cx, |t, cx| t.set_value(&contact.dic, cx));
        self.street
            .update(cx, |t, cx| t.set_value(&contact.street, cx));
        self.city.update(cx, |t, cx| t.set_value(&contact.city, cx));
        self.zip.update(cx, |t, cx| t.set_value(&contact.zip, cx));
        self.country
            .update(cx, |t, cx| t.set_value(&contact.country, cx));
        self.email
            .update(cx, |t, cx| t.set_value(&contact.email, cx));
        self.phone
            .update(cx, |t, cx| t.set_value(&contact.phone, cx));
        self.web.update(cx, |t, cx| t.set_value(&contact.web, cx));
        self.bank_account
            .update(cx, |t, cx| t.set_value(&contact.bank_account, cx));
        self.bank_code
            .update(cx, |t, cx| t.set_value(&contact.bank_code, cx));
        self.iban.update(cx, |t, cx| t.set_value(&contact.iban, cx));
        self.swift
            .update(cx, |t, cx| t.set_value(&contact.swift, cx));
        self.payment_terms.update(cx, |t, cx| {
            t.set_value(contact.payment_terms_days.to_string(), cx);
        });
        self.notes
            .update(cx, |t, cx| t.set_value(&contact.notes, cx));
        self.is_favorite.update(cx, |c, cx| {
            c.set_checked(contact.is_favorite, cx);
        });
    }

    /// Validate inputs and save the contact (create or update).
    fn save(&mut self, cx: &mut Context<Self>) {
        if self.saving {
            return;
        }

        // Validate name
        let name = self.name.read(cx).value().to_string();
        if name.trim().is_empty() {
            self.error = Some("Zadejte název kontaktu".into());
            cx.notify();
            return;
        }

        // Read contact type
        let contact_type_str = self
            .contact_type
            .read(cx)
            .selected_value()
            .unwrap_or("company")
            .to_string();
        let contact_type = match contact_type_str.as_str() {
            "individual" => ContactType::Individual,
            _ => ContactType::Company,
        };

        // Read all fields
        let ico = self.ico.read(cx).value().to_string();
        let dic = self.dic.read(cx).value().to_string();
        let street = self.street.read(cx).value().to_string();
        let city = self.city.read(cx).value().to_string();
        let zip = self.zip.read(cx).value().to_string();
        let country = self.country.read(cx).value().to_string();
        let email = self.email.read(cx).value().to_string();
        let phone = self.phone.read(cx).value().to_string();
        let web = self.web.read(cx).value().to_string();
        let bank_account = self.bank_account.read(cx).value().to_string();
        let bank_code = self.bank_code.read(cx).value().to_string();
        let iban = self.iban.read(cx).value().to_string();
        let swift = self.swift.read(cx).value().to_string();
        let payment_terms_str = self.payment_terms.read(cx).value().to_string();
        let payment_terms_days: i32 = payment_terms_str.parse().unwrap_or(14);
        let notes = self.notes.read(cx).value().to_string();
        let is_favorite = self.is_favorite.read(cx).is_checked();

        let now = Local::now().naive_local();
        let mut contact = Contact {
            id: self.contact_id.unwrap_or(0),
            contact_type,
            name,
            ico,
            dic,
            street,
            city,
            zip,
            country,
            email,
            phone,
            web,
            bank_account,
            bank_code,
            iban,
            swift,
            payment_terms_days,
            tags: String::new(),
            notes,
            is_favorite,
            vat_unreliable_at: None,
            created_at: now,
            updated_at: now,
            deleted_at: None,
        };

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.service.clone();
        let is_edit = self.is_edit;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    if is_edit {
                        service.update(&mut contact)?;
                    } else {
                        service.create(&mut contact)?;
                    }
                    Ok::<i64, DomainError>(contact.id)
                })
                .await;
            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(id) => cx.emit(NavigateEvent(Route::ContactDetail(id))),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }
}

impl Render for ContactFormView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let title = if self.is_edit {
            "Upravit kontakt".to_string()
        } else {
            "Nový kontakt".to_string()
        };

        let mut outer = div()
            .id("contact-form-scroll")
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
                    .child("Načítání kontaktu..."),
            );
        }

        // Card 1: Zakladni udaje (name, type, ico, dic)
        outer = outer.child(render_card(
            "Základní údaje",
            div()
                .flex()
                .flex_col()
                .gap_4()
                // Row 1: name (flex-1), contact_type (w-48)
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(
                            div()
                                .flex_1()
                                .child(render_form_field("Název", self.name.clone())),
                        )
                        .child(div().w(px(192.0)).child(render_labeled_field(
                            "Typ kontaktu",
                            self.contact_type.clone(),
                        ))),
                )
                // Row 2: ico, dic
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(render_form_field("IČO", self.ico.clone()))
                        .child(render_form_field("DIČ", self.dic.clone())),
                ),
        ));

        // Card 2: Adresa (street, city, zip, country)
        outer = outer.child(render_card(
            "Adresa",
            div()
                .flex()
                .flex_col()
                .gap_4()
                // Row 1: street (flex-1)
                .child(
                    div().flex().gap_4().child(
                        div()
                            .flex_1()
                            .child(render_form_field("Ulice", self.street.clone())),
                    ),
                )
                // Row 2: city, zip, country
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(
                            div()
                                .flex_1()
                                .child(render_form_field("Město", self.city.clone())),
                        )
                        .child(render_form_field("PSČ", self.zip.clone()))
                        .child(render_form_field("Země", self.country.clone())),
                ),
        ));

        // Card 3: Kontaktni udaje (email, phone, web)
        outer = outer.child(render_card(
            "Kontaktní údaje",
            div().flex().gap_4().child(
                div()
                    .flex()
                    .flex_1()
                    .gap_4()
                    .child(
                        div()
                            .flex_1()
                            .child(render_form_field("Email", self.email.clone())),
                    )
                    .child(
                        div()
                            .flex_1()
                            .child(render_form_field("Telefon", self.phone.clone())),
                    )
                    .child(
                        div()
                            .flex_1()
                            .child(render_form_field("Web", self.web.clone())),
                    ),
            ),
        ));

        // Card 4: Bankovni udaje (bank_account, bank_code, iban, swift)
        outer = outer.child(render_card(
            "Bankovní údaje",
            div()
                .flex()
                .flex_col()
                .gap_4()
                // Row 1: bank_account, bank_code
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(
                            div()
                                .flex_1()
                                .child(render_form_field("Číslo účtu", self.bank_account.clone())),
                        )
                        .child(render_form_field("Kód banky", self.bank_code.clone())),
                )
                // Row 2: iban, swift
                .child(
                    div()
                        .flex()
                        .gap_4()
                        .child(
                            div()
                                .flex_1()
                                .child(render_form_field("IBAN", self.iban.clone())),
                        )
                        .child(render_form_field("SWIFT/BIC", self.swift.clone())),
                ),
        ));

        // Card 5: Dalsi (payment_terms, is_favorite, notes)
        outer = outer.child(render_card(
            "Další",
            div()
                .flex()
                .flex_col()
                .gap_4()
                // Row 1: payment_terms (w-32), is_favorite checkbox
                .child(
                    div()
                        .flex()
                        .items_end()
                        .gap_4()
                        .child(div().w(px(128.0)).child(render_labeled_field(
                            "Splatnost (dny)",
                            self.payment_terms.clone(),
                        )))
                        .child(div().pb_2().child(self.is_favorite.clone())),
                )
                // Row 2: notes
                .child(render_labeled_field("Poznámky", self.notes.clone())),
        ));

        // Button bar
        let cancel_btn = render_button(
            "cancel-btn",
            "Zrušit",
            ButtonVariant::Secondary,
            self.saving,
            false,
            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                cx.emit(NavigateEvent(Route::ContactList));
            }),
        );

        let save_btn = render_button(
            "save-btn",
            "Uložit",
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
