<script lang="ts">
	import { onMount } from 'svelte';
	import { exchangeRateApi, type ExchangeRateResult } from '$lib/api/client';

	const CURRENCIES = ['CZK', 'EUR', 'USD', 'GBP', 'PLN', 'CHF'] as const;

	interface Props {
		currency: string;
		exchangeRate: number;
		date?: string;
		onchange?: (currency: string, rate: number) => void;
	}

	let {
		currency = $bindable('CZK'),
		exchangeRate = $bindable(1),
		date,
		onchange
	}: Props = $props();

	let loading = $state(false);
	let error = $state('');
	let rateSource = $state('');
	let mounted = false;

	onMount(() => {
		mounted = true;
	});

	$effect(() => {
		// Track currency for reactivity
		const curr = currency;
		if (!mounted) return;

		if (curr === 'CZK') {
			exchangeRate = 1;
			rateSource = '';
			error = '';
			onchange?.(curr, 1);
		} else {
			fetchRate(curr);
		}
	});

	async function fetchRate(curr?: string) {
		const targetCurrency = curr ?? currency;
		if (targetCurrency === 'CZK') return;

		loading = true;
		error = '';

		try {
			const result: ExchangeRateResult = await exchangeRateApi.getRate(targetCurrency, date);
			exchangeRate = result.rate;
			rateSource = `CNB, ${result.date}`;
			onchange?.(targetCurrency, result.rate);
		} catch (e) {
			error = 'Nepodařilo se načíst kurz';
		} finally {
			loading = false;
		}
	}

	function handleCurrencyChange(e: Event) {
		const target = e.target as HTMLSelectElement;
		currency = target.value;
	}

	function handleRateInput(e: Event) {
		const target = e.target as HTMLInputElement;
		const val = parseFloat(target.value);
		if (!isNaN(val)) {
			exchangeRate = val;
			onchange?.(currency, val);
		}
	}
</script>

<div class="flex flex-col gap-2">
	<select
		value={currency}
		onchange={handleCurrencyChange}
		class="rounded-lg border border-border bg-surface px-3 py-2.5 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
		aria-label="Měna"
	>
		{#each CURRENCIES as code (code)}
			<option value={code}>{code}</option>
		{/each}
	</select>

	{#if currency !== 'CZK'}
		<div class="flex flex-col gap-1">
			<div class="flex items-center gap-2">
				<input
					type="number"
					value={exchangeRate}
					oninput={handleRateInput}
					step="0.001"
					min="0"
					class="w-full rounded-lg border border-border bg-surface px-4 py-2.5 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
					aria-label="Směnný kurz"
				/>
				<button
					type="button"
					onclick={() => fetchRate()}
					disabled={loading}
					class="shrink-0 rounded-md px-2.5 py-1.5 text-xs font-medium text-secondary hover:bg-hover hover:text-primary transition-colors disabled:opacity-50"
				>
					{#if loading}
						<span role="status">
							<span class="sr-only">Načítání...</span>
							...
						</span>
					{:else}
						Načíst kurz
					{/if}
				</button>
			</div>
			{#if error}
				<p class="text-danger text-xs" role="alert">{error}</p>
			{/if}
			{#if rateSource && !error}
				<span class="text-muted text-xs">{rateSource}</span>
			{/if}
		</div>
	{/if}
</div>
