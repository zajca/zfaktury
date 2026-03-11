<script lang="ts">
	import { goto } from '$app/navigation';
	import { contactsApi, invoicesApi, type Contact, type InvoiceItem } from '$lib/api/client';
	import { formatCZK, toHalere, fromHalere } from '$lib/utils/money';
	import { toISODate, addDays } from '$lib/utils/date';
	import DateInput from '$lib/components/DateInput.svelte';

	interface FormItem {
		description: string;
		quantity: number;
		unit: string;
		unit_price: number; // in crowns (user input)
		vat_rate_percent: number;
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

	let contacts = $state<Contact[]>([]);
	let saving = $state(false);
	let error = $state<string | null>(null);

	let form = $state({
		customer_id: 0,
		issue_date: toISODate(new Date()),
		due_date: toISODate(new Date(Date.now() + 14 * 24 * 60 * 60 * 1000)),
		delivery_date: toISODate(new Date()),
		variable_symbol: '',
		constant_symbol: '',
		currency_code: 'CZK',
		payment_method: 'bank_transfer',
		notes: '',
		internal_notes: ''
	});

	let items = $state<FormItem[]>([
		{ description: '', quantity: 1, unit: 'ks', unit_price: 0, vat_rate_percent: 21 }
	]);

	// Calculated totals derived from items
	let subtotal = $derived(items.reduce((sum, item) => sum + item.quantity * item.unit_price, 0));

	let vatTotal = $derived(
		items.reduce((sum, item) => {
			const itemSubtotal = item.quantity * item.unit_price;
			return sum + itemSubtotal * (item.vat_rate_percent / 100);
		}, 0)
	);

	let grandTotal = $derived(subtotal + vatTotal);

	function addItem() {
		items = [
			...items,
			{ description: '', quantity: 1, unit: 'ks', unit_price: 0, vat_rate_percent: 21 }
		];
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
			const res = await contactsApi.list({ limit: 1000 });
			contacts = res.data;
		} catch {
			// Contacts loading is non-critical
		}
	}

	async function handleSubmit() {
		if (!form.customer_id) {
			error = 'Vyberte zákazníka';
			return;
		}

		saving = true;
		error = null;

		try {
			const invoiceItems: Partial<InvoiceItem>[] = items.map((item, index) => {
				const quantityInCents = Math.round(item.quantity * 100);
				const unitPriceInHalere = toHalere(item.unit_price);
				const itemSubtotal = item.quantity * item.unit_price;
				const itemVat = itemSubtotal * (item.vat_rate_percent / 100);
				const itemTotal = itemSubtotal + itemVat;

				return {
					description: item.description,
					quantity: quantityInCents,
					unit: item.unit,
					unit_price: unitPriceInHalere,
					vat_rate_percent: item.vat_rate_percent,
					vat_amount: toHalere(itemVat),
					total_amount: toHalere(itemTotal),
					sort_order: index
				};
			});

			await invoicesApi.create({
				...form,
				type: 'regular',
				status: 'draft',
				subtotal_amount: toHalere(subtotal),
				vat_amount: toHalere(vatTotal),
				total_amount: toHalere(grandTotal),
				items: invoiceItems as InvoiceItem[]
			});

			goto('/invoices');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se vytvořit fakturu';
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Nová faktura - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-4xl">
	<a href="/invoices" class="text-sm text-blue-600 hover:text-blue-800">&larr; Zpět na faktury</a>
	<h1 class="mt-2 text-2xl font-bold text-gray-900">Nová faktura</h1>

	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700"
		>
			{error}
		</div>
	{/if}

	<form
		onsubmit={(e) => {
			e.preventDefault();
			handleSubmit();
		}}
		class="mt-6 space-y-8"
	>
		<!-- Customer -->
		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Zákazník</h2>
			<div class="mt-4">
				<label for="customer" class="block text-sm font-medium text-gray-700"
					>Vyberte zákazníka</label
				>
				<select
					id="customer"
					bind:value={form.customer_id}
					class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
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

		<!-- Dates & Symbols -->
		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Údaje faktury</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
				<div>
					<label for="issue_date" class="block text-sm font-medium text-gray-700"
						>Datum vystavení</label
					>
					<DateInput
						id="issue_date"
						bind:value={form.issue_date}
						required
						onchange={handleIssueDateChange}
					/>
				</div>
				<div>
					<label for="due_date" class="block text-sm font-medium text-gray-700"
						>Datum splatnosti</label
					>
					<DateInput
						id="due_date"
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
					<label for="delivery_date" class="block text-sm font-medium text-gray-700">DUZP</label>
					<DateInput id="delivery_date" bind:value={form.delivery_date} />
				</div>
			</div>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label for="vs" class="block text-sm font-medium text-gray-700">Variabilní symbol</label>
					<input
						id="vs"
						type="text"
						bind:value={form.variable_symbol}
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					/>
				</div>
				<div>
					<label for="payment" class="block text-sm font-medium text-gray-700">Způsob platby</label>
					<select
						id="payment"
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

		<!-- Line Items -->
		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<div class="flex items-center justify-between">
				<h2 class="text-lg font-semibold text-gray-900">Položky</h2>
				<button
					type="button"
					onclick={addItem}
					class="inline-flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
				>
					<svg
						class="h-4 w-4"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
					</svg>
					Přidat položku
				</button>
			</div>

			<div class="mt-4 space-y-4">
				{#each items as item, index}
					<div class="rounded-lg border border-gray-200 bg-gray-50 p-4">
						<div class="flex items-start gap-4">
							<div class="flex-1 space-y-3">
								<div>
									<label for="desc-{index}" class="block text-sm font-medium text-gray-700"
										>Popis</label
									>
									<input
										id="desc-{index}"
										type="text"
										bind:value={item.description}
										required
										class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white"
									/>
								</div>
								<div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
									<div>
										<label for="qty-{index}" class="block text-sm font-medium text-gray-700"
											>Množství</label
										>
										<input
											id="qty-{index}"
											type="number"
											step="0.01"
											min="0"
											bind:value={item.quantity}
											class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white"
										/>
									</div>
									<div>
										<label for="unit-{index}" class="block text-sm font-medium text-gray-700"
											>Jednotka</label
										>
										<select
											id="unit-{index}"
											bind:value={item.unit}
											class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white"
										>
											<option value="ks">ks</option>
											<option value="hod">hod</option>
											<option value="m2">m2</option>
											<option value="den">den</option>
											<option value="mesic">měsíc</option>
										</select>
									</div>
									<div>
										<label for="price-{index}" class="block text-sm font-medium text-gray-700"
											>Cena/ks (CZK)</label
										>
										<input
											id="price-{index}"
											type="number"
											step="0.01"
											min="0"
											bind:value={item.unit_price}
											class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white"
										/>
									</div>
									<div>
										<label for="vat-{index}" class="block text-sm font-medium text-gray-700"
											>DPH %</label
										>
										<select
											id="vat-{index}"
											bind:value={item.vat_rate_percent}
											class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white"
										>
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
									aria-label="Odebrat položku"
								>
									<svg
										class="h-5 w-5"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										stroke-width="2"
									>
										<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
									</svg>
								</button>
							{/if}
						</div>
						<!-- Item subtotal -->
						<div class="mt-2 text-right text-sm text-gray-500">
							Základ: {formatCZK(toHalere(item.quantity * item.unit_price))} | DPH: {formatCZK(
								toHalere((item.quantity * item.unit_price * item.vat_rate_percent) / 100)
							)} | Celkem: {formatCZK(
								toHalere(item.quantity * item.unit_price * (1 + item.vat_rate_percent / 100))
							)}
						</div>
					</div>
				{/each}
			</div>

			<!-- Totals -->
			<div class="mt-6 border-t border-gray-200 pt-4">
				<div class="flex flex-col items-end gap-1 text-sm">
					<div class="flex gap-8">
						<span class="text-gray-600">Základ:</span>
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
			<h2 class="text-lg font-semibold text-gray-900">Poznámky</h2>
			<div class="mt-4 space-y-4">
				<div>
					<label for="notes" class="block text-sm font-medium text-gray-700"
						>Poznámka na faktuře</label
					>
					<textarea
						id="notes"
						bind:value={form.notes}
						rows="2"
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					></textarea>
				</div>
				<div>
					<label for="internal_notes" class="block text-sm font-medium text-gray-700"
						>Interní poznámka</label
					>
					<textarea
						id="internal_notes"
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
				{saving ? 'Ukládám...' : 'Uložit jako koncept'}
			</button>
			<a
				href="/invoices"
				class="rounded-lg border border-gray-300 px-6 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
			>
				Zrušit
			</a>
		</div>
	</form>
</div>
