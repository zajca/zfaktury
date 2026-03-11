<script lang="ts">
	import { onMount } from 'svelte';
	import { SvelteURLSearchParams, SvelteSet } from 'svelte/reactivity';
	import { formatCZK } from '$lib/utils/money';
	import { formatDate } from '$lib/utils/date';
	import DateInput from '$lib/components/DateInput.svelte';

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

<div>
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-gray-900">Daňová kontrola nákladů</h1>
			<p class="mt-1 text-sm text-gray-500">Označte výdaje jako daňově zkontrolované</p>
		</div>
		<a
			href="/expenses"
			class="inline-flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
		>
			Zpět na náklady
		</a>
	</div>

	<!-- Filters -->
	<div
		class="mt-6 flex flex-wrap items-end gap-4 rounded-xl border border-gray-200 bg-white p-4 shadow-sm"
	>
		<div class="flex flex-col gap-1">
			<label for="date-from" class="text-xs font-medium text-gray-600">Datum od</label>
			<DateInput id="date-from" bind:value={dateFrom} />
		</div>
		<div class="flex flex-col gap-1">
			<label for="date-to" class="text-xs font-medium text-gray-600">Datum do</label>
			<DateInput id="date-to" bind:value={dateTo} />
		</div>
		<div class="flex flex-col gap-1">
			<label for="tax-reviewed" class="text-xs font-medium text-gray-600">Stav kontroly</label>
			<select
				id="tax-reviewed"
				bind:value={taxReviewedFilter}
				class="rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
			>
				<option value="all">Vše</option>
				<option value="not_reviewed">Nekontrolováno</option>
				<option value="reviewed">Zkontrolováno</option>
			</select>
		</div>
		<button
			onclick={applyFilters}
			class="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 transition-colors"
		>
			Filtrovat
		</button>
	</div>

	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700"
		>
			{error}
		</div>
	{/if}

	{#if successMessage}
		<div class="mt-4 rounded-lg border border-green-200 bg-green-50 p-4 text-sm text-green-700">
			{successMessage}
		</div>
	{/if}

	<!-- Bulk actions -->
	{#if someSelected}
		<div
			class="mt-4 flex items-center gap-3 rounded-lg border border-blue-200 bg-blue-50 px-4 py-3"
		>
			<span class="text-sm font-medium text-blue-800">
				Vybráno: {selectedIds.size} výdajů
			</span>
			<button
				onclick={markReviewed}
				disabled={bulkLoading}
				class="rounded-lg bg-green-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
			>
				{bulkLoading ? 'Ukládám...' : 'Označit jako zkontrolováno'}
			</button>
			<button
				onclick={unmarkReviewed}
				disabled={bulkLoading}
				class="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
			>
				{bulkLoading ? 'Ukládám...' : 'Odznačit'}
			</button>
		</div>
	{/if}

	<!-- Table -->
	<div class="mt-4 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
		{#if loading}
			<div class="flex items-center justify-center p-12">
				<div role="status">
					<div
						class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"
					></div>
					<span class="sr-only">Nacitani...</span>
				</div>
			</div>
		{:else if expenses.length === 0}
			<div class="p-12 text-center text-gray-400">Žádné výdaje neodpovídají filtrům.</div>
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-gray-200 bg-gray-50">
					<tr>
						<th class="px-4 py-3 w-8">
							<input
								type="checkbox"
								checked={allSelected}
								onchange={toggleSelectAll}
								class="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
								title="Vybrat vše"
							/>
						</th>
						<th class="px-4 py-3 font-medium text-gray-600">Číslo</th>
						<th class="hidden px-4 py-3 font-medium text-gray-600 md:table-cell">Datum</th>
						<th class="hidden px-4 py-3 font-medium text-gray-600 lg:table-cell">Dodavatel</th>
						<th class="px-4 py-3 font-medium text-gray-600">Popis</th>
						<th class="hidden px-4 py-3 font-medium text-gray-600 md:table-cell">Kategorie</th>
						<th class="px-4 py-3 text-right font-medium text-gray-600">Částka</th>
						<th class="hidden px-4 py-3 text-right font-medium text-gray-600 md:table-cell">DPH</th>
						<th class="px-4 py-3 text-center font-medium text-gray-600">Kontrola</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-100">
					{#each expenses as expense (expense.id)}
						<tr
							class="hover:bg-gray-50 transition-colors"
							class:bg-green-50={expense.tax_reviewed_at != null}
						>
							<td class="px-4 py-3">
								<input
									type="checkbox"
									checked={selectedIds.has(expense.id)}
									onchange={() => toggleSelect(expense.id)}
									class="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
								/>
							</td>
							<td class="px-4 py-3">
								<a
									href="/expenses/{expense.id}"
									class="font-medium text-blue-600 hover:text-blue-800"
								>
									{expense.expense_number || '-'}
								</a>
							</td>
							<td class="hidden px-4 py-3 text-gray-600 md:table-cell">
								{formatDate(expense.issue_date)}
							</td>
							<td class="hidden px-4 py-3 text-gray-600 lg:table-cell">
								{expense.vendor?.name ?? '-'}
							</td>
							<td class="px-4 py-3 text-gray-900 max-w-xs truncate">
								{expense.description}
							</td>
							<td class="hidden px-4 py-3 text-gray-600 md:table-cell">
								{expense.category || '-'}
							</td>
							<td class="px-4 py-3 text-right font-medium text-gray-900">
								{formatCZK(expense.amount)}
							</td>
							<td class="hidden px-4 py-3 text-right text-gray-600 md:table-cell">
								{formatCZK(expense.vat_amount)}
							</td>
							<td class="px-4 py-3 text-center">
								{#if expense.tax_reviewed_at}
									<span
										class="inline-flex items-center gap-1 rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700"
										title={formatDate(expense.tax_reviewed_at)}
									>
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
									</span>
								{:else}
									<span
										class="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-500"
									>
										Nekontrolováno
									</span>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>

	<!-- Summary -->
	{#if expenses.length > 0}
		<div class="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-4">
			<div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
				<p class="text-xs font-medium text-gray-500">Celková částka</p>
				<p class="mt-1 text-lg font-semibold text-gray-900">{formatCZK(totalAmount)}</p>
			</div>
			<div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
				<p class="text-xs font-medium text-gray-500">Celkové DPH</p>
				<p class="mt-1 text-lg font-semibold text-gray-900">{formatCZK(totalVAT)}</p>
			</div>
			{#if someSelected}
				<div class="rounded-lg border border-blue-200 bg-blue-50 p-4 shadow-sm">
					<p class="text-xs font-medium text-blue-600">Vybráno - částka</p>
					<p class="mt-1 text-lg font-semibold text-blue-900">{formatCZK(selectedAmount)}</p>
				</div>
				<div class="rounded-lg border border-blue-200 bg-blue-50 p-4 shadow-sm">
					<p class="text-xs font-medium text-blue-600">Vybráno - DPH</p>
					<p class="mt-1 text-lg font-semibold text-blue-900">{formatCZK(selectedVAT)}</p>
				</div>
			{/if}
		</div>
	{/if}

	<!-- Pagination -->
	{#if totalPages > 1}
		<div class="mt-4 flex items-center justify-between">
			<p class="text-sm text-gray-500">Celkem {total} výdajů</p>
			<div class="flex gap-2">
				<button
					onclick={() => {
						page = Math.max(1, page - 1);
						loadExpenses();
					}}
					disabled={page <= 1}
					class="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					Předchozí
				</button>
				<span class="flex items-center px-3 text-sm text-gray-600">{page} / {totalPages}</span>
				<button
					onclick={() => {
						page = Math.min(totalPages, page + 1);
						loadExpenses();
					}}
					disabled={page >= totalPages}
					class="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					Další
				</button>
			</div>
		</div>
	{/if}
</div>
