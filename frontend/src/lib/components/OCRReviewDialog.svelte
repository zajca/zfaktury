<script lang="ts">
	import type { OCRResult } from '$lib/api/client';
	import Button from '$lib/ui/Button.svelte';

	interface Props {
		ocrResult: OCRResult;
		onclose: () => void;
		onconfirm: (data: OCRResult) => void;
	}

	let { ocrResult, onclose, onconfirm }: Props = $props();

	// Editable copies of OCR fields - intentionally capturing initial values
	function getInitial() {
		return $state.snapshot(ocrResult);
	}
	let formData = $state(getInitial());

	let confidencePercent = $derived(Math.round(ocrResult.confidence * 100));
	let confidenceColor = $derived(
		confidencePercent > 80
			? 'text-success'
			: confidencePercent >= 50
				? 'text-warning'
				: 'text-danger'
	);

	function handleConfirm() {
		onconfirm({
			...ocrResult,
			...formData
		});
	}

	function handleBackdropClick() {
		onclose();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onclose();
		}
	}

	const inputClass =
		'w-full rounded-lg border border-border bg-surface px-4 py-2.5 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none';
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="fixed inset-0 z-50 flex items-center justify-center" onkeydown={handleKeydown}>
	<!-- Backdrop -->
	<div class="fixed inset-0 bg-overlay" role="presentation" onclick={handleBackdropClick}></div>

	<!-- Dialog -->
	<div
		class="relative z-50 w-full max-w-2xl bg-surface rounded-xl border border-border shadow-xl p-6 max-h-[90vh] overflow-y-auto"
		role="dialog"
		aria-modal="true"
		aria-labelledby="ocr-review-title"
	>
		<div class="flex items-center justify-between mb-6">
			<h2 id="ocr-review-title" class="text-lg font-semibold text-primary">OCR - Kontrola dat</h2>
			<span class="text-sm font-medium {confidenceColor}" data-testid="confidence">
				Spolehlivost: {confidencePercent} %
			</span>
		</div>

		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleConfirm();
			}}
		>
			<div class="grid grid-cols-2 gap-4">
				<!-- Vendor name -->
				<div>
					<label for="ocr-vendor-name" class="block text-sm font-medium text-secondary mb-1">
						Dodavatel
					</label>
					<input
						id="ocr-vendor-name"
						type="text"
						bind:value={formData.vendor_name}
						class={inputClass}
					/>
				</div>

				<!-- Vendor ICO -->
				<div>
					<label for="ocr-vendor-ico" class="block text-sm font-medium text-secondary mb-1">
						IČO dodavatele
					</label>
					<input
						id="ocr-vendor-ico"
						type="text"
						bind:value={formData.vendor_ico}
						class={inputClass}
					/>
				</div>

				<!-- Invoice number -->
				<div>
					<label for="ocr-invoice-number" class="block text-sm font-medium text-secondary mb-1">
						Číslo faktury
					</label>
					<input
						id="ocr-invoice-number"
						type="text"
						bind:value={formData.invoice_number}
						class={inputClass}
					/>
				</div>

				<!-- Issue date -->
				<div>
					<label for="ocr-issue-date" class="block text-sm font-medium text-secondary mb-1">
						Datum vystavení
					</label>
					<input
						id="ocr-issue-date"
						type="date"
						bind:value={formData.issue_date}
						class={inputClass}
					/>
				</div>

				<!-- Due date -->
				<div>
					<label for="ocr-due-date" class="block text-sm font-medium text-secondary mb-1">
						Datum splatnosti
					</label>
					<input id="ocr-due-date" type="date" bind:value={formData.due_date} class={inputClass} />
				</div>

				<!-- Total amount -->
				<div>
					<label for="ocr-total-amount" class="block text-sm font-medium text-secondary mb-1">
						Celková částka
					</label>
					<input
						id="ocr-total-amount"
						type="number"
						bind:value={formData.total_amount}
						step="0.01"
						class={inputClass}
					/>
				</div>

				<!-- VAT amount -->
				<div>
					<label for="ocr-vat-amount" class="block text-sm font-medium text-secondary mb-1">
						DPH
					</label>
					<input
						id="ocr-vat-amount"
						type="number"
						bind:value={formData.vat_amount}
						step="0.01"
						class={inputClass}
					/>
				</div>

				<!-- Currency -->
				<div>
					<label for="ocr-currency" class="block text-sm font-medium text-secondary mb-1">
						Měna
					</label>
					<input
						id="ocr-currency"
						type="text"
						bind:value={formData.currency_code}
						class={inputClass}
					/>
				</div>
			</div>

			<!-- Actions -->
			<div class="flex justify-end gap-3 mt-6 pt-4 border-t border-border">
				<Button variant="secondary" onclick={onclose}>Zrušit</Button>
				<Button variant="primary" type="submit">Potvrdit a vyplnit</Button>
			</div>
		</form>
	</div>
</div>
