<script lang="ts">
	import type { Invoice } from '$lib/api/client';
	import { invoicesApi } from '$lib/api/client';
	import Card from '$lib/ui/Card.svelte';

	let {
		invoice,
		invoiceId,
		qrError = $bindable()
	}: {
		invoice: Invoice;
		invoiceId: number;
		qrError: boolean;
	} = $props();
</script>

{#if invoice.bank_account || invoice.iban}
	<Card>
		<h2 class="text-base font-semibold text-primary">Platební údaje</h2>
		<div class="mt-4 flex flex-col gap-6 sm:flex-row">
			<dl class="flex-1 grid grid-cols-1 gap-4 sm:grid-cols-2">
				{#if invoice.bank_account}
					<div>
						<dt class="text-sm font-medium text-tertiary">Číslo účtu</dt>
						<dd class="mt-1 text-sm text-primary">
							{invoice.bank_account}{invoice.bank_code ? `/${invoice.bank_code}` : ''}
						</dd>
					</div>
				{/if}
				{#if invoice.iban}
					<div>
						<dt class="text-sm font-medium text-tertiary">IBAN</dt>
						<dd class="mt-1 text-sm text-primary">{invoice.iban}</dd>
					</div>
				{/if}
				{#if invoice.variable_symbol}
					<div>
						<dt class="text-sm font-medium text-tertiary">Variabilní symbol</dt>
						<dd class="mt-1 text-sm text-primary">{invoice.variable_symbol}</dd>
					</div>
				{/if}
			</dl>
			{#if invoice.iban && invoice.status !== 'paid'}
				<div class="flex flex-col items-center gap-2">
					<span class="text-sm font-medium text-tertiary">QR platba</span>
					{#if qrError}
						<div
							class="flex h-32 w-32 items-center justify-center rounded border border-border bg-elevated text-xs text-muted"
						>
							QR kód není dostupný
						</div>
					{:else}
						<img
							src={invoicesApi.getQrUrl(invoiceId)}
							alt="QR kód pro platbu"
							class="h-32 w-32 rounded border border-border"
							onerror={() => {
								qrError = true;
							}}
						/>
					{/if}
				</div>
			{/if}
		</div>
	</Card>
{/if}
