<script lang="ts">
	import { formatCZK } from '$lib/utils/money';
</script>

<svelte:head>
	<title>ZFaktury - Dashboard</title>
</svelte:head>

<div>
	<h1 class="text-2xl font-bold text-gray-900">ZFaktury</h1>
	<p class="mt-1 text-sm text-gray-500">Prehled vaseho podnikani</p>

	<!-- Stats cards -->
	<div class="mt-6 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
		<div class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
			<p class="text-sm font-medium text-gray-500">Prijmy tento mesic</p>
			<p class="mt-2 text-2xl font-bold text-gray-900">{formatCZK(0)}</p>
		</div>

		<div class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
			<p class="text-sm font-medium text-gray-500">Naklady tento mesic</p>
			<p class="mt-2 text-2xl font-bold text-gray-900">{formatCZK(0)}</p>
		</div>

		<div class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
			<p class="text-sm font-medium text-gray-500">Neuhrazene faktury</p>
			<p class="mt-2 text-2xl font-bold text-orange-600">0</p>
		</div>

		<div class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
			<p class="text-sm font-medium text-gray-500">Faktury po splatnosti</p>
			<p class="mt-2 text-2xl font-bold text-red-600">0</p>
		</div>
	</div>

	<!-- Recent activity placeholder -->
	<div class="mt-8">
		<h2 class="text-lg font-semibold text-gray-900">Posledni aktivita</h2>
		<div
			class="mt-4 rounded-xl border border-gray-200 bg-white p-8 text-center text-gray-400 shadow-sm"
		>
			Zatim zadna aktivita. Zacnete vytvorenim prvni faktury.
		</div>
	</div>
</div>
