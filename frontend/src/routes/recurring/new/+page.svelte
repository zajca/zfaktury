<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		contactsApi,
		recurringInvoicesApi,
		type Contact,
		type RecurringInvoice
	} from '$lib/api/client';
	import { toISODate } from '$lib/utils/date';
	import { paymentMethodLabels, frequencyLabels } from '$lib/utils/invoice';
	import DateInput from '$lib/components/DateInput.svelte';
	import { toHalere } from '$lib/utils/money';
	import InvoiceItemsEditor, { type FormItem } from '$lib/components/InvoiceItemsEditor.svelte';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import Textarea from '$lib/ui/Textarea.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';

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

	onMount(() => {
		loadContacts();
	});

	async function loadContacts() {
		try {
			const res = await contactsApi.list({ limit: 1000 });
			contacts = res.data;
		} catch {
			// Non-critical
		}
	}

	async function handleSubmit() {
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

			await recurringInvoicesApi.create(body as Partial<RecurringInvoice>);

			goto('/recurring');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se vytvořit opakující se fakturu';
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Nová opakující se faktura - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<PageHeader title="Nová opakující se faktura" backHref="/recurring" backLabel="Zpět na opakující se faktury" />

	<ErrorAlert {error} class="mt-4" />

	<form
		onsubmit={(e) => {
			e.preventDefault();
			handleSubmit();
		}}
		class="mt-6 space-y-8"
	>
		<!-- Basic info -->
		<Card>
			<h2 class="text-base font-semibold text-primary">Základní údaje</h2>
			<div class="mt-4 space-y-4">
				<div>
					<label for="name" class="block text-sm font-medium text-secondary">Název šablony</label>
					<input
						id="name"
						type="text"
						bind:value={form.name}
						required
						class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						placeholder="např. Měsíční hosting"
					/>
				</div>
				<div>
					<label for="customer" class="block text-sm font-medium text-secondary">Zákazník</label>
					<select
						id="customer"
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
			</div>
		</Card>

		<!-- Schedule -->
		<Card>
			<h2 class="text-base font-semibold text-primary">Opakování</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
				<div>
					<label for="frequency" class="block text-sm font-medium text-secondary">Frekvence <HelpTip topic="frekvence-opakovani" /></label>
					<select
						id="frequency"
						bind:value={form.frequency}
						class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
					>
						{#each Object.entries(frequencyLabels) as [value, label] (value)}
							<option {value}>{label}</option>
						{/each}
					</select>
				</div>
				<div>
					<label for="next_issue_date" class="block text-sm font-medium text-secondary"
						>Další vystavení</label
					>
					<DateInput id="next_issue_date" bind:value={form.next_issue_date} required />
				</div>
				<div>
					<label for="end_date" class="block text-sm font-medium text-secondary"
						>Konec opakování (volitelné)</label
					>
					<DateInput id="end_date" bind:value={form.end_date} />
				</div>
			</div>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label for="payment" class="block text-sm font-medium text-secondary">Způsob platby</label>
					<select
						id="payment"
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
				<label for="notes" class="block text-sm font-medium text-secondary"
					>Poznámka na faktuře</label
				>
				<Textarea id="notes" bind:value={form.notes} rows={2} />
			</div>
		</Card>

		<!-- Actions -->
		<FormActions {saving} saveLabel="Uložit" savingLabel="Ukládám..." cancelHref="/recurring" />
	</form>
</div>
