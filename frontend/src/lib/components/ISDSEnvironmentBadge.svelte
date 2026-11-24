<script lang="ts">
	import type { ISDSEnvironment } from '$lib/api/client';
	import { environmentLabels, environmentVariant } from '$lib/utils/filing';
	import Badge from '$lib/ui/Badge.svelte';

	interface Props {
		environment: ISDSEnvironment;
		/** Larger badge for the send dialog, where a wrong environment matters most. */
		prominent?: boolean;
	}

	let { environment, prominent = false }: Props = $props();
</script>

<Badge
	variant={environmentVariant[environment]}
	class={prominent ? 'px-3 py-1 text-sm font-semibold uppercase tracking-wide' : ''}
>
	{environmentLabels[environment]}
</Badge>
