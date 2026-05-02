<script lang="ts">
	import {
		employmentApi,
		type EmploymentCertificate,
		type CertificateType,
		type ContractType
	} from '$lib/api/client';
	import Button from '$lib/ui/Button.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';

	interface Props {
		open: boolean;
		year: number;
		draft: Partial<EmploymentCertificate>;
		onclose: () => void;
		onsaved: (cert: EmploymentCertificate, confirmed: boolean) => void;
	}

	let { open = $bindable(), year, draft, onclose, onsaved }: Props = $props();

	// Form state — initialised from draft via $effect when modal opens.
	let id = $state<number | undefined>(undefined);
	let documentId = $state<number | undefined>(undefined);
	let certificateType = $state<CertificateType>('advance');
	let employerName = $state('');
	let employerIco = $state('');
	let employerAddress = $state('');
	let contractType = $state<ContractType>('dpc');
	let periodFrom = $state('');
	let periodTo = $state('');
	let grossIncome = $state(0);
	let incomeWithoutAdvance = $state(0);
	let foreignTaxPaid = $state(0);
	let advanceTaxWithheld = $state(0);
	let annualSettlementRefund = $state(0);
	let monthlyBonusPaid = $state(0);
	let withheldFinalTax = $state(0);
	let includeWithholdingInDap = $state(false);
	let notes = $state('');
	let confidence = $state<number | undefined>(undefined);

	let saving = $state(false);
	let validationError = $state<string | null>(null);

	const inputClass =
		'w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none';

	// Re-init form whenever the modal opens with a new draft.
	$effect(() => {
		if (!open) return;
		id = draft.id;
		documentId = draft.document_id;
		certificateType = draft.certificate_type ?? 'advance';
		employerName = draft.employer_name ?? '';
		employerIco = draft.employer_ico ?? '';
		employerAddress = draft.employer_address ?? '';
		contractType = draft.contract_type ?? 'dpc';
		periodFrom = draft.period_from ?? `${year}-01-01`;
		periodTo = draft.period_to ?? `${year}-12-31`;
		grossIncome = draft.gross_income_czk ?? 0;
		incomeWithoutAdvance = draft.income_without_advance_czk ?? 0;
		foreignTaxPaid = draft.foreign_tax_paid_czk ?? 0;
		advanceTaxWithheld = draft.advance_tax_withheld_czk ?? 0;
		annualSettlementRefund = draft.annual_settlement_refund_czk ?? 0;
		monthlyBonusPaid = draft.monthly_bonus_paid_czk ?? 0;
		withheldFinalTax = draft.withheld_final_tax_czk ?? 0;
		includeWithholdingInDap = draft.include_withholding_in_dap ?? false;
		notes = draft.notes ?? '';
		confidence = draft.confidence;
		validationError = null;
	});

	function validIco(ico: string): boolean {
		return /^\d{8}$/.test(ico.trim());
	}

	function validate(): string | null {
		if (!employerName.trim()) return 'Vyplňte název zaměstnavatele.';
		if (!validIco(employerIco)) return 'IČO musí být 8 číslic.';
		if (!periodFrom || !periodTo) return 'Vyplňte období od a do.';
		if (periodFrom > periodTo) return 'Datum "od" nesmí být pozdější než "do".';
		const yearStr = String(year);
		if (!periodFrom.startsWith(yearStr) || !periodTo.startsWith(yearStr)) {
			return `Období musí být v rámci roku ${year}.`;
		}
		if (grossIncome < 0) return 'Úhrn příjmů nesmí být záporný.';
		if (certificateType === 'advance') {
			if (annualSettlementRefund > advanceTaxWithheld) {
				return 'Vrácený přeplatek z RZ nesmí být vyšší než sražené zálohy.';
			}
		}
		return null;
	}

	async function save(confirm: boolean) {
		const err = validate();
		if (err) {
			validationError = err;
			return;
		}
		validationError = null;
		saving = true;
		try {
			const payload: Partial<EmploymentCertificate> = {
				year,
				document_id: documentId,
				certificate_type: certificateType,
				employer_name: employerName.trim(),
				employer_ico: employerIco.trim(),
				employer_address: employerAddress.trim() || undefined,
				contract_type: contractType,
				period_from: periodFrom,
				period_to: periodTo,
				gross_income_czk: Number(grossIncome) || 0,
				income_without_advance_czk: Number(incomeWithoutAdvance) || 0,
				foreign_tax_paid_czk: Number(foreignTaxPaid) || 0,
				advance_tax_withheld_czk: Number(advanceTaxWithheld) || 0,
				annual_settlement_refund_czk: Number(annualSettlementRefund) || 0,
				monthly_bonus_paid_czk: Number(monthlyBonusPaid) || 0,
				withheld_final_tax_czk: Number(withheldFinalTax) || 0,
				include_withholding_in_dap:
					certificateType === 'withholding' ? includeWithholdingInDap : false,
				notes: notes.trim() || undefined
			};
			let saved: EmploymentCertificate;
			if (id) {
				saved = await employmentApi.updateCertificate(id, payload);
			} else {
				saved = await employmentApi.createCertificate(payload);
			}
			if (confirm) {
				await employmentApi.confirmCertificate(saved.id);
			}
			onsaved(saved, confirm);
		} catch (e) {
			validationError = e instanceof Error ? e.message : 'Uložení selhalo.';
		} finally {
			saving = false;
		}
	}

	function close() {
		if (saving) return;
		open = false;
		onclose();
	}
</script>

{#if open}
	<div
		class="fixed inset-0 z-50 bg-black/60"
		role="presentation"
		onclick={close}
		data-testid="employment-editor-backdrop"
	></div>
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 overflow-y-auto"
		role="dialog"
		aria-modal="true"
		aria-labelledby="employment-editor-title"
	>
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
		<div
			class="w-full max-w-3xl rounded-xl border border-border bg-surface p-6 shadow-xl my-8"
			onclick={(e) => e.stopPropagation()}
		>
			<div class="flex items-center justify-between">
				<h2 id="employment-editor-title" class="text-lg font-semibold text-primary">
					{id ? 'Upravit Potvrzení' : 'Nové Potvrzení'}
					<HelpTip topic="zavisla-cinnost-s6" />
				</h2>
				{#if confidence !== undefined}
					<span
						data-testid="ocr-confidence-badge"
						class="rounded-md bg-info-bg px-2 py-1 text-xs font-medium text-info"
					>
						OCR jistota: {Math.round(confidence * 100)} %
					</span>
				{/if}
			</div>

			{#if validationError}
				<div
					role="alert"
					class="mt-4 rounded-lg border border-danger/20 bg-danger-bg p-3 text-sm text-danger"
				>
					{validationError}
				</div>
			{/if}

			<!-- Certificate type toggle -->
			<div class="mt-4">
				<label class="block text-xs font-medium text-secondary" for="cert-type">
					Typ Potvrzení
				</label>
				<select
					id="cert-type"
					bind:value={certificateType}
					data-testid="certificate-type-select"
					class="mt-1 {inputClass}"
				>
					<option value="advance">Zálohové (vzor 33)</option>
					<option value="withholding">Srážkové (vzor 12)</option>
				</select>
			</div>

			<!-- Section 1: Employer -->
			<section class="mt-6">
				<h3 class="text-sm font-semibold text-primary">Identifikace plátce</h3>
				<div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2">
					<div>
						<label class="block text-xs font-medium text-secondary" for="employer-name">
							Název zaměstnavatele
						</label>
						<input
							id="employer-name"
							type="text"
							bind:value={employerName}
							required
							data-testid="employer-name-input"
							class="mt-1 {inputClass}"
						/>
					</div>
					<div>
						<label class="block text-xs font-medium text-secondary" for="employer-ico">
							IČO (8 číslic)
						</label>
						<input
							id="employer-ico"
							type="text"
							bind:value={employerIco}
							required
							maxlength={8}
							data-testid="employer-ico-input"
							class="mt-1 {inputClass}"
						/>
					</div>
					<div class="md:col-span-2">
						<label class="block text-xs font-medium text-secondary" for="employer-address">
							Adresa
						</label>
						<input
							id="employer-address"
							type="text"
							bind:value={employerAddress}
							class="mt-1 {inputClass}"
						/>
					</div>
				</div>
			</section>

			<!-- Section 2: Period -->
			<section class="mt-6">
				<h3 class="text-sm font-semibold text-primary">
					Období <HelpTip topic="dpc-dpp-hpp" />
				</h3>
				<div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-3">
					<div>
						<label class="block text-xs font-medium text-secondary" for="period-from">Od</label>
						<input
							id="period-from"
							type="date"
							bind:value={periodFrom}
							required
							data-testid="period-from-input"
							class="mt-1 {inputClass}"
						/>
					</div>
					<div>
						<label class="block text-xs font-medium text-secondary" for="period-to">Do</label>
						<input
							id="period-to"
							type="date"
							bind:value={periodTo}
							required
							data-testid="period-to-input"
							class="mt-1 {inputClass}"
						/>
					</div>
					<div>
						<label class="block text-xs font-medium text-secondary" for="contract-type">
							Typ smlouvy
						</label>
						<select
							id="contract-type"
							bind:value={contractType}
							data-testid="contract-type-select"
							class="mt-1 {inputClass}"
						>
							<option value="dpc">DPČ</option>
							<option value="dpp">DPP</option>
							<option value="hpp">HPP</option>
							<option value="other">Jiné</option>
						</select>
					</div>
				</div>
				<div class="mt-3">
					<label class="block text-xs font-medium text-secondary" for="cert-notes">Poznámka</label>
					<textarea id="cert-notes" bind:value={notes} rows={2} class="mt-1 {inputClass}"
					></textarea>
				</div>
			</section>

			<!-- Section 3: Advance amounts -->
			{#if certificateType === 'advance'}
				<section class="mt-6" data-testid="advance-section">
					<h3 class="text-sm font-semibold text-primary">
						Částky z Potvrzení (vzor 33) <HelpTip topic="potvrzeni-zalohove" />
					</h3>
					<div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2">
						<div>
							<label class="block text-xs font-medium text-secondary" for="gross-income">
								Úhrn zúčtovaných příjmů (ř. 2 + ř. 4) → ř. 31 DAP
								<HelpTip topic="radek-31-prijmy-s6" />
							</label>
							<input
								id="gross-income"
								type="number"
								min="0"
								step="1"
								bind:value={grossIncome}
								data-testid="gross-income-input"
								class="mt-1 {inputClass}"
							/>
						</div>
						<div>
							<label class="block text-xs font-medium text-secondary" for="income-no-advance">
								Z toho příjmy bez záloh dle §38h → ř. 35 DAP
							</label>
							<input
								id="income-no-advance"
								type="number"
								min="0"
								step="1"
								bind:value={incomeWithoutAdvance}
								class="mt-1 {inputClass}"
							/>
						</div>
						<div>
							<label class="block text-xs font-medium text-secondary" for="foreign-tax">
								Daň zaplacená v zahraničí (§6 odst. 13) → ř. 33 DAP
								<HelpTip topic="radek-33-zahranicni-dan" />
							</label>
							<input
								id="foreign-tax"
								type="number"
								min="0"
								step="1"
								bind:value={foreignTaxPaid}
								class="mt-1 {inputClass}"
							/>
						</div>
						<div>
							<label class="block text-xs font-medium text-secondary" for="advance-withheld">
								Sražené zálohy po slevách (ř. 8) → ř. 84 DAP
								<HelpTip topic="radek-84-srazene-zalohy" />
							</label>
							<input
								id="advance-withheld"
								type="number"
								min="0"
								step="1"
								bind:value={advanceTaxWithheld}
								data-testid="advance-withheld-input"
								class="mt-1 {inputClass}"
							/>
						</div>
						<div>
							<label class="block text-xs font-medium text-secondary" for="rz-refund">
								Vrácený přeplatek z RZ
								<HelpTip topic="rocni-zuctovani-rz" />
							</label>
							<input
								id="rz-refund"
								type="number"
								min="0"
								step="1"
								bind:value={annualSettlementRefund}
								class="mt-1 {inputClass}"
							/>
						</div>
						<div>
							<label class="block text-xs font-medium text-secondary" for="monthly-bonus">
								Úhrn vyplacených měsíčních bonusů (ř. 5 + ř. 13) → ř. 89 DAP
								<HelpTip topic="radek-89-vyplacene-bonusy" />
							</label>
							<input
								id="monthly-bonus"
								type="number"
								min="0"
								step="1"
								bind:value={monthlyBonusPaid}
								class="mt-1 {inputClass}"
							/>
						</div>
					</div>
				</section>
			{:else}
				<!-- Withholding -->
				<section class="mt-6" data-testid="withholding-section">
					<h3 class="text-sm font-semibold text-primary">
						Částky z Potvrzení (vzor 12) <HelpTip topic="potvrzeni-srazkove" />
					</h3>
					<div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2">
						<div>
							<label class="block text-xs font-medium text-secondary" for="gross-income-w">
								Úhrn vyplacených příjmů (ř. 2)
							</label>
							<input
								id="gross-income-w"
								type="number"
								min="0"
								step="1"
								bind:value={grossIncome}
								class="mt-1 {inputClass}"
							/>
						</div>
						<div>
							<label class="block text-xs font-medium text-secondary" for="withheld-final">
								Sražená daň zvláštní sazbou → ř. 87 DAP
								<HelpTip topic="radek-87-srazena-dan" />
							</label>
							<input
								id="withheld-final"
								type="number"
								min="0"
								step="1"
								bind:value={withheldFinalTax}
								class="mt-1 {inputClass}"
							/>
						</div>
					</div>
					<div class="mt-3">
						<label class="inline-flex items-start gap-2 text-sm">
							<input
								type="checkbox"
								bind:checked={includeWithholdingInDap}
								data-testid="include-withholding-checkbox"
								class="mt-0.5"
							/>
							<span>
								<span class="font-medium text-primary">
									Zahrnout do daňového přiznání (§ 36 odst. 7 ZDP)
								</span>
								<HelpTip topic="srazkova-do-dap" />
								<span class="mt-1 block text-xs text-tertiary">
									Pokud zaškrtnete, musíte zahrnout veškeré srážkově zdaněné příjmy z daného typu (§
									38g odst. 6).
								</span>
							</span>
						</label>
					</div>
				</section>
			{/if}

			<!-- Actions -->
			<div class="mt-8 flex flex-wrap justify-end gap-3 border-t border-border pt-4">
				<Button variant="ghost" onclick={close} disabled={saving}>Zrušit</Button>
				<Button variant="secondary" onclick={() => save(false)} disabled={saving}>
					{saving ? 'Ukládám...' : 'Uložit jako koncept'}
				</Button>
				<Button variant="primary" onclick={() => save(true)} disabled={saving}>
					{saving ? 'Ukládám...' : 'Uložit a potvrdit'}
				</Button>
			</div>
		</div>
	</div>
{/if}
