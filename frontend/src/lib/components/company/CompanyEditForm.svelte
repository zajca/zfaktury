<script lang="ts">
	import type { NewCompany } from '$lib/api/client';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';

	interface Props {
		form: NewCompany;
		saving: boolean;
		// onaresLookup is optional — the create flow wires it up to fetch from
		// ARES, the edit flow leaves it undefined and the ARES button is hidden.
		onaresLookup?: () => void | Promise<void>;
		aresLoading?: boolean;
		onsubmit: () => void | Promise<void>;
		cancelHref?: string;
		saveLabel?: string;
	}

	let {
		form = $bindable(),
		saving,
		onaresLookup,
		aresLoading = false,
		onsubmit,
		cancelHref = '/companies',
		saveLabel = 'Uložit'
	}: Props = $props();

	const inputClass =
		'mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none';
</script>

<form
	onsubmit={(e) => {
		e.preventDefault();
		onsubmit();
	}}
	class="space-y-6"
>
	<!-- Identity -->
	<Card>
		<h2 class="text-base font-semibold text-primary">Údaje firmy</h2>
		<p class="mt-1 text-sm text-tertiary">
			Identifikační údaje, které se budou zobrazovat na fakturách.
		</p>
		<div class="mt-4 space-y-4">
			<!-- IČO + ARES lookup -->
			<div>
				<label for="company_ico" class="block text-sm font-medium text-secondary">
					IČO <HelpTip topic="ico" />
				</label>
				<div class="mt-1 flex gap-2">
					<input
						id="company_ico"
						type="text"
						bind:value={form.ico}
						required
						class="flex-1 rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
					/>
					{#if onaresLookup}
						<Button
							variant="secondary"
							size="sm"
							onclick={onaresLookup}
							disabled={aresLoading}
							title="Doplní název, DIČ a adresu z registru ARES podle zadaného IČO"
						>
							{aresLoading ? 'Načítám...' : 'ARES'}
						</Button>
					{/if}
				</div>
			</div>

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label for="company_name" class="block text-sm font-medium text-secondary">
						Název *
					</label>
					<input
						id="company_name"
						type="text"
						bind:value={form.name}
						required
						class={inputClass}
					/>
				</div>
				<div>
					<label for="company_legal_name" class="block text-sm font-medium text-secondary">
						Obchodní jméno
					</label>
					<input
						id="company_legal_name"
						type="text"
						bind:value={form.legal_name}
						class={inputClass}
					/>
				</div>
			</div>

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label for="company_first_name" class="block text-sm font-medium text-secondary">
						Jméno
					</label>
					<input
						id="company_first_name"
						type="text"
						bind:value={form.first_name}
						class={inputClass}
					/>
				</div>
				<div>
					<label for="company_last_name" class="block text-sm font-medium text-secondary">
						Příjmení
					</label>
					<input
						id="company_last_name"
						type="text"
						bind:value={form.last_name}
						class={inputClass}
					/>
				</div>
			</div>

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label for="company_dic" class="block text-sm font-medium text-secondary">
						DIČ <HelpTip topic="dic" />
					</label>
					<input id="company_dic" type="text" bind:value={form.dic} class={inputClass} />
				</div>
				<div class="flex items-end gap-3 pb-2">
					<input
						id="company_vat_registered"
						type="checkbox"
						bind:checked={form.vat_registered}
						class="h-4 w-4 rounded border-border accent-accent"
					/>
					<label for="company_vat_registered" class="text-sm font-medium text-secondary">
						Plátce DPH <HelpTip topic="platce-dph" />
					</label>
				</div>
			</div>
		</div>
	</Card>

	<!-- Address -->
	<Card>
		<h2 class="text-base font-semibold text-primary">Adresa</h2>
		<div class="mt-4 space-y-4">
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
				<div class="sm:col-span-2">
					<label for="company_street" class="block text-sm font-medium text-secondary">
						Ulice
					</label>
					<input
						id="company_street"
						type="text"
						bind:value={form.street}
						class={inputClass}
					/>
				</div>
				<div>
					<label for="company_house_number" class="block text-sm font-medium text-secondary">
						Číslo popisné
					</label>
					<input
						id="company_house_number"
						type="text"
						bind:value={form.house_number}
						class={inputClass}
					/>
				</div>
			</div>
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label for="company_city" class="block text-sm font-medium text-secondary">Město</label>
					<input id="company_city" type="text" bind:value={form.city} class={inputClass} />
				</div>
				<div>
					<label for="company_zip" class="block text-sm font-medium text-secondary">PSČ</label>
					<input id="company_zip" type="text" bind:value={form.zip} class={inputClass} />
				</div>
			</div>
		</div>
	</Card>

	<!-- Contact -->
	<Card>
		<h2 class="text-base font-semibold text-primary">Kontaktní údaje</h2>
		<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
			<div>
				<label for="company_email" class="block text-sm font-medium text-secondary">Email</label>
				<input
					id="company_email"
					type="email"
					bind:value={form.email}
					class={inputClass}
				/>
			</div>
			<div>
				<label for="company_phone" class="block text-sm font-medium text-secondary">Telefon</label>
				<input id="company_phone" type="text" bind:value={form.phone} class={inputClass} />
			</div>
		</div>
	</Card>

	<!-- Bank details -->
	<Card>
		<h2 class="text-base font-semibold text-primary">Bankovní účty</h2>
		<p class="mt-1 text-sm text-tertiary">Účet pro příjem plateb na fakturách.</p>
		<div class="mt-4 space-y-4">
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label for="company_bank_account" class="block text-sm font-medium text-secondary">
						Číslo účtu
					</label>
					<input
						id="company_bank_account"
						type="text"
						bind:value={form.bank_account}
						class={inputClass}
					/>
				</div>
				<div>
					<label for="company_bank_code" class="block text-sm font-medium text-secondary">
						Kód banky
					</label>
					<input
						id="company_bank_code"
						type="text"
						bind:value={form.bank_code}
						class={inputClass}
					/>
				</div>
			</div>
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label for="company_iban" class="block text-sm font-medium text-secondary">
						IBAN <HelpTip topic="iban" />
					</label>
					<input id="company_iban" type="text" bind:value={form.iban} class={inputClass} />
				</div>
				<div>
					<label for="company_swift" class="block text-sm font-medium text-secondary">
						SWIFT/BIC <HelpTip topic="swift-bic" />
					</label>
					<input id="company_swift" type="text" bind:value={form.swift} class={inputClass} />
				</div>
			</div>
		</div>
	</Card>

	<FormActions {saving} {saveLabel} {cancelHref} class="pb-8" />
</form>
