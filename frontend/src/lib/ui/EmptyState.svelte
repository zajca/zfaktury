<script lang="ts">
	interface Props {
		message: string;
		filteredMessage?: string;
		isFiltered?: boolean;
		class?: string;
	}

	let { message, filteredMessage, isFiltered = false, class: className = '' }: Props = $props();
</script>

<div class="p-12 text-center text-muted {className}">
	{isFiltered && filteredMessage ? filteredMessage : message}
</div>
