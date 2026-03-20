# ZFaktury Rust -- GUI Development Guide

## GPUI Concepts

The GUI is built with [GPUI](https://github.com/zed-industries/zed), the Zed editor's framework. Key concepts:

### Render Trait

Every view implements `Render` with this signature:

```rust
impl Render for MyView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        div().child("Hello")
    }
}
```

- `&mut self` -- mutable access to view state
- `&mut Window` -- window handle for measurements, focus, etc.
- `&mut Context<Self>` -- GPUI context for spawning tasks, subscribing to events, creating child entities

### Entities

Views are wrapped in `Entity<T>` (formerly `ModelHandle`). Created via `cx.new(|cx| MyView::new(cx))`. Entities are reference-counted and can be cloned cheaply.

### Reactivity

GPUI re-renders when `cx.notify()` is called. There are no signals or stores -- views hold their state directly as struct fields and call `cx.notify()` after mutations.

## Navigation

### Route Enum

All routes are defined in `src/navigation.rs` as `Route` variants:

```rust
pub enum Route {
    Dashboard,
    InvoiceList,
    InvoiceDetail(i64),
    // ...
}
```

### NavigationState

Mutable navigation state with history stack:

```rust
pub struct NavigationState {
    pub current: Route,
    history: Vec<Route>,
}
```

- `navigate(route)` -- pushes current to history, sets new route
- `go_back()` -- pops from history

### NavigateEvent

The `SidebarView` emits `NavigateEvent(Route)` on item click. `RootView` subscribes and calls `navigate_to()` to swap the content view.

## Data Loading Pattern

Views load data asynchronously using GPUI's spawn + background_executor:

```rust
fn load_data(&mut self, cx: &mut Context<Self>) {
    self.loading = true;
    let service = self.service.clone();

    cx.spawn(async move |this, cx| {
        // Move heavy work to background thread pool
        let result = cx
            .background_executor()
            .spawn(async move { service.list(&filter) })
            .await;

        // Update view state on the main thread
        this.update(cx, |this, cx| {
            this.loading = false;
            match result {
                Ok(data) => this.items = data,
                Err(e) => this.error = Some(e.to_string()),
            }
            cx.notify();
        });
    })
    .detach();
}
```

Key rules:
- Clone `Arc<Service>` before the spawn (services are `Send + Sync`)
- Use `background_executor().spawn()` for blocking repo calls (they use `rusqlite` which is not async)
- Update view state inside `this.update(cx, ...)` to get `&mut Self` + `&mut Context<Self>`
- Always call `cx.notify()` after changing view state

## Theme

Colors are defined as `u32` hex constants in `src/theme.rs`:

```rust
pub struct ZfColors;
impl ZfColors {
    pub const BG: u32 = 0x1e1d21;
    pub const SURFACE: u32 = 0x1e1e22;
    pub const ACCENT: u32 = 0x5e6ad2;
    pub const TEXT_PRIMARY: u32 = 0xececef;
    pub const TEXT_SECONDARY: u32 = 0xa0a0a8;
    // ... status colors, borders, etc.
}
```

Usage: `rgb(ZfColors::ACCENT)` in element builders.

## Sidebar Structure

The sidebar (`src/sidebar.rs`) is organized into:

1. **Top-level items:** Dashboard, Prehledy, Faktury, Sablony faktur, Naklady, Kontakty
2. **"Ucetnictvi" section:** DPH, Dan z prijmu, Slevy a odpocty, Zalohy, Investice
3. **"Nastaveni" section:** Firma, Email, Ciselne rady, Kategorie, PDF sablona, Import z Fakturoid, Audit log, Zalohy

All labels are in Czech. Active state is determined by matching the current route's discriminant plus parent-child relationships (e.g., `InvoiceNew` highlights "Faktury").

## Adding a New Screen

Step-by-step guide for adding a new view (e.g., a "Bank Statements" list):

### 1. Add route variant

In `src/navigation.rs`:

```rust
pub enum Route {
    // ...
    BankStatements,
}
```

Add the path match in `from_path()` and label in `label()`.

### 2. Create view file

Create `src/views/bank_statements.rs`:

```rust
use std::sync::Arc;
use gpui::*;
use crate::theme::ZfColors;

pub struct BankStatementsView {
    // service: Arc<BankService>,
    loading: bool,
    error: Option<String>,
    items: Vec<...>,
}

impl BankStatementsView {
    pub fn new(/* service, */ cx: &mut Context<Self>) -> Self {
        let mut view = Self { loading: true, error: None, items: Vec::new() };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) { /* ... */ }
}

impl Render for BankStatementsView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        // Build UI using div(), flex(), etc.
    }
}
```

### 3. Register in views/mod.rs

Add `pub mod bank_statements;` to `src/views/mod.rs`.

### 4. Wire in RootView

In `src/root.rs`:

1. Add `BankStatements(Entity<BankStatementsView>)` to `ContentView` enum
2. Add match arm in `create_content_view()`:
   ```rust
   Route::BankStatements => {
       ContentView::BankStatements(cx.new(|cx| BankStatementsView::new(cx)))
   }
   ```
3. Add match arm in `render()` for `ContentView::BankStatements(v) => v.clone().into_any_element()`

### 5. Add to sidebar (optional)

In `src/sidebar.rs`, add a `NavItem` to the appropriate group.

## Adding a New Form

### State fields pattern

```rust
pub struct MyFormView {
    // Form state
    name: String,
    amount_text: String,  // Text input, parsed to Amount on save
    date: Option<NaiveDate>,

    // UI state
    saving: bool,
    error: Option<String>,
    validation_errors: Vec<String>,
}
```

### Validation pattern

```rust
fn validate(&self) -> Vec<String> {
    let mut errors = Vec::new();
    if self.name.trim().is_empty() {
        errors.push("Nazev je povinny.".to_string());
    }
    if self.amount_text.parse::<f64>().is_err() {
        errors.push("Neplatna castka.".to_string());
    }
    errors
}
```

### Save pattern

```rust
fn save(&mut self, cx: &mut Context<Self>) {
    let errors = self.validate();
    if !errors.is_empty() {
        self.validation_errors = errors;
        cx.notify();
        return;
    }

    self.saving = true;
    let service = self.service.clone();
    let mut entity = /* build domain struct from form state */;

    cx.spawn(async move |this, cx| {
        let result = cx
            .background_executor()
            .spawn(async move { service.create(&mut entity) })
            .await;

        this.update(cx, |this, cx| {
            this.saving = false;
            match result {
                Ok(()) => { /* navigate away or show success */ }
                Err(e) => this.error = Some(e.to_string()),
            }
            cx.notify();
        });
    })
    .detach();
}
```

## Headless Testing

Screenshots can be taken without a running desktop session using `cage` (headless Wayland compositor) and `grim` (screenshot tool). Both are provided by `nix develop`.

### Usage

```bash
cd rust && nix develop --command bash -c "
  cargo build &&
  ./scripts/headless-screenshot.sh \
    './target/debug/zfaktury-app --route /invoices --exit-after 3' \
    /tmp/invoices.png \
    3
"
```

The script:
1. Creates an isolated `XDG_RUNTIME_DIR`
2. Launches `cage` with the `WLR_BACKENDS=headless` wlroots backend
3. Runs the app inside cage
4. Waits for rendering, then captures via `grim`
5. Kills cage and cleans up

This is useful for CI screenshots and visual regression testing.
