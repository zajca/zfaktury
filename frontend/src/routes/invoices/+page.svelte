<script lang="ts">
	import { invoicesApi, type Invoice } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import { formatDate } from '$lib/utils/date';

	let invoices = $state<Invoice[]>([]);
	let total = $state(0);
	let page = $state(1);
	let perPage = $state(25);
	let search = $state('');
	let statusFilter = $state('');
	let loading = $state(true);
	let error = $state<string | null>(null);

	let searchTimeout: ReturnType<typeof setTimeout>;

	const statusLabels: Record<string, string> = {
		draft: 'Koncept',
		sent: 'Odeslaná',
		paid: 'Uhrazená',
		overdue: 'Po splatnosti',
		cancelled: 'Stornovaná'
	};

	const statusColors: Record<string, string> = {
		draft: 'bg-gray-100 text-gray-700',
		sent: 'bg-blue-100 text-blue-700',
		paid: 'bg-green-100 text-green-700',
		overdue: 'bg-red-100 text-red-700',
		cancelled: 'bg-gray-100 text-gray-500'
	};

	async function loadInvoices() {
		loading = true;
		error = null;
		try {
			const res = await invoicesApi.list({
				limit: perPage,
				offset: (page - 1) * perPage,
				search: search || undefined,
				status: statusFilter || undefined
			});
			invoices = res.data;
			total = res.total;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst faktury';
		} finally {
			loading = false;
		}
	}

	function handleSearch() {
		clearTimeout(searchTimeout);
		searchTimeout = setTimeout(() => {
			page = 1;
			loadInvoices();
		}, 300);
	}

	$effect(() => {
		loadInvoices();
	});

	$effect(() => {
		// Re-trigger on search change
		search;
		handleSearch();
	});

	$effect(() => {
		// Re-trigger on status filter change
		statusFilter;
		page = 1;
		loadInvoices();
	});

	let totalPages = $derived(Math.max(1, Math.ceil(total / perPage)));
</script>

<svelte:head>
	<title>Faktury - ZFaktury</title>
</svelte:head>

<div>
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-gray-900">Faktury</h1>
			<p class="mt-1 text-sm text-gray-500">Přehled vydaných faktur</p>
		</div>
		<a
			href="/invoices/new"
			class="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 transition-colors"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
			</svg>
			Nová faktura
		</a>
	</div>

	<!-- Filters -->
	<div class="mt-6 flex flex-wrap items-center gap-4">
		<input
			type="text"
			bind:value={search}
			placeholder="Hledat podle čísla, zákazníka..."
			class="w-full max-w-md rounded-lg border border-gray-300 px-4 py-2.5 text-sm shadow-sm placeholder:text-gray-400 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
		/>
		<select
			bind:value={statusFilter}
			class="rounded-lg border border-gray-300 px-3 py-2.5 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
		>
			<option value="">Všechny stavy</option>
			<option value="draft">Koncept</option>
			<option value="sent">Odeslaná</option>
			<option value="paid">Uhrazená</option>
			<option value="overdue">Po splatnosti</option>
			<option value="cancelled">Stornovaná</option>
		</select>
	</div>

	<!-- Error -->
	{#if error}
		<div role="alert" class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
			{error}
		</div>
	{/if}

	<!-- Table -->
	<div class="mt-4 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
		{#if loading}
			<div class="flex items-center justify-center p-12">
				<div role="status"><div class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"></div><span class="sr-only">Nacitani...</span></div>
			</div>
		{:else if invoices.length === 0}
			<div class="p-12 text-center text-gray-400">
				{search || statusFilter ? 'Žádné faktury neodpovídají filtru.' : 'Zatím žádné faktury.'}
			</div>
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-gray-200 bg-gray-50">
					<tr>
						<th class="px-4 py-3 font-medium text-gray-600">Číslo</th>
						<th class="px-4 py-3 font-medium text-gray-600">Zákazník</th>
						<th class="hidden px-4 py-3 font-medium text-gray-600 md:table-cell">Datum vystavení</th>
						<th class="hidden px-4 py-3 font-medium text-gray-600 md:table-cell">Splatnost</th>
						<th class="px-4 py-3 text-right font-medium text-gray-600">Částka</th>
						<th class="px-4 py-3 font-medium text-gray-600">Stav</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-100">
					{#each invoices as invoice}
						<tr class="hover:bg-gray-50 transition-colors">
							<td class="px-4 py-3">
								<a href="/invoices/{invoice.id}" class="font-medium text-blue-600 hover:text-blue-800">
									{invoice.invoice_number}
								</a>
							</td>
							<td class="px-4 py-3 text-gray-700">
								{invoice.customer?.name ?? '-'}
							</td>
							<td class="hidden px-4 py-3 text-gray-600 md:table-cell">
								{formatDate(invoice.issue_date)}
							</td>
							<td class="hidden px-4 py-3 text-gray-600 md:table-cell">
								{formatDate(invoice.due_date)}
							</td>
							<td class="px-4 py-3 text-right font-medium text-gray-900">
								{formatCZK(invoice.total_amount)}
							</td>
							<td class="px-4 py-3">
								<span class="inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium {statusColors[invoice.status] ?? 'bg-gray-100 text-gray-700'}">
									{statusLabels[invoice.status] ?? invoice.status}
								</span>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>

	<!-- Pagination -->
	{#if totalPages > 1}
		<div class="mt-4 flex items-center justify-between">
			<p class="text-sm text-gray-500">
				Celkem {total} faktur
			</p>
			<div class="flex gap-2">
				<button
					onclick={() => { page = Math.max(1, page - 1); loadInvoices(); }}
					disabled={page <= 1}
					class="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					Předchozí
				</button>
				<span class="flex items-center px-3 text-sm text-gray-600">
					{page} / {totalPages}
				</span>
				<button
					onclick={() => { page = Math.min(totalPages, page + 1); loadInvoices(); }}
					disabled={page >= totalPages}
					class="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					Další
				</button>
			</div>
		</div>
	{/if}
</div>
