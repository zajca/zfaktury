use std::collections::HashMap;
use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::SettingsService;
use zfaktury_domain::{
    SETTING_BANK_ACCOUNT, SETTING_BANK_CODE, SETTING_CITY, SETTING_COMPANY_NAME, SETTING_DIC,
    SETTING_EMAIL, SETTING_FIRST_NAME, SETTING_HOUSE_NUMBER, SETTING_IBAN, SETTING_ICO,
    SETTING_LAST_NAME, SETTING_PHONE, SETTING_STREET, SETTING_SWIFT, SETTING_VAT_REGISTERED,
    SETTING_ZIP,
};

use crate::theme::ZfColors;

/// Company settings form view.
pub struct SettingsFirmaView {
    service: Arc<SettingsService>,
    loading: bool,
    error: Option<String>,
    settings: HashMap<String, String>,
}

impl SettingsFirmaView {
    pub fn new(service: Arc<SettingsService>, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            loading: true,
            error: None,
            settings: HashMap::new(),
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.get_all() })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(settings) => this.settings = settings,
                    Err(e) => this.error = Some(format!("Chyba pri nacitani nastaveni: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn get_setting(&self, key: &str) -> String {
        self.settings.get(key).cloned().unwrap_or_default()
    }

    fn render_field_row(&self, label: &str, value: &str) -> Div {
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
                    .px_3()
                    .py_2()
                    .bg(rgb(ZfColors::SURFACE))
                    .border_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .rounded_md()
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

    fn render_section(&self, title: &str, fields: Vec<(&str, &str)>) -> Div {
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
            section = section.child(self.render_field_row(label, value));
        }

        section
    }
}

impl Render for SettingsFirmaView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("settings-firma-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll();

        content = content.child(
            div()
                .text_xl()
                .font_weight(FontWeight::SEMIBOLD)
                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                .child("Nastaveni firmy"),
        );

        if self.loading {
            return content.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Nacitani..."),
            );
        }

        if let Some(ref error) = self.error {
            return content.child(
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

        let company_name = self.get_setting(SETTING_COMPANY_NAME);
        let first_name = self.get_setting(SETTING_FIRST_NAME);
        let last_name = self.get_setting(SETTING_LAST_NAME);
        let ico = self.get_setting(SETTING_ICO);
        let dic = self.get_setting(SETTING_DIC);
        let vat_reg = self.get_setting(SETTING_VAT_REGISTERED);
        let street = self.get_setting(SETTING_STREET);
        let house = self.get_setting(SETTING_HOUSE_NUMBER);
        let city = self.get_setting(SETTING_CITY);
        let zip = self.get_setting(SETTING_ZIP);
        let email = self.get_setting(SETTING_EMAIL);
        let phone = self.get_setting(SETTING_PHONE);
        let bank_acct = self.get_setting(SETTING_BANK_ACCOUNT);
        let bank_code = self.get_setting(SETTING_BANK_CODE);
        let iban = self.get_setting(SETTING_IBAN);
        let swift = self.get_setting(SETTING_SWIFT);

        content = content
            .child(self.render_section(
                "Zakladni udaje",
                vec![
                    ("Nazev firmy", &company_name),
                    ("Jmeno", &first_name),
                    ("Prijmeni", &last_name),
                    ("ICO", &ico),
                    ("DIC", &dic),
                    ("Platce DPH", if vat_reg == "true" { "Ano" } else { "Ne" }),
                ],
            ))
            .child(self.render_section(
                "Adresa",
                vec![
                    ("Ulice", &street),
                    ("Cislo popisne", &house),
                    ("Mesto", &city),
                    ("PSC", &zip),
                ],
            ))
            .child(self.render_section("Kontakt", vec![("Email", &email), ("Telefon", &phone)]))
            .child(self.render_section(
                "Bankovni udaje",
                vec![
                    ("Cislo uctu", &bank_acct),
                    ("Kod banky", &bank_code),
                    ("IBAN", &iban),
                    ("SWIFT/BIC", &swift),
                ],
            ));

        content
    }
}
