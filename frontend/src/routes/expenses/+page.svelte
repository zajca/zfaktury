<script lang="ts">
	import { goto } from '$app/navigation';
	import { expensesApi, type Expense } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import { formatDate } from '$lib/utils/date';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import EmptyState from '$lib/ui/EmptyState.svelte';
	import Pagination from '$lib/ui/Pagination.svelte';

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
			const res = await expensesApi.list({
				limit: perPage,
				offset: (page - 1) * perPage,
				search: search || undefined
			});
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

<div class="mx-auto max-w-6xl">
	<PageHeader title="Náklady" description="Evidence výdajů a nákladů">
		{#snippet actions()}
			<div class="flex gap-2">
				<Button variant="secondary" href="/expenses/import">Import z dokladu</Button>
				<Button variant="secondary" href="/expenses/recurring">Opakované</Button>
				<Button variant="secondary" href="/expenses/review">Daňový audit</Button>
				<Button variant="primary" href="/expenses/new">
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
					</svg>
					Přidat náklad
				</Button>
			</div>
		{/snippet}
	</PageHeader>

	<!-- Search -->
	<div class="mt-6">
		<input
			type="text"
			bind:value={search}
			placeholder="Hledat podle popisu, dodavatele..."
			class="w-full max-w-md rounded-lg border border-border bg-elevated px-4 py-2.5 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
		/>
	</div>

	<ErrorAlert {error} class="mt-4" />

	<Card padding={false} class="mt-4 overflow-hidden">
		{#if loading}
			<LoadingSpinner class="p-12" />
		{:else if expenses.length === 0}
			<EmptyState message="Zatím žádné náklady." filteredMessage="Žádné náklady neodpovídají hledání." isFiltered={!!search} />
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-border bg-elevated">
					<tr>
						<th class="th-default">Popis</th>
						<th class="th-default hidden md:table-cell">Kategorie</th>
						<th class="th-default hidden md:table-cell">Datum</th>
						<th class="th-default text-right">Částka</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border-subtle">
					{#each expenses as expense (expense.id)}
						<tr
							class="hover:bg-hover transition-colors cursor-pointer"
							role="link"
							tabindex="0"
							onclick={() => {
								goto(`/expenses/${expense.id}`);
							}}
							onkeydown={(e) => {
								if (e.key === 'Enter') goto(`/expenses/${expense.id}`);
							}}
						>
							<td class="px-4 py-2.5">
								<a
									href="/expenses/{expense.id}"
									class="font-medium text-accent-text hover:text-accent"
								>
									{expense.description}
								</a>
							</td>
							<td class="hidden px-4 py-2.5 text-secondary md:table-cell">{expense.category || '-'}</td
							>
							<td class="hidden px-4 py-2.5 text-secondary md:table-cell"
								>{formatDate(expense.issue_date)}</td
							>
							<td class="px-4 py-2.5 text-right font-medium font-mono tabular-nums text-primary"
								>{formatCZK(expense.amount)}</td
							>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</Card>

	<Pagination {page} {totalPages} {total} label="nákladů" onPageChange={(p) => { page = p; loadExpenses(); }} />
</div>
