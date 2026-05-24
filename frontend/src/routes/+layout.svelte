<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import '../app.css';
	import { isDesktopMode } from '$lib/utils/download';
	import { companiesApi } from '$lib/api/client';
	import { currentCompany } from '$lib/stores/currentCompany.svelte';
	import Layout from '$lib/components/Layout.svelte';
	import HelpDrawer from '$lib/ui/HelpDrawer.svelte';
	import ToastContainer from '$lib/ui/ToastContainer.svelte';

	let { children } = $props();
	let booted = $state(false);

	onMount(async () => {
		// Detect desktop (Wails) runtime — fire-and-forget; not in the boot critical path.
		isDesktopMode();

		try {
			const list = await companiesApi.list();
			currentCompany.setCompanies(list);

			if (list.length === 0) {
				// First-run onboarding: no companies yet, send the user to the
				// create form (unless they are already there).
				// NOTE: cast pathname to string — the /companies/* routes are
				// added in a follow-up task and aren't yet in SvelteKit's typed
				// pathname union.
				if ((page.url.pathname as string) !== '/companies/new') {
					await goto('/companies/new');
				}
				booted = true;
				return;
			}

			// Pick the previously-active company if it still exists, otherwise
			// fall back to the first in the list.
			const restored = currentCompany.restoreSelection();
			const targetId =
				restored !== null && list.find((c) => c.id === restored) ? restored : list[0].id;
			currentCompany.select(targetId);
			booted = true;
		} catch (e) {
			// We don't want to block the UI forever if the bootstrap call fails
			// (e.g. server down, transient network error). Render the chrome anyway —
			// downstream components will surface their own errors.
			console.error('Failed to bootstrap companies:', e);
			booted = true;
		}
	});
</script>

{#if !booted}
	<div role="status" class="flex h-screen items-center justify-center bg-base">
		<span class="sr-only">Načítání…</span>
		<div
			class="h-6 w-6 animate-spin rounded-full border-2 border-gray-300 border-t-blue-600"
		></div>
	</div>
{:else}
	<Layout>
		{@render children()}
	</Layout>
	<HelpDrawer />
	<ToastContainer />
{/if}
