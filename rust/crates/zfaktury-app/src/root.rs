use std::sync::Arc;

use gpui::*;

use crate::app::AppServices;

actions!(zfaktury, [ToggleSidebar]);
use crate::navigation::{NavigateEvent, NavigationState, Route};
use crate::sidebar::SidebarView;
use crate::theme::ZfColors;
use crate::views::contact_detail::ContactDetailView;
use crate::views::contact_form::ContactFormView;
use crate::views::contact_list::ContactListView;
use crate::views::dashboard::DashboardView;
use crate::views::expense_detail::ExpenseDetailView;
use crate::views::expense_form::ExpenseFormView;
use crate::views::expense_import::ExpenseImportView;
use crate::views::expense_list::ExpenseListView;
use crate::views::expense_review::ExpenseReviewView;
use crate::views::import_fakturoid::ImportFakturoidView;
use crate::views::invoice_detail::InvoiceDetailView;
use crate::views::invoice_form::InvoiceFormView;
use crate::views::invoice_list::InvoiceListView;
use crate::views::recurring_expense_detail::RecurringExpenseDetailView;
use crate::views::recurring_expense_form::RecurringExpenseFormView;
use crate::views::recurring_expense_list::RecurringExpenseListView;
use crate::views::recurring_invoice_detail::RecurringInvoiceDetailView;
use crate::views::recurring_invoice_form::RecurringInvoiceFormView;
use crate::views::recurring_invoice_list::RecurringInvoiceListView;
use crate::views::reports::ReportsView;
use crate::views::settings_audit::SettingsAuditView;
use crate::views::settings_backup::SettingsBackupView;
use crate::views::settings_categories::SettingsCategoriesView;
use crate::views::settings_email::SettingsEmailView;
use crate::views::settings_firma::SettingsFirmaView;
use crate::views::settings_pdf::SettingsPdfView;
use crate::views::settings_sequences::SettingsSequencesView;
use crate::views::stub::StubView;
use crate::views::tax_credits::TaxCreditsView;
use crate::views::tax_health_detail::TaxHealthDetailView;
use crate::views::tax_health_form::TaxHealthFormView;
use crate::views::tax_income_detail::TaxIncomeDetailView;
use crate::views::tax_income_form::TaxIncomeFormView;
use crate::views::tax_investments::TaxInvestmentsView;
use crate::views::tax_overview::TaxOverviewView;
use crate::views::tax_prepayments::TaxPrepaymentsView;
use crate::views::tax_social_detail::TaxSocialDetailView;
use crate::views::tax_social_form::TaxSocialFormView;
use crate::views::vat_control_detail::VatControlDetailView;
use crate::views::vat_control_form::VatControlFormView;
use crate::views::vat_overview::VatOverviewView;
use crate::views::vat_return_detail::VatReturnDetailView;
use crate::views::vat_return_form::VatReturnFormView;
use crate::views::vies_detail::ViesDetailView;
use crate::views::vies_form::ViesFormView;

/// Root view: sidebar + main content area. Routes to appropriate view based on
/// the current NavigationState.
pub struct RootView {
    services: Arc<AppServices>,
    nav: NavigationState,
    sidebar: Entity<SidebarView>,
    content: ContentView,
    focus_handle: FocusHandle,
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
    ExpenseImport(Entity<ExpenseImportView>),
    ExpenseReview(Entity<ExpenseReviewView>),
    ContactList(Entity<ContactListView>),
    ContactDetail(Entity<ContactDetailView>),
    ContactForm(Entity<ContactFormView>),
    RecurringInvoiceList(Entity<RecurringInvoiceListView>),
    RecurringInvoiceDetail(Entity<RecurringInvoiceDetailView>),
    RecurringInvoiceForm(Entity<RecurringInvoiceFormView>),
    RecurringExpenseList(Entity<RecurringExpenseListView>),
    RecurringExpenseDetail(Entity<RecurringExpenseDetailView>),
    RecurringExpenseForm(Entity<RecurringExpenseFormView>),
    VatOverview(Entity<VatOverviewView>),
    VatReturnDetail(Entity<VatReturnDetailView>),
    VatReturnForm(Entity<VatReturnFormView>),
    VatControlDetail(Entity<VatControlDetailView>),
    VatControlForm(Entity<VatControlFormView>),
    ViesDetail(Entity<ViesDetailView>),
    ViesForm(Entity<ViesFormView>),
    TaxOverview(Entity<TaxOverviewView>),
    TaxCredits(Entity<TaxCreditsView>),
    TaxPrepayments(Entity<TaxPrepaymentsView>),
    TaxInvestments(Entity<TaxInvestmentsView>),
    TaxIncomeDetail(Entity<TaxIncomeDetailView>),
    TaxIncomeForm(Entity<TaxIncomeFormView>),
    TaxSocialDetail(Entity<TaxSocialDetailView>),
    TaxSocialForm(Entity<TaxSocialFormView>),
    TaxHealthDetail(Entity<TaxHealthDetailView>),
    TaxHealthForm(Entity<TaxHealthFormView>),
    Reports(Entity<ReportsView>),
    SettingsFirma(Entity<SettingsFirmaView>),
    SettingsEmail(Entity<SettingsEmailView>),
    SettingsSequences(Entity<SettingsSequencesView>),
    SettingsCategories(Entity<SettingsCategoriesView>),
    SettingsPdf(Entity<SettingsPdfView>),
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
        let focus_handle = cx.focus_handle();

        Self {
            services,
            nav,
            sidebar,
            content,
            focus_handle,
        }
    }

    fn toggle_sidebar(&mut self, _: &ToggleSidebar, _window: &mut Window, cx: &mut Context<Self>) {
        self.sidebar.update(cx, |sidebar, _cx| {
            sidebar.toggle_collapse();
        });
        cx.notify();
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
            ContentView::ExpenseImport(e) => subscribe_nav!(e, cx),
            ContentView::ExpenseReview(e) => subscribe_nav!(e, cx),
            ContentView::ContactList(e) => subscribe_nav!(e, cx),
            ContentView::ContactDetail(e) => subscribe_nav!(e, cx),
            ContentView::ContactForm(e) => subscribe_nav!(e, cx),
            ContentView::RecurringInvoiceList(e) => subscribe_nav!(e, cx),
            ContentView::RecurringInvoiceDetail(e) => subscribe_nav!(e, cx),
            ContentView::RecurringInvoiceForm(e) => subscribe_nav!(e, cx),
            ContentView::RecurringExpenseList(e) => subscribe_nav!(e, cx),
            ContentView::RecurringExpenseDetail(e) => subscribe_nav!(e, cx),
            ContentView::RecurringExpenseForm(e) => subscribe_nav!(e, cx),
            ContentView::VatOverview(e) => subscribe_nav!(e, cx),
            ContentView::VatReturnDetail(e) => subscribe_nav!(e, cx),
            ContentView::VatReturnForm(e) => subscribe_nav!(e, cx),
            ContentView::VatControlDetail(e) => subscribe_nav!(e, cx),
            ContentView::VatControlForm(e) => subscribe_nav!(e, cx),
            ContentView::ViesDetail(e) => subscribe_nav!(e, cx),
            ContentView::ViesForm(e) => subscribe_nav!(e, cx),
            ContentView::TaxOverview(e) => subscribe_nav!(e, cx),
            ContentView::TaxCredits(e) => subscribe_nav!(e, cx),
            ContentView::TaxPrepayments(e) => subscribe_nav!(e, cx),
            ContentView::TaxInvestments(e) => subscribe_nav!(e, cx),
            ContentView::TaxIncomeDetail(e) => subscribe_nav!(e, cx),
            ContentView::TaxIncomeForm(e) => subscribe_nav!(e, cx),
            ContentView::TaxSocialDetail(e) => subscribe_nav!(e, cx),
            ContentView::TaxSocialForm(e) => subscribe_nav!(e, cx),
            ContentView::TaxHealthDetail(e) => subscribe_nav!(e, cx),
            ContentView::TaxHealthForm(e) => subscribe_nav!(e, cx),
            ContentView::Reports(e) => subscribe_nav!(e, cx),
            ContentView::SettingsFirma(e) => subscribe_nav!(e, cx),
            ContentView::SettingsEmail(e) => subscribe_nav!(e, cx),
            ContentView::SettingsSequences(e) => subscribe_nav!(e, cx),
            ContentView::SettingsCategories(e) => subscribe_nav!(e, cx),
            ContentView::SettingsPdf(e) => subscribe_nav!(e, cx),
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
                let inv_svc = services.invoices.clone();
                let con_svc = services.contacts.clone();
                ContentView::InvoiceForm(
                    cx.new(|cx| InvoiceFormView::new_create(inv_svc, con_svc, cx)),
                )
            }
            Route::InvoiceEdit(id) => {
                let inv_svc = services.invoices.clone();
                let con_svc = services.contacts.clone();
                let id = *id;
                ContentView::InvoiceForm(
                    cx.new(move |cx| InvoiceFormView::new_edit(inv_svc, con_svc, id, cx)),
                )
            }
            Route::InvoiceDetail(id) => {
                let svc = services.invoices.clone();
                let settings_svc = services.settings.clone();
                let id = *id;
                ContentView::InvoiceDetail(
                    cx.new(|cx| InvoiceDetailView::new(svc, settings_svc, id, cx)),
                )
            }
            Route::ExpenseList => {
                let svc = services.expenses.clone();
                ContentView::ExpenseList(cx.new(|cx| ExpenseListView::new(svc, cx)))
            }
            Route::ExpenseNew => {
                let exp_svc = services.expenses.clone();
                let con_svc = services.contacts.clone();
                let cat_svc = services.categories.clone();
                ContentView::ExpenseForm(
                    cx.new(|cx| ExpenseFormView::new_create(exp_svc, con_svc, cat_svc, cx)),
                )
            }
            Route::ExpenseEdit(id) => {
                let exp_svc = services.expenses.clone();
                let con_svc = services.contacts.clone();
                let cat_svc = services.categories.clone();
                let id = *id;
                ContentView::ExpenseForm(
                    cx.new(move |cx| ExpenseFormView::new_edit(exp_svc, con_svc, cat_svc, id, cx)),
                )
            }
            Route::ExpenseDetail(id) => {
                let svc = services.expenses.clone();
                let id = *id;
                ContentView::ExpenseDetail(cx.new(|cx| ExpenseDetailView::new(svc, id, cx)))
            }
            Route::ExpenseImport => {
                let import_svc = services.import.clone();
                let doc_svc = services.documents.clone();
                let ocr_svc = services.ocr_service.clone();
                let pending_ocr = services.pending_ocr_result.clone();
                ContentView::ExpenseImport(cx.new(|cx| {
                    ExpenseImportView::new(import_svc, doc_svc, ocr_svc, pending_ocr, cx)
                }))
            }
            Route::ExpenseReview(id) => {
                let svc = services.expenses.clone();
                let pending_ocr = services.pending_ocr_result.clone();
                let id = *id;
                let ocr_result = {
                    let mut lock = pending_ocr.lock().unwrap();
                    match lock.as_ref() {
                        Some((eid, _)) if *eid == id => lock.take().map(|(_, r)| r),
                        _ => None,
                    }
                };
                ContentView::ExpenseReview(
                    cx.new(move |cx| ExpenseReviewView::new(svc, id, ocr_result, cx)),
                )
            }
            Route::ContactList => {
                let svc = services.contacts.clone();
                ContentView::ContactList(cx.new(|cx| ContactListView::new(svc, cx)))
            }
            Route::ContactNew => {
                let svc = services.contacts.clone();
                ContentView::ContactForm(cx.new(|cx| ContactFormView::new_create(svc, cx)))
            }
            Route::ContactEdit(id) => {
                let svc = services.contacts.clone();
                let id = *id;
                ContentView::ContactForm(cx.new(move |cx| ContactFormView::new_edit(svc, id, cx)))
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
            Route::RecurringInvoiceNew => {
                let svc = services.recurring_invoices.clone();
                let con_svc = services.contacts.clone();
                ContentView::RecurringInvoiceForm(
                    cx.new(|cx| RecurringInvoiceFormView::new_create(svc, con_svc, cx)),
                )
            }
            Route::RecurringInvoiceDetail(id) => {
                let svc = services.recurring_invoices.clone();
                let id = *id;
                ContentView::RecurringInvoiceDetail(
                    cx.new(|cx| RecurringInvoiceDetailView::new(svc, id, cx)),
                )
            }
            Route::RecurringExpenseList => {
                let svc = services.recurring_expenses.clone();
                ContentView::RecurringExpenseList(
                    cx.new(|cx| RecurringExpenseListView::new(svc, cx)),
                )
            }
            Route::RecurringExpenseNew => {
                let svc = services.recurring_expenses.clone();
                let con_svc = services.contacts.clone();
                let cat_svc = services.categories.clone();
                ContentView::RecurringExpenseForm(
                    cx.new(|cx| RecurringExpenseFormView::new_create(svc, con_svc, cat_svc, cx)),
                )
            }
            Route::RecurringExpenseDetail(id) => {
                let svc = services.recurring_expenses.clone();
                let id = *id;
                ContentView::RecurringExpenseDetail(
                    cx.new(|cx| RecurringExpenseDetailView::new(svc, id, cx)),
                )
            }
            Route::VATOverview => {
                let vat_svc = services.vat_returns.clone();
                let control_svc = services.vat_control.clone();
                let vies_svc = services.vies.clone();
                ContentView::VatOverview(
                    cx.new(|cx| VatOverviewView::new(vat_svc, control_svc, vies_svc, cx)),
                )
            }
            Route::VATReturnNew => {
                let svc = services.vat_returns.clone();
                ContentView::VatReturnForm(cx.new(|cx| VatReturnFormView::new(svc, cx)))
            }
            Route::VATReturnDetail(id) => {
                let svc = services.vat_returns.clone();
                let id = *id;
                ContentView::VatReturnDetail(cx.new(|cx| VatReturnDetailView::new(svc, id, cx)))
            }
            Route::VATControlNew => {
                let svc = services.vat_control.clone();
                ContentView::VatControlForm(cx.new(|cx| VatControlFormView::new(svc, cx)))
            }
            Route::VATControlDetail(id) => {
                let svc = services.vat_control.clone();
                let id = *id;
                ContentView::VatControlDetail(cx.new(|cx| VatControlDetailView::new(svc, id, cx)))
            }
            Route::VIESNew => {
                let svc = services.vies.clone();
                ContentView::ViesForm(cx.new(|cx| ViesFormView::new(svc, cx)))
            }
            Route::VIESDetail(id) => {
                let svc = services.vies.clone();
                let id = *id;
                ContentView::ViesDetail(cx.new(|cx| ViesDetailView::new(svc, id, cx)))
            }
            Route::TaxOverview => {
                let income_svc = services.income_tax.clone();
                let social_svc = services.social_insurance.clone();
                let health_svc = services.health_insurance.clone();
                ContentView::TaxOverview(
                    cx.new(|cx| TaxOverviewView::new(income_svc, social_svc, health_svc, cx)),
                )
            }
            Route::TaxCredits => {
                let svc = services.tax_credits.clone();
                ContentView::TaxCredits(cx.new(|cx| TaxCreditsView::new(svc, cx)))
            }
            Route::TaxPrepayments => {
                let svc = services.tax_year_settings.clone();
                ContentView::TaxPrepayments(cx.new(|cx| TaxPrepaymentsView::new(svc, cx)))
            }
            Route::TaxInvestments => {
                let svc = services.investment_income.clone();
                let doc_svc = services.investment_documents.clone();
                let ocr_svc = services.ocr_service.clone();
                ContentView::TaxInvestments(
                    cx.new(|cx| TaxInvestmentsView::new(svc, doc_svc, ocr_svc, cx)),
                )
            }
            Route::TaxIncomeNew => {
                let svc = services.income_tax.clone();
                ContentView::TaxIncomeForm(cx.new(|cx| TaxIncomeFormView::new(svc, cx)))
            }
            Route::TaxIncomeDetail(id) => {
                let svc = services.income_tax.clone();
                let id = *id;
                ContentView::TaxIncomeDetail(cx.new(|cx| TaxIncomeDetailView::new(svc, id, cx)))
            }
            Route::TaxSocialNew => {
                let svc = services.social_insurance.clone();
                ContentView::TaxSocialForm(cx.new(|cx| TaxSocialFormView::new(svc, cx)))
            }
            Route::TaxSocialDetail(id) => {
                let svc = services.social_insurance.clone();
                let id = *id;
                ContentView::TaxSocialDetail(cx.new(|cx| TaxSocialDetailView::new(svc, id, cx)))
            }
            Route::TaxHealthNew => {
                let svc = services.health_insurance.clone();
                ContentView::TaxHealthForm(cx.new(|cx| TaxHealthFormView::new(svc, cx)))
            }
            Route::TaxHealthDetail(id) => {
                let svc = services.health_insurance.clone();
                let id = *id;
                ContentView::TaxHealthDetail(cx.new(|cx| TaxHealthDetailView::new(svc, id, cx)))
            }
            Route::Reports => {
                let svc = services.reports.clone();
                let inv_svc = services.invoices.clone();
                let exp_svc = services.expenses.clone();
                ContentView::Reports(cx.new(|cx| ReportsView::new(svc, inv_svc, exp_svc, cx)))
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
            Route::SettingsPdf => {
                let svc = services.settings.clone();
                ContentView::SettingsPdf(cx.new(|cx| SettingsPdfView::new(svc, cx)))
            }
            Route::SettingsAuditLog => {
                let svc = services.audit.clone();
                ContentView::SettingsAudit(cx.new(|cx| SettingsAuditView::new(svc, cx)))
            }
            Route::SettingsBackup => {
                ContentView::SettingsBackup(cx.new(|_cx| SettingsBackupView::new()))
            }
            Route::ImportFakturoid => {
                let svc = services.fakturoid_import.clone();
                ContentView::ImportFakturoid(cx.new(|cx| ImportFakturoidView::new(svc, cx)))
            }
            // All routes have dedicated views now; this arm is kept
            // for any future routes added to the Route enum.
            #[allow(unreachable_patterns)]
            other => {
                let label = other.label().to_string();
                ContentView::Stub(cx.new(|_cx| StubView::new(label)))
            }
        }
    }
}

impl Focusable for RootView {
    fn focus_handle(&self, _: &App) -> FocusHandle {
        self.focus_handle.clone()
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
            ContentView::ExpenseImport(v) => v.clone().into_any_element(),
            ContentView::ExpenseReview(v) => v.clone().into_any_element(),
            ContentView::ContactList(v) => v.clone().into_any_element(),
            ContentView::ContactDetail(v) => v.clone().into_any_element(),
            ContentView::ContactForm(v) => v.clone().into_any_element(),
            ContentView::RecurringInvoiceList(v) => v.clone().into_any_element(),
            ContentView::RecurringInvoiceDetail(v) => v.clone().into_any_element(),
            ContentView::RecurringInvoiceForm(v) => v.clone().into_any_element(),
            ContentView::RecurringExpenseList(v) => v.clone().into_any_element(),
            ContentView::RecurringExpenseDetail(v) => v.clone().into_any_element(),
            ContentView::RecurringExpenseForm(v) => v.clone().into_any_element(),
            ContentView::VatOverview(v) => v.clone().into_any_element(),
            ContentView::VatReturnDetail(v) => v.clone().into_any_element(),
            ContentView::VatReturnForm(v) => v.clone().into_any_element(),
            ContentView::VatControlDetail(v) => v.clone().into_any_element(),
            ContentView::VatControlForm(v) => v.clone().into_any_element(),
            ContentView::ViesDetail(v) => v.clone().into_any_element(),
            ContentView::ViesForm(v) => v.clone().into_any_element(),
            ContentView::TaxOverview(v) => v.clone().into_any_element(),
            ContentView::TaxCredits(v) => v.clone().into_any_element(),
            ContentView::TaxPrepayments(v) => v.clone().into_any_element(),
            ContentView::TaxInvestments(v) => v.clone().into_any_element(),
            ContentView::TaxIncomeDetail(v) => v.clone().into_any_element(),
            ContentView::TaxIncomeForm(v) => v.clone().into_any_element(),
            ContentView::TaxSocialDetail(v) => v.clone().into_any_element(),
            ContentView::TaxSocialForm(v) => v.clone().into_any_element(),
            ContentView::TaxHealthDetail(v) => v.clone().into_any_element(),
            ContentView::TaxHealthForm(v) => v.clone().into_any_element(),
            ContentView::Reports(v) => v.clone().into_any_element(),
            ContentView::SettingsFirma(v) => v.clone().into_any_element(),
            ContentView::SettingsEmail(v) => v.clone().into_any_element(),
            ContentView::SettingsSequences(v) => v.clone().into_any_element(),
            ContentView::SettingsCategories(v) => v.clone().into_any_element(),
            ContentView::SettingsPdf(v) => v.clone().into_any_element(),
            ContentView::SettingsAudit(v) => v.clone().into_any_element(),
            ContentView::SettingsBackup(v) => v.clone().into_any_element(),
            ContentView::ImportFakturoid(v) => v.clone().into_any_element(),
            ContentView::Stub(v) => v.clone().into_any_element(),
        };

        div()
            .key_context("RootView")
            .track_focus(&self.focus_handle)
            .on_action(_cx.listener(Self::toggle_sidebar))
            .flex()
            .size_full()
            .bg(rgb(ZfColors::BG))
            .child(self.sidebar.clone())
            .child(div().flex_1().overflow_hidden().child(content_element))
    }
}
