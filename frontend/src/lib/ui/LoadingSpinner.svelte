<script lang="ts">
	interface Props {
		class?: string;
	}

	let { class: className = '' }: Props = $props();
</script>

<div class="flex items-center justify-center {className}">
	<div role="status">
		<div
			class="h-8 w-8 animate-spin rounded-full border-4 border-border border-t-accent"
		></div>
		<span class="sr-only">Načítání...</span>
	</div>
</div>
