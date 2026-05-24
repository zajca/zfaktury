<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { companiesApi, type NewCompany } from '$lib/api/client';
	import { currentCompany } from '$lib/stores/currentCompany.svelte';
	import { toastSuccess, toastError } from '$lib/data/toast-state.svelte';
	import CompanyEditForm from '$lib/components/company/CompanyEditForm.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';

	let companyId = $derived(Number(page.params.id));

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

	let loading = $state(true);
	let saving = $state(false);
	let error = $state<string | null>(null);

	onMount(() => {
		loadCompany();
	});

	async function loadCompany() {
		loading = true;
		error = null;
		try {
			const c = await companiesApi.getById(companyId);
			form = {
				name: c.name,
				legal_name: c.legal_name,
				ico: c.ico,
				dic: c.dic ?? '',
				vat_registered: c.vat_registered,
				street: c.street ?? '',
				house_number: c.house_number ?? '',
				city: c.city ?? '',
				zip: c.zip ?? '',
				email: c.email ?? '',
				phone: c.phone ?? '',
				first_name: c.first_name ?? '',
				last_name: c.last_name ?? '',
				bank_account: c.bank_account ?? '',
				bank_code: c.bank_code ?? '',
				iban: c.iban ?? '',
				swift: c.swift ?? '',
				logo_path: c.logo_path,
				accent_color: c.accent_color
			};
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst firmu';
		} finally {
			loading = false;
		}
	}

	async function handleSubmit() {
		saving = true;
		try {
			await companiesApi.update(companyId, form);
			// Refresh the company list so the switcher picks up any rename /
			// detail change on this company.
			const list = await companiesApi.list();
			currentCompany.setCompanies(list);
			toastSuccess('Firma uložena');
			await goto('/companies');
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se uložit firmu');
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Upravit firmu - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<PageHeader
		title="Upravit firmu"
		description={form.name || undefined}
		backHref="/companies"
		backLabel="Zpět na firmy"
	/>

	<ErrorAlert {error} class="mt-4" />

	{#if loading}
		<LoadingSpinner class="mt-8" />
	{:else}
		<div class="mt-6">
			<CompanyEditForm
				bind:form
				{saving}
				onsubmit={handleSubmit}
				cancelHref="/companies"
				saveLabel="Uložit změny"
			/>
		</div>
	{/if}
</div>
