use gpui::*;
use zfaktury_domain::Amount;

use crate::theme::ZfColors;

/// Event emitted when the number input value changes.
pub struct NumberChanged(pub String);

/// Numeric input component that only accepts digits, minus, and decimal separator.
pub struct NumberInput {
    id: ElementId,
    value: String,
    placeholder: SharedString,
    focus_handle: FocusHandle,
    cursor_pos: usize,
    allow_decimal: bool,
    allow_negative: bool,
}

impl EventEmitter<NumberChanged> for NumberInput {}

impl NumberInput {
    pub fn new(
        id: impl Into<ElementId>,
        placeholder: impl Into<SharedString>,
        cx: &mut Context<Self>,
    ) -> Self {
        Self {
            id: id.into(),
            value: String::new(),
            placeholder: placeholder.into(),
            focus_handle: cx.focus_handle(),
            cursor_pos: 0,
            allow_decimal: true,
            allow_negative: false,
        }
    }

    pub fn with_value(mut self, value: impl Into<String>) -> Self {
        self.value = value.into();
        self.cursor_pos = self.value.len();
        self
    }

    #[allow(dead_code)]
    pub fn with_negative(mut self) -> Self {
        self.allow_negative = true;
        self
    }

    #[allow(dead_code)]
    pub fn integer_only(mut self) -> Self {
        self.allow_decimal = false;
        self
    }

    pub fn value(&self) -> &str {
        &self.value
    }

    pub fn set_value(&mut self, value: impl Into<String>, cx: &mut Context<Self>) {
        self.value = value.into();
        self.cursor_pos = self.value.chars().count();
        cx.notify();
    }

    /// Convert the current string value to Amount (halere).
    /// Parses decimal strings like "100.50" or "100,50" to Amount(10050).
    pub fn to_amount(&self) -> Option<Amount> {
        if self.value.is_empty() {
            return Some(Amount::ZERO);
        }
        let normalized = self.value.replace(',', ".");
        let f: f64 = normalized.parse().ok()?;
        Some(Amount::from_float(f))
    }

    /// Set the value from an Amount.
    pub fn set_amount(&mut self, amount: Amount, cx: &mut Context<Self>) {
        self.value = format!("{:.2}", amount.to_czk());
        self.cursor_pos = self.value.chars().count();
        cx.notify();
    }

    fn is_valid_char(&self, ch: char) -> bool {
        if ch.is_ascii_digit() {
            return true;
        }
        if self.allow_decimal
            && (ch == '.' || ch == ',')
            && !self.value.contains('.')
            && !self.value.contains(',')
        {
            return true;
        }
        if self.allow_negative && ch == '-' && self.cursor_pos == 0 && !self.value.starts_with('-')
        {
            return true;
        }
        false
    }

    fn handle_key_down(&mut self, event: &KeyDownEvent, cx: &mut Context<Self>) {
        let keystroke = &event.keystroke;
        let key = keystroke.key.as_str();

        match key {
            "backspace" => {
                if self.cursor_pos > 0 {
                    let byte_pos = self.char_to_byte_pos(self.cursor_pos);
                    let prev_byte_pos = self.char_to_byte_pos(self.cursor_pos - 1);
                    self.value.replace_range(prev_byte_pos..byte_pos, "");
                    self.cursor_pos -= 1;
                    cx.emit(NumberChanged(self.value.clone()));
                }
            }
            "delete" => {
                let char_count = self.value.chars().count();
                if self.cursor_pos < char_count {
                    let byte_pos = self.char_to_byte_pos(self.cursor_pos);
                    let next_byte_pos = self.char_to_byte_pos(self.cursor_pos + 1);
                    self.value.replace_range(byte_pos..next_byte_pos, "");
                    cx.emit(NumberChanged(self.value.clone()));
                }
            }
            "left" => {
                if self.cursor_pos > 0 {
                    self.cursor_pos -= 1;
                }
            }
            "right" => {
                let char_count = self.value.chars().count();
                if self.cursor_pos < char_count {
                    self.cursor_pos += 1;
                }
            }
            _ => {
                if let Some(ref ch) = keystroke.key_char
                    && !keystroke.modifiers.control
                    && !keystroke.modifiers.alt
                    && !keystroke.modifiers.platform
                {
                    // Only insert valid numeric characters.
                    for c in ch.chars() {
                        if self.is_valid_char(c) {
                            let byte_pos = self.char_to_byte_pos(self.cursor_pos);
                            self.value.insert(byte_pos, c);
                            self.cursor_pos += 1;
                        }
                    }
                    cx.emit(NumberChanged(self.value.clone()));
                }
            }
        }
        cx.notify();
    }

    fn char_to_byte_pos(&self, char_pos: usize) -> usize {
        self.value
            .char_indices()
            .nth(char_pos)
            .map(|(i, _)| i)
            .unwrap_or(self.value.len())
    }
}

impl Render for NumberInput {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let is_focused = self.focus_handle.is_focused(_window);

        let display_text = if self.value.is_empty() {
            self.placeholder.to_string()
        } else {
            self.value.clone()
        };

        let text_color = if self.value.is_empty() {
            rgb(ZfColors::TEXT_MUTED)
        } else {
            rgb(ZfColors::TEXT_PRIMARY)
        };

        let border_color = if is_focused {
            rgb(ZfColors::ACCENT)
        } else {
            rgb(ZfColors::BORDER)
        };

        div()
            .id(self.id.clone())
            .track_focus(&self.focus_handle)
            .on_key_down(cx.listener(|this, event: &KeyDownEvent, _window, cx| {
                this.handle_key_down(event, cx);
            }))
            .px_3()
            .py_2()
            .w_full()
            .bg(rgb(ZfColors::SURFACE))
            .border_1()
            .border_color(border_color)
            .rounded_md()
            .text_sm()
            .text_color(text_color)
            .cursor_text()
            .child(display_text)
    }
}
