<script lang="ts">
	import { onMount } from 'svelte';
	import { auditLogApi, type AuditLogEntry } from '$lib/api/client';
	import Card from '$lib/ui/Card.svelte';
	import Badge from '$lib/ui/Badge.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';

	interface Props {
		entityType: string;
		entityId: number;
	}

	let { entityType, entityId }: Props = $props();

	let entries = $state<AuditLogEntry[]>([]);
	let total = $state(0);
	let loading = $state(true);
	let error = $state('');
	let expanded = $state(false);
	let expandedEntries = $state<Set<number>>(new Set());
	let offset = $state(0);
	const limit = 20;

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

	async function loadData() {
		loading = true;
		error = '';
		try {
			const res = await auditLogApi.list({
				entity_type: entityType,
				entity_id: entityId,
				limit,
				offset
			});
			entries = res.items;
			total = res.total;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba pri nacitani historie';
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadData();
	});
</script>

<Card padding={false} class="mt-4">
	<button
		type="button"
		class="flex w-full items-center justify-between p-5 text-left transition-colors hover:bg-hover"
		onclick={() => (expanded = !expanded)}
	>
		<div class="flex items-center gap-2">
			<h3 class="text-sm font-medium text-primary">Historie zmen</h3>
			{#if !loading && total > 0}
				<Badge variant="muted">{total}</Badge>
			{/if}
		</div>
		<svg
			class="h-4 w-4 text-muted transition-transform {expanded ? 'rotate-180' : ''}"
			fill="none"
			viewBox="0 0 24 24"
			stroke="currentColor"
			stroke-width="2"
		>
			<path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
		</svg>
	</button>

	{#if expanded}
		<div class="border-t border-border px-5 pb-5">
			{#if loading}
				<LoadingSpinner class="py-6" />
			{:else if error}
				<ErrorAlert error={error} class="mt-4" />
			{:else if entries.length === 0}
				<p class="py-4 text-sm text-muted">Zadne zaznamy</p>
			{:else}
				<div class="mt-4 space-y-2">
					{#each entries as entry (entry.id)}
						<div class="rounded-lg border border-border bg-elevated text-sm">
							<button
								type="button"
								class="flex w-full items-center justify-between px-4 py-2.5 text-left transition-colors hover:bg-hover"
								onclick={() => toggleEntry(entry.id)}
							>
								<div class="flex items-center gap-3">
									<span class="text-xs text-muted">{formatDate(entry.created_at)}</span>
									<Badge variant={actionVariants[entry.action] || 'default'}>
										{formatAction(entry.action)}
									</Badge>
								</div>
								{#if entry.old_values || entry.new_values}
									<svg
										class="h-4 w-4 text-muted transition-transform {expandedEntries.has(entry.id)
											? 'rotate-180'
											: ''}"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="2"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M19 9l-7 7-7-7"
										/>
									</svg>
								{/if}
							</button>

							{#if expandedEntries.has(entry.id) && (entry.old_values || entry.new_values)}
								<div class="border-t border-border-subtle px-4 py-3">
									<div class="grid gap-3 md:grid-cols-2">
										{#if entry.old_values}
											<div>
												<p class="mb-1 text-xs font-medium text-muted">Pred zmenou</p>
												<pre
													class="max-h-48 overflow-auto rounded bg-danger-bg p-2 text-xs text-danger">{formatJson(entry.old_values)}</pre>
											</div>
										{/if}
										{#if entry.new_values}
											<div>
												<p class="mb-1 text-xs font-medium text-muted">Po zmene</p>
												<pre
													class="max-h-48 overflow-auto rounded bg-success-bg p-2 text-xs text-success">{formatJson(entry.new_values)}</pre>
											</div>
										{/if}
									</div>
								</div>
							{/if}
						</div>
					{/each}
				</div>

				{#if total > limit}
					<div class="mt-4 flex items-center justify-between">
						<span class="text-xs text-muted"
							>Zobrazeno {offset + 1}-{Math.min(offset + entries.length, total)} z {total}</span
						>
						<div class="flex gap-2">
							<button
								type="button"
								class="rounded border border-border px-2.5 py-1 text-xs text-secondary transition-colors hover:bg-hover disabled:opacity-50"
								disabled={offset === 0}
								onclick={() => {
									offset = Math.max(0, offset - limit);
									loadData();
								}}>Predchozi</button
							>
							<button
								type="button"
								class="rounded border border-border px-2.5 py-1 text-xs text-secondary transition-colors hover:bg-hover disabled:opacity-50"
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
		</div>
	{/if}
</Card>
