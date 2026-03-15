# RFC-014: UX Polish — Component Extraction & Toast System

**Status:** Draft
**Date:** 2026-03-15

## Summary

Extract 4 oversized Svelte page files (650-1037 lines) into focused components, and add a global toast notification system for user feedback after actions.

## Background

Several page files have grown beyond maintainable size. They mix form state, display logic, CRUD operations, and complex conditional rendering in a single file. This makes them hard to navigate, test, and modify.

Additionally, the app has no feedback mechanism after actions like saving, deleting, or sending emails. Users perform actions and must visually verify the result — there's no confirmation toast.

## Design

### Part A: Component Extraction

#### A1. `tax/investments/+page.svelte` (1037 → ~200 lines)

Extract 6 components:

| Component | Lines | Props |
|-----------|-------|-------|
| `InvestmentDocumentsTab.svelte` | 419-516 | `documents`, `uploading`, `saving`, event callbacks |
| `InvestmentCapitalIncomeTab.svelte` | 519-720 | `entries`, `showForm`, `saving`, `summary`, event callbacks |
| `InvestmentSecurityTransactionsTab.svelte` | 723-979 | `transactions`, `showForm`, `saving`, `summary`, event callbacks |
| `InvestmentSummaryCard.svelte` | 983-1034 | `summary`, `year` |

Parent page retains: state declarations, data loading (`onMount`, `$effect`), API calls, tab switching logic. Components receive data via props and emit events for mutations.

#### A2. `invoices/[id]/+page.svelte` (902 → ~250 lines)

Extract 5 components:

| Component | Lines | Props |
|-----------|-------|-------|
| `InvoiceHeaderActions.svelte` | 272-409 | `invoice`, `editing`, event callbacks for all actions |
| `InvoiceEditForm.svelte` | 411-531 | `form`, `items`, `contacts`, `saving`, event callbacks |
| `InvoiceDetailsCard.svelte` | 536-600 | `invoice` (read-only display) |
| `InvoiceItemsDisplay.svelte` | 602-692 | `items`, `invoice` totals (read-only table) |
| `InvoicePaymentInfoCard.svelte` | 695-765 | `invoice`, `qrError` |

Already extracted and reused: `StatusTimeline`, `ReminderCard`, `AuditLogPanel`, `ConfirmDialog`, `CreditNoteDialog`, `SendEmailDialog`.

#### A3. `tax/credits/+page.svelte` (846 → ~180 lines)

Extract 5 components:

| Component | Lines | Props |
|-----------|-------|-------|
| `PersonalCreditsCard.svelte` | 404-466 | form state, `summary.personal`, `saving`, `taxConstants` |
| `SpouseCreditsCard.svelte` | 469-552 | spouse form state, `summary.spouse`, `saving`, `taxConstants` |
| `ChildrenCreditsCard.svelte` | 555-669 | children form state, `summary.children`, `saving`, `taxConstants` |
| `TaxDeductionsCard.svelte` | 673-819 | deduction form state, `deductions`, `saving`, `taxConstants` |
| `TaxCreditsSummaryCard.svelte` | 822-843 | `summary`, `deductions` (read-only) |

#### A4. `expenses/[id]/+page.svelte` (649 → ~200 lines)

Extract 3 components (smaller file, fewer extractions needed):

| Component | Lines | Props |
|-----------|-------|-------|
| `ExpenseEditForm.svelte` | 279-452 | `form`, `items`, `contacts`, `categories`, `saving`, event callbacks |
| `ExpenseDetailsDisplay.svelte` | 459-618 | `expense`, `documents` (read-only) |
| `ExpenseDocumentsCard.svelte` | 591-611 | `documents`, `expenseId`, event callbacks |

### Part B: Toast Notification System

#### Store: `frontend/src/lib/stores/toast.svelte.ts`

```typescript
export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface Toast {
  id: string;
  type: ToastType;
  message: string;
  duration: number;
}

let toasts = $state<Toast[]>([]);

export function addToast(message: string, type: ToastType = 'success', duration = 4000) {
  const id = crypto.randomUUID();
  toasts.push({ id, type, message, duration });
  setTimeout(() => removeToast(id), duration);
}

export function removeToast(id: string) {
  toasts = toasts.filter(t => t.id !== id);
}

export function getToasts() {
  return toasts;
}

// Convenience functions
export const toast = {
  success: (msg: string) => addToast(msg, 'success'),
  error: (msg: string) => addToast(msg, 'error', 6000),
  warning: (msg: string) => addToast(msg, 'warning'),
  info: (msg: string) => addToast(msg, 'info'),
};
```

#### Component: `frontend/src/lib/components/ToastContainer.svelte`

```svelte
<script lang="ts">
  import { getToasts, removeToast } from '$lib/stores/toast.svelte';

  const typeStyles: Record<string, string> = {
    success: 'bg-success-bg border-success-border text-success',
    error: 'bg-danger-bg border-danger-border text-danger',
    warning: 'bg-warning-bg border-warning-border text-warning',
    info: 'bg-accent-bg border-accent-border text-accent',
  };

  let toasts = $derived(getToasts());
</script>

<div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2" aria-live="polite">
  {#each toasts as t (t.id)}
    <div class="rounded-lg border px-4 py-3 text-sm shadow-lg {typeStyles[t.type]}">
      <div class="flex items-center justify-between gap-3">
        <span>{t.message}</span>
        <button onclick={() => removeToast(t.id)} class="text-current opacity-60 hover:opacity-100">
          &times;
        </button>
      </div>
    </div>
  {/each}
</div>
```

#### Integration in Layout

Add `<ToastContainer />` to `frontend/src/lib/components/Layout.svelte` at the bottom of the template.

#### Usage in pages

```typescript
import { toast } from '$lib/stores/toast.svelte';

// After saving invoice:
toast.success('Faktura ulozena.');

// After sending email:
toast.success('Email odeslan.');

// On error:
toast.error('Nepodarilo se ulozit fakturu.');
```

### Component File Locations

All new components go into `frontend/src/lib/components/`:

```
frontend/src/lib/components/
├── toast/
│   └── ToastContainer.svelte
├── invoice/
│   ├── InvoiceHeaderActions.svelte
│   ├── InvoiceEditForm.svelte
│   ├── InvoiceDetailsCard.svelte
│   ├── InvoiceItemsDisplay.svelte
│   └── InvoicePaymentInfoCard.svelte
├── expense/
│   ├── ExpenseEditForm.svelte
│   ├── ExpenseDetailsDisplay.svelte
│   └── ExpenseDocumentsCard.svelte
├── tax/
│   ├── PersonalCreditsCard.svelte
│   ├── SpouseCreditsCard.svelte
│   ├── ChildrenCreditsCard.svelte
│   ├── TaxDeductionsCard.svelte
│   ├── TaxCreditsSummaryCard.svelte
│   ├── InvestmentDocumentsTab.svelte
│   ├── InvestmentCapitalIncomeTab.svelte
│   ├── InvestmentSecurityTransactionsTab.svelte
│   └── InvestmentSummaryCard.svelte
```

## Implementation Order

1. Toast system (store + component + Layout integration) — independent, quick win
2. `tax/investments` extraction — largest file, most benefit
3. `invoices/[id]` extraction
4. `tax/credits` extraction
5. `expenses/[id]` extraction
6. Add `toast.*` calls to all CRUD operations across pages
7. Run `cd frontend && npm test` and `npm run check`

## Out of Scope

- Keyboard shortcuts (separate feature)
- Dark mode (separate feature, needs design tokens review)
- Request deduplication/caching (separate optimization)
- Animated toast transitions (can add later with Svelte transitions)
