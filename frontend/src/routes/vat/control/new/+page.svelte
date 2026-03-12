<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { controlStatementApi } from '$lib/api/client';
	import { filingTypeLabels, monthLabels } from '$lib/utils/vat';
	import Card from '$lib/ui/Card.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';

	let saving = $state(false);
	let error = $state<string | null>(null);

	const paramYear = page.url.searchParams.get('year');
	const paramMonth = page.url.searchParams.get('month');

	let form = $state({
		year: paramYear ? Number(paramYear) : new Date().getFullYear(),
		month: paramMonth ? Number(paramMonth) : new Date().getMonth() + 1,
		filing_type: 'regular'
	});

	const filingTypes = Object.entries(filingTypeLabels).map(([value, label]) => ({ value, label }));
	const months = Object.entries(monthLabels).map(([value, label]) => ({
		value: Number(value),
		label
	}));

	async function handleSubmit() {
		if (!form.year || form.year < 2000) {
			error = 'Zadejte platný rok';
			return;
		}

		saving = true;
		error = null;

		try {
			const result = await controlStatementApi.create({
				year: form.year,
				month: form.month,
				filing_type: form.filing_type
			});
			goto(`/vat/control/${result.id}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se vytvořit kontrolní hlášení';
		} finally {
			saving = false;
		}
	}

	const inputClass =
		'mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none';
</script>

<svelte:head>
	<title>Nové kontrolní hlášení - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-2xl">
	<PageHeader title="Nové kontrolní hlášení" backHref="/vat" backLabel="Zpět na DPH" />

	<ErrorAlert {error} class="mt-4" />

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
					<label for="month" class="block text-sm font-medium text-secondary"
						>Měsíc <HelpTip topic="zdanovaci-obdobi" /> *</label
					>
					<select id="month" bind:value={form.month} required class={inputClass}>
						{#each months as m (m.value)}
							<option value={m.value}>{m.label}</option>
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
