<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { vatReturnApi, type VATReturn } from '$lib/api/vat';
	import { formatCZK } from '$lib/utils/money';
	import { vatStatusLabels, vatStatusColors, filingTypeLabels } from '$lib/utils/vat';

	let vatReturn = $state<VATReturn | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let actionLoading = $state<string | null>(null);

	let returnId = $derived(Number(page.params.id));

	onMount(() => {
		loadData();
	});

	async function loadData() {
		loading = true;
		error = null;
		try {
			vatReturn = await vatReturnApi.getById(returnId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst přiznání';
		} finally {
			loading = false;
		}
	}

	async function handleRecalculate() {
		actionLoading = 'recalculate';
		error = null;
		try {
			vatReturn = await vatReturnApi.recalculate(returnId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se přepočítat';
		} finally {
			actionLoading = null;
		}
	}

	async function handleGenerateXml() {
		actionLoading = 'generate';
		error = null;
		try {
			vatReturn = await vatReturnApi.generateXml(returnId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se generovat XML';
		} finally {
			actionLoading = null;
		}
	}

	async function handleDownloadXml() {
		actionLoading = 'download';
		error = null;
		try {
			const blob = await vatReturnApi.downloadXml(returnId);
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `dph-priznani-${returnId}.xml`;
			document.body.appendChild(a);
			a.click();
			document.body.removeChild(a);
			URL.revokeObjectURL(url);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se stáhnout XML';
		} finally {
			actionLoading = null;
		}
	}

	async function handleMarkFiled() {
		if (!confirm('Opravdu chcete označit DPH přiznání jako podané?')) return;
		actionLoading = 'filed';
		error = null;
		try {
			vatReturn = await vatReturnApi.markFiled(returnId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se označit jako podané';
		} finally {
			actionLoading = null;
		}
	}

	async function handleDelete() {
		if (!confirm('Opravdu chcete smazat toto přiznání?')) return;
		actionLoading = 'delete';
		error = null;
		try {
			await vatReturnApi.delete(returnId);
			goto('/vat');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se smazat přiznání';
		} finally {
			actionLoading = null;
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
</script>

<svelte:head>
	<title>{vatReturn ? `DPH ${formatPeriod(vatReturn)}` : 'DPH přiznání'} - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-4xl">
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
	{:else if vatReturn}
		<!-- Header -->
		<div class="mt-4 flex items-start justify-between">
			<div>
				<h1 class="text-2xl font-bold text-gray-900">DPH přiznání - {formatPeriod(vatReturn)}</h1>
				<div class="mt-2 flex items-center gap-3">
					<span
						class="inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium {vatStatusColors[
							vatReturn.status
						] ?? 'bg-gray-100 text-gray-700'}"
					>
						{vatStatusLabels[vatReturn.status] ?? vatReturn.status}
					</span>
					<span class="text-sm text-gray-600">
						{filingTypeLabels[vatReturn.filing_type] ?? vatReturn.filing_type}
					</span>
				</div>
			</div>
			<div class="flex flex-wrap gap-2">
				<button
					onclick={handleRecalculate}
					disabled={vatReturn.status === 'filed' || actionLoading !== null}
					class="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
				>
					{actionLoading === 'recalculate' ? 'Přepočítávám...' : 'Přepočítat'}
				</button>
				<button
					onclick={handleGenerateXml}
					disabled={vatReturn.status === 'filed' || actionLoading !== null}
					class="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
				>
					{actionLoading === 'generate' ? 'Generuji...' : 'Generovat XML'}
				</button>
				{#if vatReturn.has_xml}
					<button
						onclick={handleDownloadXml}
						disabled={actionLoading !== null}
						class="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
					>
						{actionLoading === 'download' ? 'Stahuji...' : 'Stáhnout XML'}
					</button>
				{/if}
				{#if vatReturn.status !== 'filed'}
					<button
						onclick={handleMarkFiled}
						disabled={actionLoading !== null}
						class="rounded-lg bg-green-600 px-3 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
					>
						{actionLoading === 'filed' ? 'Označuji...' : 'Označit za podané'}
					</button>
					<button
						onclick={handleDelete}
						disabled={actionLoading !== null}
						class="rounded-lg border border-red-300 px-3 py-2 text-sm font-medium text-red-600 hover:bg-red-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
					>
						Smazat
					</button>
				{/if}
			</div>
		</div>

		<!-- Output VAT -->
		<div class="mt-6 space-y-6">
			<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
				<h2 class="text-lg font-semibold text-gray-900">Výstupní DPH</h2>
				<div class="mt-4 overflow-x-auto">
					<table class="w-full text-left text-sm">
						<thead class="border-b border-gray-200">
							<tr>
								<th class="pb-2 font-medium text-gray-600">Sazba</th>
								<th class="pb-2 text-right font-medium text-gray-600">Základ daně</th>
								<th class="pb-2 text-right font-medium text-gray-600">Daň</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-gray-100">
							<tr>
								<td class="py-2 text-gray-900">21 %</td>
								<td class="py-2 text-right text-gray-700"
									>{formatCZK(vatReturn.output_vat_base_21)}</td
								>
								<td class="py-2 text-right text-gray-700"
									>{formatCZK(vatReturn.output_vat_amount_21)}</td
								>
							</tr>
							<tr>
								<td class="py-2 text-gray-900">12 %</td>
								<td class="py-2 text-right text-gray-700"
									>{formatCZK(vatReturn.output_vat_base_12)}</td
								>
								<td class="py-2 text-right text-gray-700"
									>{formatCZK(vatReturn.output_vat_amount_12)}</td
								>
							</tr>
							<tr>
								<td class="py-2 text-gray-900">0 % (osvobozeno)</td>
								<td class="py-2 text-right text-gray-700"
									>{formatCZK(vatReturn.output_vat_base_0)}</td
								>
								<td class="py-2 text-right text-gray-400">-</td>
							</tr>
						</tbody>
						<tfoot class="border-t border-gray-300">
							<tr>
								<td class="py-2 font-semibold text-gray-900">Celkem výstupní DPH</td>
								<td></td>
								<td class="py-2 text-right font-semibold text-gray-900"
									>{formatCZK(vatReturn.total_output_vat)}</td
								>
							</tr>
						</tfoot>
					</table>
				</div>
			</div>

			<!-- Reverse charge -->
			{#if vatReturn.reverse_charge_base_21 > 0 || vatReturn.reverse_charge_base_12 > 0}
				<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
					<h2 class="text-lg font-semibold text-gray-900">Přenesení daňové povinnosti</h2>
					<div class="mt-4 overflow-x-auto">
						<table class="w-full text-left text-sm">
							<thead class="border-b border-gray-200">
								<tr>
									<th class="pb-2 font-medium text-gray-600">Sazba</th>
									<th class="pb-2 text-right font-medium text-gray-600">Základ daně</th>
									<th class="pb-2 text-right font-medium text-gray-600">Daň</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-gray-100">
								<tr>
									<td class="py-2 text-gray-900">21 %</td>
									<td class="py-2 text-right text-gray-700"
										>{formatCZK(vatReturn.reverse_charge_base_21)}</td
									>
									<td class="py-2 text-right text-gray-700"
										>{formatCZK(vatReturn.reverse_charge_amount_21)}</td
									>
								</tr>
								<tr>
									<td class="py-2 text-gray-900">12 %</td>
									<td class="py-2 text-right text-gray-700"
										>{formatCZK(vatReturn.reverse_charge_base_12)}</td
									>
									<td class="py-2 text-right text-gray-700"
										>{formatCZK(vatReturn.reverse_charge_amount_12)}</td
									>
								</tr>
							</tbody>
						</table>
					</div>
				</div>
			{/if}

			<!-- Input VAT -->
			<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
				<h2 class="text-lg font-semibold text-gray-900">Vstupní DPH</h2>
				<div class="mt-4 overflow-x-auto">
					<table class="w-full text-left text-sm">
						<thead class="border-b border-gray-200">
							<tr>
								<th class="pb-2 font-medium text-gray-600">Sazba</th>
								<th class="pb-2 text-right font-medium text-gray-600">Základ daně</th>
								<th class="pb-2 text-right font-medium text-gray-600">Daň</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-gray-100">
							<tr>
								<td class="py-2 text-gray-900">21 %</td>
								<td class="py-2 text-right text-gray-700"
									>{formatCZK(vatReturn.input_vat_base_21)}</td
								>
								<td class="py-2 text-right text-gray-700"
									>{formatCZK(vatReturn.input_vat_amount_21)}</td
								>
							</tr>
							<tr>
								<td class="py-2 text-gray-900">12 %</td>
								<td class="py-2 text-right text-gray-700"
									>{formatCZK(vatReturn.input_vat_base_12)}</td
								>
								<td class="py-2 text-right text-gray-700"
									>{formatCZK(vatReturn.input_vat_amount_12)}</td
								>
							</tr>
						</tbody>
						<tfoot class="border-t border-gray-300">
							<tr>
								<td class="py-2 font-semibold text-gray-900">Celkem vstupní DPH</td>
								<td></td>
								<td class="py-2 text-right font-semibold text-gray-900"
									>{formatCZK(vatReturn.total_input_vat)}</td
								>
							</tr>
						</tfoot>
					</table>
				</div>
			</div>

			<!-- Result -->
			<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
				<h2 class="text-lg font-semibold text-gray-900">Výsledek</h2>
				<div class="mt-4 flex flex-col items-end gap-2 text-sm">
					<div class="flex gap-8">
						<span class="text-gray-600">Výstupní DPH:</span>
						<span class="font-medium text-gray-900">{formatCZK(vatReturn.total_output_vat)}</span>
					</div>
					<div class="flex gap-8">
						<span class="text-gray-600">Vstupní DPH:</span>
						<span class="font-medium text-gray-900">{formatCZK(vatReturn.total_input_vat)}</span>
					</div>
					<div class="flex gap-8 border-t border-gray-200 pt-2 text-base">
						<span class="font-semibold text-gray-900"
							>{vatReturn.net_vat >= 0 ? 'Vlastní daňová povinnost:' : 'Nadměrný odpočet:'}</span
						>
						<span class="font-bold {vatReturn.net_vat >= 0 ? 'text-red-600' : 'text-green-600'}">
							{formatCZK(Math.abs(vatReturn.net_vat))}
						</span>
					</div>
				</div>
			</div>

			<!-- Timestamps -->
			<div class="text-xs text-gray-400">
				Vytvořeno: {new Date(vatReturn.created_at).toLocaleDateString('cs-CZ')} | Upraveno: {new Date(
					vatReturn.updated_at
				).toLocaleDateString('cs-CZ')}
				{#if vatReturn.filed_at}
					| Podáno: {new Date(vatReturn.filed_at).toLocaleDateString('cs-CZ')}{/if}
			</div>
		</div>
	{/if}
</div>
