<script lang="ts">
	import { fly } from 'svelte/transition';
	import { toasts, removeToast, type ToastType } from '$lib/data/toast-state.svelte';

	const typeConfig: Record<
		ToastType,
		{ containerClass: string; role: string; ariaLive: 'polite' | 'assertive' }
	> = {
		success: {
			containerClass: 'border-success/20 bg-success-bg text-success',
			role: 'status',
			ariaLive: 'polite'
		},
		error: {
			containerClass: 'border-danger/20 bg-danger-bg text-danger',
			role: 'alert',
			ariaLive: 'assertive'
		},
		warning: {
			containerClass: 'border-warning/20 bg-warning-bg text-warning',
			role: 'alert',
			ariaLive: 'assertive'
		},
		info: {
			containerClass: 'border-accent/20 bg-accent/5 text-accent',
			role: 'status',
			ariaLive: 'polite'
		}
	};
</script>

<div
	class="fixed top-4 right-4 z-50 flex flex-col gap-2"
	style="min-width: 300px; max-width: 420px;"
>
	{#each toasts as toast (toast.id)}
		{@const config = typeConfig[toast.type]}
		<div
			class="flex items-start gap-3 rounded-lg border bg-surface px-4 py-3 shadow-lg {config.containerClass}"
			role={config.role}
			aria-live={config.ariaLive}
			transition:fly={{ x: 100, duration: 300 }}
		>
			<!-- Icon per type -->
			{#if toast.type === 'success'}
				<svg
					class="mt-0.5 h-5 w-5 shrink-0"
					aria-hidden="true"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
			{:else if toast.type === 'error'}
				<svg
					class="mt-0.5 h-5 w-5 shrink-0"
					aria-hidden="true"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
			{:else if toast.type === 'warning'}
				<svg
					class="mt-0.5 h-5 w-5 shrink-0"
					aria-hidden="true"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
					/>
				</svg>
			{:else}
				<svg
					class="mt-0.5 h-5 w-5 shrink-0"
					aria-hidden="true"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
			{/if}

			<p class="flex-1 text-sm font-medium">{toast.message}</p>

			<button
				type="button"
				class="shrink-0 rounded p-0.5 opacity-70 hover:opacity-100 transition-opacity"
				onclick={() => removeToast(toast.id)}
				aria-label="Zavřít"
			>
				<svg
					class="h-4 w-4"
					aria-hidden="true"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>
	{/each}
</div>
