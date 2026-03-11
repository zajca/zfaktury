<script lang="ts">
	interface Props {
		error: string | null;
		class?: string;
	}

	let { error, class: className = '' }: Props = $props();
</script>

{#if error}
	<div
		role="alert"
		class="rounded-lg border border-danger/20 bg-danger-bg p-4 text-sm text-danger {className}"
	>
		{error}
	</div>
{/if}
