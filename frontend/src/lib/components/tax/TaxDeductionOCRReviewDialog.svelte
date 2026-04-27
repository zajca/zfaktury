<script lang="ts" module>
	/**
	 * Values returned from the OCR review dialog after the user confirms.
	 *
	 * `claimed_amount_czk` is the amount in crowns (as entered by the user).
	 * Conversion to halere is the caller's responsibility (`toHalere`).
	 */
	export interface ConfirmedValues {
		category: string;
		description: string;
		provider_name: string;
		provider_ico: string;
		contract_number: string;
		document_date: string;
		purpose: string;
		claimed_amount_czk: number;
	}
</script>

<script lang="ts">
	import type { TaxExtractionResult } from '$lib/api/client';
	import Button from '$lib/ui/Button.svelte';
	import Input from '$lib/ui/Input.svelte';
	import Select from '$lib/ui/Select.svelte';
	import Textarea from '$lib/ui/Textarea.svelte';
	import { categoryLabels } from '$lib/utils/deduction';

	interface Props {
		result: TaxExtractionResult;
		onConfirm: (values: ConfirmedValues) => void;
		onCancel: () => void;
		saving?: boolean;
	}

	let { result, onConfirm, onCancel, saving = false }: Props = $props();

	// Initial amount display is converted from halere when available, otherwise
	// from the CZK helper, so we don't lose precision on sub-crown values.
	function initialAmount(r: TaxExtractionResult): number {
		if (r.amount_halere && r.amount_halere > 0) {
			return r.amount_halere / 100;
		}
		return r.amount_czk ?? 0;
	}

	// Prefer the description suggestion, fall back to contract number so the
	// description field is never blank when we have any useful reference.
	function initialDescription(r: TaxExtractionResult): string {
		return r.description_suggestion?.trim() || r.contract_number?.trim() || '';
	}

	// Editable mirrors of every field. Intentionally capture initial values so
	// subsequent prop changes don't clobber the user's edits.
	let category = $state(result.category || 'mortgage');
	let description = $state(initialDescription(result));
	let providerName = $state(result.provider_name ?? '');
	let providerIco = $state(result.provider_ico ?? '');
	let contractNumber = $state(result.contract_number ?? '');
	let documentDate = $state(result.document_date ?? '');
	let purpose = $state(result.purpose ?? '');
	let amountCzk = $state(initialAmount(result));

	let confidencePercent = $derived(Math.round((result.confidence ?? 0) * 100));
	let confidenceColor = $derived(
		result.confidence >= 0.8
			? 'bg-success-bg text-success'
			: result.confidence >= 0.5
				? 'bg-warning-bg text-warning'
				: 'bg-danger-bg text-danger'
	);

	let needsPurpose = $derived(category === 'donation' || category === 'union_dues');

	function handleConfirm() {
		onConfirm({
			category,
			description,
			provider_name: providerName,
			provider_ico: providerIco,
			contract_number: contractNumber,
			document_date: documentDate,
			purpose: needsPurpose ? purpose : '',
			claimed_amount_czk: Number(amountCzk) || 0
		});
	}

	function handleBackdropClick() {
		if (!saving) onCancel();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && !saving) {
			onCancel();
		}
	}
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="fixed inset-0 z-50 flex items-center justify-center" onkeydown={handleKeydown}>
	<!-- Backdrop -->
	<div class="fixed inset-0 bg-overlay" role="presentation" onclick={handleBackdropClick}></div>

	<!-- Dialog. tabindex=-1 + autofocus so keyboard users (Escape, Tab) reach
		 it immediately on open; otherwise focus stays on the page below. -->
	<div
		class="relative z-50 max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-xl border border-border bg-surface p-6 shadow-xl"
		role="dialog"
		aria-modal="true"
		aria-labelledby="tax-ocr-review-title"
		tabindex="-1"
		{@attach (node: HTMLElement) => {
			node.focus();
		}}
	>
		<div class="mb-6 flex items-center justify-between">
			<h2 id="tax-ocr-review-title" class="text-lg font-semibold text-primary">
				OCR - Kontrola dokladu
			</h2>
			<span
				class="rounded px-2 py-1 text-xs font-medium {confidenceColor}"
				data-testid="confidence"
			>
				Spolehlivost: {confidencePercent} %
			</span>
		</div>

		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleConfirm();
			}}
		>
			<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
				<!-- Category -->
				<div class="md:col-span-2">
					<label for="tax-ocr-category" class="mb-1 block text-sm font-medium text-secondary">
						Kategorie
					</label>
					<Select
						id="tax-ocr-category"
						value={category}
						onchange={(e: Event) => {
							category = (e.currentTarget as HTMLSelectElement).value;
						}}
					>
						{#each Object.entries(categoryLabels) as [key, label]}
							<option value={key}>{label}</option>
						{/each}
					</Select>
				</div>

				<!-- Description -->
				<div class="md:col-span-2">
					<label for="tax-ocr-description" class="mb-1 block text-sm font-medium text-secondary">
						Popis
					</label>
					<Input
						id="tax-ocr-description"
						type="text"
						value={description}
						oninput={(e: Event) => {
							description = (e.currentTarget as HTMLInputElement).value;
						}}
						placeholder="Název/číslo smlouvy"
					/>
				</div>

				<!-- Provider name -->
				<div>
					<label for="tax-ocr-provider-name" class="mb-1 block text-sm font-medium text-secondary">
						Poskytovatel
					</label>
					<Input
						id="tax-ocr-provider-name"
						type="text"
						value={providerName}
						oninput={(e: Event) => {
							providerName = (e.currentTarget as HTMLInputElement).value;
						}}
					/>
				</div>

				<!-- Provider ICO -->
				<div>
					<label for="tax-ocr-provider-ico" class="mb-1 block text-sm font-medium text-secondary">
						IČO poskytovatele
					</label>
					<Input
						id="tax-ocr-provider-ico"
						type="text"
						value={providerIco}
						oninput={(e: Event) => {
							providerIco = (e.currentTarget as HTMLInputElement).value;
						}}
					/>
				</div>

				<!-- Contract number -->
				<div>
					<label
						for="tax-ocr-contract-number"
						class="mb-1 block text-sm font-medium text-secondary"
					>
						Číslo smlouvy
					</label>
					<Input
						id="tax-ocr-contract-number"
						type="text"
						value={contractNumber}
						oninput={(e: Event) => {
							contractNumber = (e.currentTarget as HTMLInputElement).value;
						}}
					/>
				</div>

				<!-- Document date -->
				<div>
					<label for="tax-ocr-document-date" class="mb-1 block text-sm font-medium text-secondary">
						Datum dokladu
					</label>
					<Input
						id="tax-ocr-document-date"
						type="date"
						value={documentDate}
						oninput={(e: Event) => {
							documentDate = (e.currentTarget as HTMLInputElement).value;
						}}
					/>
				</div>

				<!-- Claimed amount (CZK) -->
				<div>
					<label for="tax-ocr-amount" class="mb-1 block text-sm font-medium text-secondary">
						Uplatňovaná částka (CZK)
					</label>
					<Input
						id="tax-ocr-amount"
						type="number"
						value={amountCzk}
						oninput={(e: Event) => {
							amountCzk = Number((e.currentTarget as HTMLInputElement).value);
						}}
						step="0.01"
						min="0"
					/>
				</div>

				<!-- Purpose -- only for donations and union dues -->
				{#if needsPurpose}
					<div class="md:col-span-2">
						<label for="tax-ocr-purpose" class="mb-1 block text-sm font-medium text-secondary">
							Účel
						</label>
						<Textarea id="tax-ocr-purpose" bind:value={purpose} rows={2} />
					</div>
				{/if}
			</div>

			<!-- Actions -->
			<div class="mt-6 flex justify-end gap-3 border-t border-border pt-4">
				<Button variant="ghost" onclick={onCancel} disabled={saving}>Zrušit</Button>
				<Button variant="primary" type="submit" disabled={saving}>Uložit</Button>
			</div>
		</form>
	</div>
</div>
