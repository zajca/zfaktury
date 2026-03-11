<script lang="ts">
	import { categoriesApi, type ExpenseCategory } from '$lib/api/client';
	import Card from '$lib/ui/Card.svelte';
	import Button from '$lib/ui/Button.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';

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
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst kategorie';
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
			error = 'Klíč, český a anglický název jsou povinné';
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
			error = e instanceof Error ? e.message : 'Nepodařilo se uložit kategorii';
		} finally {
			saving = false;
		}
	}

	async function handleDelete(cat: ExpenseCategory) {
		if (cat.is_default) {
			error = 'Výchozí kategorie nelze smazat';
			return;
		}
		if (!confirm(`Opravdu chcete smazat kategorii "${cat.label_cs}"?`)) return;

		error = null;
		try {
			await categoriesApi.delete(cat.id);
			await loadCategories();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se smazat kategorii';
		}
	}
</script>

<svelte:head>
	<title>Kategorie nákladů - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<PageHeader title="Kategorie nákladů" description="Správa kategorií pro třídění nákladů" backHref="/settings" backLabel="Zpět na nastavení">
		{#snippet actions()}
			{#if !showForm}
				<Button variant="primary" onclick={startCreate}>
					Přidat kategorii
				</Button>
			{/if}
		{/snippet}
	</PageHeader>

	<p class="mt-2 text-sm text-tertiary">
		Kategorie pomáhají třídit náklady podle typu pro přehlednou evidenci.
		<HelpTip topic="kategorie-nakladu" />
	</p>

	<ErrorAlert {error} class="mt-4" />

	{#if showForm}
		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleSave();
			}}
			class="mt-6"
		>
			<Card>
				<h2 class="text-base font-semibold text-primary">
					{editingId ? 'Upravit kategorii' : 'Nová kategorie'}
				</h2>
				<div class="mt-4 space-y-4">
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div>
							<label for="cat-key" class="block text-sm font-medium text-secondary">Klíč *</label>
							<input
								id="cat-key"
								type="text"
								bind:value={form.key}
								placeholder="např. office_supplies"
								pattern="[a-z0-9_]+"
								required
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
							<p class="mt-1 text-xs text-muted">Malá písmena, čísla a podtržítka</p>
						</div>
						<div>
							<label for="cat-color" class="block text-sm font-medium text-secondary">Barva</label>
							<div class="mt-1 flex items-center gap-2">
								<input
									id="cat-color"
									type="color"
									bind:value={form.color}
									class="h-9 w-12 cursor-pointer rounded border border-border"
								/>
								<span class="font-mono text-xs text-tertiary">{form.color}</span>
							</div>
						</div>
					</div>
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div>
							<label for="cat-label-cs" class="block text-sm font-medium text-secondary"
								>Český název *</label
							>
							<input
								id="cat-label-cs"
								type="text"
								bind:value={form.label_cs}
								required
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div>
							<label for="cat-label-en" class="block text-sm font-medium text-secondary"
								>Anglický název *</label
							>
							<input
								id="cat-label-en"
								type="text"
								bind:value={form.label_en}
								required
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
					</div>
					<div>
						<label for="cat-sort" class="block text-sm font-medium text-secondary">Pořadí řazení</label>
						<input
							id="cat-sort"
							type="number"
							min="0"
							bind:value={form.sort_order}
							class="mt-1 w-32 rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						/>
					</div>
				</div>
				<FormActions {saving} saveLabel={editingId ? 'Uložit změny' : 'Vytvořit'} oncancel={cancelForm} class="mt-6" />
			</Card>
		</form>
	{/if}

	{#if loading}
		<LoadingSpinner class="mt-8" />
	{:else}
		<div class="mt-6 overflow-hidden rounded-lg border border-border bg-surface">
			<table class="min-w-full divide-y divide-border">
				<thead class="bg-elevated">
					<tr>
						<th
							class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted"
							>Barva</th
						>
						<th
							class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted"
							>Klíč</th
						>
						<th
							class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted"
							>Název (CZ)</th
						>
						<th
							class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted"
							>Název (EN)</th
						>
						<th
							class="px-4 py-2.5 text-left text-xs font-medium uppercase tracking-wider text-muted"
							>Pořadí</th
						>
						<th
							class="px-4 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted"
							>Akce</th
						>
					</tr>
				</thead>
				<tbody class="divide-y divide-border-subtle">
					{#each categories as cat (cat.id)}
						<tr class="hover:bg-hover">
							<td class="whitespace-nowrap px-4 py-2.5">
								<div
									class="h-5 w-5 rounded-full border border-border"
									style:background-color={cat.color}
								></div>
							</td>
							<td class="whitespace-nowrap px-4 py-2.5 text-sm font-mono text-secondary">{cat.key}</td>
							<td class="whitespace-nowrap px-4 py-2.5 text-sm text-primary">{cat.label_cs}</td>
							<td class="whitespace-nowrap px-4 py-2.5 text-sm text-tertiary">{cat.label_en}</td>
							<td class="whitespace-nowrap px-4 py-2.5 text-sm text-tertiary">{cat.sort_order}</td>
							<td class="whitespace-nowrap px-4 py-2.5 text-right text-sm">
								<button
									onclick={() => startEdit(cat)}
									class="text-accent-text hover:text-accent mr-3"
								>
									Upravit
								</button>
								{#if !cat.is_default}
									<button onclick={() => handleDelete(cat)} class="text-danger hover:text-danger">
										Smazat
									</button>
								{:else}
									<span class="text-xs text-muted italic">výchozí</span>
								{/if}
							</td>
						</tr>
					{/each}
					{#if categories.length === 0}
						<tr>
							<td colspan="6" class="px-4 py-8 text-center text-sm text-tertiary">
								Žádné kategorie
							</td>
						</tr>
					{/if}
				</tbody>
			</table>
		</div>
	{/if}
</div>
