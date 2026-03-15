<script lang="ts">
	import type { Expense } from '$lib/api/client';
	import { formatCZK, fromHalere } from '$lib/utils/money';
	import { formatDate } from '$lib/utils/date';
	import { paymentMethodLabels } from '$lib/utils/invoice';
	import Card from '$lib/ui/Card.svelte';

	let {
		expense
	}: {
		expense: Expense;
	} = $props();

	let hasItems = $derived(expense.items && expense.items.length > 0);
</script>

<Card>
	<h2 class="text-base font-semibold text-primary">Základní údaje</h2>
	<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
		<div>
			<dt class="text-sm font-medium text-tertiary">Kategorie</dt>
			<dd class="mt-1 text-sm text-primary">{expense.category || '-'}</dd>
		</div>
		<div>
			<dt class="text-sm font-medium text-tertiary">Datum</dt>
			<dd class="mt-1 text-sm text-primary">{formatDate(expense.issue_date)}</dd>
		</div>
		{#if expense.expense_number}
			<div>
				<dt class="text-sm font-medium text-tertiary">Číslo dokladu</dt>
				<dd class="mt-1 text-sm text-primary">{expense.expense_number}</dd>
			</div>
		{/if}
		<div>
			<dt class="text-sm font-medium text-tertiary">Způsob platby</dt>
			<dd class="mt-1 text-sm text-primary">
				{paymentMethodLabels[expense.payment_method] ?? expense.payment_method}
			</dd>
		</div>
	</dl>
</Card>

{#if hasItems}
	<!-- Items table -->
	<Card>
		<h2 class="text-base font-semibold text-primary">Položky</h2>
		<div class="mt-4 overflow-x-auto">
			<table class="w-full text-sm">
				<thead>
					<tr class="border-b border-border text-left text-xs text-tertiary">
						<th class="pb-2 pr-4">Popis</th>
						<th class="pb-2 pr-4 text-right">Množství</th>
						<th class="pb-2 pr-4">Jedn.</th>
						<th class="pb-2 pr-4 text-right">Cena/ks</th>
						<th class="pb-2 pr-4 text-right">DPH %</th>
						<th class="pb-2 pr-4 text-right">DPH</th>
						<th class="pb-2 text-right">Celkem</th>
					</tr>
				</thead>
				<tbody>
					{#each expense.items! as item (item.id)}
						<tr class="border-b border-border/50">
							<td class="py-2 pr-4 text-primary">{item.description}</td>
							<td class="py-2 pr-4 text-right font-mono tabular-nums text-primary"
								>{fromHalere(item.quantity)}</td
							>
							<td class="py-2 pr-4 text-secondary">{item.unit}</td>
							<td class="py-2 pr-4 text-right font-mono tabular-nums text-primary"
								>{formatCZK(item.unit_price)}</td
							>
							<td class="py-2 pr-4 text-right text-secondary">{item.vat_rate_percent}%</td>
							<td class="py-2 pr-4 text-right font-mono tabular-nums text-secondary"
								>{formatCZK(item.vat_amount)}</td
							>
							<td class="py-2 text-right font-mono tabular-nums text-primary"
								>{formatCZK(item.total_amount)}</td
							>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		<!-- Totals -->
		<div class="mt-4 border-t border-border pt-3">
			<div class="flex flex-col items-end gap-1 text-sm">
				<div class="flex gap-8">
					<span class="text-tertiary">Základ:</span>
					<span class="font-medium text-primary font-mono tabular-nums"
						>{formatCZK(expense.amount - expense.vat_amount)}</span
					>
				</div>
				<div class="flex gap-8">
					<span class="text-tertiary">DPH:</span>
					<span class="font-medium text-primary font-mono tabular-nums"
						>{formatCZK(expense.vat_amount)}</span
					>
				</div>
				<div class="flex gap-8 border-t border-border pt-1 text-base">
					<span class="font-semibold text-primary">Celkem:</span>
					<span class="font-semibold text-primary font-mono tabular-nums"
						>{formatCZK(expense.amount)}</span
					>
				</div>
			</div>
		</div>
	</Card>
{:else}
	<!-- Flat amount display for legacy expenses -->
	<Card>
		<h2 class="text-base font-semibold text-primary">Částka</h2>
		<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
			<div>
				<dt class="text-sm font-medium text-tertiary">Částka s DPH</dt>
				<dd class="mt-1 text-lg font-semibold text-primary font-mono tabular-nums">
					{formatCZK(expense.amount)}
				</dd>
			</div>
			<div>
				<dt class="text-sm font-medium text-tertiary">DPH ({expense.vat_rate_percent}%)</dt>
				<dd class="mt-1 text-sm text-primary font-mono tabular-nums">
					{formatCZK(expense.vat_amount)}
				</dd>
			</div>
			<div>
				<dt class="text-sm font-medium text-tertiary">Základ</dt>
				<dd class="mt-1 text-sm text-primary font-mono tabular-nums">
					{formatCZK(expense.amount - expense.vat_amount)}
				</dd>
			</div>
		</dl>
	</Card>
{/if}

<Card>
	<h2 class="text-base font-semibold text-primary">Daňové údaje</h2>
	<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
		<div>
			<dt class="text-sm font-medium text-tertiary">Daňově uznatelný</dt>
			<dd class="mt-1 text-sm text-primary">{expense.is_tax_deductible ? 'Ano' : 'Ne'}</dd>
		</div>
		<div>
			<dt class="text-sm font-medium text-tertiary">Podíl pro podnikání</dt>
			<dd class="mt-1 text-sm text-primary">{expense.business_percent}%</dd>
		</div>
	</dl>
</Card>

{#if expense.notes}
	<Card>
		<h2 class="text-base font-semibold text-primary">Poznámky</h2>
		<p class="mt-2 text-sm text-primary whitespace-pre-wrap">{expense.notes}</p>
	</Card>
{/if}

<div class="text-xs text-muted">
	Vytvořeno: {formatDate(expense.created_at)} | Upraveno: {formatDate(expense.updated_at)}
</div>
