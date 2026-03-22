# RFC 022: Tax Credits CRUD Forms

**Status:** Draft
**Author:** Developer
**Created:** 2026-03-21
**Modified files:** `zfaktury-app/src/views/tax_credits.rs`

---

## 1. Problem Statement

The `TaxCreditsView` currently renders tax credits and deductions in a read-only card layout. Users can browse data by year but cannot create, edit, or delete any records. All four entity types (spouse credit, child credits, personal credits, deductions) lack interactive forms despite the service layer (`TaxCreditsService`) already providing full CRUD operations.

Without CRUD capability, users must enter tax credit data through an alternative mechanism (direct DB manipulation or CLI), which defeats the purpose of a desktop GUI application.

## 2. Proposed Solution

Enhance the existing `TaxCreditsView` with **inline editing forms** embedded directly into each card section. This follows the same pattern used by `SettingsCategoriesView`, which provides inline create/edit/delete within a single view.

Key design decisions:

- **No new files.** All changes are confined to `views/tax_credits.rs`.
- **Inline forms, not separate route views.** Each card toggles between display mode and edit mode.
- **One editing context at a time.** Only one section (spouse/child/personal/deduction) can be in edit mode simultaneously to avoid conflicting saves.
- **Service computes credit amounts.** The view sends raw input fields; the service/repo computes `credit_amount`, `max_amount`, `allowed_amount` on save and returns the computed values via data reload.
- **Copy from previous year.** A new action button allows cloning all credits from `year - 1`.

## 3. Architecture

### 3.1 Editing State Model

The view struct gains per-section editing state. Each section uses a dedicated struct holding `Entity<T>` form field references:

```rust
/// Inline form state for spouse credit editing.
struct SpouseEditState {
    name: Entity<TextInput>,
    birth_number: Entity<TextInput>,
    income: Entity<NumberInput>,
    ztp: Entity<Checkbox>,
    months: Entity<NumberInput>,
}

/// Inline form state for a single child credit.
struct ChildEditState {
    /// None = creating new, Some(id) = editing existing.
    editing_id: Option<i64>,
    name: Entity<TextInput>,
    birth_number: Entity<TextInput>,
    order: Entity<Select>,        // options: "1"/"2"/"3"
    months: Entity<NumberInput>,
    ztp: Entity<Checkbox>,
}

/// Inline form state for personal credits.
struct PersonalEditState {
    is_student: Entity<Checkbox>,
    student_months: Entity<NumberInput>,
    disability_level: Entity<Select>,  // options: "0"/"1"/"2"/"3"
}

/// Inline form state for a single deduction.
struct DeductionEditState {
    /// None = creating new, Some(id) = editing existing.
    editing_id: Option<i64>,
    category: Entity<Select>,     // mortgage/life_insurance/pension/donation/union_dues
    description: Entity<TextInput>,
    amount: Entity<NumberInput>,
}

/// Which section is currently being edited.
enum EditingSection {
    Spouse(SpouseEditState),
    Child(ChildEditState),
    Personal(PersonalEditState),
    Deduction(DeductionEditState),
}
```

### 3.2 Updated View Struct

```rust
pub struct TaxCreditsView {
    // -- Existing fields (unchanged) --
    service: Arc<TaxCreditsService>,
    year: i32,
    loading: bool,
    error: Option<String>,
    spouse_credit: Option<TaxSpouseCredit>,
    children: Vec<TaxChildCredit>,
    personal: Option<TaxPersonalCredits>,
    deductions: Vec<TaxDeduction>,

    // -- New fields --
    saving: bool,
    editing: Option<EditingSection>,
    confirm_dialog: Option<Entity<ConfirmDialog>>,
    /// Tracks what entity the confirm dialog is about to delete.
    pending_delete: Option<DeleteTarget>,
}

/// Identifies what to delete when the confirm dialog fires.
enum DeleteTarget {
    Spouse,
    Child(i64),
    Deduction(i64),
}
```

## 4. Section-by-Section Design

### 4.1 Spouse Credit Section

**Display mode (current behavior):**
Shows spouse name, birth number, income, ZTP status, months claimed, and computed credit amount in label-value pairs. If no spouse credit exists, shows an empty-state message.

**New elements in display mode:**
- "Upravit" (edit) button in the card header, visible when a spouse credit exists.
- "Pridat" (add) button in the empty-state area, visible when no spouse credit exists.
- "Smazat" (delete) button next to "Upravit", triggers `ConfirmDialog`.

**Edit mode:**
Replaces the label-value pairs with form fields:

| Field | Component | Placeholder | Validation |
|-------|-----------|-------------|------------|
| Jmeno | `TextInput` | "Jmeno manzelky/manzela..." | Required, non-empty |
| Rodne cislo | `TextInput` | "XXXXXX/XXXX" | Required, non-empty |
| Prijem | `NumberInput` | "0.00" | Required, >= 0 |
| ZTP/P | `Checkbox` | "Drzitel ZTP/P" | n/a |
| Mesicu | `NumberInput` (integer_only) | "12" | Required, 1-12 |

Buttons: "Ulozit" (save) and "Zrusit" (cancel).

**Data flow -- save:**

```rust
fn save_spouse(&mut self, cx: &mut Context<Self>) {
    let EditingSection::Spouse(ref state) = self.editing.as_ref().unwrap() else { return };

    // 1. Read values from Entity<T> via .read(cx).value()
    let name = state.name.read(cx).value().to_string();
    let birth_number = state.birth_number.read(cx).value().to_string();
    let income = state.income.read(cx).to_amount().unwrap_or(Amount::ZERO);
    let ztp = state.ztp.read(cx).is_checked();
    let months: i32 = state.months.read(cx).value().parse().unwrap_or(12);

    // 2. Validate
    if name.trim().is_empty() || birth_number.trim().is_empty() {
        self.error = Some("Jmeno a rodne cislo jsou povinne.".into());
        cx.notify();
        return;
    }
    if months < 1 || months > 12 {
        self.error = Some("Pocet mesicu musi byt 1-12.".into());
        cx.notify();
        return;
    }

    // 3. Build domain struct
    let mut credit = TaxSpouseCredit {
        id: self.spouse_credit.as_ref().map(|s| s.id).unwrap_or(0),
        year: self.year,
        spouse_name: name,
        spouse_birth_number: birth_number,
        spouse_income: income,
        spouse_ztp: ztp,
        months_claimed: months,
        credit_amount: Amount::ZERO,  // computed by repo/service
        created_at: chrono::Local::now().naive_local(),
        updated_at: chrono::Local::now().naive_local(),
    };

    // 4. Spawn background task
    self.saving = true;
    self.error = None;
    cx.notify();

    let service = self.service.clone();
    cx.spawn(async move |this, cx| {
        let result = cx.background_executor().spawn(async move {
            service.upsert_spouse(&mut credit)?;
            Ok::<(), DomainError>(())
        }).await;

        this.update(cx, |this, cx| {
            this.saving = false;
            match result {
                Ok(()) => {
                    this.editing = None;
                    this.load_data(cx);  // reload all data to get computed amounts
                }
                Err(e) => {
                    this.error = Some(format!("Chyba pri ukladani: {e}"));
                }
            }
            cx.notify();
        }).ok();
    }).detach();
}
```

**Data flow -- delete:**

```rust
fn request_delete_spouse(&mut self, cx: &mut Context<Self>) {
    self.pending_delete = Some(DeleteTarget::Spouse);
    let dialog = cx.new(|_cx| {
        ConfirmDialog::new(
            "Smazat slevu na manzela/ku",
            "Opravdu chcete smazat slevu na manzela/ku?",
            "Smazat",
        )
    });
    cx.subscribe(&dialog, Self::on_confirm_result).detach();
    self.confirm_dialog = Some(dialog);
    cx.notify();
}
```

### 4.2 Children Credits Section

**Display mode (current behavior):**
Lists children as compact rows showing order, name, months, ZTP status, and computed credit amount.

**New elements in display mode:**
- "Pridat dite" button in the card header.
- Per-child "Upravit" and "Smazat" icon buttons at the end of each row.

**Edit mode (inline form below the list):**
When adding or editing, an inline form appears at the bottom of the children list:

| Field | Component | Placeholder | Validation |
|-------|-----------|-------------|------------|
| Jmeno | `TextInput` | "Jmeno ditete..." | Required |
| Rodne cislo | `TextInput` | "XXXXXX/XXXX" | Required |
| Poradi | `Select` | options: 1. dite / 2. dite / 3.+ dite | Required, values "1"/"2"/"3" |
| Mesicu | `NumberInput` (integer_only) | "12" | Required, 1-12 |
| ZTP/P | `Checkbox` | "Drzitel ZTP/P" | n/a |

Buttons: "Ulozit" and "Zrusit".

**Data flow -- create:**

```rust
fn save_child(&mut self, cx: &mut Context<Self>) {
    let EditingSection::Child(ref state) = self.editing.as_ref().unwrap() else { return };

    let name = state.name.read(cx).value().to_string();
    let birth_number = state.birth_number.read(cx).value().to_string();
    let order: i32 = state.order.read(cx)
        .selected_value()
        .and_then(|v| v.parse().ok())
        .unwrap_or(1);
    let months: i32 = state.months.read(cx).value().parse().unwrap_or(12);
    let ztp = state.ztp.read(cx).is_checked();

    // Validate...

    let mut credit = TaxChildCredit {
        id: state.editing_id.unwrap_or(0),
        year: self.year,
        child_name: name,
        birth_number,
        child_order: order,
        months_claimed: months,
        ztp,
        credit_amount: Amount::ZERO,  // computed by repo
        created_at: chrono::Local::now().naive_local(),
        updated_at: chrono::Local::now().naive_local(),
    };

    let service = self.service.clone();
    let is_new = state.editing_id.is_none();

    cx.spawn(async move |this, cx| {
        let result = cx.background_executor().spawn(async move {
            if is_new {
                service.create_child(&mut credit)?;
            } else {
                service.update_child(&mut credit)?;
            }
            Ok::<(), DomainError>(())
        }).await;

        this.update(cx, |this, cx| {
            this.saving = false;
            match result {
                Ok(()) => {
                    this.editing = None;
                    this.load_data(cx);
                }
                Err(e) => this.error = Some(format!("Chyba pri ukladani: {e}")),
            }
            cx.notify();
        }).ok();
    }).detach();
}
```

**Edit flow:**
When the user clicks "Upravit" on a child row, the inline form populates with that child's data. The `ChildEditState.editing_id` is set to `Some(child.id)`.

```rust
fn start_edit_child(&mut self, child: &TaxChildCredit, cx: &mut Context<Self>) {
    let child_id = child.id;
    let name = cx.new(|cx| {
        let mut t = TextInput::new(
            SharedString::from(format!("child-name-{}", child_id)),
            "Jmeno ditete...",
            cx,
        );
        t.set_value(&child.child_name, cx);
        t
    });
    let birth_number = cx.new(|cx| {
        let mut t = TextInput::new(
            SharedString::from(format!("child-bn-{}", child_id)),
            "XXXXXX/XXXX",
            cx,
        );
        t.set_value(&child.birth_number, cx);
        t
    });
    let order = cx.new(|_cx| {
        let mut sel = Select::new(
            SharedString::from(format!("child-order-{}", child_id)),
            "Poradi...",
            child_order_options(),
        );
        sel.set_selected_value(&child.child_order.to_string(), _cx);
        sel
    });
    let months = cx.new(|cx| {
        NumberInput::new(
            SharedString::from(format!("child-months-{}", child_id)),
            "12",
            cx,
        )
        .integer_only()
        .with_value(child.months_claimed.to_string())
    });
    let ztp = cx.new(|_cx| Checkbox::new(
        SharedString::from(format!("child-ztp-{}", child_id)),
        "Drzitel ZTP/P",
        child.ztp,
    ));

    self.editing = Some(EditingSection::Child(ChildEditState {
        editing_id: Some(child_id),
        name,
        birth_number,
        order,
        months,
        ztp,
    }));
    self.error = None;
    cx.notify();
}
```

### 4.3 Personal Credits Section

**Display mode (current behavior):**
Shows student credit (if student) and disability credit (if disability_level > 0), or an empty-state message.

**New elements in display mode:**
- "Upravit" button in the card header (always visible -- personal credits are upserted).

**Edit mode:**
Replaces display content with form fields:

| Field | Component | Options/Placeholder | Validation |
|-------|-----------|---------------------|------------|
| Student | `Checkbox` | "Student" | n/a |
| Mesicu studia | `NumberInput` (integer_only) | "12" | 0-12, shown only when student is checked |
| Stupen invalidity | `Select` | "Zadny" / "I./II. stupen" / "III. stupen" / "ZTP/P" (values "0"/"1"/"2"/"3") | Required |

Buttons: "Ulozit" and "Zrusit".

**Note on conditional visibility:** When `is_student` checkbox is unchecked, the `student_months` field is still rendered but visually dimmed (opacity 0.5) and its value is ignored on save (treated as 0). This avoids GPUI layout reflows from conditional child insertion.

**Data flow -- save:**

```rust
fn save_personal(&mut self, cx: &mut Context<Self>) {
    let EditingSection::Personal(ref state) = self.editing.as_ref().unwrap() else { return };

    let is_student = state.is_student.read(cx).is_checked();
    let student_months: i32 = if is_student {
        state.student_months.read(cx).value().parse().unwrap_or(0)
    } else {
        0
    };
    let disability_level: i32 = state.disability_level.read(cx)
        .selected_value()
        .and_then(|v| v.parse().ok())
        .unwrap_or(0);

    let mut credits = TaxPersonalCredits {
        year: self.year,
        is_student,
        student_months,
        disability_level,
        credit_student: Amount::ZERO,     // computed by repo
        credit_disability: Amount::ZERO,  // computed by repo
        created_at: chrono::Local::now().naive_local(),
        updated_at: chrono::Local::now().naive_local(),
    };

    let service = self.service.clone();
    self.saving = true;
    cx.notify();

    cx.spawn(async move |this, cx| {
        let result = cx.background_executor().spawn(async move {
            service.upsert_personal(&mut credits)
        }).await;

        this.update(cx, |this, cx| {
            this.saving = false;
            match result {
                Ok(()) => {
                    this.editing = None;
                    this.load_data(cx);
                }
                Err(e) => this.error = Some(format!("Chyba pri ukladani: {e}")),
            }
            cx.notify();
        }).ok();
    }).detach();
}
```

### 4.4 Deductions Section

**Display mode (current behavior):**
Lists deductions as rows showing category label, description, and allowed amount.

**New elements in display mode:**
- "Pridat odpocet" button in the card header.
- Per-deduction "Upravit" and "Smazat" buttons at the end of each row.

**Edit mode (inline form below the list):**

| Field | Component | Options/Placeholder | Validation |
|-------|-----------|---------------------|------------|
| Kategorie | `Select` | Hypoteka / Zivotni pojisteni / Penzijni pripojisteni / Dar / Odbory | Required |
| Popis | `TextInput` | "Popis odpoctu..." | Required |
| Castka | `NumberInput` | "0.00" | Required, > 0 |

Buttons: "Ulozit" and "Zrusit".

**Data flow** mirrors the child credit pattern: `create_deduction` for new, `update_deduction` for existing, `delete_deduction` via `ConfirmDialog`.

**Category select options:**

```rust
fn deduction_category_options() -> Vec<SelectOption> {
    vec![
        SelectOption { value: "mortgage".into(), label: "Hypoteka".into() },
        SelectOption { value: "life_insurance".into(), label: "Zivotni pojisteni".into() },
        SelectOption { value: "pension".into(), label: "Penzijni pripojisteni".into() },
        SelectOption { value: "donation".into(), label: "Dar".into() },
        SelectOption { value: "union_dues".into(), label: "Odbory".into() },
    ]
}
```

Category string-to-enum mapping for save:

```rust
fn parse_deduction_category(s: &str) -> Option<DeductionCategory> {
    match s {
        "mortgage" => Some(DeductionCategory::Mortgage),
        "life_insurance" => Some(DeductionCategory::LifeInsurance),
        "pension" => Some(DeductionCategory::Pension),
        "donation" => Some(DeductionCategory::Donation),
        "union_dues" => Some(DeductionCategory::UnionDues),
        _ => None,
    }
}
```

### 4.5 Copy From Previous Year

**UI placement:** A secondary button "Kopirovat z roku {year-1}" in the page header, next to the year selector.

**Behavior:**
1. User clicks the button.
2. `ConfirmDialog` appears: "Kopirovat slevy a odpocty z roku {year-1}? Existujici data pro rok {year} budou nahrazena."
3. On confirmation, the view:
   a. Loads all data from `year - 1` via the service.
   b. For each entity, clones it into `year` (resetting IDs, updating `year` field).
   c. Calls appropriate create/upsert methods.
   d. Reloads data for the current year.

**Implementation:**

```rust
fn copy_from_previous_year(&mut self, cx: &mut Context<Self>) {
    let service = self.service.clone();
    let target_year = self.year;
    let source_year = target_year - 1;

    self.saving = true;
    self.error = None;
    cx.notify();

    cx.spawn(async move |this, cx| {
        let result = cx.background_executor().spawn(async move {
            // 1. Load source year data
            let spouse = service.get_spouse(source_year).ok();
            let children = service.list_children(source_year)?;
            let personal = service.get_personal(source_year).ok();
            let deductions = service.list_deductions(source_year)?;

            // 2. Copy spouse
            if let Some(sc) = spouse {
                let mut new_sc = sc.clone();
                new_sc.id = 0;
                new_sc.year = target_year;
                service.upsert_spouse(&mut new_sc)?;
            }

            // 3. Copy children
            for child in &children {
                let mut new_child = child.clone();
                new_child.id = 0;
                new_child.year = target_year;
                service.create_child(&mut new_child)?;
            }

            // 4. Copy personal
            if let Some(pc) = personal {
                let mut new_pc = pc.clone();
                new_pc.year = target_year;
                service.upsert_personal(&mut new_pc)?;
            }

            // 5. Copy deductions
            for ded in &deductions {
                let mut new_ded = ded.clone();
                new_ded.id = 0;
                new_ded.year = target_year;
                service.create_deduction(&mut new_ded)?;
            }

            Ok::<(), DomainError>(())
        }).await;

        this.update(cx, |this, cx| {
            this.saving = false;
            match result {
                Ok(()) => this.load_data(cx),
                Err(e) => {
                    this.error = Some(format!("Chyba pri kopirovani: {e}"));
                    cx.notify();
                }
            }
        }).ok();
    }).detach();
}
```

**Edge case:** If the source year has no data at all, the copy succeeds silently (no-op for each missing entity). This is intentional -- the user already confirmed and the result is simply an empty year.

## 5. Confirm Dialog Handling

All delete operations share a single `on_confirm_result` handler that dispatches based on `pending_delete`:

```rust
fn on_confirm_result(
    &mut self,
    _dialog: Entity<ConfirmDialog>,
    event: &ConfirmDialogResult,
    cx: &mut Context<Self>,
) {
    match event {
        ConfirmDialogResult::Confirmed => {
            if let Some(target) = self.pending_delete.take() {
                self.confirm_dialog = None;
                match target {
                    DeleteTarget::Spouse => self.do_delete_spouse(cx),
                    DeleteTarget::Child(id) => self.do_delete_child(id, cx),
                    DeleteTarget::Deduction(id) => self.do_delete_deduction(id, cx),
                }
            }
        }
        ConfirmDialogResult::Cancelled => {
            self.pending_delete = None;
            self.confirm_dialog = None;
            cx.notify();
        }
    }
}
```

Delete execution follows the same background-executor pattern as save:

```rust
fn do_delete_spouse(&mut self, cx: &mut Context<Self>) {
    let service = self.service.clone();
    let year = self.year;
    self.saving = true;
    cx.notify();

    cx.spawn(async move |this, cx| {
        let result = cx.background_executor().spawn(async move {
            service.delete_spouse(year)
        }).await;

        this.update(cx, |this, cx| {
            this.saving = false;
            match result {
                Ok(()) => {
                    this.editing = None;
                    this.load_data(cx);
                }
                Err(e) => this.error = Some(format!("Chyba pri mazani: {e}")),
            }
            cx.notify();
        }).ok();
    }).detach();
}

fn do_delete_child(&mut self, id: i64, cx: &mut Context<Self>) {
    let service = self.service.clone();
    self.saving = true;
    cx.notify();

    cx.spawn(async move |this, cx| {
        let result = cx.background_executor().spawn(async move {
            service.delete_child(id)
        }).await;

        this.update(cx, |this, cx| {
            this.saving = false;
            match result {
                Ok(()) => {
                    // Clear editing if we deleted the child being edited
                    if let Some(EditingSection::Child(ref state)) = this.editing {
                        if state.editing_id == Some(id) {
                            this.editing = None;
                        }
                    }
                    this.load_data(cx);
                }
                Err(e) => this.error = Some(format!("Chyba pri mazani: {e}")),
            }
            cx.notify();
        }).ok();
    }).detach();
}

fn do_delete_deduction(&mut self, id: i64, cx: &mut Context<Self>) {
    // Same pattern as do_delete_child, calling service.delete_deduction(id)
}
```

## 6. Computed Amounts

Credit amounts (`credit_amount`, `credit_student`, `credit_disability`, `max_amount`, `allowed_amount`) are **not** computed in the view. They are computed by the repository layer (or the calc modules in `zfaktury-core/src/calc/`) during `upsert`/`create`/`update` operations.

After every successful save, the view calls `self.load_data(cx)` which reloads all data from the database, including the freshly computed amounts. These amounts are then displayed as read-only values in the card display mode.

In edit mode, no computed amount preview is shown. The user saves first, then sees the computed result in display mode. This keeps the view logic simple and avoids duplicating the computation logic from `calc/credits.rs` and `calc/deductions.rs`.

## 7. Error Handling

### 7.1 Client-Side Validation (Before Save)

Each save method validates required fields before spawning the background task:

| Section | Validation Rules |
|---------|-----------------|
| Spouse | `spouse_name` non-empty, `birth_number` non-empty, `months_claimed` in 1..=12 |
| Child | `child_name` non-empty, `birth_number` non-empty, `child_order` selected (1/2/3), `months_claimed` in 1..=12 |
| Personal | `student_months` in 0..=12 when student is checked, `disability_level` is valid option |
| Deduction | `category` selected, `description` non-empty, `claimed_amount` > 0 |

Validation errors are shown in `self.error` (the existing red error banner), not as per-field errors. This matches the `SettingsCategoriesView` pattern.

### 7.2 Server-Side Errors

`DomainError::InvalidInput` from the service (e.g., year out of range, months out of range) is caught in the `.update()` callback and displayed in the error banner:

```rust
Err(e) => this.error = Some(format!("Chyba pri ukladani: {e}")),
```

### 7.3 Year Change During Edit

If the user clicks the year navigation buttons while a form is open, the editing state is discarded:

```rust
fn change_year(&mut self, delta: i32, cx: &mut Context<Self>) {
    self.year += delta;
    self.loading = true;
    self.error = None;
    self.editing = None;       // <-- discard any open form
    self.pending_delete = None;
    self.confirm_dialog = None;
    cx.notify();
    self.load_data(cx);
}
```

## 8. Render Structure

The render method is restructured to delegate each card to a method that handles both display and edit modes:

```rust
impl Render for TaxCreditsView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("tax-credits-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll();

        // Header with year selector + copy button
        outer = outer.child(self.render_header(cx));

        if self.loading {
            return outer.child(/* loading indicator */);
        }
        if let Some(ref error) = self.error {
            outer = outer.child(/* error banner */);
        }

        // 2-column grid: spouse + children
        outer = outer.child(
            div().flex().gap_4()
                .child(div().flex_1().child(self.render_spouse_card(cx)))
                .child(div().flex_1().child(self.render_children_card(cx))),
        );

        // 2-column grid: personal + deductions
        outer = outer.child(
            div().flex().gap_4()
                .child(div().flex_1().child(self.render_personal_card(cx)))
                .child(div().flex_1().child(self.render_deductions_card(cx))),
        );

        // Confirm dialog overlay
        if let Some(ref dialog) = self.confirm_dialog {
            outer = outer.child(dialog.clone());
        }

        outer
    }
}
```

Each `render_*_card` method now takes `cx: &mut Context<Self>` (needed for `cx.listener()`) and checks `self.editing` to decide whether to render the display or form variant.

### 8.1 Card Header Pattern

Each card header follows this layout:

```rust
// Card header with title and action buttons
div()
    .flex()
    .items_center()
    .justify_between()
    .child(
        div()
            .text_sm()
            .font_weight(FontWeight::SEMIBOLD)
            .text_color(rgb(ZfColors::TEXT_PRIMARY))
            .child("Section Title"),
    )
    .child(
        div()
            .flex()
            .gap_1()
            .child(render_button("edit-btn", "Upravit", ButtonVariant::Secondary, ...))
            .child(render_button("delete-btn", "Smazat", ButtonVariant::Danger, ...)),
    )
```

### 8.2 Form Field Layout Pattern

Form fields within a card use a vertical flex layout with labels:

```rust
// Reuse the existing render_form_field helper from text_input.rs
// For NumberInput and Select, add similar labeled wrappers:

fn render_labeled_field(label: &str, child: impl IntoElement) -> Div {
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
        .child(child)
}
```

## 9. Helper Functions

### 9.1 Select Option Builders

```rust
fn child_order_options() -> Vec<SelectOption> {
    vec![
        SelectOption { value: "1".into(), label: "1. dite".into() },
        SelectOption { value: "2".into(), label: "2. dite".into() },
        SelectOption { value: "3".into(), label: "3.+ dite".into() },
    ]
}

fn disability_level_options() -> Vec<SelectOption> {
    vec![
        SelectOption { value: "0".into(), label: "Zadny".into() },
        SelectOption { value: "1".into(), label: "I./II. stupen".into() },
        SelectOption { value: "2".into(), label: "III. stupen".into() },
        SelectOption { value: "3".into(), label: "ZTP/P".into() },
    ]
}

fn deduction_category_options() -> Vec<SelectOption> {
    vec![
        SelectOption { value: "mortgage".into(), label: "Hypoteka".into() },
        SelectOption { value: "life_insurance".into(), label: "Zivotni pojisteni".into() },
        SelectOption { value: "pension".into(), label: "Penzijni pripojisteni".into() },
        SelectOption { value: "donation".into(), label: "Dar".into() },
        SelectOption { value: "union_dues".into(), label: "Odbory".into() },
    ]
}
```

### 9.2 Deduction Category Parsing

```rust
fn parse_deduction_category(s: &str) -> Option<DeductionCategory> {
    match s {
        "mortgage" => Some(DeductionCategory::Mortgage),
        "life_insurance" => Some(DeductionCategory::LifeInsurance),
        "pension" => Some(DeductionCategory::Pension),
        "donation" => Some(DeductionCategory::Donation),
        "union_dues" => Some(DeductionCategory::UnionDues),
        _ => None,
    }
}

fn deduction_category_to_value(cat: DeductionCategory) -> &'static str {
    match cat {
        DeductionCategory::Mortgage => "mortgage",
        DeductionCategory::LifeInsurance => "life_insurance",
        DeductionCategory::Pension => "pension",
        DeductionCategory::Donation => "donation",
        DeductionCategory::UnionDues => "union_dues",
    }
}
```

## 10. Mutual Exclusion of Editing Sections

Only one section can be in edit mode at a time. When the user clicks an edit/add button, any previously open form is discarded:

```rust
fn start_editing(&mut self, section: EditingSection, cx: &mut Context<Self>) {
    self.editing = Some(section);
    self.error = None;
    cx.notify();
}
```

The edit/add buttons for other sections are disabled (via the `disabled` param of `render_button`) while any section is being edited:

```rust
let has_editing = self.editing.is_some();

// In each card's header:
render_button(
    "spouse-edit-btn",
    "Upravit",
    ButtonVariant::Secondary,
    has_editing || self.saving,  // disabled when editing or saving
    false,
    cx.listener(|this, _event: &ClickEvent, _window, cx| {
        // ...
    }),
)
```

## 11. Modified Files

| File | Change |
|------|--------|
| `zfaktury-app/src/views/tax_credits.rs` | Major enhancement: ~350 lines of display-only code becomes ~900-1100 lines with inline forms, editing state structs, CRUD methods, confirm dialog handling, and copy-from-year feature |

No new files are created. No changes to other existing files. The service layer, repository traits, and domain types already support all required operations.

## 12. New Imports Required

```rust
use crate::components::checkbox::Checkbox;
use crate::components::confirm_dialog::{ConfirmDialog, ConfirmDialogResult};
use crate::components::number_input::NumberInput;
use crate::components::select::{Select, SelectOption};
use crate::components::text_input::TextInput;
use zfaktury_domain::{Amount, DeductionCategory, DomainError};
```

## 13. Testing Plan

This RFC adds no new service or repository logic, so no new unit tests are needed in `zfaktury-core` or `zfaktury-db`.

Manual testing checklist:

1. **Spouse credit:** Add new -> verify computed amount displays. Edit -> change months -> verify recalculated. Delete -> confirm dialog -> verify removed.
2. **Children:** Add 3 children with different orders. Edit second child's months. Delete first child. Verify list updates correctly.
3. **Personal credits:** Toggle student on/off. Change disability level. Verify computed amounts after save.
4. **Deductions:** Add mortgage + pension deductions. Verify allowed amounts respect caps. Edit description. Delete one. Verify list updates.
5. **Copy from year:** Populate year 2025 with data. Navigate to 2026. Click copy. Verify all entities copied.
6. **Year change:** Open a form, change year. Verify form is discarded and data reloads.
7. **Concurrent guard:** While saving (spinner shown), verify all edit/add/delete buttons are disabled.
8. **Validation:** Submit spouse form with empty name. Verify error message. Submit child with months = 0. Verify error.
9. **Error recovery:** After a validation error, correct the field and save successfully. Verify error clears.

## 14. Open Questions

1. **Should "copy from year" delete existing data first?** Current proposal overwrites (upsert) for spouse/personal and appends for children/deductions. If the target year already has children, the copy would duplicate them. Consider deleting all target-year data before copying, or showing a warning if data exists.

2. **Toast notifications vs. error banner.** The current design uses the inline error banner (`self.error`) for both validation and server errors. An alternative is to use `ToastView` for success messages ("Ulozeno") and keep the banner for errors only. This would require the view to hold an `Entity<ToastView>` or emit an event to the root view. Deferred to implementation -- start with error banner only, add toast if UX feels incomplete.

3. **Inline computed amount preview.** Currently, computed amounts are only shown after save + reload. An enhancement would be to compute them client-side in real-time as the user edits fields (using `calc::credits` and `calc::deductions` functions). This adds complexity and is deferred to a follow-up RFC.
