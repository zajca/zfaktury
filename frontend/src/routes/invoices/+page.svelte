<script lang="ts">
	import { invoicesApi, type Invoice } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import { formatDate } from '$lib/utils/date';
	import { statusLabels, statusVariant } from '$lib/utils/invoice';
	import Badge from '$lib/ui/Badge.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import EmptyState from '$lib/ui/EmptyState.svelte';
	import Pagination from '$lib/ui/Pagination.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';

	let invoices = $state<Invoice[]>([]);
	let total = $state(0);
	let page = $state(1);
	let perPage = $state(25);
	let search = $state('');
	let statusFilter = $state('');
	let loading = $state(true);
	let error = $state<string | null>(null);

	let searchTimeout: ReturnType<typeof setTimeout>;

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

<div class="mx-auto max-w-6xl">
	<PageHeader title="Faktury" description="Přehled vydaných faktur">
		{#snippet actions()}
			<Button variant="primary" href="/invoices/new">
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
				</svg>
				Nová faktura
			</Button>
		{/snippet}
	</PageHeader>

	<!-- Filters -->
	<div class="mt-6 flex flex-wrap items-center gap-4">
		<input
			type="text"
			bind:value={search}
			placeholder="Hledat podle čísla, zákazníka..."
			class="w-full max-w-md rounded-lg border border-border bg-surface px-4 py-2.5 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
		/>
		<select
			bind:value={statusFilter}
			class="rounded-lg border border-border bg-surface px-3 py-2.5 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
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
	<ErrorAlert {error} class="mt-4" />

	<!-- Table -->
	<Card padding={false} class="mt-4 overflow-hidden">
		{#if loading}
			<LoadingSpinner class="p-12" />
		{:else if invoices.length === 0}
			<EmptyState message="Zatím žádné faktury." filteredMessage="Žádné faktury neodpovídají filtru." isFiltered={!!(search || statusFilter)} />
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-border bg-elevated">
					<tr>
						<th class="th-default">Číslo</th>
						<th class="th-default">Zákazník</th>
						<th class="th-default hidden md:table-cell">Datum vystavení</th>
						<th class="th-default hidden md:table-cell">Splatnost</th>
						<th class="th-default text-right">Částka</th>
						<th class="th-default">Stav</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border-subtle">
					{#each invoices as invoice (invoice.id)}
						<tr class="hover:bg-hover transition-colors cursor-pointer">
							<td class="px-4 py-2.5">
								<a
									href="/invoices/{invoice.id}"
									class="text-accent-text hover:text-accent font-medium"
								>
									{invoice.invoice_number}
								</a>
							</td>
							<td class="px-4 py-2.5 text-secondary">
								{invoice.customer?.name ?? '-'}
							</td>
							<td class="hidden px-4 py-2.5 text-secondary md:table-cell">
								{formatDate(invoice.issue_date)}
							</td>
							<td class="hidden px-4 py-2.5 text-secondary md:table-cell">
								{formatDate(invoice.due_date)}
							</td>
							<td class="px-4 py-2.5 text-right font-mono tabular-nums font-medium text-primary">
								{formatCZK(invoice.total_amount)}
							</td>
							<td class="px-4 py-2.5">
								<Badge variant={statusVariant[invoice.status] ?? 'default'}>
									{statusLabels[invoice.status] ?? invoice.status}
								</Badge>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</Card>

	<!-- Pagination -->
	<Pagination {page} {totalPages} {total} label="faktur" onPageChange={(p) => { page = p; loadInvoices(); }} />
</div>
