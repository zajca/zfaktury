<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import {
		controlStatementApi,
		type ControlStatement,
		type ControlStatementLine
	} from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import { vatStatusLabels, filingTypeLabels } from '$lib/utils/vat';
	import Badge from '$lib/ui/Badge.svelte';
	import Button from '$lib/ui/Button.svelte';
	import ConfirmDialog from '$lib/ui/ConfirmDialog.svelte';
	import Card from '$lib/ui/Card.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import { toastSuccess } from '$lib/data/toast-state.svelte';

	let statement = $state<ControlStatement | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let actionLoading = $state(false);
	let showFileConfirm = $state(false);
	let showDeleteConfirm = $state(false);
	let activeTab = $state<string>('A4');

	let statementId = $derived(Number(page.params.id));

	const tabs = ['A4', 'A5', 'B2', 'B3'];
	const tabLabels: Record<string, string> = {
		A4: 'A4 - Výstup nad 10 000',
		A5: 'A5 - Výstup do 10 000',
		B2: 'B2 - Vstup nad 10 000',
		B3: 'B3 - Vstup do 10 000'
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
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst kontrolní hlášení';
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
			error = e instanceof Error ? e.message : 'Nepodařilo se přepočítat';
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
			error = e instanceof Error ? e.message : 'Nepodařilo se generovat XML';
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
			document.body.appendChild(a);
			a.click();
			document.body.removeChild(a);
			URL.revokeObjectURL(url);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se stáhnout XML';
		}
	}

	function handleMarkFiled() {
		showFileConfirm = true;
	}

	async function confirmMarkFiled() {
		showFileConfirm = false;
		actionLoading = true;
		error = null;
		try {
			statement = await controlStatementApi.markFiled(statementId);
			toastSuccess('Kontrolní hlášení označeno jako podané');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se označit jako podané';
		} finally {
			actionLoading = false;
		}
	}

	function handleDelete() {
		showDeleteConfirm = true;
	}

	async function confirmDelete() {
		showDeleteConfirm = false;
		error = null;
		try {
			await controlStatementApi.delete(statementId);
			toastSuccess('Kontrolní hlášení smazáno');
			goto('/vat');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se smazat kontrolní hlášení';
		}
	}

	function formatAmountCZK(halere: number): string {
		return formatCZK(halere);
	}

	function statusBadgeVariant(status: string): 'default' | 'success' | 'danger' | 'warning' | 'info' | 'muted' {
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
	<title
		>{statement
			? `Kontrolní hlášení ${statement.period.year}/${statement.period.month}`
			: 'Kontrolní hlášení'} - ZFaktury</title
	>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<a href="/vat" class="text-sm text-secondary hover:text-primary">&larr; Zpět na DPH</a>

	<ErrorAlert {error} class="mt-4" />

	{#if loading}
		<LoadingSpinner class="mt-8" />
	{:else if statement}
		<!-- Header -->
		<div class="mt-4">
			<div class="flex items-center justify-between">
				<h1 class="text-xl font-semibold text-primary">
					Kontrolní hlášení <HelpTip topic="kontrolni-hlaseni" /> {statement.period.year}/{String(statement.period.month).padStart(
						2,
						'0'
					)}
				</h1>
				<div class="flex items-center gap-3">
					<Badge variant={statusBadgeVariant(statement.status)}>
						{vatStatusLabels[statement.status] || statement.status}
					</Badge>
					<span class="text-sm text-tertiary">
						{filingTypeLabels[statement.filing_type] || statement.filing_type}
					</span>
					{#if statement.filed_at}
						<span class="text-sm text-tertiary">
							Podáno: {new Date(statement.filed_at).toLocaleDateString('cs-CZ')}
						</span>
					{/if}
				</div>
			</div>
			<div class="mt-3 flex flex-wrap gap-2">
				<Button
					variant="secondary"
					onclick={handleRecalculate}
					disabled={actionLoading || statement.status === 'filed'}
				>
					Přepočítat
				</Button>
				<Button
					variant="secondary"
					onclick={handleGenerateXml}
					disabled={actionLoading || statement.status === 'filed'}
				>
					Generovat XML
				</Button>
				<Button
					variant="secondary"
					onclick={handleDownloadXml}
					disabled={!statement.has_xml}
				>
					Stáhnout XML
				</Button>
				<Button
					variant="success"
					onclick={handleMarkFiled}
					disabled={actionLoading || statement.status === 'filed' || !statement.has_xml}
				>
					Označit za podané
				</Button>
				<Button
					variant="danger"
					onclick={handleDelete}
					disabled={statement.status === 'filed'}
				>
					Smazat
				</Button>
			</div>
		</div>

		<!-- Tabs -->
		<div class="mt-6 border-b border-border">
			<nav class="-mb-px flex gap-4">
				{#each tabs as tab (tab)}
					<button
						onclick={() => {
							activeTab = tab;
						}}
						class="whitespace-nowrap border-b-2 px-1 py-3 text-sm font-medium transition-colors {activeTab ===
						tab
							? 'border-accent text-accent-text'
							: 'border-transparent text-tertiary hover:text-secondary hover:border-border-strong'}"
					>
						{tabLabels[tab]} <HelpTip topic="sekce-kontrolni-hlaseni" />
					</button>
				{/each}
			</nav>
		</div>

		<!-- Lines table -->
		<div class="mt-4">
			{#if filteredLines.length === 0}
				<Card>
					<div class="py-4 text-center text-sm text-muted">
						Žádné řádky v sekci {activeTab}
					</div>
				</Card>
			{:else if isDetailSection}
				<!-- A4/B2: detailed lines with partner info -->
				<Card padding={false} class="overflow-x-auto">
					<table class="w-full text-sm">
						<thead class="border-b border-border bg-elevated">
							<tr>
								<th class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted">DIC partnera <HelpTip topic="dic" /></th>
								<th class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted">Číslo dokladu</th>
								<th class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted">DPPD <HelpTip topic="dppd" /></th>
								<th class="px-4 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted">Základ (CZK)</th>
								<th class="px-4 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted">DPH (CZK)</th>
								<th class="px-4 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted">Sazba <HelpTip topic="sazba-dph" /></th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border-subtle">
							{#each filteredLines as line, i (i)}
								<tr class="hover:bg-hover transition-colors">
									<td class="px-4 py-2.5 font-mono text-xs text-primary">{line.partner_dic}</td>
									<td class="px-4 py-2.5 text-primary">{line.document_number}</td>
									<td class="px-4 py-2.5 text-primary">{line.dppd}</td>
									<td class="px-4 py-2.5 text-right font-mono tabular-nums text-secondary">{formatAmountCZK(line.base)}</td>
									<td class="px-4 py-2.5 text-right font-mono tabular-nums text-secondary">{formatAmountCZK(line.vat)}</td>
									<td class="px-4 py-2.5 text-right font-mono tabular-nums text-secondary">{line.vat_rate_percent}%</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</Card>
			{:else}
				<!-- A5/B3: aggregated lines without partner info -->
				<Card padding={false} class="overflow-x-auto">
					<table class="w-full text-sm">
						<thead class="border-b border-border bg-elevated">
							<tr>
								<th class="px-4 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted">Základ (CZK)</th>
								<th class="px-4 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted">DPH (CZK)</th>
								<th class="px-4 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted">Sazba <HelpTip topic="sazba-dph" /></th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border-subtle">
							{#each filteredLines as line, i (i)}
								<tr class="hover:bg-hover transition-colors">
									<td class="px-4 py-2.5 text-right font-mono tabular-nums text-secondary">{formatAmountCZK(line.base)}</td>
									<td class="px-4 py-2.5 text-right font-mono tabular-nums text-secondary">{formatAmountCZK(line.vat)}</td>
									<td class="px-4 py-2.5 text-right font-mono tabular-nums text-secondary">{line.vat_rate_percent}%</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</Card>
			{/if}
		</div>

		<div class="mt-4 text-xs text-muted">
			Vytvořeno: {new Date(statement.created_at).toLocaleDateString('cs-CZ')} | Upraveno: {new Date(
				statement.updated_at
			).toLocaleDateString('cs-CZ')}
		</div>
	{/if}
</div>

<ConfirmDialog
	bind:open={showFileConfirm}
	title="Označit jako podané"
	message="Opravdu chcete označit kontrolní hlášení jako podané?"
	confirmLabel="Označit jako podané"
	variant="warning"
	onconfirm={confirmMarkFiled}
	oncancel={() => showFileConfirm = false}
/>

<ConfirmDialog
	bind:open={showDeleteConfirm}
	title="Smazat kontrolní hlášení"
	message="Opravdu chcete smazat toto kontrolní hlášení?"
	confirmLabel="Smazat"
	onconfirm={confirmDelete}
	oncancel={() => showDeleteConfirm = false}
/>
