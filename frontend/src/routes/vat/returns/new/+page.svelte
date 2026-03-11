<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { vatReturnApi } from '$lib/api/vat';
	import { filingTypeLabels, monthLabels, quarterLabels } from '$lib/utils/vat';

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
</script>

<svelte:head>
	<title>Nové DPH přiznání - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-2xl">
	<a href="/vat" class="text-sm text-blue-600 hover:text-blue-800">&larr; Zpět na DPH</a>
	<h1 class="mt-2 text-2xl font-bold text-gray-900">Nové DPH přiznání</h1>

	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700"
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
		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Období</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label for="year" class="block text-sm font-medium text-gray-700">Rok</label>
					<input
						id="year"
						type="number"
						bind:value={form.year}
						min="2020"
						max="2099"
						required
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					/>
				</div>
				<div>
					<span class="block text-sm font-medium text-gray-700">Typ období</span>
					<div class="mt-1 flex rounded-lg border border-gray-300 overflow-hidden">
						<button
							type="button"
							onclick={() => {
								periodType = 'monthly';
							}}
							class="flex-1 px-3 py-2 text-sm font-medium transition-colors {periodType ===
							'monthly'
								? 'bg-blue-600 text-white'
								: 'bg-white text-gray-700 hover:bg-gray-50'}"
						>
							Měsíční
						</button>
						<button
							type="button"
							onclick={() => {
								periodType = 'quarterly';
							}}
							class="flex-1 px-3 py-2 text-sm font-medium transition-colors border-l border-gray-300 {periodType ===
							'quarterly'
								? 'bg-blue-600 text-white'
								: 'bg-white text-gray-700 hover:bg-gray-50'}"
						>
							Čtvrtletní
						</button>
					</div>
				</div>
			</div>
			<div class="mt-4">
				{#if periodType === 'monthly'}
					<label for="month" class="block text-sm font-medium text-gray-700">Měsíc</label>
					<select
						id="month"
						bind:value={form.month}
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					>
						<option value={0}>-- Nevybráno --</option>
						{#each Object.entries(monthLabels) as [value, label]}
							<option value={Number(value)}>{label}</option>
						{/each}
					</select>
				{:else}
					<label for="quarter" class="block text-sm font-medium text-gray-700">Čtvrtletí</label>
					<select
						id="quarter"
						bind:value={form.quarter}
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					>
						<option value={0}>-- Nevybráno --</option>
						{#each Object.entries(quarterLabels) as [value, label]}
							<option value={Number(value)}>{label}</option>
						{/each}
					</select>
				{/if}
			</div>
		</div>

		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Typ přiznání</h2>
			<div class="mt-4">
				<label for="filing_type" class="block text-sm font-medium text-gray-700">Typ podání</label>
				<select
					id="filing_type"
					bind:value={form.filing_type}
					class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
				>
					{#each filingTypes as ft}
						<option value={ft.value}>{ft.label}</option>
					{/each}
				</select>
			</div>
		</div>

		<!-- Actions -->
		<div class="flex gap-3">
			<button
				type="submit"
				disabled={saving}
				class="rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50 transition-colors"
			>
				{saving ? 'Vytvářím...' : 'Vytvořit přiznání'}
			</button>
			<a
				href="/vat"
				class="rounded-lg border border-gray-300 px-6 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
			>
				Zrušit
			</a>
		</div>
	</form>
</div>
