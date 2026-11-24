<script lang="ts">
	import { lastDayOfPreviousMonth } from '$lib/utils/date';

	let { onpick }: { onpick: (isoDate: string) => void } = $props();
</script>

<button
	type="button"
	onclick={() => onpick(lastDayOfPreviousMonth(new Date()))}
	class="rounded-lg border border-border bg-surface px-3 py-1.5 text-sm font-medium text-secondary hover:bg-hover hover:text-primary focus:ring-1 focus:ring-accent/50 focus:outline-none"
>
	Poslední den minulého měsíce
</button>
