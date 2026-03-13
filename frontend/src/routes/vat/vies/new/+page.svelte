<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { viesApi } from '$lib/api/client';
	import { filingTypeLabels, quarterLabels } from '$lib/utils/vat';
	import { toastError } from '$lib/data/toast-state.svelte';
	import Card from '$lib/ui/Card.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';

	let saving = $state(false);

	const paramYear = page.url.searchParams.get('year');
	const paramQuarter = page.url.searchParams.get('quarter');

	let form = $state({
		year: paramYear ? Number(paramYear) : new Date().getFullYear(),
		quarter: paramQuarter ? Number(paramQuarter) : Math.ceil((new Date().getMonth() + 1) / 3),
		filing_type: 'regular'
	});

	const filingTypes = Object.entries(filingTypeLabels).map(([value, label]) => ({ value, label }));
	const quarters = Object.entries(quarterLabels).map(([value, label]) => ({
		value: Number(value),
		label
	}));

	async function handleSubmit() {
		if (!form.year || form.year < 2000) {
			toastError('Zadejte platný rok');
			return;
		}

		saving = true;

		try {
			const result = await viesApi.create({
				year: form.year,
				quarter: form.quarter,
				filing_type: form.filing_type
			});
			goto(`/vat/vies/${result.id}`);
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se vytvořit souhrnné hlášení');
		} finally {
			saving = false;
		}
	}

	const inputClass =
		'mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none';
</script>

<svelte:head>
	<title>Nové souhrnné hlášení - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-2xl">
	<PageHeader title="Nové souhrnné hlášení" backHref="/vat" backLabel="Zpět na DPH" />

	<form
		onsubmit={(e) => {
			e.preventDefault();
			handleSubmit();
		}}
		class="mt-6 space-y-6"
	>
		<Card>
			<h2 class="text-base font-semibold text-primary">Období</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label for="year" class="block text-sm font-medium text-secondary">Rok *</label>
					<input
						id="year"
						type="number"
						min="2000"
						max="2099"
						bind:value={form.year}
						required
						class={inputClass}
					/>
				</div>
				<div>
					<label for="quarter" class="block text-sm font-medium text-secondary">Čtvrtletí *</label>
					<select id="quarter" bind:value={form.quarter} required class={inputClass}>
						{#each quarters as q (q.value)}
							<option value={q.value}>{q.label}</option>
						{/each}
					</select>
				</div>
			</div>
		</Card>

		<Card>
			<h2 class="text-base font-semibold text-primary">Typ podání</h2>
			<div class="mt-4">
				<label for="filing_type" class="block text-sm font-medium text-secondary"
					>Typ <HelpTip topic="typ-podani" /></label
				>
				<select id="filing_type" bind:value={form.filing_type} class="{inputClass} max-w-xs">
					{#each filingTypes as ft (ft.value)}
						<option value={ft.value}>{ft.label}</option>
					{/each}
				</select>
			</div>
		</Card>

		<FormActions
			{saving}
			saveLabel="Vytvořit hlášení"
			savingLabel="Vytvářím..."
			cancelHref="/vat"
		/>
	</form>
</div>
