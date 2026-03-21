use gpui::*;
use zfaktury_domain::{Amount, InvoiceItem};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::number_input::{NumberChanged, NumberInput};
use crate::components::select::{Select, SelectOption, SelectionChanged};
use crate::components::text_input::{TextChanged, TextInput};
use crate::theme::ZfColors;
use crate::util::format::format_amount;

/// Event emitted when any item field changes.
pub struct ItemsChanged;

/// A single row in the items editor.
struct ItemRow {
    description: Entity<TextInput>,
    quantity: Entity<NumberInput>,
    unit: Entity<TextInput>,
    unit_price: Entity<NumberInput>,
    vat_rate: Entity<Select>,
}

/// Reusable invoice items editor component.
/// Allows adding, removing, and editing invoice line items with live totals.
pub struct InvoiceItemsEditor {
    items: Vec<ItemRow>,
}

impl EventEmitter<ItemsChanged> for InvoiceItemsEditor {}

fn vat_options() -> Vec<SelectOption> {
    vec![
        SelectOption {
            value: "0".to_string(),
            label: "0%".to_string(),
        },
        SelectOption {
            value: "12".to_string(),
            label: "12%".to_string(),
        },
        SelectOption {
            value: "21".to_string(),
            label: "21%".to_string(),
        },
    ]
}

impl InvoiceItemsEditor {
    /// Create a new editor with one empty row.
    pub fn new(cx: &mut Context<Self>) -> Self {
        let mut editor = Self { items: Vec::new() };
        editor.add_item(cx);
        editor
    }

    /// Convert current editor state to domain InvoiceItem structs.
    /// Sets id=0, invoice_id=0, vat_amount=ZERO, total_amount=ZERO
    /// because the service layer calculates those.
    pub fn to_invoice_items(&self, cx: &App) -> Vec<InvoiceItem> {
        self.items
            .iter()
            .enumerate()
            .map(|(i, row)| {
                let description = row.description.read(cx).value().to_string();
                let quantity = row
                    .quantity
                    .read(cx)
                    .to_amount()
                    .unwrap_or(Amount::new(1, 0));
                let unit = row.unit.read(cx).value().to_string();
                let unit_price = row.unit_price.read(cx).to_amount().unwrap_or(Amount::ZERO);
                let vat_rate_percent: i32 = row
                    .vat_rate
                    .read(cx)
                    .selected_value()
                    .and_then(|v| v.parse().ok())
                    .unwrap_or(21);

                InvoiceItem {
                    id: 0,
                    invoice_id: 0,
                    description,
                    quantity,
                    unit,
                    unit_price,
                    vat_rate_percent,
                    vat_amount: Amount::ZERO,
                    total_amount: Amount::ZERO,
                    sort_order: (i + 1) as i32,
                }
            })
            .collect()
    }

    /// Populate items editor from existing invoice items (for edit mode).
    pub fn set_items(&mut self, items: &[InvoiceItem], cx: &mut Context<Self>) {
        self.items.clear();
        for item in items {
            let row = self.create_item_row(cx);
            row.description.update(cx, |input, cx| {
                input.set_value(&item.description, cx);
            });
            row.quantity.update(cx, |input, cx| {
                input.set_amount(item.quantity, cx);
            });
            row.unit.update(cx, |input, cx| {
                input.set_value(&item.unit, cx);
            });
            row.unit_price.update(cx, |input, cx| {
                input.set_amount(item.unit_price, cx);
            });
            let vat_str = item.vat_rate_percent.to_string();
            row.vat_rate.update(cx, |select, cx| {
                select.set_selected_value(&vat_str, cx);
            });
            self.items.push(row);
        }
        if self.items.is_empty() {
            self.add_item(cx);
        }
        cx.notify();
    }

    fn add_item(&mut self, cx: &mut Context<Self>) {
        let row = self.create_item_row(cx);
        self.items.push(row);
        cx.notify();
    }

    fn remove_item(&mut self, index: usize, cx: &mut Context<Self>) {
        if self.items.len() > 1 && index < self.items.len() {
            self.items.remove(index);
            cx.emit(ItemsChanged);
            cx.notify();
        }
    }

    fn create_item_row(&mut self, cx: &mut Context<Self>) -> ItemRow {
        let idx = self.items.len();

        let description = cx.new(|cx| {
            TextInput::new(
                SharedString::from(format!("item-desc-{idx}")),
                "Popis polozky...",
                cx,
            )
        });
        cx.subscribe(
            &description,
            |this: &mut Self, _entity, _event: &TextChanged, cx| {
                cx.emit(ItemsChanged);
                let _ = this;
            },
        )
        .detach();

        let quantity = cx.new(|cx| {
            NumberInput::new(SharedString::from(format!("item-qty-{idx}")), "1", cx).with_value("1")
        });
        cx.subscribe(
            &quantity,
            |this: &mut Self, _entity, _event: &NumberChanged, cx| {
                cx.emit(ItemsChanged);
                let _ = this;
            },
        )
        .detach();

        let unit =
            cx.new(|cx| TextInput::new(SharedString::from(format!("item-unit-{idx}")), "ks", cx));
        cx.subscribe(
            &unit,
            |this: &mut Self, _entity, _event: &TextChanged, cx| {
                cx.emit(ItemsChanged);
                let _ = this;
            },
        )
        .detach();

        let unit_price = cx.new(|cx| {
            NumberInput::new(SharedString::from(format!("item-price-{idx}")), "0.00", cx)
        });
        cx.subscribe(
            &unit_price,
            |this: &mut Self, _entity, _event: &NumberChanged, cx| {
                cx.emit(ItemsChanged);
                let _ = this;
            },
        )
        .detach();

        let vat_rate = cx.new(|cx| {
            let mut select = Select::new(
                SharedString::from(format!("item-vat-{idx}")),
                "DPH",
                vat_options(),
            );
            select.set_selected_value("21", cx);
            select
        });
        cx.subscribe(
            &vat_rate,
            |this: &mut Self, _entity, _event: &SelectionChanged, cx| {
                cx.emit(ItemsChanged);
                let _ = this;
            },
        )
        .detach();

        ItemRow {
            description,
            quantity,
            unit,
            unit_price,
            vat_rate,
        }
    }

    /// Calculate live totals from current item values.
    fn compute_totals(&self, cx: &App) -> (Amount, Amount, Amount) {
        let mut subtotal = Amount::ZERO;
        let mut vat_total = Amount::ZERO;

        for row in &self.items {
            let qty = row.quantity.read(cx).to_amount().unwrap_or(Amount::ZERO);
            let price = row.unit_price.read(cx).to_amount().unwrap_or(Amount::ZERO);
            let rate: i32 = row
                .vat_rate
                .read(cx)
                .selected_value()
                .and_then(|v| v.parse().ok())
                .unwrap_or(21);

            let line = Amount::from_halere(qty.halere() * price.halere() / 100);
            let line_vat = line.multiply(rate as f64 / 100.0);

            subtotal += line;
            vat_total += line_vat;
        }

        let total = subtotal + vat_total;
        (subtotal, vat_total, total)
    }

    fn render_header() -> Div {
        div()
            .flex()
            .gap_2()
            .pb_2()
            .border_b_1()
            .border_color(rgb(ZfColors::BORDER_SUBTLE))
            .child(
                div()
                    .flex_1()
                    .text_xs()
                    .font_weight(FontWeight::MEDIUM)
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child("Popis"),
            )
            .child(
                div()
                    .w(px(80.0))
                    .text_xs()
                    .font_weight(FontWeight::MEDIUM)
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child("Mnozstvi"),
            )
            .child(
                div()
                    .w(px(80.0))
                    .text_xs()
                    .font_weight(FontWeight::MEDIUM)
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child("Jednotka"),
            )
            .child(
                div()
                    .w(px(112.0))
                    .text_xs()
                    .font_weight(FontWeight::MEDIUM)
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child("Cena/ks"),
            )
            .child(
                div()
                    .w(px(80.0))
                    .text_xs()
                    .font_weight(FontWeight::MEDIUM)
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child("DPH %"),
            )
            .child(
                div()
                    .w(px(40.0))
                    .text_xs()
                    .font_weight(FontWeight::MEDIUM)
                    .text_color(rgb(ZfColors::TEXT_SECONDARY)),
            )
    }

    fn render_totals(&self, cx: &App) -> Div {
        let (subtotal, vat, total) = self.compute_totals(cx);

        div()
            .flex()
            .flex_col()
            .items_end()
            .gap_1()
            .pt_3()
            .border_t_1()
            .border_color(rgb(ZfColors::BORDER_SUBTLE))
            .child(
                div()
                    .flex()
                    .gap_4()
                    .child(
                        div()
                            .text_sm()
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .child("Zaklad dane:"),
                    )
                    .child(
                        div()
                            .text_sm()
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child(format_amount(subtotal)),
                    ),
            )
            .child(
                div()
                    .flex()
                    .gap_4()
                    .child(
                        div()
                            .text_sm()
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .child("DPH:"),
                    )
                    .child(
                        div()
                            .text_sm()
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child(format_amount(vat)),
                    ),
            )
            .child(
                div()
                    .flex()
                    .gap_4()
                    .child(
                        div()
                            .text_sm()
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child("Celkem:"),
                    )
                    .child(
                        div()
                            .text_sm()
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(rgb(ZfColors::ACCENT))
                            .child(format_amount(total)),
                    ),
            )
    }
}

impl Render for InvoiceItemsEditor {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let can_remove = self.items.len() > 1;
        let item_count = self.items.len();

        // Build item rows with proper click handlers for remove buttons.
        let mut rows_container = div().flex().flex_col().gap_1();
        for index in 0..item_count {
            let row = &self.items[index];

            let mut row_div = div()
                .flex()
                .gap_2()
                .items_start()
                .py_1()
                .child(div().flex_1().child(row.description.clone()))
                .child(div().w(px(80.0)).child(row.quantity.clone()))
                .child(div().w(px(80.0)).child(row.unit.clone()))
                .child(div().w(px(112.0)).child(row.unit_price.clone()))
                .child(div().w(px(80.0)).child(row.vat_rate.clone()));

            if can_remove {
                row_div = row_div.child(
                    div()
                        .w(px(40.0))
                        .flex()
                        .items_center()
                        .justify_center()
                        .pt_2()
                        .child(
                            div()
                                .id(SharedString::from(format!("rm-item-{index}")))
                                .text_sm()
                                .text_color(rgb(ZfColors::STATUS_RED))
                                .cursor_pointer()
                                .hover(|s| s.text_color(rgb(ZfColors::TEXT_PRIMARY)))
                                .on_click(cx.listener(
                                    move |this, _event: &ClickEvent, _window, cx| {
                                        this.remove_item(index, cx);
                                    },
                                ))
                                .child("X"),
                        ),
                );
            } else {
                row_div = row_div.child(div().w(px(40.0)));
            }

            rows_container = rows_container.child(row_div);
        }

        // Add button
        let add_btn = render_button(
            "add-item-btn",
            "+ Pridat polozku",
            ButtonVariant::Secondary,
            false,
            false,
            cx.listener(|this, _event: &ClickEvent, _window, cx| {
                this.add_item(cx);
            }),
        );

        // Totals
        let totals = self.render_totals(cx);

        div()
            .flex()
            .flex_col()
            .gap_3()
            .child(Self::render_header())
            .child(rows_container)
            .child(div().flex().child(add_btn))
            .child(totals)
    }
}
