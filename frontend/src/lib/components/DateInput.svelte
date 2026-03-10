<script lang="ts">
	import { formatDateLong, addDays, toISODate } from '$lib/utils/date';

	interface Preset {
		label: string;
		days: number;
	}

	interface Props {
		value: string;
		onchange?: (value: string) => void;
		id?: string;
		required?: boolean;
		presets?: Preset[];
		relativeToValue?: string;
	}

	let {
		value = $bindable(),
		onchange,
		id,
		required = false,
		presets,
		relativeToValue
	}: Props = $props();

	let inputEl: HTMLInputElement | undefined = $state();

	let formattedDate = $derived(formatDateLong(value));

	function handleClick() {
		try {
			inputEl?.showPicker();
		} catch {
			// showPicker() not supported in all browsers
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 't' || e.key === 'T') {
			e.preventDefault();
			setToday();
		}
	}

	function setToday() {
		const today = toISODate(new Date());
		value = today;
		onchange?.(today);
	}

	function handleInput(e: Event) {
		const target = e.target as HTMLInputElement;
		value = target.value;
		onchange?.(target.value);
	}

	function applyPreset(preset: Preset) {
		const base = relativeToValue || value || toISODate(new Date());
		const newDate = addDays(base, preset.days);
		value = newDate;
		onchange?.(newDate);
	}
</script>

<div class="flex flex-col gap-1">
	<div class="flex items-center gap-2">
		<input
			bind:this={inputEl}
			{id}
			type="date"
			value={value}
			{required}
			onclick={handleClick}
			oninput={handleInput}
			onkeydown={handleKeydown}
			class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
		/>
		<button
			type="button"
			onclick={setToday}
			class="mt-1 shrink-0 rounded-lg border border-gray-300 px-2.5 py-2 text-xs font-medium text-gray-600 hover:bg-gray-50 transition-colors"
			title="Nastavit dnešní datum (T)"
		>
			Dnes
		</button>
	</div>
	{#if presets && presets.length > 0}
		<div class="flex flex-wrap gap-1">
			{#each presets as preset}
				<button
					type="button"
					onclick={() => applyPreset(preset)}
					class="rounded-full border border-gray-200 px-2 py-0.5 text-xs text-gray-500 hover:bg-gray-100 hover:text-gray-700 transition-colors"
				>
					{preset.label}
				</button>
			{/each}
		</div>
	{/if}
	{#if formattedDate}
		<span class="text-xs text-gray-400">{formattedDate}</span>
	{/if}
</div>
