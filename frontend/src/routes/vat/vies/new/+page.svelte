<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { viesApi } from '$lib/api/vat-vies';
	import { filingTypeLabels, quarterLabels } from '$lib/utils/vat';

	let saving = $state(false);
	let error = $state<string | null>(null);

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
			error = 'Zadejte platný rok';
			return;
		}

		saving = true;
		error = null;

		try {
			const result = await viesApi.create({
				year: form.year,
				quarter: form.quarter,
				filing_type: form.filing_type
			});
			goto(`/vat/vies/${result.id}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se vytvořit souhrnné hlášení';
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Nové souhrnné hlášení - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-xl">
	<a href="/vat" class="text-sm text-blue-600 hover:text-blue-800">&larr; Zpět na DPH</a>
	<h1 class="mt-2 text-2xl font-bold text-gray-900">Nové souhrnné hlášení</h1>

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
					<label for="quarter" class="block text-sm font-medium text-gray-700">Čtvrtletí *</label>
					<select
						id="quarter"
						bind:value={form.quarter}
						required
						class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					>
						{#each quarters as q (q.value)}
							<option value={q.value}>{q.label}</option>
						{/each}
					</select>
				</div>
			</div>
		</div>

		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<h2 class="text-lg font-semibold text-gray-900">Typ podání</h2>
			<div class="mt-4">
				<label for="filing_type" class="block text-sm font-medium text-gray-700">Typ</label>
				<select
					id="filing_type"
					bind:value={form.filing_type}
					class="mt-1 w-full max-w-xs rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
				>
					{#each filingTypes as ft (ft.value)}
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
				{saving ? 'Vytvářím...' : 'Vytvořit hlášení'}
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
