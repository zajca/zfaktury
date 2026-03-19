<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { vatReturnApi, type VATReturn } from '$lib/api/client';
	import { downloadFile } from '$lib/utils/download';
	import { formatCZK } from '$lib/utils/money';
	import { vatStatusLabels, filingTypeLabels } from '$lib/utils/vat';
	import Badge from '$lib/ui/Badge.svelte';
	import Button from '$lib/ui/Button.svelte';
	import ConfirmDialog from '$lib/ui/ConfirmDialog.svelte';
	import Card from '$lib/ui/Card.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import { toastSuccess, toastError } from '$lib/data/toast-state.svelte';

	let vatReturn = $state<VATReturn | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let actionLoading = $state<string | null>(null);
	let showFileConfirm = $state(false);
	let showDeleteConfirm = $state(false);

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
		try {
			vatReturn = await vatReturnApi.recalculate(returnId);
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se přepočítat');
		} finally {
			actionLoading = null;
		}
	}

	async function handleGenerateXml() {
		actionLoading = 'generate';
		try {
			vatReturn = await vatReturnApi.generateXml(returnId);
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se generovat XML');
		} finally {
			actionLoading = null;
		}
	}

	async function handleDownloadXml() {
		actionLoading = 'download';
		try {
			await downloadFile(`/api/v1/vat-returns/${returnId}/xml`, `dph-priznani-${returnId}.xml`);
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se stáhnout XML');
		} finally {
			actionLoading = null;
		}
	}

	function handleMarkFiled() {
		showFileConfirm = true;
	}

	async function confirmMarkFiled() {
		showFileConfirm = false;
		actionLoading = 'filed';
		try {
			vatReturn = await vatReturnApi.markFiled(returnId);
			toastSuccess('Přiznání označeno jako podané');
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se označit jako podané');
		} finally {
			actionLoading = null;
		}
	}

	function handleDelete() {
		showDeleteConfirm = true;
	}

	async function confirmDelete() {
		showDeleteConfirm = false;
		actionLoading = 'delete';
		try {
			await vatReturnApi.delete(returnId);
			toastSuccess('Přiznání smazáno');
			goto('/vat');
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se smazat přiznání');
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

	function statusBadgeVariant(
		status: string
	): 'default' | 'success' | 'danger' | 'warning' | 'info' | 'muted' {
		switch (status) {
			case 'filed':
				return 'success';
			case 'ready':
				return 'info';
			default:
				return 'default';
		}
	}
</script>

<svelte:head>
	<title>{vatReturn ? `DPH ${formatPeriod(vatReturn)}` : 'DPH přiznání'} - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<a href="/vat" class="text-sm text-secondary hover:text-primary">&larr; Zpět na DPH</a>

	<ErrorAlert {error} class="mt-4" />

	{#if loading}
		<LoadingSpinner class="mt-8" />
	{:else if vatReturn}
		<!-- Header -->
		<div class="mt-4">
			<div class="flex items-center justify-between">
				<h1 class="text-xl font-semibold text-primary">DPH přiznání - {formatPeriod(vatReturn)}</h1>
				<div class="flex items-center gap-3">
					<Badge variant={statusBadgeVariant(vatReturn.status)}>
						{vatStatusLabels[vatReturn.status] ?? vatReturn.status}
					</Badge>
					<span class="text-sm text-secondary">
						{filingTypeLabels[vatReturn.filing_type] ?? vatReturn.filing_type}
					</span>
				</div>
			</div>
			<div class="mt-3 flex flex-wrap gap-2">
				<Button
					variant="secondary"
					onclick={handleRecalculate}
					disabled={vatReturn.status === 'filed' || actionLoading !== null}
				>
					{actionLoading === 'recalculate' ? 'Přepočítávám...' : 'Přepočítat'}
				</Button>
				<Button
					variant="secondary"
					onclick={handleGenerateXml}
					disabled={vatReturn.status === 'filed' || actionLoading !== null}
				>
					{actionLoading === 'generate' ? 'Generuji...' : 'Generovat XML'}
				</Button>
				{#if vatReturn.has_xml}
					<Button variant="secondary" onclick={handleDownloadXml} disabled={actionLoading !== null}>
						{actionLoading === 'download' ? 'Stahuji...' : 'Stáhnout XML'}
					</Button>
				{/if}
				{#if vatReturn.status !== 'filed'}
					<Button variant="success" onclick={handleMarkFiled} disabled={actionLoading !== null}>
						{actionLoading === 'filed' ? 'Označuji...' : 'Označit za podané'}
					</Button>
					<Button variant="danger" onclick={handleDelete} disabled={actionLoading !== null}>
						Smazat
					</Button>
				{/if}
			</div>
		</div>

		<!-- Output VAT -->
		<div class="mt-6 space-y-6">
			<Card padding={false}>
				<div class="p-5 pb-0">
					<h2 class="text-base font-semibold text-primary">
						Výstupní DPH <HelpTip topic="vystupni-dph" />
					</h2>
				</div>
				<div class="mt-4 overflow-x-auto">
					<table class="w-full text-left text-sm">
						<thead class="border-b border-border">
							<tr class="bg-elevated">
								<th class="px-5 py-2.5 text-xs font-medium uppercase tracking-wider text-muted"
									>Sazba <HelpTip topic="sazba-dph" /></th
								>
								<th
									class="px-5 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted"
									>Základ daně <HelpTip topic="zaklad-dane" /></th
								>
								<th
									class="px-5 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted"
									>Daň</th
								>
							</tr>
						</thead>
						<tbody class="divide-y divide-border-subtle">
							<tr>
								<td class="px-5 py-2.5 text-primary">21 %</td>
								<td class="px-5 py-2.5 text-right font-mono tabular-nums text-secondary"
									>{formatCZK(vatReturn.output_vat_base_21)}</td
								>
								<td class="px-5 py-2.5 text-right font-mono tabular-nums text-secondary"
									>{formatCZK(vatReturn.output_vat_amount_21)}</td
								>
							</tr>
							<tr>
								<td class="px-5 py-2.5 text-primary">12 %</td>
								<td class="px-5 py-2.5 text-right font-mono tabular-nums text-secondary"
									>{formatCZK(vatReturn.output_vat_base_12)}</td
								>
								<td class="px-5 py-2.5 text-right font-mono tabular-nums text-secondary"
									>{formatCZK(vatReturn.output_vat_amount_12)}</td
								>
							</tr>
							<tr>
								<td class="px-5 py-2.5 text-primary">0 % (osvobozeno)</td>
								<td class="px-5 py-2.5 text-right font-mono tabular-nums text-secondary"
									>{formatCZK(vatReturn.output_vat_base_0)}</td
								>
								<td class="px-5 py-2.5 text-right text-muted">-</td>
							</tr>
						</tbody>
						<tfoot class="border-t border-border">
							<tr>
								<td class="px-5 py-2.5 font-medium text-primary">Celkem výstupní DPH</td>
								<td></td>
								<td class="px-5 py-2.5 text-right font-mono tabular-nums font-medium text-primary"
									>{formatCZK(vatReturn.total_output_vat)}</td
								>
							</tr>
						</tfoot>
					</table>
				</div>
			</Card>

			<!-- Reverse charge -->
			{#if vatReturn.reverse_charge_base_21 > 0 || vatReturn.reverse_charge_base_12 > 0}
				<Card padding={false}>
					<div class="p-5 pb-0">
						<h2 class="text-base font-semibold text-primary">
							Přenesení daňové povinnosti <HelpTip topic="preneseni-danove-povinnosti" />
						</h2>
					</div>
					<div class="mt-4 overflow-x-auto">
						<table class="w-full text-left text-sm">
							<thead class="border-b border-border">
								<tr class="bg-elevated">
									<th class="px-5 py-2.5 text-xs font-medium uppercase tracking-wider text-muted"
										>Sazba</th
									>
									<th
										class="px-5 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted"
										>Základ daně</th
									>
									<th
										class="px-5 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted"
										>Daň</th
									>
								</tr>
							</thead>
							<tbody class="divide-y divide-border-subtle">
								<tr>
									<td class="px-5 py-2.5 text-primary">21 %</td>
									<td class="px-5 py-2.5 text-right font-mono tabular-nums text-secondary"
										>{formatCZK(vatReturn.reverse_charge_base_21)}</td
									>
									<td class="px-5 py-2.5 text-right font-mono tabular-nums text-secondary"
										>{formatCZK(vatReturn.reverse_charge_amount_21)}</td
									>
								</tr>
								<tr>
									<td class="px-5 py-2.5 text-primary">12 %</td>
									<td class="px-5 py-2.5 text-right font-mono tabular-nums text-secondary"
										>{formatCZK(vatReturn.reverse_charge_base_12)}</td
									>
									<td class="px-5 py-2.5 text-right font-mono tabular-nums text-secondary"
										>{formatCZK(vatReturn.reverse_charge_amount_12)}</td
									>
								</tr>
							</tbody>
						</table>
					</div>
				</Card>
			{/if}

			<!-- Input VAT -->
			<Card padding={false}>
				<div class="p-5 pb-0">
					<h2 class="text-base font-semibold text-primary">
						Vstupní DPH <HelpTip topic="vstupni-dph" />
					</h2>
				</div>
				<div class="mt-4 overflow-x-auto">
					<table class="w-full text-left text-sm">
						<thead class="border-b border-border">
							<tr class="bg-elevated">
								<th class="px-5 py-2.5 text-xs font-medium uppercase tracking-wider text-muted"
									>Sazba</th
								>
								<th
									class="px-5 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted"
									>Základ daně</th
								>
								<th
									class="px-5 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted"
									>Daň</th
								>
							</tr>
						</thead>
						<tbody class="divide-y divide-border-subtle">
							<tr>
								<td class="px-5 py-2.5 text-primary">21 %</td>
								<td class="px-5 py-2.5 text-right font-mono tabular-nums text-secondary"
									>{formatCZK(vatReturn.input_vat_base_21)}</td
								>
								<td class="px-5 py-2.5 text-right font-mono tabular-nums text-secondary"
									>{formatCZK(vatReturn.input_vat_amount_21)}</td
								>
							</tr>
							<tr>
								<td class="px-5 py-2.5 text-primary">12 %</td>
								<td class="px-5 py-2.5 text-right font-mono tabular-nums text-secondary"
									>{formatCZK(vatReturn.input_vat_base_12)}</td
								>
								<td class="px-5 py-2.5 text-right font-mono tabular-nums text-secondary"
									>{formatCZK(vatReturn.input_vat_amount_12)}</td
								>
							</tr>
						</tbody>
						<tfoot class="border-t border-border">
							<tr>
								<td class="px-5 py-2.5 font-medium text-primary">Celkem vstupní DPH</td>
								<td></td>
								<td class="px-5 py-2.5 text-right font-mono tabular-nums font-medium text-primary"
									>{formatCZK(vatReturn.total_input_vat)}</td
								>
							</tr>
						</tfoot>
					</table>
				</div>
			</Card>

			<!-- Result -->
			<Card>
				<h2 class="text-base font-semibold text-primary">
					Výsledek <HelpTip topic="nadmerny-odpocet" />
				</h2>
				<div class="mt-4 flex flex-col items-end gap-2 text-sm">
					<div class="flex gap-8">
						<span class="text-secondary">Výstupní DPH:</span>
						<span class="font-mono tabular-nums font-medium text-primary"
							>{formatCZK(vatReturn.total_output_vat)}</span
						>
					</div>
					<div class="flex gap-8">
						<span class="text-secondary">Vstupní DPH:</span>
						<span class="font-mono tabular-nums font-medium text-primary"
							>{formatCZK(vatReturn.total_input_vat)}</span
						>
					</div>
					<div class="flex gap-8 border-t border-border pt-2 text-base">
						<span class="font-semibold text-primary"
							>{vatReturn.net_vat >= 0 ? 'Vlastní daňová povinnost:' : 'Nadměrný odpočet:'}</span
						>
						<span
							class="font-mono tabular-nums font-semibold {vatReturn.net_vat >= 0
								? 'text-danger'
								: 'text-success'}"
						>
							{formatCZK(Math.abs(vatReturn.net_vat))}
						</span>
					</div>
				</div>
			</Card>

			<!-- Timestamps -->
			<div class="text-xs text-muted">
				Vytvořeno: {new Date(vatReturn.created_at).toLocaleDateString('cs-CZ')} | Upraveno: {new Date(
					vatReturn.updated_at
				).toLocaleDateString('cs-CZ')}
				{#if vatReturn.filed_at}
					| Podáno: {new Date(vatReturn.filed_at).toLocaleDateString('cs-CZ')}{/if}
			</div>
		</div>
	{/if}
</div>

<ConfirmDialog
	bind:open={showFileConfirm}
	title="Označit jako podané"
	message="Opravdu chcete označit DPH přiznání jako podané?"
	confirmLabel="Označit jako podané"
	variant="warning"
	onconfirm={confirmMarkFiled}
	oncancel={() => (showFileConfirm = false)}
/>

<ConfirmDialog
	bind:open={showDeleteConfirm}
	title="Smazat přiznání"
	message="Opravdu chcete smazat toto přiznání?"
	confirmLabel="Smazat"
	onconfirm={confirmDelete}
	oncancel={() => (showDeleteConfirm = false)}
/>
