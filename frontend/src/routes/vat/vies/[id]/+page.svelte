<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { viesApi, type VIESSummary } from '$lib/api/vat-vies';
	import { formatCZK } from '$lib/utils/money';

	let summary = $state<VIESSummary | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let actionLoading = $state(false);

	let summaryId = $derived(Number(page.params.id));

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

	const quarterLabels: Record<number, string> = {
		1: 'Q1 (leden - brezen)',
		2: 'Q2 (duben - cerven)',
		3: 'Q3 (cervenec - zari)',
		4: 'Q4 (rijen - prosinec)'
	};

	onMount(() => {
		loadSummary();
	});

	async function loadSummary() {
		loading = true;
		error = null;
		try {
			summary = await viesApi.getById(summaryId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se nacist souhrnne hlaseni';
		} finally {
			loading = false;
		}
	}

	async function handleRecalculate() {
		actionLoading = true;
		error = null;
		try {
			summary = await viesApi.recalculate(summaryId);
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
			summary = await viesApi.generateXml(summaryId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se generovat XML';
		} finally {
			actionLoading = false;
		}
	}

	async function handleDownloadXml() {
		error = null;
		try {
			const blob = await viesApi.downloadXml(summaryId);
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `souhrnne-hlaseni-${summary?.period.year}-Q${summary?.period.quarter}.xml`;
			document.body.appendChild(a);
			a.click();
			document.body.removeChild(a);
			URL.revokeObjectURL(url);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se stahnout XML';
		}
	}

	async function handleMarkFiled() {
		if (!confirm('Opravdu chcete oznacit souhrnne hlaseni jako podane?')) return;
		actionLoading = true;
		error = null;
		try {
			summary = await viesApi.markFiled(summaryId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se oznacit jako podane';
		} finally {
			actionLoading = false;
		}
	}

	async function handleDelete() {
		if (!confirm('Opravdu chcete smazat toto souhrnne hlaseni?')) return;
		error = null;
		try {
			await viesApi.delete(summaryId);
			goto('/vat');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se smazat souhrnne hlaseni';
		}
	}
</script>

<svelte:head>
	<title>{summary ? `Souhrnne hlaseni ${summary.period.year} Q${summary.period.quarter}` : 'Souhrnne hlaseni'} - ZFaktury</title>
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
	{:else if summary}
		<!-- Header -->
		<div class="mt-4 flex items-start justify-between">
			<div>
				<h1 class="text-2xl font-bold text-gray-900">
					Souhrnne hlaseni {summary.period.year} Q{summary.period.quarter}
				</h1>
				<div class="mt-2 flex items-center gap-3">
					<span class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium {statusColors[summary.status] || 'bg-gray-100 text-gray-700'}">
						{statusLabels[summary.status] || summary.status}
					</span>
					<span class="text-sm text-gray-500">
						{filingTypeLabels[summary.filing_type] || summary.filing_type}
					</span>
					<span class="text-sm text-gray-500">
						{quarterLabels[summary.period.quarter] || `Q${summary.period.quarter}`}
					</span>
					{#if summary.filed_at}
						<span class="text-sm text-gray-500">
							Podano: {new Date(summary.filed_at).toLocaleDateString('cs-CZ')}
						</span>
					{/if}
				</div>
			</div>
			<div class="flex flex-wrap gap-2">
				<button
					onclick={handleRecalculate}
					disabled={actionLoading || summary.status === 'filed'}
					class="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50"
				>
					Prepocitat
				</button>
				<button
					onclick={handleGenerateXml}
					disabled={actionLoading || summary.status === 'filed'}
					class="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50"
				>
					Generovat XML
				</button>
				<button
					onclick={handleDownloadXml}
					disabled={!summary.has_xml}
					class="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50"
				>
					Stahnout XML
				</button>
				<button
					onclick={handleMarkFiled}
					disabled={actionLoading || summary.status === 'filed' || !summary.has_xml}
					class="rounded-lg bg-green-600 px-3 py-2 text-sm font-medium text-white hover:bg-green-700 transition-colors disabled:opacity-50"
				>
					Oznacit za podane
				</button>
				<button
					onclick={handleDelete}
					disabled={summary.status === 'filed'}
					class="rounded-lg border border-red-300 px-3 py-2 text-sm font-medium text-red-600 hover:bg-red-50 transition-colors disabled:opacity-50"
				>
					Smazat
				</button>
			</div>
		</div>

		<!-- Lines table -->
		<div class="mt-6">
			{#if !summary.lines || summary.lines.length === 0}
				<div class="rounded-xl border border-gray-200 bg-white p-8 text-center text-sm text-gray-500">
					Zadne radky v souhrnnem hlaseni
				</div>
			{:else}
				<div class="overflow-x-auto rounded-xl border border-gray-200 bg-white shadow-sm">
					<table class="w-full text-sm">
						<thead class="border-b border-gray-200 bg-gray-50">
							<tr>
								<th class="px-4 py-3 text-left font-medium text-gray-700">Kod zeme</th>
								<th class="px-4 py-3 text-left font-medium text-gray-700">DIC partnera</th>
								<th class="px-4 py-3 text-right font-medium text-gray-700">Celkova castka (CZK)</th>
								<th class="px-4 py-3 text-left font-medium text-gray-700">Kod plneni</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-gray-100">
							{#each summary.lines as line}
								<tr class="hover:bg-gray-50">
									<td class="px-4 py-3 text-gray-900">{line.country_code}</td>
									<td class="px-4 py-3 text-gray-900">{line.partner_dic}</td>
									<td class="px-4 py-3 text-right text-gray-900">{formatCZK(line.total_amount)}</td>
									<td class="px-4 py-3 text-gray-900">{line.service_code}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</div>

		<div class="mt-4 text-xs text-gray-400">
			Vytvoreno: {new Date(summary.created_at).toLocaleDateString('cs-CZ')} | Upraveno: {new Date(summary.updated_at).toLocaleDateString('cs-CZ')}
		</div>
	{/if}
</div>
