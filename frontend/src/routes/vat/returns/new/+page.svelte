<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { vatReturnApi } from '$lib/api/vat';
	import { filingTypeLabels, monthLabels, quarterLabels } from '$lib/utils/vat';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';

	const filingTypes = Object.entries(filingTypeLabels).map(([value, label]) => ({ value, label }));

	const paramYear = page.url.searchParams.get('year');
	const paramMonth = page.url.searchParams.get('month');
	const paramQuarter = page.url.searchParams.get('quarter');

	let saving = $state(false);
	let error = $state<string | null>(null);
	let periodType = $state<'monthly' | 'quarterly'>(paramQuarter ? 'quarterly' : 'monthly');

	let form = $state({
		year: paramYear ? Number(paramYear) : new Date().getFullYear(),
		month: paramMonth ? Number(paramMonth) : 0,
		quarter: paramQuarter ? Number(paramQuarter) : 0,
		filing_type: 'regular'
	});

	$effect(() => {
		if (periodType === 'monthly') {
			form.quarter = 0;
		} else {
			form.month = 0;
		}
	});

	async function handleSubmit() {
		saving = true;
		error = null;

		try {
			const payload: { year: number; month?: number; quarter?: number; filing_type?: string } = {
				year: form.year,
				filing_type: form.filing_type
			};
			if (form.month > 0) payload.month = form.month;
			if (form.quarter > 0) payload.quarter = form.quarter;

			const result = await vatReturnApi.create(payload);
			goto(`/vat/returns/${result.id}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se vytvořit přiznání';
		} finally {
			saving = false;
		}
	}

	const inputClass =
		'mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none';
</script>

<svelte:head>
	<title>Nové DPH přiznání - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-2xl">
	<a href="/vat" class="text-sm text-secondary hover:text-primary">&larr; Zpět na DPH</a>
	<h1 class="mt-2 text-xl font-semibold text-primary">Nové DPH přiznání</h1>

	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-danger/20 bg-danger-bg p-4 text-sm text-danger"
		>
			{error}
		</div>
	{/if}

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
					<span class="block text-sm font-medium text-secondary">Typ období</span>
					<div class="mt-1 flex rounded-lg border border-border overflow-hidden">
						<button
							type="button"
							onclick={() => {
								periodType = 'monthly';
							}}
							class="flex-1 px-3 py-2 text-sm font-medium transition-colors {periodType ===
							'monthly'
								? 'bg-accent text-white'
								: 'bg-elevated text-secondary hover:bg-hover'}"
						>
							Měsíční
						</button>
						<button
							type="button"
							onclick={() => {
								periodType = 'quarterly';
							}}
							class="flex-1 px-3 py-2 text-sm font-medium transition-colors border-l border-border {periodType ===
							'quarterly'
								? 'bg-accent text-white'
								: 'bg-elevated text-secondary hover:bg-hover'}"
						>
							Čtvrtletní
						</button>
					</div>
				</div>
			</div>
			<div class="mt-4">
				{#if periodType === 'monthly'}
					<label for="month" class="block text-sm font-medium text-secondary">Měsíc</label>
					<select
						id="month"
						bind:value={form.month}
						class={inputClass}
					>
						<option value={0}>-- Nevybráno --</option>
						{#each Object.entries(monthLabels) as [value, label] (value)}
							<option value={Number(value)}>{label}</option>
						{/each}
					</select>
				{:else}
					<label for="quarter" class="block text-sm font-medium text-secondary">Čtvrtletí</label>
					<select
						id="quarter"
						bind:value={form.quarter}
						class={inputClass}
					>
						<option value={0}>-- Nevybráno --</option>
						{#each Object.entries(quarterLabels) as [value, label] (value)}
							<option value={Number(value)}>{label}</option>
						{/each}
					</select>
				{/if}
			</div>
		</Card>

		<Card>
			<h2 class="text-base font-semibold text-primary">Typ přiznání</h2>
			<div class="mt-4">
				<label for="filing_type" class="block text-sm font-medium text-secondary">Typ podání</label>
				<select
					id="filing_type"
					bind:value={form.filing_type}
					class={inputClass}
				>
					{#each filingTypes as ft (ft.value)}
						<option value={ft.value}>{ft.label}</option>
					{/each}
				</select>
			</div>
		</Card>

		<!-- Actions -->
		<div class="flex gap-3">
			<Button variant="primary" type="submit" disabled={saving}>
				{saving ? 'Vytvářím...' : 'Vytvořit přiznání'}
			</Button>
			<Button variant="secondary" href="/vat">
				Zrušit
			</Button>
		</div>
	</form>
</div>
