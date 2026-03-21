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

use crate::components::button::{ButtonVariant, render_button};
use crate::components::text_input::TextInput;
use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;

/// Field definition for settings form.
struct FieldDef {
    key: &'static str,
    label: &'static str,
    placeholder: &'static str,
}

const BASIC_FIELDS: &[FieldDef] = &[
    FieldDef {
        key: SETTING_COMPANY_NAME,
        label: "Nazev firmy",
        placeholder: "Nazev firmy...",
    },
    FieldDef {
        key: SETTING_FIRST_NAME,
        label: "Jmeno",
        placeholder: "Jmeno...",
    },
    FieldDef {
        key: SETTING_LAST_NAME,
        label: "Prijmeni",
        placeholder: "Prijmeni...",
    },
    FieldDef {
        key: SETTING_ICO,
        label: "ICO",
        placeholder: "ICO...",
    },
    FieldDef {
        key: SETTING_DIC,
        label: "DIC",
        placeholder: "DIC...",
    },
    FieldDef {
        key: SETTING_VAT_REGISTERED,
        label: "Platce DPH",
        placeholder: "true/false",
    },
];

const ADDRESS_FIELDS: &[FieldDef] = &[
    FieldDef {
        key: SETTING_STREET,
        label: "Ulice",
        placeholder: "Ulice...",
    },
    FieldDef {
        key: SETTING_HOUSE_NUMBER,
        label: "Cislo popisne",
        placeholder: "Cislo popisne...",
    },
    FieldDef {
        key: SETTING_CITY,
        label: "Mesto",
        placeholder: "Mesto...",
    },
    FieldDef {
        key: SETTING_ZIP,
        label: "PSC",
        placeholder: "PSC...",
    },
];

const CONTACT_FIELDS: &[FieldDef] = &[
    FieldDef {
        key: SETTING_EMAIL,
        label: "Email",
        placeholder: "Email...",
    },
    FieldDef {
        key: SETTING_PHONE,
        label: "Telefon",
        placeholder: "Telefon...",
    },
];

const BANK_FIELDS: &[FieldDef] = &[
    FieldDef {
        key: SETTING_BANK_ACCOUNT,
        label: "Cislo uctu",
        placeholder: "Cislo uctu...",
    },
    FieldDef {
        key: SETTING_BANK_CODE,
        label: "Kod banky",
        placeholder: "Kod banky...",
    },
    FieldDef {
        key: SETTING_IBAN,
        label: "IBAN",
        placeholder: "IBAN...",
    },
    FieldDef {
        key: SETTING_SWIFT,
        label: "SWIFT/BIC",
        placeholder: "SWIFT/BIC...",
    },
];

/// Company settings form view with edit/save toggle.
pub struct SettingsFirmaView {
    service: Arc<SettingsService>,
    loading: bool,
    saving: bool,
    editing: bool,
    error: Option<String>,
    settings: HashMap<String, String>,
    /// TextInput entities keyed by setting key (only created when editing).
    inputs: HashMap<String, Entity<TextInput>>,
}

impl SettingsFirmaView {
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

    fn all_field_defs() -> Vec<&'static FieldDef> {
        BASIC_FIELDS
            .iter()
            .chain(ADDRESS_FIELDS.iter())
            .chain(CONTACT_FIELDS.iter())
            .chain(BANK_FIELDS.iter())
            .collect()
    }

    fn start_editing(&mut self, cx: &mut Context<Self>) {
        self.editing = true;
        self.error = None;
        self.inputs.clear();

        for field in Self::all_field_defs() {
            let value = self.get_setting(field.key);
            let key = field.key.to_string();
            let placeholder = field.placeholder;
            let input = cx.new(|cx| {
                let mut t = TextInput::new(
                    SharedString::from(format!("firma-{}", key)),
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

        // Collect all input values into a HashMap.
        let mut new_settings = HashMap::new();
        for field in Self::all_field_defs() {
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
                        // Update local settings from inputs.
                        for field in Self::all_field_defs() {
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

    fn render_section_readonly(&self, title: &str, fields: &[FieldDef]) -> Div {
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

        for field in fields {
            let value = self.get_setting(field.key);
            let display = if field.key == SETTING_VAT_REGISTERED {
                if value == "true" {
                    "Ano".to_string()
                } else {
                    "Ne".to_string()
                }
            } else {
                value
            };
            section = section.child(self.render_field_row(field.label, &display));
        }

        section
    }

    fn render_section_editable(&self, title: &str, fields: &[FieldDef]) -> Div {
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

        for field in fields {
            section = section.child(self.render_edit_field_row(field.label, field.key));
        }

        section
    }
}

impl EventEmitter<NavigateEvent> for SettingsFirmaView {}

impl Render for SettingsFirmaView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("settings-firma-scroll")
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
                .child("Nastaveni firmy"),
        );

        if !self.loading {
            if self.editing {
                title_bar = title_bar.child(
                    div()
                        .flex()
                        .gap_2()
                        .child(render_button(
                            "firma-cancel-btn",
                            "Zrusit",
                            ButtonVariant::Secondary,
                            self.saving,
                            false,
                            cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                this.cancel_editing(cx);
                            }),
                        ))
                        .child(render_button(
                            "firma-save-btn",
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
                    "firma-edit-btn",
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

        if self.editing {
            content = content
                .child(self.render_section_editable("Zakladni udaje", BASIC_FIELDS))
                .child(self.render_section_editable("Adresa", ADDRESS_FIELDS))
                .child(self.render_section_editable("Kontakt", CONTACT_FIELDS))
                .child(self.render_section_editable("Bankovni udaje", BANK_FIELDS));
        } else {
            content = content
                .child(self.render_section_readonly("Zakladni udaje", BASIC_FIELDS))
                .child(self.render_section_readonly("Adresa", ADDRESS_FIELDS))
                .child(self.render_section_readonly("Kontakt", CONTACT_FIELDS))
                .child(self.render_section_readonly("Bankovni udaje", BANK_FIELDS));
        }

        content
    }
}
