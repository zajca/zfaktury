<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { invoicesApi, contactsApi, type Invoice, type Contact } from '$lib/api/client';
	import { formatCZK, toHalere, fromHalere } from '$lib/utils/money';
	import { formatDate, toISODate, addDays } from '$lib/utils/date';
	import DateInput from '$lib/components/DateInput.svelte';
	import { statusLabels, statusColors } from '$lib/utils/invoice';
	import InvoiceItemsEditor, {
		calcSubtotal,
		calcVatTotal,
		calcGrandTotal,
		type FormItem
	} from '$lib/components/InvoiceItemsEditor.svelte';

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

<div class="mx-auto max-w-4xl">
	<a href="/invoices" class="text-sm text-blue-600 hover:text-blue-800">&larr; Zpět na faktury</a>

	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700"
		>
			{error}
		</div>
	{/if}

	{#if loading}
		<div class="mt-8 flex items-center justify-center">
			<div role="status">
				<div
					class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"
				></div>
				<span class="sr-only">Nacitani...</span>
			</div>
		</div>
	{:else if invoice}
		<!-- Header -->
		<div class="mt-4 flex items-start justify-between">
			<div>
				<h1 class="text-2xl font-bold text-gray-900">Faktura {invoice.invoice_number}</h1>
				<div class="mt-2 flex items-center gap-3">
					<span
						class="inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium {statusColors[
							invoice.status
						] ?? 'bg-gray-100 text-gray-700'}"
					>
						{statusLabels[invoice.status] ?? invoice.status}
					</span>
					{#if invoice.customer}
						<span class="text-sm text-gray-600">{invoice.customer.name}</span>
					{/if}
				</div>
			</div>
			<div class="flex flex-wrap gap-2">
				{#if invoice.status === 'draft' && !editing}
					<button
						onclick={startEditing}
						class="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
					>
						Upravit
					</button>
					<button
						onclick={handleSend}
						class="rounded-lg bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700 transition-colors"
					>
						Odeslat
					</button>
				{/if}
				{#if invoice.status === 'sent' || invoice.status === 'overdue'}
					<button
						onclick={handleMarkPaid}
						class="rounded-lg bg-green-600 px-3 py-2 text-sm font-medium text-white hover:bg-green-700 transition-colors"
					>
						Uhrazená
					</button>
				{/if}
				{#if !editing}
					<a
						href={invoicesApi.getPdfUrl(invoiceId)}
						target="_blank"
						rel="noopener"
						class="inline-flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
					>
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
					</a>
					<a
						href={invoicesApi.getIsdocUrl(invoiceId)}
						download
						class="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors inline-flex items-center gap-1"
					>
						Export ISDOC
					</a>
					<button
						onclick={handleDuplicate}
						class="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
					>
						Duplikovat
					</button>
					{#if invoice.status !== 'paid'}
						<button
							onclick={handleDelete}
							class="rounded-lg border border-red-300 px-3 py-2 text-sm font-medium text-red-600 hover:bg-red-50 transition-colors"
						>
							Smazat
						</button>
					{/if}
				{/if}
			</div>
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
				<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
					<h2 class="text-lg font-semibold text-gray-900">Zákazník</h2>
					<div class="mt-4">
						<select
							bind:value={form.customer_id}
							class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
						>
							<option value={0}>-- Vyberte --</option>
							{#each contacts as contact}
								<option value={contact.id}
									>{contact.name} {contact.ico ? `(${contact.ico})` : ''}</option
								>
							{/each}
						</select>
					</div>
				</div>

				<!-- Dates -->
				<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
					<h2 class="text-lg font-semibold text-gray-900">Údaje faktury</h2>
					<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<label for="edit-issue" class="block text-sm font-medium text-gray-700"
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
							<label for="edit-due" class="block text-sm font-medium text-gray-700"
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
							<label for="edit-delivery" class="block text-sm font-medium text-gray-700">DUZP</label
							>
							<DateInput id="edit-delivery" bind:value={form.delivery_date} />
						</div>
					</div>
					<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div>
							<label for="edit-vs" class="block text-sm font-medium text-gray-700"
								>Variabilní symbol</label
							>
							<input
								id="edit-vs"
								type="text"
								bind:value={form.variable_symbol}
								class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
							/>
						</div>
						<div>
							<label for="edit-payment" class="block text-sm font-medium text-gray-700"
								>Způsob platby</label
							>
							<select
								id="edit-payment"
								bind:value={form.payment_method}
								class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
							>
								<option value="bank_transfer">Bankovní převod</option>
								<option value="cash">Hotovost</option>
								<option value="card">Karta</option>
							</select>
						</div>
					</div>
				</div>

				<!-- Items -->
				<InvoiceItemsEditor bind:items idPrefix="edit-" />

				<!-- Notes -->
				<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
					<h2 class="text-lg font-semibold text-gray-900">Poznámky</h2>
					<div class="mt-4 space-y-4">
						<div>
							<label for="edit-notes" class="block text-sm font-medium text-gray-700"
								>Poznámka na faktuře</label
							>
							<textarea
								id="edit-notes"
								bind:value={form.notes}
								rows="2"
								class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
							></textarea>
						</div>
						<div>
							<label for="edit-internal" class="block text-sm font-medium text-gray-700"
								>Interní poznámka</label
							>
							<textarea
								id="edit-internal"
								bind:value={form.internal_notes}
								rows="2"
								class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
							></textarea>
						</div>
					</div>
				</div>

				<!-- Actions -->
				<div class="flex gap-3">
					<button
						type="submit"
						disabled={saving}
						class="rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50 transition-colors"
					>
						{saving ? 'Ukládám...' : 'Uložit změny'}
					</button>
					<button
						type="button"
						onclick={cancelEditing}
						class="rounded-lg border border-gray-300 px-6 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
					>
						Zrušit
					</button>
				</div>
			</form>
		{:else}
			<!-- View mode -->
			<div class="mt-6 space-y-6">
				<!-- Invoice details -->
				<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
					<h2 class="text-lg font-semibold text-gray-900">Údaje faktury</h2>
					<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<dt class="text-sm font-medium text-gray-500">Datum vystavení</dt>
							<dd class="mt-1 text-sm text-gray-900">{formatDate(invoice.issue_date)}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-gray-500">Datum splatnosti</dt>
							<dd class="mt-1 text-sm text-gray-900">{formatDate(invoice.due_date)}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-gray-500">DUZP</dt>
							<dd class="mt-1 text-sm text-gray-900">{formatDate(invoice.delivery_date)}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-gray-500">Variabilní symbol</dt>
							<dd class="mt-1 text-sm text-gray-900">{invoice.variable_symbol || '-'}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-gray-500">Konstantní symbol</dt>
							<dd class="mt-1 text-sm text-gray-900">{invoice.constant_symbol || '-'}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-gray-500">Způsob platby</dt>
							<dd class="mt-1 text-sm text-gray-900">
								{#if invoice.payment_method === 'bank_transfer'}Bankovní převod
								{:else if invoice.payment_method === 'cash'}Hotovost
								{:else if invoice.payment_method === 'card'}Karta
								{:else}{invoice.payment_method}
								{/if}
							</dd>
						</div>
					</dl>
				</div>

				<!-- Customer -->
				{#if invoice.customer}
					<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
						<h2 class="text-lg font-semibold text-gray-900">Zákazník</h2>
						<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
							<div>
								<dt class="text-sm font-medium text-gray-500">Název</dt>
								<dd class="mt-1 text-sm text-gray-900">
									<a
										href="/contacts/{invoice.customer.id}"
										class="text-blue-600 hover:text-blue-800">{invoice.customer.name}</a
									>
								</dd>
							</div>
							{#if invoice.customer.ico}
								<div>
									<dt class="text-sm font-medium text-gray-500">IČO</dt>
									<dd class="mt-1 text-sm text-gray-900">{invoice.customer.ico}</dd>
								</div>
							{/if}
							{#if invoice.customer.dic}
								<div>
									<dt class="text-sm font-medium text-gray-500">DIČ</dt>
									<dd class="mt-1 text-sm text-gray-900">{invoice.customer.dic}</dd>
								</div>
							{/if}
						</dl>
					</div>
				{/if}

				<!-- Items -->
				<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
					<h2 class="text-lg font-semibold text-gray-900">Položky</h2>
					<div class="mt-4 overflow-x-auto">
						<table class="w-full text-left text-sm">
							<thead class="border-b border-gray-200">
								<tr>
									<th class="pb-2 font-medium text-gray-600">Popis</th>
									<th class="pb-2 text-right font-medium text-gray-600">Množství</th>
									<th class="pb-2 font-medium text-gray-600">Jednotka</th>
									<th class="pb-2 text-right font-medium text-gray-600">Cena/ks</th>
									<th class="pb-2 text-right font-medium text-gray-600">DPH %</th>
									<th class="pb-2 text-right font-medium text-gray-600">DPH</th>
									<th class="pb-2 text-right font-medium text-gray-600">Celkem</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-gray-100">
								{#each invoice.items ?? [] as item}
									<tr>
										<td class="py-2 text-gray-900">{item.description}</td>
										<td class="py-2 text-right text-gray-700">{fromHalere(item.quantity)}</td>
										<td class="py-2 text-gray-700">{item.unit}</td>
										<td class="py-2 text-right text-gray-700">{formatCZK(item.unit_price)}</td>
										<td class="py-2 text-right text-gray-700">{item.vat_rate_percent}%</td>
										<td class="py-2 text-right text-gray-700">{formatCZK(item.vat_amount)}</td>
										<td class="py-2 text-right font-medium text-gray-900"
											>{formatCZK(item.total_amount)}</td
										>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>

					<div class="mt-4 border-t border-gray-200 pt-4">
						<div class="flex flex-col items-end gap-1 text-sm">
							<div class="flex gap-8">
								<span class="text-gray-600">Základ:</span>
								<span class="font-medium text-gray-900">{formatCZK(invoice.subtotal_amount)}</span>
							</div>
							<div class="flex gap-8">
								<span class="text-gray-600">DPH:</span>
								<span class="font-medium text-gray-900">{formatCZK(invoice.vat_amount)}</span>
							</div>
							<div class="flex gap-8 border-t border-gray-200 pt-1 text-base">
								<span class="font-semibold text-gray-900">Celkem:</span>
								<span class="font-bold text-gray-900">{formatCZK(invoice.total_amount)}</span>
							</div>
							{#if invoice.paid_amount > 0}
								<div class="flex gap-8 text-green-700">
									<span>Uhrazeno:</span>
									<span class="font-medium">{formatCZK(invoice.paid_amount)}</span>
								</div>
							{/if}
						</div>
					</div>
				</div>

				<!-- Payment info with QR code -->
				{#if invoice.bank_account || invoice.iban}
					<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
						<h2 class="text-lg font-semibold text-gray-900">Platební údaje</h2>
						<div class="mt-4 flex flex-col gap-6 sm:flex-row">
							<dl class="flex-1 grid grid-cols-1 gap-4 sm:grid-cols-2">
								{#if invoice.bank_account}
									<div>
										<dt class="text-sm font-medium text-gray-500">Číslo účtu</dt>
										<dd class="mt-1 text-sm text-gray-900">
											{invoice.bank_account}{invoice.bank_code ? `/${invoice.bank_code}` : ''}
										</dd>
									</div>
								{/if}
								{#if invoice.iban}
									<div>
										<dt class="text-sm font-medium text-gray-500">IBAN</dt>
										<dd class="mt-1 text-sm text-gray-900">{invoice.iban}</dd>
									</div>
								{/if}
								{#if invoice.variable_symbol}
									<div>
										<dt class="text-sm font-medium text-gray-500">Variabilní symbol</dt>
										<dd class="mt-1 text-sm text-gray-900">{invoice.variable_symbol}</dd>
									</div>
								{/if}
							</dl>
							{#if invoice.iban && invoice.status !== 'paid'}
								<div class="flex flex-col items-center gap-2">
									<span class="text-sm font-medium text-gray-500">QR platba</span>
									{#if qrError}
										<div
											class="flex h-32 w-32 items-center justify-center rounded border border-gray-200 bg-gray-50 text-xs text-gray-400"
										>
											QR kód není dostupný
										</div>
									{:else}
										<img
											src={invoicesApi.getQrUrl(invoiceId)}
											alt="QR kód pro platbu"
											class="h-32 w-32 rounded border border-gray-200"
											onerror={() => {
												qrError = true;
											}}
										/>
									{/if}
								</div>
							{/if}
						</div>
					</div>
				{/if}

				<!-- Notes -->
				{#if invoice.notes || invoice.internal_notes}
					<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
						<h2 class="text-lg font-semibold text-gray-900">Poznámky</h2>
						{#if invoice.notes}
							<div class="mt-4">
								<h3 class="text-sm font-medium text-gray-500">Poznámka na faktuře</h3>
								<p class="mt-1 text-sm text-gray-900 whitespace-pre-wrap">{invoice.notes}</p>
							</div>
						{/if}
						{#if invoice.internal_notes}
							<div class="mt-4">
								<h3 class="text-sm font-medium text-gray-500">Interní poznámka</h3>
								<p class="mt-1 text-sm text-gray-900 whitespace-pre-wrap">
									{invoice.internal_notes}
								</p>
							</div>
						{/if}
					</div>
				{/if}

				<!-- Timestamps -->
				<div class="text-xs text-gray-400">
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
