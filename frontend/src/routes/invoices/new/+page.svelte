<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { contactsApi, invoicesApi, type Contact, type InvoiceItem } from '$lib/api/client';
	import { toHalere } from '$lib/utils/money';
	import { toISODate, addDays } from '$lib/utils/date';
	import DateInput from '$lib/components/DateInput.svelte';
	import InvoiceItemsEditor, {
		type FormItem,
		calcSubtotal,
		calcVatTotal,
		calcGrandTotal
	} from '$lib/components/InvoiceItemsEditor.svelte';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';
	import Textarea from '$lib/ui/Textarea.svelte';
	import { toastSuccess } from '$lib/data/toast-state.svelte';

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
	let invoiceType = $state<'regular' | 'proforma'>('regular');

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
	let subtotal = $derived(calcSubtotal(items));
	let vatTotal = $derived(calcVatTotal(items));
	let grandTotal = $derived(calcGrandTotal(items));

	onMount(() => {
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
				type: invoiceType,
				status: 'draft',
				subtotal_amount: toHalere(subtotal),
				vat_amount: toHalere(vatTotal),
				total_amount: toHalere(grandTotal),
				items: invoiceItems as InvoiceItem[]
			});

			toastSuccess('Faktura vytvořena');
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

<div class="mx-auto max-w-5xl">
	<PageHeader title="Nová faktura" backHref="/invoices" backLabel="Zpět na faktury" />

	<ErrorAlert {error} class="mt-4" />

	<form
		onsubmit={(e) => {
			e.preventDefault();
			handleSubmit();
		}}
		class="mt-6 space-y-8"
	>
		<!-- Invoice Type -->
		<Card>
			<h2 class="text-base font-semibold text-primary">
				Typ dokladu <HelpTip topic="typ-faktury" />
			</h2>
			<div class="mt-4 flex gap-4">
				<label class="flex items-center gap-2 cursor-pointer">
					<input type="radio" bind:group={invoiceType} value="regular" class="accent-accent" />
					<span class="text-sm text-primary">Faktura</span>
				</label>
				<label class="flex items-center gap-2 cursor-pointer">
					<input type="radio" bind:group={invoiceType} value="proforma" class="accent-accent" />
					<span class="text-sm text-primary">Zálohová faktura</span>
				</label>
			</div>
		</Card>

		<!-- Customer -->
		<Card>
			<h2 class="text-base font-semibold text-primary">Zákazník</h2>
			<div class="mt-4">
				<label for="customer" class="block text-sm font-medium text-secondary"
					>Vyberte zákazníka</label
				>
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
		</Card>

		<!-- Dates & Symbols -->
		<Card>
			<h2 class="text-base font-semibold text-primary">Údaje faktury</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
				<div>
					<label for="issue_date" class="block text-sm font-medium text-secondary"
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
					<label for="due_date" class="block text-sm font-medium text-secondary"
						>Datum splatnosti <HelpTip topic="datum-splatnosti" /></label
					>
					<DateInput
						id="due_date"
						bind:value={form.due_date}
						required
						onchange={handleDueDateChange}
						presets={[
							{ label: '+7 dní', days: 7 },
							{ label: '+14 dní', days: 14 },
							{ label: '+30 dní', days: 30 },
							{ label: '+60 dní', days: 60 }
						]}
						relativeToValue={form.issue_date}
					/>
				</div>
				<div>
					<label for="delivery_date" class="block text-sm font-medium text-secondary"
						>DUZP <HelpTip topic="duzp" /></label
					>
					<DateInput id="delivery_date" bind:value={form.delivery_date} />
				</div>
			</div>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label for="vs" class="block text-sm font-medium text-secondary"
						>Variabilní symbol <HelpTip topic="variabilni-symbol" /></label
					>
					<input
						id="vs"
						type="text"
						bind:value={form.variable_symbol}
						class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
					/>
				</div>
				<div>
					<label for="payment" class="block text-sm font-medium text-secondary"
						>Způsob platby <HelpTip topic="zpusob-platby" /></label
					>
					<select
						id="payment"
						bind:value={form.payment_method}
						class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
					>
						<option value="bank_transfer">Bankovní převod</option>
						<option value="cash">Hotovost</option>
						<option value="card">Karta</option>
					</select>
				</div>
			</div>
		</Card>

		<!-- Line Items -->
		<InvoiceItemsEditor bind:items />

		<!-- Notes -->
		<Card>
			<h2 class="text-base font-semibold text-primary">Poznámky</h2>
			<div class="mt-4 space-y-4">
				<div>
					<label for="notes" class="block text-sm font-medium text-secondary"
						>Poznámka na faktuře <HelpTip topic="poznamka-faktura" /></label
					>
					<Textarea id="notes" bind:value={form.notes} rows={2} class="mt-1" />
				</div>
				<div>
					<label for="internal_notes" class="block text-sm font-medium text-secondary"
						>Interní poznámka <HelpTip topic="poznamka-interni" /></label
					>
					<Textarea id="internal_notes" bind:value={form.internal_notes} rows={2} class="mt-1" />
				</div>
			</div>
		</Card>

		<!-- Actions -->
		<FormActions {saving} saveLabel="Uložit jako koncept" cancelHref="/invoices" />
	</form>
</div>
