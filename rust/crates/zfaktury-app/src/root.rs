use std::sync::Arc;

use gpui::*;

use crate::app::AppServices;
use crate::navigation::{NavigateEvent, NavigationState, Route};
use crate::sidebar::SidebarView;
use crate::theme::ZfColors;
use crate::views::contact_detail::ContactDetailView;
use crate::views::contact_list::ContactListView;
use crate::views::dashboard::DashboardView;
use crate::views::expense_detail::ExpenseDetailView;
use crate::views::expense_form::ExpenseFormView;
use crate::views::expense_list::ExpenseListView;
use crate::views::import_fakturoid::ImportFakturoidView;
use crate::views::invoice_detail::InvoiceDetailView;
use crate::views::invoice_form::InvoiceFormView;
use crate::views::invoice_list::InvoiceListView;
use crate::views::recurring_expense_list::RecurringExpenseListView;
use crate::views::recurring_invoice_list::RecurringInvoiceListView;
use crate::views::reports::ReportsView;
use crate::views::settings_audit::SettingsAuditView;
use crate::views::settings_backup::SettingsBackupView;
use crate::views::settings_categories::SettingsCategoriesView;
use crate::views::settings_email::SettingsEmailView;
use crate::views::settings_firma::SettingsFirmaView;
use crate::views::settings_sequences::SettingsSequencesView;
use crate::views::stub::StubView;
use crate::views::tax_credits::TaxCreditsView;
use crate::views::tax_investments::TaxInvestmentsView;
use crate::views::tax_overview::TaxOverviewView;
use crate::views::tax_prepayments::TaxPrepaymentsView;
use crate::views::vat_overview::VatOverviewView;
use crate::views::vat_return_detail::VatReturnDetailView;

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
    ExpenseDetail(Entity<ExpenseDetailView>),
    ExpenseForm(Entity<ExpenseFormView>),
    ContactList(Entity<ContactListView>),
    ContactDetail(Entity<ContactDetailView>),
    RecurringInvoiceList(Entity<RecurringInvoiceListView>),
    RecurringExpenseList(Entity<RecurringExpenseListView>),
    VatOverview(Entity<VatOverviewView>),
    VatReturnDetail(Entity<VatReturnDetailView>),
    TaxOverview(Entity<TaxOverviewView>),
    TaxCredits(Entity<TaxCreditsView>),
    TaxPrepayments(Entity<TaxPrepaymentsView>),
    TaxInvestments(Entity<TaxInvestmentsView>),
    Reports(Entity<ReportsView>),
    SettingsFirma(Entity<SettingsFirmaView>),
    SettingsEmail(Entity<SettingsEmailView>),
    SettingsSequences(Entity<SettingsSequencesView>),
    SettingsCategories(Entity<SettingsCategoriesView>),
    SettingsAudit(Entity<SettingsAuditView>),
    SettingsBackup(Entity<SettingsBackupView>),
    ImportFakturoid(Entity<ImportFakturoidView>),
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
        Self::subscribe_to_content(&content, cx);
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
        Self::subscribe_to_content(&self.content, cx);

        // Update sidebar route
        self.sidebar.update(cx, |sidebar, _cx| {
            sidebar.set_route(route);
        });

        cx.notify();
    }

    /// Subscribe to NavigateEvent from the content view so that row clicks,
    /// "New" buttons, and form saves can trigger navigation.
    fn subscribe_to_content(content: &ContentView, cx: &mut Context<Self>) {
        macro_rules! subscribe_nav {
            ($entity:expr, $cx:expr) => {{
                $cx.subscribe($entity, |this: &mut Self, _, event: &NavigateEvent, cx| {
                    this.navigate_to(event.0.clone(), cx);
                })
                .detach()
            }};
        }

        match content {
            ContentView::Dashboard(e) => subscribe_nav!(e, cx),
            ContentView::InvoiceList(e) => subscribe_nav!(e, cx),
            ContentView::InvoiceDetail(e) => subscribe_nav!(e, cx),
            ContentView::InvoiceForm(e) => subscribe_nav!(e, cx),
            ContentView::ExpenseList(e) => subscribe_nav!(e, cx),
            ContentView::ExpenseDetail(e) => subscribe_nav!(e, cx),
            ContentView::ExpenseForm(e) => subscribe_nav!(e, cx),
            ContentView::ContactList(e) => subscribe_nav!(e, cx),
            ContentView::ContactDetail(e) => subscribe_nav!(e, cx),
            ContentView::RecurringInvoiceList(e) => subscribe_nav!(e, cx),
            ContentView::RecurringExpenseList(e) => subscribe_nav!(e, cx),
            ContentView::VatOverview(e) => subscribe_nav!(e, cx),
            ContentView::VatReturnDetail(e) => subscribe_nav!(e, cx),
            ContentView::TaxOverview(e) => subscribe_nav!(e, cx),
            ContentView::TaxCredits(e) => subscribe_nav!(e, cx),
            ContentView::TaxPrepayments(e) => subscribe_nav!(e, cx),
            ContentView::TaxInvestments(e) => subscribe_nav!(e, cx),
            ContentView::Reports(e) => subscribe_nav!(e, cx),
            ContentView::SettingsFirma(e) => subscribe_nav!(e, cx),
            ContentView::SettingsEmail(e) => subscribe_nav!(e, cx),
            ContentView::SettingsSequences(e) => subscribe_nav!(e, cx),
            ContentView::SettingsCategories(e) => subscribe_nav!(e, cx),
            ContentView::SettingsAudit(e) => subscribe_nav!(e, cx),
            ContentView::SettingsBackup(e) => subscribe_nav!(e, cx),
            ContentView::ImportFakturoid(e) => subscribe_nav!(e, cx),
            ContentView::Stub(_) => {} // Stubs don't emit navigation events
        }
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
            Route::InvoiceEdit(id) => {
                let id = *id;
                ContentView::InvoiceForm(cx.new(move |_cx| InvoiceFormView::new_edit(id)))
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
            Route::ExpenseNew => ContentView::ExpenseForm(cx.new(|_cx| ExpenseFormView::new())),
            Route::ExpenseEdit(id) => {
                let id = *id;
                ContentView::ExpenseForm(cx.new(move |_cx| ExpenseFormView::new_edit(id)))
            }
            Route::ExpenseDetail(id) => {
                let svc = services.expenses.clone();
                let id = *id;
                ContentView::ExpenseDetail(cx.new(|cx| ExpenseDetailView::new(svc, id, cx)))
            }
            Route::ContactList => {
                let svc = services.contacts.clone();
                ContentView::ContactList(cx.new(|cx| ContactListView::new(svc, cx)))
            }
            Route::ContactNew | Route::ContactEdit(_) => {
                let label = route.label().to_string();
                ContentView::Stub(cx.new(|_cx| StubView::new(label)))
            }
            Route::ContactDetail(id) => {
                let svc = services.contacts.clone();
                let id = *id;
                ContentView::ContactDetail(cx.new(|cx| ContactDetailView::new(svc, id, cx)))
            }
            Route::RecurringInvoiceList => {
                let svc = services.recurring_invoices.clone();
                ContentView::RecurringInvoiceList(
                    cx.new(|cx| RecurringInvoiceListView::new(svc, cx)),
                )
            }
            Route::RecurringExpenseList => {
                let svc = services.recurring_expenses.clone();
                ContentView::RecurringExpenseList(
                    cx.new(|cx| RecurringExpenseListView::new(svc, cx)),
                )
            }
            Route::VATOverview => {
                let svc = services.vat_returns.clone();
                ContentView::VatOverview(cx.new(|cx| VatOverviewView::new(svc, cx)))
            }
            Route::VATReturnDetail(id) => {
                let svc = services.vat_returns.clone();
                let id = *id;
                ContentView::VatReturnDetail(cx.new(|cx| VatReturnDetailView::new(svc, id, cx)))
            }
            Route::TaxOverview => ContentView::TaxOverview(cx.new(|_cx| TaxOverviewView::new())),
            Route::TaxCredits => ContentView::TaxCredits(cx.new(|_cx| TaxCreditsView::new())),
            Route::TaxPrepayments => {
                ContentView::TaxPrepayments(cx.new(|_cx| TaxPrepaymentsView::new()))
            }
            Route::TaxInvestments => {
                ContentView::TaxInvestments(cx.new(|_cx| TaxInvestmentsView::new()))
            }
            Route::Reports => {
                let svc = services.reports.clone();
                ContentView::Reports(cx.new(|cx| ReportsView::new(svc, cx)))
            }
            Route::SettingsFirma => {
                let svc = services.settings.clone();
                ContentView::SettingsFirma(cx.new(|cx| SettingsFirmaView::new(svc, cx)))
            }
            Route::SettingsEmail => {
                let svc = services.settings.clone();
                ContentView::SettingsEmail(cx.new(|cx| SettingsEmailView::new(svc, cx)))
            }
            Route::SettingsSequences => {
                let svc = services.sequences.clone();
                ContentView::SettingsSequences(cx.new(|cx| SettingsSequencesView::new(svc, cx)))
            }
            Route::SettingsCategories => {
                let svc = services.categories.clone();
                ContentView::SettingsCategories(cx.new(|cx| SettingsCategoriesView::new(svc, cx)))
            }
            Route::SettingsAuditLog => {
                let svc = services.audit.clone();
                ContentView::SettingsAudit(cx.new(|cx| SettingsAuditView::new(svc, cx)))
            }
            Route::SettingsBackup => {
                ContentView::SettingsBackup(cx.new(|_cx| SettingsBackupView::new()))
            }
            Route::ImportFakturoid => {
                ContentView::ImportFakturoid(cx.new(|_cx| ImportFakturoidView::new()))
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
            ContentView::ExpenseDetail(v) => v.clone().into_any_element(),
            ContentView::ExpenseForm(v) => v.clone().into_any_element(),
            ContentView::ContactList(v) => v.clone().into_any_element(),
            ContentView::ContactDetail(v) => v.clone().into_any_element(),
            ContentView::RecurringInvoiceList(v) => v.clone().into_any_element(),
            ContentView::RecurringExpenseList(v) => v.clone().into_any_element(),
            ContentView::VatOverview(v) => v.clone().into_any_element(),
            ContentView::VatReturnDetail(v) => v.clone().into_any_element(),
            ContentView::TaxOverview(v) => v.clone().into_any_element(),
            ContentView::TaxCredits(v) => v.clone().into_any_element(),
            ContentView::TaxPrepayments(v) => v.clone().into_any_element(),
            ContentView::TaxInvestments(v) => v.clone().into_any_element(),
            ContentView::Reports(v) => v.clone().into_any_element(),
            ContentView::SettingsFirma(v) => v.clone().into_any_element(),
            ContentView::SettingsEmail(v) => v.clone().into_any_element(),
            ContentView::SettingsSequences(v) => v.clone().into_any_element(),
            ContentView::SettingsCategories(v) => v.clone().into_any_element(),
            ContentView::SettingsAudit(v) => v.clone().into_any_element(),
            ContentView::SettingsBackup(v) => v.clone().into_any_element(),
            ContentView::ImportFakturoid(v) => v.clone().into_any_element(),
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
