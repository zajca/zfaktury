<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { expensesApi, contactsApi, type Contact } from '$lib/api/client';
	import { formatCZK, toHalere } from '$lib/utils/money';
	import { toISODate } from '$lib/utils/date';
	import CategoryPicker from '$lib/components/CategoryPicker.svelte';
	import DateInput from '$lib/components/DateInput.svelte';
	import InvoiceItemsEditor, {
		type FormItem
	} from '$lib/components/InvoiceItemsEditor.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';
	import Textarea from '$lib/ui/Textarea.svelte';
	import { toastSuccess, toastError } from '$lib/data/toast-state.svelte';

	let contacts = $state<Contact[]>([]);
	let saving = $state(false);
	let useItems = $state(false);

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

	let items = $state<FormItem[]>([
		{ description: '', quantity: 1, unit: 'ks', unit_price: 0, vat_rate_percent: 21 }
	]);

	let vatAmount = $derived((form.amount * form.vat_rate_percent) / (100 + form.vat_rate_percent));

	onMount(() => {
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
			toastError('Popis je povinný');
			return;
		}
		if (!useItems && form.amount <= 0) {
			toastError('Částka musí být větší než 0');
			return;
		}
		if (useItems && items.length === 0) {
			toastError('Přidejte alespoň jednu položku');
			return;
		}

		saving = true;

		try {
			const payload: Record<string, unknown> = {
				vendor_id: form.vendor_id || undefined,
				expense_number: form.expense_number,
				category: form.category,
				description: form.description,
				issue_date: form.issue_date,
				currency_code: form.currency_code,
				exchange_rate: 100,
				is_tax_deductible: form.is_tax_deductible,
				business_percent: form.business_percent,
				payment_method: form.payment_method,
				notes: form.notes
			};

			if (useItems) {
				payload.items = items.map((item, i) => ({
					description: item.description,
					quantity: toHalere(item.quantity),
					unit: item.unit,
					unit_price: toHalere(item.unit_price),
					vat_rate_percent: item.vat_rate_percent,
					sort_order: i + 1
				}));
				payload.amount = 0;
				payload.vat_rate_percent = 0;
				payload.vat_amount = 0;
			} else {
				payload.amount = toHalere(form.amount);
				payload.vat_rate_percent = form.vat_rate_percent;
				payload.vat_amount = toHalere(vatAmount);
			}

			await expensesApi.create(payload);
			toastSuccess('Náklad vytvořen');
			goto('/expenses');
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se uložit náklad');
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Nový náklad - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<PageHeader title="Nový náklad" backHref="/expenses" backLabel="Zpět na náklady" />

	<form
		onsubmit={(e) => {
			e.preventDefault();
			handleSubmit();
		}}
		class="mt-6 space-y-6"
	>
		<!-- Basic info -->
		<Card>
			<h2 class="text-base font-semibold text-primary">Základní údaje</h2>
			<div class="mt-4 space-y-4">
				<div>
					<label for="description" class="block text-sm font-medium text-secondary">Popis *</label>
					<input
						id="description"
						type="text"
						bind:value={form.description}
						required
						class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
					/>
				</div>
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div>
						<label for="category" class="block text-sm font-medium text-secondary">Kategorie</label>
						<CategoryPicker
							id="category"
							value={form.category}
							onchange={(v) => {
								form.category = v;
							}}
						/>
					</div>
					<div>
						<label for="expense_number" class="block text-sm font-medium text-secondary"
							>Číslo dokladu <HelpTip topic="cislo-dokladu" /></label
						>
						<input
							id="expense_number"
							type="text"
							bind:value={form.expense_number}
							class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						/>
					</div>
				</div>
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div>
						<label for="issue_date" class="block text-sm font-medium text-secondary">Datum *</label>
						<DateInput id="issue_date" bind:value={form.issue_date} required />
					</div>
					<div>
						<label for="vendor" class="block text-sm font-medium text-secondary">Dodavatel</label>
						<select
							id="vendor"
							bind:value={form.vendor_id}
							class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						>
							<option value={null}>-- Bez dodavatele --</option>
							{#each contacts as contact (contact.id)}
								<option value={contact.id}
									>{contact.name} {contact.ico ? `(${contact.ico})` : ''}</option
								>
							{/each}
						</select>
					</div>
				</div>
			</div>
		</Card>

		<!-- Toggle: flat amount vs items -->
		<Card>
			<div class="flex items-center gap-3">
				<input
					id="use_items"
					type="checkbox"
					bind:checked={useItems}
					class="h-4 w-4 rounded border-border accent-accent"
				/>
				<label for="use_items" class="text-sm font-medium text-secondary"
					>Zadat jednotlivé položky</label
				>
			</div>
		</Card>

		{#if useItems}
			<!-- Items editor -->
			<InvoiceItemsEditor bind:items idPrefix="new-exp-" />
		{:else}
			<!-- Amount & VAT (flat) -->
			<Card>
				<h2 class="text-base font-semibold text-primary">Částka a DPH</h2>
				<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
					<div>
						<label for="amount" class="block text-sm font-medium text-secondary"
							>Částka s DPH (CZK) *</label
						>
						<input
							id="amount"
							type="number"
							step="0.01"
							min="0"
							bind:value={form.amount}
							required
							class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary font-mono tabular-nums focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						/>
					</div>
					<div>
						<label for="vat_rate" class="block text-sm font-medium text-secondary"
							>Sazba DPH <HelpTip topic="sazba-dph" /></label
						>
						<select
							id="vat_rate"
							bind:value={form.vat_rate_percent}
							class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						>
							<option value={21}>21%</option>
							<option value={12}>12%</option>
							<option value={0}>0% (bez DPH)</option>
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
		{/if}

		<!-- Tax settings -->
		<Card>
			<h2 class="text-base font-semibold text-primary">Daňové nastavení</h2>
			<div class="mt-4 space-y-4">
				<div class="flex items-center gap-3">
					<input
						id="tax_deductible"
						type="checkbox"
						bind:checked={form.is_tax_deductible}
						class="h-4 w-4 rounded border-border accent-accent"
					/>
					<label for="tax_deductible" class="text-sm font-medium text-secondary"
						>Daňově uznatelný náklad <HelpTip topic="danove-uznatelny" /></label
					>
				</div>
				<div>
					<label for="business_percent" class="block text-sm font-medium text-secondary"
						>Podíl pro podnikání (%) <HelpTip topic="podil-podnikani" /></label
					>
					<input
						id="business_percent"
						type="number"
						min="0"
						max="100"
						bind:value={form.business_percent}
						class="mt-1 w-32 rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary font-mono tabular-nums focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
					/>
				</div>
				<div>
					<label for="payment_method" class="block text-sm font-medium text-secondary"
						>Způsob platby</label
					>
					<select
						id="payment_method"
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

		<!-- Notes -->
		<Card>
			<h2 class="text-base font-semibold text-primary">Poznámky</h2>
			<div class="mt-4">
				<Textarea bind:value={form.notes} rows={3} placeholder="Volitelné poznámky..." />
			</div>
		</Card>

		<!-- Actions -->
		<FormActions {saving} saveLabel="Uložit náklad" cancelHref="/expenses" />
	</form>
</div>
