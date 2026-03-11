<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { contactsApi, type Contact } from '$lib/api/client';
	import Button from '$lib/ui/Button.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';

	let contact = $state<Contact | null>(null);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state<string | null>(null);

	let form = $state({
		type: 'company' as 'company' | 'individual',
		name: '',
		ico: '',
		dic: '',
		street: '',
		city: '',
		zip: '',
		country: 'CZ',
		email: '',
		phone: '',
		web: '',
		bank_account: '',
		bank_code: '',
		iban: '',
		swift: '',
		payment_terms_days: 14,
		tags: '',
		notes: ''
	});

	let isNew = $derived(page.params.id === 'new');
	let contactId = $derived(isNew ? null : Number(page.params.id));

	$effect(() => {
		if (isNew) {
			loading = false;
			return;
		}
		loadContact();
	});

	async function loadContact() {
		if (!contactId) return;
		loading = true;
		error = null;
		try {
			contact = await contactsApi.getById(contactId);
			form = {
				type: contact.type as 'company' | 'individual',
				name: contact.name,
				ico: contact.ico,
				dic: contact.dic,
				street: contact.street,
				city: contact.city,
				zip: contact.zip,
				country: contact.country,
				email: contact.email,
				phone: contact.phone,
				web: contact.web,
				bank_account: contact.bank_account,
				bank_code: contact.bank_code,
				iban: contact.iban,
				swift: contact.swift,
				payment_terms_days: contact.payment_terms_days,
				tags: contact.tags,
				notes: contact.notes
			};
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load contact';
		} finally {
			loading = false;
		}
	}

	async function lookupAres() {
		if (!form.ico) return;
		try {
			const result = await contactsApi.lookupAres(form.ico);
			form.name = result.name;
			form.dic = result.dic;
			form.street = result.street;
			form.city = result.city;
			form.zip = result.zip;
			form.country = result.country;
		} catch (e) {
			error = e instanceof Error ? e.message : 'ARES lookup failed';
		}
	}

	async function handleSubmit() {
		saving = true;
		error = null;
		try {
			if (isNew) {
				await contactsApi.create(form);
			} else if (contactId) {
				await contactsApi.update(contactId, form);
			}
			goto('/contacts');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save contact';
		} finally {
			saving = false;
		}
	}

	async function handleDelete() {
		if (!contactId) return;
		if (!confirm('Opravdu chcete smazat tento kontakt?')) return;
		try {
			await contactsApi.delete(contactId);
			goto('/contacts');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete contact';
		}
	}

	const inputClass =
		'mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none';
</script>

<svelte:head>
	<title>{isNew ? 'Novy kontakt' : 'Upravit kontakt'} - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<div class="flex items-center justify-between">
		<div>
			<a href="/contacts" class="text-sm text-secondary hover:text-primary">&larr; Zpet na kontakty</a>
			<h1 class="mt-2 text-xl font-semibold text-primary">
				{isNew ? 'Novy kontakt' : 'Upravit kontakt'}
			</h1>
		</div>
		{#if !isNew}
			<Button variant="danger" onclick={handleDelete}>
				Smazat
			</Button>
		{/if}
	</div>

	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-danger/20 bg-danger-bg p-4 text-sm text-danger"
		>
			{error}
		</div>
	{/if}

	{#if loading}
		<div class="mt-8 flex items-center justify-center">
			<div role="status">
				<div
					class="h-8 w-8 animate-spin rounded-full border-4 border-border border-t-accent"
				></div>
				<span class="sr-only">Nacitani...</span>
			</div>
		</div>
	{:else}
		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleSubmit();
			}}
			class="mt-6 space-y-6"
		>
			<!-- Type -->
			<div>
				<label for="type" class="block text-sm font-medium text-secondary">Typ</label>
				<select
					id="type"
					bind:value={form.type}
					class={inputClass}
				>
					<option value="company">Firma</option>
					<option value="individual">Fyzicka osoba</option>
				</select>
			</div>

			<!-- ICO + ARES lookup -->
			<div>
				<label for="ico" class="block text-sm font-medium text-secondary">ICO <HelpTip topic="ico" /></label>
				<div class="mt-1 flex gap-2">
					<input
						id="ico"
						type="text"
						bind:value={form.ico}
						class="flex-1 rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
					/>
					<Button variant="secondary" size="sm" onclick={lookupAres}>
						ARES
					</Button>
				</div>
			</div>

			<!-- Name -->
			<div>
				<label for="name" class="block text-sm font-medium text-secondary">Nazev</label>
				<input
					id="name"
					type="text"
					bind:value={form.name}
					required
					class={inputClass}
				/>
			</div>

			<!-- DIC -->
			<div>
				<label for="dic" class="block text-sm font-medium text-secondary">DIC <HelpTip topic="dic" /></label>
				<input
					id="dic"
					type="text"
					bind:value={form.dic}
					class={inputClass}
				/>
			</div>

			<!-- Address -->
			<fieldset class="space-y-4">
				<legend class="text-sm font-semibold text-primary">Adresa</legend>
				<div>
					<label for="street" class="block text-sm font-medium text-secondary">Ulice</label>
					<input
						id="street"
						type="text"
						bind:value={form.street}
						class={inputClass}
					/>
				</div>
				<div class="grid grid-cols-2 gap-4">
					<div>
						<label for="city" class="block text-sm font-medium text-secondary">Mesto</label>
						<input
							id="city"
							type="text"
							bind:value={form.city}
							class={inputClass}
						/>
					</div>
					<div>
						<label for="zip" class="block text-sm font-medium text-secondary">PSC</label>
						<input
							id="zip"
							type="text"
							bind:value={form.zip}
							class={inputClass}
						/>
					</div>
				</div>
			</fieldset>

			<!-- Contact info -->
			<fieldset class="space-y-4">
				<legend class="text-sm font-semibold text-primary">Kontaktni udaje</legend>
				<div class="grid grid-cols-2 gap-4">
					<div>
						<label for="email" class="block text-sm font-medium text-secondary">Email</label>
						<input
							id="email"
							type="email"
							bind:value={form.email}
							class={inputClass}
						/>
					</div>
					<div>
						<label for="phone" class="block text-sm font-medium text-secondary">Telefon</label>
						<input
							id="phone"
							type="text"
							bind:value={form.phone}
							class={inputClass}
						/>
					</div>
				</div>
			</fieldset>

			<!-- Bank details -->
			<fieldset class="space-y-4">
				<legend class="text-sm font-semibold text-primary">Bankovni udaje</legend>
				<div class="grid grid-cols-2 gap-4">
					<div>
						<label for="bank_account" class="block text-sm font-medium text-secondary"
							>Cislo uctu</label
						>
						<input
							id="bank_account"
							type="text"
							bind:value={form.bank_account}
							class={inputClass}
						/>
					</div>
					<div>
						<label for="bank_code" class="block text-sm font-medium text-secondary">Kod banky</label>
						<input
							id="bank_code"
							type="text"
							bind:value={form.bank_code}
							class={inputClass}
						/>
					</div>
				</div>
			</fieldset>

			<!-- Payment terms -->
			<div>
				<label for="payment_terms" class="block text-sm font-medium text-secondary"
					>Splatnost (dny)</label
				>
				<input
					id="payment_terms"
					type="number"
					bind:value={form.payment_terms_days}
					min="0"
					class="mt-1 w-32 rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
				/>
			</div>

			<!-- Notes -->
			<div>
				<label for="notes" class="block text-sm font-medium text-secondary">Poznamky</label>
				<textarea
					id="notes"
					bind:value={form.notes}
					rows="3"
					class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
				></textarea>
			</div>

			<!-- Submit -->
			<div class="flex gap-3 pt-4">
				<Button variant="primary" type="submit" disabled={saving}>
					{saving ? 'Ukladam...' : 'Ulozit'}
				</Button>
				<Button variant="secondary" href="/contacts">
					Zrusit
				</Button>
			</div>
		</form>
	{/if}
</div>
