<script lang="ts">
	import { sequencesApi, type InvoiceSequence } from '$lib/api/client';

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

	let createPreview = $derived(`${createPrefix}${createYear}${String(createNextNumber).padStart(4, '0')}`);
</script>

<svelte:head>
	<title>Číselné řady - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-4xl">
	<div class="flex items-center justify-between">
		<div>
			<div class="flex items-center gap-3">
				<a href="/settings" class="text-gray-400 hover:text-gray-600 transition-colors">
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" />
					</svg>
				</a>
				<h1 class="text-2xl font-bold text-gray-900">Číselné řady faktur</h1>
			</div>
			<p class="mt-1 text-sm text-gray-500">Správa číslování faktur podle roku a typu</p>
		</div>
		<button
			onclick={() => { showCreateForm = !showCreateForm; }}
			class="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 transition-colors"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
			</svg>
			Nová řada
		</button>
	</div>

	{#if error}
		<div class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
			{error}
		</div>
	{/if}

	<!-- Create Form -->
	{#if showCreateForm}
		<div class="mt-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Nová číselná řada</h2>
			<form onsubmit={(e) => { e.preventDefault(); handleCreate(); }} class="mt-4 space-y-4">
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
					<div>
						<label for="create-prefix" class="block text-sm font-medium text-gray-700">Prefix</label>
						<input
							id="create-prefix"
							type="text"
							bind:value={createPrefix}
							placeholder="FV"
							class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
						/>
						<p class="mt-1 text-xs text-gray-400">FV = faktura, ZF = záloha, DN = dobropis</p>
					</div>
					<div>
						<label for="create-year" class="block text-sm font-medium text-gray-700">Rok</label>
						<input
							id="create-year"
							type="number"
							bind:value={createYear}
							min="2020"
							max="2099"
							class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
						/>
					</div>
					<div>
						<label for="create-next" class="block text-sm font-medium text-gray-700">Počáteční číslo</label>
						<input
							id="create-next"
							type="number"
							bind:value={createNextNumber}
							min="1"
							class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
						/>
					</div>
					<div>
						<label for="create-format" class="block text-sm font-medium text-gray-700">Formát</label>
						<input
							id="create-format"
							type="text"
							bind:value={createFormatPattern}
							class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
						/>
					</div>
				</div>
				<div class="flex items-center gap-4">
					<p class="text-sm text-gray-500">
						Náhled: <span class="font-mono font-medium text-gray-900">{createPreview}</span>
					</p>
					<div class="ml-auto flex gap-2">
						<button
							type="button"
							onclick={() => { showCreateForm = false; }}
							class="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
						>
							Zrušit
						</button>
						<button
							type="submit"
							disabled={creating || !createPrefix || !createYear}
							class="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50 transition-colors"
						>
							{creating ? 'Vytváří se...' : 'Vytvořit'}
						</button>
					</div>
				</div>
			</form>
		</div>
	{/if}

	<!-- Table -->
	<div class="mt-6 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
		{#if loading}
			<div class="flex items-center justify-center p-12">
				<div class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"></div>
			</div>
		{:else if sequences.length === 0}
			<div class="p-12 text-center text-gray-400">
				Zatím žádné číselné řady. Vytvořte novou nebo se vytvoří automaticky při tvorbě faktury.
			</div>
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-gray-200 bg-gray-50">
					<tr>
						<th class="px-4 py-3 font-medium text-gray-600">Prefix</th>
						<th class="px-4 py-3 font-medium text-gray-600">Rok</th>
						<th class="px-4 py-3 font-medium text-gray-600">Další číslo</th>
						<th class="px-4 py-3 font-medium text-gray-600">Náhled</th>
						<th class="px-4 py-3 font-medium text-gray-600 text-right">Akce</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-100">
					{#each sequences as seq}
						<tr class="hover:bg-gray-50 transition-colors">
							<td class="px-4 py-3 font-medium text-gray-900">{seq.prefix}</td>
							<td class="px-4 py-3 text-gray-600">{seq.year}</td>
							<td class="px-4 py-3">
								{#if editingId === seq.id}
									<div class="flex items-center gap-2">
										<input
											type="number"
											bind:value={editNextNumber}
											min="1"
											class="w-24 rounded-lg border border-gray-300 px-2 py-1 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
										/>
										<span class="text-xs text-amber-600">Pozor: Změna čísla může způsobit duplicity!</span>
									</div>
								{:else}
									<span class="text-gray-600">{seq.next_number}</span>
								{/if}
							</td>
							<td class="px-4 py-3 font-mono text-sm text-gray-500">{seq.preview}</td>
							<td class="px-4 py-3 text-right">
								{#if editingId === seq.id}
									<div class="flex justify-end gap-2">
										<button
											onclick={() => cancelEdit()}
											class="rounded px-2 py-1 text-sm text-gray-600 hover:bg-gray-100 transition-colors"
										>
											Zrušit
										</button>
										<button
											onclick={() => handleUpdate(seq)}
											disabled={saving}
											class="rounded bg-blue-600 px-3 py-1 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50 transition-colors"
										>
											{saving ? 'Ukládám...' : 'Uložit'}
										</button>
									</div>
								{:else}
									<div class="flex justify-end gap-2">
										<button
											onclick={() => startEdit(seq)}
											class="rounded px-2 py-1 text-sm text-blue-600 hover:bg-blue-50 transition-colors"
										>
											Upravit
										</button>
										<button
											onclick={() => handleDelete(seq.id)}
											class="rounded px-2 py-1 text-sm text-red-600 hover:bg-red-50 transition-colors"
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
