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
	import DateInput from '$lib/components/DateInput.svelte';
	import { toHalere } from '$lib/utils/money';
	import InvoiceItemsEditor, { type FormItem } from '$lib/components/InvoiceItemsEditor.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';

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
			error = 'Zadejte nazev';
			return;
		}
		if (!form.customer_id) {
			error = 'Vyberte zakaznika';
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
			error = e instanceof Error ? e.message : 'Nepodarilo se vytvorit opakujici se fakturu';
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Nova opakujici se faktura - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<a href="/recurring" class="text-sm text-secondary hover:text-primary"
		>&larr; Zpet na opakujici se faktury</a
	>
	<h1 class="mt-2 text-xl font-semibold text-primary">Nova opakujici se faktura</h1>

	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-danger/20 bg-danger-bg p-4 text-sm text-danger"
		>
			{error}
		</div>
	{/if}

	<form
		onsubmit={(e) => {
			e.preventDefault();
			handleSubmit();
		}}
		class="mt-6 space-y-8"
	>
		<!-- Basic info -->
		<Card>
			<h2 class="text-base font-semibold text-primary">Zakladni udaje</h2>
			<div class="mt-4 space-y-4">
				<div>
					<label for="name" class="block text-sm font-medium text-secondary">Nazev sablony</label>
					<input
						id="name"
						type="text"
						bind:value={form.name}
						required
						class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						placeholder="napr. Mesicni hosting"
					/>
				</div>
				<div>
					<label for="customer" class="block text-sm font-medium text-secondary">Zakaznik</label>
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
			<h2 class="text-base font-semibold text-primary">Opakovani</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
				<div>
					<label for="frequency" class="block text-sm font-medium text-secondary">Frekvence <HelpTip topic="frekvence-opakovani" /></label>
					<select
						id="frequency"
						bind:value={form.frequency}
						class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
					>
						<option value="weekly">Tydenni</option>
						<option value="monthly">Mesicni</option>
						<option value="quarterly">Ctvrtletni</option>
						<option value="yearly">Rocni</option>
					</select>
				</div>
				<div>
					<label for="next_issue_date" class="block text-sm font-medium text-secondary"
						>Dalsi vystaveni</label
					>
					<DateInput id="next_issue_date" bind:value={form.next_issue_date} required />
				</div>
				<div>
					<label for="end_date" class="block text-sm font-medium text-secondary"
						>Konec opakovani (volitelne)</label
					>
					<DateInput id="end_date" bind:value={form.end_date} />
				</div>
			</div>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label for="payment" class="block text-sm font-medium text-secondary">Zpusob platby</label>
					<select
						id="payment"
						bind:value={form.payment_method}
						class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
					>
						<option value="bank_transfer">Bankovni prevod</option>
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
			<h2 class="text-base font-semibold text-primary">Poznamky</h2>
			<div class="mt-4">
				<label for="notes" class="block text-sm font-medium text-secondary"
					>Poznamka na fakture</label
				>
				<textarea
					id="notes"
					bind:value={form.notes}
					rows="2"
					class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
				></textarea>
			</div>
		</Card>

		<!-- Actions -->
		<div class="flex gap-3">
			<Button type="submit" variant="primary" disabled={saving}>
				{saving ? 'Ukladam...' : 'Ulozit'}
			</Button>
			<Button variant="secondary" href="/recurring">
				Zrusit
			</Button>
		</div>
	</form>
</div>
