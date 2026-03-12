<script lang="ts">
	import { onMount } from 'svelte';
	import {
		taxYearSettingsApi,
		type TaxYearSettings,
		type TaxPrepaymentMonth,
		type TaxConstants
	} from '$lib/api/client';
	import { loadTaxConstants } from '$lib/data/tax-constants.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';

	const MONTH_NAMES = [
		'Leden',
		'Únor',
		'Březen',
		'Duben',
		'Květen',
		'Červen',
		'Červenec',
		'Srpen',
		'Září',
		'Říjen',
		'Listopad',
		'Prosinec'
	];

	const FLAT_RATE_OPTIONS = [
		{ value: 0, label: 'Skutečné výdaje' },
		{ value: 30, label: '30 %' },
		{ value: 40, label: '40 %' },
		{ value: 60, label: '60 %' },
		{ value: 80, label: '80 %' }
	];

	let selectedYear = $state(new Date().getFullYear() - 1);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state<string | null>(null);
	let success = $state(false);
	let taxConstants = $state<TaxConstants | null>(null);

	let flatRatePercent = $state(0);
	let prepayments = $state<TaxPrepaymentMonth[]>(emptyPrepayments());

	// Quick-fill values (in CZK)
	let fillTax = $state('');
	let fillSocial = $state('');
	let fillHealth = $state('');

	function emptyPrepayments(): TaxPrepaymentMonth[] {
		return Array.from({ length: 12 }, (_, i) => ({
			month: i + 1,
			tax_amount: 0,
			social_amount: 0,
			health_amount: 0
		}));
	}

	// Convert halere to CZK for display
	function toCZK(halere: number): string {
		const czk = Math.round(halere / 100);
		return czk === 0 ? '' : String(czk);
	}

	// Convert CZK input to halere for storage
	function toHalere(czk: string): number {
		const num = parseInt(czk, 10);
		return isNaN(num) ? 0 : num * 100;
	}

	let totalTax = $derived(prepayments.reduce((sum, p) => sum + p.tax_amount, 0));
	let totalSocial = $derived(prepayments.reduce((sum, p) => sum + p.social_amount, 0));
	let totalHealth = $derived(prepayments.reduce((sum, p) => sum + p.health_amount, 0));

	async function loadData() {
		loading = true;
		error = null;
		// Load tax constants independently (never fails)
		loadTaxConstants(selectedYear).then((tc) => {
			taxConstants = tc;
		});
		try {
			const data: TaxYearSettings = await taxYearSettingsApi.getByYear(selectedYear);
			flatRatePercent = data.flat_rate_percent;
			if (data.prepayments && data.prepayments.length === 12) {
				prepayments = data.prepayments;
			} else {
				prepayments = emptyPrepayments();
			}
		} catch (e) {
			// If 404 or not found, start with empty defaults
			flatRatePercent = 0;
			prepayments = emptyPrepayments();
		} finally {
			loading = false;
		}
	}

	async function handleSave() {
		saving = true;
		error = null;
		success = false;
		try {
			await taxYearSettingsApi.save(selectedYear, {
				flat_rate_percent: flatRatePercent,
				prepayments
			});
			success = true;
			setTimeout(() => {
				success = false;
			}, 3000);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se uložit nastavení';
		} finally {
			saving = false;
		}
	}

	function fillAllMonths() {
		const taxVal = toHalere(fillTax);
		const socialVal = toHalere(fillSocial);
		const healthVal = toHalere(fillHealth);
		prepayments = prepayments.map((p) => ({
			...p,
			tax_amount: taxVal,
			social_amount: socialVal,
			health_amount: healthVal
		}));
	}

	function formatTotal(halere: number): string {
		return new Intl.NumberFormat('cs-CZ').format(Math.round(halere / 100));
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
</script>

<svelte:head>
	<title>Daňové zálohy a nastavení {selectedYear} - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<PageHeader
		title="Daňové zálohy a nastavení"
		description="Paušální výdaje a měsíční zálohy na daň a pojistné"
	/>

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
	</div>

	<ErrorAlert {error} class="mt-4" />

	{#if success}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-success/20 bg-success-bg p-3 text-sm text-success"
		>
			Nastavení bylo uloženo.
		</div>
	{/if}

	{#if loading}
		<LoadingSpinner class="mt-8 p-12" />
	{:else}
		<div class="mt-6 space-y-6">
			<!-- Pausalni vydaje -->
			<Card>
				<h2 class="text-base font-semibold text-primary">
					Paušální výdaje <HelpTip topic="pausalni-vydaje" {taxConstants} />
				</h2>
				<p class="mt-1 text-sm text-tertiary">
					Procento paušálních výdajů pro výpočet daňového základu.
				</p>
				<div class="mt-4 max-w-xs">
					<label for="flat_rate_percent" class="block text-sm font-medium text-secondary"
						>Sazba paušálních výdajů</label
					>
					<select
						id="flat_rate_percent"
						value={flatRatePercent}
						onchange={(e) => {
							flatRatePercent = parseInt((e.target as HTMLSelectElement).value, 10);
						}}
						class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
					>
						{#each FLAT_RATE_OPTIONS as opt}
							<option value={opt.value}>{opt.label}</option>
						{/each}
					</select>
				</div>
			</Card>

			<!-- Měsíční zálohy -->
			<Card>
				<h2 class="text-base font-semibold text-primary">
					Měsíční zálohy <HelpTip topic="nova-zaloha" {taxConstants} />
				</h2>
				<p class="mt-1 text-sm text-tertiary">
					Zadejte měsíční zálohy na daň z příjmu, sociální a zdravotní pojištění v Kč.
				</p>

				<div class="mt-4 overflow-x-auto">
					<table class="w-full text-sm">
						<thead>
							<tr class="border-b border-border">
								<th class="py-2 pr-3 text-left font-medium text-secondary">Měsíc</th>
								<th class="px-2 py-2 text-right font-medium text-secondary">Daň z příjmu (Kč)</th>
								<th class="px-2 py-2 text-right font-medium text-secondary">Sociální (Kč)</th>
								<th class="px-2 py-2 text-right font-medium text-secondary">Zdravotní (Kč)</th>
							</tr>
						</thead>
						<tbody>
							<!-- Quick-fill row -->
							<tr class="border-b border-border bg-elevated/50">
								<td class="py-2 pr-3">
									<span class="text-xs font-medium text-accent">Vyplnit vše</span>
								</td>
								<td class="px-2 py-2">
									<input
										type="number"
										bind:value={fillTax}
										placeholder="0"
										aria-label="Vyplnit vše - Daň z příjmu"
										class="w-full rounded-lg border border-border bg-surface px-3 py-1.5 text-right text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
									/>
								</td>
								<td class="px-2 py-2">
									<input
										type="number"
										bind:value={fillSocial}
										placeholder="0"
										aria-label="Vyplnit vše - Sociální"
										class="w-full rounded-lg border border-border bg-surface px-3 py-1.5 text-right text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
									/>
								</td>
								<td class="px-2 py-2">
									<div class="flex items-center gap-2">
										<input
											type="number"
											bind:value={fillHealth}
											placeholder="0"
											aria-label="Vyplnit vše - Zdravotní"
											class="w-full rounded-lg border border-border bg-surface px-3 py-1.5 text-right text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
										/>
										<Button variant="secondary" size="sm" onclick={fillAllMonths}>Vyplnit</Button>
									</div>
								</td>
							</tr>

							<!-- Month rows -->
							{#each prepayments as prepayment, i (prepayment.month)}
								<tr class="border-b border-border">
									<td class="py-2 pr-3 font-medium text-primary">{MONTH_NAMES[i]}</td>
									<td class="px-2 py-2">
										<input
											type="number"
											value={toCZK(prepayment.tax_amount)}
											oninput={(e) => {
												prepayments[i] = {
													...prepayments[i],
													tax_amount: toHalere((e.target as HTMLInputElement).value)
												};
											}}
											placeholder="0"
											aria-label="{MONTH_NAMES[i]} - Daň z příjmu"
											class="w-full rounded-lg border border-border bg-surface px-3 py-1.5 text-right text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
										/>
									</td>
									<td class="px-2 py-2">
										<input
											type="number"
											value={toCZK(prepayment.social_amount)}
											oninput={(e) => {
												prepayments[i] = {
													...prepayments[i],
													social_amount: toHalere((e.target as HTMLInputElement).value)
												};
											}}
											placeholder="0"
											aria-label="{MONTH_NAMES[i]} - Sociální"
											class="w-full rounded-lg border border-border bg-surface px-3 py-1.5 text-right text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
										/>
									</td>
									<td class="px-2 py-2">
										<input
											type="number"
											value={toCZK(prepayment.health_amount)}
											oninput={(e) => {
												prepayments[i] = {
													...prepayments[i],
													health_amount: toHalere((e.target as HTMLInputElement).value)
												};
											}}
											placeholder="0"
											aria-label="{MONTH_NAMES[i]} - Zdravotní"
											class="w-full rounded-lg border border-border bg-surface px-3 py-1.5 text-right text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
										/>
									</td>
								</tr>
							{/each}

							<!-- Totals row -->
							<tr class="bg-elevated/50 font-semibold">
								<td class="py-2 pr-3 text-primary">Celkem za rok</td>
								<td class="px-2 py-2 text-right text-primary">{formatTotal(totalTax)} Kč</td>
								<td class="px-2 py-2 text-right text-primary">{formatTotal(totalSocial)} Kč</td>
								<td class="px-2 py-2 text-right text-primary">{formatTotal(totalHealth)} Kč</td>
							</tr>
						</tbody>
					</table>
				</div>
			</Card>

			<!-- Save button -->
			<div class="flex justify-end pb-8">
				<Button variant="primary" onclick={handleSave} disabled={saving}>
					{#if saving}
						Ukládám...
					{:else}
						Uložit nastavení
					{/if}
				</Button>
			</div>
		</div>
	{/if}
</div>
