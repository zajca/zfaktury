<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { recurringExpensesApi, type RecurringExpense } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import { formatDate } from '$lib/utils/date';
	import { frequencyLabels } from '$lib/utils/invoice';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import Badge from '$lib/ui/Badge.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import EmptyState from '$lib/ui/EmptyState.svelte';
	import Pagination from '$lib/ui/Pagination.svelte';

	let items = $state<RecurringExpense[]>([]);
	let total = $state(0);
	let page = $state(1);
	let perPage = $state(25);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let generating = $state(false);
	let successMessage = $state<string | null>(null);

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
		successMessage = null;
		try {
			const result = await recurringExpensesApi.generate();
			if (result.generated > 0) {
				successMessage = `Vygenerováno ${result.generated} nákladů.`;
			} else {
				successMessage = 'Žádné náklady k vygenerování.';
			}
			setTimeout(() => { successMessage = null; }, 3000);
			await loadItems();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se vygenerovat náklady';
		} finally {
			generating = false;
		}
	}

	onMount(() => {
		loadItems();
	});

	let totalPages = $derived(Math.max(1, Math.ceil(total / perPage)));

</script>

<svelte:head>
	<title>Opakované náklady - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-6xl">
	<PageHeader title="Opakované náklady" description="Automaticky generované pravidelné výdaje">
		{#snippet actions()}
			<div class="flex gap-2">
				<Button variant="secondary" onclick={handleGenerate} disabled={generating}>
					{generating ? 'Generuji...' : 'Vygenerovat splatné'}
				</Button>
				<Button variant="primary" href="/expenses/recurring/new">
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
					</svg>
					Nový opakovaný náklad
				</Button>
			</div>
		{/snippet}
	</PageHeader>

	<ErrorAlert {error} class="mt-4" />

	{#if successMessage}
		<div class="mt-4 rounded-lg border border-success/30 bg-success-bg px-4 py-3 text-sm text-success" role="status">
			{successMessage}
		</div>
	{/if}

	<Card padding={false} class="mt-6 overflow-hidden">
		{#if loading}
			<LoadingSpinner class="p-12" />
		{:else if items.length === 0}
			<EmptyState message="Zatím žádné opakované náklady." />
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-border bg-elevated">
					<tr>
						<th class="th-default">Název</th>
						<th class="th-default hidden md:table-cell">Frekvence</th>
						<th class="th-default hidden md:table-cell">Další datum</th>
						<th class="th-default hidden sm:table-cell">Stav</th>
						<th class="th-default text-right">Částka</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border-subtle">
					{#each items as item (item.id)}
						<tr
							class="hover:bg-hover transition-colors cursor-pointer"
							role="link"
							tabindex="0"
							onclick={() => {
								goto(`/expenses/recurring/${item.id}`);
							}}
							onkeydown={(e) => {
								if (e.key === 'Enter') goto(`/expenses/recurring/${item.id}`);
							}}
						>
							<td class="px-4 py-2.5">
								<a
									href="/expenses/recurring/{item.id}"
									class="font-medium text-accent-text hover:text-accent"
								>
									{item.name}
								</a>
								<p class="text-xs text-tertiary">{item.description}</p>
							</td>
							<td class="hidden px-4 py-2.5 text-secondary md:table-cell"
								>{frequencyLabels[item.frequency] ?? item.frequency}</td
							>
							<td class="hidden px-4 py-2.5 text-secondary md:table-cell"
								>{formatDate(item.next_issue_date)}</td
							>
							<td class="hidden px-4 py-2.5 sm:table-cell">
								{#if item.is_active}
									<Badge variant="success">Aktivní</Badge>
								{:else}
									<Badge variant="muted">Neaktivní</Badge>
								{/if}
							</td>
							<td class="px-4 py-2.5 text-right font-medium font-mono tabular-nums text-primary"
								>{formatCZK(item.amount)}</td
							>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</Card>

	<Pagination {page} {totalPages} {total} label="opakovaných nákladů" onPageChange={(p) => { page = p; loadItems(); }} />
</div>
