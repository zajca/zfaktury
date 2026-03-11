<script lang="ts">
	import { onMount } from 'svelte';
	import { SvelteMap } from 'svelte/reactivity';
	import { goto } from '$app/navigation';
	import { vatReturnApi, type VATReturn } from '$lib/api/vat';
	import { controlStatementApi, type ControlStatement } from '$lib/api/vat-control';
	import { viesApi, type VIESSummary } from '$lib/api/vat-vies';
	import { vatStatusLabels, monthLabels, quarters } from '$lib/utils/vat';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';

	let selectedYear = $state(new Date().getFullYear());
	let loading = $state(true);
	let error = $state<string | null>(null);

	let vatReturns = $state<VATReturn[]>([]);
	let controlStatements = $state<ControlStatement[]>([]);
	let viesSummaries = $state<VIESSummary[]>([]);

	let returnsByMonth = $derived(
		buildMap(
			vatReturns.filter((r) => r.period.month > 0),
			(r) => r.period.month
		)
	);
	let returnsByQuarter = $derived(
		buildMap(
			vatReturns.filter((r) => r.period.month === 0 && r.period.quarter > 0),
			(r) => r.period.quarter
		)
	);
	let controlByMonth = $derived(buildMap(controlStatements, (r) => r.period.month));
	let viesByQuarter = $derived(buildMap(viesSummaries, (r) => r.period.quarter));

	function buildMap<T>(items: T[], keyFn: (item: T) => number): SvelteMap<number, T> {
		const map = new SvelteMap<number, T>();
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
	onMount(() => {
		loadData();
		mounted = true;
	});

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
		if (!status)
			return 'border border-dashed border-border-strong text-muted hover:border-border-strong hover:text-secondary hover:bg-hover';
		switch (status) {
			case 'filed':
				return 'bg-success-bg text-success';
			case 'ready':
				return 'bg-info-bg text-info';
			default:
				return 'bg-elevated text-secondary hover:bg-hover';
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

<div class="mx-auto max-w-6xl">
	<h1 class="text-xl font-semibold text-primary">DPH za rok {selectedYear}</h1>

	<!-- Year selector -->
	<div class="mt-4 flex items-center gap-3">
		<Button
			variant="ghost"
			size="sm"
			onclick={() => {
				selectedYear--;
			}}
			title="Předchozí rok"
			aria-label="Předchozí rok"
		>
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
			</svg>
		</Button>
		<span class="min-w-[4rem] text-center text-xl font-semibold text-primary tabular-nums">{selectedYear}</span>
		<Button
			variant="ghost"
			size="sm"
			onclick={() => {
				selectedYear++;
			}}
			title="Následující rok"
			aria-label="Následující rok"
		>
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
			</svg>
		</Button>
	</div>

	<!-- Error -->
	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-danger/20 bg-danger-bg p-4 text-sm text-danger"
		>
			{error}
		</div>
	{/if}

	<!-- Loading -->
	{#if loading}
		<div class="mt-8 flex items-center justify-center p-12">
			<div role="status">
				<div
					class="h-8 w-8 animate-spin rounded-full border-4 border-border border-t-accent"
				></div>
				<span class="sr-only">Načítání...</span>
			</div>
		</div>
	{:else}
		<!-- Quarter sections -->
		<div class="mt-6 space-y-6">
			{#each quarters as q (q.quarter)}
				<Card padding={false} class="overflow-hidden">
					<div class="bg-elevated px-4 py-2.5 text-sm font-medium text-secondary border-b border-border">
						{q.label} {selectedYear}
					</div>
					<div class="divide-y divide-border-subtle">
						{#each q.months as month (month)}
							<div class="flex items-center gap-3 px-4 py-2.5 border-b border-border-subtle">
								<span class="w-24 text-sm font-medium text-primary">{monthLabels[month]}</span>
								<div class="flex flex-wrap gap-2">
									<!-- DPH button -->
									<button
										onclick={() => handleReturnClick(month)}
										class="inline-flex items-center gap-1.5 rounded-md px-3 py-1 text-xs font-medium transition-colors {statusBtnClass(
											getReturnForMonth(month)?.status
										)}"
									>
										{#if getReturnForMonth(month)?.status === 'filed'}
											<svg
												class="h-3.5 w-3.5"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
												stroke-width="2"
											>
												<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
											</svg>
										{/if}
										DPH
										{#if getReturnForMonth(month)}
											<span class="opacity-75"
												>({statusLabel(getReturnForMonth(month)?.status)})</span
											>
										{/if}
									</button>

									<!-- KH button -->
									<button
										onclick={() => handleControlClick(month)}
										class="inline-flex items-center gap-1.5 rounded-md px-3 py-1 text-xs font-medium transition-colors {statusBtnClass(
											getControlForMonth(month)?.status
										)}"
									>
										{#if getControlForMonth(month)?.status === 'filed'}
											<svg
												class="h-3.5 w-3.5"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
												stroke-width="2"
											>
												<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
											</svg>
										{/if}
										KH
										{#if getControlForMonth(month)}
											<span class="opacity-75"
												>({statusLabel(getControlForMonth(month)?.status)})</span
											>
										{/if}
									</button>

									<!-- SH button (only on quarter-end months) -->
									{#if isQuarterEnd(month)}
										<button
											onclick={() => handleViesClick(quarterForMonth(month))}
											class="inline-flex items-center gap-1.5 rounded-md px-3 py-1 text-xs font-medium transition-colors {statusBtnClass(
												getViesForQuarter(quarterForMonth(month))?.status
											)}"
										>
											{#if getViesForQuarter(quarterForMonth(month))?.status === 'filed'}
												<svg
													class="h-3.5 w-3.5"
													fill="none"
													viewBox="0 0 24 24"
													stroke="currentColor"
													stroke-width="2"
												>
													<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
												</svg>
											{/if}
											SH
											{#if getViesForQuarter(quarterForMonth(month))}
												<span class="opacity-75"
													>({statusLabel(getViesForQuarter(quarterForMonth(month))?.status)})</span
												>
											{/if}
										</button>
									{/if}
								</div>
							</div>
						{/each}

						<!-- Cele ctvrtleti row -->
						<div class="flex items-center gap-3 bg-elevated/50 px-4 py-2.5">
							<span class="w-24 text-sm font-medium text-tertiary italic">Celé čtvrtletí</span>
							<div class="flex flex-wrap gap-2">
								<!-- Quarterly DPH -->
								<button
									onclick={() => handleQuarterReturnClick(q.quarter)}
									class="inline-flex items-center gap-1.5 rounded-md px-3 py-1 text-xs font-medium transition-colors {statusBtnClass(
										getReturnForQuarter(q.quarter)?.status
									)}"
								>
									{#if getReturnForQuarter(q.quarter)?.status === 'filed'}
										<svg
											class="h-3.5 w-3.5"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
											stroke-width="2"
										>
											<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
										</svg>
									{/if}
									DPH Q{q.quarter}
									{#if getReturnForQuarter(q.quarter)}
										<span class="opacity-75"
											>({statusLabel(getReturnForQuarter(q.quarter)?.status)})</span
										>
									{/if}
								</button>

								<!-- Quarterly SH -->
								<button
									onclick={() => handleViesClick(q.quarter)}
									class="inline-flex items-center gap-1.5 rounded-md px-3 py-1 text-xs font-medium transition-colors {statusBtnClass(
										getViesForQuarter(q.quarter)?.status
									)}"
								>
									{#if getViesForQuarter(q.quarter)?.status === 'filed'}
										<svg
											class="h-3.5 w-3.5"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
											stroke-width="2"
										>
											<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
										</svg>
									{/if}
									SH Q{q.quarter}
									{#if getViesForQuarter(q.quarter)}
										<span class="opacity-75"
											>({statusLabel(getViesForQuarter(q.quarter)?.status)})</span
										>
									{/if}
								</button>
							</div>
						</div>
					</div>
				</Card>
			{/each}
		</div>
	{/if}
</div>
