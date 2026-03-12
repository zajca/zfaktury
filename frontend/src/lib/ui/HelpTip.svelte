<script lang="ts">
	import type { TaxConstants } from '$lib/api/client';
	import type { HelpTopicId } from '$lib/data/help-content';
	import { getHelpTopics } from '$lib/data/help-content';
	import { openHelp } from '$lib/data/help-state.svelte';

	interface Props {
		topic: HelpTopicId;
		taxConstants?: TaxConstants | null;
		class?: string;
	}

	let { topic, taxConstants = null, class: className = '' }: Props = $props();

	let title = $derived(getHelpTopics(taxConstants)[topic].title);

	function handleClick(event: MouseEvent) {
		event.preventDefault();
		event.stopPropagation();
		openHelp(topic, taxConstants);
	}
</script>

<button
	type="button"
	class="inline-flex items-center align-middle ml-1 text-muted hover:text-secondary cursor-help transition-colors {className}"
	onclick={handleClick}
	aria-label="Nápověda: {title}"
	aria-haspopup="dialog"
>
	<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
		<path stroke-linecap="round" stroke-linejoin="round" d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9 5.25h.008v.008H12v-.008z" />
	</svg>
</button>
