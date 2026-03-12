<script lang="ts">
	import { documentsApi, type ExpenseDocument } from '$lib/api/client';

	interface Props {
		expenseId: number;
		onuploaded?: (doc: ExpenseDocument) => void;
	}

	let { expenseId, onuploaded }: Props = $props();

	const MAX_SIZE = 20 * 1024 * 1024; // 20MB
	const ACCEPTED_TYPES = ['application/pdf', 'image/jpeg', 'image/png'];
	const ACCEPTED_EXTENSIONS = '.pdf,.jpg,.jpeg,.png';

	let uploading = $state(false);
	let error = $state<string | null>(null);
	let dragOver = $state(false);
	let fileInput: HTMLInputElement | undefined = $state();

	function validateFile(file: File): string | null {
		if (!ACCEPTED_TYPES.includes(file.type)) {
			return 'Nepodporovaný typ souboru. Povolené jsou PDF, JPG a PNG.';
		}
		if (file.size > MAX_SIZE) {
			return 'Soubor je příliš velký. Maximální velikost je 20 MB.';
		}
		return null;
	}

	async function uploadFile(file: File) {
		const validationError = validateFile(file);
		if (validationError) {
			error = validationError;
			return;
		}

		error = null;
		uploading = true;

		try {
			const doc = await documentsApi.upload(expenseId, file);
			onuploaded?.(doc);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nahrávání se nezdařilo.';
		} finally {
			uploading = false;
			if (fileInput) {
				fileInput.value = '';
			}
		}
	}

	function handleFileChange(e: Event) {
		const target = e.target as HTMLInputElement;
		const file = target.files?.[0];
		if (file) {
			uploadFile(file);
		}
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		dragOver = false;
		const file = e.dataTransfer?.files?.[0];
		if (file) {
			uploadFile(file);
		}
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		dragOver = true;
	}

	function handleDragLeave() {
		dragOver = false;
	}

	function handleClick() {
		fileInput?.click();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			fileInput?.click();
		}
	}
</script>

<div>
	<input
		bind:this={fileInput}
		type="file"
		accept={ACCEPTED_EXTENSIONS}
		onchange={handleFileChange}
		class="hidden"
		data-testid="file-input"
	/>

	<button
		type="button"
		onclick={handleClick}
		onkeydown={handleKeydown}
		ondrop={handleDrop}
		ondragover={handleDragOver}
		ondragleave={handleDragLeave}
		disabled={uploading}
		class="w-full rounded-lg border-2 border-dashed p-6 text-center transition-colors
			{dragOver
			? 'border-accent bg-accent/5'
			: 'border-border hover:border-secondary hover:bg-hover'}
			{uploading ? 'cursor-wait opacity-60' : 'cursor-pointer'}"
	>
		{#if uploading}
			<div role="status">
				<svg
					class="mx-auto mb-2 h-8 w-8 animate-spin text-accent"
					viewBox="0 0 24 24"
					fill="none"
				>
					<circle
						class="opacity-25"
						cx="12"
						cy="12"
						r="10"
						stroke="currentColor"
						stroke-width="4"
					/>
					<path
						class="opacity-75"
						fill="currentColor"
						d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
					/>
				</svg>
				<p class="text-sm text-muted">Nahrávám...</p>
				<span class="sr-only">Načítání...</span>
			</div>
		{:else}
			<svg
				class="mx-auto mb-2 h-8 w-8 text-muted"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				stroke-width="1.5"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5"
				/>
			</svg>
			<p class="text-sm text-secondary">Přetáhněte soubor nebo klikněte pro výběr</p>
			<p class="mt-1 text-xs text-muted">PDF, JPG, PNG — max 20 MB</p>
		{/if}
	</button>

	{#if error}
		<div role="alert" class="mt-2 rounded-lg border border-danger/20 bg-danger-bg px-3 py-2 text-sm text-danger">
			{error}
		</div>
	{/if}
</div>
