<script lang="ts">
	import { contactsApi, type Contact } from '$lib/api/client';

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
			const res = await contactsApi.list({ page, per_page: perPage, search: search || undefined });
			contacts = res.data;
			total = res.total;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load contacts';
		} finally {
			loading = false;
		}
	}

	function handleSearch(value: string) {
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

<div>
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-gray-900">Kontakty</h1>
			<p class="mt-1 text-sm text-gray-500">Sprava zakazniku a dodavatelu</p>
		</div>
		<a
			href="/contacts/new"
			class="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 transition-colors"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
			</svg>
			Pridat kontakt
		</a>
	</div>

	<!-- Search -->
	<div class="mt-6">
		<input
			type="text"
			bind:value={search}
			placeholder="Hledat podle nazvu, ICO, emailu..."
			class="w-full max-w-md rounded-lg border border-gray-300 px-4 py-2.5 text-sm shadow-sm placeholder:text-gray-400 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
		/>
	</div>

	<!-- Error -->
	{#if error}
		<div class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
			{error}
		</div>
	{/if}

	<!-- Table -->
	<div class="mt-4 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
		{#if loading}
			<div class="flex items-center justify-center p-12">
				<div class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"></div>
			</div>
		{:else if contacts.length === 0}
			<div class="p-12 text-center text-gray-400">
				{search ? 'Zadne kontakty neodpovidaji hledani.' : 'Zatim zadne kontakty.'}
			</div>
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-gray-200 bg-gray-50">
					<tr>
						<th class="px-4 py-3 font-medium text-gray-600">Nazev</th>
						<th class="px-4 py-3 font-medium text-gray-600">ICO</th>
						<th class="px-4 py-3 font-medium text-gray-600">DIC</th>
						<th class="hidden px-4 py-3 font-medium text-gray-600 md:table-cell">Mesto</th>
						<th class="hidden px-4 py-3 font-medium text-gray-600 lg:table-cell">Email</th>
						<th class="hidden px-4 py-3 font-medium text-gray-600 lg:table-cell">Telefon</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-100">
					{#each contacts as contact}
						<tr class="hover:bg-gray-50 transition-colors">
							<td class="px-4 py-3">
								<a href="/contacts/{contact.id}" class="font-medium text-blue-600 hover:text-blue-800">
									{contact.name}
								</a>
							</td>
							<td class="px-4 py-3 text-gray-600">{contact.ico || '-'}</td>
							<td class="px-4 py-3 text-gray-600">{contact.dic || '-'}</td>
							<td class="hidden px-4 py-3 text-gray-600 md:table-cell">{contact.city || '-'}</td>
							<td class="hidden px-4 py-3 text-gray-600 lg:table-cell">{contact.email || '-'}</td>
							<td class="hidden px-4 py-3 text-gray-600 lg:table-cell">{contact.phone || '-'}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>

	<!-- Pagination -->
	{#if totalPages > 1}
		<div class="mt-4 flex items-center justify-between">
			<p class="text-sm text-gray-500">
				Celkem {total} kontaktu
			</p>
			<div class="flex gap-2">
				<button
					onclick={() => { page = Math.max(1, page - 1); loadContacts(); }}
					disabled={page <= 1}
					class="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					Predchozi
				</button>
				<span class="flex items-center px-3 text-sm text-gray-600">
					{page} / {totalPages}
				</span>
				<button
					onclick={() => { page = Math.min(totalPages, page + 1); loadContacts(); }}
					disabled={page >= totalPages}
					class="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					Dalsi
				</button>
			</div>
		</div>
	{/if}
</div>
