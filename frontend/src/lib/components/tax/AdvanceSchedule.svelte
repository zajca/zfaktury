<script lang="ts">
	import type { AdvancePeriod } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import { formatMonthYear, insuranceWarningText } from '$lib/utils/insurance';

	interface Props {
		overviewYear: number;
		startMonth: number;
		periods: AdvancePeriod[];
		warnings: string[];
		reportedPrepay: number | null;
	}

	let { overviewYear, startMonth, periods, warnings, reportedPrepay }: Props = $props();

	let advanceYear = $derived(overviewYear + 1);
	let differsFromReported = $derived(
		reportedPrepay !== null && periods.length > 0 && periods[0].monthly_prepay !== reportedPrepay
	);
</script>

<div class="mt-4 space-y-2 text-sm">
	{#if startMonth > 0}
		<p class="text-secondary">
			Nová záloha od: <span class="font-medium text-primary"
				>{formatMonthYear(startMonth, advanceYear)}</span
			>
		</p>
	{/if}

	{#if periods.length > 1}
		<ul class="space-y-1">
			{#each periods as period (period.from_month)}
				<li class="flex justify-between text-secondary">
					<span>od {formatMonthYear(period.from_month, advanceYear)}:</span>
					<span class="font-medium text-primary tabular-nums"
						>{formatCZK(period.monthly_prepay)}</span
					>
				</li>
			{/each}
		</ul>
	{/if}

	{#if differsFromReported && reportedPrepay !== null}
		<p class="text-secondary">
			Záloha uvedená v přehledu (ř. 36):
			<span class="font-medium text-primary tabular-nums">{formatCZK(reportedPrepay)}</span>
		</p>
	{/if}

	{#each warnings as warning (warning)}
		<div role="alert" class="rounded-lg border border-warning/40 bg-warning/10 p-3 text-primary">
			<p>{insuranceWarningText(warning, overviewYear)}</p>
			{#if warning === 'filing_method_missing'}
				<a href="/tax/prepayments" class="mt-1 inline-block text-accent underline"
					>Nastavit způsob podání</a
				>
			{/if}
		</div>
	{/each}
</div>
