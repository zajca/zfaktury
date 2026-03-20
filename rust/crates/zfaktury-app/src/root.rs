use std::sync::Arc;

use gpui::*;

use crate::app::AppServices;
use crate::navigation::{NavigationState, Route};
use crate::sidebar::{NavigateEvent, SidebarView};
use crate::theme::ZfColors;
use crate::views::contact_list::ContactListView;
use crate::views::dashboard::DashboardView;
use crate::views::expense_list::ExpenseListView;
use crate::views::invoice_detail::InvoiceDetailView;
use crate::views::invoice_form::InvoiceFormView;
use crate::views::invoice_list::InvoiceListView;
use crate::views::settings_firma::SettingsFirmaView;
use crate::views::stub::StubView;

/// Root view: sidebar + main content area. Routes to appropriate view based on
/// the current NavigationState.
pub struct RootView {
    services: Arc<AppServices>,
    nav: NavigationState,
    sidebar: Entity<SidebarView>,
    content: ContentView,
}

/// Wrapper for the current content view entity.
enum ContentView {
    Dashboard(Entity<DashboardView>),
    InvoiceList(Entity<InvoiceListView>),
    InvoiceDetail(Entity<InvoiceDetailView>),
    InvoiceForm(Entity<InvoiceFormView>),
    ExpenseList(Entity<ExpenseListView>),
    ContactList(Entity<ContactListView>),
    SettingsFirma(Entity<SettingsFirmaView>),
    Stub(Entity<StubView>),
}

impl RootView {
    pub fn new(services: Arc<AppServices>, initial_route: Route, cx: &mut Context<Self>) -> Self {
        let sidebar = cx.new(|_cx| SidebarView::new(initial_route.clone()));

        // Subscribe to navigation events from sidebar
        cx.subscribe(
            &sidebar,
            |this: &mut Self, _sidebar, event: &NavigateEvent, cx| {
                this.navigate_to(event.0.clone(), cx);
            },
        )
        .detach();

        let content = Self::create_content_view(&services, &initial_route, cx);
        let nav = NavigationState::new(initial_route);

        Self {
            services,
            nav,
            sidebar,
            content,
        }
    }

    /// Navigate to a new route, updating sidebar and content.
    fn navigate_to(&mut self, route: Route, cx: &mut Context<Self>) {
        self.nav.navigate(route);
        self.update_content(cx);
    }

    fn update_content(&mut self, cx: &mut Context<Self>) {
        let route = self.nav.current.clone();
        self.content = Self::create_content_view(&self.services, &route, cx);

        // Update sidebar route
        self.sidebar.update(cx, |sidebar, _cx| {
            sidebar.set_route(route);
        });

        cx.notify();
    }

    fn create_content_view(
        services: &Arc<AppServices>,
        route: &Route,
        cx: &mut Context<Self>,
    ) -> ContentView {
        match route {
            Route::Dashboard => {
                let svc = services.dashboard.clone();
                ContentView::Dashboard(cx.new(|cx| DashboardView::new(svc, cx)))
            }
            Route::InvoiceList => {
                let svc = services.invoices.clone();
                ContentView::InvoiceList(cx.new(|cx| InvoiceListView::new(svc, cx)))
            }
            Route::InvoiceNew => {
                ContentView::InvoiceForm(cx.new(|_cx| InvoiceFormView::new_create()))
            }
            Route::InvoiceDetail(id) => {
                let svc = services.invoices.clone();
                let id = *id;
                ContentView::InvoiceDetail(cx.new(|cx| InvoiceDetailView::new(svc, id, cx)))
            }
            Route::ExpenseList => {
                let svc = services.expenses.clone();
                ContentView::ExpenseList(cx.new(|cx| ExpenseListView::new(svc, cx)))
            }
            Route::ContactList => {
                let svc = services.contacts.clone();
                ContentView::ContactList(cx.new(|cx| ContactListView::new(svc, cx)))
            }
            Route::SettingsFirma => {
                let svc = services.settings.clone();
                ContentView::SettingsFirma(cx.new(|cx| SettingsFirmaView::new(svc, cx)))
            }
            other => {
                let label = other.label().to_string();
                ContentView::Stub(cx.new(|_cx| StubView::new(label)))
            }
        }
    }
}

impl Render for RootView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let content_element: AnyElement = match &self.content {
            ContentView::Dashboard(v) => v.clone().into_any_element(),
            ContentView::InvoiceList(v) => v.clone().into_any_element(),
            ContentView::InvoiceDetail(v) => v.clone().into_any_element(),
            ContentView::InvoiceForm(v) => v.clone().into_any_element(),
            ContentView::ExpenseList(v) => v.clone().into_any_element(),
            ContentView::ContactList(v) => v.clone().into_any_element(),
            ContentView::SettingsFirma(v) => v.clone().into_any_element(),
            ContentView::Stub(v) => v.clone().into_any_element(),
        };

        div()
            .flex()
            .size_full()
            .bg(rgb(ZfColors::BG))
            .child(self.sidebar.clone())
            .child(div().flex_1().overflow_hidden().child(content_element))
    }
}
