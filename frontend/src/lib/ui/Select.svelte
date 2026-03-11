<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLSelectAttributes } from 'svelte/elements';

	let {
		children,
		class: className = '',
		...rest
	}: HTMLSelectAttributes & { children: Snippet; class?: string } = $props();
</script>

<select
	class="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none {className}"
	{...rest}
>
	{@render children()}
</select>
