<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { formatDate } from '$lib/utils/date';
	import { formatCZK, toHalere, fromHalere } from '$lib/utils/money';

	interface Contact {
		id: number;
		name: string;
		ico: string;
	}

	interface RecurringInvoiceItem {
		id: number;
		recurring_invoice_id: number;
		description: string;
		quantity: number;
		unit: string;
		unit_price: number;
		vat_rate_percent: number;
		sort_order: number;
	}

	interface RecurringInvoice {
		id: number;
		name: string;
		customer_id: number;
		customer?: Contact;
		frequency: string;
		next_issue_date: string;
		end_date?: string;
		currency_code: string;
		exchange_rate: number;
		payment_method: string;
		bank_account: string;
		bank_code: string;
		iban: string;
		swift: string;
		constant_symbol: string;
		notes: string;
		is_active: boolean;
		items: RecurringInvoiceItem[];
		created_at: string;
		updated_at: string;
	}

	interface FormItem {
		description: string;
		quantity: number;
		unit: string;
		unit_price: number;
		vat_rate_percent: number;
	}

	const API_BASE = '/api/v1';

	let id = $derived(Number(page.params.id));
	let contacts = $state<Contact[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let generating = $state(false);
	let error = $state<string | null>(null);
	let editing = $state(false);

	let recurringInvoice = $state<RecurringInvoice | null>(null);

	let form = $state({
		name: '',
		customer_id: 0,
		frequency: 'monthly',
		next_issue_date: '',
		end_date: '',
		currency_code: 'CZK',
		payment_method: 'bank_transfer',
		bank_account: '',
		bank_code: '',
		iban: '',
		swift: '',
		constant_symbol: '',
		notes: '',
		is_active: true
	});

	let items = $state<FormItem[]>([]);

	const frequencyLabels: Record<string, string> = {
		weekly: 'Tydenni',
		monthly: 'Mesicni',
		quarterly: 'Ctvrtletni',
		yearly: 'Rocni'
	};

	let subtotal = $derived(
		items.reduce((sum, item) => sum + item.quantity * item.unit_price, 0)
	);

	let vatTotal = $derived(
		items.reduce((sum, item) => {
			const itemSubtotal = item.quantity * item.unit_price;
			return sum + itemSubtotal * (item.vat_rate_percent / 100);
		}, 0)
	);

	let grandTotal = $derived(subtotal + vatTotal);

	function addItem() {
		items = [...items, { description: '', quantity: 1, unit: 'ks', unit_price: 0, vat_rate_percent: 21 }];
	}

	function removeItem(index: number) {
		if (items.length <= 1) return;
		items = items.filter((_, i) => i !== index);
	}

	function startEdit() {
		if (!recurringInvoice) return;
		form = {
			name: recurringInvoice.name,
			customer_id: recurringInvoice.customer_id,
			frequency: recurringInvoice.frequency,
			next_issue_date: recurringInvoice.next_issue_date,
			end_date: recurringInvoice.end_date ?? '',
			currency_code: recurringInvoice.currency_code,
			payment_method: recurringInvoice.payment_method,
			bank_account: recurringInvoice.bank_account,
			bank_code: recurringInvoice.bank_code,
			iban: recurringInvoice.iban,
			swift: recurringInvoice.swift,
			constant_symbol: recurringInvoice.constant_symbol,
			notes: recurringInvoice.notes,
			is_active: recurringInvoice.is_active
		};
		items = (recurringInvoice.items ?? []).map(item => ({
			description: item.description,
			quantity: item.quantity / 100,
			unit: item.unit,
			unit_price: fromHalere(item.unit_price),
			vat_rate_percent: item.vat_rate_percent
		}));
		if (items.length === 0) {
			items = [{ description: '', quantity: 1, unit: 'ks', unit_price: 0, vat_rate_percent: 21 }];
		}
		editing = true;
	}

	async function loadData() {
		loading = true;
		error = null;
		try {
			const [riRes, contactsRes] = await Promise.all([
				fetch(`${API_BASE}/recurring-invoices/${id}`),
				fetch(`${API_BASE}/contacts?limit=1000`)
			]);

			if (!riRes.ok) throw new Error('Nepodarilo se nacist opakujici se fakturu');
			recurringInvoice = await riRes.json();

			if (contactsRes.ok) {
				const data = await contactsRes.json();
				contacts = data.data;
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se nacist data';
		} finally {
			loading = false;
		}
	}

	async function handleSave() {
		if (!form.name.trim()) {
			error = 'Zadejte nazev';
			return;
		}
		if (!form.customer_id) {
			error = 'Vyberte zakaznika';
			return;
		}

		saving = true;
		error = null;

		try {
			const requestItems = items.map((item, index) => ({
				description: item.description,
				quantity: Math.round(item.quantity * 100),
				unit: item.unit,
				unit_price: toHalere(item.unit_price),
				vat_rate_percent: item.vat_rate_percent,
				sort_order: index
			}));

			const body = {
				...form,
				end_date: form.end_date || null,
				exchange_rate: 0,
				items: requestItems
			};

			const res = await fetch(`${API_BASE}/recurring-invoices/${id}`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(body)
			});

			if (!res.ok) {
				const data = await res.json();
				throw new Error(data.error || 'Nepodarilo se ulozit');
			}

			recurringInvoice = await res.json();
			editing = false;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se ulozit zmeny';
		} finally {
			saving = false;
		}
	}

	async function generateInvoice() {
		generating = true;
		error = null;
		try {
			const res = await fetch(`${API_BASE}/recurring-invoices/${id}/generate`, { method: 'POST' });
			if (!res.ok) {
				const data = await res.json();
				throw new Error(data.error || 'Nepodarilo se vygenerovat fakturu');
			}
			const invoice = await res.json();
			goto(`/invoices/${invoice.id}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se vygenerovat fakturu';
		} finally {
			generating = false;
		}
	}

	$effect(() => {
		id;
		loadData();
	});
</script>

<svelte:head>
	<title>{recurringInvoice?.name ?? 'Detail'} - Opakujici se faktura - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-4xl">
	<a href="/recurring" class="text-sm text-blue-600 hover:text-blue-800">&larr; Zpet na opakujici se faktury</a>

	{#if loading}
		<div class="mt-8 flex items-center justify-center p-12">
			<div class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"></div>
		</div>
	{:else if error && !recurringInvoice}
		<div class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
			{error}
		</div>
	{:else if recurringInvoice && !editing}
		<!-- Detail view -->
		<div class="mt-2 flex items-center justify-between">
			<h1 class="text-2xl font-bold text-gray-900">{recurringInvoice.name}</h1>
			<div class="flex gap-3">
				<button
					onclick={generateInvoice}
					disabled={generating}
					class="inline-flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 transition-colors"
				>
					{generating ? 'Generuji...' : 'Vygenerovat fakturu'}
				</button>
				<button
					onclick={startEdit}
					class="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 transition-colors"
				>
					Upravit
				</button>
			</div>
		</div>

		{#if error}
			<div class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
				{error}
			</div>
		{/if}

		<div class="mt-6 space-y-6">
			<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
				<h2 class="text-lg font-semibold text-gray-900">Zakladni udaje</h2>
				<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div>
						<dt class="text-sm text-gray-500">Zakaznik</dt>
						<dd class="mt-1 text-sm font-medium text-gray-900">{recurringInvoice.customer?.name ?? '-'}</dd>
					</div>
					<div>
						<dt class="text-sm text-gray-500">Stav</dt>
						<dd class="mt-1">
							<span class="inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium {recurringInvoice.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}">
								{recurringInvoice.is_active ? 'Aktivni' : 'Neaktivni'}
							</span>
						</dd>
					</div>
					<div>
						<dt class="text-sm text-gray-500">Frekvence</dt>
						<dd class="mt-1 text-sm font-medium text-gray-900">{frequencyLabels[recurringInvoice.frequency] ?? recurringInvoice.frequency}</dd>
					</div>
					<div>
						<dt class="text-sm text-gray-500">Dalsi vystaveni</dt>
						<dd class="mt-1 text-sm font-medium text-gray-900">{formatDate(recurringInvoice.next_issue_date)}</dd>
					</div>
					{#if recurringInvoice.end_date}
						<div>
							<dt class="text-sm text-gray-500">Konec opakovani</dt>
							<dd class="mt-1 text-sm font-medium text-gray-900">{formatDate(recurringInvoice.end_date)}</dd>
						</div>
					{/if}
					<div>
						<dt class="text-sm text-gray-500">Zpusob platby</dt>
						<dd class="mt-1 text-sm font-medium text-gray-900">{recurringInvoice.payment_method === 'bank_transfer' ? 'Bankovni prevod' : recurringInvoice.payment_method === 'cash' ? 'Hotovost' : 'Karta'}</dd>
					</div>
				</dl>
				{#if recurringInvoice.notes}
					<div class="mt-4">
						<dt class="text-sm text-gray-500">Poznamka</dt>
						<dd class="mt-1 text-sm text-gray-900">{recurringInvoice.notes}</dd>
					</div>
				{/if}
			</div>

			<!-- Items -->
			<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
				<h2 class="text-lg font-semibold text-gray-900">Polozky</h2>
				{#if recurringInvoice.items && recurringInvoice.items.length > 0}
					<table class="mt-4 w-full text-left text-sm">
						<thead class="border-b border-gray-200 bg-gray-50">
							<tr>
								<th class="px-4 py-2 font-medium text-gray-600">Popis</th>
								<th class="px-4 py-2 text-right font-medium text-gray-600">Mnozstvi</th>
								<th class="px-4 py-2 font-medium text-gray-600">Jednotka</th>
								<th class="px-4 py-2 text-right font-medium text-gray-600">Cena/ks</th>
								<th class="px-4 py-2 text-right font-medium text-gray-600">DPH</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-gray-100">
							{#each recurringInvoice.items as item}
								<tr>
									<td class="px-4 py-2 text-gray-900">{item.description}</td>
									<td class="px-4 py-2 text-right text-gray-700">{(item.quantity / 100).toFixed(2)}</td>
									<td class="px-4 py-2 text-gray-700">{item.unit}</td>
									<td class="px-4 py-2 text-right text-gray-700">{formatCZK(item.unit_price)}</td>
									<td class="px-4 py-2 text-right text-gray-700">{item.vat_rate_percent}%</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{:else}
					<p class="mt-4 text-sm text-gray-400">Zadne polozky</p>
				{/if}
			</div>
		</div>
	{:else if editing}
		<!-- Edit form -->
		<h1 class="mt-2 text-2xl font-bold text-gray-900">Upravit: {recurringInvoice?.name}</h1>

		{#if error}
			<div class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
				{error}
			</div>
		{/if}

		<form onsubmit={(e) => { e.preventDefault(); handleSave(); }} class="mt-6 space-y-8">
			<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
				<h2 class="text-lg font-semibold text-gray-900">Zakladni udaje</h2>
				<div class="mt-4 space-y-4">
					<div>
						<label for="edit-name" class="block text-sm font-medium text-gray-700">Nazev sablony</label>
						<input id="edit-name" type="text" bind:value={form.name} required class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none" />
					</div>
					<div>
						<label for="edit-customer" class="block text-sm font-medium text-gray-700">Zakaznik</label>
						<select id="edit-customer" bind:value={form.customer_id} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none">
							<option value={0}>-- Vyberte --</option>
							{#each contacts as contact}
								<option value={contact.id}>{contact.name} {contact.ico ? `(${contact.ico})` : ''}</option>
							{/each}
						</select>
					</div>
					<div class="flex items-center gap-2">
						<input id="edit-active" type="checkbox" bind:checked={form.is_active} class="rounded border-gray-300" />
						<label for="edit-active" class="text-sm font-medium text-gray-700">Aktivni</label>
					</div>
				</div>
			</div>

			<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
				<h2 class="text-lg font-semibold text-gray-900">Opakovani</h2>
				<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
					<div>
						<label for="edit-frequency" class="block text-sm font-medium text-gray-700">Frekvence</label>
						<select id="edit-frequency" bind:value={form.frequency} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none">
							<option value="weekly">Tydenni</option>
							<option value="monthly">Mesicni</option>
							<option value="quarterly">Ctvrtletni</option>
							<option value="yearly">Rocni</option>
						</select>
					</div>
					<div>
						<label for="edit-next-date" class="block text-sm font-medium text-gray-700">Dalsi vystaveni</label>
						<input id="edit-next-date" type="date" bind:value={form.next_issue_date} required class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none" />
					</div>
					<div>
						<label for="edit-end-date" class="block text-sm font-medium text-gray-700">Konec opakovani</label>
						<input id="edit-end-date" type="date" bind:value={form.end_date} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none" />
					</div>
				</div>
				<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div>
						<label for="edit-payment" class="block text-sm font-medium text-gray-700">Zpusob platby</label>
						<select id="edit-payment" bind:value={form.payment_method} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none">
							<option value="bank_transfer">Bankovni prevod</option>
							<option value="cash">Hotovost</option>
							<option value="card">Karta</option>
						</select>
					</div>
				</div>
			</div>

			<!-- Line Items -->
			<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
				<div class="flex items-center justify-between">
					<h2 class="text-lg font-semibold text-gray-900">Polozky</h2>
					<button
						type="button"
						onclick={addItem}
						class="inline-flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
					>
						Pridat polozku
					</button>
				</div>

				<div class="mt-4 space-y-4">
					{#each items as item, index}
						<div class="rounded-lg border border-gray-200 bg-gray-50 p-4">
							<div class="flex items-start gap-4">
								<div class="flex-1 space-y-3">
									<div>
										<label for="edit-desc-{index}" class="block text-sm font-medium text-gray-700">Popis</label>
										<input id="edit-desc-{index}" type="text" bind:value={item.description} required class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white" />
									</div>
									<div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
										<div>
											<label for="edit-qty-{index}" class="block text-sm font-medium text-gray-700">Mnozstvi</label>
											<input id="edit-qty-{index}" type="number" step="0.01" min="0" bind:value={item.quantity} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white" />
										</div>
										<div>
											<label for="edit-unit-{index}" class="block text-sm font-medium text-gray-700">Jednotka</label>
											<select id="edit-unit-{index}" bind:value={item.unit} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white">
												<option value="ks">ks</option>
												<option value="hod">hod</option>
												<option value="m2">m2</option>
												<option value="den">den</option>
												<option value="mesic">mesic</option>
											</select>
										</div>
										<div>
											<label for="edit-price-{index}" class="block text-sm font-medium text-gray-700">Cena/ks (CZK)</label>
											<input id="edit-price-{index}" type="number" step="0.01" min="0" bind:value={item.unit_price} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white" />
										</div>
										<div>
											<label for="edit-vat-{index}" class="block text-sm font-medium text-gray-700">DPH %</label>
											<select id="edit-vat-{index}" bind:value={item.vat_rate_percent} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white">
												<option value={21}>21%</option>
												<option value={12}>12%</option>
												<option value={0}>0%</option>
											</select>
										</div>
									</div>
								</div>
								{#if items.length > 1}
									<button
										type="button"
										onclick={() => removeItem(index)}
										class="mt-6 rounded p-1 text-gray-400 hover:text-red-500 transition-colors"
										aria-label="Odebrat polozku"
									>
										<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
											<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
										</svg>
									</button>
								{/if}
							</div>
							<div class="mt-2 text-right text-sm text-gray-500">
								Zaklad: {formatCZK(toHalere(item.quantity * item.unit_price))} | DPH: {formatCZK(toHalere(item.quantity * item.unit_price * item.vat_rate_percent / 100))} | Celkem: {formatCZK(toHalere(item.quantity * item.unit_price * (1 + item.vat_rate_percent / 100)))}
							</div>
						</div>
					{/each}
				</div>

				<div class="mt-6 border-t border-gray-200 pt-4">
					<div class="flex flex-col items-end gap-1 text-sm">
						<div class="flex gap-8">
							<span class="text-gray-600">Zaklad:</span>
							<span class="font-medium text-gray-900">{formatCZK(toHalere(subtotal))}</span>
						</div>
						<div class="flex gap-8">
							<span class="text-gray-600">DPH:</span>
							<span class="font-medium text-gray-900">{formatCZK(toHalere(vatTotal))}</span>
						</div>
						<div class="flex gap-8 border-t border-gray-200 pt-1 text-base">
							<span class="font-semibold text-gray-900">Celkem:</span>
							<span class="font-bold text-gray-900">{formatCZK(toHalere(grandTotal))}</span>
						</div>
					</div>
				</div>
			</div>

			<!-- Notes -->
			<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
				<h2 class="text-lg font-semibold text-gray-900">Poznamky</h2>
				<div class="mt-4">
					<label for="edit-notes" class="block text-sm font-medium text-gray-700">Poznamka na fakture</label>
					<textarea id="edit-notes" bind:value={form.notes} rows="2" class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"></textarea>
				</div>
			</div>

			<div class="flex gap-3">
				<button
					type="submit"
					disabled={saving}
					class="rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50 transition-colors"
				>
					{saving ? 'Ukladam...' : 'Ulozit zmeny'}
				</button>
				<button
					type="button"
					onclick={() => { editing = false; error = null; }}
					class="rounded-lg border border-gray-300 px-6 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
				>
					Zrusit
				</button>
			</div>
		</form>
	{/if}
</div>
