<script lang="ts">
	import Button from './Button.svelte';

	interface Props {
		page: number;
		totalPages: number;
		total: number;
		label?: string;
		onPageChange: (newPage: number) => void;
		class?: string;
	}

	let { page, totalPages, total, label = 'položek', onPageChange, class: className = '' }: Props = $props();
</script>

{#if totalPages > 1}
	<div class="mt-4 flex items-center justify-between {className}">
		<p class="text-sm text-tertiary">
			Celkem {total} {label}
		</p>
		<div class="flex gap-2">
			<Button
				variant="secondary"
				size="sm"
				onclick={() => onPageChange(Math.max(1, page - 1))}
				disabled={page <= 1}
			>
				Předchozí
			</Button>
			<span class="flex items-center px-3 text-sm text-secondary">
				{page} / {totalPages}
			</span>
			<Button
				variant="secondary"
				size="sm"
				onclick={() => onPageChange(Math.min(totalPages, page + 1))}
				disabled={page >= totalPages}
			>
				Další
			</Button>
		</div>
	</div>
{/if}
