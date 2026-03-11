<script lang="ts">
	import { goto } from '$app/navigation';
	import { recurringExpensesApi, type RecurringExpense } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import { formatDate } from '$lib/utils/date';

	let items = $state<RecurringExpense[]>([]);
	let total = $state(0);
	let page = $state(1);
	let perPage = $state(25);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let generating = $state(false);

	async function loadItems() {
		loading = true;
		error = null;
		try {
			const res = await recurringExpensesApi.list({ limit: perPage, offset: (page - 1) * perPage });
			items = res.data;
			total = res.total;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst opakované náklady';
		} finally {
			loading = false;
		}
	}

	async function handleGenerate() {
		generating = true;
		error = null;
		try {
			const result = await recurringExpensesApi.generate();
			if (result.generated > 0) {
				alert(`Vygenerováno ${result.generated} nákladů.`);
			} else {
				alert('Žádné náklady k vygenerování.');
			}
			await loadItems();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se vygenerovat náklady';
		} finally {
			generating = false;
		}
	}

	$effect(() => {
		loadItems();
	});

	let totalPages = $derived(Math.max(1, Math.ceil(total / perPage)));

	function frequencyLabel(freq: string): string {
		switch (freq) {
			case 'weekly':
				return 'Týdně';
			case 'monthly':
				return 'Měsíčně';
			case 'quarterly':
				return 'Čtvrtletně';
			case 'yearly':
				return 'Ročně';
			default:
				return freq;
		}
	}
</script>

<svelte:head>
	<title>Opakované náklady - ZFaktury</title>
</svelte:head>

<div>
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-gray-900">Opakované náklady</h1>
			<p class="mt-1 text-sm text-gray-500">Automaticky generované pravidelné výdaje</p>
		</div>
		<div class="flex gap-2">
			<button
				onclick={handleGenerate}
				disabled={generating}
				class="inline-flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2.5 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 disabled:opacity-50 transition-colors"
			>
				{generating ? 'Generuji...' : 'Vygenerovat splatné'}
			</button>
			<a
				href="/expenses/recurring/new"
				class="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 transition-colors"
			>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
				</svg>
				Nový opakovaný náklad
			</a>
		</div>
	</div>

	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700"
		>
			{error}
		</div>
	{/if}

	<div class="mt-6 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
		{#if loading}
			<div class="flex items-center justify-center p-12">
				<div role="status">
					<div
						class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"
					></div>
					<span class="sr-only">Nacitani...</span>
				</div>
			</div>
		{:else if items.length === 0}
			<div class="p-12 text-center text-gray-400">Zatím žádné opakované náklady.</div>
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-gray-200 bg-gray-50">
					<tr>
						<th class="px-4 py-3 font-medium text-gray-600">Název</th>
						<th class="hidden px-4 py-3 font-medium text-gray-600 md:table-cell">Frekvence</th>
						<th class="hidden px-4 py-3 font-medium text-gray-600 md:table-cell">Další datum</th>
						<th class="hidden px-4 py-3 font-medium text-gray-600 sm:table-cell">Stav</th>
						<th class="px-4 py-3 text-right font-medium text-gray-600">Částka</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-100">
					{#each items as item}
						<tr
							class="hover:bg-gray-50 transition-colors cursor-pointer"
							role="link"
							tabindex="0"
							onclick={() => {
								goto(`/expenses/recurring/${item.id}`);
							}}
							onkeydown={(e) => {
								if (e.key === 'Enter') goto(`/expenses/recurring/${item.id}`);
							}}
						>
							<td class="px-4 py-3">
								<a
									href="/expenses/recurring/{item.id}"
									class="font-medium text-blue-600 hover:text-blue-800"
								>
									{item.name}
								</a>
								<p class="text-xs text-gray-500">{item.description}</p>
							</td>
							<td class="hidden px-4 py-3 text-gray-600 md:table-cell"
								>{frequencyLabel(item.frequency)}</td
							>
							<td class="hidden px-4 py-3 text-gray-600 md:table-cell"
								>{formatDate(item.next_issue_date)}</td
							>
							<td class="hidden px-4 py-3 sm:table-cell">
								{#if item.is_active}
									<span
										class="inline-flex items-center rounded-full bg-green-50 px-2 py-1 text-xs font-medium text-green-700"
										>Aktivní</span
									>
								{:else}
									<span
										class="inline-flex items-center rounded-full bg-gray-100 px-2 py-1 text-xs font-medium text-gray-600"
										>Neaktivní</span
									>
								{/if}
							</td>
							<td class="px-4 py-3 text-right font-medium text-gray-900"
								>{formatCZK(item.amount)}</td
							>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>

	{#if totalPages > 1}
		<div class="mt-4 flex items-center justify-between">
			<p class="text-sm text-gray-500">Celkem {total} opakovaných nákladů</p>
			<div class="flex gap-2">
				<button
					onclick={() => {
						page = Math.max(1, page - 1);
						loadItems();
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
						loadItems();
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
