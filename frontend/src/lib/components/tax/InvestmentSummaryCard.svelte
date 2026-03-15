<script lang="ts">
	import type { InvestmentYearSummary } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import Card from '$lib/ui/Card.svelte';

	interface Props {
		summary: InvestmentYearSummary;
		selectedYear: number;
	}

	let { summary, selectedYear }: Props = $props();

	function formatAmount(amountInHalere: number): string {
		return formatCZK(amountInHalere);
	}
</script>

<Card>
	<h2 class="text-base font-semibold text-primary">
		Souhrn investičních příjmů {selectedYear}
	</h2>
	<div class="mt-4 grid grid-cols-1 gap-6 md:grid-cols-2">
		<!-- §8 Capital income -->
		<div>
			<h3 class="text-sm font-medium text-tertiary">Kapitálové příjmy (§8)</h3>
			<div class="mt-2 space-y-1 text-sm">
				<div class="flex justify-between">
					<span class="text-tertiary">Hrubé příjmy</span>
					<strong class="text-primary">{formatAmount(summary.capital_income_gross)}</strong>
				</div>
				<div class="flex justify-between">
					<span class="text-tertiary">Sražená daň</span>
					<strong class="text-primary">{formatAmount(summary.capital_income_tax)}</strong>
				</div>
				<div class="flex justify-between border-t border-border-subtle pt-1">
					<span class="text-tertiary">Čisté příjmy</span>
					<strong class="text-primary">{formatAmount(summary.capital_income_net)}</strong>
				</div>
			</div>
		</div>

		<!-- §10 Other income -->
		<div>
			<h3 class="text-sm font-medium text-tertiary">Ostatní příjmy - CP (§10)</h3>
			<div class="mt-2 space-y-1 text-sm">
				<div class="flex justify-between">
					<span class="text-tertiary">Hrubé příjmy</span>
					<strong class="text-primary">{formatAmount(summary.other_income_gross)}</strong>
				</div>
				<div class="flex justify-between">
					<span class="text-tertiary">Výdaje (FIFO)</span>
					<strong class="text-primary">{formatAmount(summary.other_income_expenses)}</strong>
				</div>
				<div class="flex justify-between">
					<span class="text-tertiary">Osvobozeno (časový test)</span>
					<strong class="text-primary">{formatAmount(summary.other_income_exempt)}</strong>
				</div>
				<div class="flex justify-between border-t border-border-subtle pt-1">
					<span class="text-tertiary">Zdanitelný příjem</span>
					<strong class="text-primary">{formatAmount(summary.other_income_net)}</strong>
				</div>
			</div>
		</div>
	</div>
</Card>
