use gpui::*;

use crate::theme::ZfColors;

/// Event emitted when the checkbox is toggled.
pub struct CheckboxToggled(pub bool);

/// Boolean toggle checkbox component.
pub struct Checkbox {
    id: ElementId,
    checked: bool,
    label: SharedString,
}

impl EventEmitter<CheckboxToggled> for Checkbox {}

impl Checkbox {
    pub fn new(id: impl Into<ElementId>, label: impl Into<SharedString>, checked: bool) -> Self {
        Self {
            id: id.into(),
            checked,
            label: label.into(),
        }
    }

    pub fn is_checked(&self) -> bool {
        self.checked
    }

    pub fn set_checked(&mut self, checked: bool, cx: &mut Context<Self>) {
        self.checked = checked;
        cx.notify();
    }
}

impl Render for Checkbox {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let checked = self.checked;

        let (box_bg, box_border) = if checked {
            (rgb(ZfColors::ACCENT), rgb(ZfColors::ACCENT))
        } else {
            (rgb(ZfColors::SURFACE), rgb(ZfColors::BORDER))
        };

        let check_mark = if checked { "✓" } else { "" };

        div()
            .id(self.id.clone())
            .flex()
            .items_center()
            .gap_2()
            .cursor_pointer()
            .on_click(cx.listener(move |this, _event: &ClickEvent, _window, cx| {
                this.checked = !this.checked;
                cx.emit(CheckboxToggled(this.checked));
                cx.notify();
            }))
            .child(
                div()
                    .w(px(16.0))
                    .h(px(16.0))
                    .flex()
                    .items_center()
                    .justify_center()
                    .bg(box_bg)
                    .border_1()
                    .border_color(box_border)
                    .rounded(px(3.0))
                    .text_xs()
                    .text_color(rgb(0xffffff))
                    .child(check_mark),
            )
            .child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(self.label.clone()),
            )
    }
}
