use gpui::*;

use crate::theme::ZfColors;

/// Event emitted when the text area value changes.
pub struct TextAreaChanged(pub String);

/// Multi-line text area component.
pub struct TextArea {
    id: ElementId,
    value: String,
    placeholder: SharedString,
    focus_handle: FocusHandle,
    cursor_pos: usize,
    rows: u32,
}

impl EventEmitter<TextAreaChanged> for TextArea {}

impl TextArea {
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
            rows: 4,
        }
    }

    #[allow(dead_code)]
    pub fn with_rows(mut self, rows: u32) -> Self {
        self.rows = rows;
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
                    cx.emit(TextAreaChanged(self.value.clone()));
                }
            }
            "delete" => {
                let char_count = self.value.chars().count();
                if self.cursor_pos < char_count {
                    let byte_pos = self.char_to_byte_pos(self.cursor_pos);
                    let next_byte_pos = self.char_to_byte_pos(self.cursor_pos + 1);
                    self.value.replace_range(byte_pos..next_byte_pos, "");
                    cx.emit(TextAreaChanged(self.value.clone()));
                }
            }
            "enter" => {
                // Insert newline for multi-line editing.
                let byte_pos = self.char_to_byte_pos(self.cursor_pos);
                self.value.insert(byte_pos, '\n');
                self.cursor_pos += 1;
                cx.emit(TextAreaChanged(self.value.clone()));
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
            "home" => {
                self.cursor_pos = 0;
            }
            "end" => {
                self.cursor_pos = self.value.chars().count();
            }
            _ => {
                if let Some(ref ch) = keystroke.key_char
                    && !keystroke.modifiers.control
                    && !keystroke.modifiers.alt
                    && !keystroke.modifiers.platform
                    && !ch.chars().all(|c| c.is_control())
                {
                    let byte_pos = self.char_to_byte_pos(self.cursor_pos);
                    self.value.insert_str(byte_pos, ch);
                    self.cursor_pos += ch.chars().count();
                    cx.emit(TextAreaChanged(self.value.clone()));
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

impl Render for TextArea {
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

        let min_height = px(self.rows as f32 * 20.0);

        div()
            .id(self.id.clone())
            .track_focus(&self.focus_handle)
            .on_key_down(cx.listener(|this, event: &KeyDownEvent, _window, cx| {
                this.handle_key_down(event, cx);
            }))
            .px_3()
            .py_2()
            .w_full()
            .min_h(min_height)
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
