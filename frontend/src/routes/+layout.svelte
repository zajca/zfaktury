<script lang="ts">
	import { onMount } from 'svelte';
	import '../app.css';
	import { isDesktopMode } from '$lib/utils/download';
	import Layout from '$lib/components/Layout.svelte';
	import HelpDrawer from '$lib/ui/HelpDrawer.svelte';
	import ToastContainer from '$lib/ui/ToastContainer.svelte';

	let { children } = $props();

	onMount(() => {
		isDesktopMode();
	});
</script>

<Layout>
	{@render children()}
</Layout>
<HelpDrawer />
<ToastContainer />
