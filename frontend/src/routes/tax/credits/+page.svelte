<script lang="ts">
	import { onMount } from 'svelte';
	import {
		taxCreditsApi,
		taxDeductionsApi,
		type TaxCreditsSummary,
		type TaxDeduction,
		type TaxDeductionDocument,
		type TaxExtractionResult,
		type TaxConstants
	} from '$lib/api/client';
	import { loadTaxConstants } from '$lib/data/tax-constants.svelte';
	import { formatCZK, fromHalere, toHalere } from '$lib/utils/money';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import Input from '$lib/ui/Input.svelte';
	import Select from '$lib/ui/Select.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';

	let selectedYear = $state(new Date().getFullYear() - 1);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let saving = $state(false);
	let taxConstants = $state<TaxConstants | null>(null);

	let summary = $state<TaxCreditsSummary | null>(null);
	let deductions = $state<TaxDeduction[]>([]);

	// Spouse form
	let spouseName = $state('');
	let spouseBirthNumber = $state('');
	let spouseIncome = $state(0);
	let spouseZtp = $state(false);
	let spouseMonths = $state(12);
	let showSpouseForm = $state(false);

	// Personal form
	let isStudent = $state(false);
	let studentMonths = $state(12);
	let disabilityLevel = $state(0);

	// Child form
	let showChildForm = $state(false);
	let editingChildId = $state<number | null>(null);
	let childName = $state('');
	let childBirthNumber = $state('');
	let childOrder = $state(1);
	let childMonths = $state(12);
	let childZtp = $state(false);

	// Deduction form
	let showDeductionForm = $state(false);
	let editingDeductionId = $state<number | null>(null);
	let deductionCategory = $state('mortgage');
	let deductionDescription = $state('');
	let deductionAmount = $state(0);

	const categoryLabels: Record<string, string> = {
		mortgage: 'Úroky z hypotéky',
		life_insurance: 'Životní pojištění',
		pension: 'Penzijní spoření',
		donation: 'Dary',
		union_dues: 'Odborové příspěvky'
	};

	async function loadData() {
		loading = true;
		error = null;
		try {
			const [s, d, tc] = await Promise.all([
				taxCreditsApi.getSummary(selectedYear),
				taxDeductionsApi.list(selectedYear),
				loadTaxConstants(selectedYear)
			]);
			summary = s;
			deductions = d ?? [];
			taxConstants = tc;

			// Populate forms from loaded data
			if (summary?.spouse) {
				spouseName = summary.spouse.spouse_name;
				spouseBirthNumber = summary.spouse.spouse_birth_number;
				spouseIncome = fromHalere(summary.spouse.spouse_income);
				spouseZtp = summary.spouse.spouse_ztp;
				spouseMonths = summary.spouse.months_claimed;
				showSpouseForm = true;
			} else {
				resetSpouseForm();
			}
			if (summary?.personal) {
				isStudent = summary.personal.is_student;
				studentMonths = summary.personal.student_months;
				disabilityLevel = summary.personal.disability_level;
			} else {
				isStudent = false;
				studentMonths = 12;
				disabilityLevel = 0;
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst data';
		} finally {
			loading = false;
		}
	}

	function resetSpouseForm() {
		spouseName = '';
		spouseBirthNumber = '';
		spouseIncome = 0;
		spouseZtp = false;
		spouseMonths = 12;
		showSpouseForm = false;
	}

	function resetChildForm() {
		showChildForm = false;
		editingChildId = null;
		childName = '';
		childBirthNumber = '';
		childOrder = 1;
		childMonths = 12;
		childZtp = false;
	}

	function resetDeductionForm() {
		showDeductionForm = false;
		editingDeductionId = null;
		deductionCategory = 'mortgage';
		deductionDescription = '';
		deductionAmount = 0;
	}

	// --- Actions ---

	async function saveSpouse() {
		saving = true;
		error = null;
		try {
			await taxCreditsApi.upsertSpouse(selectedYear, {
				spouse_name: spouseName,
				spouse_birth_number: spouseBirthNumber,
				spouse_income: toHalere(spouseIncome),
				spouse_ztp: spouseZtp,
				months_claimed: spouseMonths
			});
			await loadData();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba při ukládání';
		} finally {
			saving = false;
		}
	}

	async function deleteSpouse() {
		saving = true;
		error = null;
		try {
			await taxCreditsApi.deleteSpouse(selectedYear);
			resetSpouseForm();
			await loadData();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba při mazání';
		} finally {
			saving = false;
		}
	}

	async function savePersonal() {
		saving = true;
		error = null;
		try {
			await taxCreditsApi.upsertPersonal(selectedYear, {
				is_student: isStudent,
				student_months: isStudent ? studentMonths : 0,
				disability_level: disabilityLevel
			});
			await loadData();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba při ukládání';
		} finally {
			saving = false;
		}
	}

	async function saveChild() {
		saving = true;
		error = null;
		try {
			const data = {
				child_name: childName,
				birth_number: childBirthNumber,
				child_order: childOrder,
				months_claimed: childMonths,
				ztp: childZtp
			};
			if (editingChildId) {
				await taxCreditsApi.updateChild(selectedYear, editingChildId, data);
			} else {
				await taxCreditsApi.createChild(selectedYear, data);
			}
			resetChildForm();
			await loadData();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba při ukládání';
		} finally {
			saving = false;
		}
	}

	async function deleteChild(id: number) {
		saving = true;
		error = null;
		try {
			await taxCreditsApi.deleteChild(selectedYear, id);
			await loadData();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba při mazání';
		} finally {
			saving = false;
		}
	}

	function editChild(child: { id: number; child_name: string; birth_number: string; child_order: number; months_claimed: number; ztp: boolean }) {
		editingChildId = child.id;
		childName = child.child_name;
		childBirthNumber = child.birth_number;
		childOrder = child.child_order;
		childMonths = child.months_claimed;
		childZtp = child.ztp;
		showChildForm = true;
	}

	async function saveDeduction() {
		saving = true;
		error = null;
		try {
			const data = {
				category: deductionCategory,
				description: deductionDescription,
				claimed_amount: toHalere(deductionAmount)
			};
			if (editingDeductionId) {
				await taxDeductionsApi.update(selectedYear, editingDeductionId, data);
			} else {
				await taxDeductionsApi.create(selectedYear, data);
			}
			resetDeductionForm();
			await loadData();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba při ukládání';
		} finally {
			saving = false;
		}
	}

	async function deleteDeduction(id: number) {
		saving = true;
		error = null;
		try {
			await taxDeductionsApi.delete(selectedYear, id);
			await loadData();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba při mazání';
		} finally {
			saving = false;
		}
	}

	function editDeduction(ded: TaxDeduction) {
		editingDeductionId = ded.id;
		deductionCategory = ded.category;
		deductionDescription = ded.description;
		deductionAmount = fromHalere(ded.claimed_amount);
		showDeductionForm = true;
	}

	async function uploadDocument(deductionId: number) {
		const input = document.createElement('input');
		input.type = 'file';
		input.accept = 'image/*,application/pdf';
		input.onchange = async () => {
			const file = input.files?.[0];
			if (!file) return;
			saving = true;
			error = null;
			try {
				await taxDeductionsApi.uploadDocument(selectedYear, deductionId, file);
				await loadData();
			} catch (e) {
				error = e instanceof Error ? e.message : 'Chyba při nahrávání';
			} finally {
				saving = false;
			}
		};
		input.click();
	}

	async function extractAmount(docId: number) {
		saving = true;
		error = null;
		try {
			const result: TaxExtractionResult = await taxDeductionsApi.extractDocument(docId);
			if (result.amount_czk > 0) {
				// Reload to see updated extraction
				await loadData();
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba při extrakci';
		} finally {
			saving = false;
		}
	}

	async function deleteDocument(docId: number) {
		saving = true;
		error = null;
		try {
			await taxDeductionsApi.deleteDocument(docId);
			await loadData();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba při mazání';
		} finally {
			saving = false;
		}
	}

	async function copyFromPreviousYear() {
		saving = true;
		error = null;
		try {
			await taxCreditsApi.copyFromYear(selectedYear, selectedYear - 1);
			await loadData();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba při kopírování';
		} finally {
			saving = false;
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
</script>

<svelte:head>
	<title>Slevy a odpočty {selectedYear} - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-4xl">
	<h1 class="text-xl font-semibold text-primary">Slevy a odpočty za rok {selectedYear}</h1>

	<!-- Year selector + copy -->
	<div class="mt-4 flex items-center gap-3">
		<Button variant="ghost" size="sm" onclick={() => { selectedYear--; }} title="Předchozí rok" aria-label="Předchozí rok">
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
			</svg>
		</Button>
		<span class="min-w-[4rem] text-center text-xl font-semibold text-primary tabular-nums">{selectedYear}</span>
		<Button variant="ghost" size="sm" onclick={() => { selectedYear++; }} title="Následující rok" aria-label="Následující rok">
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
			</svg>
		</Button>
		<div class="ml-auto">
			<Button variant="secondary" size="sm" onclick={copyFromPreviousYear} disabled={saving}>
				Kopírovat z {selectedYear - 1}
			</Button>
		</div>
	</div>

	<ErrorAlert {error} class="mt-4" />

	{#if loading}
		<LoadingSpinner class="mt-8 p-12" />
	{:else}
		<div class="mt-6 space-y-6">
			<!-- 1. Personal credits -->
			<Card>
				<div class="flex items-center justify-between">
					<h2 class="text-base font-semibold text-primary">Osobní slevy <HelpTip topic="sleva-na-poplatnika" {taxConstants} /></h2>
					<Button variant="primary" size="sm" onclick={savePersonal} disabled={saving}>Uložit</Button>
				</div>
				<div class="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
					<label class="flex items-center gap-2 text-sm text-primary">
						<input type="checkbox" bind:checked={isStudent} class="rounded border-border" />
						Student
					</label>
					{#if isStudent}
						<div>
							<span class="text-xs text-tertiary">Počet měsíců</span>
							<Select value={studentMonths} onchange={(e: Event) => { studentMonths = Number((e.currentTarget as HTMLSelectElement).value); }}>
								{#each Array.from({ length: 12 }, (_, i) => i + 1) as m}
									<option value={m}>{m}</option>
								{/each}
							</Select>
						</div>
					{/if}
					<div>
						<span class="text-xs text-tertiary">Invalidita</span>
						<Select value={disabilityLevel} onchange={(e: Event) => { disabilityLevel = Number((e.currentTarget as HTMLSelectElement).value); }}>
							<option value={0}>Žádná</option>
							<option value={1}>1. a 2. stupeň</option>
							<option value={2}>3. stupeň</option>
							<option value={3}>Držitel ZTP/P</option>
						</Select>
					</div>
				</div>
				{#if summary?.personal}
					<div class="mt-3 flex gap-6 text-sm text-tertiary">
						{#if summary.personal.credit_student > 0}
							<span>Sleva student: <strong class="text-primary">{formatCZK(summary.personal.credit_student)}</strong></span>
						{/if}
						{#if summary.personal.credit_disability > 0}
							<span>Sleva invalidita: <strong class="text-primary">{formatCZK(summary.personal.credit_disability)}</strong></span>
						{/if}
					</div>
				{/if}
			</Card>

			<!-- 2. Spouse -->
			<Card>
				<div class="flex items-center justify-between">
					<h2 class="text-base font-semibold text-primary">Manžel/ka <HelpTip topic="sleva-na-manzela" {taxConstants} /></h2>
					{#if !showSpouseForm}
						<Button variant="primary" size="sm" onclick={() => (showSpouseForm = true)}>Přidat</Button>
					{/if}
				</div>
				{#if showSpouseForm}
					<div class="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
						<div>
							<span class="text-xs text-tertiary">Jméno</span>
							<Input value={spouseName} oninput={(e: Event) => { spouseName = (e.currentTarget as HTMLInputElement).value; }} placeholder="Jméno a příjmení" />
						</div>
						<div>
							<span class="text-xs text-tertiary">Rodné číslo</span>
							<Input value={spouseBirthNumber} oninput={(e: Event) => { spouseBirthNumber = (e.currentTarget as HTMLInputElement).value; }} placeholder="000000/0000" />
						</div>
						<div>
							<span class="text-xs text-tertiary">Roční příjem (CZK)</span>
							<Input type="number" value={spouseIncome} oninput={(e: Event) => { spouseIncome = Number((e.currentTarget as HTMLInputElement).value); }} step="1" />
						</div>
						<div>
							<span class="text-xs text-tertiary">Měsíců <HelpTip topic="mesice-proporcializace" {taxConstants} /></span>
							<Select value={spouseMonths} onchange={(e: Event) => { spouseMonths = Number((e.currentTarget as HTMLSelectElement).value); }}>
								{#each Array.from({ length: 12 }, (_, i) => i + 1) as m}
									<option value={m}>{m}</option>
								{/each}
							</Select>
						</div>
						<label class="flex items-center gap-2 text-sm text-primary">
							<input type="checkbox" bind:checked={spouseZtp} class="rounded border-border" />
							ZTP/P <HelpTip topic="ztpp" {taxConstants} />
						</label>
					</div>
					{#if summary?.spouse}
						<div class="mt-3 text-sm text-tertiary">
							Sleva: <strong class="text-primary">{formatCZK(summary.spouse.credit_amount)}</strong>
							{#if spouseIncome >= 68000}
								<span class="ml-2 text-warning">(příjem >= 68 000 CZK, sleva se neuplatní)</span>
							{/if}
						</div>
					{/if}
					<div class="mt-4 flex gap-2">
						<Button variant="primary" size="sm" onclick={saveSpouse} disabled={saving}>Uložit</Button>
						<Button variant="danger" size="sm" onclick={deleteSpouse} disabled={saving}>Odebrat</Button>
					</div>
				{:else}
					<p class="mt-2 text-sm text-tertiary">Neuplatňováno</p>
				{/if}
			</Card>

			<!-- 3. Children -->
			<Card>
				<div class="flex items-center justify-between">
					<h2 class="text-base font-semibold text-primary">Děti <HelpTip topic="zvyhodneni-na-deti" {taxConstants} /></h2>
					<Button variant="primary" size="sm" onclick={() => { resetChildForm(); showChildForm = true; }}>Přidat dítě</Button>
				</div>
				{#if summary?.children && summary.children.length > 0}
					<div class="mt-4 space-y-2">
						{#each summary.children as child (child.id)}
							<div class="flex items-center justify-between rounded-lg border border-border p-3">
								<div>
									<span class="text-sm font-medium text-primary">{child.child_name || `Dítě ${child.child_order}`}</span>
									<span class="ml-2 text-xs text-tertiary">
										{child.child_order}. dítě, {child.months_claimed} mes.
										{#if child.ztp}<span class="text-accent">ZTP</span>{/if}
									</span>
									<span class="ml-2 text-sm font-medium text-primary">{formatCZK(child.credit_amount)}</span>
								</div>
								<div class="flex gap-1">
									<Button variant="ghost" size="sm" onclick={() => editChild(child)}>Upravit</Button>
									<Button variant="danger" size="sm" onclick={() => deleteChild(child.id)}>Smazat</Button>
								</div>
							</div>
						{/each}
					</div>
					<div class="mt-2 text-sm text-tertiary">
						Celkem zvýhodnění: <strong class="text-primary">{formatCZK(summary.total_child_benefit)}</strong>
					</div>
				{:else}
					<p class="mt-2 text-sm text-tertiary">Žádné děti</p>
				{/if}
				{#if showChildForm}
					<div class="mt-4 rounded-lg border border-border-subtle bg-elevated p-4">
						<h3 class="text-sm font-medium text-primary">{editingChildId ? 'Upravit dítě' : 'Přidat dítě'}</h3>
						<div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2">
							<div>
								<span class="text-xs text-tertiary">Jméno</span>
								<Input value={childName} oninput={(e: Event) => { childName = (e.currentTarget as HTMLInputElement).value; }} placeholder="Jméno dítěte" />
							</div>
							<div>
								<span class="text-xs text-tertiary">Rodné číslo</span>
								<Input value={childBirthNumber} oninput={(e: Event) => { childBirthNumber = (e.currentTarget as HTMLInputElement).value; }} placeholder="000000/0000" />
							</div>
							<div>
								<span class="text-xs text-tertiary">Pořadí</span>
								<Select value={childOrder} onchange={(e: Event) => { childOrder = Number((e.currentTarget as HTMLSelectElement).value); }}>
									<option value={1}>1. dítě</option>
									<option value={2}>2. dítě</option>
									<option value={3}>3. a další</option>
								</Select>
							</div>
							<div>
								<span class="text-xs text-tertiary">Měsíců <HelpTip topic="mesice-proporcializace" {taxConstants} /></span>
								<Select value={childMonths} onchange={(e: Event) => { childMonths = Number((e.currentTarget as HTMLSelectElement).value); }}>
									{#each Array.from({ length: 12 }, (_, i) => i + 1) as m}
										<option value={m}>{m}</option>
									{/each}
								</Select>
							</div>
							<label class="flex items-center gap-2 text-sm text-primary">
								<input type="checkbox" bind:checked={childZtp} class="rounded border-border" />
								ZTP/P <HelpTip topic="ztpp" {taxConstants} />
							</label>
						</div>
						<div class="mt-3 flex gap-2">
							<Button variant="primary" size="sm" onclick={saveChild} disabled={saving}>Uložit</Button>
							<Button variant="ghost" size="sm" onclick={resetChildForm}>Zrušit</Button>
						</div>
					</div>
				{/if}
			</Card>

			<!-- 4. Deductions -->
			<Card>
				<div class="flex items-center justify-between">
					<h2 class="text-base font-semibold text-primary">Nezdanitelné části (odpočty) <HelpTip topic="nezdanitelne-odpocty" {taxConstants} /></h2>
					<Button variant="primary" size="sm" onclick={() => { resetDeductionForm(); showDeductionForm = true; }}>Přidat odpočet</Button>
				</div>
				{#if deductions.length > 0}
					<div class="mt-4 space-y-3">
						{#each deductions as ded (ded.id)}
							<div class="rounded-lg border border-border p-3">
								<div class="flex items-center justify-between">
									<div>
										<span class="text-xs font-medium uppercase text-accent">{categoryLabels[ded.category] ?? ded.category}</span>
										{#if ded.description}
											<span class="ml-2 text-sm text-tertiary">{ded.description}</span>
										{/if}
									</div>
									<div class="flex gap-1">
										<Button variant="ghost" size="sm" onclick={() => editDeduction(ded)}>Upravit</Button>
										<Button variant="danger" size="sm" onclick={() => deleteDeduction(ded.id)}>Smazat</Button>
									</div>
								</div>
								<div class="mt-2 flex gap-6 text-sm">
									<span class="text-tertiary">Uplatňováno: <strong class="text-primary">{formatCZK(ded.claimed_amount)}</strong></span>
									{#if ded.max_amount > 0}
										<span class="text-tertiary">Max: {formatCZK(ded.max_amount)}</span>
									{/if}
									<span class="text-tertiary">Uznáno: <strong class="text-primary">{formatCZK(ded.allowed_amount)}</strong></span>
								</div>
								<!-- Documents -->
								{#if ded.documents && ded.documents.length > 0}
									<div class="mt-2 space-y-1">
										{#each ded.documents as doc (doc.id)}
											<div class="flex items-center gap-2 text-xs text-tertiary">
												<a href={taxDeductionsApi.downloadDocument(doc.id)} class="text-accent hover:underline" target="_blank">{doc.filename}</a>
												{#if doc.extracted_amount > 0}
													<span class="rounded bg-success-bg px-1.5 py-0.5 text-success">
														Extrahováno: {formatCZK(doc.extracted_amount)} ({Math.round(doc.confidence * 100)}%)
													</span>
												{/if}
												<Button variant="ghost" size="sm" onclick={() => extractAmount(doc.id)} disabled={saving}>Extrahovat</Button>
												<Button variant="danger" size="sm" onclick={() => deleteDocument(doc.id)} disabled={saving}>Smazat</Button>
											</div>
										{/each}
									</div>
								{/if}
								<div class="mt-2">
									<Button variant="secondary" size="sm" onclick={() => uploadDocument(ded.id)} disabled={saving}>Nahrát doklad</Button>
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<p class="mt-2 text-sm text-tertiary">Žádné odpočty</p>
				{/if}
				{#if showDeductionForm}
					<div class="mt-4 rounded-lg border border-border-subtle bg-elevated p-4">
						<h3 class="text-sm font-medium text-primary">{editingDeductionId ? 'Upravit odpočet' : 'Přidat odpočet'}</h3>
						<div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2">
							<div>
								<span class="text-xs text-tertiary">Kategorie</span>
								<Select value={deductionCategory} onchange={(e: Event) => { deductionCategory = (e.currentTarget as HTMLSelectElement).value; }}>
									{#each Object.entries(categoryLabels) as [key, label]}
										<option value={key}>{label}</option>
									{/each}
								</Select>
							</div>
							<div>
								<span class="text-xs text-tertiary">Popis</span>
								<Input value={deductionDescription} oninput={(e: Event) => { deductionDescription = (e.currentTarget as HTMLInputElement).value; }} placeholder="Název/číslo smlouvy" />
							</div>
							<div>
								<span class="text-xs text-tertiary">Částka (CZK)</span>
								<Input type="number" value={deductionAmount} oninput={(e: Event) => { deductionAmount = Number((e.currentTarget as HTMLInputElement).value); }} step="0.01" />
							</div>
						</div>
						<div class="mt-3 flex gap-2">
							<Button variant="primary" size="sm" onclick={saveDeduction} disabled={saving}>Uložit</Button>
							<Button variant="ghost" size="sm" onclick={resetDeductionForm}>Zrušit</Button>
						</div>
					</div>
				{/if}
			</Card>

			<!-- Summary -->
			{#if summary}
				<Card>
					<h2 class="text-base font-semibold text-primary">Souhrn</h2>
					<div class="mt-3 space-y-1 text-sm">
						<div class="flex justify-between">
							<span class="text-tertiary">Celkové slevy (bez základní)</span>
							<strong class="text-primary">{formatCZK(summary.total_credits)}</strong>
						</div>
						<div class="flex justify-between">
							<span class="text-tertiary">Daňové zvýhodnění na děti</span>
							<strong class="text-primary">{formatCZK(summary.total_child_benefit)}</strong>
						</div>
						{#if deductions.length > 0}
							{@const totalDeductions = deductions.reduce((sum, d) => sum + d.allowed_amount, 0)}
							<div class="flex justify-between">
								<span class="text-tertiary">Nezdanitelné odpočty</span>
								<strong class="text-primary">{formatCZK(totalDeductions)}</strong>
							</div>
						{/if}
					</div>
				</Card>
			{/if}
		</div>
	{/if}
</div>
