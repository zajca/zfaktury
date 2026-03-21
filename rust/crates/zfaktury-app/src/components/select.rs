use gpui::*;

use crate::theme::ZfColors;

/// Event emitted when a selection changes.
pub struct SelectionChanged {
    pub value: String,
    pub label: String,
}

/// A single option in the dropdown.
#[derive(Clone)]
pub struct SelectOption {
    pub value: String,
    pub label: String,
}

/// Dropdown select component.
pub struct Select {
    id: ElementId,
    options: Vec<SelectOption>,
    selected: Option<usize>,
    placeholder: SharedString,
    open: bool,
}

impl EventEmitter<SelectionChanged> for Select {}

impl Select {
    pub fn new(
        id: impl Into<ElementId>,
        placeholder: impl Into<SharedString>,
        options: Vec<SelectOption>,
    ) -> Self {
        Self {
            id: id.into(),
            options,
            selected: None,
            placeholder: placeholder.into(),
            open: false,
        }
    }

    pub fn selected_value(&self) -> Option<&str> {
        self.selected.map(|i| self.options[i].value.as_str())
    }

    pub fn set_selected_value(&mut self, value: &str, cx: &mut Context<Self>) {
        self.selected = self.options.iter().position(|o| o.value == value);
        cx.notify();
    }

    pub fn set_options(&mut self, options: Vec<SelectOption>, cx: &mut Context<Self>) {
        self.options = options;
        self.selected = None;
        cx.notify();
    }
}

impl Render for Select {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let display_text = match self.selected {
            Some(i) => self.options[i].label.clone(),
            None => self.placeholder.to_string(),
        };

        let text_color = if self.selected.is_some() {
            rgb(ZfColors::TEXT_PRIMARY)
        } else {
            rgb(ZfColors::TEXT_MUTED)
        };

        let mut container = div().relative().w_full();

        // Trigger button
        container = container.child(
            div()
                .id(self.id.clone())
                .flex()
                .items_center()
                .justify_between()
                .px_3()
                .py_2()
                .w_full()
                .bg(rgb(ZfColors::SURFACE))
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
                .rounded_md()
                .text_sm()
                .text_color(text_color)
                .cursor_pointer()
                .on_click(cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.open = !this.open;
                    cx.notify();
                }))
                .child(display_text)
                .child(
                    div()
                        .text_xs()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child("▼"),
                ),
        );

        // Dropdown overlay
        if self.open {
            let mut dropdown = div()
                .id(SharedString::from(format!("{}-dropdown", self.id)))
                .absolute()
                .top(px(36.0))
                .left_0()
                .w_full()
                .max_h(px(200.0))
                .overflow_y_scroll()
                .bg(rgb(ZfColors::SURFACE))
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
                .rounded_md()
                .shadow_md()
                .flex()
                .flex_col();

            for (index, option) in self.options.iter().enumerate() {
                let is_selected = self.selected == Some(index);
                let value = option.value.clone();
                let label = option.label.clone();

                let bg = if is_selected {
                    rgb(ZfColors::ACCENT_MUTED)
                } else {
                    rgb(0x00000000)
                };

                dropdown = dropdown.child(
                    div()
                        .id(SharedString::from(format!("opt-{}-{index}", self.id)))
                        .px_3()
                        .py_2()
                        .text_sm()
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .bg(bg)
                        .cursor_pointer()
                        .hover(|s| s.bg(rgb(ZfColors::SURFACE_HOVER)))
                        .on_click(cx.listener(move |this, _event: &ClickEvent, _window, cx| {
                            this.selected = Some(index);
                            this.open = false;
                            cx.emit(SelectionChanged {
                                value: value.clone(),
                                label: label.clone(),
                            });
                            cx.notify();
                        }))
                        .child(option.label.clone()),
                );
            }

            container = container.child(dropdown);
        }

        container
    }
}
