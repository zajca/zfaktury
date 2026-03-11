<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { vatReturnApi, type VATReturn } from '$lib/api/vat';
	import { controlStatementApi, type ControlStatement } from '$lib/api/vat-control';
	import { viesApi, type VIESSummary } from '$lib/api/vat-vies';
	import { vatStatusLabels, monthLabels, quarters } from '$lib/utils/vat';

	let selectedYear = $state(new Date().getFullYear());
	let loading = $state(true);
	let error = $state<string | null>(null);

	let vatReturns = $state<VATReturn[]>([]);
	let controlStatements = $state<ControlStatement[]>([]);
	let viesSummaries = $state<VIESSummary[]>([]);

	let returnsByMonth = $derived(buildMap(vatReturns.filter(r => r.period.month > 0), r => r.period.month));
	let returnsByQuarter = $derived(buildMap(vatReturns.filter(r => r.period.month === 0 && r.period.quarter > 0), r => r.period.quarter));
	let controlByMonth = $derived(buildMap(controlStatements, r => r.period.month));
	let viesByQuarter = $derived(buildMap(viesSummaries, r => r.period.quarter));

	function buildMap<T>(items: T[], keyFn: (item: T) => number): Map<number, T> {
		const map = new Map<number, T>();
		for (const item of items) {
			map.set(keyFn(item), item);
		}
		return map;
	}

	async function loadData() {
		loading = true;
		error = null;
		try {
			const [returns, controls, vies] = await Promise.all([
				vatReturnApi.list(selectedYear),
				controlStatementApi.list(selectedYear),
				viesApi.list(selectedYear)
			]);
			vatReturns = returns ?? [];
			controlStatements = controls ?? [];
			viesSummaries = vies ?? [];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst data';
		} finally {
			loading = false;
		}
	}

	let mounted = false;
	onMount(() => { loadData(); mounted = true; });

	$effect(() => {
		selectedYear;
		if (!mounted) return;
		loadData();
	});

	function isQuarterEnd(month: number): boolean {
		return month === 3 || month === 6 || month === 9 || month === 12;
	}

	function quarterForMonth(month: number): number {
		return Math.ceil(month / 3);
	}

	function statusBtnClass(status: string | undefined): string {
		if (!status) return 'border border-dashed border-gray-300 text-gray-400 hover:border-gray-400 hover:text-gray-600 hover:bg-gray-50';
		switch (status) {
			case 'filed': return 'bg-green-600 text-white hover:bg-green-700';
			case 'ready': return 'bg-blue-600 text-white hover:bg-blue-700';
			default: return 'bg-gray-200 text-gray-800 hover:bg-gray-300';
		}
	}

	function handleReturnClick(month: number) {
		const existing = returnsByMonth.get(month);
		if (existing) {
			goto(`/vat/returns/${existing.id}`);
		} else {
			goto(`/vat/returns/new?year=${selectedYear}&month=${month}`);
		}
	}

	function handleQuarterReturnClick(quarter: number) {
		const existing = returnsByQuarter.get(quarter);
		if (existing) {
			goto(`/vat/returns/${existing.id}`);
		} else {
			goto(`/vat/returns/new?year=${selectedYear}&quarter=${quarter}`);
		}
	}

	function handleControlClick(month: number) {
		const existing = controlByMonth.get(month);
		if (existing) {
			goto(`/vat/control/${existing.id}`);
		} else {
			goto(`/vat/control/new?year=${selectedYear}&month=${month}`);
		}
	}

	function handleViesClick(quarter: number) {
		const existing = viesByQuarter.get(quarter);
		if (existing) {
			goto(`/vat/vies/${existing.id}`);
		} else {
			goto(`/vat/vies/new?year=${selectedYear}&quarter=${quarter}`);
		}
	}

	function statusLabel(status: string | undefined): string {
		if (!status) return '';
		return vatStatusLabels[status] ?? status;
	}

	function getReturnForMonth(month: number): VATReturn | undefined {
		return returnsByMonth.get(month);
	}

	function getControlForMonth(month: number): ControlStatement | undefined {
		return controlByMonth.get(month);
	}

	function getViesForQuarter(quarter: number): VIESSummary | undefined {
		return viesByQuarter.get(quarter);
	}

	function getReturnForQuarter(quarter: number): VATReturn | undefined {
		return returnsByQuarter.get(quarter);
	}
</script>

<svelte:head>
	<title>DPH za rok {selectedYear} - ZFaktury</title>
</svelte:head>

<div>
	<h1 class="text-2xl font-bold text-gray-900">DPH za rok {selectedYear}</h1>

	<!-- Year selector -->
	<div class="mt-4 flex items-center gap-3">
		<button
			onclick={() => { selectedYear--; }}
			class="rounded-lg border border-gray-300 p-2 text-gray-600 hover:bg-gray-50 transition-colors"
			aria-label="Předchozí rok"
		>
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
			</svg>
		</button>
		<span class="min-w-[4rem] text-center text-lg font-semibold text-gray-900">{selectedYear}</span>
		<button
			onclick={() => { selectedYear++; }}
			class="rounded-lg border border-gray-300 p-2 text-gray-600 hover:bg-gray-50 transition-colors"
			aria-label="Následující rok"
		>
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
			</svg>
		</button>
	</div>

	<!-- Error -->
	{#if error}
		<div role="alert" class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
			{error}
		</div>
	{/if}

	<!-- Loading -->
	{#if loading}
		<div class="mt-8 flex items-center justify-center p-12">
			<div role="status">
				<div class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"></div>
				<span class="sr-only">Načítání...</span>
			</div>
		</div>
	{:else}
		<!-- Quarter sections -->
		<div class="mt-6 space-y-6">
			{#each quarters as q}
				<div class="rounded-xl border border-gray-200 bg-white shadow-sm overflow-hidden">
					<div class="border-b border-gray-200 bg-gray-50 px-4 py-3">
						<h2 class="text-sm font-semibold text-gray-700">{q.label} {selectedYear}</h2>
					</div>
					<div class="divide-y divide-gray-100">
						{#each q.months as month}
							<div class="flex items-center gap-3 px-4 py-3">
								<span class="w-24 text-sm font-medium text-gray-900">{monthLabels[month]}</span>
								<div class="flex flex-wrap gap-2">
									<!-- DPH button -->
									<button
										onclick={() => handleReturnClick(month)}
										class="inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors {statusBtnClass(getReturnForMonth(month)?.status)}"
									>
										{#if getReturnForMonth(month)?.status === 'filed'}
											<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
												<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
											</svg>
										{/if}
										DPH
										{#if getReturnForMonth(month)}
											<span class="opacity-75">({statusLabel(getReturnForMonth(month)?.status)})</span>
										{/if}
									</button>

									<!-- KH button -->
									<button
										onclick={() => handleControlClick(month)}
										class="inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors {statusBtnClass(getControlForMonth(month)?.status)}"
									>
										{#if getControlForMonth(month)?.status === 'filed'}
											<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
												<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
											</svg>
										{/if}
										KH
										{#if getControlForMonth(month)}
											<span class="opacity-75">({statusLabel(getControlForMonth(month)?.status)})</span>
										{/if}
									</button>

									<!-- SH button (only on quarter-end months) -->
									{#if isQuarterEnd(month)}
										<button
											onclick={() => handleViesClick(quarterForMonth(month))}
											class="inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors {statusBtnClass(getViesForQuarter(quarterForMonth(month))?.status)}"
										>
											{#if getViesForQuarter(quarterForMonth(month))?.status === 'filed'}
												<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
													<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
												</svg>
											{/if}
											SH
											{#if getViesForQuarter(quarterForMonth(month))}
												<span class="opacity-75">({statusLabel(getViesForQuarter(quarterForMonth(month))?.status)})</span>
											{/if}
										</button>
									{/if}
								</div>
							</div>
						{/each}

						<!-- Cele ctvrtleti row -->
						<div class="flex items-center gap-3 bg-gray-50/50 px-4 py-3">
							<span class="w-24 text-sm font-medium text-gray-500 italic">Celé čtvrtletí</span>
							<div class="flex flex-wrap gap-2">
								<!-- Quarterly DPH -->
								<button
									onclick={() => handleQuarterReturnClick(q.quarter)}
									class="inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors {statusBtnClass(getReturnForQuarter(q.quarter)?.status)}"
								>
									{#if getReturnForQuarter(q.quarter)?.status === 'filed'}
										<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
											<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
										</svg>
									{/if}
									DPH Q{q.quarter}
									{#if getReturnForQuarter(q.quarter)}
										<span class="opacity-75">({statusLabel(getReturnForQuarter(q.quarter)?.status)})</span>
									{/if}
								</button>

								<!-- Quarterly SH -->
								<button
									onclick={() => handleViesClick(q.quarter)}
									class="inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors {statusBtnClass(getViesForQuarter(q.quarter)?.status)}"
								>
									{#if getViesForQuarter(q.quarter)?.status === 'filed'}
										<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
											<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
										</svg>
									{/if}
									SH Q{q.quarter}
									{#if getViesForQuarter(q.quarter)}
										<span class="opacity-75">({statusLabel(getViesForQuarter(q.quarter)?.status)})</span>
									{/if}
								</button>
							</div>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
