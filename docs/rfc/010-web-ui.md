# RFC-010: Web UI — Dashboard & Reports

**Status:** Draft
**Date:** 2026-03-12

## Summary

Real-time dashboard replacing hardcoded zeros, financial reports with charts, CSV export for invoices and expenses, and a Czech tax calendar with deadline tracking. All data is read-only aggregation over existing tables — no new migrations needed.

## Background

The current dashboard (`/`) shows hardcoded zeros for revenue, expenses, unpaid invoices, and overdue count. There is no reporting functionality. Czech OSVC need:

1. Quick financial overview to understand their current position
2. Monthly/yearly revenue and expense reports for tax planning
3. CSV exports compatible with Czech accounting software (semicolon delimiter, UTF-8 BOM)
4. Tax deadline awareness — Czech tax calendar with public holidays shifting deadlines

The frontend currently has no chart library. Chart.js 4 will be added with a thin Svelte 5 wrapper component.

## Design

### Dashboard API

A single endpoint returns all dashboard data in one request, avoiding N+1 round-trips from the frontend.

```go
type DashboardData struct {
    // Current month revenue (sum of issued invoice totals by DUZP)
    RevenueCurrentMonth Amount
    // Current month expenses
    ExpensesCurrentMonth Amount
    // Count of unpaid invoices (status: issued, sent, overdue)
    UnpaidInvoicesCount int
    // Total amount of unpaid invoices
    UnpaidInvoicesAmount Amount
    // Count of overdue invoices (due_date < today, not paid)
    OverdueInvoicesCount int
    // Total amount of overdue invoices
    OverdueInvoicesAmount Amount
    // Monthly revenue for the current year (12 entries, index 0 = January)
    MonthlyRevenue [12]Amount
    // Monthly expenses for the current year (12 entries)
    MonthlyExpenses [12]Amount
    // Recent invoices (last 5, sorted by created_at desc)
    RecentInvoices []RecentInvoice
    // Recent expenses (last 5, sorted by created_at desc)
    RecentExpenses []RecentExpense
}

type RecentInvoice struct {
    ID           int64
    Number       string
    CustomerName string
    TotalAmount  Amount
    Status       string
    DueDate      time.Time
}

type RecentExpense struct {
    ID           int64
    Description  string
    TotalAmount  Amount
    Date         time.Time
    CategoryName string
}
```

Revenue is grouped by `delivery_date` (DUZP), not `issue_date` — this is the tax-correct grouping for Czech OSVC. An invoice issued in December with DUZP in January counts as January revenue.

### Reports API

All report endpoints accept a `year` query parameter (default: current year). Reports return pre-aggregated data suitable for direct rendering — the frontend does no computation.

```go
type RevenueReport struct {
    Year           int
    MonthlyTotals  [12]Amount  // Revenue by DUZP month
    QuarterlyTotals [4]Amount
    YearTotal      Amount
}

type ExpenseReport struct {
    Year           int
    MonthlyTotals  [12]Amount
    QuarterlyTotals [4]Amount
    YearTotal      Amount
    ByCategory     []CategoryBreakdown
}

type CategoryBreakdown struct {
    CategoryID   int64
    CategoryName string
    Total        Amount
    Percentage   float64  // pre-computed, 0-100
}

type TopCustomersReport struct {
    Year      int
    Customers []CustomerRevenue
}

type CustomerRevenue struct {
    ContactID    int64
    Name         string
    InvoiceCount int
    TotalRevenue Amount
}

type ProfitLossReport struct {
    Year             int
    MonthlyRevenue   [12]Amount
    MonthlyExpenses  [12]Amount
    MonthlyProfit    [12]Amount  // revenue - expenses
    TotalRevenue     Amount
    TotalExpenses    Amount
    TotalProfit      Amount
}

type TaxCalendar struct {
    Year      int
    Deadlines []TaxDeadline
}

type TaxDeadline struct {
    Name        string     // e.g., "Přiznání k DPH za leden"
    Description string     // what needs to be done
    OriginalDate time.Time // statutory deadline
    AdjustedDate time.Time // shifted if falls on weekend/holiday
    Type        string     // "vat", "income_tax", "social", "health", "advance"
    Recurring   string     // "monthly", "quarterly", "yearly"
}
```

### Tax Calendar Logic

Czech public holidays (Law No. 245/2000 Sb.) are hardcoded since they change rarely. If a deadline falls on a Saturday, Sunday, or public holiday, it shifts to the next business day.

Key deadlines included:
- **VAT returns** — 25th of following month (monthly/quarterly per VAT period)
- **VAT control statement** — 25th of following month (monthly)
- **Income tax return** — April 1 (or July 1 with advisor)
- **Social insurance overview** — follows income tax deadline + 1 month
- **Health insurance overview** — follows income tax deadline + 8 days
- **Income tax advances** — quarterly (March 15, June 15, September 15, December 15)

### CSV Export

CSV files use:
- **Semicolon delimiter** (Czech convention — Excel in Czech locale auto-detects semicolons)
- **UTF-8 with BOM** (0xEF, 0xBB, 0xBF prefix — ensures Czech diacritics display correctly in Excel)
- **Czech date format** in data: `DD.MM.YYYY`
- **Czech decimal separator** for amounts: comma (e.g., `1 234,50`)
- Filename includes entity type and date range: `faktury_2026.csv`, `naklady_2026.csv`

Invoice CSV columns:
```
Číslo;Odběratel;IČO;Datum vystavení;DÚZP;Datum splatnosti;Základ DPH;DPH;Celkem;Měna;Stav;Zaplaceno dne
```

Expense CSV columns:
```
Popis;Dodavatel;IČO;Datum;Kategorie;Základ DPH;DPH;Celkem;Měna;Daňově uznatelný
```

### Database Queries

No new tables or migrations. All data comes from aggregation queries on existing tables:

- **Revenue:** `SELECT strftime('%m', delivery_date) as month, SUM(total_amount) FROM invoices WHERE delivery_date BETWEEN ? AND ? AND type = 'standard' AND deleted_at IS NULL GROUP BY month`
- **Expenses:** `SELECT strftime('%m', date) as month, SUM(total_amount) FROM expenses WHERE date BETWEEN ? AND ? AND deleted_at IS NULL GROUP BY month`
- **Unpaid invoices:** `SELECT COUNT(*), SUM(total_amount) FROM invoices WHERE status IN ('issued', 'sent', 'overdue') AND deleted_at IS NULL`
- **Top customers:** `SELECT contact_id, SUM(total_amount) FROM invoices WHERE ... GROUP BY contact_id ORDER BY SUM(total_amount) DESC LIMIT 10`
- **Category breakdown:** `SELECT category_id, SUM(total_amount) FROM expenses WHERE ... GROUP BY category_id`

Credit notes are excluded from revenue aggregation (type = 'credit_note'). Proforma invoices are also excluded (type = 'proforma').

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/dashboard` | Dashboard summary (current month stats, monthly chart data, recent items) |
| GET | `/api/v1/reports/revenue?year=2026` | Monthly/quarterly/yearly revenue breakdown |
| GET | `/api/v1/reports/expenses?year=2026` | Monthly/quarterly/yearly expense breakdown with category split |
| GET | `/api/v1/reports/top-customers?year=2026` | Top 10 customers by revenue |
| GET | `/api/v1/reports/profit-loss?year=2026` | Revenue vs. expenses P&L |
| GET | `/api/v1/reports/tax-calendar?year=2026` | Tax deadlines with holiday adjustments |
| GET | `/api/v1/export/invoices?year=2026` | CSV export of invoices (Content-Type: text/csv) |
| GET | `/api/v1/export/expenses?year=2026` | CSV export of expenses (Content-Type: text/csv) |

All report/export endpoints default to the current year if `year` is omitted.

CSV export endpoints return `Content-Type: text/csv; charset=utf-8` with `Content-Disposition: attachment; filename="faktury_2026.csv"`.

## Frontend

### Chart.js Integration

Install `chart.js` (v4) as a frontend dependency. Create a thin Svelte 5 wrapper:

```
frontend/src/lib/components/
├── BarChart.svelte      # Wraps Chart.js bar chart
├── DoughnutChart.svelte # Wraps Chart.js doughnut chart
└── LineChart.svelte     # Wraps Chart.js line chart (P&L overlay)
```

Each wrapper:
- Accepts `data` and `options` props (typed)
- Creates/destroys Chart instance in `onMount`/`onDestroy`
- Reacts to data changes via `$effect` to call `chart.update()`
- Uses Czech locale for labels (month names: Leden, Únor, ...)
- Consistent color palette matching the app theme

### Dashboard Page (`/`)

Replace current hardcoded content with:

1. **Stats row** (4 cards): Revenue this month, Expenses this month, Unpaid invoices (count + amount), Overdue invoices (count + amount)
2. **Revenue vs. Expenses chart** (bar chart, 12 months, stacked or grouped)
3. **Recent invoices** table (last 5, with status badge and link)
4. **Recent expenses** table (last 5, with category and link)

Data loaded via single `GET /api/v1/dashboard` call in `onMount`.

### Reports Page (`/reports`)

New page with year selector and tabbed/sectioned layout:

1. **Revenue tab** — Bar chart (monthly) + summary cards (Q1-Q4, year total)
2. **Expenses tab** — Bar chart (monthly) + doughnut chart (by category)
3. **Profit & Loss tab** — Line/bar chart overlaying revenue and expenses, monthly profit row
4. **Top Customers tab** — Horizontal bar chart + table (name, invoice count, total)
5. **Tax Calendar tab** — Timeline/list of upcoming deadlines, color-coded by type, past deadlines grayed out
6. **CSV Export section** — Two buttons: "Export faktur" and "Export nakladu", year selector applies

### Navigation Update

Add "Přehledy" (Reports) link to the sidebar navigation in `Layout.svelte`, between existing items. Icon: chart/bar-chart icon from the existing icon set.

## Implementation Order

1. **Dashboard service + handler** — `DashboardService` with aggregation queries, `DashboardHandler` with single GET endpoint
2. **Report service + handler** — `ReportService` with revenue/expenses/P&L/top-customers queries, `ReportHandler`
3. **Tax calendar service** — Czech holidays, deadline calculation, business day shifting
4. **CSV export handler** — Reuses report service queries, formats as CSV with BOM
5. **Frontend: Chart.js wrappers** — `BarChart`, `DoughnutChart`, `LineChart` components
6. **Frontend: Dashboard rewrite** — Replace hardcoded page with live data + chart
7. **Frontend: Reports page** — New route with tabs, charts, CSV buttons
8. **Frontend: Navigation update** — Add "Přehledy" to sidebar
9. **Wire into router.go, serve.go, client.ts**
10. **Tests** — Service layer unit tests (aggregation logic, tax calendar, CSV formatting)

## Out of Scope

- **PDF export of reports** — CSV covers the immediate need; PDF reports can be added later
- **VAT liability overview** — Requires deeper VAT calculation logic, better suited for a separate RFC
- **Year-over-year comparison** — Listed in idea.md but deferred; single-year reports come first
- **Cash flow chart** — Requires payment date tracking which is incomplete without banking integration (RFC-009)
- **Custom date ranges** — Reports are year-based only; custom ranges add complexity with minimal benefit for annual tax filing workflow
- **Expense breakdown by tax-deductible vs. non-deductible** — Can be derived from existing category data but deferred
- **Real-time updates / WebSocket** — Dashboard refreshes on page load, no live push

## Open Questions

1. **Chart color palette** — Should charts use the existing Tailwind theme colors, or a dedicated data-visualization palette with better contrast for colorblind users?
2. **Dashboard cache** — Aggregation queries on large datasets could be slow. Should we add in-memory caching with a short TTL (e.g., 60s), or is SQLite fast enough for typical OSVC volumes (hundreds to low thousands of invoices)?
3. **Tax calendar configurability** — Should the user be able to configure their VAT period (monthly vs. quarterly) in settings, or should we detect it from existing VAT returns?
