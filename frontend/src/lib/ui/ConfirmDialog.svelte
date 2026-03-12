<script lang="ts">
	import { browser } from '$app/environment';
	import Button from './Button.svelte';

	interface Props {
		open: boolean;
		title: string;
		message: string;
		confirmLabel?: string;
		cancelLabel?: string;
		variant?: 'danger' | 'warning' | 'default';
		loading?: boolean;
		onconfirm: () => void;
		oncancel: () => void;
	}

	let {
		open = $bindable(),
		title,
		message,
		confirmLabel = 'Potvrdit',
		cancelLabel = 'Zrušit',
		variant = 'danger',
		loading = false,
		onconfirm,
		oncancel
	}: Props = $props();

	let dialogEl: HTMLDivElement | undefined = $state();

	function handleKeydown(event: KeyboardEvent) {
		if (!open) return;

		if (event.key === 'Escape') {
			event.preventDefault();
			oncancel();
			return;
		}

		if (event.key === 'Tab' && dialogEl) {
			const focusable = dialogEl.querySelectorAll<HTMLElement>(
				'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
			);
			if (focusable.length === 0) return;

			const first = focusable[0];
			const last = focusable[focusable.length - 1];

			if (event.shiftKey) {
				if (document.activeElement === first) {
					event.preventDefault();
					last.focus();
				}
			} else {
				if (document.activeElement === last) {
					event.preventDefault();
					first.focus();
				}
			}
		}
	}

	$effect(() => {
		if (open && browser && dialogEl) {
			const firstBtn = dialogEl.querySelector<HTMLElement>('button');
			firstBtn?.focus();
		}
	});

	const confirmVariant = $derived(
		variant === 'danger'
			? ('danger' as const)
			: variant === 'warning'
				? ('secondary' as const)
				: ('primary' as const)
	);
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<div class="fixed inset-0 z-50 bg-black/60" role="presentation" onclick={oncancel}></div>
	<div
		bind:this={dialogEl}
		class="fixed inset-0 z-50 flex items-center justify-center p-4"
		role="alertdialog"
		aria-modal="true"
		aria-labelledby="confirm-dialog-title"
		aria-describedby="confirm-dialog-message"
	>
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
		<div
			class="w-full max-w-md rounded-xl border border-border bg-surface p-6 shadow-xl"
			onclick={(e) => e.stopPropagation()}
		>
			<h2 id="confirm-dialog-title" class="text-lg font-semibold text-primary">{title}</h2>
			<p id="confirm-dialog-message" class="mt-2 text-sm text-secondary">{message}</p>
			<div class="mt-6 flex justify-end gap-3">
				<Button variant="secondary" onclick={oncancel} disabled={loading}>{cancelLabel}</Button>
				<Button variant={confirmVariant} onclick={onconfirm} disabled={loading}>
					{#if loading}
						Zpracovávám...
					{:else}
						{confirmLabel}
					{/if}
				</Button>
			</div>
		</div>
	</div>
{/if}
