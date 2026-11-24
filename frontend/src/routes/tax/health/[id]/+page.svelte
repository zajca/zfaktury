<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		healthInsuranceApi,
		ApiError,
		type HealthInsuranceOverview,
		type HealthNoMinimumReason,
		type TaxConstants
	} from '$lib/api/client';
	import { loadTaxConstants } from '$lib/data/tax-constants.svelte';
	import AdvanceSchedule from '$lib/components/tax/AdvanceSchedule.svelte';
	import ActivityMonthsPicker from '$lib/components/tax/ActivityMonthsPicker.svelte';
	import { formatCZK } from '$lib/utils/money';
	import Badge from '$lib/ui/Badge.svelte';
	import Button from '$lib/ui/Button.svelte';
	import ConfirmDialog from '$lib/ui/ConfirmDialog.svelte';
	import Card from '$lib/ui/Card.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import { toastError, toastSuccess } from '$lib/data/toast-state.svelte';

	let data = $state<HealthInsuranceOverview | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let actionLoading = $state<string | null>(null);
	let taxConstants = $state<TaxConstants | null>(null);
	let showFileConfirm = $state(false);
	let showDeleteConfirm = $state(false);
	let editingMonths = $state(false);
	let monthsDraft = $state<number[]>([]);
	let editingNoMinimum = $state(false);
	let noMinimumDraft = $state<number[]>([]);
	let noMinimumReasonDraft = $state<HealthNoMinimumReason>('');

	let returnId = $derived(Number(page.params.id));

	const statusLabels: Record<string, string> = {
		draft: 'Koncept',
		ready: 'Připraveno',
		filed: 'Podáno'
	};

	const filingTypeLabels: Record<string, string> = {
		regular: 'Řádné',
		corrective: 'Následné',
		supplementary: 'Opravné'
	};

	onMount(() => {
		loadData();
	});

	async function loadData() {
		loading = true;
		error = null;
		try {
			data = await healthInsuranceApi.getById(returnId);
			if (data) {
				loadTaxConstants(data.year).then((tc) => {
					taxConstants = tc;
				});
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst přehled';
		} finally {
			loading = false;
		}
	}

	async function handleRecalculate() {
		actionLoading = 'recalculate';
		try {
			data = await healthInsuranceApi.recalculate(returnId);
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se přepočítat');
		} finally {
			actionLoading = null;
		}
	}

	const monthNames = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12].map((month) =>
		new Intl.DateTimeFormat('cs-CZ', { month: 'long' }).format(new Date(2024, month - 1, 1))
	);

	let monthsLabel = $derived(
		data
			? data.activity_months.length === 12
				? 'celý rok'
				: `${data.activity_months.map((m) => monthNames[m - 1]).join(', ')} (${data.activity_months.length} měs.)`
			: ''
	);

	function startEditingMonths() {
		monthsDraft = [...(data?.activity_months ?? [])];
		editingMonths = true;
	}

	async function saveActivityMonths() {
		if (monthsDraft.length === 0) {
			return;
		}

		actionLoading = 'months';
		try {
			data = await healthInsuranceApi.setActivityMonths(returnId, monthsDraft);
			toastSuccess('Měsíce činnosti uloženy a přehled přepočítán');
			editingMonths = false;
		} catch (e) {
			toastError(activityMonthsError(e));
		} finally {
			actionLoading = null;
		}
	}

	function activityMonthsError(e: unknown): string {
		if (e instanceof ApiError) {
			if (e.status === 409) {
				return 'Přehled je už podaný, měsíce nelze změnit.';
			}
			if (e.status === 400) {
				// The API answers an empty selection in English, a year without
				// rules in Czech; show the server wording unless it is the
				// generic fallback ApiError builds when no message came.
				const message = e.message.startsWith('API Error ') ? '' : e.message;
				if (message && !message.includes('at least one activity month')) {
					return message;
				}
				return 'Vyberte alespoň jeden měsíc činnosti.';
			}
		}
		return 'Nepodařilo se uložit měsíce činnosti';
	}

	const NO_MINIMUM_REASON_OPTIONS: { value: HealthNoMinimumReason; label: string }[] = [
		{ value: '', label: 'Vyberte důvod' },
		{ value: 'employee', label: 'Zaměstnanec s pojistným alespoň z minimální mzdy' },
		{
			value: 'state_insured',
			label: 'Pojištěnec státu (např. student, rodič na rodičovské, důchodce)'
		},
		{ value: 'ztp', label: 'Držitel průkazu ZTP nebo ZTP/P' },
		{ value: 'retirement_age', label: 'Dosažený důchodový věk bez nároku na důchod' }
	];

	const noMinimumReasonLabels: Record<HealthNoMinimumReason, string> = Object.fromEntries(
		NO_MINIMUM_REASON_OPTIONS.map((opt) => [opt.value, opt.label])
	) as Record<HealthNoMinimumReason, string>;

	let noMinimumLabel = $derived(
		data
			? `${data.no_minimum_months.map((m) => monthNames[m - 1]).join(', ')} (${data.no_minimum_months.length} měs.)`
			: ''
	);

	function startEditingNoMinimum() {
		noMinimumDraft = [...(data?.no_minimum_months ?? [])];
		noMinimumReasonDraft = data?.no_minimum_reason ?? '';
		editingNoMinimum = true;
	}

	async function saveNoMinimum() {
		actionLoading = 'no-minimum';
		try {
			data = await healthInsuranceApi.setNoMinimum(returnId, noMinimumDraft, noMinimumReasonDraft);
			toastSuccess('Měsíce bez minima uloženy a přehled přepočítán');
			editingNoMinimum = false;
		} catch (e) {
			toastError(noMinimumError(e));
		} finally {
			actionLoading = null;
		}
	}

	function noMinimumError(e: unknown): string {
		if (e instanceof ApiError) {
			if (e.status === 409) {
				return 'Přehled je už podaný, nelze ho změnit.';
			}
			if (e.status === 400 && !e.message.startsWith('API Error ')) {
				return e.message;
			}
		}
		return 'Nepodařilo se uložit měsíce bez minima';
	}

	function handleMarkFiled() {
		showFileConfirm = true;
	}

	async function confirmMarkFiled() {
		showFileConfirm = false;
		actionLoading = 'filed';
		try {
			data = await healthInsuranceApi.markFiled(returnId);
			toastSuccess('Přehled ZP označen jako podaný');
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se označit jako podané');
		} finally {
			actionLoading = null;
		}
	}

	function handleDelete() {
		showDeleteConfirm = true;
	}

	async function confirmDelete() {
		showDeleteConfirm = false;
		actionLoading = 'delete';
		try {
			await healthInsuranceApi.delete(returnId);
			toastSuccess('Přehled smazán');
			goto('/tax');
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se smazat přehled');
		} finally {
			actionLoading = null;
		}
	}

	function statusBadgeVariant(
		status: string
	): 'default' | 'success' | 'danger' | 'warning' | 'info' | 'muted' {
		switch (status) {
			case 'filed':
				return 'success';
			case 'ready':
				return 'info';
			default:
				return 'default';
		}
	}
</script>

<svelte:head>
	<title>{data ? `Zdravotní pojištění ${data.year}` : 'Zdravotní pojištění'} - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<a href="/tax" class="text-sm text-secondary hover:text-primary">&larr; Zpět na daně</a>

	<ErrorAlert {error} class="mt-4" />

	{#if loading}
		<LoadingSpinner class="mt-8" />
	{:else if data}
		<!-- Header -->
		<div class="mt-4">
			<div class="flex items-center justify-between">
				<h1 class="text-xl font-semibold text-primary">Přehled ZP - {data.year}</h1>
				<div class="flex items-center gap-3">
					<Badge variant={statusBadgeVariant(data.status)}>
						{statusLabels[data.status] ?? data.status}
					</Badge>
					<span class="text-sm text-secondary">
						{filingTypeLabels[data.filing_type] ?? data.filing_type}
					</span>
				</div>
			</div>
			<div class="mt-3 flex flex-wrap gap-2">
				<Button
					variant="secondary"
					onclick={handleRecalculate}
					disabled={data.status === 'filed' || actionLoading !== null}
				>
					{actionLoading === 'recalculate' ? 'Přepočítávám...' : 'Přepočítat'}
				</Button>
				<Button variant="secondary" disabled={true} title="XML generování zatím není k dispozici">
					Generovat XML
				</Button>
				<Button variant="secondary" disabled={true} title="XML generování zatím není k dispozici">
					Stáhnout XML
				</Button>
				{#if data.status !== 'filed'}
					<Button variant="success" onclick={handleMarkFiled} disabled={actionLoading !== null}>
						{actionLoading === 'filed' ? 'Označuji...' : 'Označit jako podané'}
					</Button>
					<Button variant="danger" onclick={handleDelete} disabled={actionLoading !== null}>
						Smazat
					</Button>
				{/if}
			</div>
			<p class="mt-2 text-xs text-muted">
				XML generování zatím není k dispozici pro zdravotní pojišťovnu.
			</p>
		</div>

		<div class="mt-6 space-y-6">
			<!-- Revenue and Expenses -->
			<Card>
				<h2 class="text-base font-semibold text-primary">Příjmy a výdaje</h2>
				<div class="mt-4 grid grid-cols-2 gap-y-3 gap-x-4 text-sm">
					<dt class="text-secondary">Celkové příjmy</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.total_revenue)}
					</dd>

					<dt class="text-secondary">Použité výdaje</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.total_expenses)}
					</dd>
				</div>
			</Card>

			<!-- Assessment Base -->
			<Card>
				<h2 class="text-base font-semibold text-primary">
					Vyměřovací základ <HelpTip topic="vymerovaci-zaklad" {taxConstants} />
				</h2>
				<div class="mt-4 grid grid-cols-2 gap-y-3 gap-x-4 text-sm">
					<dt class="text-secondary">Měsíce činnosti</dt>
					<dd class="text-right font-medium text-primary">{monthsLabel}</dd>

					<dt class="text-secondary">Základ daně</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.tax_base)}
					</dd>

					<dt class="text-secondary">Vyměřovací základ (50%)</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.assessment_base)}
					</dd>

					<dt class="text-secondary">Minimální vyměřovací základ</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.min_assessment_base)}
					</dd>

					<dt class="text-secondary">Výsledný vyměřovací základ</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.final_assessment_base)}
					</dd>
				</div>

				{#if data.status !== 'filed'}
					<div class="mt-4 border-t border-border pt-4">
						{#if !editingMonths}
							<Button
								variant="secondary"
								onclick={startEditingMonths}
								disabled={actionLoading !== null}
							>
								Upravit měsíce
							</Button>
						{:else}
							<ActivityMonthsPicker
								idPrefix="health-months-edit"
								value={monthsDraft}
								onchange={(months) => (monthsDraft = months)}
								disabled={actionLoading !== null}
							/>
							<div class="mt-3 flex gap-3">
								<Button
									variant="primary"
									onclick={saveActivityMonths}
									disabled={actionLoading !== null || monthsDraft.length === 0}
								>
									{actionLoading === 'months' ? 'Ukládám...' : 'Uložit'}
								</Button>
								<Button
									variant="secondary"
									onclick={() => (editingMonths = false)}
									disabled={actionLoading !== null}
								>
									Zrušit
								</Button>
							</div>
						{/if}
					</div>
				{/if}
			</Card>

			<!-- Months without minimum -->
			<Card>
				<h2 class="text-base font-semibold text-primary">Měsíce bez minima</h2>
				<p class="mt-1 text-sm text-secondary">
					{#if data.no_minimum_months.length === 0}
						Minimum platí ve všech měsících činnosti
					{:else}
						{noMinimumLabel} – {noMinimumReasonLabels[data.no_minimum_reason]}
					{/if}
				</p>

				{#if data.status !== 'filed'}
					<div class="mt-4 border-t border-border pt-4">
						{#if !editingNoMinimum}
							<Button
								variant="secondary"
								onclick={startEditingNoMinimum}
								disabled={actionLoading !== null}
							>
								Upravit
							</Button>
						{:else}
							<ActivityMonthsPicker
								idPrefix="health-no-minimum-edit"
								legend="Měsíce bez minimálního vyměřovacího základu"
								hint="Celé měsíce, ve kterých se na vás minimum nevztahovalo (§ 3a odst. 3 zákona č. 592/1992 Sb.)."
								allowedMonths={data.activity_months}
								required={false}
								value={noMinimumDraft}
								onchange={(months) => (noMinimumDraft = months)}
								disabled={actionLoading !== null}
							/>
							<div class="mt-3 max-w-md">
								<label for="no-minimum-reason" class="block text-sm font-medium text-secondary"
									>Důvod</label
								>
								<select
									id="no-minimum-reason"
									bind:value={noMinimumReasonDraft}
									disabled={actionLoading !== null}
									class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
								>
									{#each NO_MINIMUM_REASON_OPTIONS as opt}
										<option value={opt.value}>{opt.label}</option>
									{/each}
								</select>
							</div>
							<div class="mt-3 flex gap-3">
								<Button variant="primary" onclick={saveNoMinimum} disabled={actionLoading !== null}>
									{actionLoading === 'no-minimum' ? 'Ukládám...' : 'Uložit'}
								</Button>
								<Button
									variant="secondary"
									onclick={() => (editingNoMinimum = false)}
									disabled={actionLoading !== null}
								>
									Zrušit
								</Button>
							</div>
						{/if}
					</div>
				{/if}
			</Card>

			<!-- Insurance -->
			<Card>
				<h2 class="text-base font-semibold text-primary">
					Pojistné <HelpTip topic="prehled-zp" {taxConstants} />
				</h2>
				<div class="mt-4 grid grid-cols-2 gap-y-3 gap-x-4 text-sm">
					<dt class="text-secondary">Sazba pojistného</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{(data.insurance_rate / 10).toFixed(1)}%
					</dd>

					<dt class="text-secondary">Celkové pojistné</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.total_insurance)}
					</dd>

					<dt class="text-secondary">Zaplacené zálohy</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.prepayments)}
					</dd>
				</div>
				<div class="mt-4 flex items-center justify-between border-t border-border pt-4">
					<span class="font-semibold text-primary">
						{data.difference >= 0 ? 'Doplatek:' : 'Přeplatek:'}
					</span>
					<span
						class="text-lg font-semibold tabular-nums {data.difference >= 0
							? 'text-danger'
							: 'text-success'}"
					>
						{formatCZK(Math.abs(data.difference))}
					</span>
				</div>
			</Card>

			<!-- New Prepayment -->
			<Card>
				<h2 class="text-base font-semibold text-primary">
					Nová záloha <HelpTip topic="nova-zaloha" {taxConstants} />
				</h2>
				<div class="mt-4 grid grid-cols-2 gap-y-3 gap-x-4 text-sm">
					<dt class="text-secondary">Nová měsíční záloha</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.new_monthly_prepay)}
					</dd>
				</div>
				<AdvanceSchedule
					overviewYear={data.year}
					startMonth={data.advance_start_month}
					periods={[]}
					warnings={data.warnings}
					reportedPrepay={null}
				/>
			</Card>

			<!-- Timestamps -->
			<div class="text-xs text-muted">
				Vytvořeno: {new Date(data.created_at).toLocaleDateString('cs-CZ')} | Upraveno: {new Date(
					data.updated_at
				).toLocaleDateString('cs-CZ')}
				{#if data.filed_at}
					| Podáno: {new Date(data.filed_at).toLocaleDateString('cs-CZ')}{/if}
			</div>
		</div>
	{/if}
</div>

<ConfirmDialog
	bind:open={showFileConfirm}
	title="Označit jako podané"
	message="Opravdu chcete označit přehled ZP jako podaný?"
	confirmLabel="Označit jako podané"
	variant="warning"
	onconfirm={confirmMarkFiled}
	oncancel={() => (showFileConfirm = false)}
/>

<ConfirmDialog
	bind:open={showDeleteConfirm}
	title="Smazat přehled"
	message="Opravdu chcete smazat tento přehled?"
	confirmLabel="Smazat"
	onconfirm={confirmDelete}
	oncancel={() => (showDeleteConfirm = false)}
/>
