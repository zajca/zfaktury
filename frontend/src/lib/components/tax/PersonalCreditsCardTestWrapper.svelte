<script lang="ts">
	import PersonalCreditsCard from './PersonalCreditsCard.svelte';

	let isStudent = $state(false);
	let studentMonths = $state(12);
	let disabilityLevel = $state(1);
	let disabilityMonths = $state(12);
	let ztpp = $state(true);
	let ztppMonths = $state(12);
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
