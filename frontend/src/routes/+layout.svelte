<script lang="ts">
	import '../app.css';
	import Layout from '$lib/components/Layout.svelte';
	import HelpDrawer from '$lib/ui/HelpDrawer.svelte';

	let { children } = $props();
</script>

<Layout>
	{@render children()}
</Layout>
<HelpDrawer />
