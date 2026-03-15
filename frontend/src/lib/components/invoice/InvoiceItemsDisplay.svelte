<script lang="ts">
	import type { Invoice } from '$lib/api/client';
	import { formatCZK, fromHalere } from '$lib/utils/money';
	import Card from '$lib/ui/Card.svelte';

	let { invoice }: { invoice: Invoice } = $props();
</script>

<Card>
	<h2 class="text-base font-semibold text-primary">Položky</h2>
	<div class="mt-4 overflow-x-auto">
		<table class="w-full text-left text-sm">
			<thead class="border-b border-border">
				<tr>
					<th class="pb-2 text-xs font-medium uppercase tracking-wider text-muted">Popis</th>
					<th class="pb-2 text-right text-xs font-medium uppercase tracking-wider text-muted"
						>Množství</th
					>
					<th class="pb-2 text-xs font-medium uppercase tracking-wider text-muted">Jednotka</th>
					<th class="pb-2 text-right text-xs font-medium uppercase tracking-wider text-muted"
						>Cena/ks</th
					>
					<th class="pb-2 text-right text-xs font-medium uppercase tracking-wider text-muted"
						>DPH %</th
					>
					<th class="pb-2 text-right text-xs font-medium uppercase tracking-wider text-muted"
						>DPH</th
					>
					<th class="pb-2 text-right text-xs font-medium uppercase tracking-wider text-muted"
						>Celkem</th
					>
				</tr>
			</thead>
			<tbody class="divide-y divide-border-subtle">
				{#each invoice.items ?? [] as item (item.id)}
					<tr>
						<td class="py-2.5 text-primary">{item.description}</td>
						<td class="py-2.5 text-right font-mono tabular-nums text-secondary"
							>{fromHalere(item.quantity)}</td
						>
						<td class="py-2.5 text-secondary">{item.unit}</td>
						<td class="py-2.5 text-right font-mono tabular-nums text-secondary"
							>{formatCZK(item.unit_price)}</td
						>
						<td class="py-2.5 text-right font-mono tabular-nums text-secondary"
							>{item.vat_rate_percent}%</td
						>
						<td class="py-2.5 text-right font-mono tabular-nums text-secondary"
							>{formatCZK(item.vat_amount)}</td
						>
						<td class="py-2.5 text-right font-mono tabular-nums font-medium text-primary"
							>{formatCZK(item.total_amount)}</td
						>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>

	<div class="mt-4 border-t border-border pt-4">
		<div class="flex flex-col items-end gap-1 text-sm">
			<div class="flex gap-8">
				<span class="text-secondary">Základ:</span>
				<span class="font-medium font-mono tabular-nums text-primary"
					>{formatCZK(invoice.subtotal_amount)}</span
				>
			</div>
			<div class="flex gap-8">
				<span class="text-secondary">DPH:</span>
				<span class="font-medium font-mono tabular-nums text-primary"
					>{formatCZK(invoice.vat_amount)}</span
				>
			</div>
			<div class="flex gap-8 border-t border-border pt-1 text-base">
				<span class="font-semibold text-primary">Celkem:</span>
				<span class="font-bold font-mono tabular-nums text-primary"
					>{formatCZK(invoice.total_amount)}</span
				>
			</div>
			{#if invoice.paid_amount > 0}
				<div class="flex gap-8 text-success">
					<span>Uhrazeno:</span>
					<span class="font-medium font-mono tabular-nums">{formatCZK(invoice.paid_amount)}</span>
				</div>
			{/if}
		</div>
	</div>
</Card>
