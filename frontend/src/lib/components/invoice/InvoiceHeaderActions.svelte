<script lang="ts">
	import type { Invoice } from '$lib/api/client';
	import { invoicesApi } from '$lib/api/client';
	import { downloadFile } from '$lib/utils/download';
	import { statusLabels, statusVariant, invoiceTypeLabels } from '$lib/utils/invoice';
	import Badge from '$lib/ui/Badge.svelte';
	import Button from '$lib/ui/Button.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';

	let {
		invoice,
		invoiceId,
		editing,
		settling,
		onstartedit,
		onsend,
		onmarkpaid,
		onsettle,
		onduplicate,
		ondelete,
		onshowcreditnote,
		onshowsendemail
	}: {
		invoice: Invoice;
		invoiceId: number;
		editing: boolean;
		settling: boolean;
		onstartedit: () => void;
		onsend: () => void;
		onmarkpaid: () => void;
		onsettle: () => void;
		onduplicate: () => void;
		ondelete: () => void;
		onshowcreditnote: () => void;
		onshowsendemail: () => void;
	} = $props();
</script>

<div class="mt-4">
	<div class="flex items-center justify-end gap-3">
		{#if invoice.type !== 'regular'}
			<Badge variant="default">
				{invoiceTypeLabels[invoice.type] ?? invoice.type}
			</Badge>
		{/if}
		<Badge variant={statusVariant[invoice.status] ?? 'default'}>
			{statusLabels[invoice.status] ?? invoice.status}
		</Badge>
		{#if invoice.customer}
			<span class="text-sm text-secondary">{invoice.customer.name}</span>
		{/if}
	</div>

	{#if invoice.related_invoice_id}
		<div class="mt-2 text-sm text-secondary">
			{#if invoice.type === 'credit_note'}
				Dobropis k faktuře:
			{:else if invoice.type === 'proforma' && invoice.relation_type === 'settlement'}
				Vyrovnávací faktura:
			{:else if invoice.relation_type === 'settlement'}
				Zálohová faktura:
			{:else}
				Související faktura:
			{/if}
			<a
				href="/invoices/{invoice.related_invoice_id}"
				class="text-accent-text hover:text-accent font-medium"
			>
				#{invoice.related_invoice_id}
			</a>
		</div>
	{/if}
	{#if invoice.related_invoices?.length}
		{#each invoice.related_invoices as rel (rel.id)}
			<div class="mt-1 text-sm text-secondary">
				{#if rel.type === 'credit_note'}
					Dobropis:
				{:else if rel.relation_type === 'settlement'}
					Vyrovnávací faktura:
				{:else}
					Související faktura:
				{/if}
				<a href="/invoices/{rel.id}" class="text-accent-text hover:text-accent font-medium">
					{rel.invoice_number}
				</a>
			</div>
		{/each}
	{/if}

	{#if !editing}
		<div class="mt-3 flex flex-wrap gap-2">
			{#if invoice.status === 'draft'}
				<Button variant="secondary" onclick={onstartedit}>Upravit</Button>
				<Button
					variant="primary"
					onclick={onsend}
					title="Změní stav faktury na 'Odeslaná'. Pro odeslání emailem použijte 'Odeslat emailem'"
				>
					Odeslat
				</Button>
			{/if}
			{#if invoice.status === 'sent' || invoice.status === 'overdue'}
				<Button
					variant="success"
					onclick={onmarkpaid}
					title="Označí fakturu jako uhrazenou k dnešnímu datu"
				>
					Uhrazená
				</Button>
			{/if}
			{#if invoice.type === 'proforma' && invoice.status === 'paid' && !invoice.related_invoice_id}
				<Button
					variant="primary"
					onclick={onsettle}
					disabled={settling}
					title="Vytvoří řádnou fakturu s odečtenou zálohou"
				>
					{settling ? 'Vyrovnávám...' : 'Vyrovnat zálohu'}
					<HelpTip topic="vyrovnani-zalohy" />
				</Button>
			{/if}
			<Button
				variant="secondary"
				onclick={() =>
					downloadFile(invoicesApi.getPdfUrl(invoiceId), `faktura-${invoice.invoice_number}.pdf`)}
			>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
					/>
				</svg>
				Stáhnout PDF
			</Button>
			<Button variant="secondary" onclick={onshowsendemail}>Odeslat emailem</Button>
			<Button
				variant="secondary"
				onclick={() =>
					downloadFile(
						invoicesApi.getIsdocUrl(invoiceId),
						`faktura-${invoice.invoice_number}.isdoc`
					)}
				title="Stáhne fakturu ve formátu ISDOC (český standard elektronické fakturace)"
			>
				Export ISDOC <HelpTip topic="isdoc-export" />
			</Button>
			<Button
				variant="secondary"
				onclick={onduplicate}
				title="Vytvoří novou fakturu jako kopii -- zkopíruje zákazníka, položky a nastavení, přiřadí nové číslo"
			>
				Duplikovat
			</Button>
			{#if invoice.type === 'regular' && (invoice.status === 'sent' || invoice.status === 'paid')}
				<Button
					variant="secondary"
					onclick={onshowcreditnote}
					title="Vytvoří opravný daňový doklad, který stornuje tuto fakturu"
				>
					Vytvořit dobropis <HelpTip topic="dobropis" />
				</Button>
			{/if}
			{#if invoice.status !== 'paid'}
				<Button variant="danger" onclick={ondelete}>Smazat</Button>
			{/if}
		</div>
	{/if}
</div>
