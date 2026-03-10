<script lang="ts">
	import { categoriesApi, type ExpenseCategory } from '$lib/api/client';

	interface Props {
		value: string;
		onchange: (value: string) => void;
		id?: string;
	}

	let { value, onchange, id = 'category' }: Props = $props();

	let categories = $state<ExpenseCategory[]>([]);
	let loaded = $state(false);
	let customMode = $state(false);
	let customValue = $state('');

	$effect(() => {
		loadCategories();
	});

	async function loadCategories() {
		try {
			categories = await categoriesApi.list();
			loaded = true;
			// If current value doesn't match any category key, show custom input
			if (value && !categories.some((c) => c.key === value)) {
				customMode = true;
				customValue = value;
			}
		} catch {
			loaded = true;
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
		class="mt-1 w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 text-sm shadow-sm"
	>
		<option>Nacitam...</option>
	</select>
{:else if customMode}
	<div class="flex gap-2">
		<input
			{id}
			type="text"
			value={customValue}
			oninput={handleCustomInput}
			placeholder="Vlastni kategorie..."
			class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
		/>
		<button
			type="button"
			onclick={exitCustomMode}
			class="mt-1 shrink-0 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 transition-colors"
			title="Zpet na vyber"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
				<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
			</svg>
		</button>
	</div>
{:else}
	<select
		{id}
		value={value}
		onchange={handleSelectChange}
		class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
	>
		<option value="">-- Vyberte --</option>
		{#each categories as cat}
			<option value={cat.key}>
				{cat.label_cs}
			</option>
		{/each}
		<option value="__custom__">+ Vlastni kategorie...</option>
	</select>
{/if}
