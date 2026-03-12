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
			toastSuccess('Email nastavení uloženo');
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
	<title>Email - Nastavení - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<PageHeader title="Email" description="Výchozí nastavení pro odesílání faktur emailem" />

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
			<Card>
				<h2 class="text-base font-semibold text-primary">Šablony emailů <HelpTip topic="email-sablony" /></h2>
				<p class="mt-1 text-sm text-tertiary">Výchozí nastavení pro odesílání faktur emailem.</p>
				<div class="mt-4 space-y-4">
					<div class="flex gap-6">
						<div class="flex items-center gap-3">
							<input
								id="email_attach_pdf"
								type="checkbox"
								checked={field('email_attach_pdf') !== 'false'}
								onchange={(e) =>
									setField(
										'email_attach_pdf',
										(e.target as HTMLInputElement).checked ? 'true' : 'false'
									)}
								class="h-4 w-4 rounded border-border accent-accent"
							/>
							<label for="email_attach_pdf" class="text-sm font-medium text-secondary">Přikládat PDF</label>
						</div>
						<div class="flex items-center gap-3">
							<input
								id="email_attach_isdoc"
								type="checkbox"
								checked={field('email_attach_isdoc') === 'true'}
								onchange={(e) =>
									setField(
										'email_attach_isdoc',
										(e.target as HTMLInputElement).checked ? 'true' : 'false'
									)}
								class="h-4 w-4 rounded border-border accent-accent"
							/>
							<label for="email_attach_isdoc" class="text-sm font-medium text-secondary">Přikládat ISDOC</label>
						</div>
					</div>
					<div>
						<label for="email_subject_template" class="block text-sm font-medium text-secondary">
							Šablona předmětu
						</label>
						<input
							id="email_subject_template"
							type="text"
							value={field('email_subject_template') || 'Faktura {invoice_number}'}
							oninput={(e) => setField('email_subject_template', (e.target as HTMLInputElement).value)}
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						/>
						<p class="mt-1 text-xs text-muted">Použijte <code class="text-accent">{'{invoice_number}'}</code> pro číslo faktury.</p>
					</div>
					<div>
						<label for="email_body_template" class="block text-sm font-medium text-secondary">
							Šablona textu emailu
						</label>
						<textarea
							id="email_body_template"
							rows="4"
							value={field('email_body_template') || 'Dobrý den,\n\nv příloze zasíláme fakturu {invoice_number}.\n\nS pozdravem'}
							oninput={(e) => setField('email_body_template', (e.target as HTMLTextAreaElement).value)}
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none resize-y"
						></textarea>
						<p class="mt-1 text-xs text-muted">Použijte <code class="text-accent">{'{invoice_number}'}</code> pro číslo faktury.</p>
					</div>
				</div>
			</Card>

			<!-- Save -->
			<FormActions {saving} saveLabel="Uložit nastavení" class="pb-8" />
		</form>
	{/if}
</div>
