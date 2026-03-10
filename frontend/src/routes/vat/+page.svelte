<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { vatReturnApi, type VATReturn } from '$lib/api/vat';
	import { controlStatementApi, type ControlStatement } from '$lib/api/vat-control';
	import { viesApi, type VIESSummary } from '$lib/api/vat-vies';
	import { formatCZK } from '$lib/utils/money';

	const vatStatusLabels: Record<string, string> = {
		draft: 'Koncept',
		ready: 'Pripraveno',
		filed: 'Podano'
	};

	const vatStatusColors: Record<string, string> = {
		draft: 'bg-gray-100 text-gray-700',
		ready: 'bg-blue-100 text-blue-700',
		filed: 'bg-green-100 text-green-700'
	};

	type TabKey = 'returns' | 'control' | 'vies';

	let selectedYear = $state(new Date().getFullYear());
	let activeTab = $state<TabKey>('returns');
	let vatReturns = $state<VATReturn[]>([]);
	let controlStatements = $state<ControlStatement[]>([]);
	let viesSummaries = $state<VIESSummary[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

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

	async function loadData() {
		loading = true;
		error = null;
		try {
			const [vr, cs, vs] = await Promise.all([
				vatReturnApi.list(selectedYear),
				controlStatementApi.list(selectedYear),
				viesApi.list(selectedYear),
			]);
			vatReturns = vr;
			controlStatements = cs;
			viesSummaries = vs;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se nacist data';
		} finally {
			loading = false;
		}
	}

	function formatPeriod(vr: VATReturn): string {
		if (vr.period.month > 0) {
			return `${vr.period.month}/${vr.period.year}`;
		}
		if (vr.period.quarter > 0) {
			return `Q${vr.period.quarter}/${vr.period.year}`;
		}
		return `${vr.period.year}`;
	}

	function navigateToDetail(id: number) {
		goto(`/vat/returns/${id}`);
	}

	const tabs: { key: TabKey; label: string }[] = [
		{ key: 'returns', label: 'DPH Priznani' },
		{ key: 'control', label: 'Kontrolni hlaseni' },
		{ key: 'vies', label: 'Souhrnne hlaseni' }
	];
</script>

<svelte:head>
	<title>DPH - ZFaktury</title>
</svelte:head>

<div>
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-gray-900">DPH</h1>
			<p class="mt-1 text-sm text-gray-500">Sprava danovych priznani a hlaseni</p>
		</div>
		<a
			href={activeTab === 'control' ? '/vat/control/new' : activeTab === 'vies' ? '/vat/vies/new' : '/vat/returns/new'}
			class="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 transition-colors"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
			</svg>
			{activeTab === 'control' ? 'Nove hlaseni' : activeTab === 'vies' ? 'Nove hlaseni' : 'Nove priznani'}
		</a>
	</div>

	<!-- Year selector -->
	<div class="mt-6 flex items-center gap-4">
		<label for="year" class="text-sm font-medium text-gray-700">Rok</label>
		<input
			id="year"
			type="number"
			bind:value={selectedYear}
			min="2020"
			max="2099"
			class="w-24 rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
		/>
	</div>

	<!-- Tabs -->
	<div class="mt-6 border-b border-gray-200">
		<nav class="-mb-px flex gap-6" aria-label="Tabs">
			{#each tabs as tab}
				<button
					onclick={() => { activeTab = tab.key; }}
					class="whitespace-nowrap border-b-2 px-1 py-3 text-sm font-medium transition-colors {activeTab === tab.key ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700'}"
				>
					{tab.label}
				</button>
			{/each}
		</nav>
	</div>

	<!-- Error -->
	{#if error}
		<div role="alert" class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
			{error}
		</div>
	{/if}

	<!-- Tab content -->
	{#if activeTab === 'returns'}
		<div class="mt-4 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
			{#if loading}
				<div class="flex items-center justify-center p-12">
					<div role="status">
						<div class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"></div>
						<span class="sr-only">Nacitani...</span>
					</div>
				</div>
			{:else if vatReturns.length === 0}
				<div class="p-12 text-center text-gray-400">
					Zadna DPH priznani pro rok {selectedYear}.
				</div>
			{:else}
				<table class="w-full text-left text-sm">
					<thead class="border-b border-gray-200 bg-gray-50">
						<tr>
							<th class="px-4 py-3 font-medium text-gray-600">Obdobi</th>
							<th class="px-4 py-3 font-medium text-gray-600">Typ</th>
							<th class="px-4 py-3 font-medium text-gray-600">Stav</th>
							<th class="px-4 py-3 text-right font-medium text-gray-600">DPH k odvodu</th>
							<th class="px-4 py-3 font-medium text-gray-600">Akce</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-gray-100">
						{#each vatReturns as vr}
							<tr
								class="hover:bg-gray-50 transition-colors cursor-pointer"
								role="link"
								tabindex="0"
								onclick={() => navigateToDetail(vr.id)}
								onkeydown={(e) => { if (e.key === 'Enter') navigateToDetail(vr.id); }}
							>
								<td class="px-4 py-3 font-medium text-gray-900">
									{formatPeriod(vr)}
								</td>
								<td class="px-4 py-3 text-gray-700">
									{vr.filing_type === 'regular' ? 'Radne' : vr.filing_type === 'corrective' ? 'Nasledne' : vr.filing_type === 'supplementary' ? 'Opravne' : vr.filing_type}
								</td>
								<td class="px-4 py-3">
									<span class="inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium {vatStatusColors[vr.status] ?? 'bg-gray-100 text-gray-700'}">
										{vatStatusLabels[vr.status] ?? vr.status}
									</span>
								</td>
								<td class="px-4 py-3 text-right font-medium text-gray-900">
									{formatCZK(vr.net_vat)}
								</td>
								<td class="px-4 py-3">
									<a
										href="/vat/returns/{vr.id}"
										class="text-blue-600 hover:text-blue-800 text-sm font-medium"
										onclick={(e) => e.stopPropagation()}
									>
										Detail
									</a>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}
		</div>
	{:else if activeTab === 'control'}
		<div class="mt-4 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
			{#if loading}
				<div class="flex items-center justify-center p-12">
					<div role="status">
						<div class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"></div>
						<span class="sr-only">Nacitani...</span>
					</div>
				</div>
			{:else if controlStatements.length === 0}
				<div class="p-12 text-center text-gray-400">
					Zadna kontrolni hlaseni pro rok {selectedYear}.
				</div>
			{:else}
				<table class="w-full text-left text-sm">
					<thead class="border-b border-gray-200 bg-gray-50">
						<tr>
							<th class="px-4 py-3 font-medium text-gray-600">Obdobi</th>
							<th class="px-4 py-3 font-medium text-gray-600">Typ</th>
							<th class="px-4 py-3 font-medium text-gray-600">Stav</th>
							<th class="px-4 py-3 font-medium text-gray-600">Akce</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-gray-100">
						{#each controlStatements as cs}
							<tr
								class="hover:bg-gray-50 transition-colors cursor-pointer"
								role="link"
								tabindex="0"
								onclick={() => goto(`/vat/control/${cs.id}`)}
								onkeydown={(e) => { if (e.key === 'Enter') goto(`/vat/control/${cs.id}`); }}
							>
								<td class="px-4 py-3 font-medium text-gray-900">{cs.period.month}/{cs.period.year}</td>
								<td class="px-4 py-3 text-gray-700">
									{cs.filing_type === 'regular' ? 'Radne' : cs.filing_type === 'corrective' ? 'Nasledne' : 'Opravne'}
								</td>
								<td class="px-4 py-3">
									<span class="inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium {vatStatusColors[cs.status] ?? 'bg-gray-100 text-gray-700'}">
										{vatStatusLabels[cs.status] ?? cs.status}
									</span>
								</td>
								<td class="px-4 py-3">
									<a href="/vat/control/{cs.id}" class="text-blue-600 hover:text-blue-800 text-sm font-medium" onclick={(e) => e.stopPropagation()}>Detail</a>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}
		</div>
	{:else if activeTab === 'vies'}
		<div class="mt-4 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
			{#if loading}
				<div class="flex items-center justify-center p-12">
					<div role="status">
						<div class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"></div>
						<span class="sr-only">Nacitani...</span>
					</div>
				</div>
			{:else if viesSummaries.length === 0}
				<div class="p-12 text-center text-gray-400">
					Zadna souhrnna hlaseni pro rok {selectedYear}.
				</div>
			{:else}
				<table class="w-full text-left text-sm">
					<thead class="border-b border-gray-200 bg-gray-50">
						<tr>
							<th class="px-4 py-3 font-medium text-gray-600">Obdobi</th>
							<th class="px-4 py-3 font-medium text-gray-600">Typ</th>
							<th class="px-4 py-3 font-medium text-gray-600">Stav</th>
							<th class="px-4 py-3 font-medium text-gray-600">Akce</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-gray-100">
						{#each viesSummaries as vs}
							<tr
								class="hover:bg-gray-50 transition-colors cursor-pointer"
								role="link"
								tabindex="0"
								onclick={() => goto(`/vat/vies/${vs.id}`)}
								onkeydown={(e) => { if (e.key === 'Enter') goto(`/vat/vies/${vs.id}`); }}
							>
								<td class="px-4 py-3 font-medium text-gray-900">Q{vs.period.quarter}/{vs.period.year}</td>
								<td class="px-4 py-3 text-gray-700">
									{vs.filing_type === 'regular' ? 'Radne' : vs.filing_type === 'corrective' ? 'Nasledne' : 'Opravne'}
								</td>
								<td class="px-4 py-3">
									<span class="inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium {vatStatusColors[vs.status] ?? 'bg-gray-100 text-gray-700'}">
										{vatStatusLabels[vs.status] ?? vs.status}
									</span>
								</td>
								<td class="px-4 py-3">
									<a href="/vat/vies/{vs.id}" class="text-blue-600 hover:text-blue-800 text-sm font-medium" onclick={(e) => e.stopPropagation()}>Detail</a>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}
		</div>
	{/if}
</div>
