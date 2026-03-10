<script lang="ts">
	import { goto } from '$app/navigation';
	import { vatReturnApi } from '$lib/api/vat';

	const filingTypes = [
		{ value: 'regular', label: 'Radne' },
		{ value: 'corrective', label: 'Nasledne' },
		{ value: 'supplementary', label: 'Opravne' }
	];

	let saving = $state(false);
	let error = $state<string | null>(null);

	let form = $state({
		year: new Date().getFullYear(),
		month: 0,
		quarter: 0,
		filing_type: 'regular'
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
			error = e instanceof Error ? e.message : 'Nepodarilo se vytvorit priznani';
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Nove DPH priznani - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-2xl">
	<a href="/vat" class="text-sm text-blue-600 hover:text-blue-800">&larr; Zpet na DPH</a>
	<h1 class="mt-2 text-2xl font-bold text-gray-900">Nove DPH priznani</h1>

	{#if error}
		<div role="alert" class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
			{error}
		</div>
	{/if}

	<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="mt-6 space-y-6">
		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Obdobi</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
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
					<label for="month" class="block text-sm font-medium text-gray-700">Mesic</label>
					<select
						id="month"
						bind:value={form.month}
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					>
						<option value={0}>-- Nevybrano --</option>
						{#each Array.from({ length: 12 }, (_, i) => i + 1) as m}
							<option value={m}>{m}</option>
						{/each}
					</select>
				</div>
				<div>
					<label for="quarter" class="block text-sm font-medium text-gray-700">Ctvrtleti</label>
					<select
						id="quarter"
						bind:value={form.quarter}
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					>
						<option value={0}>-- Nevybrano --</option>
						<option value={1}>Q1</option>
						<option value={2}>Q2</option>
						<option value={3}>Q3</option>
						<option value={4}>Q4</option>
					</select>
				</div>
			</div>
		</div>

		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Typ priznani</h2>
			<div class="mt-4">
				<label for="filing_type" class="block text-sm font-medium text-gray-700">Typ podani</label>
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
				{saving ? 'Vytvarim...' : 'Vytvorit priznani'}
			</button>
			<a
				href="/vat"
				class="rounded-lg border border-gray-300 px-6 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
			>
				Zrusit
			</a>
		</div>
	</form>
</div>
