use std::collections::HashMap;
use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::SettingsService;
use zfaktury_domain::{
    SETTING_EMAIL, SETTING_EMAIL_ATTACH_ISDOC, SETTING_EMAIL_ATTACH_PDF, SETTING_EMAIL_BODY_TPL,
    SETTING_EMAIL_SUBJECT_TPL,
};

use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;

/// Email settings view displaying SMTP configuration.
pub struct SettingsEmailView {
    service: Arc<SettingsService>,
    loading: bool,
    error: Option<String>,
    settings: HashMap<String, String>,
}

impl SettingsEmailView {
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
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani nastaveni: {e}"));
                    }
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
}

impl EventEmitter<NavigateEvent> for SettingsEmailView {}

impl Render for SettingsEmailView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("settings-email-scroll")
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
                .child("Nastaveni emailu"),
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

        let email = self.get_setting(SETTING_EMAIL);
        let attach_pdf = self.get_setting(SETTING_EMAIL_ATTACH_PDF);
        let attach_isdoc = self.get_setting(SETTING_EMAIL_ATTACH_ISDOC);
        let subject_tpl = self.get_setting(SETTING_EMAIL_SUBJECT_TPL);
        let body_tpl = self.get_setting(SETTING_EMAIL_BODY_TPL);

        // Email settings section
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
                    .child("Odchozi email"),
            );

        section = section.child(self.render_field_row("Email odesilatele", &email));
        section = section.child(self.render_field_row(
            "Priloha PDF",
            if attach_pdf == "true" { "Ano" } else { "Ne" },
        ));
        section = section.child(self.render_field_row(
            "Priloha ISDOC",
            if attach_isdoc == "true" { "Ano" } else { "Ne" },
        ));
        section = section.child(self.render_field_row("Sablona predmetu", &subject_tpl));
        section = section.child(self.render_field_row("Sablona tela", &body_tpl));

        content = content.child(section);

        // Info note
        content = content.child(
            div()
                .p_4()
                .bg(rgb(ZfColors::SURFACE))
                .rounded_md()
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
                .text_sm()
                .text_color(rgb(ZfColors::TEXT_MUTED))
                .child("SMTP konfigurace se nastavuje v config.toml souboru."),
        );

        content
    }
}
