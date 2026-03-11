<script lang="ts">
	import { formatCZK } from '$lib/utils/money';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
</script>

<svelte:head>
	<title>ZFaktury - Dashboard</title>
</svelte:head>

<div class="mx-auto max-w-6xl">
	<h1 class="text-xl font-semibold text-primary">ZFaktury</h1>
	<p class="mt-1 text-sm text-tertiary">Prehled vaseho podnikani</p>

	<!-- Stats cards -->
	<div class="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
		<Card>
			<p class="text-xs font-medium uppercase tracking-wider text-muted">Prijmy tento mesic <HelpTip topic="prijmy-naklady" /></p>
			<p class="mt-2 text-2xl font-semibold text-primary font-mono tabular-nums">{formatCZK(0)}</p>
		</Card>

		<Card>
			<p class="text-xs font-medium uppercase tracking-wider text-muted">Naklady tento mesic <HelpTip topic="prijmy-naklady" /></p>
			<p class="mt-2 text-2xl font-semibold text-primary font-mono tabular-nums">{formatCZK(0)}</p>
		</Card>

		<Card>
			<p class="text-xs font-medium uppercase tracking-wider text-muted">Neuhrazene faktury <HelpTip topic="neuhrazene-faktury" /></p>
			<p class="mt-2 text-2xl font-semibold text-warning font-mono tabular-nums">0</p>
		</Card>

		<Card>
			<p class="text-xs font-medium uppercase tracking-wider text-muted">Faktury po splatnosti <HelpTip topic="faktury-po-splatnosti" /></p>
			<p class="mt-2 text-2xl font-semibold text-danger font-mono tabular-nums">0</p>
		</Card>
	</div>

	<!-- Recent activity placeholder -->
	<div class="mt-8">
		<h2 class="text-base font-semibold text-primary">Posledni aktivita</h2>
		<Card class="mt-4 text-center text-sm text-muted">
			Zatim zadna aktivita. Zacnete vytvorenim prvni faktury.
		</Card>
	</div>
</div>
