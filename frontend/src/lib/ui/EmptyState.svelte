<script lang="ts">
	interface Props {
		message: string;
		filteredMessage?: string;
		isFiltered?: boolean;
		actionHref?: string;
		actionLabel?: string;
		class?: string;
	}

	let {
		message,
		filteredMessage,
		isFiltered = false,
		actionHref,
		actionLabel,
		class: className = ''
	}: Props = $props();
</script>

<div class="p-12 text-center text-muted {className}">
	<p>{isFiltered && filteredMessage ? filteredMessage : message}</p>
	{#if actionHref && actionLabel && !isFiltered}
		<a
			href={actionHref}
			class="mt-4 inline-flex items-center gap-2 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-on-accent hover:bg-accent/90 transition-colors"
		>
			{actionLabel}
		</a>
	{/if}
</div>
