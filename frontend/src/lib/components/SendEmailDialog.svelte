<script lang="ts">
	import { invoicesApi } from '$lib/api/client';

	interface Props {
		invoiceId: number;
		invoiceNumber: string;
		customerEmail?: string;
		onclose: () => void;
		onsuccess: () => void;
	}

	let { invoiceId, invoiceNumber, customerEmail = '', onclose, onsuccess }: Props = $props();

	// svelte-ignore state_referenced_locally
	let to = $state(customerEmail);
	// svelte-ignore state_referenced_locally
	let subject = $state(`Faktura ${invoiceNumber}`);
	// svelte-ignore state_referenced_locally
	let body = $state(`Dobrý den,\n\nv příloze zasíláme fakturu ${invoiceNumber}.\n\nS pozdravem`);
	let error = $state('');
	let sending = $state(false);

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		sending = true;

		try {
			await invoicesApi.sendEmail(invoiceId, { to, subject, body });
			onsuccess();
		} catch (err: unknown) {
			if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Nepodařilo se odeslat email';
			}
		} finally {
			sending = false;
		}
	}

	function handleBackdropClick() {
		if (!sending) {
			onclose();
		}
	}
</script>

<!-- Backdrop -->
<div
	class="fixed inset-0 z-50 bg-overlay"
	role="presentation"
	onclick={handleBackdropClick}
></div>

<!-- Dialog -->
<div
	class="fixed inset-0 z-50 flex items-center justify-center p-4 pointer-events-none"
>
	<div
		class="pointer-events-auto w-full max-w-lg rounded-xl border border-border bg-surface shadow-xl shadow-black/30"
		role="dialog"
		aria-modal="true"
		aria-labelledby="send-email-title"
	>
		<div class="border-b border-border px-6 py-4">
			<h2 id="send-email-title" class="text-lg font-semibold text-primary">
				Odeslat fakturu emailem
			</h2>
		</div>

		<form onsubmit={handleSubmit} class="px-6 py-4 space-y-4">
			{#if error}
				<div role="alert" class="rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-400">
					{error}
				</div>
			{/if}

			<div>
				<label for="email-to" class="mb-1.5 block text-sm font-medium text-secondary">
					Příjemce
				</label>
				<input
					id="email-to"
					type="email"
					bind:value={to}
					required
					class="w-full rounded-lg border border-border bg-surface px-4 py-2.5 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
					placeholder="email@example.com"
				/>
			</div>

			<div>
				<label for="email-subject" class="mb-1.5 block text-sm font-medium text-secondary">
					Předmět
				</label>
				<input
					id="email-subject"
					type="text"
					bind:value={subject}
					required
					class="w-full rounded-lg border border-border bg-surface px-4 py-2.5 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
				/>
			</div>

			<div>
				<label for="email-body" class="mb-1.5 block text-sm font-medium text-secondary">
					Text emailu
				</label>
				<textarea
					id="email-body"
					bind:value={body}
					required
					rows="6"
					class="w-full rounded-lg border border-border bg-surface px-4 py-2.5 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none resize-y"
				></textarea>
			</div>

			<div class="flex justify-end gap-3 pt-2">
				<button
					type="button"
					onclick={onclose}
					disabled={sending}
					class="rounded-lg bg-elevated px-4 py-2.5 text-sm font-medium text-secondary hover:bg-hover transition-colors disabled:opacity-50"
				>
					Zrušit
				</button>
				<button
					type="submit"
					disabled={sending}
					class="rounded-lg bg-accent px-4 py-2.5 text-sm font-medium text-on-accent hover:bg-accent-hover transition-colors disabled:opacity-50"
				>
					{#if sending}
						<span role="status" class="inline-flex items-center gap-2">
							<svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
							</svg>
							Odesílám...
							<span class="sr-only">Načítání...</span>
						</span>
					{:else}
						Odeslat
					{/if}
				</button>
			</div>
		</form>
	</div>
</div>
