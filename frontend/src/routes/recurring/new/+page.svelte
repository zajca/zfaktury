<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		contactsApi,
		recurringInvoicesApi,
		sequencesApi,
		type Contact,
		type RecurringInvoice,
		type InvoiceSequence
	} from '$lib/api/client';
	import { onCompanyChange } from '$lib/stores/currentCompany.svelte';
	import { toISODate } from '$lib/utils/date';
	import { paymentMethodLabels, frequencyLabels } from '$lib/utils/invoice';
	import DateInput from '$lib/components/DateInput.svelte';
	import { toHalere } from '$lib/utils/money';
	import InvoiceItemsEditor, { type FormItem } from '$lib/components/InvoiceItemsEditor.svelte';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import Textarea from '$lib/ui/Textarea.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';
	import { toastSuccess, toastError } from '$lib/data/toast-state.svelte';

	let contacts = $state<Contact[]>([]);
	let sequences = $state<InvoiceSequence[]>([]);
	let saving = $state(false);

	let form = $state({
		name: '',
		customer_id: 0,
		sequence_id: 0,
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
		is_active: true,
		auto_send: false,
		auto_send_recipient: ''
	});

	let items = $state<FormItem[]>([
		{ description: '', quantity: 1, unit: 'ks', unit_price: 0, vat_rate_percent: 21 }
	]);

	onMount(() => {
		loadContacts();
		loadSequences();
	});

	onCompanyChange(() => {
		loadContacts();
		loadSequences();
	});

	async function loadSequences() {
		try {
			const result = await sequencesApi.list();
			sequences = Array.isArray(result) ? result : [];
			// Default to the first sequence so generated invoices use the
			// company's own numbering instead of the auto "FV" fallback.
			if (sequences.length > 0 && !form.sequence_id) {
				form.sequence_id = sequences[0].id;
			}
		} catch {
			sequences = [];
		}
	}

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
			toastError('Zadejte název');
			return;
		}
		if (!form.customer_id) {
			toastError('Vyberte zákazníka');
			return;
		}

		saving = true;

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
				// Don't send a stale recipient when auto-send is off.
				auto_send_recipient: form.auto_send ? form.auto_send_recipient : '',
				items: requestItems
			};

			await recurringInvoicesApi.create(body as Partial<RecurringInvoice>);

			toastSuccess('Opakující se faktura vytvořena');
			goto('/recurring');
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se vytvořit opakující se fakturu');
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Nová opakující se faktura - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<PageHeader
		title="Nová opakující se faktura"
		backHref="/recurring"
		backLabel="Zpět na opakující se faktury"
	/>

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
				<div>
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
							> před uložením.
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
			</div>
		</Card>

		<!-- Schedule -->
		<Card>
			<h2 class="text-base font-semibold text-primary">Opakování</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
				<div>
					<label for="frequency" class="block text-sm font-medium text-secondary"
						>Frekvence <HelpTip topic="frekvence-opakovani" /></label
					>
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
					<label for="payment" class="block text-sm font-medium text-secondary">Způsob platby</label
					>
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

		<!-- Auto-send -->
		<Card>
			<h2 class="text-base font-semibold text-primary">Automatické odesílání</h2>
			<div class="mt-4 space-y-4">
				<div class="flex items-center gap-2">
					<input
						id="auto-send"
						type="checkbox"
						bind:checked={form.auto_send}
						class="rounded border-border"
					/>
					<label for="auto-send" class="text-sm font-medium text-secondary"
						>Automaticky poslat fakturu e-mailem</label
					>
				</div>
				{#if form.auto_send}
					<div>
						<label for="auto-send-recipient" class="block text-sm font-medium text-secondary"
							>Přepsat příjemce (volitelné)</label
						>
						<input
							id="auto-send-recipient"
							type="email"
							bind:value={form.auto_send_recipient}
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						/>
						<p class="mt-1 text-xs text-tertiary">
							Pokud necháte prázdné, použije se e-mail zákazníka.
						</p>
					</div>
				{/if}
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
