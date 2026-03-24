use gpui::*;

use crate::theme::ZfColors;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ButtonVariant {
    Primary,
    Secondary,
    Danger,
}

/// Reusable button component with loading/disabled states.
pub fn render_button(
    id: impl Into<ElementId>,
    label: &str,
    variant: ButtonVariant,
    disabled: bool,
    loading: bool,
    on_click: impl Fn(&ClickEvent, &mut Window, &mut App) + 'static,
) -> Stateful<Div> {
    let (bg, bg_hover, text_color) = match variant {
        ButtonVariant::Primary => (ZfColors::ACCENT, ZfColors::ACCENT_HOVER, 0xffffffu32),
        ButtonVariant::Secondary => (
            ZfColors::SURFACE,
            ZfColors::SURFACE_HOVER,
            ZfColors::TEXT_PRIMARY,
        ),
        ButtonVariant::Danger => (
            ZfColors::STATUS_RED_BG,
            ZfColors::STATUS_RED,
            ZfColors::STATUS_RED,
        ),
    };

    let display_label = if loading {
        "Načítání...".to_string()
    } else {
        label.to_string()
    };

    let is_disabled = disabled || loading;

    let mut btn = div()
        .id(id.into())
        .px_4()
        .py_2()
        .rounded_md()
        .text_sm()
        .font_weight(FontWeight::MEDIUM)
        .text_color(rgb(text_color))
        .bg(rgb(bg))
        .child(display_label);

    if is_disabled {
        btn = btn.opacity(0.5);
    } else {
        btn = btn
            .cursor_pointer()
            .hover(move |s| s.bg(rgb(bg_hover)))
            .on_click(on_click);
    }

    btn
}
