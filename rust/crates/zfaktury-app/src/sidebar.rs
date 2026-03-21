use gpui::*;

use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;

/// Navigation item definition.
struct NavItem {
    route: Route,
    label: &'static str,
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
        NavEntry::Item(NavItem {
            route: Route::Dashboard,
            label: "Dashboard",
        }),
        NavEntry::Item(NavItem {
            route: Route::Reports,
            label: "Prehledy",
        }),
        NavEntry::Item(NavItem {
            route: Route::InvoiceList,
            label: "Faktury",
        }),
        NavEntry::Item(NavItem {
            route: Route::RecurringInvoiceList,
            label: "Sablony faktur",
        }),
        NavEntry::Item(NavItem {
            route: Route::ExpenseList,
            label: "Naklady",
        }),
        NavEntry::Item(NavItem {
            route: Route::ContactList,
            label: "Kontakty",
        }),
        NavEntry::Group(NavGroup {
            section: "Ucetnictvi",
            items: vec![
                NavItem {
                    route: Route::VATOverview,
                    label: "DPH",
                },
                NavItem {
                    route: Route::TaxOverview,
                    label: "Dan z prijmu",
                },
                NavItem {
                    route: Route::TaxCredits,
                    label: "Slevy a odpocty",
                },
                NavItem {
                    route: Route::TaxPrepayments,
                    label: "Zalohy",
                },
                NavItem {
                    route: Route::TaxInvestments,
                    label: "Investice",
                },
            ],
        }),
        NavEntry::Group(NavGroup {
            section: "Nastaveni",
            items: vec![
                NavItem {
                    route: Route::SettingsFirma,
                    label: "Firma",
                },
                NavItem {
                    route: Route::SettingsEmail,
                    label: "Email",
                },
                NavItem {
                    route: Route::SettingsSequences,
                    label: "Ciselne rady",
                },
                NavItem {
                    route: Route::SettingsCategories,
                    label: "Kategorie",
                },
                NavItem {
                    route: Route::SettingsPdf,
                    label: "PDF sablona",
                },
                NavItem {
                    route: Route::ImportFakturoid,
                    label: "Import z Fakturoid",
                },
                NavItem {
                    route: Route::SettingsAuditLog,
                    label: "Audit log",
                },
                NavItem {
                    route: Route::SettingsBackup,
                    label: "Zalohy",
                },
            ],
        }),
    ]
}

/// Sidebar view with navigation.
pub struct SidebarView {
    current_route: Route,
}

impl EventEmitter<NavigateEvent> for SidebarView {}

impl SidebarView {
    pub fn new(current_route: Route) -> Self {
        Self { current_route }
    }

    pub fn set_route(&mut self, route: Route) {
        self.current_route = route;
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
                    | Route::ExpenseReview
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

    fn render_nav_item(&self, item: &NavItem, cx: &mut Context<Self>) -> Stateful<Div> {
        let active = self.is_active(&item.route);
        let route = item.route.clone();

        let bg = if active {
            rgb(ZfColors::ACCENT_MUTED)
        } else {
            rgb(0x00000000) // transparent
        };
        let text_color = if active {
            rgb(ZfColors::ACCENT)
        } else {
            rgb(ZfColors::TEXT_SECONDARY)
        };

        div()
            .id(SharedString::from(format!("nav-{}", item.label)))
            .flex()
            .items_center()
            .px_3()
            .py(px(6.0))
            .rounded_md()
            .text_sm()
            .font_weight(FontWeight::MEDIUM)
            .bg(bg)
            .text_color(text_color)
            .cursor_pointer()
            .hover(|s| {
                if active {
                    s
                } else {
                    s.bg(rgb(ZfColors::SURFACE_HOVER))
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                }
            })
            .on_click(cx.listener(move |this, _ev, _window, cx| {
                cx.emit(NavigateEvent(route.clone()));
                let _ = this;
            }))
            .child(item.label.to_string())
    }

    fn render_section_header(&self, label: &str) -> Div {
        div()
            .mt_4()
            .mb_1()
            .px_3()
            .text_color(rgb(ZfColors::TEXT_MUTED))
            .text_xs()
            .font_weight(FontWeight::MEDIUM)
            .child(label.to_string().to_uppercase())
    }
}

impl Render for SidebarView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let nav = build_nav();

        let mut sidebar = div()
            .w(px(200.0))
            .h_full()
            .bg(rgb(ZfColors::SURFACE))
            .border_r_1()
            .border_color(rgb(ZfColors::BORDER))
            .flex()
            .flex_col()
            .overflow_hidden();

        // Logo header
        sidebar = sidebar.child(
            div()
                .h(px(48.0))
                .flex()
                .items_center()
                .px_3()
                .border_b_1()
                .border_color(rgb(ZfColors::BORDER))
                .child(
                    div()
                        .text_sm()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("ZFaktury"),
                ),
        );

        // Navigation
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

        sidebar = sidebar.child(nav_container);

        // Footer
        sidebar = sidebar.child(
            div()
                .border_t_1()
                .border_color(rgb(ZfColors::BORDER))
                .px_3()
                .py_2()
                .child(
                    div()
                        .text_xs()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child("ZFaktury v0.1.0"),
                ),
        );

        sidebar
    }
}
