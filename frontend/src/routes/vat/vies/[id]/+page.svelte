<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { viesApi, type VIESSummary } from '$lib/api/vat-vies';
	import { formatCZK } from '$lib/utils/money';
	import {
		vatStatusLabels,
		filingTypeLabels,
		quarterLabels
	} from '$lib/utils/vat';
	import Button from '$lib/ui/Button.svelte';
	import Badge from '$lib/ui/Badge.svelte';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';

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
		>{summary
			? `Souhrnné hlášení ${summary.period.year} Q${summary.period.quarter}`
			: 'Souhrnné hlášení'} - ZFaktury</title
	>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<a href="/vat" class="text-sm text-secondary hover:text-primary">&larr; Zpět na DPH</a>

	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-danger/20 bg-danger-bg p-4 text-sm text-danger"
		>
			{error}
		</div>
	{/if}

	{#if loading}
		<div class="mt-8 flex items-center justify-center">
			<div role="status">
				<div
					class="h-8 w-8 animate-spin rounded-full border-4 border-border border-t-accent"
				></div>
				<span class="sr-only">Načítání...</span>
			</div>
		</div>
	{:else if summary}
		<!-- Header -->
		<div class="mt-4">
			<div class="flex items-center justify-between">
				<h1 class="text-xl font-semibold text-primary">
					Souhrnné hlášení <HelpTip topic="souhrnne-hlaseni" /> {summary.period.year} Q{summary.period.quarter}
				</h1>
				<div class="flex items-center gap-3">
					<Badge variant={statusBadgeVariant(summary.status)}>
						{vatStatusLabels[summary.status] || summary.status}
					</Badge>
					<span class="text-sm text-tertiary">
						{filingTypeLabels[summary.filing_type] || summary.filing_type}
					</span>
					<span class="text-sm text-tertiary">
						{quarterLabels[summary.period.quarter] || `Q${summary.period.quarter}`}
					</span>
					{#if summary.filed_at}
						<span class="text-sm text-tertiary">
							Podáno: {new Date(summary.filed_at).toLocaleDateString('cs-CZ')}
						</span>
					{/if}
				</div>
			</div>
			<div class="mt-3 flex flex-wrap gap-2">
				<Button
					variant="secondary"
					onclick={handleRecalculate}
					disabled={actionLoading || summary.status === 'filed'}
				>
					Přepočítat
				</Button>
				<Button
					variant="secondary"
					onclick={handleGenerateXml}
					disabled={actionLoading || summary.status === 'filed'}
				>
					Generovat XML
				</Button>
				<Button
					variant="secondary"
					onclick={handleDownloadXml}
					disabled={!summary.has_xml}
				>
					Stáhnout XML
				</Button>
				<Button
					variant="success"
					onclick={handleMarkFiled}
					disabled={actionLoading || summary.status === 'filed' || !summary.has_xml}
				>
					Označit za podané
				</Button>
				<Button
					variant="danger"
					onclick={handleDelete}
					disabled={summary.status === 'filed'}
				>
					Smazat
				</Button>
			</div>
		</div>

		<!-- Lines table -->
		<div class="mt-6">
			{#if !summary.lines || summary.lines.length === 0}
				<Card>
					<div class="py-4 text-center text-sm text-muted">
						Žádné řádky v souhrnném hlášení
					</div>
				</Card>
			{:else}
				<Card padding={false} class="overflow-x-auto">
					<table class="w-full text-sm">
						<thead class="border-b border-border bg-elevated">
							<tr>
								<th class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted">Kód země</th>
								<th class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted">DIC partnera <HelpTip topic="dic" /></th>
								<th class="px-4 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted">Celková částka (CZK)</th>
								<th class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted">Kód plnění <HelpTip topic="kod-plneni" /></th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border-subtle">
							{#each summary.lines as line, i (i)}
								<tr class="hover:bg-hover transition-colors">
									<td class="px-4 py-2.5 text-primary">{line.country_code}</td>
									<td class="px-4 py-2.5 font-mono text-xs text-primary">{line.partner_dic}</td>
									<td class="px-4 py-2.5 text-right font-mono tabular-nums text-secondary">{formatCZK(line.total_amount)}</td>
									<td class="px-4 py-2.5 text-primary">{line.service_code}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</Card>
			{/if}
		</div>

		<div class="mt-4 text-xs text-muted">
			Vytvořeno: {new Date(summary.created_at).toLocaleDateString('cs-CZ')} | Upraveno: {new Date(
				summary.updated_at
			).toLocaleDateString('cs-CZ')}
		</div>
	{/if}
</div>
