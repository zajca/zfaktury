<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { dashboardApi } from '$lib/api/client';
	import type { DashboardData } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import { statusLabels, statusColors } from '$lib/utils/invoice';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import BarChart from '$lib/components/BarChart.svelte';

	let data = $state<DashboardData | null>(null);
	let loading = $state(true);
	let error = $state('');

	const monthLabels = ['Led', 'Uno', 'Bre', 'Dub', 'Kve', 'Cer', 'Crv', 'Srp', 'Zar', 'Rij', 'Lis', 'Pro'];

	function toMonthlyArray(items: { month: number; amount: number }[]): number[] {
		const arr = new Array(12).fill(0);
		for (const item of items) {
			arr[item.month - 1] = item.amount / 100;
		}
		return arr;
	}

	let revenueData = $derived(data ? toMonthlyArray(data.monthly_revenue) : []);
	let expenseData = $derived(data ? toMonthlyArray(data.monthly_expenses) : []);

	let chartDatasets = $derived([
		{ label: 'Prijmy', data: revenueData, backgroundColor: 'rgba(34, 197, 94, 0.7)' },
		{ label: 'Naklady', data: expenseData, backgroundColor: 'rgba(239, 68, 68, 0.7)' }
	]);

	function formatDate(dateStr: string): string {
		const parts = dateStr.split('-');
		if (parts.length !== 3) return dateStr;
		return `${parts[2]}.${parts[1]}.${parts[0]}`;
	}

	function handleRowKeydown(event: KeyboardEvent, path: string) {
		if (event.key === 'Enter') {
			goto(path);
		}
	}

	onMount(async () => {
		try {
			data = await dashboardApi.get();
		} catch (e) {
			error = 'Nepodarilo se nacist data dashboardu.';
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>ZFaktury - Dashboard</title>
</svelte:head>

<div class="mx-auto max-w-6xl">
	<h1 class="text-xl font-semibold text-primary">ZFaktury</h1>
	<p class="mt-1 text-sm text-tertiary">Prehled vaseho podnikani</p>

	{#if loading}
		<div class="mt-12 flex justify-center" role="status">
			<svg class="h-8 w-8 animate-spin text-muted" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
				<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
			</svg>
			<span class="sr-only">Nacitani...</span>
		</div>
	{:else if error}
		<div class="mt-6 rounded-lg border border-danger-border bg-danger-bg p-4 text-sm text-danger" role="alert">
			{error}
		</div>
	{:else if data}
		<!-- Stats cards -->
		<div class="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<Card>
				<p class="text-xs font-medium uppercase tracking-wider text-muted">
					Prijmy tento mesic <HelpTip topic="prijmy-naklady" />
				</p>
				<p class="mt-2 text-lg font-semibold text-primary font-mono tabular-nums sm:text-xl" title={formatCZK(data.revenue_current_month)}>{formatCZK(data.revenue_current_month)}</p>
			</Card>

			<Card>
				<p class="text-xs font-medium uppercase tracking-wider text-muted">
					Naklady tento mesic <HelpTip topic="prijmy-naklady" />
				</p>
				<p class="mt-2 text-lg font-semibold text-primary font-mono tabular-nums sm:text-xl" title={formatCZK(data.expenses_current_month)}>{formatCZK(data.expenses_current_month)}</p>
			</Card>

			<Card>
				<p class="text-xs font-medium uppercase tracking-wider text-muted">
					Neuhrazene faktury <HelpTip topic="neuhrazene-faktury" />
				</p>
				<p class="mt-2 text-lg font-semibold text-warning font-mono tabular-nums sm:text-xl">{data.unpaid_count}</p>
				<p class="mt-1 text-xs text-muted font-mono tabular-nums" title={formatCZK(data.unpaid_total)}>{formatCZK(data.unpaid_total)}</p>
			</Card>

			<Card>
				<p class="text-xs font-medium uppercase tracking-wider text-muted">
					Faktury po splatnosti <HelpTip topic="faktury-po-splatnosti" />
				</p>
				<p class="mt-2 text-lg font-semibold text-danger font-mono tabular-nums sm:text-xl">{data.overdue_count}</p>
				<p class="mt-1 text-xs text-muted font-mono tabular-nums" title={formatCZK(data.overdue_total)}>{formatCZK(data.overdue_total)}</p>
			</Card>
		</div>

		<!-- Bar chart -->
		<div class="mt-8">
			<h2 class="text-base font-semibold text-primary">Prijmy vs Naklady</h2>
			<Card class="mt-4">
				<BarChart labels={monthLabels} datasets={chartDatasets} height={300} />
			</Card>
		</div>

		<!-- Recent tables -->
		<div class="mt-8 grid grid-cols-1 gap-6 lg:grid-cols-2">
			<!-- Recent invoices -->
			<div>
				<h2 class="text-base font-semibold text-primary">Posledni faktury</h2>
				{#if data.recent_invoices.length > 0}
					<Card padding={false} class="mt-4">
						<div class="overflow-x-auto">
							<table class="w-full text-sm">
								<thead>
									<tr class="border-b border-border text-left text-xs font-medium uppercase tracking-wider text-muted">
										<th class="px-4 py-3">Cislo</th>
										<th class="px-4 py-3">Stav</th>
										<th class="px-4 py-3 text-right">Castka</th>
										<th class="px-4 py-3 text-right">Datum</th>
									</tr>
								</thead>
								<tbody class="divide-y divide-border">
									{#each data.recent_invoices as invoice}
										<tr
											class="cursor-pointer hover:bg-elevated transition-colors"
											role="link"
											tabindex="0"
											onclick={() => goto(`/invoices/${invoice.id}`)}
											onkeydown={(e) => handleRowKeydown(e, `/invoices/${invoice.id}`)}
										>
											<td class="px-4 py-3 font-medium text-primary">{invoice.invoice_number}</td>
											<td class="px-4 py-3">
												<span class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium {statusColors[invoice.status] ?? ''}">
													{statusLabels[invoice.status] ?? invoice.status}
												</span>
											</td>
											<td class="px-4 py-3 text-right font-mono tabular-nums">{formatCZK(invoice.total_amount)}</td>
											<td class="px-4 py-3 text-right text-muted">{formatDate(invoice.issue_date)}</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</Card>
				{:else}
					<Card class="mt-4 text-center text-sm text-muted">
						Zatim zadne faktury.
					</Card>
				{/if}
			</div>

			<!-- Recent expenses -->
			<div>
				<h2 class="text-base font-semibold text-primary">Posledni naklady</h2>
				{#if data.recent_expenses.length > 0}
					<Card padding={false} class="mt-4">
						<div class="overflow-x-auto">
							<table class="w-full text-sm">
								<thead>
									<tr class="border-b border-border text-left text-xs font-medium uppercase tracking-wider text-muted">
										<th class="px-4 py-3">Popis</th>
										<th class="px-4 py-3">Kategorie</th>
										<th class="px-4 py-3 text-right">Castka</th>
										<th class="px-4 py-3 text-right">Datum</th>
									</tr>
								</thead>
								<tbody class="divide-y divide-border">
									{#each data.recent_expenses as expense}
										<tr
											class="cursor-pointer hover:bg-elevated transition-colors"
											role="link"
											tabindex="0"
											onclick={() => goto(`/expenses/${expense.id}`)}
											onkeydown={(e) => handleRowKeydown(e, `/expenses/${expense.id}`)}
										>
											<td class="px-4 py-3 font-medium text-primary">{expense.description}</td>
											<td class="px-4 py-3 text-muted">{expense.category}</td>
											<td class="px-4 py-3 text-right font-mono tabular-nums">{formatCZK(expense.amount)}</td>
											<td class="px-4 py-3 text-right text-muted">{formatDate(expense.issue_date)}</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</Card>
				{:else}
					<Card class="mt-4 text-center text-sm text-muted">
						Zatim zadne naklady.
					</Card>
				{/if}
			</div>
		</div>
	{/if}
</div>
