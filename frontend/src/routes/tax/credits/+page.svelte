<script lang="ts">
	import { onMount } from 'svelte';
	import {
		taxCreditsApi,
		taxDeductionsApi,
		type TaxCreditsSummary,
		type TaxDeduction,
		type TaxExtractionResult,
		type TaxConstants
	} from '$lib/api/client';
	import { loadTaxConstants } from '$lib/data/tax-constants.svelte';
	import { fromHalere, toHalere } from '$lib/utils/money';
	import { toastError } from '$lib/data/toast-state.svelte';
	import Button from '$lib/ui/Button.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import PersonalCreditsCard from '$lib/components/tax/PersonalCreditsCard.svelte';
	import SpouseCreditsCard from '$lib/components/tax/SpouseCreditsCard.svelte';
	import ChildrenCreditsCard from '$lib/components/tax/ChildrenCreditsCard.svelte';
	import TaxDeductionsCard from '$lib/components/tax/TaxDeductionsCard.svelte';
	import TaxCreditsSummaryCard from '$lib/components/tax/TaxCreditsSummaryCard.svelte';

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
			toastError(e instanceof Error ? e.message : 'Chyba při ukládání');
		} finally {
			saving = false;
		}
	}

	async function deleteSpouse() {
		saving = true;
		try {
			await taxCreditsApi.deleteSpouse(selectedYear);
			resetSpouseForm();
			await loadData();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Chyba při mazání');
		} finally {
			saving = false;
		}
	}

	async function savePersonal() {
		saving = true;
		try {
			await taxCreditsApi.upsertPersonal(selectedYear, {
				is_student: isStudent,
				student_months: isStudent ? studentMonths : 0,
				disability_level: disabilityLevel
			});
			await loadData();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Chyba při ukládání');
		} finally {
			saving = false;
		}
	}

	async function saveChild() {
		saving = true;
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
			toastError(e instanceof Error ? e.message : 'Chyba při ukládání');
		} finally {
			saving = false;
		}
	}

	async function deleteChild(id: number) {
		saving = true;
		try {
			await taxCreditsApi.deleteChild(selectedYear, id);
			await loadData();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Chyba při mazání');
		} finally {
			saving = false;
		}
	}

	function editChild(child: {
		id: number;
		child_name: string;
		birth_number: string;
		child_order: number;
		months_claimed: number;
		ztp: boolean;
	}) {
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
			toastError(e instanceof Error ? e.message : 'Chyba při ukládání');
		} finally {
			saving = false;
		}
	}

	async function deleteDeduction(id: number) {
		saving = true;
		try {
			await taxDeductionsApi.delete(selectedYear, id);
			await loadData();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Chyba při mazání');
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
			try {
				await taxDeductionsApi.uploadDocument(selectedYear, deductionId, file);
				await loadData();
			} catch (e) {
				toastError(e instanceof Error ? e.message : 'Chyba při nahrávání');
			} finally {
				saving = false;
			}
		};
		input.click();
	}

	async function extractAmount(docId: number) {
		saving = true;
		try {
			const result: TaxExtractionResult = await taxDeductionsApi.extractDocument(docId);
			if (result.amount_czk > 0) {
				await loadData();
			}
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Chyba při extrakci');
		} finally {
			saving = false;
		}
	}

	async function deleteDocument(docId: number) {
		saving = true;
		try {
			await taxDeductionsApi.deleteDocument(docId);
			await loadData();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Chyba při mazání');
		} finally {
			saving = false;
		}
	}

	async function copyFromPreviousYear() {
		saving = true;
		try {
			await taxCreditsApi.copyFromYear(selectedYear, selectedYear - 1);
			await loadData();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Chyba při kopírování');
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
			<PersonalCreditsCard
				bind:isStudent
				bind:studentMonths
				bind:disabilityLevel
				{summary}
				{taxConstants}
				{saving}
				onSave={savePersonal}
			/>

			<SpouseCreditsCard
				bind:spouseName
				bind:spouseBirthNumber
				bind:spouseIncome
				bind:spouseZtp
				bind:spouseMonths
				bind:showSpouseForm
				{summary}
				{taxConstants}
				{saving}
				onSave={saveSpouse}
				onDelete={deleteSpouse}
			/>

			<ChildrenCreditsCard
				bind:showChildForm
				bind:editingChildId
				bind:childName
				bind:childBirthNumber
				bind:childOrder
				bind:childMonths
				bind:childZtp
				{summary}
				{taxConstants}
				{saving}
				onSave={saveChild}
				onDelete={deleteChild}
				onEdit={editChild}
				onReset={resetChildForm}
			/>

			<TaxDeductionsCard
				{deductions}
				bind:showDeductionForm
				bind:editingDeductionId
				bind:deductionCategory
				bind:deductionDescription
				bind:deductionAmount
				{taxConstants}
				{saving}
				onSave={saveDeduction}
				onDelete={deleteDeduction}
				onEdit={editDeduction}
				onReset={resetDeductionForm}
				onUploadDocument={uploadDocument}
				onExtractAmount={extractAmount}
				onDeleteDocument={deleteDocument}
			/>

			{#if summary}
				<TaxCreditsSummaryCard {summary} {deductions} />
			{/if}
		</div>
	{/if}
</div>
