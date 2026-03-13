<script lang="ts">
	import { fakturoidApi, type FakturoidImportResult } from '$lib/api/client';
	import { toastSuccess, toastError } from '$lib/data/toast-state.svelte';

	type Step = 'idle' | 'importing' | 'done';

	let step: Step = $state('idle');
	let result: FakturoidImportResult | null = $state(null);

	let slug = $state('');
	let email = $state('');
	let apiToken = $state('');
	let downloadAttachments = $state(false);

	let submitting = $state(false);

	async function doImport() {
		submitting = true;
		step = 'importing';
		try {
			result = await fakturoidApi.import({
				slug,
				email,
				api_token: apiToken,
				download_attachments: downloadAttachments
			});
			step = 'done';
			const total = result.contacts_created + result.invoices_created + result.expenses_created;
			if (total > 0) {
				toastSuccess(`Import dokončen: ${total} záznamů vytvořeno`);
			}
			if (result.errors.length > 0) {
				toastError(`${result.errors.length} chyb při importu`);
			}
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Chyba při importu');
			step = 'idle';
		} finally {
			submitting = false;
		}
	}
</script>

<div class="mx-auto max-w-2xl">
	<div class="mb-6">
		<h1 class="text-xl font-semibold text-primary">Import z Fakturoidu</h1>
		<p class="mt-1 text-sm text-secondary">
			Přeneste kontakty, faktury a náklady z Fakturoidu do ZFaktury.
		</p>
	</div>

	{#if step === 'idle' || step === 'importing'}
		<form
			onsubmit={(e) => {
				e.preventDefault();
				doImport();
			}}
			class="rounded-lg border border-border bg-surface p-6 space-y-4"
		>
			<div>
				<label for="slug" class="block text-sm font-medium text-primary">Slug účtu</label>
				<input
					id="slug"
					type="text"
					bind:value={slug}
					required
					placeholder="vas-ucet"
					disabled={submitting}
					class="mt-1 block w-full rounded-md border border-border bg-hover px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent disabled:opacity-50"
				/>
				<p class="mt-1 text-xs text-muted">
					Slug z URL vašeho Fakturoid účtu (app.fakturoid.cz/api/v3/accounts/<strong>slug</strong>)
				</p>
			</div>

			<div>
				<label for="email" class="block text-sm font-medium text-primary">Email</label>
				<input
					id="email"
					type="email"
					bind:value={email}
					required
					placeholder="vas@email.cz"
					disabled={submitting}
					class="mt-1 block w-full rounded-md border border-border bg-hover px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent disabled:opacity-50"
				/>
			</div>

			<div>
				<label for="api-token" class="block text-sm font-medium text-primary">API Token</label>
				<input
					id="api-token"
					type="password"
					bind:value={apiToken}
					required
					placeholder="vas-api-token"
					disabled={submitting}
					class="mt-1 block w-full rounded-md border border-border bg-hover px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent disabled:opacity-50"
				/>
				<p class="mt-1 text-xs text-muted">API token najdete v nastavení vašeho Fakturoid účtu.</p>
			</div>

			<div class="flex items-center gap-2">
				<input
					id="download-attachments"
					type="checkbox"
					bind:checked={downloadAttachments}
					disabled={submitting}
					class="h-4 w-4 rounded border-border text-accent focus:ring-accent"
				/>
				<label for="download-attachments" class="text-sm text-primary">
					Stáhnout přílohy
				</label>
			</div>
			<p class="text-xs text-muted -mt-2">
				Stáhne přílohy (účtenky, smlouvy) u nově importovaných faktur a nákladů. Zvyšuje počet API požadavků.
			</p>

			<button
				type="submit"
				disabled={submitting}
				class="w-full rounded-md bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
			>
				{#if submitting}
					<span class="flex items-center justify-center gap-2" role="status">
						<svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="4"
							></circle>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
							></path>
						</svg>
						Stahování a import dat z Fakturoidu...
						<span class="sr-only">Načítání...</span>
					</span>
				{:else}
					Importovat
				{/if}
			</button>
		</form>
	{:else if step === 'done' && result}
		<div class="rounded-lg border border-border bg-surface p-6">
			<h2 class="text-base font-medium text-primary mb-4">Import dokončen</h2>
			<div class="grid grid-cols-3 gap-4 mb-4">
				<div class="rounded-md bg-hover p-3">
					<p class="text-xs text-muted uppercase">Kontakty</p>
					<p class="text-lg font-semibold text-primary">{result.contacts_created}</p>
					<p class="text-xs text-muted">vytvořeno, {result.contacts_skipped} přeskočeno</p>
				</div>
				<div class="rounded-md bg-hover p-3">
					<p class="text-xs text-muted uppercase">Faktury</p>
					<p class="text-lg font-semibold text-primary">{result.invoices_created}</p>
					<p class="text-xs text-muted">vytvořeno, {result.invoices_skipped} přeskočeno</p>
				</div>
				<div class="rounded-md bg-hover p-3">
					<p class="text-xs text-muted uppercase">Náklady</p>
					<p class="text-lg font-semibold text-primary">{result.expenses_created}</p>
					<p class="text-xs text-muted">vytvořeno, {result.expenses_skipped} přeskočeno</p>
				</div>
			</div>
			{#if result.attachments_downloaded > 0 || result.attachments_skipped > 0}
				<div class="rounded-md bg-hover p-3 col-span-3">
					<p class="text-xs text-muted uppercase">Přílohy</p>
					<p class="text-lg font-semibold text-primary">{result.attachments_downloaded}</p>
					<p class="text-xs text-muted">staženo, {result.attachments_skipped} přeskočeno</p>
				</div>
			{/if}
			{#if result.errors.length > 0}
				<div class="rounded-md bg-red-500/10 p-3" role="alert">
					<p class="text-sm font-medium text-red-400 mb-1">Chyby ({result.errors.length}):</p>
					<ul class="list-disc list-inside text-xs text-red-300 space-y-0.5">
						{#each result.errors as err}
							<li>{err}</li>
						{/each}
					</ul>
				</div>
			{/if}
			<div class="mt-4">
				<button
					onclick={() => {
						step = 'idle';
						result = null;
						slug = '';
						email = '';
						apiToken = '';
						downloadAttachments = false;
					}}
					class="rounded-md border border-border px-4 py-2 text-sm font-medium text-secondary hover:bg-hover transition-colors"
				>
					Nový import
				</button>
			</div>
		</div>
	{/if}
</div>
