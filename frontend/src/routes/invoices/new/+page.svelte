<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		contactsApi,
		invoicesApi,
		sequencesApi,
		type Contact,
		type InvoiceItem,
		type InvoiceSequence
	} from '$lib/api/client';
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
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';
	import Textarea from '$lib/ui/Textarea.svelte';
	import { toastSuccess, toastError } from '$lib/data/toast-state.svelte';

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
	let sequences = $state<InvoiceSequence[]>([]);
	let saving = $state(false);
	let invoiceType = $state<'regular' | 'proforma'>('regular');

	let form = $state({
		customer_id: 0,
		sequence_id: 0,
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

	// Match invoice type to the conventional prefix used by the legacy
	// auto-assignment: regular -> FV, proforma -> ZF, credit_note -> DN. Used as
	// a hint for the default sequence pick — never enforced.
	const typeToPrefixHint: Record<string, string> = {
		regular: 'FV',
		proforma: 'ZF',
		credit_note: 'DN'
	};

	function pickDefaultSequence(): number {
		if (sequences.length === 0) return 0;
		const year = Number(form.issue_date.slice(0, 4));
		const yearShort = year % 100;
		const prefixHint = typeToPrefixHint[invoiceType];

		// 1. Same year (full or short YY) + matching prefix hint.
		const exact = sequences.find(
			(s) =>
				(s.year === year || s.year === yearShort) && s.prefix.toUpperCase() === prefixHint
		);
		if (exact) return exact.id;
		// 2. Same year, any prefix.
		const sameYear = sequences.find((s) => s.year === year || s.year === yearShort);
		if (sameYear) return sameYear.id;
		// 3. Otherwise pick whichever (first) — user can change.
		return sequences[0].id;
	}

	// Re-derive the default when the year changes or the type flips, but only
	// if the user hasn't manually overridden it. We track manual override by
	// keeping the previous auto pick and comparing.
	let lastAutoPick = $state(0);
	$effect(() => {
		// Re-run when issue_date, invoiceType, or sequences change.
		void form.issue_date;
		void invoiceType;
		void sequences;
		if (form.sequence_id !== 0 && form.sequence_id !== lastAutoPick) {
			// User has manually selected — leave it alone.
			return;
		}
		const next = pickDefaultSequence();
		form.sequence_id = next;
		lastAutoPick = next;
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
		loadSequences();
	});

	async function loadContacts() {
		try {
			const res = await contactsApi.list({ limit: 1000 });
			contacts = res.data;
		} catch {
			// Contacts loading is non-critical
		}
	}

	async function loadSequences() {
		try {
			const result = await sequencesApi.list();
			sequences = Array.isArray(result) ? result : [];
		} catch {
			sequences = [];
		}
	}

	async function handleSubmit() {
		if (!form.customer_id) {
			toastError('Vyberte zákazníka');
			return;
		}
		if (sequences.length > 0 && !form.sequence_id) {
			toastError('Vyberte číselnou řadu');
			return;
		}

		saving = true;

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
			toastError(e instanceof Error ? e.message : 'Nepodařilo se vytvořit fakturu');
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

			<div class="mt-6">
				<label for="sequence" class="block text-sm font-medium text-secondary">
					Číselná řada <HelpTip topic="ciselne-rady" />
				</label>
				{#if sequences.length === 0}
					<p
						class="mt-1 rounded-lg border border-warning/40 bg-warning-bg px-3 py-2 text-sm text-warning"
						role="alert"
					>
						Žádná číselná řada není vytvořená pro tuto firmu. <a
							href="/settings/sequences"
							class="font-medium underline">Vytvořit první řadu</a
						> před uložením faktury.
					</p>
				{:else}
					<select
						id="sequence"
						bind:value={form.sequence_id}
						class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
					>
						{#each sequences as seq (seq.id)}
							<option value={seq.id}
								>{seq.prefix} / {seq.year} &mdash; další: {seq.preview}</option
							>
						{/each}
					</select>
				{/if}
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
