use gpui::*;

use crate::icons::Icons;
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;

const SIDEBAR_EXPANDED_WIDTH: f32 = 208.0;
const SIDEBAR_COLLAPSED_WIDTH: f32 = 48.0;

/// Quick action (e.g. "+ Nova faktura") shown under a nav item when expanded.
struct NavAction {
    route: Route,
    label: &'static str,
}

/// Navigation item with icon, optional quick actions, and optional children.
struct NavItem {
    route: Route,
    label: &'static str,
    icon: &'static str,
    actions: Vec<NavAction>,
    children: Vec<NavItem>,
}

impl NavItem {
    fn new(route: Route, label: &'static str, icon: &'static str) -> Self {
        Self {
            route,
            label,
            icon,
            actions: vec![],
            children: vec![],
        }
    }

    fn with_actions(mut self, actions: Vec<NavAction>) -> Self {
        self.actions = actions;
        self
    }

    fn with_children(mut self, children: Vec<NavItem>) -> Self {
        self.children = children;
        self
    }
}

/// Navigation group (section header + items).
struct NavGroup {
    section: &'static str,
    items: Vec<NavItem>,
}

/// Either a standalone item or a group with a section header.
enum NavEntry {
    Item(NavItem),
    Group(NavGroup),
}

fn build_nav() -> Vec<NavEntry> {
    vec![
        NavEntry::Item(NavItem::new(Route::Dashboard, "Dashboard", Icons::HOME)),
        NavEntry::Item(NavItem::new(Route::Reports, "Prehledy", Icons::BAR_CHART)),
        NavEntry::Item(
            NavItem::new(Route::InvoiceList, "Faktury", Icons::DOCUMENT_TEXT).with_actions(vec![
                NavAction {
                    route: Route::InvoiceNew,
                    label: "Nova faktura",
                },
            ]),
        ),
        NavEntry::Item(
            NavItem::new(
                Route::RecurringInvoiceList,
                "Sablony faktur",
                Icons::REFRESH_CW,
            )
            .with_actions(vec![NavAction {
                route: Route::RecurringInvoiceNew,
                label: "Nova sablona",
            }]),
        ),
        NavEntry::Item(
            NavItem::new(Route::ExpenseList, "Naklady", Icons::CREDIT_CARD).with_actions(vec![
                NavAction {
                    route: Route::ExpenseNew,
                    label: "Pridat naklad",
                },
                NavAction {
                    route: Route::ExpenseImport,
                    label: "Import dokladu",
                },
                NavAction {
                    route: Route::RecurringExpenseNew,
                    label: "Opakovany naklad",
                },
            ]),
        ),
        NavEntry::Item(
            NavItem::new(Route::ContactList, "Kontakty", Icons::USERS).with_actions(vec![
                NavAction {
                    route: Route::ContactNew,
                    label: "Novy kontakt",
                },
            ]),
        ),
        NavEntry::Group(NavGroup {
            section: "Ucetnictvi",
            items: vec![
                NavItem::new(Route::VATOverview, "DPH", Icons::GRID),
                NavItem::new(Route::TaxOverview, "Dan z prijmu", Icons::BOOKMARK).with_children(
                    vec![
                        NavItem::new(Route::TaxCredits, "Slevy a odpocty", Icons::SHIELD_CHECK),
                        NavItem::new(Route::TaxPrepayments, "Zalohy", Icons::CALENDAR),
                        NavItem::new(Route::TaxInvestments, "Investice", Icons::TRENDING_UP),
                    ],
                ),
            ],
        }),
        NavEntry::Group(NavGroup {
            section: "Nastaveni",
            items: vec![
                NavItem::new(Route::SettingsFirma, "Firma", Icons::BUILDING),
                NavItem::new(Route::SettingsEmail, "Email", Icons::ENVELOPE),
                NavItem::new(Route::SettingsSequences, "Ciselne rady", Icons::HASH),
                NavItem::new(Route::SettingsCategories, "Kategorie", Icons::TAG),
                NavItem::new(Route::SettingsPdf, "PDF sablona", Icons::DOCUMENT),
                NavItem::new(Route::ImportFakturoid, "Import z Fakturoid", Icons::UPLOAD),
                NavItem::new(Route::SettingsAuditLog, "Audit log", Icons::CLIPBOARD_CHECK),
                NavItem::new(Route::SettingsBackup, "Zalohy", Icons::DATABASE),
            ],
        }),
    ]
}

/// Simulate CSS `letter-spacing: wider` by inserting thin spaces between characters.
/// GPUI has no `letter_spacing()` method, so we manually space out the text.
fn spaced_uppercase(text: &str) -> String {
    let upper = text.to_uppercase();
    let chars: Vec<char> = upper.chars().collect();
    let mut result = String::with_capacity(chars.len() * 2);
    for (i, ch) in chars.iter().enumerate() {
        result.push(*ch);
        // Add a thin space between characters (not after last)
        if i < chars.len() - 1 {
            // Unicode thin space U+2009
            result.push('\u{2009}');
        }
    }
    result
}

/// Sidebar view with navigation, collapse/expand, hover overlay, icons, and quick actions.
pub struct SidebarView {
    current_route: Route,
    collapsed: bool,
    hovered: bool,
}

impl EventEmitter<NavigateEvent> for SidebarView {}

impl SidebarView {
    pub fn new(current_route: Route) -> Self {
        Self {
            current_route,
            collapsed: false,
            hovered: false,
        }
    }

    pub fn set_route(&mut self, route: Route) {
        self.current_route = route;
    }

    pub fn toggle_collapse(&mut self) {
        self.collapsed = !self.collapsed;
        self.hovered = false;
    }

    fn is_expanded(&self) -> bool {
        !self.collapsed || self.hovered
    }

    fn sidebar_width(&self) -> f32 {
        if self.collapsed && !self.hovered {
            SIDEBAR_COLLAPSED_WIDTH
        } else {
            SIDEBAR_EXPANDED_WIDTH
        }
    }

    fn is_active(&self, route: &Route) -> bool {
        std::mem::discriminant(&self.current_route) == std::mem::discriminant(route)
            || self.is_parent_active(route)
    }

    fn is_parent_active(&self, route: &Route) -> bool {
        match route {
            Route::InvoiceList => matches!(
                self.current_route,
                Route::InvoiceNew | Route::InvoiceEdit(_) | Route::InvoiceDetail(_)
            ),
            Route::ExpenseList => matches!(
                self.current_route,
                Route::ExpenseNew
                    | Route::ExpenseEdit(_)
                    | Route::ExpenseDetail(_)
                    | Route::ExpenseImport
                    | Route::ExpenseReview(_)
            ),
            Route::ContactList => matches!(
                self.current_route,
                Route::ContactNew | Route::ContactEdit(_) | Route::ContactDetail(_)
            ),
            Route::RecurringInvoiceList => matches!(
                self.current_route,
                Route::RecurringInvoiceNew | Route::RecurringInvoiceDetail(_)
            ),
            Route::VATOverview => matches!(
                self.current_route,
                Route::VATReturnNew
                    | Route::VATReturnDetail(_)
                    | Route::VATControlNew
                    | Route::VATControlDetail(_)
                    | Route::VIESNew
                    | Route::VIESDetail(_)
            ),
            Route::TaxOverview => matches!(
                self.current_route,
                Route::TaxIncomeNew
                    | Route::TaxIncomeDetail(_)
                    | Route::TaxSocialNew
                    | Route::TaxSocialDetail(_)
                    | Route::TaxHealthNew
                    | Route::TaxHealthDetail(_)
            ),
            _ => false,
        }
    }

    fn render_icon(&self, icon: &'static str, size: Pixels, color: impl Into<Hsla>) -> Svg {
        svg()
            .path(icon)
            .size(size)
            .flex_shrink_0()
            .text_color(color)
    }

    fn render_nav_item(&self, item: &NavItem, cx: &mut Context<Self>) -> Div {
        let active = self.is_active(&item.route);
        let expanded = self.is_expanded();
        let route = item.route.clone();

        // Active state: semi-transparent accent bg (matches SvelteKit #5e6ad226)
        // with lighter accent text (ACCENT_TEXT) for readability
        let bg_color = if active {
            rgba(0x5e6ad226)
        } else {
            rgba(0x00000000)
        };
        let text_col = if active {
            rgb(ZfColors::ACCENT_TEXT)
        } else {
            rgb(ZfColors::TEXT_SECONDARY)
        };
        let hover_bg = if active {
            rgba(0x5e6ad233)
        } else {
            rgb(ZfColors::SURFACE_HOVER)
        };
        let hover_text = if active {
            rgb(ZfColors::ACCENT_TEXT)
        } else {
            rgb(ZfColors::TEXT_PRIMARY)
        };
        let icon_color = text_col;

        let icon_el = self.render_icon(item.icon, px(16.0), icon_color);

        let mut row = div()
            .id(SharedString::from(format!("nav-{}", item.label)))
            .flex()
            .items_center()
            .rounded_md()
            .text_sm()
            .font_weight(FontWeight::MEDIUM)
            .bg(bg_color)
            .text_color(text_col)
            .cursor_pointer()
            .hover(move |s| s.bg(hover_bg).text_color(hover_text));

        // Active left accent border indicator
        if active {
            row = row.border_l_2().border_color(rgb(ZfColors::ACCENT));
        }

        row = row.on_click(cx.listener(move |this, _ev, _window, cx| {
            cx.emit(NavigateEvent(route.clone()));
            let _ = this;
        }));

        if expanded {
            // SvelteKit: gap-2.5(10px), px-2(8px), py-1.5(6px)
            row = row.gap(px(10.0)).px_2().py(px(6.0)).child(icon_el).child(
                div()
                    .overflow_hidden()
                    .text_ellipsis()
                    .whitespace_nowrap()
                    .child(item.label.to_string()),
            );
        } else {
            row = row.justify_center().py(px(6.0)).child(icon_el);
        }

        let mut container = div().child(row);

        // Quick actions and children (expanded only)
        if expanded {
            for action in &item.actions {
                container = container.child(self.render_action(action, cx));
            }
            for child in &item.children {
                container = container.child(self.render_child_item(child, cx));
            }
        }

        container
    }

    fn render_action(&self, action: &NavAction, cx: &mut Context<Self>) -> Stateful<Div> {
        let route = action.route.clone();
        let plus_icon = self.render_icon(Icons::PLUS, px(12.0), rgb(ZfColors::ACCENT));

        // SvelteKit: pl-8.5(34px), pr-2(8px), py-0.5(2px), gap-1.5(6px), text-xs
        div()
            .id(SharedString::from(format!("action-{}", action.label)))
            .flex()
            .items_center()
            .gap(px(6.0))
            .rounded_md()
            .pl(px(34.0))
            .pr_2()
            .py(px(2.0))
            .text_xs()
            .font_weight(FontWeight::MEDIUM)
            .text_color(rgb(ZfColors::ACCENT))
            .cursor_pointer()
            .hover(|s| s.bg(rgba(0x5e6ad226)))
            .on_click(cx.listener(move |this, _ev, _window, cx| {
                cx.emit(NavigateEvent(route.clone()));
                let _ = this;
            }))
            .child(plus_icon)
            .child(
                div()
                    .overflow_hidden()
                    .text_ellipsis()
                    .whitespace_nowrap()
                    .child(action.label.to_string()),
            )
    }

    fn render_child_item(&self, child: &NavItem, cx: &mut Context<Self>) -> Stateful<Div> {
        let active = self.is_active(&child.route);
        let route = child.route.clone();

        let text_col = if active {
            rgb(ZfColors::ACCENT_TEXT)
        } else {
            rgb(ZfColors::TEXT_SECONDARY)
        };
        let bg_color = if active {
            rgba(0x5e6ad226)
        } else {
            rgba(0x00000000)
        };
        let hover_bg = if active {
            rgba(0x5e6ad233)
        } else {
            rgb(ZfColors::SURFACE_HOVER)
        };
        let hover_text = if active {
            rgb(ZfColors::ACCENT_TEXT)
        } else {
            rgb(ZfColors::TEXT_PRIMARY)
        };

        let icon_el = self.render_icon(child.icon, px(12.0), text_col);

        // SvelteKit: pl-8.5(34px), pr-2(8px), py-1(4px), gap-1.5(6px), text-xs
        div()
            .id(SharedString::from(format!("child-{}", child.label)))
            .flex()
            .items_center()
            .gap(px(6.0))
            .rounded_md()
            .pl(px(34.0))
            .pr_2()
            .py(px(4.0))
            .text_xs()
            .font_weight(FontWeight::MEDIUM)
            .bg(bg_color)
            .text_color(text_col)
            .cursor_pointer()
            .hover(move |s| s.bg(hover_bg).text_color(hover_text))
            .on_click(cx.listener(move |this, _ev, _window, cx| {
                cx.emit(NavigateEvent(route.clone()));
                let _ = this;
            }))
            .child(icon_el)
            .child(
                div()
                    .overflow_hidden()
                    .text_ellipsis()
                    .whitespace_nowrap()
                    .child(child.label.to_string()),
            )
    }

    fn render_section_header(&self, label: &str) -> Div {
        if self.is_expanded() {
            // SvelteKit: mt-4(16px), mb-1.5(6px), px-2(8px)
            // text-[11px], font-medium, uppercase, tracking-wider, text-muted
            // Using TEXT_SECTION (0x4a4a52) to match SvelteKit --color-muted
            div()
                .mt_4()
                .mb(px(6.0))
                .px_2()
                .text_color(rgb(ZfColors::TEXT_SECTION))
                .text_size(px(11.0))
                .font_weight(FontWeight::MEDIUM)
                .child(spaced_uppercase(label))
        } else {
            // Collapsed: thin separator line
            div()
                .mt_3()
                .mb_1()
                .mx_1()
                .border_t_1()
                .border_color(rgb(ZfColors::BORDER_SUBTLE))
        }
    }

    fn render_toggle_button(&self, cx: &mut Context<Self>) -> Stateful<Div> {
        let icon_path = if self.collapsed {
            Icons::ARROW_RIGHT
        } else {
            Icons::ARROW_LEFT
        };
        let icon_el = self.render_icon(icon_path, px(16.0), rgb(ZfColors::TEXT_MUTED));

        // SvelteKit: shrink-0, rounded, p-1, text-tertiary, hover:text-primary
        div()
            .id("sidebar-toggle")
            .flex_shrink_0()
            .flex()
            .items_center()
            .justify_center()
            .rounded(px(4.0))
            .p(px(4.0))
            .cursor_pointer()
            .text_color(rgb(ZfColors::TEXT_MUTED))
            .hover(|s| {
                s.bg(rgb(ZfColors::SURFACE_HOVER))
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
            })
            .on_click(cx.listener(|this, _ev, _window, cx| {
                this.toggle_collapse();
                cx.notify();
            }))
            .child(icon_el)
    }

    fn render_header(&self, cx: &mut Context<Self>) -> Div {
        let expanded = self.is_expanded();
        let toggle = self.render_toggle_button(cx);

        // SvelteKit: h-12(48px), items-center, border-b, border-border, overflow-hidden
        let mut header = div()
            .h(px(48.0))
            .flex()
            .items_center()
            .border_b_1()
            .border_color(rgb(ZfColors::BORDER))
            .overflow_hidden();

        if expanded {
            // SvelteKit: px-3(12px), gap-2(8px), justify-between
            header = header
                .px_3()
                .gap_2()
                .justify_between()
                .child(
                    div()
                        .flex()
                        .items_center()
                        .gap(px(8.0))
                        .overflow_hidden()
                        .child(
                            div()
                                .text_sm()
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .overflow_hidden()
                                .text_ellipsis()
                                .whitespace_nowrap()
                                .child("ZFaktury"),
                        ),
                )
                .child(toggle);
        } else {
            header = header.justify_center().child(toggle);
        }

        header
    }
}

impl Render for SidebarView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let nav = build_nav();
        let expanded = self.is_expanded();
        let is_overlay = self.collapsed && self.hovered;

        // Navigation items
        // SvelteKit: flex-1, overflow-y-auto, px-2(8px), py-3(12px), space-y-0.5(2px)
        let mut nav_container = div()
            .id("sidebar-nav-scroll")
            .flex_1()
            .overflow_y_scroll()
            .px_2()
            .py_3()
            .flex()
            .flex_col()
            .gap(px(2.0));

        for entry in &nav {
            match entry {
                NavEntry::Item(item) => {
                    nav_container = nav_container.child(self.render_nav_item(item, cx));
                }
                NavEntry::Group(group) => {
                    nav_container = nav_container.child(self.render_section_header(group.section));
                    for item in &group.items {
                        nav_container = nav_container.child(self.render_nav_item(item, cx));
                    }
                }
            }
        }

        // Footer
        // SvelteKit: border-t, border-border, px-3(12px), py-2.5(10px)
        let footer = if expanded {
            Some(
                div()
                    .border_t_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .px_3()
                    .py(px(10.0))
                    .child(
                        div()
                            .text_size(px(11.0))
                            .text_color(rgb(ZfColors::TEXT_SECTION))
                            .child("ZFaktury v0.1.0"),
                    ),
            )
        } else {
            None
        };

        // Build the sidebar container
        let mut sidebar = div()
            .id("sidebar")
            .w(px(self.sidebar_width()))
            .h_full()
            .bg(rgb(ZfColors::SURFACE))
            .border_r_1()
            .border_color(rgb(ZfColors::BORDER))
            .flex()
            .flex_col()
            .overflow_hidden()
            .on_hover(cx.listener(|this, hovered: &bool, _window, cx| {
                if this.collapsed {
                    this.hovered = *hovered;
                    cx.notify();
                } else if this.hovered {
                    this.hovered = false;
                    cx.notify();
                }
            }));

        // Overlay mode: absolute positioning + shadow
        if is_overlay {
            sidebar = sidebar.absolute().top_0().left_0().shadow_xl();
        }

        sidebar = sidebar.child(self.render_header(cx)).child(nav_container);

        if let Some(footer) = footer {
            sidebar = sidebar.child(footer);
        }

        sidebar
    }
}
