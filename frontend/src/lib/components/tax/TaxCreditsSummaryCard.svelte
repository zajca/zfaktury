<script lang="ts">
	import type { TaxCreditsSummary, TaxDeduction } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import Card from '$lib/ui/Card.svelte';

	let {
		summary,
		deductions
	}: {
		summary: TaxCreditsSummary;
		deductions: TaxDeduction[];
	} = $props();
</script>

<Card>
	<h2 class="text-base font-semibold text-primary">Souhrn</h2>
	<div class="mt-3 space-y-1 text-sm">
		<div class="flex justify-between">
			<span class="text-tertiary">Celkové slevy (bez základní)</span>
			<strong class="text-primary">{formatCZK(summary.total_credits)}</strong>
		</div>
		<div class="flex justify-between">
			<span class="text-tertiary">Daňové zvýhodnění na děti</span>
			<strong class="text-primary">{formatCZK(summary.total_child_benefit)}</strong>
		</div>
		{#if deductions.length > 0}
			{@const totalDeductions = deductions.reduce((sum, d) => sum + d.allowed_amount, 0)}
			<div class="flex justify-between">
				<span class="text-tertiary">Nezdanitelné odpočty</span>
				<strong class="text-primary">{formatCZK(totalDeductions)}</strong>
			</div>
		{/if}
	</div>
</Card>
