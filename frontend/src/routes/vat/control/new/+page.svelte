<script lang="ts">
	import { goto } from '$app/navigation';
	import { controlStatementApi } from '$lib/api/vat-control';

	let saving = $state(false);
	let error = $state<string | null>(null);

	const currentYear = new Date().getFullYear();
	const currentMonth = new Date().getMonth() + 1;

	let form = $state({
		year: currentYear,
		month: currentMonth,
		filing_type: 'regular'
	});

	const filingTypes = [
		{ value: 'regular', label: 'Radne' },
		{ value: 'corrective', label: 'Nasledne' },
		{ value: 'supplementary', label: 'Opravne' }
	];

	const months = [
		{ value: 1, label: 'Leden' },
		{ value: 2, label: 'Unor' },
		{ value: 3, label: 'Brezen' },
		{ value: 4, label: 'Duben' },
		{ value: 5, label: 'Kveten' },
		{ value: 6, label: 'Cerven' },
		{ value: 7, label: 'Cervenec' },
		{ value: 8, label: 'Srpen' },
		{ value: 9, label: 'Zari' },
		{ value: 10, label: 'Rijen' },
		{ value: 11, label: 'Listopad' },
		{ value: 12, label: 'Prosinec' }
	];

	async function handleSubmit() {
		if (!form.year || form.year < 2000) {
			error = 'Zadejte platny rok';
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
			error = e instanceof Error ? e.message : 'Nepodarilo se vytvorit kontrolni hlaseni';
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Nove kontrolni hlaseni - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-xl">
	<a href="/vat" class="text-sm text-blue-600 hover:text-blue-800">&larr; Zpet na DPH</a>
	<h1 class="mt-2 text-2xl font-bold text-gray-900">Nove kontrolni hlaseni</h1>

	{#if error}
		<div role="alert" class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
			{error}
		</div>
	{/if}

	<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="mt-6 space-y-6">
		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Obdobi</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div>
					<label for="year" class="block text-sm font-medium text-gray-700">Rok *</label>
					<input
						id="year"
						type="number"
						min="2000"
						max="2099"
						bind:value={form.year}
						required
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					/>
				</div>
				<div>
					<label for="month" class="block text-sm font-medium text-gray-700">Mesic *</label>
					<select
						id="month"
						bind:value={form.month}
						required
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					>
						{#each months as m}
							<option value={m.value}>{m.label}</option>
						{/each}
					</select>
				</div>
			</div>
		</div>

		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Typ podani</h2>
			<div class="mt-4">
				<label for="filing_type" class="block text-sm font-medium text-gray-700">Typ</label>
				<select
					id="filing_type"
					bind:value={form.filing_type}
					class="mt-1 w-full max-w-xs rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
				>
					{#each filingTypes as ft}
						<option value={ft.value}>{ft.label}</option>
					{/each}
				</select>
			</div>
		</div>

		<div class="flex gap-3">
			<button
				type="submit"
				disabled={saving}
				class="rounded-lg bg-blue-600 px-6 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50 transition-colors"
			>
				{saving ? 'Vytvari se...' : 'Vytvorit hlaseni'}
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
