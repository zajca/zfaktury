<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		invoicesApi,
		contactsApi,
		statusHistoryApi,
		invoiceDocumentsApi,
		type Invoice,
		type Contact,
		type InvoiceStatusChange,
		type InvoiceDocument
	} from '$lib/api/client';
	import { toHalere, fromHalere } from '$lib/utils/money';
	import { formatDate, toISODate } from '$lib/utils/date';
	import type { FormItem } from '$lib/components/InvoiceItemsEditor.svelte';
	import StatusTimeline from '$lib/components/StatusTimeline.svelte';
	import CreditNoteDialog from '$lib/components/CreditNoteDialog.svelte';
	import SendEmailDialog from '$lib/components/SendEmailDialog.svelte';
	import ReminderCard from '$lib/components/ReminderCard.svelte';
	import Card from '$lib/ui/Card.svelte';
	import ConfirmDialog from '$lib/ui/ConfirmDialog.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import { toastSuccess, toastError } from '$lib/data/toast-state.svelte';
	import AuditLogPanel from '$lib/components/AuditLogPanel.svelte';
	import { invoiceTypeLabels } from '$lib/utils/invoice';
	import InvoiceHeaderActions from '$lib/components/invoice/InvoiceHeaderActions.svelte';
	import InvoiceEditForm from '$lib/components/invoice/InvoiceEditForm.svelte';
	import InvoiceDetailsCard from '$lib/components/invoice/InvoiceDetailsCard.svelte';
	import InvoiceItemsDisplay from '$lib/components/invoice/InvoiceItemsDisplay.svelte';
	import InvoicePaymentInfoCard from '$lib/components/invoice/InvoicePaymentInfoCard.svelte';

	let invoice = $state<Invoice | null>(null);
	let contacts = $state<Contact[]>([]);
	let statusHistory = $state<InvoiceStatusChange[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state<string | null>(null);
	let editing = $state(false);
	let qrError = $state(false);
	let showCreditNoteDialog = $state(false);
	let showSendEmailDialog = $state(false);
	let settling = $state(false);
	let showDeleteConfirm = $state(false);
	let invoiceDocuments = $state<InvoiceDocument[]>([]);

	let invoiceId = $derived(Number(page.params.id));

	let form = $state({
		customer_id: 0,
		issue_date: '',
		due_date: '',
		delivery_date: '',
		variable_symbol: '',
		constant_symbol: '',
		currency_code: 'CZK',
		payment_method: 'bank_transfer',
		notes: '',
		internal_notes: ''
	});

	let items = $state<FormItem[]>([]);
	let dueDateOffset = $state(14);

	let mounted = false;
	onMount(() => {
		loadInvoice();
		loadInvoiceDocuments();
		mounted = true;
	});
	$effect(() => {
		invoiceId;
		if (!mounted) return;
		loadInvoice();
	});

	async function loadInvoice() {
		loading = true;
		error = null;
		qrError = false;
		try {
			const [inv, history] = await Promise.all([
				invoicesApi.getById(invoiceId),
				statusHistoryApi.getHistory(invoiceId).catch(() => [] as InvoiceStatusChange[])
			]);
			invoice = inv;
			statusHistory = history;
			populateForm();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst fakturu';
		} finally {
			loading = false;
		}
	}

	async function loadInvoiceDocuments() {
		try {
			invoiceDocuments = await invoiceDocumentsApi.listByInvoice(invoiceId);
		} catch {
			// non-critical
		}
	}

	function formatFileSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}

	function populateForm() {
		if (!invoice) return;
		form = {
			customer_id: invoice.customer_id,
			issue_date: invoice.issue_date,
			due_date: invoice.due_date,
			delivery_date: invoice.delivery_date,
			variable_symbol: invoice.variable_symbol,
			constant_symbol: invoice.constant_symbol,
			currency_code: invoice.currency_code,
			payment_method: invoice.payment_method,
			notes: invoice.notes,
			internal_notes: invoice.internal_notes
		};
		items = (invoice.items ?? []).map((item) => ({
			description: item.description,
			quantity: fromHalere(item.quantity),
			unit: item.unit,
			unit_price: fromHalere(item.unit_price),
			vat_rate_percent: item.vat_rate_percent
		}));
	}

	async function startEditing() {
		editing = true;
		try {
			const res = await contactsApi.list({ limit: 1000 });
			contacts = res.data;
		} catch {
			// non-critical
		}
	}

	function cancelEditing() {
		editing = false;
		populateForm();
	}

	async function handleSave() {
		saving = true;
		try {
			const invoiceItems = items.map((item, index) => ({
				description: item.description,
				quantity: toHalere(item.quantity),
				unit: item.unit,
				unit_price: toHalere(item.unit_price),
				vat_rate_percent: item.vat_rate_percent,
				sort_order: index
			}));

			await invoicesApi.update(invoiceId, {
				...form,
				type: invoice?.type ?? 'regular',
				items: invoiceItems as Invoice['items']
			});
			toastSuccess('Faktura uložena');
			editing = false;
			await loadInvoice();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se uložit fakturu');
		} finally {
			saving = false;
		}
	}

	async function handleSend() {
		try {
			await invoicesApi.send(invoiceId);
			toastSuccess('Faktura odeslána');
			await loadInvoice();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se odeslat fakturu');
		}
	}

	async function handleMarkPaid() {
		if (!invoice) return;
		try {
			await invoicesApi.markPaid(invoiceId, invoice.total_amount, toISODate(new Date()));
			toastSuccess('Faktura označena jako uhrazená');
			await loadInvoice();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se označit jako uhrazenou');
		}
	}

	async function handleDuplicate() {
		try {
			const dup = await invoicesApi.duplicate(invoiceId);
			toastSuccess('Faktura duplikována');
			goto(`/invoices/${dup.id}`);
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se duplikovat fakturu');
		}
	}

	function handleDelete() {
		showDeleteConfirm = true;
	}

	async function confirmDelete() {
		showDeleteConfirm = false;
		try {
			await invoicesApi.delete(invoiceId);
			toastSuccess('Faktura smazána');
			goto('/invoices');
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se smazat fakturu');
		}
	}

	async function handleSettle() {
		settling = true;
		try {
			const settled = await invoicesApi.settle(invoiceId);
			toastSuccess('Záloha vypořádána');
			goto(`/invoices/${settled.id}`);
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se vyrovnat zálohu');
		} finally {
			settling = false;
		}
	}
</script>

<svelte:head>
	<title>{invoice ? `Faktura ${invoice.invoice_number}` : 'Faktura'} - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<PageHeader
		title={invoice
			? `${invoiceTypeLabels[invoice.type] ?? 'Faktura'} ${invoice.invoice_number}`
			: 'Faktura'}
		backHref="/invoices"
		backLabel="Zpět na faktury"
	/>

	<ErrorAlert {error} class="mt-4" />

	{#if loading}
		<LoadingSpinner class="mt-8" />
	{:else if invoice}
		<InvoiceHeaderActions
			{invoice}
			{invoiceId}
			{editing}
			{settling}
			onstartedit={startEditing}
			onsend={handleSend}
			onmarkpaid={handleMarkPaid}
			onsettle={handleSettle}
			onduplicate={handleDuplicate}
			ondelete={handleDelete}
			onshowcreditnote={() => {
				showCreditNoteDialog = true;
			}}
			onshowsendemail={() => {
				showSendEmailDialog = true;
			}}
		/>

		{#if editing}
			<InvoiceEditForm
				bind:form
				bind:items
				{contacts}
				{saving}
				bind:dueDateOffset
				onsave={handleSave}
				oncancel={cancelEditing}
			/>
		{:else}
			<!-- View mode -->
			<div class="mt-6 space-y-6">
				<InvoiceDetailsCard {invoice} />
				<InvoiceItemsDisplay {invoice} />
				<InvoicePaymentInfoCard {invoice} {invoiceId} bind:qrError />

				<!-- Notes -->
				{#if invoice.notes || invoice.internal_notes}
					<Card>
						<h2 class="text-base font-semibold text-primary">Poznámky</h2>
						{#if invoice.notes}
							<div class="mt-4">
								<h3 class="text-sm font-medium text-tertiary">Poznámka na faktuře</h3>
								<p class="mt-1 text-sm text-primary whitespace-pre-wrap">{invoice.notes}</p>
							</div>
						{/if}
						{#if invoice.internal_notes}
							<div class="mt-4">
								<h3 class="text-sm font-medium text-tertiary">Interní poznámka</h3>
								<p class="mt-1 text-sm text-primary whitespace-pre-wrap">
									{invoice.internal_notes}
								</p>
							</div>
						{/if}
					</Card>
				{/if}

				<!-- Documents -->
				{#if invoiceDocuments.length > 0}
					<Card>
						<h2 class="text-base font-semibold text-primary">Dokumenty</h2>
						<ul class="mt-4 divide-y divide-border">
							{#each invoiceDocuments as doc (doc.id)}
								<li class="flex items-center justify-between gap-3 py-3">
									<div class="min-w-0 flex-1">
										<p class="truncate text-sm font-medium text-primary">{doc.filename}</p>
										<p class="text-xs text-muted">
											{formatFileSize(doc.size)} — {formatDate(doc.created_at)}
										</p>
									</div>
									<div class="flex shrink-0 items-center gap-1.5">
										<a
											href={invoiceDocumentsApi.getDownloadUrl(doc.id)}
											target="_blank"
											rel="noopener noreferrer"
											class="rounded-md px-2.5 py-1.5 text-xs text-secondary hover:bg-hover hover:text-primary transition-colors"
											title="Stáhnout"
										>
											<svg
												class="h-4 w-4"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
												stroke-width="1.5"
											>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3"
												/>
											</svg>
										</a>
										<button
											type="button"
											onclick={async () => {
												try {
													await invoiceDocumentsApi.delete(doc.id);
													await loadInvoiceDocuments();
													toastSuccess('Dokument smazán');
												} catch {
													toastError('Nepodařilo se smazat dokument');
												}
											}}
											class="rounded-md px-2.5 py-1.5 text-xs text-danger hover:bg-danger-bg transition-colors"
											title="Smazat"
										>
											<svg
												class="h-4 w-4"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
												stroke-width="1.5"
											>
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0"
												/>
											</svg>
										</button>
									</div>
								</li>
							{/each}
						</ul>
					</Card>
				{/if}

				<!-- Status History -->
				{#if statusHistory.length > 0}
					<Card>
						<h2 class="text-base font-semibold text-primary">Historie stavů</h2>
						<div class="mt-4">
							<StatusTimeline history={statusHistory} />
						</div>
					</Card>
				{/if}

				<!-- Reminders -->
				<ReminderCard {invoiceId} invoiceStatus={invoice.status} />

				<!-- Timestamps -->
				<div class="text-xs text-muted">
					Vytvořeno: {formatDate(invoice.created_at)} | Upraveno: {formatDate(invoice.updated_at)}
					{#if invoice.sent_at}
						| Odesláno: {formatDate(invoice.sent_at)}{/if}
					{#if invoice.paid_at}
						| Uhrazeno: {formatDate(invoice.paid_at)}{/if}
				</div>
			</div>

			<!-- Dialogs -->
			{#if showCreditNoteDialog}
				<CreditNoteDialog
					{invoiceId}
					onclose={() => {
						showCreditNoteDialog = false;
					}}
					onsuccess={(newInvoice: Invoice) => {
						showCreditNoteDialog = false;
						goto(`/invoices/${newInvoice.id}`);
					}}
				/>
			{/if}

			{#if showSendEmailDialog}
				<SendEmailDialog
					{invoiceId}
					invoiceNumber={invoice.invoice_number}
					customerEmail={invoice.customer?.email}
					onclose={() => {
						showSendEmailDialog = false;
					}}
					onsuccess={() => {
						showSendEmailDialog = false;
					}}
				/>
			{/if}
		{/if}
	{/if}

	{#if invoice}
		<AuditLogPanel entityType="invoice" entityId={invoice.id} />
	{/if}
</div>

<ConfirmDialog
	bind:open={showDeleteConfirm}
	title="Smazat fakturu"
	message="Opravdu chcete smazat tuto fakturu?"
	confirmLabel="Smazat"
	onconfirm={confirmDelete}
	oncancel={() => (showDeleteConfirm = false)}
/>
