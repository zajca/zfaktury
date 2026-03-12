<script lang="ts">
	import { page } from '$app/state';
	import { browser } from '$app/environment';

	interface Props {
		children: import('svelte').Snippet;
	}

	let { children }: Props = $props();

	let sidebarOpen = $state(false);
	let sidebarCollapsed = $state(
		browser ? localStorage.getItem('zf-sidebar') === 'true' : false
	);
	let sidebarHovered = $state(false);
	let sidebarExpanded = $derived(!sidebarCollapsed || sidebarHovered);

	function toggleCollapse() {
		sidebarCollapsed = !sidebarCollapsed;
		if (browser) {
			localStorage.setItem('zf-sidebar', String(sidebarCollapsed));
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.ctrlKey && e.shiftKey && e.key === 'L') {
			e.preventDefault();
			toggleCollapse();
		}
		if (e.key === 'Escape' && sidebarOpen) {
			sidebarOpen = false;
		}
	}

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
			href: '/recurring',
			label: 'Sablony faktur',
			icon: 'M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15'
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
				},
				{
					href: '/tax',
					label: 'Dan z prijmu',
					icon: 'M9 14l6-6m-5.5.5h.01m4.99 5h.01M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16l3.5-2 3.5 2 3.5-2 3.5 2z'
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

<svelte:window onkeydown={handleKeydown} />

<div class="flex h-screen bg-base">
	<!-- Mobile sidebar overlay -->
	{#if sidebarOpen}
		<div
			class="fixed inset-0 z-30 bg-black/60 lg:hidden"
			role="presentation"
			onclick={() => (sidebarOpen = false)}
		></div>
	{/if}

	<!-- Desktop Sidebar -->
	<aside
		class="hidden lg:flex flex-col border-r border-border bg-surface transition-[width] duration-200 ease-out shrink-0
		{sidebarCollapsed ? (sidebarHovered ? 'absolute inset-y-0 left-0 z-50 w-52 shadow-xl shadow-black/30' : 'w-12') : 'w-52'}"
		onmouseenter={() => sidebarCollapsed && (sidebarHovered = true)}
		onmouseleave={() => (sidebarHovered = false)}
	>
		<!-- Logo + Toggle -->
		<div class="flex h-12 items-center border-b border-border overflow-hidden
			{sidebarExpanded ? 'px-3 gap-2 justify-between' : 'justify-center'}">
			{#if sidebarExpanded}
				<span class="text-sm font-semibold text-primary truncate">ZFaktury</span>
				<button
					onclick={toggleCollapse}
					class="shrink-0 rounded p-1 text-tertiary hover:text-primary transition-colors"
					title="Toggle sidebar (Ctrl+Shift+L)"
				>
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
						{#if sidebarCollapsed}
							<path stroke-linecap="round" stroke-linejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
						{:else}
							<path stroke-linecap="round" stroke-linejoin="round" d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18" />
						{/if}
					</svg>
				</button>
			{:else}
				<button
					onclick={toggleCollapse}
					class="rounded p-1 text-tertiary hover:text-primary transition-colors"
					title="Toggle sidebar (Ctrl+Shift+L)"
				>
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
						<path stroke-linecap="round" stroke-linejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
					</svg>
				</button>
			{/if}
		</div>

		<!-- Navigation -->
		<nav class="flex-1 overflow-y-auto px-2 py-3 space-y-0.5">
			{#each navEntries as entry, i (i)}
				{#if isGroup(entry)}
					{#if sidebarExpanded}
						<div class="mt-4 mb-1.5 px-2">
							<span class="text-[11px] font-medium uppercase tracking-wider text-muted">{entry.section}</span>
						</div>
					{:else}
						<div class="mt-3 mb-1 border-t border-border-subtle mx-1"></div>
					{/if}
					{#each entry.items as item (item.href)}
						<a
							href={item.href}
							class="flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm font-medium transition-colors
							{isActive(item.href)
								? 'bg-accent-muted text-accent-text'
								: 'text-secondary hover:text-primary hover:bg-hover'}"
							title={sidebarExpanded ? undefined : item.label}
						>
							<svg class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
								<path stroke-linecap="round" stroke-linejoin="round" d={item.icon} />
							</svg>
							{#if sidebarExpanded}
								<span class="truncate">{item.label}</span>
							{/if}
						</a>
					{/each}
				{:else}
					<a
						href={entry.href}
						class="flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm font-medium transition-colors
						{isActive(entry.href)
							? 'bg-accent-muted text-accent-text'
							: 'text-secondary hover:text-primary hover:bg-hover'}"
						title={sidebarExpanded ? undefined : entry.label}
					>
						<svg class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
							<path stroke-linecap="round" stroke-linejoin="round" d={entry.icon} />
						</svg>
						{#if sidebarExpanded}
							<span class="truncate">{entry.label}</span>
						{/if}
					</a>
				{/if}
			{/each}
		</nav>

		<!-- Footer -->
		{#if sidebarExpanded}
			<div class="border-t border-border px-3 py-2.5">
				<p class="text-[11px] text-muted">ZFaktury v0.1.0</p>
			</div>
		{/if}
	</aside>

	<!-- Mobile Sidebar -->
	<aside
		class="fixed inset-y-0 left-0 z-40 flex w-64 flex-col border-r border-border bg-surface transition-transform duration-200 lg:hidden
		{sidebarOpen ? 'translate-x-0' : '-translate-x-full'}"
	>
		<div class="flex h-12 items-center border-b border-border px-4">
			<h1 class="text-sm font-semibold text-primary">ZFaktury</h1>
		</div>
		<nav class="flex-1 overflow-y-auto px-2 py-3 space-y-0.5">
			{#each navEntries as entry, i (i)}
				{#if isGroup(entry)}
					<div class="mt-4 mb-1.5 px-2">
						<span class="text-[11px] font-medium uppercase tracking-wider text-muted">{entry.section}</span>
					</div>
					{#each entry.items as item (item.href)}
						<a
							href={item.href}
							onclick={() => (sidebarOpen = false)}
							class="flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm font-medium transition-colors
							{isActive(item.href)
								? 'bg-accent-muted text-accent-text'
								: 'text-secondary hover:text-primary hover:bg-hover'}"
						>
							<svg class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
								<path stroke-linecap="round" stroke-linejoin="round" d={item.icon} />
							</svg>
							{item.label}
						</a>
					{/each}
				{:else}
					<a
						href={entry.href}
						onclick={() => (sidebarOpen = false)}
						class="flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm font-medium transition-colors
						{isActive(entry.href)
							? 'bg-accent-muted text-accent-text'
							: 'text-secondary hover:text-primary hover:bg-hover'}"
					>
						<svg class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
							<path stroke-linecap="round" stroke-linejoin="round" d={entry.icon} />
						</svg>
						{entry.label}
					</a>
				{/if}
			{/each}
		</nav>
		<div class="border-t border-border px-3 py-2.5">
			<p class="text-[11px] text-muted">ZFaktury v0.1.0</p>
		</div>
	</aside>

	<!-- Main content area -->
	<div class="flex flex-1 flex-col overflow-hidden">
		<!-- Top bar (mobile) -->
		<header class="flex h-12 items-center border-b border-border bg-surface px-4 lg:hidden">
			<button
				onclick={() => (sidebarOpen = !sidebarOpen)}
				class="rounded-md p-1.5 text-secondary hover:bg-hover hover:text-primary transition-colors"
				aria-label="Toggle menu"
			>
				<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
					<path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" />
				</svg>
			</button>
			<h1 class="ml-3 text-sm font-semibold text-primary">ZFaktury</h1>
		</header>

		<!-- Page content -->
		<main class="flex-1 overflow-y-auto p-5">
			{@render children()}
		</main>
	</div>
</div>
