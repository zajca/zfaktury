<script lang="ts">
	import type { TaxCreditsSummary, TaxConstants } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import Input from '$lib/ui/Input.svelte';
	import Select from '$lib/ui/Select.svelte';

	let {
		spouseName = $bindable(),
		spouseBirthNumber = $bindable(),
		spouseIncome = $bindable(),
		spouseZtp = $bindable(),
		spouseMonths = $bindable(),
		showSpouseForm = $bindable(),
		summary,
		taxConstants,
		saving,
		onSave,
		onDelete
	}: {
		spouseName: string;
		spouseBirthNumber: string;
		spouseIncome: number;
		spouseZtp: boolean;
		spouseMonths: number;
		showSpouseForm: boolean;
		summary: TaxCreditsSummary | null;
		taxConstants: TaxConstants | null;
		saving: boolean;
		onSave: () => void;
		onDelete: () => void;
	} = $props();
</script>

<Card>
	<div class="flex items-center justify-between">
		<h2 class="text-base font-semibold text-primary">
			Manžel/ka <HelpTip topic="sleva-na-manzela" {taxConstants} />
		</h2>
		{#if !showSpouseForm}
			<Button variant="primary" size="sm" onclick={() => (showSpouseForm = true)}>Přidat</Button>
		{/if}
	</div>
	{#if showSpouseForm}
		<div class="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
			<div>
				<span class="text-xs text-tertiary">Jméno</span>
				<Input
					value={spouseName}
					oninput={(e: Event) => {
						spouseName = (e.currentTarget as HTMLInputElement).value;
					}}
					placeholder="Jméno a příjmení"
				/>
			</div>
			<div>
				<span class="text-xs text-tertiary">Rodné číslo</span>
				<Input
					value={spouseBirthNumber}
					oninput={(e: Event) => {
						spouseBirthNumber = (e.currentTarget as HTMLInputElement).value;
					}}
					placeholder="000000/0000"
				/>
			</div>
			<div>
				<span class="text-xs text-tertiary">Roční příjem (CZK)</span>
				<Input
					type="number"
					value={spouseIncome}
					oninput={(e: Event) => {
						spouseIncome = Number((e.currentTarget as HTMLInputElement).value);
					}}
					step="1"
				/>
			</div>
			<div>
				<span class="text-xs text-tertiary"
					>Měsíců <HelpTip topic="mesice-proporcializace" {taxConstants} /></span
				>
				<Select
					value={spouseMonths}
					onchange={(e: Event) => {
						spouseMonths = Number((e.currentTarget as HTMLSelectElement).value);
					}}
				>
					{#each Array.from({ length: 12 }, (_, i) => i + 1) as m}
						<option value={m}>{m}</option>
					{/each}
				</Select>
			</div>
			<label class="flex items-center gap-2 text-sm text-primary">
				<input type="checkbox" bind:checked={spouseZtp} class="rounded border-border" />
				ZTP/P <HelpTip topic="ztpp" {taxConstants} />
			</label>
		</div>
		{#if summary?.spouse}
			<div class="mt-3 text-sm text-tertiary">
				Sleva: <strong class="text-primary">{formatCZK(summary.spouse.credit_amount)}</strong>
				{#if spouseIncome >= 68000}
					<span class="ml-2 text-warning">(příjem >= 68 000 CZK, sleva se neuplatní)</span>
				{/if}
			</div>
		{/if}
		<div class="mt-4 flex gap-2">
			<Button variant="primary" size="sm" onclick={onSave} disabled={saving}>Uložit</Button>
			<Button variant="danger" size="sm" onclick={onDelete} disabled={saving}>Odebrat</Button>
		</div>
	{:else}
		<p class="mt-2 text-sm text-tertiary">Neuplatňováno</p>
	{/if}
</Card>
