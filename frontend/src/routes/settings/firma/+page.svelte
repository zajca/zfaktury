<script lang="ts">
	import { settingsApi, type Settings } from '$lib/api/client';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';
	import { toastSuccess } from '$lib/data/toast-state.svelte';

	let settings = $state<Settings>({});
	let loading = $state(true);
	let saving = $state(false);
	let error = $state<string | null>(null);
	import { onMount } from 'svelte';

	onMount(() => {
		loadSettings();
	});

	async function loadSettings() {
		loading = true;
		error = null;
		try {
			settings = await settingsApi.getAll();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst nastavení';
		} finally {
			loading = false;
		}
	}

	async function handleSave() {
		saving = true;
		error = null;
		try {
			settings = await settingsApi.update(settings);
			toastSuccess('Nastavení firmy uloženo');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se uložit nastavení';
		} finally {
			saving = false;
		}
	}

	function field(key: string): string {
		return settings[key] ?? '';
	}

	function setField(key: string, value: string) {
		settings[key] = value;
	}
</script>

<svelte:head>
	<title>Firma - Nastavení - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<PageHeader title="Firma" description="Údaje podnikatele, adresa a bankovní účty" />

	<ErrorAlert {error} class="mt-4" />

	{#if loading}
		<LoadingSpinner class="mt-8" />
	{:else}
		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleSave();
			}}
			class="mt-6 space-y-6"
		>
			<!-- Identity -->
			<Card>
				<h2 class="text-base font-semibold text-primary">Údaje podnikatele</h2>
				<p class="mt-1 text-sm text-tertiary">Tyto údaje se budou zobrazovat na fakturách.</p>
				<div class="mt-4 space-y-4">
					<div>
						<label for="company_name" class="block text-sm font-medium text-secondary"
							>Název / Jméno *</label
						>
						<input
							id="company_name"
							type="text"
							value={field('company_name')}
							oninput={(e) => setField('company_name', (e.target as HTMLInputElement).value)}
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						/>
					</div>
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div>
							<label for="ico" class="block text-sm font-medium text-secondary">IČO <HelpTip topic="ico" /></label>
							<input
								id="ico"
								type="text"
								value={field('ico')}
								oninput={(e) => setField('ico', (e.target as HTMLInputElement).value)}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div>
							<label for="dic" class="block text-sm font-medium text-secondary">DIČ <HelpTip topic="dic" /></label>
							<input
								id="dic"
								type="text"
								value={field('dic')}
								oninput={(e) => setField('dic', (e.target as HTMLInputElement).value)}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
					</div>
					<div class="flex items-center gap-3">
						<input
							id="vat_registered"
							type="checkbox"
							checked={field('vat_registered') === 'true'}
							onchange={(e) =>
								setField(
									'vat_registered',
									(e.target as HTMLInputElement).checked ? 'true' : 'false'
								)}
							class="h-4 w-4 rounded border-border accent-accent"
						/>
						<label for="vat_registered" class="text-sm font-medium text-secondary">Plátce DPH <HelpTip topic="platce-dph" /></label>
					</div>
				</div>
			</Card>

			<!-- Address -->
			<Card>
				<h2 class="text-base font-semibold text-primary">Adresa</h2>
				<div class="mt-4 space-y-4">
					<div>
						<label for="street" class="block text-sm font-medium text-secondary">Ulice</label>
						<input
							id="street"
							type="text"
							value={field('street')}
							oninput={(e) => setField('street', (e.target as HTMLInputElement).value)}
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						/>
					</div>
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div>
							<label for="city" class="block text-sm font-medium text-secondary">Město</label>
							<input
								id="city"
								type="text"
								value={field('city')}
								oninput={(e) => setField('city', (e.target as HTMLInputElement).value)}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div>
							<label for="zip" class="block text-sm font-medium text-secondary">PSČ</label>
							<input
								id="zip"
								type="text"
								value={field('zip')}
								oninput={(e) => setField('zip', (e.target as HTMLInputElement).value)}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
					</div>
				</div>
			</Card>

			<!-- Contact -->
			<Card>
				<h2 class="text-base font-semibold text-primary">Kontaktní údaje</h2>
				<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div>
						<label for="email" class="block text-sm font-medium text-secondary">Email</label>
						<input
							id="email"
							type="email"
							value={field('email')}
							oninput={(e) => setField('email', (e.target as HTMLInputElement).value)}
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						/>
					</div>
					<div>
						<label for="phone" class="block text-sm font-medium text-secondary">Telefon</label>
						<input
							id="phone"
							type="text"
							value={field('phone')}
							oninput={(e) => setField('phone', (e.target as HTMLInputElement).value)}
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						/>
					</div>
				</div>
			</Card>

			<!-- Bank accounts -->
			<Card>
				<h2 class="text-base font-semibold text-primary">Bankovní účty</h2>
				<p class="mt-1 text-sm text-tertiary">Účet pro příjem plateb na fakturách.</p>
				<div class="mt-4 space-y-4">
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div>
							<label for="bank_account" class="block text-sm font-medium text-secondary"
								>Číslo účtu</label
							>
							<input
								id="bank_account"
								type="text"
								value={field('bank_account')}
								oninput={(e) => setField('bank_account', (e.target as HTMLInputElement).value)}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div>
							<label for="bank_code" class="block text-sm font-medium text-secondary"
								>Kód banky</label
							>
							<input
								id="bank_code"
								type="text"
								value={field('bank_code')}
								oninput={(e) => setField('bank_code', (e.target as HTMLInputElement).value)}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
					</div>
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div>
							<label for="iban" class="block text-sm font-medium text-secondary">IBAN <HelpTip topic="iban" /></label>
							<input
								id="iban"
								type="text"
								value={field('iban')}
								oninput={(e) => setField('iban', (e.target as HTMLInputElement).value)}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div>
							<label for="swift" class="block text-sm font-medium text-secondary">SWIFT/BIC <HelpTip topic="swift-bic" /></label>
							<input
								id="swift"
								type="text"
								value={field('swift')}
								oninput={(e) => setField('swift', (e.target as HTMLInputElement).value)}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
					</div>
				</div>
			</Card>

			<!-- Kody uradu -->
			<Card>
				<h2 class="text-base font-semibold text-primary">Kódy úřadů</h2>
				<p class="mt-1 text-sm text-tertiary">Kódy pro daňové přiznání a přehledy OSVČ.</p>
				<div class="mt-4 space-y-4">
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<label for="financni_urad_code" class="block text-sm font-medium text-secondary">Kód finančního úřadu</label>
							<input
								id="financni_urad_code"
								type="text"
								value={field('financni_urad_code')}
								oninput={(e) => setField('financni_urad_code', (e.target as HTMLInputElement).value)}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
								placeholder="např. 0451"
							/>
						</div>
						<div>
							<label for="cssz_code" class="block text-sm font-medium text-secondary">Kód OSSZ</label>
							<input
								id="cssz_code"
								type="text"
								value={field('cssz_code')}
								oninput={(e) => setField('cssz_code', (e.target as HTMLInputElement).value)}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
								placeholder="např. prehledosvc"
							/>
						</div>
						<div>
							<label for="health_insurance_code" class="block text-sm font-medium text-secondary">Kód zdravotní pojišťovny</label>
							<input
								id="health_insurance_code"
								type="text"
								value={field('health_insurance_code')}
								oninput={(e) => setField('health_insurance_code', (e.target as HTMLInputElement).value)}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
								placeholder="např. 111 (VZP)"
							/>
						</div>
					</div>
					<div class="mt-2">
						<a href="/tax/prepayments" class="text-sm text-accent hover:underline">
							Daňové zálohy a paušální výdaje →
						</a>
					</div>
				</div>
			</Card>

			<!-- Save -->
			<FormActions {saving} saveLabel="Uložit nastavení" class="pb-8" />
		</form>
	{/if}
</div>
