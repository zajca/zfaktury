<script lang="ts">
	import PersonalCreditsCard from './PersonalCreditsCard.svelte';

	let {
		initialLevel = 1,
		initialDisabilityMonths = 12,
		initialZtpp = true,
		initialZtppMonths = 12
	}: {
		initialLevel?: number;
		initialDisabilityMonths?: number;
		initialZtpp?: boolean;
		initialZtppMonths?: number;
	} = $props();

	let isStudent = $state(false);
	let studentMonths = $state(12);
	let disabilityLevel = $state(initialLevel);
	let disabilityMonths = $state(initialDisabilityMonths);
	let ztpp = $state(initialZtpp);
	let ztppMonths = $state(initialZtppMonths);
	let saved = $state(false);
</script>

<PersonalCreditsCard
	bind:isStudent
	bind:studentMonths
	bind:disabilityLevel
	bind:disabilityMonths
	bind:ztpp
	bind:ztppMonths
	summary={null}
	taxConstants={null}
	saving={false}
	onSave={() => (saved = true)}
/>
<p data-testid="disabilityMonths">{disabilityMonths}</p>
<p data-testid="ztppMonths">{ztppMonths}</p>
<p data-testid="saved">{saved}</p>
