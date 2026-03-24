use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::SettingsService;
use zfaktury_domain::PDFSettings;

use crate::components::button::{ButtonVariant, render_button};
use crate::components::checkbox::Checkbox;
use crate::components::text_input::TextInput;
use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;

/// PDF template settings view with edit/save toggle.
///
/// Follows the same pattern as SettingsFirmaView: view mode shows key-value pairs,
/// edit mode shows input fields.
pub struct SettingsPdfView {
    settings_service: Arc<SettingsService>,
    loading: bool,
    saving: bool,
    editing: bool,
    error: Option<String>,
    success: Option<String>,

    // Current settings (for view mode display)
    current_settings: PDFSettings,

    // Edit mode inputs (created when entering edit mode)
    accent_color_input: Option<Entity<TextInput>>,
    footer_text_input: Option<Entity<TextInput>>,
    font_size_input: Option<Entity<TextInput>>,
    logo_path_input: Option<Entity<TextInput>>,
    show_qr_input: Option<Entity<Checkbox>>,
    show_bank_details_input: Option<Entity<Checkbox>>,
}

impl SettingsPdfView {
    pub fn new(settings_service: Arc<SettingsService>, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            settings_service,
            loading: true,
            saving: false,
            editing: false,
            error: None,
            success: None,
            current_settings: PDFSettings::default(),
            accent_color_input: None,
            footer_text_input: None,
            font_size_input: None,
            logo_path_input: None,
            show_qr_input: None,
            show_bank_details_input: None,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.settings_service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.get_pdf_settings() })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(settings) => this.current_settings = settings,
                    Err(e) => {
                        this.error = Some(format!("Chyba při načítání nastavení PDF: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn start_editing(&mut self, cx: &mut Context<Self>) {
        self.editing = true;
        self.error = None;
        self.success = None;

        let settings = &self.current_settings;

        let accent_color_val = settings
            .accent_color
            .clone()
            .unwrap_or_else(|| "#2563eb".to_string());
        let footer_text_val = settings.footer_text.clone().unwrap_or_default();
        let font_size_val = settings
            .font_size
            .map(|f| f.to_string())
            .unwrap_or_default();
        let logo_path_val = settings.logo_path.clone().unwrap_or_default();

        self.accent_color_input = Some(cx.new(|cx| {
            let mut t = TextInput::new("pdf-accent-color", "#2563eb", cx);
            t.set_value(accent_color_val, cx);
            t
        }));

        self.footer_text_input = Some(cx.new(|cx| {
            let mut t = TextInput::new("pdf-footer-text", "Text v patce faktury...", cx);
            t.set_value(footer_text_val, cx);
            t
        }));

        self.font_size_input = Some(cx.new(|cx| {
            let mut t = TextInput::new("pdf-font-size", "Velikost písma (např. 10.0)...", cx);
            t.set_value(font_size_val, cx);
            t
        }));

        self.logo_path_input = Some(cx.new(|cx| {
            let mut t = TextInput::new("pdf-logo-path", "Cesta k logu...", cx);
            t.set_value(logo_path_val, cx);
            t
        }));

        self.show_qr_input =
            Some(cx.new(|_cx| {
                Checkbox::new("pdf-show-qr", "Zobrazit QR platební kód", settings.show_qr)
            }));

        self.show_bank_details_input = Some(cx.new(|_cx| {
            Checkbox::new(
                "pdf-show-bank-details",
                "Zobrazit bankovní údaje",
                settings.show_bank_details,
            )
        }));

        cx.notify();
    }

    fn cancel_editing(&mut self, cx: &mut Context<Self>) {
        self.editing = false;
        self.error = None;
        self.clear_inputs();
        cx.notify();
    }

    fn clear_inputs(&mut self) {
        self.accent_color_input = None;
        self.footer_text_input = None;
        self.font_size_input = None;
        self.logo_path_input = None;
        self.show_qr_input = None;
        self.show_bank_details_input = None;
    }

    fn save(&mut self, cx: &mut Context<Self>) {
        if self.saving {
            return;
        }

        // Read input values.
        let accent_color = self
            .accent_color_input
            .as_ref()
            .map(|i| i.read(cx).value().to_string())
            .unwrap_or_default();
        let footer_text = self
            .footer_text_input
            .as_ref()
            .map(|i| i.read(cx).value().to_string())
            .unwrap_or_default();
        let font_size_str = self
            .font_size_input
            .as_ref()
            .map(|i| i.read(cx).value().to_string())
            .unwrap_or_default();
        let logo_path = self
            .logo_path_input
            .as_ref()
            .map(|i| i.read(cx).value().to_string())
            .unwrap_or_default();
        let show_qr = self
            .show_qr_input
            .as_ref()
            .map(|i| i.read(cx).is_checked())
            .unwrap_or(true);
        let show_bank_details = self
            .show_bank_details_input
            .as_ref()
            .map(|i| i.read(cx).is_checked())
            .unwrap_or(true);

        // Validate hex color format if provided.
        if !accent_color.is_empty() && !is_valid_hex_color(&accent_color) {
            self.error =
                Some("Neplatný formát barvy. Použijte hex formát, např. #2563eb".to_string());
            cx.notify();
            return;
        }

        // Parse font size if provided.
        let font_size = if font_size_str.is_empty() {
            None
        } else {
            match font_size_str.parse::<f32>() {
                Ok(f) if f > 0.0 => Some(f),
                _ => {
                    self.error = Some("Neplatná velikost písma. Zadejte kladné číslo.".to_string());
                    cx.notify();
                    return;
                }
            }
        };

        let pdf_settings = PDFSettings {
            accent_color: if accent_color.is_empty() {
                None
            } else {
                Some(accent_color)
            },
            font_size,
            footer_text: if footer_text.is_empty() {
                None
            } else {
                Some(footer_text)
            },
            logo_path: if logo_path.is_empty() {
                None
            } else {
                Some(logo_path)
            },
            show_qr,
            show_bank_details,
        };

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.settings_service.clone();
        let settings_clone = pdf_settings.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.save_pdf_settings(&settings_clone) })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok(()) => {
                        this.current_settings = pdf_settings;
                        this.editing = false;
                        this.success = Some("Nastavení PDF uloženo".to_string());
                        this.clear_inputs();
                    }
                    Err(e) => this.error = Some(format!("Chyba při ukládání: {e}")),
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

    fn render_edit_field_row(&self, label: &str, input: Option<&Entity<TextInput>>) -> Div {
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
                .child(div().flex_1().child(input.clone())),
            None => self.render_field_row(label, ""),
        }
    }

    fn render_checkbox_row(&self, checkbox: Option<&Entity<Checkbox>>) -> Div {
        match checkbox {
            Some(cb) => div().py_1().child(cb.clone()),
            None => div(),
        }
    }

    fn render_color_preview(&self) -> Div {
        let color_str = self
            .current_settings
            .accent_color
            .as_deref()
            .unwrap_or("#2563eb");

        let color_val = parse_hex_color(color_str).unwrap_or(0x2563eb);

        div()
            .flex()
            .items_center()
            .gap_2()
            .child(
                div()
                    .w(px(24.0))
                    .h(px(24.0))
                    .rounded(px(4.0))
                    .bg(rgb(color_val))
                    .border_1()
                    .border_color(rgb(ZfColors::BORDER)),
            )
            .child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(color_str.to_string()),
            )
    }
}

/// Validate hex color format: #RRGGBB or #RGB.
fn is_valid_hex_color(s: &str) -> bool {
    if !s.starts_with('#') {
        return false;
    }
    let hex = &s[1..];
    (hex.len() == 3 || hex.len() == 6) && hex.chars().all(|c| c.is_ascii_hexdigit())
}

/// Parse hex color string to u32 (for GPUI rgb()).
fn parse_hex_color(s: &str) -> Option<u32> {
    let hex = s.strip_prefix('#')?;
    match hex.len() {
        6 => u32::from_str_radix(hex, 16).ok(),
        3 => {
            let expanded: String = hex
                .chars()
                .flat_map(|c| std::iter::repeat_n(c, 2))
                .collect();
            u32::from_str_radix(&expanded, 16).ok()
        }
        _ => None,
    }
}

impl EventEmitter<NavigateEvent> for SettingsPdfView {}

impl Render for SettingsPdfView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("settings-pdf-scroll")
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
                .child("Nastavení PDF šablony"),
        );

        if !self.loading {
            if self.editing {
                title_bar = title_bar.child(
                    div()
                        .flex()
                        .gap_2()
                        .child(render_button(
                            "pdf-cancel-btn",
                            "Zrušit",
                            ButtonVariant::Secondary,
                            self.saving,
                            false,
                            cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                this.cancel_editing(cx);
                            }),
                        ))
                        .child(render_button(
                            "pdf-save-btn",
                            "Uložit",
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
                    "pdf-edit-btn",
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

        // Loading state
        if self.loading {
            return content.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Načítání..."),
            );
        }

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

        // Error message
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
            // Edit mode: show input fields
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
                        .child("Vzhled"),
                );

            section = section.child(
                self.render_edit_field_row("Barva akcentu", self.accent_color_input.as_ref()),
            );
            section = section
                .child(self.render_edit_field_row("Text v patce", self.footer_text_input.as_ref()));
            section = section
                .child(self.render_edit_field_row("Velikost písma", self.font_size_input.as_ref()));
            section = section
                .child(self.render_edit_field_row("Cesta k logu", self.logo_path_input.as_ref()));

            content = content.child(section);

            // Checkboxes section
            let mut cb_section = div()
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
                        .child("Obsah"),
                );

            cb_section = cb_section.child(self.render_checkbox_row(self.show_qr_input.as_ref()));
            cb_section =
                cb_section.child(self.render_checkbox_row(self.show_bank_details_input.as_ref()));

            content = content.child(cb_section);
        } else {
            // View mode: display current settings
            let settings = &self.current_settings;

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
                        .child("Vzhled"),
                );

            // Color with preview
            section = section.child(
                div()
                    .flex()
                    .items_center()
                    .gap_4()
                    .child(
                        div()
                            .w_40()
                            .text_sm()
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .child("Barva akcentu"),
                    )
                    .child(self.render_color_preview()),
            );

            section = section.child(self.render_field_row(
                "Text v patce",
                settings.footer_text.as_deref().unwrap_or(""),
            ));
            section = section.child(
                self.render_field_row(
                    "Velikost písma",
                    &settings
                        .font_size
                        .map(|f| f.to_string())
                        .unwrap_or_default(),
                ),
            );
            section = section.child(
                self.render_field_row("Cesta k logu", settings.logo_path.as_deref().unwrap_or("")),
            );

            content = content.child(section);

            // Boolean settings section
            let mut bool_section = div()
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
                        .child("Obsah"),
                );

            bool_section = bool_section.child(self.render_field_row(
                "QR platební kód",
                if settings.show_qr { "Ano" } else { "Ne" },
            ));
            bool_section = bool_section.child(self.render_field_row(
                "Bankovní údaje",
                if settings.show_bank_details {
                    "Ano"
                } else {
                    "Ne"
                },
            ));

            content = content.child(bool_section);
        }

        content
    }
}
