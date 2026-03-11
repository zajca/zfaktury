<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		padding?: boolean;
		children: Snippet;
		class?: string;
	}

	let { padding = true, children, class: className = '' }: Props = $props();
</script>

<div class="rounded-lg border border-border bg-surface {padding ? 'p-5' : ''} {className}">
	{@render children()}
</div>
