<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { expensesApi, contactsApi, type Expense, type Contact } from '$lib/api/client';
	import { formatCZK, toHalere, fromHalere } from '$lib/utils/money';
	import { formatDate } from '$lib/utils/date';
	import CategoryPicker from '$lib/components/CategoryPicker.svelte';

	let expense = $state<Expense | null>(null);
	let contacts = $state<Contact[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state<string | null>(null);
	let editing = $state(false);

	let expenseId = $derived(Number(page.params.id));

	let form = $state({
		vendor_id: null as number | null,
		expense_number: '',
		category: '',
		description: '',
		issue_date: '',
		amount: 0,
		currency_code: 'CZK',
		vat_rate_percent: 0,
		is_tax_deductible: true,
		business_percent: 100,
		payment_method: 'bank_transfer',
		notes: ''
	});

	let vatAmount = $derived(
		form.amount * form.vat_rate_percent / (100 + form.vat_rate_percent)
	);

	$effect(() => {
		loadExpense();
	});

	async function loadExpense() {
		loading = true;
		error = null;
		try {
			expense = await expensesApi.getById(expenseId);
			populateForm();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst náklad';
		} finally {
			loading = false;
		}
	}

	function populateForm() {
		if (!expense) return;
		form = {
			vendor_id: expense.vendor_id ?? null,
			expense_number: expense.expense_number,
			category: expense.category,
			description: expense.description,
			issue_date: expense.issue_date,
			amount: fromHalere(expense.amount),
			currency_code: expense.currency_code,
			vat_rate_percent: expense.vat_rate_percent,
			is_tax_deductible: expense.is_tax_deductible,
			business_percent: expense.business_percent,
			payment_method: expense.payment_method,
			notes: expense.notes
		};
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
			await expensesApi.update(expenseId, {
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
			editing = false;
			await loadExpense();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se uložit náklad';
		} finally {
			saving = false;
		}
	}

	async function handleDelete() {
		if (!confirm('Opravdu chcete smazat tento náklad?')) return;
		error = null;
		try {
			await expensesApi.delete(expenseId);
			goto('/expenses');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se smazat náklad';
		}
	}
</script>

<svelte:head>
	<title>{expense ? `Náklad - ${expense.description}` : 'Náklad'} - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-3xl">
	<a href="/expenses" class="text-sm text-blue-600 hover:text-blue-800">&larr; Zpět na náklady</a>

	{#if error}
		<div class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
			{error}
		</div>
	{/if}

	{#if loading}
		<div class="mt-8 flex items-center justify-center">
			<div class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"></div>
		</div>
	{:else if expense}
		<!-- Header -->
		<div class="mt-4 flex items-start justify-between">
			<div>
				<h1 class="text-2xl font-bold text-gray-900">{expense.description}</h1>
				{#if expense.expense_number}
					<p class="mt-1 text-sm text-gray-500">Doklad: {expense.expense_number}</p>
				{/if}
			</div>
			<div class="flex gap-2">
				{#if !editing}
					<button onclick={startEditing} class="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
						Upravit
					</button>
					<button onclick={handleDelete} class="rounded-lg border border-red-300 px-3 py-2 text-sm font-medium text-red-600 hover:bg-red-50 transition-colors">
						Smazat
					</button>
				{/if}
			</div>
		</div>

		{#if editing}
			<!-- Edit mode -->
			<form onsubmit={(e) => { e.preventDefault(); handleSave(); }} class="mt-6 space-y-6">
				<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
					<h2 class="text-lg font-semibold text-gray-900">Základní údaje</h2>
					<div class="mt-4 space-y-4">
						<div>
							<label for="edit-desc" class="block text-sm font-medium text-gray-700">Popis *</label>
							<input id="edit-desc" type="text" bind:value={form.description} required class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none" />
						</div>
						<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
							<div>
								<label for="edit-cat" class="block text-sm font-medium text-gray-700">Kategorie</label>
								<CategoryPicker
									id="edit-cat"
									value={form.category}
									onchange={(v) => { form.category = v; }}
								/>
							</div>
							<div>
								<label for="edit-num" class="block text-sm font-medium text-gray-700">Číslo dokladu</label>
								<input id="edit-num" type="text" bind:value={form.expense_number} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none" />
							</div>
						</div>
						<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
							<div>
								<label for="edit-date" class="block text-sm font-medium text-gray-700">Datum</label>
								<input id="edit-date" type="date" bind:value={form.issue_date} required class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none" />
							</div>
							<div>
								<label for="edit-vendor" class="block text-sm font-medium text-gray-700">Dodavatel</label>
								<select id="edit-vendor" bind:value={form.vendor_id} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none">
									<option value={null}>-- Bez dodavatele --</option>
									{#each contacts as contact}
										<option value={contact.id}>{contact.name}</option>
									{/each}
								</select>
							</div>
						</div>
					</div>
				</div>

				<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
					<h2 class="text-lg font-semibold text-gray-900">Částka a DPH</h2>
					<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<label for="edit-amount" class="block text-sm font-medium text-gray-700">Částka s DPH (CZK)</label>
							<input id="edit-amount" type="number" step="0.01" min="0" bind:value={form.amount} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none" />
						</div>
						<div>
							<label for="edit-vat" class="block text-sm font-medium text-gray-700">Sazba DPH</label>
							<select id="edit-vat" bind:value={form.vat_rate_percent} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none">
								<option value={21}>21%</option>
								<option value={12}>12%</option>
								<option value={0}>0%</option>
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

				<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
					<h2 class="text-lg font-semibold text-gray-900">Daňové nastavení</h2>
					<div class="mt-4 space-y-4">
						<div class="flex items-center gap-3">
							<input id="edit-deductible" type="checkbox" bind:checked={form.is_tax_deductible} class="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500" />
							<label for="edit-deductible" class="text-sm font-medium text-gray-700">Daňově uznatelný náklad</label>
						</div>
						<div>
							<label for="edit-biz" class="block text-sm font-medium text-gray-700">Podíl pro podnikání (%)</label>
							<input id="edit-biz" type="number" min="0" max="100" bind:value={form.business_percent} class="mt-1 w-32 rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none" />
						</div>
						<div>
							<label for="edit-pm" class="block text-sm font-medium text-gray-700">Způsob platby</label>
							<select id="edit-pm" bind:value={form.payment_method} class="mt-1 w-full max-w-xs rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none">
								<option value="bank_transfer">Bankovní převod</option>
								<option value="cash">Hotovost</option>
								<option value="card">Karta</option>
							</select>
						</div>
					</div>
				</div>

				<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
					<h2 class="text-lg font-semibold text-gray-900">Poznámky</h2>
					<div class="mt-4">
						<textarea bind:value={form.notes} rows="3" class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"></textarea>
					</div>
				</div>

				<div class="flex gap-3">
					<button type="submit" disabled={saving} class="rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50 transition-colors">
						{saving ? 'Ukládám...' : 'Uložit změny'}
					</button>
					<button type="button" onclick={cancelEditing} class="rounded-lg border border-gray-300 px-6 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
						Zrušit
					</button>
				</div>
			</form>
		{:else}
			<!-- View mode -->
			<div class="mt-6 space-y-6">
				<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
					<h2 class="text-lg font-semibold text-gray-900">Základní údaje</h2>
					<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div>
							<dt class="text-sm font-medium text-gray-500">Kategorie</dt>
							<dd class="mt-1 text-sm text-gray-900">{expense.category || '-'}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-gray-500">Datum</dt>
							<dd class="mt-1 text-sm text-gray-900">{formatDate(expense.issue_date)}</dd>
						</div>
						{#if expense.expense_number}
							<div>
								<dt class="text-sm font-medium text-gray-500">Číslo dokladu</dt>
								<dd class="mt-1 text-sm text-gray-900">{expense.expense_number}</dd>
							</div>
						{/if}
						<div>
							<dt class="text-sm font-medium text-gray-500">Způsob platby</dt>
							<dd class="mt-1 text-sm text-gray-900">
								{#if expense.payment_method === 'bank_transfer'}Bankovní převod
								{:else if expense.payment_method === 'cash'}Hotovost
								{:else if expense.payment_method === 'card'}Karta
								{:else}{expense.payment_method}
								{/if}
							</dd>
						</div>
					</dl>
				</div>

				<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
					<h2 class="text-lg font-semibold text-gray-900">Částka</h2>
					<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<dt class="text-sm font-medium text-gray-500">Částka s DPH</dt>
							<dd class="mt-1 text-lg font-bold text-gray-900">{formatCZK(expense.amount)}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-gray-500">DPH ({expense.vat_rate_percent}%)</dt>
							<dd class="mt-1 text-sm text-gray-900">{formatCZK(expense.vat_amount)}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-gray-500">Základ</dt>
							<dd class="mt-1 text-sm text-gray-900">{formatCZK(expense.amount - expense.vat_amount)}</dd>
						</div>
					</dl>
				</div>

				<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
					<h2 class="text-lg font-semibold text-gray-900">Daňové údaje</h2>
					<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<dt class="text-sm font-medium text-gray-500">Daňově uznatelný</dt>
							<dd class="mt-1 text-sm text-gray-900">{expense.is_tax_deductible ? 'Ano' : 'Ne'}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-gray-500">Podíl pro podnikání</dt>
							<dd class="mt-1 text-sm text-gray-900">{expense.business_percent}%</dd>
						</div>
					</dl>
				</div>

				{#if expense.notes}
					<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
						<h2 class="text-lg font-semibold text-gray-900">Poznámky</h2>
						<p class="mt-2 text-sm text-gray-900 whitespace-pre-wrap">{expense.notes}</p>
					</div>
				{/if}

				<div class="text-xs text-gray-400">
					Vytvořeno: {formatDate(expense.created_at)} | Upraveno: {formatDate(expense.updated_at)}
				</div>
			</div>
		{/if}
	{/if}
</div>
