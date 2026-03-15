<script lang="ts">
	import { pdfSettingsApi, type PDFSettings } from '$lib/api/client';
	import Card from '$lib/ui/Card.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import Button from '$lib/ui/Button.svelte';
	import { toastSuccess, toastError } from '$lib/data/toast-state.svelte';
	import { onMount } from 'svelte';

	let settings = $state<PDFSettings | null>(null);
	let loading = $state(true);
	let saving = $state(false);
	let uploading = $state(false);
	let error = $state<string | null>(null);

	let accentColor = $state('#2563eb');
	let footerText = $state('');
	let showQR = $state(true);
	let showBankDetails = $state(true);
	let fontSize = $state<'small' | 'normal' | 'large'>('normal');

	onMount(() => {
		loadSettings();
	});

	async function loadSettings() {
		loading = true;
		error = null;
		try {
			settings = await pdfSettingsApi.get();
			accentColor = settings.accent_color;
			footerText = settings.footer_text;
			showQR = settings.show_qr;
			showBankDetails = settings.show_bank_details;
			fontSize = settings.font_size;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst nastavení PDF';
		} finally {
			loading = false;
		}
	}

	async function handleSave() {
		saving = true;
		try {
			settings = await pdfSettingsApi.update({
				accent_color: accentColor,
				footer_text: footerText,
				show_qr: showQR,
				show_bank_details: showBankDetails,
				font_size: fontSize
			});
			toastSuccess('Nastavení PDF šablony uloženo');
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se uložit nastavení');
		} finally {
			saving = false;
		}
	}

	async function handleLogoUpload(e: Event) {
		const input = e.target as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;

		uploading = true;
		try {
			await pdfSettingsApi.uploadLogo(file);
			await loadSettings();
			toastSuccess('Logo nahráno');
		} catch (err) {
			toastError(err instanceof Error ? err.message : 'Nepodařilo se nahrát logo');
		} finally {
			uploading = false;
			input.value = '';
		}
	}

	async function handleLogoDelete() {
		try {
			await pdfSettingsApi.deleteLogo();
			await loadSettings();
			toastSuccess('Logo odstraněno');
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se odstranit logo');
		}
	}

	function openPreview() {
		window.open(pdfSettingsApi.previewUrl(), '_blank');
	}
</script>

<svelte:head>
	<title>PDF šablona - Nastavení - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-3xl">
	<PageHeader
		title="PDF šablona"
		description="Přizpůsobení vzhledu generovaných faktur ve formátu PDF"
	/>

	{#if loading}
		<LoadingSpinner />
	{:else if error}
		<ErrorAlert {error} />
	{:else if settings}
		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleSave();
			}}
			class="space-y-6"
		>
			<!-- Logo -->
			<Card>
				<h3 class="text-sm font-semibold text-primary mb-3">Logo firmy</h3>
				<div class="space-y-4">
					{#if settings.has_logo}
						<div class="flex items-center gap-4">
							<img
								src={pdfSettingsApi.logoUrl()}
								alt="Logo firmy"
								class="h-16 max-w-48 object-contain rounded border border-border p-2"
							/>
							<button
								type="button"
								onclick={handleLogoDelete}
								class="text-sm text-danger hover:underline"
							>
								Odstranit logo
							</button>
						</div>
					{/if}
					<div>
						<label class="block text-sm font-medium text-secondary mb-1">
							{settings.has_logo ? 'Změnit logo' : 'Nahrát logo'}
						</label>
						<input
							type="file"
							accept="image/png,image/jpeg,image/svg+xml"
							onchange={handleLogoUpload}
							disabled={uploading}
							class="block text-sm text-secondary file:mr-3 file:rounded-md file:border-0 file:bg-accent-muted file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-accent-text hover:file:bg-accent-muted/80"
						/>
						<p class="mt-1 text-xs text-muted">PNG, JPEG nebo SVG, max 2 MB</p>
					</div>
				</div>
			</Card>

			<!-- Vzhled -->
			<Card>
				<h3 class="text-sm font-semibold text-primary mb-3">Vzhled</h3>
				<div class="space-y-4">
					<div>
						<label for="accent-color" class="block text-sm font-medium text-secondary mb-1">
							Barva akcentu
						</label>
						<div class="flex items-center gap-3">
							<input
								id="accent-color"
								type="color"
								bind:value={accentColor}
								class="h-9 w-14 cursor-pointer rounded border border-border"
							/>
							<input
								type="text"
								bind:value={accentColor}
								class="w-28 rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-primary"
								pattern="#[0-9a-fA-F]{6}"
								maxlength="7"
							/>
						</div>
						<p class="mt-1 text-xs text-muted">Barva záhlaví a akcentových prvků</p>
					</div>

					<div>
						<label for="footer-text" class="block text-sm font-medium text-secondary mb-1">
							Text v patičce
						</label>
						<input
							id="footer-text"
							type="text"
							bind:value={footerText}
							maxlength="200"
							placeholder="např. Děkujeme za spolupráci"
							class="w-full rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-primary placeholder:text-muted"
						/>
						<p class="mt-1 text-xs text-muted">{footerText.length}/200 znaků</p>
					</div>

					<div>
						<span class="block text-sm font-medium text-secondary mb-2">Velikost písma</span>
						<div class="flex gap-4">
							{#each [{ value: 'small', label: 'Malý (9pt)' }, { value: 'normal', label: 'Normální (10pt)' }, { value: 'large', label: 'Velký (11pt)' }] as option (option.value)}
								<label class="flex items-center gap-2 text-sm text-secondary cursor-pointer">
									<input
										type="radio"
										name="font-size"
										value={option.value}
										checked={fontSize === option.value}
										onchange={() => (fontSize = option.value as 'small' | 'normal' | 'large')}
										class="text-accent"
									/>
									{option.label}
								</label>
							{/each}
						</div>
					</div>
				</div>
			</Card>

			<!-- Volby -->
			<Card>
				<h3 class="text-sm font-semibold text-primary mb-3">Volby</h3>
				<div class="space-y-3">
					<label class="flex items-center gap-3 cursor-pointer">
						<input type="checkbox" bind:checked={showQR} class="rounded text-accent" />
						<div>
							<span class="text-sm font-medium text-primary">QR platební kód</span>
							<p class="text-xs text-muted">Zobrazit QR kód pro rychlou platbu</p>
						</div>
					</label>
					<label class="flex items-center gap-3 cursor-pointer">
						<input type="checkbox" bind:checked={showBankDetails} class="rounded text-accent" />
						<div>
							<span class="text-sm font-medium text-primary">Bankovní údaje</span>
							<p class="text-xs text-muted">Zobrazit číslo účtu, IBAN a SWIFT</p>
						</div>
					</label>
				</div>
			</Card>

			<!-- Akce -->
			<div class="flex items-center justify-between">
				<button
					type="button"
					onclick={openPreview}
					class="rounded-md border border-border bg-surface px-4 py-2 text-sm font-medium text-secondary hover:bg-hover transition-colors"
				>
					Náhled PDF
				</button>
				<Button type="submit" variant="primary" disabled={saving}>
					{saving ? 'Ukládám...' : 'Uložit nastavení'}
				</Button>
			</div>
		</form>
	{/if}
</div>
