<script lang="ts">
	import { getToasts, removeToast } from '$lib/stores/toast.svelte';

	let toasts = $derived(getToasts());
</script>

<div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2" aria-live="polite">
	{#each toasts as t (t.id)}
		<div
			class="flex items-center justify-between gap-3 rounded-lg border px-4 py-3 text-sm shadow-lg
			{t.type === 'success' ? 'border-green-200 bg-green-50 text-green-800' : ''}
			{t.type === 'error' ? 'border-red-200 bg-red-50 text-red-800' : ''}
			{t.type === 'warning' ? 'border-yellow-200 bg-yellow-50 text-yellow-800' : ''}
			{t.type === 'info' ? 'border-blue-200 bg-blue-50 text-blue-800' : ''}"
		>
			<span>{t.message}</span>
			<button
				onclick={() => removeToast(t.id)}
				class="text-current opacity-60 hover:opacity-100 text-lg leading-none"
			>
				&times;
			</button>
		</div>
	{/each}
</div>
