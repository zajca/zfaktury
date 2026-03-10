<script lang="ts">
	import { goto } from '$app/navigation';
	import { toISODate } from '$lib/utils/date';
	import { formatCZK, toHalere } from '$lib/utils/money';

	interface Contact {
		id: number;
		name: string;
		ico: string;
	}

	interface FormItem {
		description: string;
		quantity: number;
		unit: string;
		unit_price: number;
		vat_rate_percent: number;
	}

	const API_BASE = '/api/v1';

	let contacts = $state<Contact[]>([]);
	let saving = $state(false);
	let error = $state<string | null>(null);

	let form = $state({
		name: '',
		customer_id: 0,
		frequency: 'monthly',
		next_issue_date: toISODate(new Date()),
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

	let items = $state<FormItem[]>([
		{ description: '', quantity: 1, unit: 'ks', unit_price: 0, vat_rate_percent: 21 }
	]);

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

	$effect(() => {
		loadContacts();
	});

	async function loadContacts() {
		try {
			const res = await fetch(`${API_BASE}/contacts?limit=1000`);
			if (res.ok) {
				const data = await res.json();
				contacts = data.data;
			}
		} catch {
			// Non-critical
		}
	}

	async function handleSubmit() {
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

			const res = await fetch(`${API_BASE}/recurring-invoices`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(body)
			});

			if (!res.ok) {
				const data = await res.json();
				throw new Error(data.error || 'Nepodarilo se vytvorit');
			}

			goto('/recurring');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se vytvorit opakujici se fakturu';
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Nova opakujici se faktura - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-4xl">
	<a href="/recurring" class="text-sm text-blue-600 hover:text-blue-800">&larr; Zpet na opakujici se faktury</a>
	<h1 class="mt-2 text-2xl font-bold text-gray-900">Nova opakujici se faktura</h1>

	{#if error}
		<div class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
			{error}
		</div>
	{/if}

	<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="mt-6 space-y-8">
		<!-- Basic info -->
		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Zakladni udaje</h2>
			<div class="mt-4 space-y-4">
				<div>
					<label for="name" class="block text-sm font-medium text-gray-700">Nazev sablony</label>
					<input id="name" type="text" bind:value={form.name} required class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none" placeholder="napr. Mesicni hosting" />
				</div>
				<div>
					<label for="customer" class="block text-sm font-medium text-gray-700">Zakaznik</label>
					<select
						id="customer"
						bind:value={form.customer_id}
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					>
						<option value={0}>-- Vyberte --</option>
						{#each contacts as contact}
							<option value={contact.id}>{contact.name} {contact.ico ? `(${contact.ico})` : ''}</option>
						{/each}
					</select>
				</div>
			</div>
		</div>

		<!-- Schedule -->
		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Opakovani</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
				<div>
					<label for="frequency" class="block text-sm font-medium text-gray-700">Frekvence</label>
					<select id="frequency" bind:value={form.frequency} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none">
						<option value="weekly">Tydenni</option>
						<option value="monthly">Mesicni</option>
						<option value="quarterly">Ctvrtletni</option>
						<option value="yearly">Rocni</option>
					</select>
				</div>
				<div>
					<label for="next_issue_date" class="block text-sm font-medium text-gray-700">Dalsi vystaveni</label>
					<input id="next_issue_date" type="date" bind:value={form.next_issue_date} required class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none" />
				</div>
				<div>
					<label for="end_date" class="block text-sm font-medium text-gray-700">Konec opakovani (volitelne)</label>
					<input id="end_date" type="date" bind:value={form.end_date} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none" />
				</div>
			</div>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label for="payment" class="block text-sm font-medium text-gray-700">Zpusob platby</label>
					<select id="payment" bind:value={form.payment_method} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none">
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
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
					</svg>
					Pridat polozku
				</button>
			</div>

			<div class="mt-4 space-y-4">
				{#each items as item, index}
					<div class="rounded-lg border border-gray-200 bg-gray-50 p-4">
						<div class="flex items-start gap-4">
							<div class="flex-1 space-y-3">
								<div>
									<label for="desc-{index}" class="block text-sm font-medium text-gray-700">Popis</label>
									<input id="desc-{index}" type="text" bind:value={item.description} required class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white" />
								</div>
								<div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
									<div>
										<label for="qty-{index}" class="block text-sm font-medium text-gray-700">Mnozstvi</label>
										<input id="qty-{index}" type="number" step="0.01" min="0" bind:value={item.quantity} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white" />
									</div>
									<div>
										<label for="unit-{index}" class="block text-sm font-medium text-gray-700">Jednotka</label>
										<select id="unit-{index}" bind:value={item.unit} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white">
											<option value="ks">ks</option>
											<option value="hod">hod</option>
											<option value="m2">m2</option>
											<option value="den">den</option>
											<option value="mesic">mesic</option>
										</select>
									</div>
									<div>
										<label for="price-{index}" class="block text-sm font-medium text-gray-700">Cena/ks (CZK)</label>
										<input id="price-{index}" type="number" step="0.01" min="0" bind:value={item.unit_price} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white" />
									</div>
									<div>
										<label for="vat-{index}" class="block text-sm font-medium text-gray-700">DPH %</label>
										<select id="vat-{index}" bind:value={item.vat_rate_percent} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white">
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
				<label for="notes" class="block text-sm font-medium text-gray-700">Poznamka na fakture</label>
				<textarea id="notes" bind:value={form.notes} rows="2" class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"></textarea>
			</div>
		</div>

		<!-- Actions -->
		<div class="flex gap-3">
			<button
				type="submit"
				disabled={saving}
				class="rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50 transition-colors"
			>
				{saving ? 'Ukladam...' : 'Ulozit'}
			</button>
			<a
				href="/recurring"
				class="rounded-lg border border-gray-300 px-6 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
			>
				Zrusit
			</a>
		</div>
	</form>
</div>
