<script lang="ts">
	import Button from './Button.svelte';

	interface Props {
		saving: boolean;
		saveLabel?: string;
		savingLabel?: string;
		cancelHref?: string;
		oncancel?: () => void;
		class?: string;
	}

	let {
		saving,
		saveLabel = 'Uložit',
		savingLabel = 'Ukládám...',
		cancelHref,
		oncancel,
		class: className = ''
	}: Props = $props();
</script>

<div class="flex gap-3 {className}">
	<Button type="submit" variant="primary" disabled={saving}>
		{saving ? savingLabel : saveLabel}
	</Button>
	{#if cancelHref}
		<Button variant="secondary" href={cancelHref}>Zrušit</Button>
	{:else if oncancel}
		<Button variant="secondary" onclick={oncancel}>Zrušit</Button>
	{/if}
</div>
