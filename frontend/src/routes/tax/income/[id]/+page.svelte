<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { incomeTaxApi, type IncomeTaxReturn, type TaxConstants } from '$lib/api/client';
	import { loadTaxConstants } from '$lib/data/tax-constants.svelte';
	import { formatCZK } from '$lib/utils/money';
	import Badge from '$lib/ui/Badge.svelte';
	import Button from '$lib/ui/Button.svelte';
	import ConfirmDialog from '$lib/ui/ConfirmDialog.svelte';
	import Card from '$lib/ui/Card.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import { toastSuccess } from '$lib/data/toast-state.svelte';

	let data = $state<IncomeTaxReturn | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let actionLoading = $state<string | null>(null);
	let taxConstants = $state<TaxConstants | null>(null);
	let showFileConfirm = $state(false);
	let showDeleteConfirm = $state(false);

	let returnId = $derived(Number(page.params.id));

	const statusLabels: Record<string, string> = {
		draft: 'Koncept',
		ready: 'Připraveno',
		filed: 'Podáno'
	};

	const filingTypeLabels: Record<string, string> = {
		regular: 'Řádné',
		corrective: 'Následné',
		supplementary: 'Opravné'
	};

	onMount(() => {
		loadData();
	});

	async function loadData() {
		loading = true;
		error = null;
		try {
			data = await incomeTaxApi.getById(returnId);
			if (data) {
				loadTaxConstants(data.year).then((tc) => {
					taxConstants = tc;
				});
			}
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
			data = await incomeTaxApi.recalculate(returnId);
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
			data = await incomeTaxApi.generateXml(returnId);
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
			const blob = await incomeTaxApi.downloadXml(returnId);
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `dpfo-${returnId}.xml`;
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

	function handleMarkFiled() {
		showFileConfirm = true;
	}

	async function confirmMarkFiled() {
		showFileConfirm = false;
		actionLoading = 'filed';
		error = null;
		try {
			data = await incomeTaxApi.markFiled(returnId);
			toastSuccess('Přiznání označeno jako podané');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se označit jako podané';
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
		error = null;
		try {
			await incomeTaxApi.delete(returnId);
			toastSuccess('Přiznání smazáno');
			goto('/tax');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se smazat přiznání';
		} finally {
			actionLoading = null;
		}
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
	<title>{data ? `Daň z příjmů ${data.year}` : 'Daň z příjmů'} - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<a href="/tax" class="text-sm text-secondary hover:text-primary">&larr; Zpět na daně</a>

	<ErrorAlert {error} class="mt-4" />

	{#if loading}
		<LoadingSpinner class="mt-8" />
	{:else if data}
		<!-- Header -->
		<div class="mt-4">
			<div class="flex items-center justify-between">
				<h1 class="text-xl font-semibold text-primary">Daň z příjmů - {data.year}</h1>
				<div class="flex items-center gap-3">
					<Badge variant={statusBadgeVariant(data.status)}>
						{statusLabels[data.status] ?? data.status}
					</Badge>
					<span class="text-sm text-secondary">
						{filingTypeLabels[data.filing_type] ?? data.filing_type}
					</span>
				</div>
			</div>
			<div class="mt-3 flex flex-wrap gap-2">
				<Button
					variant="secondary"
					onclick={handleRecalculate}
					disabled={data.status === 'filed' || actionLoading !== null}
				>
					{actionLoading === 'recalculate' ? 'Přepočítávám...' : 'Přepočítat'}
				</Button>
				<Button
					variant="secondary"
					onclick={handleGenerateXml}
					disabled={data.status === 'filed' || actionLoading !== null}
				>
					{actionLoading === 'generate' ? 'Generuji...' : 'Generovat XML'}
				</Button>
				{#if data.has_xml}
					<Button variant="secondary" onclick={handleDownloadXml} disabled={actionLoading !== null}>
						{actionLoading === 'download' ? 'Stahuji...' : 'Stáhnout XML'}
					</Button>
				{/if}
				{#if data.status !== 'filed'}
					<Button variant="success" onclick={handleMarkFiled} disabled={actionLoading !== null}>
						{actionLoading === 'filed' ? 'Označuji...' : 'Označit jako podané'}
					</Button>
					<Button variant="danger" onclick={handleDelete} disabled={actionLoading !== null}>
						Smazat
					</Button>
				{/if}
			</div>
		</div>

		<div class="mt-6 space-y-6">
			<!-- Section 7: Revenue and Expenses -->
			<Card>
				<h2 class="text-base font-semibold text-primary">Příjmy a výdaje (Oddíl 7)</h2>
				<div class="mt-4 grid grid-cols-2 gap-y-3 gap-x-4 text-sm">
					<dt class="text-secondary">Příjmy z podnikání</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.total_revenue)}
					</dd>

					<dt class="text-secondary">Skutečné výdaje</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.actual_expenses)}
					</dd>

					<dt class="text-secondary">
						Paušální výdaje <HelpTip topic="pausalni-vydaje" {taxConstants} />
					</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{data.flat_rate_percent}%
						{#if data.flat_rate_amount > 0}
							({formatCZK(data.flat_rate_amount)})
						{/if}
					</dd>

					<dt class="text-secondary">Použité výdaje</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.used_expenses)}
					</dd>
				</div>
			</Card>

			<!-- Tax Base -->
			<Card>
				<h2 class="text-base font-semibold text-primary">Základ daně</h2>
				<div class="mt-4 grid grid-cols-2 gap-y-3 gap-x-4 text-sm">
					<dt class="text-secondary">Základ daně</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.tax_base)}
					</dd>

					<dt class="text-secondary">Zaokrouhlený základ</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.tax_base_rounded)}
					</dd>
				</div>
			</Card>

			<!-- Tax Calculation -->
			<Card>
				<h2 class="text-base font-semibold text-primary">
					Výpočet daně <HelpTip topic="dan-15-23" {taxConstants} />
				</h2>
				<div class="mt-4 grid grid-cols-2 gap-y-3 gap-x-4 text-sm">
					<dt class="text-secondary">Daň 15%</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.tax_at_15)}
					</dd>

					<dt class="text-secondary">Daň 23%</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.tax_at_23)}
					</dd>

					<dt class="text-secondary">Celková daň</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.total_tax)}
					</dd>
				</div>
			</Card>

			<!-- Tax Credits -->
			<Card>
				<h2 class="text-base font-semibold text-primary">
					Slevy na dani <HelpTip topic="sleva-na-poplatnika" {taxConstants} />
				</h2>
				<div class="mt-4 grid grid-cols-2 gap-y-3 gap-x-4 text-sm">
					<dt class="text-secondary">Sleva na poplatníka</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.credit_basic)}
					</dd>

					{#if data.credit_spouse > 0}
						<dt class="text-secondary">Sleva na manžela/ku</dt>
						<dd class="text-right font-medium text-primary tabular-nums">
							{formatCZK(data.credit_spouse)}
						</dd>
					{/if}

					<dt class="text-secondary">Celkové slevy</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.total_credits)}
					</dd>

					<dt class="text-secondary">Daň po slevách</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.tax_after_credits)}
					</dd>
				</div>
			</Card>

			<!-- Child Benefit -->
			{#if data.child_benefit > 0}
				<Card>
					<h2 class="text-base font-semibold text-primary">
						Daňové zvýhodnění <HelpTip topic="zvyhodneni-na-deti" {taxConstants} />
					</h2>
					<div class="mt-4 grid grid-cols-2 gap-y-3 gap-x-4 text-sm">
						<dt class="text-secondary">Zvýhodnění na děti</dt>
						<dd class="text-right font-medium text-primary tabular-nums">
							{formatCZK(data.child_benefit)}
						</dd>

						<dt class="text-secondary">Daň po zvýhodnění</dt>
						<dd class="text-right font-medium text-primary tabular-nums">
							{formatCZK(data.tax_after_benefit)}
						</dd>
					</div>
				</Card>
			{/if}

			<!-- Result -->
			<Card>
				<h2 class="text-base font-semibold text-primary">Výsledek</h2>
				<div class="mt-4 grid grid-cols-2 gap-y-3 gap-x-4 text-sm">
					<dt class="text-secondary">Zaplacené zálohy</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.prepayments)}
					</dd>
				</div>
				<div class="mt-4 flex items-center justify-between border-t border-border pt-4">
					<span class="font-semibold text-primary">
						{data.tax_due >= 0 ? 'Doplatek:' : 'Přeplatek:'}
						<HelpTip topic="doplatek-preplatek" {taxConstants} />
					</span>
					<span
						class="text-lg font-semibold tabular-nums {data.tax_due >= 0
							? 'text-danger'
							: 'text-success'}"
					>
						{formatCZK(Math.abs(data.tax_due))}
					</span>
				</div>
			</Card>

			<!-- Timestamps -->
			<div class="text-xs text-muted">
				Vytvořeno: {new Date(data.created_at).toLocaleDateString('cs-CZ')} | Upraveno: {new Date(
					data.updated_at
				).toLocaleDateString('cs-CZ')}
				{#if data.filed_at}
					| Podáno: {new Date(data.filed_at).toLocaleDateString('cs-CZ')}{/if}
			</div>
		</div>
	{/if}
</div>

<ConfirmDialog
	bind:open={showFileConfirm}
	title="Označit jako podané"
	message="Opravdu chcete označit daňové přiznání jako podané?"
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
