<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		incomeTaxApi,
		socialInsuranceApi,
		healthInsuranceApi,
		type IncomeTaxReturn,
		type SocialInsuranceOverview,
		type HealthInsuranceOverview,
		type TaxConstants
	} from '$lib/api/client';
	import { loadTaxConstants } from '$lib/data/tax-constants.svelte';
	import { formatCZK } from '$lib/utils/money';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';

	let selectedYear = $state(new Date().getFullYear() - 1);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let taxConstants = $state<TaxConstants | null>(null);

	let incomeTaxReturns = $state<IncomeTaxReturn[]>([]);
	let socialOverviews = $state<SocialInsuranceOverview[]>([]);
	let healthOverviews = $state<HealthInsuranceOverview[]>([]);

	async function loadData() {
		loading = true;
		error = null;
		try {
			const [itr, sio, hio, tc] = await Promise.all([
				incomeTaxApi.list(selectedYear),
				socialInsuranceApi.list(selectedYear),
				healthInsuranceApi.list(selectedYear),
				loadTaxConstants(selectedYear)
			]);
			incomeTaxReturns = itr ?? [];
			socialOverviews = sio ?? [];
			healthOverviews = hio ?? [];
			taxConstants = tc;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst data';
		} finally {
			loading = false;
		}
	}

	let mounted = false;
	onMount(() => {
		loadData();
		mounted = true;
	});

	$effect(() => {
		selectedYear;
		if (!mounted) return;
		loadData();
	});

	const statusLabels: Record<string, string> = {
		draft: 'Koncept',
		ready: 'Připraveno',
		filed: 'Podáno'
	};

	function statusLabel(status: string): string {
		return statusLabels[status] ?? status;
	}

	function statusColor(status: string): string {
		switch (status) {
			case 'filed':
				return 'bg-success-bg text-success';
			case 'ready':
				return 'bg-info-bg text-info';
			default:
				return 'bg-elevated text-secondary';
		}
	}
</script>

<svelte:head>
	<title>Daně za rok {selectedYear} - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-6xl">
	<h1 class="text-xl font-semibold text-primary">
		Daně za rok {selectedYear}
		<HelpTip topic="rocni-dane" {taxConstants} />
	</h1>

	<!-- Year selector -->
	<div class="mt-4 flex items-center gap-3">
		<Button
			variant="ghost"
			size="sm"
			onclick={() => {
				selectedYear--;
			}}
			title="Předchozí rok"
			aria-label="Předchozí rok"
		>
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
			</svg>
		</Button>
		<span class="min-w-[4rem] text-center text-xl font-semibold text-primary tabular-nums"
			>{selectedYear}</span
		>
		<Button
			variant="ghost"
			size="sm"
			onclick={() => {
				selectedYear++;
			}}
			title="Následující rok"
			aria-label="Následující rok"
		>
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
			</svg>
		</Button>
		<a href="/tax/prepayments" class="ml-auto text-sm text-accent hover:underline">
			Nastavení záloh
		</a>
	</div>

	<ErrorAlert {error} class="mt-4" />

	{#if loading}
		<LoadingSpinner class="mt-8 p-12" />
	{:else}
		<div class="mt-6 grid grid-cols-1 gap-6 md:grid-cols-3">
			<!-- DPFO card -->
			<Card>
				<h2 class="text-base font-semibold text-primary">
					Daňové přiznání (DPFO) <HelpTip topic="dan-15-23" {taxConstants} />
				</h2>
				{#if incomeTaxReturns.length === 0}
					<p class="mt-4 text-sm text-tertiary">Zatím nevytvořeno</p>
					<div class="mt-4">
						<Button
							variant="primary"
							size="sm"
							onclick={() => goto(`/tax/income/new?year=${selectedYear}`)}
						>
							Vytvořit
						</Button>
					</div>
				{:else}
					{#each incomeTaxReturns as itr (itr.id)}
						<button
							role="link"
							class="mt-4 block w-full rounded-lg border border-border p-3 text-left transition-colors hover:bg-hover"
							onclick={() => goto(`/tax/income/${itr.id}`)}
						>
							<div class="flex items-center justify-between">
								<span
									class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium {statusColor(
										itr.status
									)}"
								>
									{statusLabel(itr.status)}
								</span>
							</div>
							<div class="mt-2 space-y-1 text-sm">
								<div class="flex justify-between">
									<span class="text-tertiary">Základ daně</span>
									<span class="font-medium text-primary">{formatCZK(itr.tax_base)}</span>
								</div>
								<div class="flex justify-between">
									<span class="text-tertiary">Celková daň</span>
									<span class="font-medium text-primary">{formatCZK(itr.total_tax)}</span>
								</div>
								<div class="flex justify-between">
									<span class="text-tertiary">Doplatek</span>
									<span class="font-medium text-primary">{formatCZK(itr.tax_due)}</span>
								</div>
							</div>
						</button>
					{/each}
				{/if}
			</Card>

			<!-- CSSZ card -->
			<Card>
				<h2 class="text-base font-semibold text-primary">
					Přehled OSVČ pro ČSSZ <HelpTip topic="prehled-cssz" {taxConstants} />
				</h2>
				{#if socialOverviews.length === 0}
					<p class="mt-4 text-sm text-tertiary">Zatím nevytvořeno</p>
					<div class="mt-4">
						<Button
							variant="primary"
							size="sm"
							onclick={() => goto(`/tax/social/new?year=${selectedYear}`)}
						>
							Vytvořit
						</Button>
					</div>
				{:else}
					{#each socialOverviews as so (so.id)}
						<button
							role="link"
							class="mt-4 block w-full rounded-lg border border-border p-3 text-left transition-colors hover:bg-hover"
							onclick={() => goto(`/tax/social/${so.id}`)}
						>
							<div class="flex items-center justify-between">
								<span
									class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium {statusColor(
										so.status
									)}"
								>
									{statusLabel(so.status)}
								</span>
							</div>
							<div class="mt-2 space-y-1 text-sm">
								<div class="flex justify-between">
									<span class="text-tertiary">Vyměřovací základ</span>
									<span class="font-medium text-primary">{formatCZK(so.assessment_base)}</span>
								</div>
								<div class="flex justify-between">
									<span class="text-tertiary">Roční pojistné</span>
									<span class="font-medium text-primary">{formatCZK(so.total_insurance)}</span>
								</div>
								<div class="flex justify-between">
									<span class="text-tertiary">Doplatek</span>
									<span class="font-medium text-primary">{formatCZK(so.difference)}</span>
								</div>
							</div>
						</button>
					{/each}
				{/if}
			</Card>

			<!-- ZP card -->
			<Card>
				<h2 class="text-base font-semibold text-primary">
					Přehled OSVČ pro ZP <HelpTip topic="prehled-zp" {taxConstants} />
				</h2>
				{#if healthOverviews.length === 0}
					<p class="mt-4 text-sm text-tertiary">Zatím nevytvořeno</p>
					<div class="mt-4">
						<Button
							variant="primary"
							size="sm"
							onclick={() => goto(`/tax/health/new?year=${selectedYear}`)}
						>
							Vytvořit
						</Button>
					</div>
				{:else}
					{#each healthOverviews as ho (ho.id)}
						<button
							role="link"
							class="mt-4 block w-full rounded-lg border border-border p-3 text-left transition-colors hover:bg-hover"
							onclick={() => goto(`/tax/health/${ho.id}`)}
						>
							<div class="flex items-center justify-between">
								<span
									class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium {statusColor(
										ho.status
									)}"
								>
									{statusLabel(ho.status)}
								</span>
							</div>
							<div class="mt-2 space-y-1 text-sm">
								<div class="flex justify-between">
									<span class="text-tertiary">Vyměřovací základ</span>
									<span class="font-medium text-primary">{formatCZK(ho.assessment_base)}</span>
								</div>
								<div class="flex justify-between">
									<span class="text-tertiary">Roční pojistné</span>
									<span class="font-medium text-primary">{formatCZK(ho.total_insurance)}</span>
								</div>
								<div class="flex justify-between">
									<span class="text-tertiary">Doplatek</span>
									<span class="font-medium text-primary">{formatCZK(ho.difference)}</span>
								</div>
							</div>
						</button>
					{/each}
				{/if}
			</Card>
		</div>
	{/if}
</div>
