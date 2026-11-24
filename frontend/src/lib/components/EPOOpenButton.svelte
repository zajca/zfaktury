<script lang="ts">
	import { onMount } from 'svelte';
	import type { EPOKind } from '$lib/api/client';
	import type { EPOOpenState } from '$lib/utils/epo-open.svelte';
	import Button from '$lib/ui/Button.svelte';

	interface Props {
		/** Shared with the `EPOOpenPanel` that renders the outcome. */
		epo: EPOOpenState;
		kind: EPOKind;
		id: number;
		hasXml: boolean;
		/** Disables the button while another action of the page is running. */
		disabled?: boolean;
	}

	let { epo, kind, id, hasXml, disabled = false }: Props = $props();

	onMount(() => {
		void epo.init();
	});
</script>

<Button
	variant="secondary"
	onclick={() => epo.open(kind, id)}
	disabled={!hasXml || disabled || epo.loading || !epo.desktopChecked}
	title={hasXml ? undefined : 'Nejdřív vygenerujte XML'}
>
	{epo.loading ? 'Otevírám...' : 'Otevřít v EPO'}
</Button>
