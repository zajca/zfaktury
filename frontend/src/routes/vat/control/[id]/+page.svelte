<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { controlStatementApi, type ControlStatement, type ControlStatementLine } from '$lib/api/vat-control';
	import { formatCZK } from '$lib/utils/money';

	let statement = $state<ControlStatement | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let actionLoading = $state(false);
	let activeTab = $state<string>('A4');

	let statementId = $derived(Number(page.params.id));

	const tabs = ['A4', 'A5', 'B2', 'B3'];
	const tabLabels: Record<string, string> = {
		'A4': 'A4 - Vystup nad 10 000',
		'A5': 'A5 - Vystup do 10 000',
		'B2': 'B2 - Vstup nad 10 000',
		'B3': 'B3 - Vstup do 10 000'
	};

	const filingTypeLabels: Record<string, string> = {
		regular: 'Radne',
		corrective: 'Nasledne',
		supplementary: 'Opravne'
	};

	const statusColors: Record<string, string> = {
		draft: 'bg-gray-100 text-gray-700',
		ready: 'bg-blue-100 text-blue-700',
		filed: 'bg-green-100 text-green-700'
	};

	const statusLabels: Record<string, string> = {
		draft: 'Koncept',
		ready: 'Pripraveno',
		filed: 'Podano'
	};

	let filteredLines = $derived(
		(statement?.lines ?? []).filter((l: ControlStatementLine) => l.section === activeTab)
	);

	let isDetailSection = $derived(activeTab === 'A4' || activeTab === 'B2');

	onMount(() => {
		loadStatement();
	});

	async function loadStatement() {
		loading = true;
		error = null;
		try {
			statement = await controlStatementApi.getById(statementId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se nacist kontrolni hlaseni';
		} finally {
			loading = false;
		}
	}

	async function handleRecalculate() {
		actionLoading = true;
		error = null;
		try {
			statement = await controlStatementApi.recalculate(statementId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se prepocitat';
		} finally {
			actionLoading = false;
		}
	}

	async function handleGenerateXml() {
		actionLoading = true;
		error = null;
		try {
			statement = await controlStatementApi.generateXml(statementId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se generovat XML';
		} finally {
			actionLoading = false;
		}
	}

	async function handleDownloadXml() {
		error = null;
		try {
			const blob = await controlStatementApi.downloadXml(statementId);
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `kontrolni-hlaseni-${statement?.period.year}-${String(statement?.period.month).padStart(2, '0')}.xml`;
			a.click();
			URL.revokeObjectURL(url);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se stahnout XML';
		}
	}

	async function handleMarkFiled() {
		if (!confirm('Opravdu chcete oznacit kontrolni hlaseni jako podane?')) return;
		actionLoading = true;
		error = null;
		try {
			statement = await controlStatementApi.markFiled(statementId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se oznacit jako podane';
		} finally {
			actionLoading = false;
		}
	}

	async function handleDelete() {
		if (!confirm('Opravdu chcete smazat toto kontrolni hlaseni?')) return;
		error = null;
		try {
			await controlStatementApi.delete(statementId);
			goto('/vat');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se smazat kontrolni hlaseni';
		}
	}

	function formatAmountCZK(halere: number): string {
		return formatCZK(halere);
	}
</script>

<svelte:head>
	<title>{statement ? `Kontrolni hlaseni ${statement.period.year}/${statement.period.month}` : 'Kontrolni hlaseni'} - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<a href="/vat" class="text-sm text-blue-600 hover:text-blue-800">&larr; Zpet na DPH</a>

	{#if error}
		<div role="alert" class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
			{error}
		</div>
	{/if}

	{#if loading}
		<div class="mt-8 flex items-center justify-center">
			<div role="status"><div class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"></div><span class="sr-only">Nacitani...</span></div>
		</div>
	{:else if statement}
		<!-- Header -->
		<div class="mt-4 flex items-start justify-between">
			<div>
				<h1 class="text-2xl font-bold text-gray-900">
					Kontrolni hlaseni {statement.period.year}/{String(statement.period.month).padStart(2, '0')}
				</h1>
				<div class="mt-2 flex items-center gap-3">
					<span class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium {statusColors[statement.status] || 'bg-gray-100 text-gray-700'}">
						{statusLabels[statement.status] || statement.status}
					</span>
					<span class="text-sm text-gray-500">
						{filingTypeLabels[statement.filing_type] || statement.filing_type}
					</span>
					{#if statement.filed_at}
						<span class="text-sm text-gray-500">
							Podano: {new Date(statement.filed_at).toLocaleDateString('cs-CZ')}
						</span>
					{/if}
				</div>
			</div>
			<div class="flex gap-2">
				<button
					onclick={handleRecalculate}
					disabled={actionLoading || statement.status === 'filed'}
					class="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50"
				>
					Prepocitat
				</button>
				<button
					onclick={handleGenerateXml}
					disabled={actionLoading || statement.status === 'filed'}
					class="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50"
				>
					Generovat XML
				</button>
				<button
					onclick={handleDownloadXml}
					disabled={!statement.has_xml}
					class="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50"
				>
					Stahnout XML
				</button>
				<button
					onclick={handleMarkFiled}
					disabled={actionLoading || statement.status === 'filed' || !statement.has_xml}
					class="rounded-lg bg-green-600 px-3 py-2 text-sm font-medium text-white hover:bg-green-700 transition-colors disabled:opacity-50"
				>
					Oznacit za podane
				</button>
				<button
					onclick={handleDelete}
					disabled={statement.status === 'filed'}
					class="rounded-lg border border-red-300 px-3 py-2 text-sm font-medium text-red-600 hover:bg-red-50 transition-colors disabled:opacity-50"
				>
					Smazat
				</button>
			</div>
		</div>

		<!-- Tabs -->
		<div class="mt-6 border-b border-gray-200">
			<nav class="-mb-px flex gap-4">
				{#each tabs as tab}
					<button
						onclick={() => { activeTab = tab; }}
						class="whitespace-nowrap border-b-2 px-1 py-3 text-sm font-medium transition-colors {activeTab === tab ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700'}"
					>
						{tabLabels[tab]}
					</button>
				{/each}
			</nav>
		</div>

		<!-- Lines table -->
		<div class="mt-4">
			{#if filteredLines.length === 0}
				<div class="rounded-xl border border-gray-200 bg-white p-8 text-center text-sm text-gray-500">
					Zadne radky v sekci {activeTab}
				</div>
			{:else if isDetailSection}
				<!-- A4/B2: detailed lines with partner info -->
				<div class="overflow-x-auto rounded-xl border border-gray-200 bg-white shadow-sm">
					<table class="w-full text-sm">
						<thead class="border-b border-gray-200 bg-gray-50">
							<tr>
								<th class="px-4 py-3 text-left font-medium text-gray-700">DIC partnera</th>
								<th class="px-4 py-3 text-left font-medium text-gray-700">Cislo dokladu</th>
								<th class="px-4 py-3 text-left font-medium text-gray-700">DPPD</th>
								<th class="px-4 py-3 text-right font-medium text-gray-700">Zaklad (CZK)</th>
								<th class="px-4 py-3 text-right font-medium text-gray-700">DPH (CZK)</th>
								<th class="px-4 py-3 text-right font-medium text-gray-700">Sazba</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-gray-100">
							{#each filteredLines as line}
								<tr class="hover:bg-gray-50">
									<td class="px-4 py-3 text-gray-900">{line.partner_dic}</td>
									<td class="px-4 py-3 text-gray-900">{line.document_number}</td>
									<td class="px-4 py-3 text-gray-900">{line.dppd}</td>
									<td class="px-4 py-3 text-right text-gray-900">{formatAmountCZK(line.base)}</td>
									<td class="px-4 py-3 text-right text-gray-900">{formatAmountCZK(line.vat)}</td>
									<td class="px-4 py-3 text-right text-gray-900">{line.vat_rate_percent}%</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}
				<!-- A5/B3: aggregated lines without partner info -->
				<div class="overflow-x-auto rounded-xl border border-gray-200 bg-white shadow-sm">
					<table class="w-full text-sm">
						<thead class="border-b border-gray-200 bg-gray-50">
							<tr>
								<th class="px-4 py-3 text-right font-medium text-gray-700">Zaklad (CZK)</th>
								<th class="px-4 py-3 text-right font-medium text-gray-700">DPH (CZK)</th>
								<th class="px-4 py-3 text-right font-medium text-gray-700">Sazba</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-gray-100">
							{#each filteredLines as line}
								<tr class="hover:bg-gray-50">
									<td class="px-4 py-3 text-right text-gray-900">{formatAmountCZK(line.base)}</td>
									<td class="px-4 py-3 text-right text-gray-900">{formatAmountCZK(line.vat)}</td>
									<td class="px-4 py-3 text-right text-gray-900">{line.vat_rate_percent}%</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</div>

		<div class="mt-4 text-xs text-gray-400">
			Vytvoreno: {new Date(statement.created_at).toLocaleDateString('cs-CZ')} | Upraveno: {new Date(statement.updated_at).toLocaleDateString('cs-CZ')}
		</div>
	{/if}
</div>
