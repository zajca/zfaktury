<script lang="ts">
	import { invoicesApi, type Invoice } from '$lib/api/client';

	interface Props {
		invoiceId: number;
		onclose: () => void;
		onsuccess: (newInvoice: Invoice) => void;
	}

	let { invoiceId, onclose, onsuccess }: Props = $props();

	let reason = $state('');
	let error = $state<string | null>(null);
	let submitting = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();

		if (!reason.trim()) {
			error = 'Důvod dobropisu je povinný';
			return;
		}

		submitting = true;
		error = null;

		try {
			const invoice = await invoicesApi.createCreditNote(invoiceId, {
				reason: reason.trim()
			});
			onsuccess(invoice);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se vytvořit dobropis';
		} finally {
			submitting = false;
		}
	}
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="fixed inset-0 z-50 flex items-center justify-center"
>
	<!-- Backdrop -->
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div
		class="absolute inset-0 bg-overlay"
		role="presentation"
		onclick={onclose}
	></div>

	<!-- Dialog -->
	<div class="relative z-10 w-full max-w-md rounded-lg border border-border bg-surface p-6 shadow-xl">
		<h2 class="text-lg font-semibold text-primary">Vytvořit dobropis</h2>
		<p class="mt-1 text-sm text-secondary">
			Dobropis bude vytvořen k faktuře a automaticky propojen.
		</p>

		{#if error}
			<div role="alert" class="mt-4 rounded-lg border border-danger/20 bg-danger-bg px-4 py-3 text-sm text-danger">
				{error}
			</div>
		{/if}

		<form onsubmit={handleSubmit} class="mt-4">
			<label for="credit-note-reason" class="block text-sm font-medium text-secondary">
				Důvod dobropisu
			</label>
			<input
				id="credit-note-reason"
				type="text"
				required
				bind:value={reason}
				placeholder="Např. reklamace, sleva, oprava chyby..."
				class="mt-1.5 w-full rounded-lg border border-border bg-surface px-4 py-2.5 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
			/>

			<div class="mt-6 flex justify-end gap-3">
				<button
					type="button"
					onclick={onclose}
					class="rounded-lg bg-elevated px-4 py-2 text-sm font-medium text-secondary hover:bg-hover transition-colors"
				>
					Zrušit
				</button>
				<button
					type="submit"
					disabled={submitting}
					class="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-on-accent hover:bg-accent-hover transition-colors disabled:opacity-50"
				>
					{#if submitting}
						<span role="status" class="flex items-center gap-2">
							<svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
							</svg>
							Vytvářím...
							<span class="sr-only">Načítání...</span>
						</span>
					{:else}
						Vytvořit dobropis
					{/if}
				</button>
			</div>
		</form>
	</div>
</div>
