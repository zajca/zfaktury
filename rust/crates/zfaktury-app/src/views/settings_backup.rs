use gpui::*;

use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;

/// Backup settings view showing backup history.
pub struct SettingsBackupView;

impl SettingsBackupView {
    pub fn new() -> Self {
        Self
    }
}

impl EventEmitter<NavigateEvent> for SettingsBackupView {}

impl Render for SettingsBackupView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("settings-backup-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll();

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
                        .child("Zalohy dat"),
                )
                .child(
                    div()
                        .px_4()
                        .py_2()
                        .bg(rgb(ZfColors::ACCENT))
                        .rounded_md()
                        .text_sm()
                        .font_weight(FontWeight::MEDIUM)
                        .text_color(rgb(0xffffff))
                        .cursor_pointer()
                        .hover(|s| s.bg(rgb(ZfColors::ACCENT_HOVER)))
                        .child("Vytvorit zalohu"),
                ),
        );

        // Table
        let mut table = div()
            .flex()
            .flex_col()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .overflow_hidden();

        table = table.child(
            div()
                .flex()
                .px_4()
                .py_3()
                .text_xs()
                .font_weight(FontWeight::MEDIUM)
                .text_color(rgb(ZfColors::TEXT_MUTED))
                .border_b_1()
                .border_color(rgb(ZfColors::BORDER))
                .child(div().w(px(150.0)).child("Datum"))
                .child(div().w(px(112.0)).child("Typ"))
                .child(div().w(px(112.0)).text_right().child("Velikost"))
                .child(div().flex_1().child("Soubor"))
                .child(div().w_20().text_right().child("Stav")),
        );

        table = table.child(
            div()
                .px_4()
                .py_8()
                .text_sm()
                .text_color(rgb(ZfColors::TEXT_MUTED))
                .child("Zadne zalohy. Vytvorte prvni zalohu dat."),
        );

        content.child(table)
    }
}
