# RFC 024: Tax Prepayments Edit

**Status:** Draft
**Author:** Claude
**Date:** 2026-03-21

## 1. Problem Statement

The `TaxPrepaymentsView` currently displays a read-only table of 12-month prepayment
schedules for income tax, social insurance, and health insurance. Users can browse by
year but cannot modify any values. The underlying `TaxYearSettingsService` already
supports full CRUD -- `save(year, flat_rate_percent, prepayments)` -- but the UI does
not expose it. Users must currently resort to direct database manipulation to set or
update their prepayment schedule, which is unacceptable for a desktop application.

The Go+SvelteKit original provides full editing with a flat rate selector, 12-row
grid of NumberInputs, quick-fill from month 1, running totals, and a save button.
The Rust GPUI port must reach feature parity.

## 2. Proposed Solution

Enhance `TaxPrepaymentsView` to support an edit/view toggle, following the pattern
established by `SettingsFirmaView`: start in read-only table mode, switch to edit
mode with an "Upravit" button, and provide "Ulozit"/"Zrusit" buttons while editing.

The view will:
- Add a `Select` component for `flat_rate_percent` at the top
- Replace static text cells with `NumberInput` components when in edit mode
- Add a "Vyplnit z prvniho mesice" quick-fill button
- Compute and display running totals from current input values
- Save all 12 months + flat rate via `service.save()` on submit
- Show success/error feedback inline (error banner, same pattern as other views)

## 3. Modified Files

Only one file requires changes:

| File | Change |
|------|--------|
| `crates/zfaktury-app/src/views/tax_prepayments.rs` | Major enhancement: add edit mode, inputs, save logic |

No new files, no new crate dependencies, no migration changes. The service layer and
domain types are already complete.

## 4. Struct Changes

### 4.1. MonthRow

A helper struct holding the three `NumberInput` entities for a single month row:

```rust
struct MonthRow {
    month: i32,  // 1-12
    tax_input: Entity<NumberInput>,
    social_input: Entity<NumberInput>,
    health_input: Entity<NumberInput>,
}
```

### 4.2. Updated TaxPrepaymentsView

```rust
pub struct TaxPrepaymentsView {
    service: Arc<TaxYearSettingsService>,
    year: i32,
    loading: bool,
    saving: bool,
    editing: bool,
    error: Option<String>,
    success: Option<String>,

    // Read-only data (always loaded)
    prepayments: Vec<TaxPrepayment>,
    flat_rate_percent: i32,

    // Edit mode entities (created on enter edit, cleared on exit)
    flat_rate_select: Option<Entity<Select>>,
    month_rows: Vec<MonthRow>,  // always 12 items when editing
}
```

Key design decisions:
- `flat_rate_select` and `month_rows` are `Option`/empty when not editing, matching
  the `SettingsFirmaView` pattern of creating input entities on demand.
- `success` field provides positive feedback after save without requiring the global
  toast system (which is not wired into individual views currently).
- `flat_rate_percent` is loaded alongside prepayments via `service.get_by_year()`.

## 5. Czech Month Names

Constant array with proper diacritics:

```rust
const CZECH_MONTHS: [&str; 12] = [
    "Leden", "Unor", "Brezen", "Duben", "Kveten", "Cerven",
    "Cervenec", "Srpen", "Zari", "Rijen", "Listopad", "Prosinec",
];
```

Note: The existing code already uses these names without diacritics. We keep them
ASCII-only for consistency with the rest of the UI (which uses "Nacitani", "Ulozit",
etc. throughout -- no diacritics anywhere in the GPUI views).

## 6. Flat Rate Selector

A `Select` component at the top of the view (visible only in edit mode) lets users
pick the flat-rate expense percentage:

```rust
fn flat_rate_options() -> Vec<SelectOption> {
    vec![
        SelectOption { value: "0".into(),  label: "Skutecne vydaje".into() },
        SelectOption { value: "30".into(), label: "30%".into() },
        SelectOption { value: "40".into(), label: "40%".into() },
        SelectOption { value: "60".into(), label: "60%".into() },
        SelectOption { value: "80".into(), label: "80%".into() },
    ]
}
```

In read-only mode, the current flat rate is displayed as a static label next to the
year selector (e.g., "Pausalni vydaje: 60%"). In edit mode, it becomes a `Select`.

## 7. Amount Display and Conversion

### 7.1. Read-Only Mode

Amounts are displayed using `format_amount()` from `crate::util::format`, which
renders `Amount` as "1 234,56 Kc" with Czech comma and thousands separator. This is
the existing behavior and remains unchanged.

### 7.2. Edit Mode

`NumberInput` already has built-in `Amount` conversion methods:

- **Loading into input:** `number_input.set_amount(amount, cx)` -- converts
  `Amount(halere)` to a string like `"1234.56"` via `amount.to_czk()`.
- **Reading from input:** `number_input.to_amount()` -- parses the string back to
  `Amount`, handling both `.` and `,` as decimal separators.

The `NumberInput` component accepts decimal input by default (`allow_decimal: true`),
which is correct for CZK amounts. No special conversion logic is needed in the view.

**Example flow:**
1. Database stores `Amount::from_halere(350000)` (3,500.00 CZK)
2. `set_amount()` populates input with `"3500.00"`
3. User types `"4200.50"`
4. `to_amount()` returns `Some(Amount::from_halere(420050))`

## 8. Data Initialization

When entering edit mode (`start_editing`), the view creates all input entities and
populates them from the current `self.prepayments` data:

```rust
fn start_editing(&mut self, cx: &mut Context<Self>) {
    self.editing = true;
    self.error = None;
    self.success = None;

    // Create flat rate select
    let flat_rate_select = cx.new(|cx| {
        let mut s = Select::new("flat-rate-select", "Pausalni vydaje", flat_rate_options());
        s.set_selected_value(&self.flat_rate_percent.to_string(), cx);
        s
    });
    self.flat_rate_select = Some(flat_rate_select);

    // Create 12 month rows
    self.month_rows.clear();
    for i in 0..12 {
        let month = (i + 1) as i32;
        let prepayment = self.prepayments.get(i);

        let tax_input = cx.new(|cx| {
            let mut n = NumberInput::new(
                SharedString::from(format!("tax-{month}")),
                "0,00",
                cx,
            );
            if let Some(p) = prepayment {
                n.set_amount(p.tax_amount, cx);
            }
            n
        });

        let social_input = cx.new(|cx| {
            let mut n = NumberInput::new(
                SharedString::from(format!("social-{month}")),
                "0,00",
                cx,
            );
            if let Some(p) = prepayment {
                n.set_amount(p.social_amount, cx);
            }
            n
        });

        let health_input = cx.new(|cx| {
            let mut n = NumberInput::new(
                SharedString::from(format!("health-{month}")),
                "0,00",
                cx,
            );
            if let Some(p) = prepayment {
                n.set_amount(p.health_amount, cx);
            }
            n
        });

        self.month_rows.push(MonthRow {
            month,
            tax_input,
            social_input,
            health_input,
        });
    }

    cx.notify();
}
```

## 9. Quick-Fill Button

A "Vyplnit z prvniho mesice" button copies the values from month 1 inputs to months
2-12. This is useful when all months have the same prepayment amount (the common case).

```rust
fn fill_from_first_month(&mut self, cx: &mut Context<Self>) {
    if self.month_rows.is_empty() {
        return;
    }

    // Read values from month 1
    let tax_val = self.month_rows[0].tax_input.read(cx).value().to_string();
    let social_val = self.month_rows[0].social_input.read(cx).value().to_string();
    let health_val = self.month_rows[0].health_input.read(cx).value().to_string();

    // Apply to months 2-12
    for row in self.month_rows.iter().skip(1) {
        row.tax_input.update(cx, |n, cx| n.set_value(&tax_val, cx));
        row.social_input.update(cx, |n, cx| n.set_value(&social_val, cx));
        row.health_input.update(cx, |n, cx| n.set_value(&health_val, cx));
    }

    cx.notify();
}
```

Placement: rendered below the flat rate selector, above the grid, only visible in
edit mode. Uses `ButtonVariant::Secondary`.

## 10. Running Totals

Totals are computed from the current state of the data -- either from `self.prepayments`
(read-only mode) or from the live `NumberInput` values (edit mode).

### 10.1. Read-Only Mode (unchanged)

Totals are computed inline during render from `self.prepayments`, exactly as the
current code does.

### 10.2. Edit Mode

During render, iterate `self.month_rows` and sum the `to_amount()` results:

```rust
fn compute_edit_totals(&self, cx: &App) -> (Amount, Amount, Amount) {
    let mut total_tax = Amount::ZERO;
    let mut total_social = Amount::ZERO;
    let mut total_health = Amount::ZERO;

    for row in &self.month_rows {
        total_tax += row.tax_input.read(cx).to_amount().unwrap_or(Amount::ZERO);
        total_social += row.social_input.read(cx).to_amount().unwrap_or(Amount::ZERO);
        total_health += row.health_input.read(cx).to_amount().unwrap_or(Amount::ZERO);
    }

    (total_tax, total_social, total_health)
}
```

The totals row at the bottom of the table displays these sums using `format_amount()`.

**Live update concern:** GPUI re-renders on every keystroke (the `NumberInput` calls
`cx.notify()` after each key event), so the totals update in real time as the user
types. No explicit subscription or `$effect` equivalent is needed.

## 11. Save Flow

### 11.1. Collect and Validate

```rust
fn save(&mut self, cx: &mut Context<Self>) {
    if self.saving {
        return;
    }

    // Read flat rate
    let flat_rate_percent: i32 = self.flat_rate_select
        .as_ref()
        .and_then(|s| s.read(cx).selected_value())
        .and_then(|v| v.parse().ok())
        .unwrap_or(0);

    // Collect all 12 prepayments
    let mut prepayments = Vec::with_capacity(12);
    for row in &self.month_rows {
        let tax = row.tax_input.read(cx).to_amount();
        let social = row.social_input.read(cx).to_amount();
        let health = row.health_input.read(cx).to_amount();

        // Validate: all must parse successfully
        if tax.is_none() || social.is_none() || health.is_none() {
            self.error = Some(format!(
                "Neplatna castka v mesici {}",
                CZECH_MONTHS[(row.month - 1) as usize]
            ));
            cx.notify();
            return;
        }

        prepayments.push(TaxPrepayment {
            year: self.year,
            month: row.month,
            tax_amount: tax.unwrap(),
            social_amount: social.unwrap(),
            health_amount: health.unwrap(),
        });
    }

    self.saving = true;
    self.error = None;
    cx.notify();

    // ...spawn background save task...
}
```

### 11.2. Background Save

```rust
    let service = self.service.clone();
    let year = self.year;
    cx.spawn(async move |this, cx| {
        let result = cx
            .background_executor()
            .spawn(async move {
                service.save(year, flat_rate_percent, &prepayments)
            })
            .await;

        this.update(cx, |this, cx| {
            this.saving = false;
            match result {
                Ok(()) => {
                    // Update local state from saved data
                    this.prepayments = prepayments;
                    this.flat_rate_percent = flat_rate_percent;

                    // Exit edit mode
                    this.editing = false;
                    this.month_rows.clear();
                    this.flat_rate_select = None;

                    // Show success feedback
                    this.success = Some("Zalohy ulozeny".into());
                }
                Err(e) => {
                    this.error = Some(format!("Chyba pri ukladani zaloh: {e}"));
                }
            }
            cx.notify();
        }).ok();
    })
    .detach();
```

### 11.3. Why Not Global Toast

The global `ToastView` component exists but is not currently wired into individual
views -- it lives in the root layout and no existing view emits toast events. To
avoid cross-cutting changes, we use an inline `success` message banner (green
background, same styling as error but with success colors), which auto-clears on
the next action (year change, re-entering edit mode). This is consistent with the
error display pattern already used in this view.

If the project later adopts a global toast event bus, this can be trivially migrated.

## 12. Cancel / Exit Edit Mode

```rust
fn cancel_editing(&mut self, cx: &mut Context<Self>) {
    self.editing = false;
    self.error = None;
    self.month_rows.clear();
    self.flat_rate_select = None;
    cx.notify();
}
```

All `Entity<NumberInput>` and `Entity<Select>` handles are dropped, freeing their
GPUI allocations. Re-entering edit mode creates fresh entities from `self.prepayments`.

## 13. Year Change Behavior

When the user changes the year (via the existing `<`/`>` buttons):

1. If currently editing, **exit edit mode first** (discard unsaved changes). This
   prevents confusion about which year's data the inputs contain.
2. Set `self.loading = true`, clear success/error.
3. Load both `get_prepayments(year)` and `get_by_year(year)` in a single background
   task (they share the same `background_executor` spawn).
4. On completion, update `self.prepayments` and `self.flat_rate_percent`.

```rust
fn change_year(&mut self, delta: i32, cx: &mut Context<Self>) {
    // Exit edit mode if active
    if self.editing {
        self.cancel_editing(cx);
    }

    self.year += delta;
    self.loading = true;
    self.error = None;
    self.success = None;
    cx.notify();
    self.load_data(cx);
}
```

The `load_data` method is updated to also fetch `flat_rate_percent`:

```rust
fn load_data(&mut self, cx: &mut Context<Self>) {
    let service = self.service.clone();
    let year = self.year;

    cx.spawn(async move |this, cx| {
        let result = cx
            .background_executor()
            .spawn(async move {
                let prepayments = service.get_prepayments(year)?;
                let settings = service.get_by_year(year)?;
                Ok::<_, DomainError>((prepayments, settings.flat_rate_percent))
            })
            .await;

        this.update(cx, |this, cx| {
            this.loading = false;
            match result {
                Ok((prepayments, flat_rate)) => {
                    this.prepayments = prepayments;
                    this.flat_rate_percent = flat_rate;
                }
                Err(e) => {
                    this.error = Some(format!("Chyba pri nacitani zaloh: {e}"));
                }
            }
            cx.notify();
        }).ok();
    })
    .detach();
}
```

## 14. Render Structure

### 14.1. Header Row

```
+------------------------------------------------------+
| Zalohy  [<] [2026] [>]   Pausalni vydaje: 60%        |
|                                          [Upravit]    |
+------------------------------------------------------+
```

In edit mode:

```
+------------------------------------------------------+
| Zalohy  [<] [2026] [>]   [Select: Pausalni vydaje v] |
|  [Vyplnit z prvniho mesice]    [Zrusit]   [Ulozit]   |
+------------------------------------------------------+
```

### 14.2. Table Grid

Read-only (unchanged from current):

```
| Mesic          | Dan z prijmu  | Socialni poj. | Zdravotni poj. | Celkem  |
|----------------|---------------|---------------|----------------|---------|
| 1. Leden       |    3 500 Kc   |    2 944 Kc   |     2 968 Kc   | 9 412   |
| 2. Unor        |    3 500 Kc   |    2 944 Kc   |     2 968 Kc   | 9 412   |
| ...            |               |               |                |         |
| Celkem         |   42 000 Kc   |   35 328 Kc   |    35 616 Kc   | 112 944 |
```

Edit mode:

```
| Mesic          | Dan z prijmu    | Socialni poj.  | Zdravotni poj.  | Celkem  |
|----------------|-----------------|----------------|-----------------|---------|
| 1. Leden       | [NumberInput]   | [NumberInput]   | [NumberInput]   | 9 412   |
| 2. Unor        | [NumberInput]   | [NumberInput]   | [NumberInput]   | 9 412   |
| ...            |                 |                 |                 |         |
| Celkem         |   42 000 Kc     |   35 328 Kc     |    35 616 Kc    | 112 944 |
```

The "Celkem" column per row and the totals row at the bottom remain computed text
(not inputs), updated live as the user types.

### 14.3. Render Dispatch

The `Render` impl dispatches between two rendering paths based on `self.editing`:

```rust
impl Render for TaxPrepaymentsView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        // ... header (always rendered) ...
        // ... loading/error states ...

        if self.editing {
            self.render_edit_mode(cx)
        } else {
            self.render_view_mode(cx)
        }
    }
}
```

`render_view_mode` contains the existing table rendering logic (extracted from the
current monolithic `render` method).

`render_edit_mode` replaces static text cells with `NumberInput` entities from
`self.month_rows`.

## 15. Data Flow Summary

### 15.1. Initial Load

```
new() -> load_data()
  |-> background: service.get_prepayments(year) + service.get_by_year(year)
  |-> update: self.prepayments = [...], self.flat_rate_percent = N
  |-> cx.notify() -> render_view_mode()
```

### 15.2. Enter Edit Mode

```
click "Upravit" -> start_editing()
  |-> create Entity<Select> for flat_rate, set to self.flat_rate_percent
  |-> for each month 1-12:
  |     create 3x Entity<NumberInput>, set_amount from self.prepayments[i]
  |     push MonthRow into self.month_rows
  |-> self.editing = true
  |-> cx.notify() -> render_edit_mode()
```

### 15.3. User Edits Values

```
user types in NumberInput
  |-> NumberInput.handle_key_down() -> cx.notify()
  |-> GPUI re-renders TaxPrepaymentsView
  |-> render_edit_mode() reads all NumberInput values for totals row
  |-> totals update live
```

### 15.4. Quick-Fill

```
click "Vyplnit z prvniho mesice" -> fill_from_first_month()
  |-> read month_rows[0].{tax,social,health}_input values
  |-> for month_rows[1..12]: set_value() on each input
  |-> cx.notify() -> re-render with copied values
```

### 15.5. Save

```
click "Ulozit" -> save()
  |-> validate: all 36 NumberInputs parse to Amount
  |-> if invalid: set self.error, return
  |-> self.saving = true
  |-> background: service.save(year, flat_rate, prepayments)
  |-> on success:
  |     update self.prepayments, self.flat_rate_percent from saved data
  |     exit edit mode (clear month_rows, flat_rate_select)
  |     self.success = "Zalohy ulozeny"
  |-> on error:
  |     self.error = "Chyba pri ukladani zaloh: ..."
  |-> cx.notify()
```

### 15.6. Cancel

```
click "Zrusit" -> cancel_editing()
  |-> self.editing = false
  |-> clear month_rows, flat_rate_select
  |-> cx.notify() -> render_view_mode() (from unchanged self.prepayments)
```

### 15.7. Change Year

```
click "<" or ">" -> change_year(delta)
  |-> if editing: cancel_editing() first
  |-> self.year += delta, self.loading = true
  |-> background: service.get_prepayments(new_year) + get_by_year(new_year)
  |-> update self.prepayments, self.flat_rate_percent
  |-> cx.notify() -> render_view_mode()
```

## 16. Error Handling

### 16.1. Load Errors

Handled identically to current implementation: set `self.error` and display a red
banner. User can retry by changing year back and forth.

### 16.2. Validation Errors

When `to_amount()` returns `None` for any input (invalid number format), display
an error banner identifying the offending month: "Neplatna castka v mesici Brezen".
The view stays in edit mode so the user can correct the value.

### 16.3. Save Errors

Service-level errors (database failures) are displayed in the error banner. The view
stays in edit mode with all input values preserved, so the user can retry.

### 16.4. Success Feedback

After a successful save, display a green banner: "Zalohy ulozeny". This message
clears when the user changes year or re-enters edit mode.

```rust
// In render, before the table:
if let Some(ref msg) = self.success {
    content = content.child(
        div()
            .px_4()
            .py_3()
            .bg(rgb(ZfColors::STATUS_GREEN_BG))
            .rounded_md()
            .text_sm()
            .text_color(rgb(ZfColors::STATUS_GREEN))
            .child(msg.clone()),
    );
}
```

## 17. Edge Cases

### 17.1. Empty Year

When no prepayments exist for a year, `service.get_prepayments()` returns 12 months
of zeros. Entering edit mode creates 12 rows of NumberInputs all initialized to
`"0.00"`. This is correct behavior -- the user fills in the values they want.

### 17.2. Partial Data

If some months have values and others are zero, all 12 rows are always shown. The
service always returns exactly 12 entries (filling gaps with zeros).

### 17.3. Negative Amounts

`NumberInput` has `allow_negative: false` by default. Prepayment amounts should never
be negative, so this default is correct and no changes are needed.

### 17.4. Large Amounts

`NumberInput` accepts arbitrary decimal strings. `Amount::from_float()` handles
values up to `i64::MAX / 100` (about 92 quadrillion CZK), which is more than
sufficient.

### 17.5. Concurrent Year Changes

If the user rapidly clicks year navigation, multiple background loads may be in
flight. The last one to complete wins (overwrites `self.prepayments`). This is the
same behavior as the current implementation and is acceptable -- the final state
is always consistent with some valid year's data, and `self.year` is updated
synchronously before the async load starts.

## 18. Implementation Checklist

1. Add new fields to `TaxPrepaymentsView` struct (`editing`, `saving`, `success`,
   `flat_rate_percent`, `flat_rate_select`, `month_rows`)
2. Add `MonthRow` struct
3. Add `flat_rate_options()` helper function
4. Add `CZECH_MONTHS` constant (rename existing inline array)
5. Update `load_data()` to also fetch `flat_rate_percent` via `get_by_year()`
6. Update `change_year()` to cancel editing if active
7. Implement `start_editing()` -- create Select + 36 NumberInputs
8. Implement `cancel_editing()` -- clear entities, set `editing = false`
9. Implement `fill_from_first_month()` -- copy month 1 values to 2-12
10. Implement `save()` -- validate, collect, background save, handle result
11. Extract `render_view_mode()` from existing render code (refactor, no behavior change)
12. Implement `render_edit_mode()` -- flat rate select, grid with NumberInputs, totals
13. Update `render()` to dispatch between view/edit and show header buttons
14. Add imports: `Select`, `SelectOption`, `NumberInput`, `SharedString`, `DomainError`
15. Update the `new()` constructor to initialize new fields with defaults

## 19. Testing Considerations

This is a pure UI enhancement with no new service or domain logic. The
`TaxYearSettingsService` already has full test coverage for `save()`,
`get_prepayments()`, and `get_by_year()`.

Manual testing should verify:
- Loading prepayments for years with and without existing data
- Entering/exiting edit mode preserves data correctly
- Quick-fill copies all three columns from month 1
- Totals update live as values are typed
- Save persists data (verify by switching away and back to the year)
- Validation rejects non-numeric input
- Year change while editing discards changes without error
- Flat rate select value persists across save/reload cycles

## 20. Future Considerations

- **Global toast system:** If a toast event bus is added to the app, the success/error
  feedback can be migrated from inline banners to toast notifications.
- **Undo support:** A future RFC could add undo/redo for the 36-input grid, though
  the quick-fill + cancel flow provides adequate escape hatches for now.
- **Tax constants sidebar:** The Go original shows a reference panel with current-year
  minimum prepayment amounts. This could be added as a follow-up enhancement once
  the tax constants are available in the domain layer.
