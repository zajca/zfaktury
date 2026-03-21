use std::collections::HashMap;
use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::SettingsService;
use zfaktury_domain::{
    SETTING_EMAIL, SETTING_EMAIL_ATTACH_ISDOC, SETTING_EMAIL_ATTACH_PDF, SETTING_EMAIL_BODY_TPL,
    SETTING_EMAIL_SUBJECT_TPL,
};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::text_input::TextInput;
use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;

/// Field definition for email settings form.
struct FieldDef {
    key: &'static str,
    label: &'static str,
    placeholder: &'static str,
}

const EMAIL_FIELDS: &[FieldDef] = &[
    FieldDef {
        key: SETTING_EMAIL,
        label: "Email odesilatele",
        placeholder: "email@example.com...",
    },
    FieldDef {
        key: SETTING_EMAIL_ATTACH_PDF,
        label: "Priloha PDF",
        placeholder: "true/false",
    },
    FieldDef {
        key: SETTING_EMAIL_ATTACH_ISDOC,
        label: "Priloha ISDOC",
        placeholder: "true/false",
    },
    FieldDef {
        key: SETTING_EMAIL_SUBJECT_TPL,
        label: "Sablona predmetu",
        placeholder: "Faktura {{cislo}}...",
    },
    FieldDef {
        key: SETTING_EMAIL_BODY_TPL,
        label: "Sablona tela",
        placeholder: "Dobry den, zasilam fakturu...",
    },
];

/// Boolean display keys for read-only mode.
const BOOLEAN_KEYS: &[&str] = &[SETTING_EMAIL_ATTACH_PDF, SETTING_EMAIL_ATTACH_ISDOC];

/// Email settings view with edit/save toggle.
pub struct SettingsEmailView {
    service: Arc<SettingsService>,
    loading: bool,
    saving: bool,
    editing: bool,
    error: Option<String>,
    settings: HashMap<String, String>,
    inputs: HashMap<String, Entity<TextInput>>,
}

impl SettingsEmailView {
    pub fn new(service: Arc<SettingsService>, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            loading: true,
            saving: false,
            editing: false,
            error: None,
            settings: HashMap::new(),
            inputs: HashMap::new(),
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

    fn start_editing(&mut self, cx: &mut Context<Self>) {
        self.editing = true;
        self.error = None;
        self.inputs.clear();

        for field in EMAIL_FIELDS {
            let value = self.get_setting(field.key);
            let key = field.key.to_string();
            let placeholder = field.placeholder;
            let input = cx.new(|cx| {
                let mut t = TextInput::new(
                    SharedString::from(format!("email-{}", key)),
                    placeholder,
                    cx,
                );
                t.set_value(value, cx);
                t
            });
            self.inputs.insert(key, input);
        }
        cx.notify();
    }

    fn cancel_editing(&mut self, cx: &mut Context<Self>) {
        self.editing = false;
        self.error = None;
        self.inputs.clear();
        cx.notify();
    }

    fn save(&mut self, cx: &mut Context<Self>) {
        if self.saving {
            return;
        }

        let mut new_settings = HashMap::new();
        for field in EMAIL_FIELDS {
            if let Some(input) = self.inputs.get(field.key) {
                let val = input.read(cx).value().to_string();
                new_settings.insert(field.key.to_string(), val);
            }
        }

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.set_bulk(&new_settings) })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(()) => {
                        for field in EMAIL_FIELDS {
                            if let Some(input) = this.inputs.get(field.key) {
                                let val = input.read(cx).value().to_string();
                                this.settings.insert(field.key.to_string(), val);
                            }
                        }
                        this.editing = false;
                        this.inputs.clear();
                    }
                    Err(e) => this.error = Some(format!("Chyba pri ukladani: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
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

    fn render_edit_field_row(&self, label: &str, key: &str) -> Div {
        let input = self.inputs.get(key).cloned();
        match input {
            Some(input) => div()
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
                .child(div().flex_1().child(input)),
            None => self.render_field_row(label, ""),
        }
    }

    fn is_boolean_key(key: &str) -> bool {
        BOOLEAN_KEYS.contains(&key)
    }
}

impl EventEmitter<NavigateEvent> for SettingsEmailView {}

impl Render for SettingsEmailView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("settings-email-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll();

        // Title bar with buttons
        let mut title_bar = div().flex().items_center().justify_between().child(
            div()
                .text_xl()
                .font_weight(FontWeight::SEMIBOLD)
                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                .child("Nastaveni emailu"),
        );

        if !self.loading {
            if self.editing {
                title_bar = title_bar.child(
                    div()
                        .flex()
                        .gap_2()
                        .child(render_button(
                            "email-cancel-btn",
                            "Zrusit",
                            ButtonVariant::Secondary,
                            self.saving,
                            false,
                            cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                this.cancel_editing(cx);
                            }),
                        ))
                        .child(render_button(
                            "email-save-btn",
                            "Ulozit",
                            ButtonVariant::Primary,
                            false,
                            self.saving,
                            cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                this.save(cx);
                            }),
                        )),
                );
            } else {
                title_bar = title_bar.child(render_button(
                    "email-edit-btn",
                    "Upravit",
                    ButtonVariant::Primary,
                    false,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.start_editing(cx);
                    }),
                ));
            }
        }

        content = content.child(title_bar);

        if self.loading {
            return content.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Nacitani..."),
            );
        }

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

        if self.editing {
            for field in EMAIL_FIELDS {
                section = section.child(self.render_edit_field_row(field.label, field.key));
            }
        } else {
            for field in EMAIL_FIELDS {
                let raw_value = self.get_setting(field.key);
                let display = if Self::is_boolean_key(field.key) {
                    if raw_value == "true" {
                        "Ano".to_string()
                    } else {
                        "Ne".to_string()
                    }
                } else {
                    raw_value
                };
                section = section.child(self.render_field_row(field.label, &display));
            }
        }

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
