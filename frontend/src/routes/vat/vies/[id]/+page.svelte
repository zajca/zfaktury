<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { viesApi, type VIESSummary } from '$lib/api/vat-vies';
	import { formatCZK } from '$lib/utils/money';
	import {
		vatStatusLabels,
		vatStatusColors,
		filingTypeLabels,
		quarterLabels
	} from '$lib/utils/vat';

	let summary = $state<VIESSummary | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let actionLoading = $state(false);

	let summaryId = $derived(Number(page.params.id));

	onMount(() => {
		loadSummary();
	});

	async function loadSummary() {
		loading = true;
		error = null;
		try {
			summary = await viesApi.getById(summaryId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst souhrnné hlášení';
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
			error = e instanceof Error ? e.message : 'Nepodařilo se přepočítat';
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
			error = e instanceof Error ? e.message : 'Nepodařilo se generovat XML';
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
			error = e instanceof Error ? e.message : 'Nepodařilo se stáhnout XML';
		}
	}

	async function handleMarkFiled() {
		if (!confirm('Opravdu chcete označit souhrnné hlášení jako podané?')) return;
		actionLoading = true;
		error = null;
		try {
			summary = await viesApi.markFiled(summaryId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se označit jako podané';
		} finally {
			actionLoading = false;
		}
	}

	async function handleDelete() {
		if (!confirm('Opravdu chcete smazat toto souhrnné hlášení?')) return;
		error = null;
		try {
			await viesApi.delete(summaryId);
			goto('/vat');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se smazat souhrnné hlášení';
		}
	}
</script>

<svelte:head>
	<title
		>{summary
			? `Souhrnné hlášení ${summary.period.year} Q${summary.period.quarter}`
			: 'Souhrnné hlášení'} - ZFaktury</title
	>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<a href="/vat" class="text-sm text-blue-600 hover:text-blue-800">&larr; Zpět na DPH</a>

	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700"
		>
			{error}
		</div>
	{/if}

	{#if loading}
		<div class="mt-8 flex items-center justify-center">
			<div role="status">
				<div
					class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"
				></div>
				<span class="sr-only">Načítání...</span>
			</div>
		</div>
	{:else if summary}
		<!-- Header -->
		<div class="mt-4 flex items-start justify-between">
			<div>
				<h1 class="text-2xl font-bold text-gray-900">
					Souhrnné hlášení {summary.period.year} Q{summary.period.quarter}
				</h1>
				<div class="mt-2 flex items-center gap-3">
					<span
						class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium {vatStatusColors[
							summary.status
						] || 'bg-gray-100 text-gray-700'}"
					>
						{vatStatusLabels[summary.status] || summary.status}
					</span>
					<span class="text-sm text-gray-500">
						{filingTypeLabels[summary.filing_type] || summary.filing_type}
					</span>
					<span class="text-sm text-gray-500">
						{quarterLabels[summary.period.quarter] || `Q${summary.period.quarter}`}
					</span>
					{#if summary.filed_at}
						<span class="text-sm text-gray-500">
							Podáno: {new Date(summary.filed_at).toLocaleDateString('cs-CZ')}
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
					Přepočítat
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
					Stáhnout XML
				</button>
				<button
					onclick={handleMarkFiled}
					disabled={actionLoading || summary.status === 'filed' || !summary.has_xml}
					class="rounded-lg bg-green-600 px-3 py-2 text-sm font-medium text-white hover:bg-green-700 transition-colors disabled:opacity-50"
				>
					Označit za podané
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
				<div
					class="rounded-xl border border-gray-200 bg-white p-8 text-center text-sm text-gray-500"
				>
					Žádné řádky v souhrnném hlášení
				</div>
			{:else}
				<div class="overflow-x-auto rounded-xl border border-gray-200 bg-white shadow-sm">
					<table class="w-full text-sm">
						<thead class="border-b border-gray-200 bg-gray-50">
							<tr>
								<th class="px-4 py-3 text-left font-medium text-gray-700">Kód země</th>
								<th class="px-4 py-3 text-left font-medium text-gray-700">DIC partnera</th>
								<th class="px-4 py-3 text-right font-medium text-gray-700">Celková částka (CZK)</th>
								<th class="px-4 py-3 text-left font-medium text-gray-700">Kód plnění</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-gray-100">
							{#each summary.lines as line, i (i)}
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
			Vytvořeno: {new Date(summary.created_at).toLocaleDateString('cs-CZ')} | Upraveno: {new Date(
				summary.updated_at
			).toLocaleDateString('cs-CZ')}
		</div>
	{/if}
</div>
