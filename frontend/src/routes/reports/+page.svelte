<script lang="ts">
	import { onMount } from 'svelte';
	import { reportsApi, exportApi } from '$lib/api/client';
	import type {
		RevenueReport,
		ExpenseReport,
		ProfitLossReport,
		TopCustomer,
		TaxCalendar,
		MonthlyAmount
	} from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import Card from '$lib/ui/Card.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import BarChart from '$lib/components/BarChart.svelte';
	import DoughnutChart from '$lib/components/DoughnutChart.svelte';

	// --- State ---

	type Tab = 'revenue' | 'expenses' | 'profit-loss' | 'top-customers' | 'tax-calendar';

	const tabs: { id: Tab; label: string }[] = [
		{ id: 'revenue', label: 'Příjmy' },
		{ id: 'expenses', label: 'Náklady' },
		{ id: 'profit-loss', label: 'Zisk a ztráta' },
		{ id: 'top-customers', label: 'Top zákazníci' },
		{ id: 'tax-calendar', label: 'Daňový kalendář' }
	];

	let activeTab = $state<Tab>('revenue');
	let year = $state(new Date().getFullYear());
	let loading = $state(false);
	let error = $state<string | null>(null);
	let mounted = false;

	// Tab data
	let revenueData = $state<RevenueReport | null>(null);
	let expenseData = $state<ExpenseReport | null>(null);
	let profitLossData = $state<ProfitLossReport | null>(null);
	let topCustomersData = $state<TopCustomer[] | null>(null);
	let taxCalendarData = $state<TaxCalendar | null>(null);

	const monthLabels = ['Led', 'Úno', 'Bře', 'Dub', 'Kvě', 'Čer', 'Črc', 'Srp', 'Zář', 'Říj', 'Lis', 'Pro'];

	const chartColors = [
		'#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#8b5cf6',
		'#ec4899', '#14b8a6', '#f97316', '#6366f1', '#84cc16',
		'#06b6d4', '#d946ef'
	];

	const deadlineTypeLabels: Record<string, string> = {
		vat: 'DPH',
		income_tax: 'Daň z příjmů',
		social: 'Sociální pojištění',
		health: 'Zdravotní pojištění',
		advance: 'Záloha na daň'
	};

	const deadlineTypeColors: Record<string, string> = {
		vat: 'bg-blue-100 text-blue-800',
		income_tax: 'bg-purple-100 text-purple-800',
		social: 'bg-green-100 text-green-800',
		health: 'bg-orange-100 text-orange-800',
		advance: 'bg-gray-100 text-gray-800'
	};

	// --- Helpers ---

	function toMonthlyArray(items: MonthlyAmount[]): number[] {
		const arr = new Array(12).fill(0);
		for (const item of items) {
			arr[item.month - 1] = item.amount / 100;
		}
		return arr;
	}

	function formatDateCZ(dateStr: string): string {
		const d = new Date(dateStr);
		const day = d.getDate().toString().padStart(2, '0');
		const month = (d.getMonth() + 1).toString().padStart(2, '0');
		const y = d.getFullYear();
		return `${day}.${month}.${y}`;
	}

	function isDeadlinePast(dateStr: string): boolean {
		const today = new Date();
		today.setHours(0, 0, 0, 0);
		const deadline = new Date(dateStr);
		deadline.setHours(0, 0, 0, 0);
		return deadline < today;
	}

	// --- Data loading ---

	async function loadTabData() {
		loading = true;
		error = null;
		try {
			switch (activeTab) {
				case 'revenue':
					revenueData = await reportsApi.revenue(year);
					break;
				case 'expenses':
					expenseData = await reportsApi.expenses(year);
					break;
				case 'profit-loss':
					profitLossData = await reportsApi.profitLoss(year);
					break;
				case 'top-customers':
					topCustomersData = await reportsApi.topCustomers(year);
					break;
				case 'tax-calendar':
					taxCalendarData = await reportsApi.taxCalendar(year);
					break;
			}
		} catch {
			error = 'Nepodařilo se načíst data.';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadTabData();
		mounted = true;
	});

	$effect(() => {
		activeTab;
		year;
		if (!mounted) return;
		loadTabData();
	});

	// --- Derived chart data ---

	let revenueMonthly = $derived(revenueData ? toMonthlyArray(revenueData.monthly) : []);

	let expenseMonthly = $derived(expenseData ? toMonthlyArray(expenseData.monthly) : []);

	let profitRevenueMonthly = $derived(profitLossData ? toMonthlyArray(profitLossData.monthly_revenue) : []);
	let profitExpenseMonthly = $derived(profitLossData ? toMonthlyArray(profitLossData.monthly_expenses) : []);

	let profitMonthlyTable = $derived(
		profitRevenueMonthly.map((rev, i) => ({
			month: monthLabels[i],
			revenue: rev * 100,
			expenses: profitExpenseMonthly[i] * 100,
			profit: (rev - profitExpenseMonthly[i]) * 100
		}))
	);

	let plRevenueTotal = $derived(
		profitLossData ? profitLossData.monthly_revenue.reduce((sum, m) => sum + m.amount, 0) : 0
	);
	let plExpenseTotal = $derived(
		profitLossData ? profitLossData.monthly_expenses.reduce((sum, m) => sum + m.amount, 0) : 0
	);
	let plProfitTotal = $derived(plRevenueTotal - plExpenseTotal);

	let pastDeadlines = $derived(
		taxCalendarData?.deadlines.filter((d) => isDeadlinePast(d.date)) ?? []
	);
	let upcomingDeadlines = $derived(
		taxCalendarData?.deadlines.filter((d) => !isDeadlinePast(d.date)) ?? []
	);
</script>

<svelte:head>
	<title>Přehledy - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-6xl">
	<div class="flex items-start justify-between gap-4">
		<PageHeader title="Přehledy" description="Finanční přehledy a statistiky" />

		<div class="flex items-center gap-3">
			<a
				href={exportApi.invoicesUrl(year)}
				download
				class="inline-flex items-center gap-1.5 rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-secondary hover:bg-hover"
			>
				Export faktur (CSV)
			</a>
			<a
				href={exportApi.expensesUrl(year)}
				download
				class="inline-flex items-center gap-1.5 rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-secondary hover:bg-hover"
			>
				Export nákladů (CSV)
			</a>
		</div>
	</div>

	<!-- Year selector -->
	<div class="mt-4 flex items-center gap-3">
		<label for="year-input" class="text-sm font-medium text-secondary">Rok:</label>
		<input
			id="year-input"
			type="number"
			min="2000"
			max="2100"
			bind:value={year}
			class="w-24 rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-primary"
		/>
	</div>

	<!-- Tab navigation -->
	<div class="mt-6 flex gap-1 border-b border-border">
		{#each tabs as tab}
			<button
				class="px-4 py-2 text-sm font-medium transition-colors {activeTab === tab.id
					? 'border-b-2 border-accent text-accent'
					: 'text-secondary hover:text-primary'}"
				onclick={() => (activeTab = tab.id)}
			>
				{tab.label}
			</button>
		{/each}
	</div>

	<!-- Error -->
	<ErrorAlert error={error} class="mt-4" />

	<!-- Loading -->
	{#if loading}
		<LoadingSpinner class="mt-12" />
	{:else}
		<!-- Tab content -->
		<div class="mt-6">
			<!-- Revenue tab -->
			{#if activeTab === 'revenue' && revenueData}
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5">
					<Card class="lg:col-span-1">
						<p class="text-xs font-medium uppercase tracking-wider text-muted">Celkové příjmy</p>
						<p class="mt-2 text-2xl font-semibold text-primary font-mono tabular-nums">
							{formatCZK(revenueData.total)}
						</p>
					</Card>
					{#each revenueData.quarterly as q}
						<Card>
							<p class="text-xs font-medium uppercase tracking-wider text-muted">Q{q.quarter}</p>
							<p class="mt-2 text-lg font-semibold text-primary font-mono tabular-nums">
								{formatCZK(q.amount)}
							</p>
						</Card>
					{/each}
				</div>

				<Card class="mt-6">
					<h3 class="mb-4 text-sm font-medium text-secondary">Měsíční příjmy</h3>
					<BarChart
						labels={monthLabels}
						datasets={[
							{
								label: 'Příjmy (Kč)',
								data: revenueMonthly,
								backgroundColor: '#3b82f6'
							}
						]}
						height={350}
					/>
				</Card>
			{/if}

			<!-- Expenses tab -->
			{#if activeTab === 'expenses' && expenseData}
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
					{#each expenseData.quarterly as q}
						<Card>
							<p class="text-xs font-medium uppercase tracking-wider text-muted">Q{q.quarter}</p>
							<p class="mt-2 text-lg font-semibold text-primary font-mono tabular-nums">
								{formatCZK(q.amount)}
							</p>
						</Card>
					{/each}
				</div>

				<div class="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-2">
					<Card>
						<h3 class="mb-4 text-sm font-medium text-secondary">Měsíční náklady</h3>
						<BarChart
							labels={monthLabels}
							datasets={[
								{
									label: 'Náklady (Kč)',
									data: expenseMonthly,
									backgroundColor: '#ef4444'
								}
							]}
							height={350}
						/>
					</Card>

					<Card>
						<h3 class="mb-4 text-sm font-medium text-secondary">Náklady dle kategorie</h3>
						{#if expenseData.categories.length > 0}
							<DoughnutChart
								labels={expenseData.categories.map((c) => c.category)}
								data={expenseData.categories.map((c) => c.amount / 100)}
								backgroundColor={chartColors.slice(0, expenseData.categories.length)}
								height={350}
							/>
						{:else}
							<p class="py-8 text-center text-sm text-muted">Žádné náklady v tomto roce.</p>
						{/if}
					</Card>
				</div>
			{/if}

			<!-- Profit & Loss tab -->
			{#if activeTab === 'profit-loss' && profitLossData}
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
					<Card>
						<p class="text-xs font-medium uppercase tracking-wider text-muted">Celkové příjmy</p>
						<p class="mt-2 text-xl font-semibold text-success font-mono tabular-nums">
							{formatCZK(plRevenueTotal)}
						</p>
					</Card>
					<Card>
						<p class="text-xs font-medium uppercase tracking-wider text-muted">Celkové náklady</p>
						<p class="mt-2 text-xl font-semibold text-danger font-mono tabular-nums">
							{formatCZK(plExpenseTotal)}
						</p>
					</Card>
					<Card>
						<p class="text-xs font-medium uppercase tracking-wider text-muted">Zisk / Ztráta</p>
						<p
							class="mt-2 text-xl font-semibold font-mono tabular-nums {plProfitTotal >= 0
								? 'text-success'
								: 'text-danger'}"
						>
							{formatCZK(plProfitTotal)}
						</p>
					</Card>
				</div>

				<Card class="mt-6">
					<h3 class="mb-4 text-sm font-medium text-secondary">Příjmy vs. Náklady</h3>
					<BarChart
						labels={monthLabels}
						datasets={[
							{
								label: 'Příjmy (Kč)',
								data: profitRevenueMonthly,
								backgroundColor: '#22c55e'
							},
							{
								label: 'Náklady (Kč)',
								data: profitExpenseMonthly,
								backgroundColor: '#ef4444'
							}
						]}
						height={350}
					/>
				</Card>

				<Card class="mt-6" padding={false}>
					<div class="overflow-x-auto">
						<table class="w-full text-sm">
							<thead>
								<tr class="border-b border-border bg-muted-bg">
									<th class="px-4 py-2 text-left font-medium text-secondary">Měsíc</th>
									<th class="px-4 py-2 text-right font-medium text-secondary">Příjmy</th>
									<th class="px-4 py-2 text-right font-medium text-secondary">Náklady</th>
									<th class="px-4 py-2 text-right font-medium text-secondary">Zisk / Ztráta</th>
								</tr>
							</thead>
							<tbody>
								{#each profitMonthlyTable as row}
									<tr class="border-b border-border last:border-0">
										<td class="px-4 py-2 text-primary">{row.month}</td>
										<td class="px-4 py-2 text-right font-mono tabular-nums text-success">
											{formatCZK(row.revenue)}
										</td>
										<td class="px-4 py-2 text-right font-mono tabular-nums text-danger">
											{formatCZK(row.expenses)}
										</td>
										<td
											class="px-4 py-2 text-right font-mono tabular-nums {row.profit >= 0
												? 'text-success'
												: 'text-danger'}"
										>
											{formatCZK(row.profit)}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</Card>
			{/if}

			<!-- Top Customers tab -->
			{#if activeTab === 'top-customers' && topCustomersData}
				{#if topCustomersData.length === 0}
					<Card class="text-center text-sm text-muted">
						Žádní zákazníci v tomto roce.
					</Card>
				{:else}
					<Card padding={false}>
						<div class="overflow-x-auto">
							<table class="w-full text-sm">
								<thead>
									<tr class="border-b border-border bg-muted-bg">
										<th class="px-4 py-2 text-left font-medium text-secondary">#</th>
										<th class="px-4 py-2 text-left font-medium text-secondary">Zákazník</th>
										<th class="px-4 py-2 text-right font-medium text-secondary">Počet faktur</th>
										<th class="px-4 py-2 text-right font-medium text-secondary">Příjmy</th>
									</tr>
								</thead>
								<tbody>
									{#each topCustomersData as customer, i}
										<tr class="border-b border-border last:border-0">
											<td class="px-4 py-2 text-muted">{i + 1}</td>
											<td class="px-4 py-2 text-primary">{customer.customer_name}</td>
											<td class="px-4 py-2 text-right font-mono tabular-nums text-secondary">
												{customer.invoice_count}
											</td>
											<td class="px-4 py-2 text-right font-mono tabular-nums text-primary">
												{formatCZK(customer.total)}
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</Card>
				{/if}
			{/if}

			<!-- Tax Calendar tab -->
			{#if activeTab === 'tax-calendar' && taxCalendarData}
				{#if taxCalendarData.deadlines.length === 0}
					<Card class="text-center text-sm text-muted">
						Žádné daňové povinnosti pro tento rok.
					</Card>
				{:else}
					<!-- Upcoming deadlines -->
					{#if upcomingDeadlines.length > 0}
						<h3 class="mb-3 text-sm font-semibold uppercase tracking-wider text-secondary">
							Nadcházející termíny
						</h3>
						<div class="space-y-2">
							{#each upcomingDeadlines as deadline}
								<Card>
									<div class="flex items-start justify-between gap-4">
										<div>
											<div class="flex items-center gap-2">
												<span class="font-medium text-primary">{deadline.name}</span>
												<span
													class="inline-block rounded-full px-2 py-0.5 text-xs font-medium {deadlineTypeColors[
														deadline.type
													] ?? 'bg-gray-100 text-gray-800'}"
												>
													{deadlineTypeLabels[deadline.type] ?? deadline.type}
												</span>
											</div>
											{#if deadline.description}
												<p class="mt-1 text-sm text-muted">{deadline.description}</p>
											{/if}
										</div>
										<span class="shrink-0 text-sm font-mono tabular-nums text-secondary">
											{formatDateCZ(deadline.date)}
										</span>
									</div>
								</Card>
							{/each}
						</div>
					{/if}

					<!-- Past deadlines -->
					{#if pastDeadlines.length > 0}
						<h3
							class="mb-3 text-sm font-semibold uppercase tracking-wider text-secondary {upcomingDeadlines.length > 0
								? 'mt-8'
								: ''}"
						>
							Uplynulé termíny
						</h3>
						<div class="space-y-2 opacity-60">
							{#each pastDeadlines as deadline}
								<Card>
									<div class="flex items-start justify-between gap-4">
										<div>
											<div class="flex items-center gap-2">
												<span class="font-medium text-primary">{deadline.name}</span>
												<span
													class="inline-block rounded-full px-2 py-0.5 text-xs font-medium {deadlineTypeColors[
														deadline.type
													] ?? 'bg-gray-100 text-gray-800'}"
												>
													{deadlineTypeLabels[deadline.type] ?? deadline.type}
												</span>
											</div>
											{#if deadline.description}
												<p class="mt-1 text-sm text-muted">{deadline.description}</p>
											{/if}
										</div>
										<span class="shrink-0 text-sm font-mono tabular-nums text-secondary">
											{formatDateCZ(deadline.date)}
										</span>
									</div>
								</Card>
							{/each}
						</div>
					{/if}
				{/if}
			{/if}
		</div>
	{/if}
</div>
