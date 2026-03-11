<script lang="ts">
	import { onMount } from 'svelte';
	import { categoriesApi, type ExpenseCategory } from '$lib/api/client';

	interface Props {
		value: string;
		onchange: (value: string) => void;
		id?: string;
	}

	let { value, onchange, id = 'category' }: Props = $props();

	let categories = $state<ExpenseCategory[]>([]);
	let loaded = $state(false);
	let loadError = $state<string | null>(null);
	let customMode = $state(false);
	let customValue = $state('');

	onMount(() => {
		loadCategories();
	});

	async function loadCategories() {
		try {
			categories = await categoriesApi.list();
			loaded = true;
			loadError = null;
			if (value && !categories.some((c) => c.key === value)) {
				customMode = true;
				customValue = value;
			}
		} catch (e) {
			loaded = true;
			loadError = e instanceof Error ? e.message : 'Nepodařilo se načíst kategorie';
		}
	}

	function handleSelectChange(e: Event) {
		const selected = (e.target as HTMLSelectElement).value;
		if (selected === '__custom__') {
			customMode = true;
			customValue = '';
			onchange('');
		} else {
			customMode = false;
			customValue = '';
			onchange(selected);
		}
	}

	function handleCustomInput(e: Event) {
		customValue = (e.target as HTMLInputElement).value;
		onchange(customValue);
	}

	function exitCustomMode() {
		customMode = false;
		customValue = '';
		onchange('');
	}
</script>

{#if !loaded}
	<select
		{id}
		disabled
		class="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-muted"
	>
		<option>Načítám...</option>
	</select>
{:else if loadError}
	<div class="rounded-lg border border-danger/20 bg-danger-bg px-3 py-2 text-sm text-danger">
		{loadError}
	</div>
{:else if customMode}
	<div class="flex gap-2">
		<input
			{id}
			type="text"
			value={customValue}
			oninput={handleCustomInput}
			placeholder="Vlastní kategorie..."
			class="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
		/>
		<button
			type="button"
			onclick={exitCustomMode}
			class="shrink-0 rounded-md px-2.5 py-1.5 text-xs text-secondary hover:bg-hover hover:text-primary transition-colors"
			title="Zpět na výběr"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
				<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
			</svg>
		</button>
	</div>
{:else}
	<select
		{id}
		{value}
		onchange={handleSelectChange}
		class="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
	>
		<option value="">-- Vyberte --</option>
		{#each categories as cat (cat.key)}
			<option value={cat.key}>
				{cat.label_cs}
			</option>
		{/each}
		<option value="__custom__">+ Vlastní kategorie...</option>
	</select>
{/if}
