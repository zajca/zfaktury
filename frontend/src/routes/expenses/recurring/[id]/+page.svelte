<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { recurringExpensesApi, contactsApi, type RecurringExpense, type Contact } from '$lib/api/client';
	import { formatCZK, toHalere, fromHalere } from '$lib/utils/money';
	import { formatDate } from '$lib/utils/date';
	import CategoryPicker from '$lib/components/CategoryPicker.svelte';

	let item = $state<RecurringExpense | null>(null);
	let contacts = $state<Contact[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state<string | null>(null);
	let editing = $state(false);

	let itemId = $derived(Number(page.params.id));

	let form = $state({
		name: '',
		vendor_id: null as number | null,
		category: '',
		description: '',
		amount: 0,
		currency_code: 'CZK',
		vat_rate_percent: 0,
		is_tax_deductible: true,
		business_percent: 100,
		payment_method: 'bank_transfer',
		notes: '',
		frequency: 'monthly',
		next_issue_date: '',
		end_date: '',
		is_active: true
	});

	let vatAmount = $derived(
		form.amount * form.vat_rate_percent / (100 + form.vat_rate_percent)
	);

	$effect(() => {
		loadItem();
	});

	async function loadItem() {
		loading = true;
		error = null;
		try {
			item = await recurringExpensesApi.getById(itemId);
			populateForm();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst opakovaný náklad';
		} finally {
			loading = false;
		}
	}

	function populateForm() {
		if (!item) return;
		form = {
			name: item.name,
			vendor_id: item.vendor_id ?? null,
			category: item.category,
			description: item.description,
			amount: fromHalere(item.amount),
			currency_code: item.currency_code,
			vat_rate_percent: item.vat_rate_percent,
			is_tax_deductible: item.is_tax_deductible,
			business_percent: item.business_percent,
			payment_method: item.payment_method,
			notes: item.notes,
			frequency: item.frequency,
			next_issue_date: item.next_issue_date,
			end_date: item.end_date ?? '',
			is_active: item.is_active
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
		if (!form.name) {
			error = 'Název je povinný';
			return;
		}
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
			await recurringExpensesApi.update(itemId, {
				name: form.name,
				vendor_id: form.vendor_id || undefined,
				category: form.category,
				description: form.description,
				amount: toHalere(form.amount),
				currency_code: form.currency_code,
				exchange_rate: 100,
				vat_rate_percent: form.vat_rate_percent,
				vat_amount: toHalere(vatAmount),
				is_tax_deductible: form.is_tax_deductible,
				business_percent: form.business_percent,
				payment_method: form.payment_method,
				notes: form.notes,
				frequency: form.frequency as RecurringExpense['frequency'],
				next_issue_date: form.next_issue_date,
				end_date: form.end_date || undefined,
				is_active: form.is_active
			});
			editing = false;
			await loadItem();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se uložit opakovaný náklad';
		} finally {
			saving = false;
		}
	}

	async function handleDelete() {
		if (!confirm('Opravdu chcete smazat tento opakovaný náklad?')) return;
		error = null;
		try {
			await recurringExpensesApi.delete(itemId);
			goto('/expenses/recurring');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se smazat opakovaný náklad';
		}
	}

	async function handleToggleActive() {
		error = null;
		try {
			if (item?.is_active) {
				await recurringExpensesApi.deactivate(itemId);
			} else {
				await recurringExpensesApi.activate(itemId);
			}
			await loadItem();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se změnit stav';
		}
	}

	function frequencyLabel(freq: string): string {
		switch (freq) {
			case 'weekly': return 'Týdně';
			case 'monthly': return 'Měsíčně';
			case 'quarterly': return 'Čtvrtletně';
			case 'yearly': return 'Ročně';
			default: return freq;
		}
	}
</script>

<svelte:head>
	<title>{item ? `${item.name} - Opakovaný náklad` : 'Opakovaný náklad'} - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-3xl">
	<a href="/expenses/recurring" class="text-sm text-blue-600 hover:text-blue-800">&larr; Zpět na opakované náklady</a>

	{#if error}
		<div class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
			{error}
		</div>
	{/if}

	{#if loading}
		<div class="mt-8 flex items-center justify-center">
			<div class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"></div>
		</div>
	{:else if item}
		<!-- Header -->
		<div class="mt-4 flex items-start justify-between">
			<div>
				<h1 class="text-2xl font-bold text-gray-900">{item.name}</h1>
				<div class="mt-1 flex items-center gap-2">
					{#if item.is_active}
						<span class="inline-flex items-center rounded-full bg-green-50 px-2 py-1 text-xs font-medium text-green-700">Aktivní</span>
					{:else}
						<span class="inline-flex items-center rounded-full bg-gray-100 px-2 py-1 text-xs font-medium text-gray-600">Neaktivní</span>
					{/if}
					<span class="text-sm text-gray-500">{frequencyLabel(item.frequency)}</span>
				</div>
			</div>
			<div class="flex gap-2">
				{#if !editing}
					<button onclick={handleToggleActive} class="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors">
						{item.is_active ? 'Deaktivovat' : 'Aktivovat'}
					</button>
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
							<label for="edit-name" class="block text-sm font-medium text-gray-700">Název *</label>
							<input id="edit-name" type="text" bind:value={form.name} required class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none" />
						</div>
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
					<h2 class="text-lg font-semibold text-gray-900">Plánování</h2>
					<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<label for="edit-freq" class="block text-sm font-medium text-gray-700">Frekvence</label>
							<select id="edit-freq" bind:value={form.frequency} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none">
								<option value="weekly">Týdně</option>
								<option value="monthly">Měsíčně</option>
								<option value="quarterly">Čtvrtletně</option>
								<option value="yearly">Ročně</option>
							</select>
						</div>
						<div>
							<label for="edit-next" class="block text-sm font-medium text-gray-700">Další datum</label>
							<input id="edit-next" type="date" bind:value={form.next_issue_date} required class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none" />
						</div>
						<div>
							<label for="edit-end" class="block text-sm font-medium text-gray-700">Datum ukončení</label>
							<input id="edit-end" type="date" bind:value={form.end_date} class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none" />
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
							<dt class="text-sm font-medium text-gray-500">Popis</dt>
							<dd class="mt-1 text-sm text-gray-900">{item.description}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-gray-500">Kategorie</dt>
							<dd class="mt-1 text-sm text-gray-900">{item.category || '-'}</dd>
						</div>
						{#if item.vendor}
							<div>
								<dt class="text-sm font-medium text-gray-500">Dodavatel</dt>
								<dd class="mt-1 text-sm text-gray-900">{item.vendor.name}</dd>
							</div>
						{/if}
						<div>
							<dt class="text-sm font-medium text-gray-500">Způsob platby</dt>
							<dd class="mt-1 text-sm text-gray-900">
								{#if item.payment_method === 'bank_transfer'}Bankovní převod
								{:else if item.payment_method === 'cash'}Hotovost
								{:else if item.payment_method === 'card'}Karta
								{:else}{item.payment_method}
								{/if}
							</dd>
						</div>
					</dl>
				</div>

				<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
					<h2 class="text-lg font-semibold text-gray-900">Plánování</h2>
					<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<dt class="text-sm font-medium text-gray-500">Frekvence</dt>
							<dd class="mt-1 text-sm text-gray-900">{frequencyLabel(item.frequency)}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-gray-500">Další datum</dt>
							<dd class="mt-1 text-sm text-gray-900">{formatDate(item.next_issue_date)}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-gray-500">Datum ukončení</dt>
							<dd class="mt-1 text-sm text-gray-900">{item.end_date ? formatDate(item.end_date) : 'Neomezeno'}</dd>
						</div>
					</dl>
				</div>

				<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
					<h2 class="text-lg font-semibold text-gray-900">Částka</h2>
					<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<dt class="text-sm font-medium text-gray-500">Částka s DPH</dt>
							<dd class="mt-1 text-lg font-bold text-gray-900">{formatCZK(item.amount)}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-gray-500">DPH ({item.vat_rate_percent}%)</dt>
							<dd class="mt-1 text-sm text-gray-900">{formatCZK(item.vat_amount)}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-gray-500">Základ</dt>
							<dd class="mt-1 text-sm text-gray-900">{formatCZK(item.amount - item.vat_amount)}</dd>
						</div>
					</dl>
				</div>

				<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
					<h2 class="text-lg font-semibold text-gray-900">Daňové údaje</h2>
					<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<dt class="text-sm font-medium text-gray-500">Daňově uznatelný</dt>
							<dd class="mt-1 text-sm text-gray-900">{item.is_tax_deductible ? 'Ano' : 'Ne'}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-gray-500">Podíl pro podnikání</dt>
							<dd class="mt-1 text-sm text-gray-900">{item.business_percent}%</dd>
						</div>
					</dl>
				</div>

				{#if item.notes}
					<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
						<h2 class="text-lg font-semibold text-gray-900">Poznámky</h2>
						<p class="mt-2 text-sm text-gray-900 whitespace-pre-wrap">{item.notes}</p>
					</div>
				{/if}

				<div class="text-xs text-gray-400">
					Vytvořeno: {formatDate(item.created_at)} | Upraveno: {formatDate(item.updated_at)}
				</div>
			</div>
		{/if}
	{/if}
</div>
