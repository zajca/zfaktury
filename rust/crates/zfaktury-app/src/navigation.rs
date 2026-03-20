/// Application routes matching the frontend SvelteKit routes.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Route {
    // Top-level
    Dashboard,
    Reports,

    // Invoices
    InvoiceList,
    InvoiceNew,
    InvoiceDetail(i64),

    // Recurring invoices (templates)
    RecurringInvoiceList,
    RecurringInvoiceNew,
    RecurringInvoiceDetail(i64),

    // Expenses
    ExpenseList,
    ExpenseNew,
    ExpenseDetail(i64),
    ExpenseImport,
    ExpenseReview,

    // Recurring expenses
    RecurringExpenseList,
    RecurringExpenseNew,
    RecurringExpenseDetail(i64),

    // Contacts
    ContactList,
    ContactDetail(i64),

    // VAT
    VATOverview,
    VATReturnNew,
    VATReturnDetail(i64),
    VATControlNew,
    VATControlDetail(i64),
    VIESNew,
    VIESDetail(i64),

    // Tax
    TaxOverview,
    TaxCredits,
    TaxPrepayments,
    TaxInvestments,
    TaxIncomeNew,
    TaxIncomeDetail(i64),
    TaxSocialNew,
    TaxSocialDetail(i64),
    TaxHealthNew,
    TaxHealthDetail(i64),

    // Settings
    SettingsFirma,
    SettingsEmail,
    SettingsSequences,
    SettingsCategories,
    SettingsPdf,
    SettingsAuditLog,
    SettingsBackup,

    // Import
    ImportFakturoid,
}

impl Route {
    /// Parse a URL path string into a Route.
    pub fn from_path(path: &str) -> Option<Self> {
        let path = path.trim_end_matches('/');
        match path {
            "/" | "" => Some(Route::Dashboard),
            "/reports" => Some(Route::Reports),
            "/invoices" => Some(Route::InvoiceList),
            "/invoices/new" => Some(Route::InvoiceNew),
            "/recurring" => Some(Route::RecurringInvoiceList),
            "/recurring/new" => Some(Route::RecurringInvoiceNew),
            "/expenses" => Some(Route::ExpenseList),
            "/expenses/new" => Some(Route::ExpenseNew),
            "/expenses/import" => Some(Route::ExpenseImport),
            "/expenses/review" => Some(Route::ExpenseReview),
            "/expenses/recurring" => Some(Route::RecurringExpenseList),
            "/expenses/recurring/new" => Some(Route::RecurringExpenseNew),
            "/contacts" => Some(Route::ContactList),
            "/vat" => Some(Route::VATOverview),
            "/vat/returns/new" => Some(Route::VATReturnNew),
            "/vat/control/new" => Some(Route::VATControlNew),
            "/vat/vies/new" => Some(Route::VIESNew),
            "/tax" => Some(Route::TaxOverview),
            "/tax/credits" => Some(Route::TaxCredits),
            "/tax/prepayments" => Some(Route::TaxPrepayments),
            "/tax/investments" => Some(Route::TaxInvestments),
            "/tax/income/new" => Some(Route::TaxIncomeNew),
            "/tax/social/new" => Some(Route::TaxSocialNew),
            "/tax/health/new" => Some(Route::TaxHealthNew),
            "/settings/firma" => Some(Route::SettingsFirma),
            "/settings/email" => Some(Route::SettingsEmail),
            "/settings/sequences" => Some(Route::SettingsSequences),
            "/settings/categories" => Some(Route::SettingsCategories),
            "/settings/pdf" => Some(Route::SettingsPdf),
            "/settings/audit-log" => Some(Route::SettingsAuditLog),
            "/settings/backup" => Some(Route::SettingsBackup),
            "/import/fakturoid" => Some(Route::ImportFakturoid),
            _ => parse_dynamic_route(path),
        }
    }

    /// Czech display label for the route.
    pub fn label(&self) -> &'static str {
        match self {
            Route::Dashboard => "Dashboard",
            Route::Reports => "Prehled",
            Route::InvoiceList => "Faktury",
            Route::InvoiceNew => "Nova faktura",
            Route::InvoiceDetail(_) => "Detail faktury",
            Route::RecurringInvoiceList => "Sablony faktur",
            Route::RecurringInvoiceNew => "Nova sablona",
            Route::RecurringInvoiceDetail(_) => "Detail sablony",
            Route::ExpenseList => "Naklady",
            Route::ExpenseNew => "Novy naklad",
            Route::ExpenseDetail(_) => "Detail nakladu",
            Route::ExpenseImport => "Import dokladu",
            Route::ExpenseReview => "Kontrola nakladu",
            Route::RecurringExpenseList => "Opakovane naklady",
            Route::RecurringExpenseNew => "Novy opakovany naklad",
            Route::RecurringExpenseDetail(_) => "Detail opak. nakladu",
            Route::ContactList => "Kontakty",
            Route::ContactDetail(_) => "Detail kontaktu",
            Route::VATOverview => "DPH",
            Route::VATReturnNew => "Nove DPH prizani",
            Route::VATReturnDetail(_) => "Detail DPH priznani",
            Route::VATControlNew => "Nove kontrolni hlaseni",
            Route::VATControlDetail(_) => "Detail kontrolniho hlaseni",
            Route::VIESNew => "Novy souhrnne hlaseni",
            Route::VIESDetail(_) => "Detail souhrnneho hlaseni",
            Route::TaxOverview => "Dan z prijmu",
            Route::TaxCredits => "Slevy a odpocty",
            Route::TaxPrepayments => "Zalohy",
            Route::TaxInvestments => "Investice",
            Route::TaxIncomeNew => "Nove danove priznani",
            Route::TaxIncomeDetail(_) => "Detail danoveho priznani",
            Route::TaxSocialNew => "Novy prehled OSSZ",
            Route::TaxSocialDetail(_) => "Detail prehledu OSSZ",
            Route::TaxHealthNew => "Novy prehled ZP",
            Route::TaxHealthDetail(_) => "Detail prehledu ZP",
            Route::SettingsFirma => "Firma",
            Route::SettingsEmail => "Email",
            Route::SettingsSequences => "Ciselne rady",
            Route::SettingsCategories => "Kategorie",
            Route::SettingsPdf => "PDF sablona",
            Route::SettingsAuditLog => "Audit log",
            Route::SettingsBackup => "Zalohy dat",
            Route::ImportFakturoid => "Import z Fakturoid",
        }
    }
}

fn parse_dynamic_route(path: &str) -> Option<Route> {
    let parts: Vec<&str> = path.split('/').filter(|s| !s.is_empty()).collect();
    match parts.as_slice() {
        ["invoices", id] => id.parse().ok().map(Route::InvoiceDetail),
        ["recurring", id] => id.parse().ok().map(Route::RecurringInvoiceDetail),
        ["expenses", "recurring", id] => id.parse().ok().map(Route::RecurringExpenseDetail),
        ["expenses", id] => id.parse().ok().map(Route::ExpenseDetail),
        ["contacts", id] => id.parse().ok().map(Route::ContactDetail),
        ["vat", "returns", id] => id.parse().ok().map(Route::VATReturnDetail),
        ["vat", "control", id] => id.parse().ok().map(Route::VATControlDetail),
        ["vat", "vies", id] => id.parse().ok().map(Route::VIESDetail),
        ["tax", "income", id] => id.parse().ok().map(Route::TaxIncomeDetail),
        ["tax", "social", id] => id.parse().ok().map(Route::TaxSocialDetail),
        ["tax", "health", id] => id.parse().ok().map(Route::TaxHealthDetail),
        _ => None,
    }
}

/// Mutable navigation state with history stack.
#[allow(dead_code)]
pub struct NavigationState {
    pub current: Route,
    history: Vec<Route>,
}

impl NavigationState {
    pub fn new(initial: Route) -> Self {
        Self {
            current: initial,
            history: Vec::new(),
        }
    }

    pub fn navigate(&mut self, route: Route) {
        let prev = std::mem::replace(&mut self.current, route);
        self.history.push(prev);
    }

    #[allow(dead_code)]
    pub fn go_back(&mut self) -> bool {
        if let Some(prev) = self.history.pop() {
            self.current = prev;
            true
        } else {
            false
        }
    }

    #[allow(dead_code)]
    pub fn can_go_back(&self) -> bool {
        !self.history.is_empty()
    }
}
