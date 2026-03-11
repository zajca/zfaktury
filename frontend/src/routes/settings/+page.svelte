<script lang="ts">
	import { settingsApi, type Settings } from '$lib/api/client';
	import Card from '$lib/ui/Card.svelte';
	import Button from '$lib/ui/Button.svelte';

	let settings = $state<Settings>({});
	let loading = $state(true);
	let saving = $state(false);
	let error = $state<string | null>(null);
	let success = $state(false);

	$effect(() => {
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
		success = false;
		try {
			settings = await settingsApi.update(settings);
			success = true;
			setTimeout(() => {
				success = false;
			}, 3000);
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
	<title>Nastavení - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<h1 class="text-xl font-semibold text-primary">Nastavení</h1>
	<p class="mt-1 text-sm text-tertiary">Konfigurace aplikace a údaje OSVČ</p>

	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-danger/20 bg-danger-bg p-3 text-sm text-danger"
		>
			{error}
		</div>
	{/if}

	{#if success}
		<div class="mt-4 rounded-lg border border-success/20 bg-success-bg p-3 text-sm text-success">
			Nastavení bylo uloženo.
		</div>
	{/if}

	<!-- Quick links -->
	<Card class="mt-6">
		<h2 class="text-base font-semibold text-primary">Správa</h2>
		<div class="mt-3 space-y-2">
			<a
				href="/settings/sequences"
				class="flex items-center justify-between rounded-lg border border-border px-4 py-3 hover:bg-hover transition-colors"
			>
				<div>
					<span class="text-sm font-medium text-primary">Číselné řady faktur</span>
					<p class="mt-0.5 text-xs text-tertiary">Správa číselných řad pro faktury</p>
				</div>
				<svg
					class="h-5 w-5 text-muted"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="1.5"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
				</svg>
			</a>
			<a
				href="/settings/categories"
				class="flex items-center justify-between rounded-lg border border-border px-4 py-3 hover:bg-hover transition-colors"
			>
				<div>
					<span class="text-sm font-medium text-primary">Kategorie nákladů</span>
					<p class="mt-0.5 text-xs text-tertiary">Správa kategorií pro třídění nákladů</p>
				</div>
				<svg
					class="h-5 w-5 text-muted"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="1.5"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
				</svg>
			</a>
		</div>
	</Card>

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
							<label for="ico" class="block text-sm font-medium text-secondary">IČO</label>
							<input
								id="ico"
								type="text"
								value={field('ico')}
								oninput={(e) => setField('ico', (e.target as HTMLInputElement).value)}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div>
							<label for="dic" class="block text-sm font-medium text-secondary">DIČ</label>
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
						<label for="vat_registered" class="text-sm font-medium text-secondary">Plátce DPH</label>
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
							<label for="iban" class="block text-sm font-medium text-secondary">IBAN</label>
							<input
								id="iban"
								type="text"
								value={field('iban')}
								oninput={(e) => setField('iban', (e.target as HTMLInputElement).value)}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div>
							<label for="swift" class="block text-sm font-medium text-secondary">SWIFT/BIC</label>
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

			<!-- Save -->
			<div class="flex gap-3 pb-8">
				<Button type="submit" variant="primary" disabled={saving}>
					{saving ? 'Ukládám...' : 'Uložit nastavení'}
				</Button>
			</div>
		</form>
	{/if}
</div>
