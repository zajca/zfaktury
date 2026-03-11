<script lang="ts">
	import { contactsApi, type Contact } from '$lib/api/client';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';

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
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-xl font-semibold text-primary">Kontakty</h1>
			<p class="mt-1 text-sm text-tertiary">Správa zákazníků a dodavatelů</p>
		</div>
		<Button variant="primary" href="/contacts/new">
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
			</svg>
			Přidat kontakt
		</Button>
	</div>

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
	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-danger/20 bg-danger-bg p-4 text-sm text-danger"
		>
			{error}
		</div>
	{/if}

	<!-- Table -->
	<Card padding={false} class="mt-4 overflow-hidden">
		{#if loading}
			<div class="flex items-center justify-center p-12">
				<div role="status">
					<div
						class="h-8 w-8 animate-spin rounded-full border-4 border-border border-t-accent"
					></div>
					<span class="sr-only">Nacitani...</span>
				</div>
			</div>
		{:else if contacts.length === 0}
			<div class="p-12 text-center text-muted">
				{search ? 'Žádné kontakty neodpovídají hledání.' : 'Zatím žádné kontakty.'}
			</div>
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-border bg-elevated">
					<tr>
						<th class="px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">Název</th>
						<th class="px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">IČO</th>
						<th class="px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">DIČ</th>
						<th class="hidden px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted md:table-cell">Město</th>
						<th class="hidden px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted lg:table-cell">Email</th>
						<th class="hidden px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted lg:table-cell">Telefon</th>
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
	{#if totalPages > 1}
		<div class="mt-4 flex items-center justify-between">
			<p class="text-sm text-tertiary">
				Celkem {total} kontaktů
			</p>
			<div class="flex gap-2">
				<Button
					variant="secondary"
					size="sm"
					onclick={() => {
						page = Math.max(1, page - 1);
						loadContacts();
					}}
					disabled={page <= 1}
				>
					Předchozí
				</Button>
				<span class="flex items-center px-3 text-sm text-secondary">
					{page} / {totalPages}
				</span>
				<Button
					variant="secondary"
					size="sm"
					onclick={() => {
						page = Math.min(totalPages, page + 1);
						loadContacts();
					}}
					disabled={page >= totalPages}
				>
					Další
				</Button>
			</div>
		</div>
	{/if}
</div>
