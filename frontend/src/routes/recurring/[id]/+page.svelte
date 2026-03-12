<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		recurringInvoicesApi,
		contactsApi,
		type RecurringInvoice,
		type Contact
	} from '$lib/api/client';
	import { formatDate } from '$lib/utils/date';
	import { formatCZK, toHalere, fromHalere } from '$lib/utils/money';
	import { frequencyLabels, paymentMethodLabels } from '$lib/utils/invoice';
	import DateInput from '$lib/components/DateInput.svelte';
	import InvoiceItemsEditor, { type FormItem } from '$lib/components/InvoiceItemsEditor.svelte';
	import Badge from '$lib/ui/Badge.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import Textarea from '$lib/ui/Textarea.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';
	import { toastSuccess } from '$lib/data/toast-state.svelte';

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
		items = (recurringInvoice.items ?? []).map((item) => ({
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
			const [ri, contactsRes] = await Promise.all([
				recurringInvoicesApi.getById(id),
				contactsApi.list({ limit: 1000 })
			]);

			recurringInvoice = ri;
			contacts = contactsRes.data;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst data';
		} finally {
			loading = false;
		}
	}

	async function handleSave() {
		if (!form.name.trim()) {
			error = 'Zadejte název';
			return;
		}
		if (!form.customer_id) {
			error = 'Vyberte zákazníka';
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

			recurringInvoice = await recurringInvoicesApi.update(id, body as Partial<RecurringInvoice>);
			toastSuccess('Opakující se faktura uložena');
			editing = false;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se uložit změny';
		} finally {
			saving = false;
		}
	}

	async function generateInvoice() {
		generating = true;
		error = null;
		try {
			const invoice = await recurringInvoicesApi.generate(id);
			toastSuccess('Faktura vygenerována');
			goto(`/invoices/${invoice.id}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se vygenerovat fakturu';
		} finally {
			generating = false;
		}
	}

	onMount(() => {
		loadData();
	});
</script>

<svelte:head>
	<title>{recurringInvoice?.name ?? 'Detail'} - Opakující se faktura - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<PageHeader title={recurringInvoice?.name ?? 'Detail'} backHref="/recurring" backLabel="Zpět na opakující se faktury" />

	{#if loading}
		<LoadingSpinner class="mt-8 p-12" />
	{:else if error && !recurringInvoice}
		<ErrorAlert {error} class="mt-4" />
	{:else if recurringInvoice && !editing}
		<!-- Detail view -->
		<div class="mt-3 flex flex-wrap gap-2">
			<Button variant="secondary" onclick={generateInvoice} disabled={generating}>
				{generating ? 'Generuji...' : 'Vygenerovat fakturu'}
			</Button>
			<Button variant="primary" onclick={startEdit}>
				Upravit
			</Button>
		</div>

		<ErrorAlert {error} class="mt-4" />

		<div class="mt-6 space-y-6">
			<Card>
				<h2 class="text-base font-semibold text-primary">Základní údaje</h2>
				<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div>
						<dt class="text-sm font-medium text-tertiary">Zákazník</dt>
						<dd class="mt-1 text-sm text-primary">
							{recurringInvoice.customer?.name ?? '-'}
						</dd>
					</div>
					<div>
						<dt class="text-sm font-medium text-tertiary">Stav</dt>
						<dd class="mt-1">
							<Badge variant={recurringInvoice.is_active ? 'success' : 'muted'}>
								{recurringInvoice.is_active ? 'Aktivní' : 'Neaktivní'}
							</Badge>
						</dd>
					</div>
					<div>
						<dt class="text-sm font-medium text-tertiary">Frekvence</dt>
						<dd class="mt-1 text-sm text-primary">
							{frequencyLabels[recurringInvoice.frequency] ?? recurringInvoice.frequency}
						</dd>
					</div>
					<div>
						<dt class="text-sm font-medium text-tertiary">Další vystavení</dt>
						<dd class="mt-1 text-sm text-primary">
							{formatDate(recurringInvoice.next_issue_date)}
						</dd>
					</div>
					{#if recurringInvoice.end_date}
						<div>
							<dt class="text-sm font-medium text-tertiary">Konec opakování</dt>
							<dd class="mt-1 text-sm text-primary">
								{formatDate(recurringInvoice.end_date)}
							</dd>
						</div>
					{/if}
					<div>
						<dt class="text-sm font-medium text-tertiary">Způsob platby</dt>
						<dd class="mt-1 text-sm text-primary">
							{paymentMethodLabels[recurringInvoice.payment_method] ?? recurringInvoice.payment_method}
						</dd>
					</div>
				</dl>
				{#if recurringInvoice.notes}
					<div class="mt-4">
						<dt class="text-sm font-medium text-tertiary">Poznámka</dt>
						<dd class="mt-1 text-sm text-primary">{recurringInvoice.notes}</dd>
					</div>
				{/if}
			</Card>

			<!-- Items -->
			<Card>
				<h2 class="text-base font-semibold text-primary">Položky</h2>
				{#if recurringInvoice.items && recurringInvoice.items.length > 0}
					<table class="mt-4 w-full text-left text-sm">
						<thead class="border-b border-border bg-elevated">
							<tr>
								<th class="th-default">Popis</th>
								<th class="th-default text-right">Množství</th>
								<th class="th-default">Jednotka</th>
								<th class="th-default text-right">Cena/ks</th>
								<th class="th-default text-right">DPH</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border-subtle">
							{#each recurringInvoice.items as item (item.id)}
								<tr>
									<td class="px-4 py-2.5 text-primary">{item.description}</td>
									<td class="px-4 py-2.5 text-right font-mono tabular-nums text-secondary"
										>{(item.quantity / 100).toFixed(2)}</td
									>
									<td class="px-4 py-2.5 text-secondary">{item.unit}</td>
									<td class="px-4 py-2.5 text-right font-mono tabular-nums text-secondary">{formatCZK(item.unit_price)}</td>
									<td class="px-4 py-2.5 text-right font-mono tabular-nums text-secondary">{item.vat_rate_percent}%</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{:else}
					<p class="mt-4 text-sm text-muted">Žádné položky</p>
				{/if}
			</Card>
		</div>
	{:else if editing}
		<!-- Edit form -->
		<h1 class="mt-2 text-xl font-semibold text-primary">Upravit: {recurringInvoice?.name}</h1>

		<ErrorAlert {error} class="mt-4" />

		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleSave();
			}}
			class="mt-6 space-y-8"
		>
			<Card>
				<h2 class="text-base font-semibold text-primary">Základní údaje</h2>
				<div class="mt-4 space-y-4">
					<div>
						<label for="edit-name" class="block text-sm font-medium text-secondary"
							>Název šablony</label
						>
						<input
							id="edit-name"
							type="text"
							bind:value={form.name}
							required
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						/>
					</div>
					<div>
						<label for="edit-customer" class="block text-sm font-medium text-secondary"
							>Zákazník</label
						>
						<select
							id="edit-customer"
							bind:value={form.customer_id}
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						>
							<option value={0}>-- Vyberte --</option>
							{#each contacts as contact (contact.id)}
								<option value={contact.id}
									>{contact.name} {contact.ico ? `(${contact.ico})` : ''}</option
								>
							{/each}
						</select>
					</div>
					<div class="flex items-center gap-2">
						<input
							id="edit-active"
							type="checkbox"
							bind:checked={form.is_active}
							class="rounded border-border"
						/>
						<label for="edit-active" class="text-sm font-medium text-secondary">Aktivní</label>
					</div>
				</div>
			</Card>

			<Card>
				<h2 class="text-base font-semibold text-primary">Opakování</h2>
				<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
					<div>
						<label for="edit-frequency" class="block text-sm font-medium text-secondary"
							>Frekvence</label
						>
						<select
							id="edit-frequency"
							bind:value={form.frequency}
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						>
							{#each Object.entries(frequencyLabels) as [value, label] (value)}
								<option {value}>{label}</option>
							{/each}
						</select>
					</div>
					<div>
						<label for="edit-next-date" class="block text-sm font-medium text-secondary"
							>Další vystavení</label
						>
						<DateInput id="edit-next-date" bind:value={form.next_issue_date} required />
					</div>
					<div>
						<label for="edit-end-date" class="block text-sm font-medium text-secondary"
							>Konec opakování</label
						>
						<DateInput id="edit-end-date" bind:value={form.end_date} />
					</div>
				</div>
				<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div>
						<label for="edit-payment" class="block text-sm font-medium text-secondary"
							>Způsob platby</label
						>
						<select
							id="edit-payment"
							bind:value={form.payment_method}
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						>
							{#each Object.entries(paymentMethodLabels) as [value, label] (value)}
								<option {value}>{label}</option>
							{/each}
						</select>
					</div>
				</div>
			</Card>

			<!-- Line Items -->
			<InvoiceItemsEditor bind:items />

			<!-- Notes -->
			<Card>
				<h2 class="text-base font-semibold text-primary">Poznámky</h2>
				<div class="mt-4">
					<label for="edit-notes" class="block text-sm font-medium text-secondary"
						>Poznámka na faktuře</label
					>
					<Textarea id="edit-notes" bind:value={form.notes} rows={2} />
				</div>
			</Card>

			<FormActions {saving} saveLabel="Uložit změny" savingLabel="Ukládám..." oncancel={() => { editing = false; error = null; }} />
		</form>
	{/if}
</div>
