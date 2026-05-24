<script lang="ts">
	import { onMount } from 'svelte';
	import { companiesApi, type Company } from '$lib/api/client';
	import { currentCompany } from '$lib/stores/currentCompany.svelte';
	import { toastSuccess, toastError } from '$lib/data/toast-state.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import ConfirmDialog from '$lib/ui/ConfirmDialog.svelte';
	import EmptyState from '$lib/ui/EmptyState.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';

	let companies = $state<Company[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let deleteTarget = $state<Company | null>(null);
	let showDeleteConfirm = $state(false);
	let deleting = $state(false);

	onMount(() => {
		loadCompanies();
	});

	async function loadCompanies() {
		loading = true;
		error = null;
		try {
			const list = await companiesApi.list();
			companies = list;
			// Keep the global store in sync — the switcher needs to reflect any
			// add/edit/delete that happened on this page or others.
			currentCompany.setCompanies(list);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst firmy';
		} finally {
			loading = false;
		}
	}

	function askDelete(company: Company) {
		deleteTarget = company;
		showDeleteConfirm = true;
	}

	function cancelDelete() {
		showDeleteConfirm = false;
		deleteTarget = null;
	}

	async function confirmDelete() {
		if (!deleteTarget) return;
		deleting = true;
		try {
			await companiesApi.delete(deleteTarget.id);
			toastSuccess('Firma smazána');
			showDeleteConfirm = false;
			deleteTarget = null;
			await loadCompanies();
		} catch (e) {
			// Server returns 409 for last-company / in-use; the API client
			// converts that into a thrown Error with the server's message.
			toastError(e instanceof Error ? e.message : 'Nepodařilo se smazat firmu');
		} finally {
			deleting = false;
		}
	}
</script>

<svelte:head>
	<title>Firmy - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-6xl">
	<PageHeader title="Firmy" description="Správa firem evidovaných v aplikaci">
		{#snippet actions()}
			<Button variant="primary" href="/companies/new">
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
				</svg>
				Přidat firmu
			</Button>
		{/snippet}
	</PageHeader>

	<ErrorAlert {error} class="mt-4" />

	<Card padding={false} class="mt-4 overflow-hidden">
		{#if loading}
			<LoadingSpinner class="p-12" />
		{:else if companies.length === 0}
			<EmptyState
				message="Zatím žádné firmy."
				actionHref="/companies/new"
				actionLabel="Přidat první firmu"
			/>
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-border bg-elevated">
					<tr>
						<th class="th-default">Název</th>
						<th class="th-default">IČO</th>
						<th class="th-default">DIČ</th>
						<th class="th-default hidden md:table-cell">Město</th>
						<th class="th-default hidden lg:table-cell">Plátce DPH</th>
						<th class="th-default text-right">Akce</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border-subtle">
					{#each companies as company (company.id)}
						<tr class="hover:bg-hover transition-colors">
							<td class="px-4 py-2.5">
								<a
									href="/companies/{company.id}"
									class="font-medium text-accent-text hover:text-accent"
								>
									{company.name}
								</a>
								{#if currentCompany.current?.id === company.id}
									<span
										class="ml-2 rounded-full bg-accent-muted px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-accent-text"
									>
										Aktivní
									</span>
								{/if}
							</td>
							<td class="px-4 py-2.5 text-secondary">{company.ico || '-'}</td>
							<td class="px-4 py-2.5 text-secondary">{company.dic || '-'}</td>
							<td class="hidden px-4 py-2.5 text-secondary md:table-cell">{company.city || '-'}</td>
							<td class="hidden px-4 py-2.5 text-secondary lg:table-cell">
								{company.vat_registered ? 'Ano' : 'Ne'}
							</td>
							<td class="px-4 py-2.5 text-right">
								<Button variant="danger" size="sm" onclick={() => askDelete(company)}>
									Smazat
								</Button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</Card>
</div>

<ConfirmDialog
	bind:open={showDeleteConfirm}
	title="Smazat firmu"
	message={deleteTarget
		? `Opravdu chcete smazat firmu „${deleteTarget.name}“? Tuto akci nelze vrátit zpět.`
		: ''}
	confirmLabel="Smazat"
	loading={deleting}
	onconfirm={confirmDelete}
	oncancel={cancelDelete}
/>
