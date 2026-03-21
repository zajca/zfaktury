use gpui::*;

use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;

/// Expense creation form view.
/// Currently a placeholder with the form layout structure.
pub struct ExpenseFormView {
    is_edit: bool,
    expense_id: Option<i64>,
}

impl ExpenseFormView {
    pub fn new() -> Self {
        Self {
            is_edit: false,
            expense_id: None,
        }
    }

    pub fn new_edit(id: i64) -> Self {
        Self {
            is_edit: true,
            expense_id: Some(id),
        }
    }

    fn render_form_field(&self, label: &str, placeholder: &str) -> Div {
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
            .child(
                div()
                    .px_3()
                    .py_2()
                    .bg(rgb(ZfColors::SURFACE))
                    .border_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .rounded_md()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child(placeholder.to_string()),
            )
    }
}

impl EventEmitter<NavigateEvent> for ExpenseFormView {}

impl Render for ExpenseFormView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let title = if self.is_edit {
            format!("Upravit naklad #{}", self.expense_id.unwrap_or_default())
        } else {
            "Novy naklad".to_string()
        };

        div()
            .id("expense-form-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll()
            // Title
            .child(
                div()
                    .text_xl()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(title),
            )
            // Basic info section
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
                            .child("Zakladni udaje"),
                    )
                    .child(
                        div()
                            .flex()
                            .gap_4()
                            .child(self.render_form_field("Cislo dokladu", "Napr. N2024001"))
                            .child(
                                div()
                                    .flex_1()
                                    .child(self.render_form_field("Popis", "Popis nakladu...")),
                            ),
                    )
                    .child(
                        div()
                            .flex()
                            .gap_4()
                            .child(self.render_form_field("Kategorie", "Vyberte kategorii..."))
                            .child(self.render_form_field("Datum", "dd.mm.yyyy"))
                            .child(self.render_form_field("Dodavatel", "Vyberte dodavatele...")),
                    ),
            )
            // Amount section
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
                            .child("Castka a DPH"),
                    )
                    .child(
                        div()
                            .flex()
                            .gap_4()
                            .child(self.render_form_field("Castka", "0,00"))
                            .child(self.render_form_field("Mena", "CZK"))
                            .child(self.render_form_field("DPH sazba", "21%")),
                    )
                    .child(
                        div()
                            .flex()
                            .gap_4()
                            .child(self.render_form_field("Obchodni podil", "100%"))
                            .child(self.render_form_field("Zpusob platby", "Prevod")),
                    ),
            )
            // Notes
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
                    .child(self.render_form_field("Poznamky", "Volitelne poznamky...")),
            )
            // Save button
            .child(
                div().flex().justify_end().child(
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
                        .child("Ulozit naklad"),
                ),
            )
    }
}
