<script lang="ts">
	import type { Invoice } from '$lib/api/client';
	import { formatDate } from '$lib/utils/date';
	import Card from '$lib/ui/Card.svelte';

	let { invoice }: { invoice: Invoice } = $props();
</script>

<!-- Invoice details -->
<Card>
	<h2 class="text-base font-semibold text-primary">Údaje faktury</h2>
	<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
		<div>
			<dt class="text-sm font-medium text-tertiary">Datum vystavení</dt>
			<dd class="mt-1 text-sm text-primary">{formatDate(invoice.issue_date)}</dd>
		</div>
		<div>
			<dt class="text-sm font-medium text-tertiary">Datum splatnosti</dt>
			<dd class="mt-1 text-sm text-primary">{formatDate(invoice.due_date)}</dd>
		</div>
		<div>
			<dt class="text-sm font-medium text-tertiary">DUZP</dt>
			<dd class="mt-1 text-sm text-primary">{formatDate(invoice.delivery_date)}</dd>
		</div>
		<div>
			<dt class="text-sm font-medium text-tertiary">Variabilní symbol</dt>
			<dd class="mt-1 text-sm text-primary">{invoice.variable_symbol || '-'}</dd>
		</div>
		<div>
			<dt class="text-sm font-medium text-tertiary">Konstantní symbol</dt>
			<dd class="mt-1 text-sm text-primary">{invoice.constant_symbol || '-'}</dd>
		</div>
		<div>
			<dt class="text-sm font-medium text-tertiary">Způsob platby</dt>
			<dd class="mt-1 text-sm text-primary">
				{#if invoice.payment_method === 'bank_transfer'}Bankovní převod
				{:else if invoice.payment_method === 'cash'}Hotovost
				{:else if invoice.payment_method === 'card'}Karta
				{:else}{invoice.payment_method}
				{/if}
			</dd>
		</div>
	</dl>
</Card>

<!-- Customer -->
{#if invoice.customer}
	<Card>
		<h2 class="text-base font-semibold text-primary">Zákazník</h2>
		<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
			<div>
				<dt class="text-sm font-medium text-tertiary">Název</dt>
				<dd class="mt-1 text-sm text-primary">
					<a href="/contacts/{invoice.customer.id}" class="text-accent-text hover:text-accent"
						>{invoice.customer.name}</a
					>
				</dd>
			</div>
			{#if invoice.customer.ico}
				<div>
					<dt class="text-sm font-medium text-tertiary">IČO</dt>
					<dd class="mt-1 text-sm text-primary">{invoice.customer.ico}</dd>
				</div>
			{/if}
			{#if invoice.customer.dic}
				<div>
					<dt class="text-sm font-medium text-tertiary">DIČ</dt>
					<dd class="mt-1 text-sm text-primary">{invoice.customer.dic}</dd>
				</div>
			{/if}
		</dl>
	</Card>
{/if}
