use gpui::*;

use crate::components::text_input::{TextChanged, TextInput};
use crate::theme::ZfColors;

/// Event emitted when the date value changes. Contains ISO date string (YYYY-MM-DD) or empty.
pub struct DateChanged(pub String);

/// Date input component that accepts Czech format (dd.mm.yyyy) and stores as ISO.
pub struct DateInput {
    input: Entity<TextInput>,
    iso_value: String,
    error: Option<String>,
}

impl EventEmitter<DateChanged> for DateInput {}

impl DateInput {
    pub fn new(id: impl Into<ElementId>, cx: &mut Context<Self>) -> Self {
        let input = cx.new(|cx| TextInput::new(id, "dd.mm.rrrr", cx));

        cx.subscribe(
            &input,
            |this: &mut Self, _input, event: &TextChanged, cx| {
                this.parse_and_emit(&event.0, cx);
            },
        )
        .detach();

        Self {
            input,
            iso_value: String::new(),
            error: None,
        }
    }

    pub fn iso_value(&self) -> &str {
        &self.iso_value
    }

    pub fn set_iso_value(&mut self, iso: &str, cx: &mut Context<Self>) {
        self.iso_value = iso.to_string();
        // Convert ISO to Czech display format.
        if let Some(czech) = iso_to_czech(iso) {
            self.input.update(cx, |input, cx| {
                input.set_value(czech, cx);
            });
            self.error = None;
        }
        cx.notify();
    }

    fn parse_and_emit(&mut self, text: &str, cx: &mut Context<Self>) {
        if text.is_empty() {
            self.iso_value.clear();
            self.error = None;
            cx.emit(DateChanged(String::new()));
            return;
        }

        match czech_to_iso(text) {
            Some(iso) => {
                self.iso_value = iso.clone();
                self.error = None;
                cx.emit(DateChanged(iso));
            }
            None => {
                self.error = Some("Neplatný formát data".to_string());
            }
        }
        cx.notify();
    }
}

impl Render for DateInput {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let mut container = div().flex().flex_col().gap_1().child(self.input.clone());

        if let Some(ref err) = self.error {
            container = container.child(
                div()
                    .text_xs()
                    .text_color(rgb(ZfColors::STATUS_RED))
                    .child(err.clone()),
            );
        }

        container
    }
}

/// Parse Czech date format "dd.mm.yyyy" to ISO "YYYY-MM-DD".
fn czech_to_iso(text: &str) -> Option<String> {
    let parts: Vec<&str> = text.split('.').collect();
    if parts.len() != 3 {
        return None;
    }
    let day: u32 = parts[0].parse().ok()?;
    let month: u32 = parts[1].parse().ok()?;
    let year: i32 = parts[2].parse().ok()?;

    if !(1..=31).contains(&day) || !(1..=12).contains(&month) || !(2000..=2099).contains(&year) {
        return None;
    }

    Some(format!("{year:04}-{month:02}-{day:02}"))
}

/// Convert ISO "YYYY-MM-DD" to Czech "dd.mm.yyyy".
fn iso_to_czech(iso: &str) -> Option<String> {
    let parts: Vec<&str> = iso.split('-').collect();
    if parts.len() != 3 {
        return None;
    }
    let year: i32 = parts[0].parse().ok()?;
    let month: u32 = parts[1].parse().ok()?;
    let day: u32 = parts[2].parse().ok()?;
    Some(format!("{day}.{month}.{year}"))
}
