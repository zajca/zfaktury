<script lang="ts">
	import { contactsApi, type Contact } from '$lib/api/client';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import EmptyState from '$lib/ui/EmptyState.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import Pagination from '$lib/ui/Pagination.svelte';

	let contacts = $state<Contact[]>([]);
	let total = $state(0);
	let page = $state(1);
	let perPage = $state(25);
	let search = $state('');
	let loading = $state(true);
	let error = $state<string | null>(null);

	let searchTimeout: ReturnType<typeof setTimeout>;

	async function loadContacts() {
		loading = true;
		error = null;
		try {
			const res = await contactsApi.list({
				limit: perPage,
				offset: (page - 1) * perPage,
				search: search || undefined
			});
			contacts = res.data;
			total = res.total;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst kontakty';
		} finally {
			loading = false;
		}
	}

	function handleSearch(_value: string) {
		clearTimeout(searchTimeout);
		searchTimeout = setTimeout(() => {
			page = 1;
			loadContacts();
		}, 300);
	}

	$effect(() => {
		loadContacts();
	});

	$effect(() => {
		handleSearch(search);
	});

	let totalPages = $derived(Math.max(1, Math.ceil(total / perPage)));
</script>

<svelte:head>
	<title>Kontakty - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-6xl">
	<PageHeader title="Kontakty" description="Správa zákazníků a dodavatelů">
		{#snippet actions()}
			<Button variant="primary" href="/contacts/new">
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
				</svg>
				Přidat kontakt
			</Button>
		{/snippet}
	</PageHeader>

	<!-- Search -->
	<div class="mt-6">
		<input
			type="text"
			bind:value={search}
			placeholder="Hledat podle názvu, IČO, emailu..."
			class="w-full max-w-md rounded-lg border border-border bg-elevated px-4 py-2.5 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
		/>
	</div>

	<!-- Error -->
	<ErrorAlert {error} class="mt-4" />

	<!-- Table -->
	<Card padding={false} class="mt-4 overflow-hidden">
		{#if loading}
			<LoadingSpinner class="p-12" />
		{:else if contacts.length === 0}
			<EmptyState message="Zatím žádné kontakty." filteredMessage="Žádné kontakty neodpovídají hledání." isFiltered={!!search} actionHref="/contacts/new" actionLabel="Přidat první kontakt" />
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-border bg-elevated">
					<tr>
						<th class="th-default">Název</th>
						<th class="th-default">IČO</th>
						<th class="th-default">DIČ</th>
						<th class="th-default hidden md:table-cell">Město</th>
						<th class="th-default hidden lg:table-cell">Email</th>
						<th class="th-default hidden lg:table-cell">Telefon</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border-subtle">
					{#each contacts as contact (contact.id)}
						<tr class="hover:bg-hover transition-colors cursor-pointer">
							<td class="px-4 py-2.5">
								<a
									href="/contacts/{contact.id}"
									class="font-medium text-accent-text hover:text-accent"
								>
									{contact.name}
								</a>
							</td>
							<td class="px-4 py-2.5 text-secondary">{contact.ico || '-'}</td>
							<td class="px-4 py-2.5 text-secondary">{contact.dic || '-'}</td>
							<td class="hidden px-4 py-2.5 text-secondary md:table-cell">{contact.city || '-'}</td>
							<td class="hidden px-4 py-2.5 text-secondary lg:table-cell">{contact.email || '-'}</td>
							<td class="hidden px-4 py-2.5 text-secondary lg:table-cell">{contact.phone || '-'}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</Card>

	<!-- Pagination -->
	<Pagination {page} {totalPages} {total} label="kontaktů" onPageChange={(p) => { page = p; loadContacts(); }} />
</div>
