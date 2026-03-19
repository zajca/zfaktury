<script lang="ts">
	import { nativeDownload } from '$lib/actions/download';
	import type { TaxDeduction, TaxConstants } from '$lib/api/client';
	import { taxDeductionsApi } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import Input from '$lib/ui/Input.svelte';
	import Select from '$lib/ui/Select.svelte';

	let {
		deductions,
		showDeductionForm = $bindable(),
		editingDeductionId = $bindable(),
		deductionCategory = $bindable(),
		deductionDescription = $bindable(),
		deductionAmount = $bindable(),
		taxConstants,
		saving,
		onSave,
		onDelete,
		onEdit,
		onReset,
		onUploadDocument,
		onExtractAmount,
		onDeleteDocument
	}: {
		deductions: TaxDeduction[];
		showDeductionForm: boolean;
		editingDeductionId: number | null;
		deductionCategory: string;
		deductionDescription: string;
		deductionAmount: number;
		taxConstants: TaxConstants | null;
		saving: boolean;
		onSave: () => void;
		onDelete: (id: number) => void;
		onEdit: (ded: TaxDeduction) => void;
		onReset: () => void;
		onUploadDocument: (deductionId: number) => void;
		onExtractAmount: (docId: number) => void;
		onDeleteDocument: (docId: number) => void;
	} = $props();

	const categoryLabels: Record<string, string> = {
		mortgage: 'Úroky z hypotéky',
		life_insurance: 'Životní pojištění',
		pension: 'Penzijní spoření',
		donation: 'Dary',
		union_dues: 'Odborové příspěvky'
	};
</script>

<Card>
	<div class="flex items-center justify-between">
		<h2 class="text-base font-semibold text-primary">
			Nezdanitelné části (odpočty) <HelpTip topic="nezdanitelne-odpocty" {taxConstants} />
		</h2>
		<Button
			variant="primary"
			size="sm"
			onclick={() => {
				onReset();
				showDeductionForm = true;
			}}>Přidat odpočet</Button
		>
	</div>
	{#if deductions.length > 0}
		<div class="mt-4 space-y-3">
			{#each deductions as ded (ded.id)}
				<div class="rounded-lg border border-border p-3">
					<div class="flex items-center justify-between">
						<div>
							<span class="text-xs font-medium uppercase text-accent"
								>{categoryLabels[ded.category] ?? ded.category}</span
							>
							{#if ded.description}
								<span class="ml-2 text-sm text-tertiary">{ded.description}</span>
							{/if}
						</div>
						<div class="flex gap-1">
							<Button variant="ghost" size="sm" onclick={() => onEdit(ded)}>Upravit</Button>
							<Button variant="danger" size="sm" onclick={() => onDelete(ded.id)}>Smazat</Button>
						</div>
					</div>
					<div class="mt-2 flex gap-6 text-sm">
						<span class="text-tertiary"
							>Uplatňováno: <strong class="text-primary">{formatCZK(ded.claimed_amount)}</strong
							></span
						>
						{#if ded.max_amount > 0}
							<span class="text-tertiary">Max: {formatCZK(ded.max_amount)}</span>
						{/if}
						<span class="text-tertiary"
							>Uznáno: <strong class="text-primary">{formatCZK(ded.allowed_amount)}</strong></span
						>
					</div>
					<!-- Documents -->
					{#if ded.documents && ded.documents.length > 0}
						<div class="mt-2 space-y-1">
							{#each ded.documents as doc (doc.id)}
								<div class="flex items-center gap-2 text-xs text-tertiary">
									<a
										href={taxDeductionsApi.downloadDocument(doc.id)}
										class="text-accent hover:underline"
										target="_blank"
										use:nativeDownload={doc.filename}>{doc.filename}</a
									>
									{#if doc.extracted_amount > 0}
										<span class="rounded bg-success-bg px-1.5 py-0.5 text-success">
											Extrahováno: {formatCZK(doc.extracted_amount)} ({Math.round(
												doc.confidence * 100
											)}%)
										</span>
									{/if}
									<Button
										variant="ghost"
										size="sm"
										onclick={() => onExtractAmount(doc.id)}
										disabled={saving}>Extrahovat</Button
									>
									<Button
										variant="danger"
										size="sm"
										onclick={() => onDeleteDocument(doc.id)}
										disabled={saving}>Smazat</Button
									>
								</div>
							{/each}
						</div>
					{/if}
					<div class="mt-2">
						<Button
							variant="secondary"
							size="sm"
							onclick={() => onUploadDocument(ded.id)}
							disabled={saving}>Nahrát doklad</Button
						>
					</div>
				</div>
			{/each}
		</div>
	{:else}
		<p class="mt-2 text-sm text-tertiary">Žádné odpočty</p>
	{/if}
	{#if showDeductionForm}
		<div class="mt-4 rounded-lg border border-border-subtle bg-elevated p-4">
			<h3 class="text-sm font-medium text-primary">
				{editingDeductionId ? 'Upravit odpočet' : 'Přidat odpočet'}
			</h3>
			<div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2">
				<div>
					<span class="text-xs text-tertiary">Kategorie</span>
					<Select
						value={deductionCategory}
						onchange={(e: Event) => {
							deductionCategory = (e.currentTarget as HTMLSelectElement).value;
						}}
					>
						{#each Object.entries(categoryLabels) as [key, label]}
							<option value={key}>{label}</option>
						{/each}
					</Select>
				</div>
				<div>
					<span class="text-xs text-tertiary">Popis</span>
					<Input
						value={deductionDescription}
						oninput={(e: Event) => {
							deductionDescription = (e.currentTarget as HTMLInputElement).value;
						}}
						placeholder="Název/číslo smlouvy"
					/>
				</div>
				<div>
					<span class="text-xs text-tertiary">Částka (CZK)</span>
					<Input
						type="number"
						value={deductionAmount}
						oninput={(e: Event) => {
							deductionAmount = Number((e.currentTarget as HTMLInputElement).value);
						}}
						step="0.01"
					/>
				</div>
			</div>
			<div class="mt-3 flex gap-2">
				<Button variant="primary" size="sm" onclick={onSave} disabled={saving}>Uložit</Button>
				<Button variant="ghost" size="sm" onclick={onReset}>Zrušit</Button>
			</div>
		</div>
	{/if}
</Card>
