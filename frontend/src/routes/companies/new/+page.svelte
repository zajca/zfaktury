<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import {
		companiesApi,
		contactsApi,
		NoCompanyError,
		type NewCompany
	} from '$lib/api/client';
	import { currentCompany } from '$lib/stores/currentCompany.svelte';
	import { toastSuccess, toastError } from '$lib/data/toast-state.svelte';
	import CompanyEditForm from '$lib/components/company/CompanyEditForm.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';

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

	let saving = $state(false);
	let aresLoading = $state(false);
	let error = $state<string | null>(null);

	onMount(async () => {
		// Optional ?ico=... query param pre-fills the form via ARES lookup.
		// Useful when a teammate shares a deep-link to onboard a particular IČO.
		const prefillIco = page.url.searchParams.get('ico');
		if (prefillIco) {
			form.ico = prefillIco;
			await lookupAres();
		}
	});

	async function lookupAres() {
		if (!form.ico) return;
		aresLoading = true;
		try {
			const result = await contactsApi.lookupAres(form.ico);
			form.name = result.name;
			form.legal_name = result.name;
			form.dic = result.dic;
			form.street = result.street;
			form.city = result.city;
			form.zip = result.zip;
		} catch (e) {
			// First-run onboarding: there's no active company yet, so the
			// per-company ARES URL refuses to build. Surface a gentle hint
			// rather than the raw "no active company" error.
			if (e instanceof NoCompanyError) {
				toastError(
					'ARES vyhledávání bude dostupné po vytvoření první firmy. Vyplňte údaje ručně.'
				);
			} else {
				toastError(e instanceof Error ? e.message : 'Nepodařilo se vyhledat v ARES');
			}
		} finally {
			aresLoading = false;
		}
	}

	async function handleSubmit() {
		saving = true;
		error = null;
		try {
			const created = await companiesApi.create(form);
			// Refresh the company list so the switcher reflects the addition.
			const list = await companiesApi.list();
			currentCompany.setCompanies(list);
			// Activate the newly-created company and route home. This makes the
			// first-run onboarding flow seamless: create -> immediate dashboard.
			currentCompany.select(created.id);
			toastSuccess('Firma vytvořena');
			await goto('/');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se vytvořit firmu';
			toastError(error);
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Nová firma - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<PageHeader title="Nová firma" backHref="/companies" backLabel="Zpět na firmy" />

	<ErrorAlert {error} class="mt-4" />

	<div class="mt-6">
		<CompanyEditForm
			bind:form
			{saving}
			{aresLoading}
			onaresLookup={lookupAres}
			onsubmit={handleSubmit}
			cancelHref="/companies"
			saveLabel="Vytvořit firmu"
		/>
	</div>
</div>
