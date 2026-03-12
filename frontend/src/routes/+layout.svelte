<script lang="ts">
	import '../app.css';
	import Layout from '$lib/components/Layout.svelte';
	import HelpDrawer from '$lib/ui/HelpDrawer.svelte';
	import ToastContainer from '$lib/ui/ToastContainer.svelte';

	let { children } = $props();
</script>

<Layout>
	{@render children()}
</Layout>
<HelpDrawer />
<ToastContainer />
