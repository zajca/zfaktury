<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		socialInsuranceApi,
		ApiError,
		type SecondaryActivityReason,
		type SocialInsuranceOverview,
		type TaxConstants
	} from '$lib/api/client';
	import { loadTaxConstants } from '$lib/data/tax-constants.svelte';
	import AdvanceSchedule from '$lib/components/tax/AdvanceSchedule.svelte';
	import ActivityMonthsPicker from '$lib/components/tax/ActivityMonthsPicker.svelte';
	import { downloadFile } from '$lib/utils/download';
	import { formatCZK } from '$lib/utils/money';
	import ISDSSendButton from '$lib/components/ISDSSendButton.svelte';
	import ISDSSubmissionHistory from '$lib/components/ISDSSubmissionHistory.svelte';
	import Badge from '$lib/ui/Badge.svelte';
	import Button from '$lib/ui/Button.svelte';
	import ConfirmDialog from '$lib/ui/ConfirmDialog.svelte';
	import Card from '$lib/ui/Card.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import { toastError, toastSuccess } from '$lib/data/toast-state.svelte';

	let isdsHistory = $state<ISDSSubmissionHistory | undefined>();
	let data = $state<SocialInsuranceOverview | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let actionLoading = $state<string | null>(null);
	let taxConstants = $state<TaxConstants | null>(null);
	let showFileConfirm = $state(false);
	let showDeleteConfirm = $state(false);
	let editingMonths = $state(false);
	let monthsDraft = $state<number[]>([]);
	let editingSecondary = $state(false);
	let secondaryDraft = $state<number[]>([]);
	let secondaryReasonDraft = $state<SecondaryActivityReason>('');
	let employmentDraft = $state<number | null>(null);

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
			data = await socialInsuranceApi.getById(returnId);
			if (data) {
				employmentDraft = employmentToDraft(data.employment_assessment_base);
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
			data = await socialInsuranceApi.recalculate(returnId);
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
			data = await socialInsuranceApi.setActivityMonths(returnId, monthsDraft);
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

	const SECONDARY_REASON_OPTIONS: { value: SecondaryActivityReason; label: string }[] = [
		{ value: '', label: 'Vyberte důvod' },
		{
			value: 'employment',
			label: 'Zaměstnání (účast na nemocenském pojištění zaměstnanců)'
		},
		{ value: 'pension', label: 'Nárok na invalidní důchod nebo přiznaný starobní důchod' },
		{ value: 'child_care', label: 'Osobní péče o dítě do 4 let' },
		{
			value: 'maternity',
			label: 'Peněžitá pomoc v mateřství nebo nemocenské z důvodu těhotenství a porodu'
		},
		{ value: 'care', label: 'Osobní péče o osobu závislou na pomoci jiné osoby' },
		{ value: 'student', label: 'Nezaopatřené dítě (student do 26 let)' }
	];

	const secondaryReasonLabels = Object.fromEntries(
		SECONDARY_REASON_OPTIONS.map((opt) => [opt.value, opt.label])
	) as Record<SecondaryActivityReason, string>;

	// In a year with main and secondary months ř. 27 is the highest of three
	// sums, which the overview stores as its final base; min_assessment_base is
	// the sum of ř. 26.
	let mixedYear = $derived(
		data !== null &&
			data.secondary_months.length > 0 &&
			data.secondary_months.length < data.activity_months.length
	);

	let secondaryLabel = $derived(
		data
			? `${data.secondary_months.map((m) => monthNames[m - 1]).join(', ')} (${data.secondary_months.length} měs.)`
			: ''
	);

	function startEditingSecondary() {
		secondaryDraft = [...(data?.secondary_months ?? [])];
		secondaryReasonDraft = data?.secondary_reason ?? '';
		editingSecondary = true;
	}

	async function saveSecondary() {
		actionLoading = 'secondary';
		try {
			data = await socialInsuranceApi.setSecondaryActivity(
				returnId,
				secondaryDraft,
				secondaryReasonDraft
			);
			toastSuccess('Vedlejší činnost uložena a přehled přepočítán');
			editingSecondary = false;
		} catch (e) {
			toastError(overviewChangeError(e, 'Nepodařilo se uložit vedlejší činnost'));
		} finally {
			actionLoading = null;
		}
	}

	function employmentToDraft(halere: number | null): number | null {
		return halere === null ? null : halere / 100;
	}

	async function saveEmployment() {
		const czk = employmentDraft;
		if (czk === null || czk === undefined || !Number.isInteger(czk) || czk < 0) {
			toastError('Zadejte vyměřovací základ ze zaměstnání jako celé nezáporné číslo v Kč.');
			return;
		}
		await changeEmployment(
			czk * 100,
			'Vyměřovací základ ze zaměstnání uložen a přehled přepočítán'
		);
	}

	async function removeEmployment() {
		await changeEmployment(null, 'Vyměřovací základ ze zaměstnání odebrán a přehled přepočítán');
	}

	async function changeEmployment(halere: number | null, successMessage: string) {
		actionLoading = 'employment';
		try {
			data = await socialInsuranceApi.setEmploymentBase(returnId, halere);
			employmentDraft = employmentToDraft(data.employment_assessment_base);
			toastSuccess(successMessage);
		} catch (e) {
			toastError(overviewChangeError(e, 'Nepodařilo se uložit vyměřovací základ ze zaměstnání'));
		} finally {
			actionLoading = null;
		}
	}

	function overviewChangeError(e: unknown, fallback: string): string {
		if (e instanceof ApiError) {
			if (e.status === 409) {
				return 'Přehled je už podaný, nelze ho změnit.';
			}
			if (e.status === 400 && !e.message.startsWith('API Error ')) {
				return e.message;
			}
		}
		return fallback;
	}

	async function handleGenerateXml() {
		actionLoading = 'generate';
		try {
			data = await socialInsuranceApi.generateXml(returnId);
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se generovat XML');
		} finally {
			actionLoading = null;
		}
	}

	async function handleDownloadXml() {
		actionLoading = 'download';
		try {
			await downloadFile(`/api/v1/social-insurance/${returnId}/xml`);
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se stáhnout XML');
		} finally {
			actionLoading = null;
		}
	}

	function handleMarkFiled() {
		showFileConfirm = true;
	}

	async function confirmMarkFiled() {
		showFileConfirm = false;
		actionLoading = 'filed';
		try {
			data = await socialInsuranceApi.markFiled(returnId);
			toastSuccess('Přehled ČSSZ označen jako podaný');
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
			await socialInsuranceApi.delete(returnId);
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
	<title>{data ? `Sociální pojištění ${data.year}` : 'Sociální pojištění'} - ZFaktury</title>
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
				<h1 class="text-xl font-semibold text-primary">Přehled ČSSZ - {data.year}</h1>
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
				<Button
					variant="secondary"
					onclick={handleGenerateXml}
					disabled={data.status === 'filed' || actionLoading !== null}
				>
					{actionLoading === 'generate' ? 'Generuji...' : 'Generovat XML'}
				</Button>
				{#if data.has_xml}
					<Button variant="secondary" onclick={handleDownloadXml} disabled={actionLoading !== null}>
						{actionLoading === 'download' ? 'Stahuji...' : 'Stáhnout XML'}
					</Button>
				{/if}
				<ISDSSendButton
					kind="social-insurance"
					id={returnId}
					hasXml={data.has_xml}
					filed={data.status === 'filed'}
					disabled={actionLoading !== null}
					onsent={() => isdsHistory?.reload()}
					onattempt={() => isdsHistory?.reload()}
				/>
				{#if data.status !== 'filed'}
					<Button variant="success" onclick={handleMarkFiled} disabled={actionLoading !== null}>
						{actionLoading === 'filed' ? 'Označuji...' : 'Označit jako podané'}
					</Button>
					<Button variant="danger" onclick={handleDelete} disabled={actionLoading !== null}>
						Smazat
					</Button>
				{/if}
			</div>
			<ISDSSubmissionHistory
				bind:this={isdsHistory}
				kind="social-insurance"
				id={returnId}
				ondocumentfiled={loadData}
			/>
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

					<dt class="text-secondary">Vypočtený vyměřovací základ (55 %)</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.assessment_base)}
					</dd>

					{#if data.secondary_months.length > 0}
						<dt class="text-secondary">Vypočtený vyměřovací základ – vedlejší činnost</dt>
						<dd class="text-right font-medium text-primary tabular-nums">
							{formatCZK(data.secondary_assessment_base)}
						</dd>

						<dt class="text-secondary">Rozhodná částka pro vedlejší činnost</dt>
						<dd class="text-right font-medium text-primary tabular-nums">
							{formatCZK(data.secondary_decisive_amount)}
						</dd>
					{/if}

					<dt class="text-secondary">
						{mixedYear
							? 'Minimální vyměřovací základ (ř. 27, nejvyšší ze tří součtů)'
							: 'Minimální vyměřovací základ'}
					</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(mixedYear ? data.final_assessment_base : data.min_assessment_base)}
					</dd>

					<dt class="text-secondary">Výsledný vyměřovací základ</dt>
					<dd class="text-right font-medium text-primary tabular-nums">
						{formatCZK(data.final_assessment_base)}
					</dd>

					{#if data.employment_assessment_base !== null}
						<dt class="text-secondary">Vyměřovací základ ze zaměstnání</dt>
						<dd class="text-right font-medium text-primary tabular-nums">
							{formatCZK(data.employment_assessment_base)}
						</dd>

						<dt class="text-secondary">Vyměřovací základ pro pojistné (ř. 31)</dt>
						<dd class="text-right font-medium text-primary tabular-nums">
							{formatCZK(data.insured_base)}
						</dd>
					{/if}
				</div>

				{#if data.secondary_months.length > 0 && data.secondary_assessment_base === 0}
					<p class="mt-3 text-sm text-secondary">
						Příjem z vedlejší činnosti rozhodné částky nedosáhl, pojistné z ní se neplatí
					</p>
				{/if}

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
								idPrefix="social-months-edit"
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

			<!-- Secondary activity -->
			<Card>
				<div role="group" aria-label="Vedlejší činnost">
					<h2 class="text-base font-semibold text-primary">Vedlejší činnost</h2>
					<p class="mt-1 text-sm text-secondary">
						{#if data.secondary_months.length === 0}
							Žádné měsíce vedlejší činnosti
						{:else}
							{secondaryLabel} – {secondaryReasonLabels[data.secondary_reason]}
						{/if}
					</p>

					{#if data.status !== 'filed'}
						<div class="mt-4 border-t border-border pt-4">
							{#if !editingSecondary}
								<Button
									variant="secondary"
									onclick={startEditingSecondary}
									disabled={actionLoading !== null}
								>
									Upravit
								</Button>
							{:else}
								<ActivityMonthsPicker
									idPrefix="social-secondary-edit"
									legend="Měsíce vedlejší činnosti"
									hint="Měsíce, ve kterých jste podnikali jako vedlejší činnost."
									allowedMonths={data.activity_months}
									required={false}
									value={secondaryDraft}
									onchange={(months) => (secondaryDraft = months)}
									disabled={actionLoading !== null}
								/>
								<div class="mt-3 max-w-md">
									<label for="secondary-reason" class="block text-sm font-medium text-secondary"
										>Důvod</label
									>
									<select
										id="secondary-reason"
										bind:value={secondaryReasonDraft}
										disabled={actionLoading !== null}
										class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
									>
										{#each SECONDARY_REASON_OPTIONS as opt (opt.value)}
											<option value={opt.value}>{opt.label}</option>
										{/each}
									</select>
								</div>
								<p class="mt-3 text-xs text-muted">
									Důvod doložte ČSSZ nejpozději do konce měsíce následujícího po podání přehledu,
									jinak se činnost považuje za hlavní.
								</p>
								<div class="mt-3 flex gap-3">
									<Button
										variant="primary"
										onclick={saveSecondary}
										disabled={actionLoading !== null}
										aria-label="Uložit vedlejší činnost"
									>
										{actionLoading === 'secondary' ? 'Ukládám...' : 'Uložit'}
									</Button>
									<Button
										variant="secondary"
										onclick={() => (editingSecondary = false)}
										disabled={actionLoading !== null}
									>
										Zrušit
									</Button>
								</div>
							{/if}
						</div>
					{/if}
				</div>
			</Card>

			<!-- Employment -->
			<Card>
				<div role="group" aria-label="Zaměstnání">
					<h2 class="text-base font-semibold text-primary">Zaměstnání</h2>
					{#if data.status !== 'filed'}
						<div class="mt-3 max-w-md">
							<label for="employment-base" class="block text-sm font-medium text-secondary">
								Vyměřovací základ ze zaměstnání za rok {data.year} (Kč)
							</label>
							<input
								id="employment-base"
								type="number"
								min="0"
								step="1"
								bind:value={employmentDraft}
								disabled={actionLoading !== null}
								class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
							<p class="mt-1 text-xs text-muted">
								Podle potvrzení zaměstnavatele. Do přehledu se uvede, jen když spolu s podnikáním
								přesáhne maximální vyměřovací základ.
							</p>
						</div>
						<div class="mt-3 flex gap-3">
							<Button
								variant="primary"
								onclick={saveEmployment}
								disabled={actionLoading !== null}
								aria-label="Uložit vyměřovací základ ze zaměstnání"
							>
								{actionLoading === 'employment' ? 'Ukládám...' : 'Uložit'}
							</Button>
							<Button
								variant="secondary"
								aria-label="Odebrat vyměřovací základ ze zaměstnání"
								onclick={removeEmployment}
								disabled={actionLoading !== null || data.employment_assessment_base === null}
							>
								Odebrat
							</Button>
						</div>
					{:else}
						<p class="mt-1 text-sm text-secondary">
							{data.employment_assessment_base === null
								? 'Vyměřovací základ ze zaměstnání nebyl zadán'
								: `Vyměřovací základ ze zaměstnání: ${formatCZK(data.employment_assessment_base)}`}
						</p>
					{/if}
				</div>
			</Card>

			<!-- Insurance -->
			<Card>
				<h2 class="text-base font-semibold text-primary">
					Pojistné <HelpTip topic="prehled-cssz" {taxConstants} />
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
					periods={data.advance_periods}
					warnings={data.warnings}
					reportedPrepay={data.new_monthly_prepay}
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
	message="Opravdu chcete označit přehled ČSSZ jako podaný?"
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
