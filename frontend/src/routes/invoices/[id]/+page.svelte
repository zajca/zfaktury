<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { invoicesApi, contactsApi, type Invoice, type Contact } from '$lib/api/client';
	import { formatCZK, toHalere, fromHalere } from '$lib/utils/money';
	import { formatDate, toISODate, addDays } from '$lib/utils/date';
	import DateInput from '$lib/components/DateInput.svelte';
	import { statusLabels, statusVariant } from '$lib/utils/invoice';
	import InvoiceItemsEditor, { type FormItem } from '$lib/components/InvoiceItemsEditor.svelte';
	import Badge from '$lib/ui/Badge.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';

	let invoice = $state<Invoice | null>(null);
	let contacts = $state<Contact[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state<string | null>(null);
	let editing = $state(false);
	let qrError = $state(false);

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

	onMount(() => {
		loadInvoice();
	});

	async function loadInvoice() {
		loading = true;
		error = null;
		qrError = false;
		try {
			invoice = await invoicesApi.getById(invoiceId);
			populateForm();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst fakturu';
		} finally {
			loading = false;
		}
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

	let dueDateOffset = $state(14);

	function handleIssueDateChange(newValue: string) {
		form.issue_date = newValue;
		if (newValue) form.due_date = addDays(newValue, dueDateOffset);
	}

	function handleDueDateChange(newValue: string) {
		form.due_date = newValue;
		if (form.issue_date && newValue) {
			const diff = (new Date(newValue).getTime() - new Date(form.issue_date).getTime()) / 86400000;
			dueDateOffset = Math.round(diff);
		}
	}

	async function handleSave() {
		saving = true;
		error = null;
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
			editing = false;
			await loadInvoice();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se uložit fakturu';
		} finally {
			saving = false;
		}
	}

	async function handleSend() {
		error = null;
		try {
			await invoicesApi.send(invoiceId);
			await loadInvoice();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se odeslat fakturu';
		}
	}

	async function handleMarkPaid() {
		if (!invoice) return;
		error = null;
		try {
			await invoicesApi.markPaid(invoiceId, invoice.total_amount, toISODate(new Date()));
			await loadInvoice();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se označit jako uhrazenou';
		}
	}

	async function handleDuplicate() {
		error = null;
		try {
			const dup = await invoicesApi.duplicate(invoiceId);
			goto(`/invoices/${dup.id}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se duplikovat fakturu';
		}
	}

	async function handleDelete() {
		if (!confirm('Opravdu chcete smazat tuto fakturu?')) return;
		error = null;
		try {
			await invoicesApi.delete(invoiceId);
			goto('/invoices');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se smazat fakturu';
		}
	}
</script>

<svelte:head>
	<title>{invoice ? `Faktura ${invoice.invoice_number}` : 'Faktura'} - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<a href="/invoices" class="text-sm text-secondary hover:text-primary">&larr; Zpět na faktury</a>

	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-danger/20 bg-danger-bg p-4 text-sm text-danger"
		>
			{error}
		</div>
	{/if}

	{#if loading}
		<div class="mt-8 flex items-center justify-center">
			<div role="status">
				<div
					class="h-8 w-8 animate-spin rounded-full border-4 border-border border-t-accent"
				></div>
				<span class="sr-only">Nacitani...</span>
			</div>
		</div>
	{:else if invoice}
		<!-- Header -->
		<div class="mt-4">
			<div class="flex items-center justify-between">
				<h1 class="text-xl font-semibold text-primary">Faktura {invoice.invoice_number}</h1>
				<div class="flex items-center gap-3">
					<Badge variant={statusVariant[invoice.status] ?? 'default'}>
						{statusLabels[invoice.status] ?? invoice.status}
					</Badge>
					{#if invoice.customer}
						<span class="text-sm text-secondary">{invoice.customer.name}</span>
					{/if}
				</div>
			</div>
			{#if !editing}
				<div class="mt-3 flex flex-wrap gap-2">
					{#if invoice.status === 'draft'}
						<Button variant="secondary" onclick={startEditing}>
							Upravit
						</Button>
						<Button variant="primary" onclick={handleSend}>
							Odeslat
						</Button>
					{/if}
					{#if invoice.status === 'sent' || invoice.status === 'overdue'}
						<Button variant="success" onclick={handleMarkPaid}>
							Uhrazená
						</Button>
					{/if}
					<Button variant="secondary" href={invoicesApi.getPdfUrl(invoiceId)}>
						<svg
							class="h-4 w-4"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
							/>
						</svg>
						Stáhnout PDF
					</Button>
					<Button variant="secondary" href={invoicesApi.getIsdocUrl(invoiceId)}>
						Export ISDOC
					</Button>
					<Button variant="secondary" onclick={handleDuplicate}>
						Duplikovat
					</Button>
					{#if invoice.status !== 'paid'}
						<Button variant="danger" onclick={handleDelete}>
							Smazat
						</Button>
					{/if}
				</div>
			{/if}
		</div>

		{#if editing}
			<!-- Edit mode -->
			<form
				onsubmit={(e) => {
					e.preventDefault();
					handleSave();
				}}
				class="mt-6 space-y-6"
			>
				<!-- Customer -->
				<Card>
					<h2 class="text-base font-semibold text-primary">Zákazník</h2>
					<div class="mt-4">
						<select
							bind:value={form.customer_id}
							class="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						>
							<option value={0}>-- Vyberte --</option>
							{#each contacts as contact (contact.id)}
								<option value={contact.id}
									>{contact.name} {contact.ico ? `(${contact.ico})` : ''}</option
								>
							{/each}
						</select>
					</div>
				</Card>

				<!-- Dates -->
				<Card>
					<h2 class="text-base font-semibold text-primary">Údaje faktury</h2>
					<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<label for="edit-issue" class="block text-sm font-medium text-secondary"
								>Datum vystavení</label
							>
							<DateInput
								id="edit-issue"
								bind:value={form.issue_date}
								required
								onchange={handleIssueDateChange}
							/>
						</div>
						<div>
							<label for="edit-due" class="block text-sm font-medium text-secondary"
								>Datum splatnosti</label
							>
							<DateInput
								id="edit-due"
								bind:value={form.due_date}
								required
								onchange={handleDueDateChange}
								presets={[
									{ label: '+7 dni', days: 7 },
									{ label: '+14 dni', days: 14 },
									{ label: '+30 dni', days: 30 },
									{ label: '+60 dni', days: 60 }
								]}
								relativeToValue={form.issue_date}
							/>
						</div>
						<div>
							<label for="edit-delivery" class="block text-sm font-medium text-secondary">DUZP</label
							>
							<DateInput id="edit-delivery" bind:value={form.delivery_date} />
						</div>
					</div>
					<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div>
							<label for="edit-vs" class="block text-sm font-medium text-secondary"
								>Variabilní symbol</label
							>
							<input
								id="edit-vs"
								type="text"
								bind:value={form.variable_symbol}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div>
							<label for="edit-payment" class="block text-sm font-medium text-secondary"
								>Způsob platby</label
							>
							<select
								id="edit-payment"
								bind:value={form.payment_method}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							>
								<option value="bank_transfer">Bankovní převod</option>
								<option value="cash">Hotovost</option>
								<option value="card">Karta</option>
							</select>
						</div>
					</div>
				</Card>

				<!-- Items -->
				<InvoiceItemsEditor bind:items idPrefix="edit-" />

				<!-- Notes -->
				<Card>
					<h2 class="text-base font-semibold text-primary">Poznámky</h2>
					<div class="mt-4 space-y-4">
						<div>
							<label for="edit-notes" class="block text-sm font-medium text-secondary"
								>Poznámka na faktuře</label
							>
							<textarea
								id="edit-notes"
								bind:value={form.notes}
								rows="2"
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							></textarea>
						</div>
						<div>
							<label for="edit-internal" class="block text-sm font-medium text-secondary"
								>Interní poznámka</label
							>
							<textarea
								id="edit-internal"
								bind:value={form.internal_notes}
								rows="2"
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							></textarea>
						</div>
					</div>
				</Card>

				<!-- Actions -->
				<div class="flex gap-3">
					<Button type="submit" variant="primary" disabled={saving}>
						{saving ? 'Ukládám...' : 'Uložit změny'}
					</Button>
					<Button variant="secondary" onclick={cancelEditing}>
						Zrušit
					</Button>
				</div>
			</form>
		{:else}
			<!-- View mode -->
			<div class="mt-6 space-y-6">
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
									<a
										href="/contacts/{invoice.customer.id}"
										class="text-accent-text hover:text-accent">{invoice.customer.name}</a
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

				<!-- Items -->
				<Card>
					<h2 class="text-base font-semibold text-primary">Položky</h2>
					<div class="mt-4 overflow-x-auto">
						<table class="w-full text-left text-sm">
							<thead class="border-b border-border">
								<tr>
									<th class="pb-2 text-xs font-medium uppercase tracking-wider text-muted">Popis</th>
									<th class="pb-2 text-right text-xs font-medium uppercase tracking-wider text-muted">Množství</th>
									<th class="pb-2 text-xs font-medium uppercase tracking-wider text-muted">Jednotka</th>
									<th class="pb-2 text-right text-xs font-medium uppercase tracking-wider text-muted">Cena/ks</th>
									<th class="pb-2 text-right text-xs font-medium uppercase tracking-wider text-muted">DPH %</th>
									<th class="pb-2 text-right text-xs font-medium uppercase tracking-wider text-muted">DPH</th>
									<th class="pb-2 text-right text-xs font-medium uppercase tracking-wider text-muted">Celkem</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-border-subtle">
								{#each invoice.items ?? [] as item (item.id)}
									<tr>
										<td class="py-2.5 text-primary">{item.description}</td>
										<td class="py-2.5 text-right font-mono tabular-nums text-secondary">{fromHalere(item.quantity)}</td>
										<td class="py-2.5 text-secondary">{item.unit}</td>
										<td class="py-2.5 text-right font-mono tabular-nums text-secondary">{formatCZK(item.unit_price)}</td>
										<td class="py-2.5 text-right font-mono tabular-nums text-secondary">{item.vat_rate_percent}%</td>
										<td class="py-2.5 text-right font-mono tabular-nums text-secondary">{formatCZK(item.vat_amount)}</td>
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
								<span class="font-medium font-mono tabular-nums text-primary">{formatCZK(invoice.subtotal_amount)}</span>
							</div>
							<div class="flex gap-8">
								<span class="text-secondary">DPH:</span>
								<span class="font-medium font-mono tabular-nums text-primary">{formatCZK(invoice.vat_amount)}</span>
							</div>
							<div class="flex gap-8 border-t border-border pt-1 text-base">
								<span class="font-semibold text-primary">Celkem:</span>
								<span class="font-bold font-mono tabular-nums text-primary">{formatCZK(invoice.total_amount)}</span>
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

				<!-- Payment info with QR code -->
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

				<!-- Timestamps -->
				<div class="text-xs text-muted">
					Vytvořeno: {formatDate(invoice.created_at)} | Upraveno: {formatDate(invoice.updated_at)}
					{#if invoice.sent_at}
						| Odesláno: {formatDate(invoice.sent_at)}{/if}
					{#if invoice.paid_at}
						| Uhrazeno: {formatDate(invoice.paid_at)}{/if}
				</div>
			</div>
		{/if}
	{/if}
</div>
