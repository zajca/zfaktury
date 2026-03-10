<script lang="ts">
	import { goto } from '$app/navigation';
	import { expensesApi, type Expense } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import { formatDate } from '$lib/utils/date';

	let expenses = $state<Expense[]>([]);
	let total = $state(0);
	let page = $state(1);
	let perPage = $state(25);
	let search = $state('');
	let loading = $state(true);
	let error = $state<string | null>(null);

	let searchTimeout: ReturnType<typeof setTimeout>;

	async function loadExpenses() {
		loading = true;
		error = null;
		try {
			const res = await expensesApi.list({ limit: perPage, offset: (page - 1) * perPage, search: search || undefined });
			expenses = res.data;
			total = res.total;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst náklady';
		} finally {
			loading = false;
		}
	}

	function handleSearch() {
		clearTimeout(searchTimeout);
		searchTimeout = setTimeout(() => {
			page = 1;
			loadExpenses();
		}, 300);
	}

	$effect(() => {
		loadExpenses();
	});

	$effect(() => {
		search;
		handleSearch();
	});

	let totalPages = $derived(Math.max(1, Math.ceil(total / perPage)));
</script>

<svelte:head>
	<title>Náklady - ZFaktury</title>
</svelte:head>

<div>
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-gray-900">Náklady</h1>
			<p class="mt-1 text-sm text-gray-500">Evidence výdajů a nákladů</p>
		</div>
		<a
			href="/expenses/new"
			class="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 transition-colors"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
			</svg>
			Přidat náklad
		</a>
	</div>

	<!-- Search -->
	<div class="mt-6">
		<input
			type="text"
			bind:value={search}
			placeholder="Hledat podle popisu, dodavatele..."
			class="w-full max-w-md rounded-lg border border-gray-300 px-4 py-2.5 text-sm shadow-sm placeholder:text-gray-400 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
		/>
	</div>

	{#if error}
		<div role="alert" class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
			{error}
		</div>
	{/if}

	<div class="mt-4 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
		{#if loading}
			<div class="flex items-center justify-center p-12">
				<div role="status"><div class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"></div><span class="sr-only">Nacitani...</span></div>
			</div>
		{:else if expenses.length === 0}
			<div class="p-12 text-center text-gray-400">
				{search ? 'Žádné náklady neodpovídají hledání.' : 'Zatím žádné náklady.'}
			</div>
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-gray-200 bg-gray-50">
					<tr>
						<th class="px-4 py-3 font-medium text-gray-600">Popis</th>
						<th class="hidden px-4 py-3 font-medium text-gray-600 md:table-cell">Kategorie</th>
						<th class="hidden px-4 py-3 font-medium text-gray-600 md:table-cell">Datum</th>
						<th class="px-4 py-3 text-right font-medium text-gray-600">Částka</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-100">
					{#each expenses as expense}
						<tr
							class="hover:bg-gray-50 transition-colors cursor-pointer"
							role="link"
							tabindex="0"
							onclick={() => { goto(`/expenses/${expense.id}`); }}
							onkeydown={(e) => { if (e.key === 'Enter') goto(`/expenses/${expense.id}`); }}
						>
							<td class="px-4 py-3">
								<a href="/expenses/{expense.id}" class="font-medium text-blue-600 hover:text-blue-800">
									{expense.description}
								</a>
							</td>
							<td class="hidden px-4 py-3 text-gray-600 md:table-cell">{expense.category || '-'}</td>
							<td class="hidden px-4 py-3 text-gray-600 md:table-cell">{formatDate(expense.issue_date)}</td>
							<td class="px-4 py-3 text-right font-medium text-gray-900">{formatCZK(expense.amount)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>

	{#if totalPages > 1}
		<div class="mt-4 flex items-center justify-between">
			<p class="text-sm text-gray-500">Celkem {total} nákladů</p>
			<div class="flex gap-2">
				<button
					onclick={() => { page = Math.max(1, page - 1); loadExpenses(); }}
					disabled={page <= 1}
					class="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					Předchozí
				</button>
				<span class="flex items-center px-3 text-sm text-gray-600">{page} / {totalPages}</span>
				<button
					onclick={() => { page = Math.min(totalPages, page + 1); loadExpenses(); }}
					disabled={page >= totalPages}
					class="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					Další
				</button>
			</div>
		</div>
	{/if}
</div>
