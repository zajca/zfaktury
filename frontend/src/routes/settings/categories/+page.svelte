<script lang="ts">
	import { categoriesApi, type ExpenseCategory } from '$lib/api/client';

	let categories = $state<ExpenseCategory[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let saving = $state(false);

	// Form state for create/edit
	let showForm = $state(false);
	let editingId = $state<number | null>(null);
	let form = $state({
		key: '',
		label_cs: '',
		label_en: '',
		color: '#6B7280',
		sort_order: 0
	});

	$effect(() => {
		loadCategories();
	});

	async function loadCategories() {
		loading = true;
		error = null;
		try {
			categories = await categoriesApi.list();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se nacist kategorie';
		} finally {
			loading = false;
		}
	}

	function startCreate() {
		editingId = null;
		form = { key: '', label_cs: '', label_en: '', color: '#6B7280', sort_order: 0 };
		showForm = true;
	}

	function startEdit(cat: ExpenseCategory) {
		editingId = cat.id;
		form = {
			key: cat.key,
			label_cs: cat.label_cs,
			label_en: cat.label_en,
			color: cat.color,
			sort_order: cat.sort_order
		};
		showForm = true;
	}

	function cancelForm() {
		showForm = false;
		editingId = null;
	}

	async function handleSave() {
		if (!form.key || !form.label_cs || !form.label_en) {
			error = 'Klic, cesky a anglicky nazev jsou povinne';
			return;
		}

		saving = true;
		error = null;

		try {
			if (editingId) {
				await categoriesApi.update(editingId, form);
			} else {
				await categoriesApi.create(form);
			}
			showForm = false;
			editingId = null;
			await loadCategories();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se ulozit kategorii';
		} finally {
			saving = false;
		}
	}

	async function handleDelete(cat: ExpenseCategory) {
		if (cat.is_default) {
			error = 'Vychozi kategorie nelze smazat';
			return;
		}
		if (!confirm(`Opravdu chcete smazat kategorii "${cat.label_cs}"?`)) return;

		error = null;
		try {
			await categoriesApi.delete(cat.id);
			await loadCategories();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se smazat kategorii';
		}
	}
</script>

<svelte:head>
	<title>Kategorie nakladu - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-3xl">
	<a href="/settings" class="text-sm text-blue-600 hover:text-blue-800">&larr; Zpet na nastaveni</a>
	<div class="mt-2 flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-gray-900">Kategorie nakladu</h1>
			<p class="mt-1 text-sm text-gray-500">Sprava kategorii pro trideni nakladu</p>
		</div>
		{#if !showForm}
			<button
				onclick={startCreate}
				class="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 transition-colors"
			>
				Pridat kategorii
			</button>
		{/if}
	</div>

	{#if error}
		<div class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
			{error}
		</div>
	{/if}

	{#if showForm}
		<form onsubmit={(e) => { e.preventDefault(); handleSave(); }} class="mt-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">
				{editingId ? 'Upravit kategorii' : 'Nova kategorie'}
			</h2>
			<div class="mt-4 space-y-4">
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div>
						<label for="cat-key" class="block text-sm font-medium text-gray-700">Klic *</label>
						<input
							id="cat-key"
							type="text"
							bind:value={form.key}
							placeholder="napr. office_supplies"
							pattern="[a-z0-9_]+"
							required
							class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
						/>
						<p class="mt-1 text-xs text-gray-400">Mala pismena, cisla a podtrzitka</p>
					</div>
					<div>
						<label for="cat-color" class="block text-sm font-medium text-gray-700">Barva</label>
						<div class="mt-1 flex items-center gap-2">
							<input
								id="cat-color"
								type="color"
								bind:value={form.color}
								class="h-9 w-12 cursor-pointer rounded border border-gray-300"
							/>
							<span class="text-sm text-gray-500">{form.color}</span>
						</div>
					</div>
				</div>
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div>
						<label for="cat-label-cs" class="block text-sm font-medium text-gray-700">Cesky nazev *</label>
						<input
							id="cat-label-cs"
							type="text"
							bind:value={form.label_cs}
							required
							class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
						/>
					</div>
					<div>
						<label for="cat-label-en" class="block text-sm font-medium text-gray-700">Anglicky nazev *</label>
						<input
							id="cat-label-en"
							type="text"
							bind:value={form.label_en}
							required
							class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
						/>
					</div>
				</div>
				<div>
					<label for="cat-sort" class="block text-sm font-medium text-gray-700">Poradi razeni</label>
					<input
						id="cat-sort"
						type="number"
						min="0"
						bind:value={form.sort_order}
						class="mt-1 w-32 rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					/>
				</div>
			</div>
			<div class="mt-6 flex gap-3">
				<button
					type="submit"
					disabled={saving}
					class="rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50 transition-colors"
				>
					{saving ? 'Ukladam...' : (editingId ? 'Ulozit zmeny' : 'Vytvorit')}
				</button>
				<button
					type="button"
					onclick={cancelForm}
					class="rounded-lg border border-gray-300 px-6 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
				>
					Zrusit
				</button>
			</div>
		</form>
	{/if}

	{#if loading}
		<div class="mt-8 flex items-center justify-center">
			<div class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"></div>
		</div>
	{:else}
		<div class="mt-6 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
			<table class="min-w-full divide-y divide-gray-200">
				<thead class="bg-gray-50">
					<tr>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Barva</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Klic</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Nazev (CZ)</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Nazev (EN)</th>
						<th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Poradi</th>
						<th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500">Akce</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-200">
					{#each categories as cat}
						<tr class="hover:bg-gray-50">
							<td class="whitespace-nowrap px-4 py-3">
								<div
									class="h-5 w-5 rounded-full border border-gray-200"
									style="background-color: {cat.color}"
								></div>
							</td>
							<td class="whitespace-nowrap px-4 py-3 text-sm font-mono text-gray-600">{cat.key}</td>
							<td class="whitespace-nowrap px-4 py-3 text-sm text-gray-900">{cat.label_cs}</td>
							<td class="whitespace-nowrap px-4 py-3 text-sm text-gray-500">{cat.label_en}</td>
							<td class="whitespace-nowrap px-4 py-3 text-sm text-gray-500">{cat.sort_order}</td>
							<td class="whitespace-nowrap px-4 py-3 text-right text-sm">
								<button
									onclick={() => startEdit(cat)}
									class="text-blue-600 hover:text-blue-800 mr-3"
								>
									Upravit
								</button>
								{#if !cat.is_default}
									<button
										onclick={() => handleDelete(cat)}
										class="text-red-600 hover:text-red-800"
									>
										Smazat
									</button>
								{:else}
									<span class="text-xs text-gray-400">vychozi</span>
								{/if}
							</td>
						</tr>
					{/each}
					{#if categories.length === 0}
						<tr>
							<td colspan="6" class="px-4 py-8 text-center text-sm text-gray-500">
								Zadne kategorie
							</td>
						</tr>
					{/if}
				</tbody>
			</table>
		</div>
	{/if}
</div>
