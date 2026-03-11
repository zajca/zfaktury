<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		description?: string;
		backHref?: string;
		backLabel?: string;
		actions?: Snippet;
		class?: string;
	}

	let { title, description, backHref, backLabel, actions, class: className = '' }: Props = $props();
</script>

<div class={className}>
	{#if backHref}
		<a href={backHref} class="text-sm text-secondary hover:text-primary">&larr; {backLabel ?? 'Zpět'}</a>
	{/if}
	<div class="flex items-center justify-between {backHref ? 'mt-2' : ''}">
		<div>
			<h1 class="text-xl font-semibold text-primary">{title}</h1>
			{#if description}
				<p class="mt-1 text-sm text-tertiary">{description}</p>
			{/if}
		</div>
		{#if actions}
			{@render actions()}
		{/if}
	</div>
</div>
