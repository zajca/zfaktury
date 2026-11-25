<script lang="ts">
	import SpouseCreditsCard from './SpouseCreditsCard.svelte';

	let spouseName = $state('Jana Novakova');
	let spouseBirthNumber = $state('8555012345');
	let spouseIncome = $state(50000);
	let spouseZtp = $state(false);
	let spouseMonths = $state(12);
	let spouseChildUnder3Months = $state(12);
	let showSpouseForm = $state(true);
	let saved = $state(false);
</script>

<SpouseCreditsCard
	bind:spouseName
	bind:spouseBirthNumber
	bind:spouseIncome
	bind:spouseZtp
	bind:spouseMonths
	bind:spouseChildUnder3Months
	bind:showSpouseForm
	summary={null}
	taxConstants={null}
	saving={false}
	onSave={() => (saved = true)}
	onDelete={() => {}}
/>
<p data-testid="childUnder3Months">{spouseChildUnder3Months}</p>
<p data-testid="saved">{saved}</p>
