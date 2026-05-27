<script lang="ts">
	import { onMount } from 'svelte';
	import {
		companiesApi,
		settingsApi,
		type Settings,
		type NewCompany
	} from '$lib/api/client';
	import {
		currentCompany,
		notifyIfSwitchedCompany,
		onCompanyChange
	} from '$lib/stores/currentCompany.svelte';
	import CompanyEditForm from '$lib/components/company/CompanyEditForm.svelte';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';
	import { toastSuccess, toastError } from '$lib/data/toast-state.svelte';

	// Identity / address / contact / bank — persisted on the active companies row.
	let form = $state<NewCompany>({
		name: '',
		legal_name: '',
		ico: '',
		dic: '',
		vat_registered: false,
		street: '',
		house_number: '',
		city: '',
		zip: '',
		email: '',
		phone: '',
		first_name: '',
		last_name: '',
		bank_account: '',
		bank_code: '',
		iban: '',
		swift: ''
	});

	// Tax-office codes still live in the per-company settings KV.
	const TAX_CODE_KEYS = [
		'c_ufo',
		'c_pracufo',
		'c_okec',
		'financni_urad_code',
		'cssz_code',
		'health_insurance_code'
	] as const;
	let taxCodes = $state<Settings>({});

	let loading = $state(true);
	let saving = $state(false);
	let error = $state<string | null>(null);

	onMount(() => {
		load();
	});

	onCompanyChange(() => load());

	async function load() {
		loading = true;
		error = null;
		try {
			const id = currentCompany.current?.id;
			if (!id) {
				error = 'Nejdříve vyberte firmu.';
				return;
			}
			const [company, allSettings] = await Promise.all([
				companiesApi.getById(id),
				settingsApi.getAll()
			]);
			form = {
				name: company.name,
				legal_name: company.legal_name,
				ico: company.ico,
				dic: company.dic ?? '',
				vat_registered: company.vat_registered,
				street: company.street ?? '',
				house_number: company.house_number ?? '',
				city: company.city ?? '',
				zip: company.zip ?? '',
				email: company.email ?? '',
				phone: company.phone ?? '',
				first_name: company.first_name ?? '',
				last_name: company.last_name ?? '',
				bank_account: company.bank_account ?? '',
				bank_code: company.bank_code ?? '',
				iban: company.iban ?? '',
				swift: company.swift ?? '',
				logo_path: company.logo_path,
				accent_color: company.accent_color
			};
			taxCodes = Object.fromEntries(
				TAX_CODE_KEYS.map((k) => [k, allSettings[k] ?? ''])
			);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst firmu';
		} finally {
			loading = false;
		}
	}

	async function handleSave() {
		const id = currentCompany.current?.id;
		if (!id) return;
		saving = true;
		try {
			await companiesApi.update(id, form);
			const result = await settingsApi.update(taxCodes);
			if (notifyIfSwitchedCompany(result.submittedFor, result.respondedFor)) {
				return;
			}
			// Refresh the company list so the switcher picks up any rename.
			const list = await companiesApi.list();
			currentCompany.setCompanies(list);
			toastSuccess('Údaje firmy uloženy');
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se uložit údaje firmy');
		} finally {
			saving = false;
		}
	}

	function taxField(key: (typeof TAX_CODE_KEYS)[number]): string {
		return taxCodes[key] ?? '';
	}

	function setTaxField(key: (typeof TAX_CODE_KEYS)[number], value: string) {
		taxCodes[key] = value;
	}
</script>

<svelte:head>
	<title>Údaje firmy - Nastavení - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<PageHeader
		title="Údaje firmy"
		description={currentCompany.current
			? `Identifikační a kontaktní údaje firmy ${currentCompany.current.name}, které se zobrazují na fakturách.`
			: 'Údaje aktivní firmy.'}
	/>

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
			<!-- Identity / address / contact / bank — persisted to the companies row. -->
			<CompanyEditForm bind:form {saving} hideActions />

			<!-- Tax-office codes — persisted to the per-company settings KV. -->
			<Card>
				<h2 class="text-base font-semibold text-primary">Kódy úřadů</h2>
				<p class="mt-1 text-sm text-tertiary">
					Kódy pro daňové přiznání DPH, ročního přiznání k dani z příjmů a přehledů OSVČ.
				</p>
				<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div>
						<label for="c_ufo" class="block text-sm font-medium text-secondary"
							>Kód FÚ pro EPO (3-místný)</label
						>
						<input
							id="c_ufo"
							type="text"
							value={taxField('c_ufo')}
							oninput={(e) => setTaxField('c_ufo', e.currentTarget.value)}
							placeholder="např. 464"
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						/>
					</div>
					<div>
						<label for="c_pracufo" class="block text-sm font-medium text-secondary"
							>Kód pracoviště FÚ (4-místný)</label
						>
						<input
							id="c_pracufo"
							type="text"
							value={taxField('c_pracufo')}
							oninput={(e) => setTaxField('c_pracufo', e.currentTarget.value)}
							placeholder="např. 3305"
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						/>
					</div>
					<div>
						<label for="c_okec" class="block text-sm font-medium text-secondary"
							>NACE kód činnosti</label
						>
						<input
							id="c_okec"
							type="text"
							value={taxField('c_okec')}
							oninput={(e) => setTaxField('c_okec', e.currentTarget.value)}
							placeholder="např. 582900"
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						/>
					</div>
					<div>
						<label for="financni_urad_code" class="block text-sm font-medium text-secondary"
							>Kód finančního úřadu</label
						>
						<input
							id="financni_urad_code"
							type="text"
							value={taxField('financni_urad_code')}
							oninput={(e) => setTaxField('financni_urad_code', e.currentTarget.value)}
							placeholder="např. 0451"
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						/>
					</div>
					<div>
						<label for="cssz_code" class="block text-sm font-medium text-secondary"
							>Kód OSSZ</label
						>
						<input
							id="cssz_code"
							type="text"
							value={taxField('cssz_code')}
							oninput={(e) => setTaxField('cssz_code', e.currentTarget.value)}
							placeholder="např. prehledosvc"
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						/>
					</div>
					<div>
						<label for="health_insurance_code" class="block text-sm font-medium text-secondary"
							>Kód zdravotní pojišťovny</label
						>
						<input
							id="health_insurance_code"
							type="text"
							value={taxField('health_insurance_code')}
							oninput={(e) => setTaxField('health_insurance_code', e.currentTarget.value)}
							placeholder="např. 111 (VZP)"
							class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
						/>
					</div>
				</div>
			</Card>

			<FormActions {saving} cancelHref="/settings" saveLabel="Uložit změny" />
		</form>
	{/if}
</div>
