<script lang="ts">
	import type { TaxCreditsSummary, TaxConstants } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import Input from '$lib/ui/Input.svelte';
	import Select from '$lib/ui/Select.svelte';

	let {
		showChildForm = $bindable(),
		editingChildId = $bindable(),
		childName = $bindable(),
		childBirthNumber = $bindable(),
		childOrder = $bindable(),
		childMonths = $bindable(),
		childZtp = $bindable(),
		summary,
		taxConstants,
		saving,
		onSave,
		onDelete,
		onEdit,
		onReset
	}: {
		showChildForm: boolean;
		editingChildId: number | null;
		childName: string;
		childBirthNumber: string;
		childOrder: number;
		childMonths: number;
		childZtp: boolean;
		summary: TaxCreditsSummary | null;
		taxConstants: TaxConstants | null;
		saving: boolean;
		onSave: () => void;
		onDelete: (id: number) => void;
		onEdit: (child: {
			id: number;
			child_name: string;
			birth_number: string;
			child_order: number;
			months_claimed: number;
			ztp: boolean;
		}) => void;
		onReset: () => void;
	} = $props();
</script>

<Card>
	<div class="flex items-center justify-between">
		<h2 class="text-base font-semibold text-primary">
			Děti <HelpTip topic="zvyhodneni-na-deti" {taxConstants} />
		</h2>
		<Button
			variant="primary"
			size="sm"
			onclick={() => {
				onReset();
				showChildForm = true;
			}}>Přidat dítě</Button
		>
	</div>
	{#if summary?.children && summary.children.length > 0}
		<div class="mt-4 space-y-2">
			{#each summary.children as child (child.id)}
				<div class="flex items-center justify-between rounded-lg border border-border p-3">
					<div>
						<span class="text-sm font-medium text-primary"
							>{child.child_name || `Dítě ${child.child_order}`}</span
						>
						<span class="ml-2 text-xs text-tertiary">
							{child.child_order}. dítě, {child.months_claimed} mes.
							{#if child.ztp}<span class="text-accent">ZTP</span>{/if}
						</span>
						<span class="ml-2 text-sm font-medium text-primary"
							>{formatCZK(child.credit_amount)}</span
						>
					</div>
					<div class="flex gap-1">
						<Button variant="ghost" size="sm" onclick={() => onEdit(child)}>Upravit</Button>
						<Button variant="danger" size="sm" onclick={() => onDelete(child.id)}>Smazat</Button>
					</div>
				</div>
			{/each}
		</div>
		<div class="mt-2 text-sm text-tertiary">
			Celkem zvýhodnění: <strong class="text-primary"
				>{formatCZK(summary.total_child_benefit)}</strong
			>
		</div>
	{:else}
		<p class="mt-2 text-sm text-tertiary">Žádné děti</p>
	{/if}
	{#if showChildForm}
		<div class="mt-4 rounded-lg border border-border-subtle bg-elevated p-4">
			<h3 class="text-sm font-medium text-primary">
				{editingChildId ? 'Upravit dítě' : 'Přidat dítě'}
			</h3>
			<div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2">
				<div>
					<span class="text-xs text-tertiary">Jméno</span>
					<Input
						value={childName}
						oninput={(e: Event) => {
							childName = (e.currentTarget as HTMLInputElement).value;
						}}
						placeholder="Jméno dítěte"
					/>
				</div>
				<div>
					<span class="text-xs text-tertiary">Rodné číslo</span>
					<Input
						value={childBirthNumber}
						oninput={(e: Event) => {
							childBirthNumber = (e.currentTarget as HTMLInputElement).value;
						}}
						placeholder="000000/0000"
					/>
				</div>
				<div>
					<span class="text-xs text-tertiary">Pořadí</span>
					<Select
						value={childOrder}
						onchange={(e: Event) => {
							childOrder = Number((e.currentTarget as HTMLSelectElement).value);
						}}
					>
						<option value={1}>1. dítě</option>
						<option value={2}>2. dítě</option>
						<option value={3}>3. a další</option>
					</Select>
				</div>
				<div>
					<span class="text-xs text-tertiary"
						>Měsíců <HelpTip topic="mesice-proporcializace" {taxConstants} /></span
					>
					<Select
						value={childMonths}
						onchange={(e: Event) => {
							childMonths = Number((e.currentTarget as HTMLSelectElement).value);
						}}
					>
						{#each Array.from({ length: 12 }, (_, i) => i + 1) as m}
							<option value={m}>{m}</option>
						{/each}
					</Select>
				</div>
				<label class="flex items-center gap-2 text-sm text-primary">
					<input type="checkbox" bind:checked={childZtp} class="rounded border-border" />
					ZTP/P <HelpTip topic="ztpp" {taxConstants} />
				</label>
			</div>
			<div class="mt-3 flex gap-2">
				<Button variant="primary" size="sm" onclick={onSave} disabled={saving}>Uložit</Button>
				<Button variant="ghost" size="sm" onclick={onReset}>Zrušit</Button>
			</div>
		</div>
	{/if}
</Card>
