<script lang="ts">
	import { goto } from '$app/navigation';
	import { expensesApi, contactsApi, type Contact } from '$lib/api/client';
	import { formatCZK, toHalere } from '$lib/utils/money';
	import { toISODate } from '$lib/utils/date';
	import CategoryPicker from '$lib/components/CategoryPicker.svelte';
	import DateInput from '$lib/components/DateInput.svelte';

	let contacts = $state<Contact[]>([]);
	let saving = $state(false);
	let error = $state<string | null>(null);

	let form = $state({
		vendor_id: null as number | null,
		expense_number: '',
		category: '',
		description: '',
		issue_date: toISODate(new Date()),
		amount: 0,
		currency_code: 'CZK',
		vat_rate_percent: 21,
		is_tax_deductible: true,
		business_percent: 100,
		payment_method: 'bank_transfer',
		notes: ''
	});

	let vatAmount = $derived(
		form.amount * form.vat_rate_percent / (100 + form.vat_rate_percent)
	);

	$effect(() => {
		loadContacts();
	});

	async function loadContacts() {
		try {
			const res = await contactsApi.list({ limit: 1000 });
			contacts = res.data;
		} catch {
			// non-critical
		}
	}

	async function handleSubmit() {
		if (!form.description) {
			error = 'Popis je povinný';
			return;
		}
		if (form.amount <= 0) {
			error = 'Částka musí být větší než 0';
			return;
		}

		saving = true;
		error = null;

		try {
			await expensesApi.create({
				vendor_id: form.vendor_id || undefined,
				expense_number: form.expense_number,
				category: form.category,
				description: form.description,
				issue_date: form.issue_date,
				amount: toHalere(form.amount),
				currency_code: form.currency_code,
				exchange_rate: 100,
				vat_rate_percent: form.vat_rate_percent,
				vat_amount: toHalere(vatAmount),
				is_tax_deductible: form.is_tax_deductible,
				business_percent: form.business_percent,
				payment_method: form.payment_method,
				notes: form.notes
			});
			goto('/expenses');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se uložit náklad';
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Nový náklad - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-3xl">
	<a href="/expenses" class="text-sm text-blue-600 hover:text-blue-800">&larr; Zpět na náklady</a>
	<h1 class="mt-2 text-2xl font-bold text-gray-900">Nový náklad</h1>

	{#if error}
		<div class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
			{error}
		</div>
	{/if}

	<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="mt-6 space-y-6">
		<!-- Basic info -->
		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Základní údaje</h2>
			<div class="mt-4 space-y-4">
				<div>
					<label for="description" class="block text-sm font-medium text-gray-700">Popis *</label>
					<input
						id="description"
						type="text"
						bind:value={form.description}
						required
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					/>
				</div>
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div>
						<label for="category" class="block text-sm font-medium text-gray-700">Kategorie</label>
						<CategoryPicker
							id="category"
							value={form.category}
							onchange={(v) => { form.category = v; }}
						/>
					</div>
					<div>
						<label for="expense_number" class="block text-sm font-medium text-gray-700">Číslo dokladu</label>
						<input
							id="expense_number"
							type="text"
							bind:value={form.expense_number}
							class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
						/>
					</div>
				</div>
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div>
						<label for="issue_date" class="block text-sm font-medium text-gray-700">Datum *</label>
						<DateInput id="issue_date" bind:value={form.issue_date} required />
					</div>
					<div>
						<label for="vendor" class="block text-sm font-medium text-gray-700">Dodavatel</label>
						<select
							id="vendor"
							bind:value={form.vendor_id}
							class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
						>
							<option value={null}>-- Bez dodavatele --</option>
							{#each contacts as contact}
								<option value={contact.id}>{contact.name} {contact.ico ? `(${contact.ico})` : ''}</option>
							{/each}
						</select>
					</div>
				</div>
			</div>
		</div>

		<!-- Amount & VAT -->
		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Částka a DPH</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
				<div>
					<label for="amount" class="block text-sm font-medium text-gray-700">Částka s DPH (CZK) *</label>
					<input
						id="amount"
						type="number"
						step="0.01"
						min="0"
						bind:value={form.amount}
						required
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					/>
				</div>
				<div>
					<label for="vat_rate" class="block text-sm font-medium text-gray-700">Sazba DPH</label>
					<select
						id="vat_rate"
						bind:value={form.vat_rate_percent}
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					>
						<option value={21}>21%</option>
						<option value={12}>12%</option>
						<option value={0}>0% (bez DPH)</option>
					</select>
				</div>
				<div>
					<label class="block text-sm font-medium text-gray-700">DPH</label>
					<div class="mt-1 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-700">
						{formatCZK(toHalere(vatAmount))}
					</div>
				</div>
			</div>
		</div>

		<!-- Tax settings -->
		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Daňové nastavení</h2>
			<div class="mt-4 space-y-4">
				<div class="flex items-center gap-3">
					<input
						id="tax_deductible"
						type="checkbox"
						bind:checked={form.is_tax_deductible}
						class="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
					/>
					<label for="tax_deductible" class="text-sm font-medium text-gray-700">Daňově uznatelný náklad</label>
				</div>
				<div>
					<label for="business_percent" class="block text-sm font-medium text-gray-700">Podíl pro podnikání (%)</label>
					<input
						id="business_percent"
						type="number"
						min="0"
						max="100"
						bind:value={form.business_percent}
						class="mt-1 w-32 rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					/>
				</div>
				<div>
					<label for="payment_method" class="block text-sm font-medium text-gray-700">Způsob platby</label>
					<select
						id="payment_method"
						bind:value={form.payment_method}
						class="mt-1 w-full max-w-xs rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					>
						<option value="bank_transfer">Bankovní převod</option>
						<option value="cash">Hotovost</option>
						<option value="card">Karta</option>
					</select>
				</div>
			</div>
		</div>

		<!-- Notes -->
		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Poznámky</h2>
			<div class="mt-4">
				<textarea
					bind:value={form.notes}
					rows="3"
					placeholder="Volitelné poznámky..."
					class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
				></textarea>
			</div>
		</div>

		<!-- Actions -->
		<div class="flex gap-3">
			<button
				type="submit"
				disabled={saving}
				class="rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50 transition-colors"
			>
				{saving ? 'Ukládám...' : 'Uložit náklad'}
			</button>
			<a
				href="/expenses"
				class="rounded-lg border border-gray-300 px-6 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
			>
				Zrušit
			</a>
		</div>
	</form>
</div>
