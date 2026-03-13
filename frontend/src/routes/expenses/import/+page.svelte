<script lang="ts">
	import { goto } from '$app/navigation';
	import { importApi, expensesApi, type OCRResult } from '$lib/api/client';
	import { toastSuccess, toastError } from '$lib/data/toast-state.svelte';
	import OCRReviewDialog from '$lib/components/OCRReviewDialog.svelte';
	import Button from '$lib/ui/Button.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import Card from '$lib/ui/Card.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';

	let pageState: 'idle' | 'processing' | 'review' | 'saving' | 'done' = $state('idle');
	let dragOver = $state(false);

	let expenseId = $state<number | null>(null);
	let ocrResult = $state<OCRResult | null>(null);

	const acceptedTypes = ['application/pdf', 'image/jpeg', 'image/png', 'image/webp'];

	function validateFile(file: File): string | null {
		if (!acceptedTypes.includes(file.type)) {
			return 'Nepodporovaný formát souboru. Povolené: PDF, JPG, PNG, WebP.';
		}
		if (file.size > 20 * 1024 * 1024) {
			return 'Soubor je příliš velký. Maximum je 20 MB.';
		}
		return null;
	}

	async function processFile(file: File) {
		const validationError = validateFile(file);
		if (validationError) {
			toastError(validationError);
			return;
		}

		pageState = 'processing';

		try {
			const result = await importApi.importDocument(file);
			expenseId = result.expense.id;

			if (result.ocr) {
				ocrResult = result.ocr;
				pageState = 'review';
			} else {
				// No OCR -- show info and redirect to expense for manual editing
				pageState = 'done';
				setTimeout(() => {
					goto(`/expenses/${result.expense.id}`);
				}, 3000);
			}
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Import se nezdaril');
			pageState = 'idle';
		}
	}

	function handleFileInput(e: Event) {
		const input = e.target as HTMLInputElement;
		const file = input.files?.[0];
		if (file) {
			processFile(file);
		}
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		dragOver = false;
		const file = e.dataTransfer?.files[0];
		if (file) {
			processFile(file);
		}
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		dragOver = true;
	}

	function handleDragLeave() {
		dragOver = false;
	}

	function handleOCRClose() {
		// User cancelled OCR review -- redirect to expense for manual editing
		if (expenseId) {
			goto(`/expenses/${expenseId}`);
		}
	}

	async function handleOCRConfirm(data: OCRResult) {
		if (!expenseId) return;
		pageState = 'saving';

		try {
			await expensesApi.update(expenseId, {
				description: data.description || data.vendor_name || undefined,
				expense_number: data.invoice_number || undefined,
				amount: data.total_amount || undefined,
				vat_amount: data.vat_amount || undefined,
				vat_rate_percent: data.vat_rate_percent || undefined,
				currency_code: data.currency_code || undefined,
				issue_date: data.issue_date || undefined
			});
			toastSuccess('Náklad uložen');
			goto(`/expenses/${expenseId}`);
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Ukládání se nezdařilo');
			pageState = 'review';
		}
	}
</script>

<svelte:head>
	<title>Import nákladů - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-3xl">
	<PageHeader
		title="Import z dokladu"
		description="Nahrajte doklad a data se automaticky rozpoznají pomocí OCR"
	>
		{#snippet actions()}
			<Button variant="secondary" href="/expenses">Zpět na náklady</Button>
		{/snippet}
	</PageHeader>

	<p class="mt-2 text-sm text-tertiary">
		Nahrání faktury nebo účtenky (PDF, JPG, PNG, WebP) s automatickým rozpoznáním dat.
		<HelpTip topic="ocr-import" />
	</p>

	{#if pageState === 'idle'}
		<Card class="mt-6">
			<div
				class="flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-12 transition-colors {dragOver
					? 'border-accent bg-accent/5'
					: 'border-border'}"
				role="button"
				tabindex="0"
				ondrop={handleDrop}
				ondragover={handleDragOver}
				ondragleave={handleDragLeave}
				onclick={() => document.getElementById('file-input')?.click()}
				onkeydown={(e) => {
					if (e.key === 'Enter' || e.key === ' ') document.getElementById('file-input')?.click();
				}}
			>
				<svg
					class="h-12 w-12 text-muted mb-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="1.5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 16.5V9.75m0 0l3 3m-3-3l-3 3M6.75 19.5a4.5 4.5 0 01-1.41-8.775 5.25 5.25 0 0110.233-2.33 3 3 0 013.758 3.848A3.752 3.752 0 0118 19.5H6.75z"
					/>
				</svg>
				<p class="text-primary font-medium mb-1">Přetáhni soubor sem</p>
				<p class="text-sm text-muted mb-4">nebo klikni pro výběr</p>
				<p class="text-xs text-muted">PDF, JPG, PNG, WebP (max 20 MB)</p>
				<input
					id="file-input"
					type="file"
					accept=".pdf,.jpg,.jpeg,.png,.webp"
					class="hidden"
					onchange={handleFileInput}
				/>
			</div>
		</Card>
	{/if}

	{#if pageState === 'processing'}
		<Card class="mt-6">
			<LoadingSpinner class="p-12" />
			<p class="text-center text-secondary pb-6">Zpracovávám dokument...</p>
		</Card>
	{/if}

	{#if pageState === 'saving'}
		<Card class="mt-6">
			<LoadingSpinner class="p-12" />
			<p class="text-center text-secondary pb-6">Ukládám data...</p>
		</Card>
	{/if}

	{#if pageState === 'done'}
		<Card class="mt-6">
			<div class="flex flex-col items-center p-8 gap-3">
				<svg
					class="h-10 w-10 text-warning"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="1.5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z"
					/>
				</svg>
				<p class="text-primary font-medium">Dokument nahrán, ale OCR není nastaveno</p>
				<p class="text-sm text-secondary text-center">
					Automatické rozpoznávání dat není k dispozici. Nastavte OCR API klíč v konfiguraci.
					<br />Přesměrovávám na náklad pro manuální vyplnění...
				</p>
				{#if expenseId}
					<Button variant="primary" href="/expenses/{expenseId}" class="mt-2">
						Přejít na náklad
					</Button>
				{/if}
			</div>
		</Card>
	{/if}

	{#if pageState === 'review' && ocrResult}
		<OCRReviewDialog {ocrResult} onclose={handleOCRClose} onconfirm={handleOCRConfirm} />
	{/if}
</div>
