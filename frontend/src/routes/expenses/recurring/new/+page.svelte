<script lang="ts">
	import { goto } from '$app/navigation';
	import {
		recurringExpensesApi,
		contactsApi,
		type Contact,
		type RecurringExpense
	} from '$lib/api/client';
	import { formatCZK, toHalere } from '$lib/utils/money';
	import { toISODate } from '$lib/utils/date';
	import DateInput from '$lib/components/DateInput.svelte';
	import CategoryPicker from '$lib/components/CategoryPicker.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';

	let contacts = $state<Contact[]>([]);
	let saving = $state(false);
	let error = $state<string | null>(null);

	let form = $state({
		name: '',
		vendor_id: null as number | null,
		category: '',
		description: '',
		amount: 0,
		currency_code: 'CZK',
		vat_rate_percent: 21,
		is_tax_deductible: true,
		business_percent: 100,
		payment_method: 'bank_transfer',
		notes: '',
		frequency: 'monthly',
		next_issue_date: toISODate(new Date()),
		end_date: '',
		is_active: true
	});

	let vatAmount = $derived((form.amount * form.vat_rate_percent) / (100 + form.vat_rate_percent));

	$effect(() => {
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
			await recurringExpensesApi.create({
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
			goto('/expenses/recurring');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se uložit opakovaný náklad';
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Nový opakovaný náklad - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<a href="/expenses/recurring" class="text-sm text-secondary hover:text-primary"
		>&larr; Zpět na opakované náklady</a
	>
	<h1 class="mt-2 text-xl font-semibold text-primary">Nový opakovaný náklad</h1>

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
		class="mt-6 space-y-6"
	>
		<!-- Name & Schedule -->
		<Card>
			<h2 class="text-base font-semibold text-primary">Základní údaje</h2>
			<div class="mt-4 space-y-4">
				<div>
					<label for="name" class="block text-sm font-medium text-secondary">Název *</label>
					<input
						id="name"
						type="text"
						bind:value={form.name}
						required
						placeholder="např. Hosting, Kancelář, Telefon..."
						class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
					/>
				</div>
				<div>
					<label for="description" class="block text-sm font-medium text-secondary"
						>Popis nákladu *</label
					>
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

		<!-- Schedule -->
		<Card>
			<h2 class="text-base font-semibold text-primary">Plánování</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
				<div>
					<label for="frequency" class="block text-sm font-medium text-secondary">Frekvence *</label>
					<select
						id="frequency"
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
					<label for="next_issue_date" class="block text-sm font-medium text-secondary"
						>Další datum *</label
					>
					<DateInput id="next_issue_date" bind:value={form.next_issue_date} required />
				</div>
				<div>
					<label for="end_date" class="block text-sm font-medium text-secondary"
						>Datum ukončení</label
					>
					<DateInput id="end_date" bind:value={form.end_date} />
				</div>
			</div>
		</Card>

		<!-- Amount & VAT -->
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
					<label for="vat_rate" class="block text-sm font-medium text-secondary">Sazba DPH</label>
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
						>Daňově uznatelný náklad</label
					>
				</div>
				<div>
					<label for="business_percent" class="block text-sm font-medium text-secondary"
						>Podíl pro podnikání (%)</label
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
				<textarea
					bind:value={form.notes}
					rows="3"
					placeholder="Volitelné poznámky..."
					class="w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
				></textarea>
			</div>
		</Card>

		<!-- Actions -->
		<div class="flex gap-3">
			<Button variant="primary" type="submit" disabled={saving}>
				{saving ? 'Ukládám...' : 'Uložit'}
			</Button>
			<Button variant="secondary" href="/expenses/recurring">
				Zrušit
			</Button>
		</div>
	</form>
</div>
