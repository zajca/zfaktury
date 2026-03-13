<script lang="ts">
	import { onMount } from 'svelte';
	import { auditLogApi, type AuditLogEntry } from '$lib/api/client';
	import Card from '$lib/ui/Card.svelte';
	import Badge from '$lib/ui/Badge.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';

	let entries = $state<AuditLogEntry[]>([]);
	let total = $state(0);
	let loading = $state(true);
	let error = $state('');
	let offset = $state(0);
	const limit = 50;

	let filterEntityType = $state('');
	let filterAction = $state('');
	let filterFrom = $state('');
	let filterTo = $state('');

	let expandedEntries = $state<Set<number>>(new Set());
	let mounted = false;

	const entityTypeOptions = [
		{ value: '', label: 'Vse' },
		{ value: 'contact', label: 'Kontakt' },
		{ value: 'invoice', label: 'Faktura' },
		{ value: 'expense', label: 'Naklad' },
		{ value: 'recurring_invoice', label: 'Opakujici faktura' },
		{ value: 'recurring_expense', label: 'Opakujici naklad' },
		{ value: 'settings', label: 'Nastaveni' },
		{ value: 'category', label: 'Kategorie' },
		{ value: 'sequence', label: 'Ciselna rada' },
		{ value: 'vat_return', label: 'Priznani DPH' },
		{ value: 'vat_control_statement', label: 'Kontrolni hlaseni' },
		{ value: 'vies_summary', label: 'Souhrnne hlaseni' },
		{ value: 'income_tax_return', label: 'Dan z prijmu' },
		{ value: 'social_insurance', label: 'Socialni pojisteni' },
		{ value: 'health_insurance', label: 'Zdravotni pojisteni' },
		{ value: 'tax_year_settings', label: 'Nastaveni roku' },
		{ value: 'tax_spouse_credit', label: 'Sleva manzel/ka' },
		{ value: 'tax_child_credit', label: 'Sleva deti' },
		{ value: 'tax_personal_credits', label: 'Osobni slevy' },
		{ value: 'tax_deduction', label: 'Odpocet dane' },
		{ value: 'document', label: 'Dokument' },
		{ value: 'tax_deduction_document', label: 'Doklad k odpoctu' },
		{ value: 'investment_document', label: 'Investicni doklad' },
		{ value: 'capital_income', label: 'Kapitalovy prijem' },
		{ value: 'security_transaction', label: 'Obchod s CP' }
	];

	const actionOptions = [
		{ value: '', label: 'Vse' },
		{ value: 'create', label: 'Vytvoreni' },
		{ value: 'update', label: 'Uprava' },
		{ value: 'delete', label: 'Smazani' },
		{ value: 'activate', label: 'Aktivace' },
		{ value: 'deactivate', label: 'Deaktivace' },
		{ value: 'generate_xml', label: 'Generovani XML' },
		{ value: 'mark_filed', label: 'Podani' },
		{ value: 'recalculate', label: 'Prepocet' }
	];

	const actionLabels: Record<string, string> = {
		create: 'Vytvoreno',
		update: 'Upraveno',
		delete: 'Smazano',
		activate: 'Aktivovano',
		deactivate: 'Deaktivovano',
		set: 'Nastaveno',
		set_bulk: 'Hromadne nastaveno',
		generate_xml: 'XML vygenerovano',
		mark_filed: 'Podano',
		recalculate: 'Prepocteno'
	};

	const actionVariants: Record<string, 'success' | 'danger' | 'info' | 'warning' | 'default'> = {
		create: 'success',
		update: 'default',
		delete: 'danger',
		activate: 'info',
		deactivate: 'warning',
		set: 'default',
		set_bulk: 'default',
		generate_xml: 'info',
		mark_filed: 'success',
		recalculate: 'default'
	};

	function entityTypeLabel(type: string): string {
		return entityTypeOptions.find((o) => o.value === type)?.label || type;
	}

	function formatAction(action: string): string {
		return actionLabels[action] || action;
	}

	function formatDate(dateStr: string): string {
		const d = new Date(dateStr);
		return d.toLocaleString('cs-CZ', {
			day: '2-digit',
			month: '2-digit',
			year: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	function toggleEntry(id: number) {
		const next = new Set(expandedEntries);
		if (next.has(id)) {
			next.delete(id);
		} else {
			next.add(id);
		}
		expandedEntries = next;
	}

	function formatJson(raw: string): string {
		if (!raw) return '';
		try {
			return JSON.stringify(JSON.parse(raw), null, 2);
		} catch {
			return raw;
		}
	}

	function entityDetailUrl(entityType: string, entityId: number): string | null {
		switch (entityType) {
			case 'contact':
				return `/contacts/${entityId}`;
			case 'invoice':
				return `/invoices/${entityId}`;
			case 'expense':
				return `/expenses/${entityId}`;
			default:
				return null;
		}
	}

	async function loadData() {
		loading = true;
		error = '';
		try {
			const res = await auditLogApi.list({
				entity_type: filterEntityType || undefined,
				action: filterAction || undefined,
				from: filterFrom || undefined,
				to: filterTo || undefined,
				limit,
				offset
			});
			entries = res.items;
			total = res.total;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba pri nacitani audit logu';
		} finally {
			loading = false;
		}
	}

	function applyFilters() {
		offset = 0;
		loadData();
	}

	onMount(() => {
		loadData();
		mounted = true;
	});

	$effect(() => {
		filterEntityType;
		filterAction;
		filterFrom;
		filterTo;
		if (!mounted) return;
		applyFilters();
	});
</script>

<PageHeader title="Audit log" />

<Card padding={false} class="mt-4">
	<div class="flex flex-wrap items-end gap-3 border-b border-border p-4">
		<div>
			<label for="filter-entity" class="mb-1 block text-xs font-medium text-muted">Typ entity</label
			>
			<select
				id="filter-entity"
				bind:value={filterEntityType}
				class="rounded-md border border-border bg-elevated px-2.5 py-1.5 text-sm text-primary"
			>
				{#each entityTypeOptions as opt}
					<option value={opt.value}>{opt.label}</option>
				{/each}
			</select>
		</div>

		<div>
			<label for="filter-action" class="mb-1 block text-xs font-medium text-muted">Akce</label>
			<select
				id="filter-action"
				bind:value={filterAction}
				class="rounded-md border border-border bg-elevated px-2.5 py-1.5 text-sm text-primary"
			>
				{#each actionOptions as opt}
					<option value={opt.value}>{opt.label}</option>
				{/each}
			</select>
		</div>

		<div>
			<label for="filter-from" class="mb-1 block text-xs font-medium text-muted">Od</label>
			<input
				id="filter-from"
				type="date"
				bind:value={filterFrom}
				class="rounded-md border border-border bg-elevated px-2.5 py-1.5 text-sm text-primary"
			/>
		</div>

		<div>
			<label for="filter-to" class="mb-1 block text-xs font-medium text-muted">Do</label>
			<input
				id="filter-to"
				type="date"
				bind:value={filterTo}
				class="rounded-md border border-border bg-elevated px-2.5 py-1.5 text-sm text-primary"
			/>
		</div>
	</div>

	{#if loading}
		<LoadingSpinner class="py-8" />
	{:else if error}
		<div class="p-4">
			<ErrorAlert {error} />
		</div>
	{:else if entries.length === 0}
		<p class="py-8 text-center text-sm text-muted">Zadne zaznamy</p>
	{:else}
		<div class="overflow-x-auto">
			<table class="w-full text-left text-sm">
				<thead class="border-b border-border bg-elevated">
					<tr>
						<th class="th-default">Datum</th>
						<th class="th-default">Typ</th>
						<th class="th-default">ID</th>
						<th class="th-default">Akce</th>
						<th class="th-default">Detail</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border-subtle">
					{#each entries as entry (entry.id)}
						<tr class="transition-colors hover:bg-hover">
							<td class="whitespace-nowrap px-4 py-2.5 text-xs text-muted"
								>{formatDate(entry.created_at)}</td
							>
							<td class="px-4 py-2.5 text-primary">{entityTypeLabel(entry.entity_type)}</td>
							<td class="px-4 py-2.5">
								{#if entityDetailUrl(entry.entity_type, entry.entity_id)}
									<a
										href={entityDetailUrl(entry.entity_type, entry.entity_id)}
										class="text-accent-text hover:underline">{entry.entity_id}</a
									>
								{:else}
									<span class="text-primary">{entry.entity_id}</span>
								{/if}
							</td>
							<td class="px-4 py-2.5">
								<Badge variant={actionVariants[entry.action] || 'default'}>
									{formatAction(entry.action)}
								</Badge>
							</td>
							<td class="px-4 py-2.5">
								{#if entry.old_values || entry.new_values}
									<button
										type="button"
										class="text-xs text-accent-text hover:underline"
										onclick={() => toggleEntry(entry.id)}
									>
										{expandedEntries.has(entry.id) ? 'Skryt' : 'Zobrazit'}
									</button>
								{/if}
							</td>
						</tr>
						{#if expandedEntries.has(entry.id) && (entry.old_values || entry.new_values)}
							<tr>
								<td colspan="5" class="bg-elevated px-4 py-3">
									<div class="grid gap-3 md:grid-cols-2">
										{#if entry.old_values}
											<div>
												<p class="mb-1 text-xs font-medium text-muted">Pred zmenou</p>
												<pre
													class="max-h-48 overflow-auto rounded bg-danger-bg p-2 text-xs text-danger">{formatJson(
														entry.old_values
													)}</pre>
											</div>
										{/if}
										{#if entry.new_values}
											<div>
												<p class="mb-1 text-xs font-medium text-muted">Po zmene</p>
												<pre
													class="max-h-48 overflow-auto rounded bg-success-bg p-2 text-xs text-success">{formatJson(
														entry.new_values
													)}</pre>
											</div>
										{/if}
									</div>
								</td>
							</tr>
						{/if}
					{/each}
				</tbody>
			</table>
		</div>

		{#if total > limit}
			<div class="flex items-center justify-between border-t border-border px-4 py-3">
				<span class="text-sm text-muted"
					>Zobrazeno {offset + 1}-{Math.min(offset + entries.length, total)} z {total}</span
				>
				<div class="flex gap-2">
					<button
						type="button"
						class="rounded border border-border px-3 py-1 text-sm text-secondary transition-colors hover:bg-hover disabled:opacity-50"
						disabled={offset === 0}
						onclick={() => {
							offset = Math.max(0, offset - limit);
							loadData();
						}}>Predchozi</button
					>
					<button
						type="button"
						class="rounded border border-border px-3 py-1 text-sm text-secondary transition-colors hover:bg-hover disabled:opacity-50"
						disabled={offset + limit >= total}
						onclick={() => {
							offset += limit;
							loadData();
						}}>Dalsi</button
					>
				</div>
			</div>
		{/if}
	{/if}
</Card>
