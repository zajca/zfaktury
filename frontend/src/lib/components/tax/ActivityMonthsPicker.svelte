<script lang="ts">
	import Button from '$lib/ui/Button.svelte';

	interface Props {
		value: number[];
		onchange: (months: number[]) => void;
		disabled?: boolean;
		idPrefix: string;
	}

	let { value, onchange, disabled = false, idPrefix }: Props = $props();

	const fullYear = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12];

	const monthNames = fullYear.map((month) =>
		new Intl.DateTimeFormat('cs-CZ', { month: 'long' }).format(new Date(2024, month - 1, 1))
	);

	function toggle(month: number) {
		const next = value.includes(month)
			? value.filter((m) => m !== month)
			: [...value, month].sort((a, b) => a - b);
		onchange(next);
	}
</script>

<fieldset class="mt-4">
	<legend class="text-sm font-medium text-secondary">Měsíce činnosti</legend>
	<p class="mt-1 text-xs text-muted">
		Měsíce, ve kterých jste činnost vykonávali alespoň část měsíce.
	</p>
	<div class="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-3">
		{#each fullYear as month (month)}
			<label
				for="{idPrefix}-m{month}"
				class="flex items-center gap-2 rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary"
			>
				<input
					id="{idPrefix}-m{month}"
					type="checkbox"
					checked={value.includes(month)}
					{disabled}
					onchange={() => toggle(month)}
					class="h-4 w-4 rounded border-border text-accent focus:ring-1 focus:ring-accent/50"
				/>
				{monthNames[month - 1]}
			</label>
		{/each}
	</div>
	<div class="mt-3">
		<Button variant="secondary" {disabled} onclick={() => onchange([...fullYear])}>Celý rok</Button>
	</div>
	{#if value.length === 0}
		<div role="alert" class="mt-3 text-sm text-danger">Vyberte alespoň jeden měsíc činnosti.</div>
	{/if}
</fieldset>
