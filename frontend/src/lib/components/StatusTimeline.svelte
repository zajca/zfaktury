<script lang="ts">
	import type { InvoiceStatusChange } from '$lib/api/client';
	import Badge from '$lib/ui/Badge.svelte';
	import { statusLabels, statusVariant } from '$lib/utils/invoice';
	import { formatDateTime } from '$lib/utils/date';

	interface Props {
		history: InvoiceStatusChange[];
	}

	let { history }: Props = $props();

	// Sort newest first
	const sorted = $derived(
		[...history].sort(
			(a, b) => new Date(b.changed_at).getTime() - new Date(a.changed_at).getTime()
		)
	);

	// Dot color classes keyed by status
	const dotColors: Record<string, string> = {
		draft: 'bg-secondary',
		sent: 'bg-info',
		paid: 'bg-success',
		overdue: 'bg-danger',
		cancelled: 'bg-muted'
	};

	function getDotColor(status: string): string {
		return dotColors[status] ?? 'bg-secondary';
	}

	function getLabel(status: string): string {
		return statusLabels[status] ?? status;
	}
</script>

{#if sorted.length === 0}
	<p class="text-muted text-sm" data-testid="empty-state">Zatim zadne zmeny stavu.</p>
{:else}
	<div class="relative" data-testid="status-timeline">
		{#each sorted as entry, i (entry.id)}
			<div class="relative flex gap-4 pb-6 last:pb-0">
				<!-- Vertical line -->
				{#if i < sorted.length - 1}
					<div
						class="absolute left-[7px] top-5 bottom-0 w-px bg-border"
						aria-hidden="true"
					></div>
				{/if}

				<!-- Dot -->
				<div class="relative flex-shrink-0 mt-1.5">
					<div
						class="h-3.5 w-3.5 rounded-full ring-2 ring-surface {getDotColor(entry.new_status)}"
						aria-hidden="true"
					></div>
				</div>

				<!-- Content -->
				<div class="flex-1 min-w-0">
					<div class="flex items-center gap-2 flex-wrap">
						<Badge variant={statusVariant[entry.old_status] ?? 'default'}>
							{getLabel(entry.old_status)}
						</Badge>
						<span class="text-muted text-sm" aria-hidden="true">&rarr;</span>
						<Badge variant={statusVariant[entry.new_status] ?? 'default'}>
							{getLabel(entry.new_status)}
						</Badge>
					</div>

					<p class="text-tertiary text-xs mt-1" data-testid="timestamp">
						{formatDateTime(entry.changed_at)}
					</p>

					{#if entry.note}
						<p class="text-secondary text-sm mt-1" data-testid="note">
							{entry.note}
						</p>
					{/if}
				</div>
			</div>
		{/each}
	</div>
{/if}
