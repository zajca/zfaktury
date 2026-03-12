<script lang="ts">
	import type { HTMLTextareaAttributes } from 'svelte/elements';

	let {
		value = $bindable(''),
		class: className = '',
		...rest
	}: HTMLTextareaAttributes & { value?: string; class?: string } = $props();
</script>

<textarea
	bind:value
	class="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none {className}"
	{...rest}
></textarea>
