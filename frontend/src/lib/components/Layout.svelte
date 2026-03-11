<script lang="ts">
	import { page } from '$app/state';

	interface Props {
		children: import('svelte').Snippet;
	}

	let { children }: Props = $props();

	let sidebarOpen = $state(false);

	type NavItem = { href: string; label: string; icon: string };
	type NavGroup = { section: string; items: NavItem[] };
	type NavEntry = NavItem | NavGroup;

	const navEntries: NavEntry[] = [
		{
			href: '/',
			label: 'Dashboard',
			icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-4 0a1 1 0 01-1-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 01-1 1'
		},
		{
			href: '/invoices',
			label: 'Faktury',
			icon: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z'
		},
		{
			href: '/expenses',
			label: 'Naklady',
			icon: 'M17 9V7a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2m2 4h10a2 2 0 002-2v-6a2 2 0 00-2-2H9a2 2 0 00-2 2v6a2 2 0 002 2zm7-5a2 2 0 11-4 0 2 2 0 014 0z'
		},
		{
			section: 'Ucetnictvi',
			items: [
				{
					href: '/vat',
					label: 'DPH',
					icon: 'M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z'
				}
			]
		},
		{
			href: '/contacts',
			label: 'Kontakty',
			icon: 'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z'
		},
		{
			href: '/settings',
			label: 'Nastaveni',
			icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z'
		}
	];

	function isGroup(entry: NavEntry): entry is NavGroup {
		return 'section' in entry;
	}

	function isActive(href: string): boolean {
		if (href === '/') return page.url.pathname === '/';
		return page.url.pathname.startsWith(href);
	}
</script>

<div class="flex h-screen bg-gray-50">
	<!-- Mobile sidebar overlay -->
	{#if sidebarOpen}
		<div
			class="fixed inset-0 z-30 bg-black/50 lg:hidden"
			role="presentation"
			onclick={() => (sidebarOpen = false)}
			onkeydown={(e) => e.key === 'Escape' && (sidebarOpen = false)}
		></div>
	{/if}

	<!-- Sidebar -->
	<aside
		class="fixed inset-y-0 left-0 z-40 flex w-64 flex-col border-r border-gray-200 bg-white transition-transform duration-200 lg:static lg:translate-x-0
		{sidebarOpen ? 'translate-x-0' : '-translate-x-full'}"
	>
		<!-- Logo -->
		<div class="flex h-16 items-center border-b border-gray-200 px-6">
			<h1 class="text-xl font-bold text-gray-900">ZFaktury</h1>
		</div>

		<!-- Navigation -->
		<nav class="flex-1 space-y-1 px-3 py-4">
			{#each navEntries as entry, i (i)}
				{#if isGroup(entry)}
					<div class="mt-4 mb-1">
						<span class="px-3 text-xs font-semibold uppercase tracking-wider text-gray-400"
							>{entry.section}</span
						>
					</div>
					{#each entry.items as item (item.href)}
						<a
							href={item.href}
							onclick={() => (sidebarOpen = false)}
							class="flex items-center gap-3 rounded-lg py-2.5 pl-5 pr-3 text-sm font-medium transition-colors
							{isActive(item.href)
								? 'bg-blue-50 text-blue-700'
								: 'text-gray-700 hover:bg-gray-100 hover:text-gray-900'}"
						>
							<svg
								class="h-5 w-5 shrink-0"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="1.5"
							>
								<path stroke-linecap="round" stroke-linejoin="round" d={item.icon} />
							</svg>
							{item.label}
						</a>
					{/each}
				{:else}
					<a
						href={entry.href}
						onclick={() => (sidebarOpen = false)}
						class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors
						{isActive(entry.href)
							? 'bg-blue-50 text-blue-700'
							: 'text-gray-700 hover:bg-gray-100 hover:text-gray-900'}"
					>
						<svg
							class="h-5 w-5 shrink-0"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="1.5"
						>
							<path stroke-linecap="round" stroke-linejoin="round" d={entry.icon} />
						</svg>
						{entry.label}
					</a>
				{/if}
			{/each}
		</nav>

		<!-- Footer -->
		<div class="border-t border-gray-200 p-4">
			<p class="text-xs text-gray-500">ZFaktury v0.1.0</p>
		</div>
	</aside>

	<!-- Main content area -->
	<div class="flex flex-1 flex-col overflow-hidden">
		<!-- Top bar (mobile) -->
		<header class="flex h-16 items-center border-b border-gray-200 bg-white px-4 lg:hidden">
			<button
				onclick={() => (sidebarOpen = !sidebarOpen)}
				class="rounded-lg p-2 text-gray-600 hover:bg-gray-100"
				aria-label="Toggle menu"
			>
				<svg
					class="h-6 w-6"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="1.5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5"
					/>
				</svg>
			</button>
			<h1 class="ml-3 text-lg font-semibold text-gray-900">ZFaktury</h1>
		</header>

		<!-- Page content -->
		<main class="flex-1 overflow-y-auto p-6">
			{@render children()}
		</main>
	</div>
</div>
