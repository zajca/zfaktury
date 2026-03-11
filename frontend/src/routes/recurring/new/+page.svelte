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

<div class="mx-auto max-w-4xl">
	<a href="/recurring" class="text-sm text-blue-600 hover:text-blue-800"
		>&larr; Zpet na opakujici se faktury</a
	>
	<h1 class="mt-2 text-2xl font-bold text-gray-900">Nova opakujici se faktura</h1>

	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700"
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
		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Zakladni udaje</h2>
			<div class="mt-4 space-y-4">
				<div>
					<label for="name" class="block text-sm font-medium text-gray-700">Nazev sablony</label>
					<input
						id="name"
						type="text"
						bind:value={form.name}
						required
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
						placeholder="napr. Mesicni hosting"
					/>
				</div>
				<div>
					<label for="customer" class="block text-sm font-medium text-gray-700">Zakaznik</label>
					<select
						id="customer"
						bind:value={form.customer_id}
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
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
		</div>

		<!-- Schedule -->
		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Opakovani</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
				<div>
					<label for="frequency" class="block text-sm font-medium text-gray-700">Frekvence</label>
					<select
						id="frequency"
						bind:value={form.frequency}
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					>
						<option value="weekly">Tydenni</option>
						<option value="monthly">Mesicni</option>
						<option value="quarterly">Ctvrtletni</option>
						<option value="yearly">Rocni</option>
					</select>
				</div>
				<div>
					<label for="next_issue_date" class="block text-sm font-medium text-gray-700"
						>Dalsi vystaveni</label
					>
					<DateInput id="next_issue_date" bind:value={form.next_issue_date} required />
				</div>
				<div>
					<label for="end_date" class="block text-sm font-medium text-gray-700"
						>Konec opakovani (volitelne)</label
					>
					<DateInput id="end_date" bind:value={form.end_date} />
				</div>
			</div>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label for="payment" class="block text-sm font-medium text-gray-700">Zpusob platby</label>
					<select
						id="payment"
						bind:value={form.payment_method}
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					>
						<option value="bank_transfer">Bankovni prevod</option>
						<option value="cash">Hotovost</option>
						<option value="card">Karta</option>
					</select>
				</div>
			</div>
		</div>

		<!-- Line Items -->
		<InvoiceItemsEditor bind:items />

		<!-- Notes -->
		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Poznamky</h2>
			<div class="mt-4">
				<label for="notes" class="block text-sm font-medium text-gray-700"
					>Poznamka na fakture</label
				>
				<textarea
					id="notes"
					bind:value={form.notes}
					rows="2"
					class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
				></textarea>
			</div>
		</div>

		<!-- Actions -->
		<div class="flex gap-3">
			<button
				type="submit"
				disabled={saving}
				class="rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50 transition-colors"
			>
				{saving ? 'Ukladam...' : 'Ulozit'}
			</button>
			<a
				href="/recurring"
				class="rounded-lg border border-gray-300 px-6 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
			>
				Zrusit
			</a>
		</div>
	</form>
</div>
