<script lang="ts">
	import { onMount } from 'svelte';
	import { SvelteURLSearchParams, SvelteSet } from 'svelte/reactivity';
	import { formatCZK } from '$lib/utils/money';
	import { formatDate } from '$lib/utils/date';
	import DateInput from '$lib/components/DateInput.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import Badge from '$lib/ui/Badge.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import EmptyState from '$lib/ui/EmptyState.svelte';
	import Pagination from '$lib/ui/Pagination.svelte';

	// Extended Expense type with tax_reviewed_at field not yet in shared client.ts
	interface ExpenseWithReview {
		id: number;
		vendor_id?: number;
		vendor?: { id: number; name: string };
		expense_number: string;
		category: string;
		description: string;
		issue_date: string;
		amount: number;
		currency_code: string;
		vat_amount: number;
		is_tax_deductible: boolean;
		tax_reviewed_at?: string | null;
	}

	interface ListResponse {
		data: ExpenseWithReview[];
		total: number;
		limit: number;
		offset: number;
	}

	// Filter state
	let dateFrom = $state('');
	let dateTo = $state('');
	let taxReviewedFilter = $state<'all' | 'reviewed' | 'not_reviewed'>('all');
	let page = $state(1);
	let perPage = $state(50);

	// Data state
	let expenses = $state<ExpenseWithReview[]>([]);
	let total = $state(0);
	let loading = $state(false);
	let error = $state<string | null>(null);

	// Selection state
	let selectedIds: Set<number> = new SvelteSet();
	let bulkLoading = $state(false);
	let successMessage = $state<string | null>(null);

	// Derived
	let totalPages = $derived(Math.max(1, Math.ceil(total / perPage)));
	let allSelected = $derived(expenses.length > 0 && expenses.every((e) => selectedIds.has(e.id)));
	let someSelected = $derived(selectedIds.size > 0);

	let totalAmount = $derived(expenses.reduce((sum, e) => sum + e.amount, 0));
	let totalVAT = $derived(expenses.reduce((sum, e) => sum + e.vat_amount, 0));
	let selectedAmount = $derived(
		expenses.filter((e) => selectedIds.has(e.id)).reduce((sum, e) => sum + e.amount, 0)
	);
	let selectedVAT = $derived(
		expenses.filter((e) => selectedIds.has(e.id)).reduce((sum, e) => sum + e.vat_amount, 0)
	);

	async function loadExpenses() {
		loading = true;
		error = null;
		try {
			const query = new SvelteURLSearchParams();
			query.set('limit', String(perPage));
			query.set('offset', String((page - 1) * perPage));
			if (dateFrom) query.set('date_from', dateFrom);
			if (dateTo) query.set('date_to', dateTo);
			if (taxReviewedFilter === 'reviewed') query.set('tax_reviewed', 'true');
			if (taxReviewedFilter === 'not_reviewed') query.set('tax_reviewed', 'false');

			const res = await fetch(`/api/v1/expenses?${query.toString()}`);
			if (!res.ok) {
				throw new Error(`Chyba ${res.status}: ${res.statusText}`);
			}
			const data: ListResponse = await res.json();
			expenses = data.data;
			total = data.total;
			// Clear selection when list changes
			selectedIds.clear();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst výdaje';
		} finally {
			loading = false;
		}
	}

	function toggleSelectAll() {
		if (allSelected) {
			selectedIds.clear();
		} else {
			selectedIds.clear();
			for (const e of expenses) {
				selectedIds.add(e.id);
			}
		}
	}

	function toggleSelect(id: number) {
		if (selectedIds.has(id)) {
			selectedIds.delete(id);
		} else {
			selectedIds.add(id);
		}
	}

	async function markReviewed() {
		if (selectedIds.size === 0) return;
		bulkLoading = true;
		error = null;
		successMessage = null;
		try {
			const res = await fetch('/api/v1/expenses/review', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ ids: Array.from(selectedIds) })
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({}));
				throw new Error((body as { error?: string }).error ?? `Chyba ${res.status}`);
			}
			successMessage = `${selectedIds.size} výdajů označeno jako zkontrolováno`;
			await loadExpenses();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se označit výdaje';
		} finally {
			bulkLoading = false;
		}
	}

	async function unmarkReviewed() {
		if (selectedIds.size === 0) return;
		bulkLoading = true;
		error = null;
		successMessage = null;
		try {
			const res = await fetch('/api/v1/expenses/unreview', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ ids: Array.from(selectedIds) })
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({}));
				throw new Error((body as { error?: string }).error ?? `Chyba ${res.status}`);
			}
			successMessage = `${selectedIds.size} výdajů odznačeno`;
			await loadExpenses();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se odznačit výdaje';
		} finally {
			bulkLoading = false;
		}
	}

	function applyFilters() {
		page = 1;
		loadExpenses();
	}

	onMount(() => {
		loadExpenses();
	});
</script>

<svelte:head>
	<title>Daňová kontrola nákladů - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-6xl">
	<PageHeader title="Daňová kontrola nákladů" description="Označte výdaje jako daňově zkontrolované">
		{#snippet actions()}
			<Button variant="secondary" href="/expenses">
				Zpět na náklady
			</Button>
		{/snippet}
	</PageHeader>

	<!-- Filters -->
	<Card class="mt-6">
		<div class="flex flex-wrap items-end gap-4">
			<div class="flex flex-col gap-1">
				<label for="date-from" class="text-xs font-medium text-muted">Datum od</label>
				<DateInput id="date-from" bind:value={dateFrom} />
			</div>
			<div class="flex flex-col gap-1">
				<label for="date-to" class="text-xs font-medium text-muted">Datum do</label>
				<DateInput id="date-to" bind:value={dateTo} />
			</div>
			<div class="flex flex-col gap-1">
				<label for="tax-reviewed" class="text-xs font-medium text-muted">Stav kontroly</label>
				<select
					id="tax-reviewed"
					bind:value={taxReviewedFilter}
					class="rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
				>
					<option value="all">Vše</option>
					<option value="not_reviewed">Nekontrolováno</option>
					<option value="reviewed">Zkontrolováno</option>
				</select>
			</div>
			<Button variant="primary" onclick={applyFilters}>
				Filtrovat
			</Button>
		</div>
	</Card>

	<ErrorAlert {error} class="mt-4" />

	{#if successMessage}
		<div class="mt-4 rounded-lg border border-success/20 bg-success-bg p-4 text-sm text-success">
			{successMessage}
		</div>
	{/if}

	<!-- Bulk actions -->
	{#if someSelected}
		<div
			class="mt-4 flex items-center gap-3 rounded-lg border border-accent/20 bg-accent-muted px-4 py-2.5"
		>
			<span class="text-sm text-accent-text">
				Vybráno: {selectedIds.size} výdajů
			</span>
			<Button variant="success" size="sm" onclick={markReviewed} disabled={bulkLoading}>
				{bulkLoading ? 'Ukládám...' : 'Označit jako zkontrolováno'}
			</Button>
			<Button variant="secondary" size="sm" onclick={unmarkReviewed} disabled={bulkLoading}>
				{bulkLoading ? 'Ukládám...' : 'Odznačit'}
			</Button>
		</div>
	{/if}

	<!-- Table -->
	<Card padding={false} class="mt-4 overflow-hidden">
		{#if loading}
			<LoadingSpinner class="p-12" />
		{:else if expenses.length === 0}
			<EmptyState message="Žádné výdaje neodpovídají filtrům." />
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-border bg-elevated">
					<tr>
						<th class="px-4 py-2.5 w-8">
							<input
								type="checkbox"
								checked={allSelected}
								onchange={toggleSelectAll}
								class="rounded border-border accent-accent"
								title="Vybrat vše"
							/>
						</th>
						<th class="th-default">Číslo</th>
						<th class="th-default hidden md:table-cell">Datum</th>
						<th class="th-default hidden lg:table-cell">Dodavatel</th>
						<th class="th-default">Popis</th>
						<th class="th-default hidden md:table-cell">Kategorie</th>
						<th class="th-default text-right">Částka</th>
						<th class="th-default hidden text-right md:table-cell">DPH</th>
						<th class="th-default text-center">Kontrola</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border-subtle">
					{#each expenses as expense (expense.id)}
						<tr
							class="hover:bg-hover transition-colors"
							class:bg-success-bg={expense.tax_reviewed_at != null}
						>
							<td class="px-4 py-2.5">
								<input
									type="checkbox"
									checked={selectedIds.has(expense.id)}
									onchange={() => toggleSelect(expense.id)}
									class="rounded border-border accent-accent"
								/>
							</td>
							<td class="px-4 py-2.5">
								<a
									href="/expenses/{expense.id}"
									class="font-medium text-accent-text hover:text-accent"
								>
									{expense.expense_number || '-'}
								</a>
							</td>
							<td class="hidden px-4 py-2.5 text-secondary md:table-cell">
								{formatDate(expense.issue_date)}
							</td>
							<td class="hidden px-4 py-2.5 text-secondary lg:table-cell">
								{expense.vendor?.name ?? '-'}
							</td>
							<td class="px-4 py-2.5 text-primary max-w-xs truncate">
								{expense.description}
							</td>
							<td class="hidden px-4 py-2.5 text-secondary md:table-cell">
								{expense.category || '-'}
							</td>
							<td class="px-4 py-2.5 text-right font-medium font-mono tabular-nums text-primary">
								{formatCZK(expense.amount)}
							</td>
							<td class="hidden px-4 py-2.5 text-right font-mono tabular-nums text-secondary md:table-cell">
								{formatCZK(expense.vat_amount)}
							</td>
							<td class="px-4 py-2.5 text-center">
								{#if expense.tax_reviewed_at}
									<span title={formatDate(expense.tax_reviewed_at)}>
										<Badge variant="success" class="gap-1">
											<svg
												class="h-3 w-3"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
												stroke-width="3"
											>
												<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
											</svg>
											Zkontrolováno
										</Badge>
									</span>
								{:else}
									<Badge variant="muted">
										Nekontrolováno
									</Badge>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</Card>

	<!-- Summary -->
	{#if expenses.length > 0}
		<div class="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-4">
			<Card>
				<p class="text-xs font-medium text-muted">Celková částka</p>
				<p class="mt-1 text-lg font-semibold text-primary font-mono tabular-nums">{formatCZK(totalAmount)}</p>
			</Card>
			<Card>
				<p class="text-xs font-medium text-muted">Celkové DPH</p>
				<p class="mt-1 text-lg font-semibold text-primary font-mono tabular-nums">{formatCZK(totalVAT)}</p>
			</Card>
			{#if someSelected}
				<div class="rounded-lg border border-accent/20 bg-accent-muted p-5">
					<p class="text-xs font-medium text-muted">Vybráno - částka</p>
					<p class="mt-1 text-lg font-semibold text-primary font-mono tabular-nums">{formatCZK(selectedAmount)}</p>
				</div>
				<div class="rounded-lg border border-accent/20 bg-accent-muted p-5">
					<p class="text-xs font-medium text-muted">Vybráno - DPH</p>
					<p class="mt-1 text-lg font-semibold text-primary font-mono tabular-nums">{formatCZK(selectedVAT)}</p>
				</div>
			{/if}
		</div>
	{/if}

	<!-- Pagination -->
	<Pagination {page} {totalPages} {total} label="výdajů" onPageChange={(p) => { page = p; loadExpenses(); }} />
</div>
