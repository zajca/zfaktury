<script lang="ts">
	import type { TaxCreditsSummary, TaxConstants } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import Select from '$lib/ui/Select.svelte';

	let {
		isStudent = $bindable(),
		studentMonths = $bindable(),
		disabilityLevel = $bindable(),
		summary,
		taxConstants,
		saving,
		onSave
	}: {
		isStudent: boolean;
		studentMonths: number;
		disabilityLevel: number;
		summary: TaxCreditsSummary | null;
		taxConstants: TaxConstants | null;
		saving: boolean;
		onSave: () => void;
	} = $props();
</script>

<Card>
	<div class="flex items-center justify-between">
		<h2 class="text-base font-semibold text-primary">
			Osobní slevy <HelpTip topic="sleva-na-poplatnika" {taxConstants} />
		</h2>
		<Button variant="primary" size="sm" onclick={onSave} disabled={saving}>Uložit</Button>
	</div>
	<div class="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
		<label class="flex items-center gap-2 text-sm text-primary">
			<input type="checkbox" bind:checked={isStudent} class="rounded border-border" />
			Student
		</label>
		{#if isStudent}
			<div>
				<span class="text-xs text-tertiary">Počet měsíců</span>
				<Select
					value={studentMonths}
					onchange={(e: Event) => {
						studentMonths = Number((e.currentTarget as HTMLSelectElement).value);
					}}
				>
					{#each Array.from({ length: 12 }, (_, i) => i + 1) as m}
						<option value={m}>{m}</option>
					{/each}
				</Select>
			</div>
		{/if}
		<div>
			<span class="text-xs text-tertiary">Invalidita</span>
			<Select
				value={disabilityLevel}
				onchange={(e: Event) => {
					disabilityLevel = Number((e.currentTarget as HTMLSelectElement).value);
				}}
			>
				<option value={0}>Žádná</option>
				<option value={1}>1. a 2. stupeň</option>
				<option value={2}>3. stupeň</option>
				<option value={3}>Držitel ZTP/P</option>
			</Select>
		</div>
	</div>
	{#if summary?.personal}
		<div class="mt-3 flex gap-6 text-sm text-tertiary">
			{#if summary.personal.credit_student > 0}
				<span
					>Sleva student: <strong class="text-primary"
						>{formatCZK(summary.personal.credit_student)}</strong
					></span
				>
			{/if}
			{#if summary.personal.credit_disability > 0}
				<span
					>Sleva invalidita: <strong class="text-primary"
						>{formatCZK(summary.personal.credit_disability)}</strong
					></span
				>
			{/if}
		</div>
	{/if}
</Card>
