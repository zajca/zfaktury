<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		recurringExpensesApi,
		contactsApi,
		type RecurringExpense,
		type Contact
	} from '$lib/api/client';
	import { formatCZK, toHalere, fromHalere } from '$lib/utils/money';
	import { formatDate } from '$lib/utils/date';
	import { paymentMethodLabels, frequencyLabels } from '$lib/utils/invoice';
	import DateInput from '$lib/components/DateInput.svelte';
	import CategoryPicker from '$lib/components/CategoryPicker.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import Badge from '$lib/ui/Badge.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';
	import Textarea from '$lib/ui/Textarea.svelte';

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

	let vatAmount = $derived((form.amount * form.vat_rate_percent) / (100 + form.vat_rate_percent));

	onMount(() => {
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

</script>

<svelte:head>
	<title>{item ? `${item.name} - Opakovaný náklad` : 'Opakovaný náklad'} - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<a href="/expenses/recurring" class="text-sm text-secondary hover:text-primary"
		>&larr; Zpět na opakované náklady</a
	>

	<ErrorAlert {error} class="mt-4" />

	{#if loading}
		<LoadingSpinner class="mt-8" />
	{:else if item}
		<!-- Header -->
		<div class="mt-4">
			<div class="flex items-center justify-between">
				<h1 class="text-xl font-semibold text-primary">{item.name}</h1>
				<div class="flex items-center gap-2">
					{#if item.is_active}
						<Badge variant="success">Aktivní</Badge>
					{:else}
						<Badge variant="muted">Neaktivní</Badge>
					{/if}
					<span class="text-sm text-tertiary">{frequencyLabels[item.frequency] ?? item.frequency}</span>
				</div>
			</div>
			{#if !editing}
				<div class="mt-3 flex flex-wrap gap-2">
					<Button variant="secondary" onclick={handleToggleActive}>
						{item.is_active ? 'Deaktivovat' : 'Aktivovat'}
					</Button>
					<Button variant="secondary" onclick={startEditing}>
						Upravit
					</Button>
					<Button variant="danger" onclick={handleDelete}>
						Smazat
					</Button>
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
				<Card>
					<h2 class="text-base font-semibold text-primary">Základní údaje</h2>
					<div class="mt-4 space-y-4">
						<div>
							<label for="edit-name" class="block text-sm font-medium text-secondary">Název *</label>
							<input
								id="edit-name"
								type="text"
								bind:value={form.name}
								required
								class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div>
							<label for="edit-desc" class="block text-sm font-medium text-secondary">Popis *</label>
							<input
								id="edit-desc"
								type="text"
								bind:value={form.description}
								required
								class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
							<div>
								<label for="edit-cat" class="block text-sm font-medium text-secondary"
									>Kategorie</label
								>
								<CategoryPicker
									id="edit-cat"
									value={form.category}
									onchange={(v) => {
										form.category = v;
									}}
								/>
							</div>
							<div>
								<label for="edit-vendor" class="block text-sm font-medium text-secondary"
									>Dodavatel</label
								>
								<select
									id="edit-vendor"
									bind:value={form.vendor_id}
									class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
								>
									<option value={null}>-- Bez dodavatele --</option>
									{#each contacts as contact (contact.id)}
										<option value={contact.id}>{contact.name}</option>
									{/each}
								</select>
							</div>
						</div>
					</div>
				</Card>

				<Card>
					<h2 class="text-base font-semibold text-primary">Plánování</h2>
					<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<label for="edit-freq" class="block text-sm font-medium text-secondary"
								>Frekvence</label
							>
							<select
								id="edit-freq"
								bind:value={form.frequency}
								class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							>
								<option value="weekly">Týdně</option>
								<option value="monthly">Měsíčně</option>
								<option value="quarterly">Čtvrtletně</option>
								<option value="yearly">Ročně</option>
							</select>
						</div>
						<div>
							<label for="edit-next" class="block text-sm font-medium text-secondary"
								>Další datum</label
							>
							<DateInput id="edit-next" bind:value={form.next_issue_date} required />
						</div>
						<div>
							<label for="edit-end" class="block text-sm font-medium text-secondary"
								>Datum ukončení</label
							>
							<DateInput id="edit-end" bind:value={form.end_date} />
						</div>
					</div>
				</Card>

				<Card>
					<h2 class="text-base font-semibold text-primary">Částka a DPH</h2>
					<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<label for="edit-amount" class="block text-sm font-medium text-secondary"
								>Částka s DPH (CZK)</label
							>
							<input
								id="edit-amount"
								type="number"
								step="0.01"
								min="0"
								bind:value={form.amount}
								class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary font-mono tabular-nums focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div>
							<label for="edit-vat" class="block text-sm font-medium text-secondary">Sazba DPH</label
							>
							<select
								id="edit-vat"
								bind:value={form.vat_rate_percent}
								class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							>
								<option value={21}>21%</option>
								<option value={12}>12%</option>
								<option value={0}>0%</option>
							</select>
						</div>
						<div>
							<span class="block text-sm font-medium text-secondary">DPH</span>
							<div
								class="mt-1 bg-elevated border-border text-secondary rounded-lg px-3 py-2 text-sm font-mono tabular-nums"
							>
								{formatCZK(toHalere(vatAmount))}
							</div>
						</div>
					</div>
				</Card>

				<Card>
					<h2 class="text-base font-semibold text-primary">Daňové nastavení</h2>
					<div class="mt-4 space-y-4">
						<div class="flex items-center gap-3">
							<input
								id="edit-deductible"
								type="checkbox"
								bind:checked={form.is_tax_deductible}
								class="h-4 w-4 rounded border-border accent-accent"
							/>
							<label for="edit-deductible" class="text-sm font-medium text-secondary"
								>Daňově uznatelný náklad</label
							>
						</div>
						<div>
							<label for="edit-biz" class="block text-sm font-medium text-secondary"
								>Podíl pro podnikání (%)</label
							>
							<input
								id="edit-biz"
								type="number"
								min="0"
								max="100"
								bind:value={form.business_percent}
								class="mt-1 w-32 rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary font-mono tabular-nums focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div>
							<label for="edit-pm" class="block text-sm font-medium text-secondary"
								>Způsob platby</label
							>
							<select
								id="edit-pm"
								bind:value={form.payment_method}
								class="mt-1 w-full max-w-xs rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							>
								<option value="bank_transfer">Bankovní převod</option>
								<option value="cash">Hotovost</option>
								<option value="card">Karta</option>
							</select>
						</div>
					</div>
				</Card>

				<Card>
					<h2 class="text-base font-semibold text-primary">Poznámky</h2>
					<div class="mt-4">
						<Textarea bind:value={form.notes} rows={3} />
					</div>
				</Card>

				<FormActions {saving} saveLabel="Uložit změny" oncancel={cancelEditing} />
			</form>
		{:else}
			<!-- View mode -->
			<div class="mt-6 space-y-6">
				<Card>
					<h2 class="text-base font-semibold text-primary">Základní údaje</h2>
					<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div>
							<dt class="text-sm font-medium text-tertiary">Popis</dt>
							<dd class="mt-1 text-sm text-primary">{item.description}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-tertiary">Kategorie</dt>
							<dd class="mt-1 text-sm text-primary">{item.category || '-'}</dd>
						</div>
						{#if item.vendor}
							<div>
								<dt class="text-sm font-medium text-tertiary">Dodavatel</dt>
								<dd class="mt-1 text-sm text-primary">{item.vendor.name}</dd>
							</div>
						{/if}
						<div>
							<dt class="text-sm font-medium text-tertiary">Způsob platby</dt>
							<dd class="mt-1 text-sm text-primary">
								{paymentMethodLabels[item.payment_method] ?? item.payment_method}
							</dd>
						</div>
					</dl>
				</Card>

				<Card>
					<h2 class="text-base font-semibold text-primary">Plánování</h2>
					<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<dt class="text-sm font-medium text-tertiary">Frekvence</dt>
							<dd class="mt-1 text-sm text-primary">{frequencyLabels[item.frequency] ?? item.frequency}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-tertiary">Další datum</dt>
							<dd class="mt-1 text-sm text-primary">{formatDate(item.next_issue_date)}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-tertiary">Datum ukončení</dt>
							<dd class="mt-1 text-sm text-primary">
								{item.end_date ? formatDate(item.end_date) : 'Neomezeno'}
							</dd>
						</div>
					</dl>
				</Card>

				<Card>
					<h2 class="text-base font-semibold text-primary">Částka</h2>
					<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<dt class="text-sm font-medium text-tertiary">Částka s DPH</dt>
							<dd class="mt-1 text-lg font-semibold text-primary font-mono tabular-nums">{formatCZK(item.amount)}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-tertiary">DPH ({item.vat_rate_percent}%)</dt>
							<dd class="mt-1 text-sm text-primary font-mono tabular-nums">{formatCZK(item.vat_amount)}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-tertiary">Základ</dt>
							<dd class="mt-1 text-sm text-primary font-mono tabular-nums">{formatCZK(item.amount - item.vat_amount)}</dd>
						</div>
					</dl>
				</Card>

				<Card>
					<h2 class="text-base font-semibold text-primary">Daňové údaje</h2>
					<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<dt class="text-sm font-medium text-tertiary">Daňově uznatelný</dt>
							<dd class="mt-1 text-sm text-primary">{item.is_tax_deductible ? 'Ano' : 'Ne'}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-tertiary">Podíl pro podnikání</dt>
							<dd class="mt-1 text-sm text-primary">{item.business_percent}%</dd>
						</div>
					</dl>
				</Card>

				{#if item.notes}
					<Card>
						<h2 class="text-base font-semibold text-primary">Poznámky</h2>
						<p class="mt-2 text-sm text-primary whitespace-pre-wrap">{item.notes}</p>
					</Card>
				{/if}

				<div class="text-xs text-muted">
					Vytvořeno: {formatDate(item.created_at)} | Upraveno: {formatDate(item.updated_at)}
				</div>
			</div>
		{/if}
	{/if}
</div>
