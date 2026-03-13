<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { healthInsuranceApi } from '$lib/api/client';
	import { toastError } from '$lib/data/toast-state.svelte';
	import Card from '$lib/ui/Card.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';

	const filingTypes = [
		{ value: 'regular', label: 'Řádné' },
		{ value: 'corrective', label: 'Následné' },
		{ value: 'supplementary', label: 'Opravné' }
	];

	const paramYear = page.url.searchParams.get('year');

	let saving = $state(false);

	let form = $state({
		year: paramYear ? Number(paramYear) : new Date().getFullYear() - 1,
		filing_type: 'regular'
	});

	async function handleSubmit() {
		saving = true;
		try {
			const result = await healthInsuranceApi.create({
				year: form.year,
				filing_type: form.filing_type
			});
			goto(`/tax/health/${result.id}`);
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se vytvořit přehled pro ZP');
		} finally {
			saving = false;
		}
	}

	const inputClass =
		'mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none';
</script>

<svelte:head>
	<title>Nový přehled OSVČ pro ZP - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-2xl">
	<PageHeader title="Nový přehled OSVČ pro ZP" backHref="/tax" backLabel="Zpět na daně" />

	<form
		onsubmit={(e) => {
			e.preventDefault();
			handleSubmit();
		}}
		class="mt-6 space-y-6"
	>
		<Card>
			<h2 class="text-base font-semibold text-primary">Zdaňovací období</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label for="year" class="block text-sm font-medium text-secondary">Rok</label>
					<input
						id="year"
						type="number"
						bind:value={form.year}
						min="2020"
						max="2099"
						required
						class={inputClass}
					/>
				</div>
				<div>
					<label for="filing_type" class="block text-sm font-medium text-secondary"
						>Typ podání</label
					>
					<select id="filing_type" bind:value={form.filing_type} class={inputClass}>
						{#each filingTypes as ft (ft.value)}
							<option value={ft.value}>{ft.label}</option>
						{/each}
					</select>
				</div>
			</div>
		</Card>

		<FormActions
			{saving}
			saveLabel="Vytvořit přehled"
			savingLabel="Vytvářím..."
			cancelHref="/tax"
		/>
	</form>
</div>
