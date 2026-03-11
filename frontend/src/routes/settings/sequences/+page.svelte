<script lang="ts">
	import { sequencesApi, type InvoiceSequence } from '$lib/api/client';
	import Card from '$lib/ui/Card.svelte';
	import Button from '$lib/ui/Button.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import EmptyState from '$lib/ui/EmptyState.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';

	let sequences = $state<InvoiceSequence[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Create form state
	let showCreateForm = $state(false);
	let createPrefix = $state('FV');
	let createYear = $state(new Date().getFullYear());
	let createNextNumber = $state(1);
	let createFormatPattern = $state('{prefix}{year}{number:04d}');
	let creating = $state(false);

	// Edit state
	let editingId = $state<number | null>(null);
	let editNextNumber = $state(1);
	let saving = $state(false);

	$effect(() => {
		loadSequences();
	});

	async function loadSequences() {
		loading = true;
		error = null;
		try {
			sequences = await sequencesApi.list();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst číselné řady';
		} finally {
			loading = false;
		}
	}

	async function handleCreate() {
		creating = true;
		error = null;
		try {
			await sequencesApi.create({
				prefix: createPrefix,
				year: createYear,
				next_number: createNextNumber,
				format_pattern: createFormatPattern
			});
			showCreateForm = false;
			createPrefix = 'FV';
			createYear = new Date().getFullYear();
			createNextNumber = 1;
			createFormatPattern = '{prefix}{year}{number:04d}';
			await loadSequences();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se vytvořit číselnou řadu';
		} finally {
			creating = false;
		}
	}

	function startEdit(seq: InvoiceSequence) {
		editingId = seq.id;
		editNextNumber = seq.next_number;
	}

	function cancelEdit() {
		editingId = null;
	}

	async function handleUpdate(seq: InvoiceSequence) {
		saving = true;
		error = null;
		try {
			await sequencesApi.update(seq.id, {
				prefix: seq.prefix,
				year: seq.year,
				next_number: editNextNumber,
				format_pattern: seq.format_pattern
			});
			editingId = null;
			await loadSequences();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se uložit změny';
		} finally {
			saving = false;
		}
	}

	async function handleDelete(id: number) {
		if (!confirm('Opravdu chcete smazat tuto číselnou řadu?')) return;
		error = null;
		try {
			await sequencesApi.delete(id);
			await loadSequences();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se smazat číselnou řadu';
		}
	}

	let createPreview = $derived(
		`${createPrefix}${createYear}${String(createNextNumber).padStart(4, '0')}`
	);
</script>

<svelte:head>
	<title>Číselné řady - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<PageHeader title="Číselné řady faktur" description="Správa číslování faktur podle roku a typu" backHref="/settings" backLabel="Zpět na nastavení">
		{#snippet actions()}
			<Button
				variant="primary"
				onclick={() => {
					showCreateForm = !showCreateForm;
				}}
			>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
				</svg>
				Nová řada
			</Button>
		{/snippet}
	</PageHeader>

	<ErrorAlert {error} class="mt-4" />

	<!-- Create Form -->
	{#if showCreateForm}
		<div class="mt-6">
			<Card>
				<h2 class="text-base font-semibold text-primary">Nová číselná řada</h2>
				<form
					onsubmit={(e) => {
						e.preventDefault();
						handleCreate();
					}}
					class="mt-4 space-y-4"
				>
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
						<div>
							<label for="create-prefix" class="block text-sm font-medium text-secondary">Prefix <HelpTip topic="prefix-format" /></label>
							<input
								id="create-prefix"
								type="text"
								bind:value={createPrefix}
								placeholder="FV"
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
							<p class="mt-1 text-xs text-muted">FV = faktura, ZF = záloha, DN = dobropis</p>
						</div>
						<div>
							<label for="create-year" class="block text-sm font-medium text-secondary">Rok</label>
							<input
								id="create-year"
								type="number"
								bind:value={createYear}
								min="2020"
								max="2099"
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div>
							<label for="create-next" class="block text-sm font-medium text-secondary"
								>Počáteční číslo</label
							>
							<input
								id="create-next"
								type="number"
								bind:value={createNextNumber}
								min="1"
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div>
							<label for="create-format" class="block text-sm font-medium text-secondary">Formát <HelpTip topic="prefix-format" /></label>
							<input
								id="create-format"
								type="text"
								bind:value={createFormatPattern}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
					</div>
					<div class="flex items-center gap-4">
						<p class="text-sm text-tertiary">
							Náhled: <span class="font-mono font-medium text-primary">{createPreview}</span>
						</p>
						<div class="ml-auto flex gap-2">
							<Button
								variant="secondary"
								onclick={() => {
									showCreateForm = false;
								}}
							>
								Zrušit
							</Button>
							<Button
								type="submit"
								variant="primary"
								disabled={creating || !createPrefix || !createYear}
							>
								{creating ? 'Vytváří se...' : 'Vytvořit'}
							</Button>
						</div>
					</div>
				</form>
			</Card>
		</div>
	{/if}

	<!-- Table -->
	<div class="mt-6 overflow-hidden rounded-lg border border-border bg-surface">
		{#if loading}
			<LoadingSpinner class="p-12" />
		{:else if sequences.length === 0}
			<EmptyState message="Zatím žádné číselné řady. Vytvořte novou nebo se vytvoří automaticky při tvorbě faktury." />
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-border bg-elevated">
					<tr>
						<th class="th-default">Prefix</th>
						<th class="th-default">Rok</th>
						<th class="th-default">Další číslo</th>
						<th class="th-default">Náhled</th>
						<th class="th-default text-right">Akce</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border-subtle">
					{#each sequences as seq (seq.id)}
						<tr class="hover:bg-hover transition-colors">
							<td class="px-4 py-2.5 font-mono text-sm font-medium text-primary">{seq.prefix}</td>
							<td class="px-4 py-2.5 font-mono text-sm text-secondary">{seq.year}</td>
							<td class="px-4 py-2.5">
								{#if editingId === seq.id}
									<div class="flex items-center gap-2">
										<input
											type="number"
											bind:value={editNextNumber}
											min="1"
											class="w-24 rounded-lg border border-border bg-surface px-2 py-1 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
										/>
										<span class="text-xs text-warning"
											>Pozor: Změna čísla může způsobit duplicity!</span
										>
									</div>
								{:else}
									<span class="text-secondary">{seq.next_number}</span>
								{/if}
							</td>
							<td class="px-4 py-2.5 font-mono text-sm text-tertiary">{seq.preview}</td>
							<td class="px-4 py-2.5 text-right">
								{#if editingId === seq.id}
									<div class="flex justify-end gap-2">
										<Button variant="ghost" size="sm" onclick={() => cancelEdit()}>
											Zrušit
										</Button>
										<Button variant="primary" size="sm" onclick={() => handleUpdate(seq)} disabled={saving}>
											{saving ? 'Ukládám...' : 'Uložit'}
										</Button>
									</div>
								{:else}
									<div class="flex justify-end gap-2">
										<button
											onclick={() => startEdit(seq)}
											class="rounded px-2 py-1 text-sm text-accent-text hover:text-accent transition-colors"
										>
											Upravit
										</button>
										<button
											onclick={() => handleDelete(seq.id)}
											class="rounded px-2 py-1 text-sm text-danger hover:text-danger hover:bg-danger-bg transition-colors"
										>
											Smazat
										</button>
									</div>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>
</div>
