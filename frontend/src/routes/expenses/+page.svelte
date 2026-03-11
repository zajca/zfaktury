<script lang="ts">
	import { goto } from '$app/navigation';
	import { expensesApi, type Expense } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import { formatDate } from '$lib/utils/date';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';

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
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-xl font-semibold text-primary">Náklady</h1>
			<p class="mt-1 text-sm text-tertiary">Evidence výdajů a nákladů</p>
		</div>
		<Button variant="primary" href="/expenses/new">
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
			</svg>
			Přidat náklad
		</Button>
	</div>

	<!-- Search -->
	<div class="mt-6">
		<input
			type="text"
			bind:value={search}
			placeholder="Hledat podle popisu, dodavatele..."
			class="w-full max-w-md rounded-lg border border-border bg-elevated px-4 py-2.5 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
		/>
	</div>

	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-danger/20 bg-danger-bg p-4 text-sm text-danger"
		>
			{error}
		</div>
	{/if}

	<Card padding={false} class="mt-4 overflow-hidden">
		{#if loading}
			<div class="flex items-center justify-center p-12">
				<div role="status">
					<div
						class="h-8 w-8 animate-spin rounded-full border-4 border-border border-t-accent"
					></div>
					<span class="sr-only">Nacitani...</span>
				</div>
			</div>
		{:else if expenses.length === 0}
			<div class="p-12 text-center text-muted">
				{search ? 'Žádné náklady neodpovídají hledání.' : 'Zatím žádné náklady.'}
			</div>
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-border bg-elevated">
					<tr>
						<th class="px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">Popis</th>
						<th class="hidden px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted md:table-cell">Kategorie</th>
						<th class="hidden px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted md:table-cell">Datum</th>
						<th class="px-4 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted">Částka</th>
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

	{#if totalPages > 1}
		<div class="mt-4 flex items-center justify-between">
			<p class="text-sm text-tertiary">Celkem {total} nákladů</p>
			<div class="flex gap-2">
				<Button
					variant="secondary"
					size="sm"
					onclick={() => {
						page = Math.max(1, page - 1);
						loadExpenses();
					}}
					disabled={page <= 1}
				>
					Předchozí
				</Button>
				<span class="flex items-center px-3 text-sm text-secondary">{page} / {totalPages}</span>
				<Button
					variant="secondary"
					size="sm"
					onclick={() => {
						page = Math.min(totalPages, page + 1);
						loadExpenses();
					}}
					disabled={page >= totalPages}
				>
					Další
				</Button>
			</div>
		</div>
	{/if}
</div>
