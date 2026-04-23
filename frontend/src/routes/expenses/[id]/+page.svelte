<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		expensesApi,
		contactsApi,
		documentsApi,
		ocrApi,
		type Expense,
		type Contact,
		type ExpenseDocument,
		type OCRResult
	} from '$lib/api/client';
	import { toHalere, fromHalere } from '$lib/utils/money';
	import type { FormItem } from '$lib/components/InvoiceItemsEditor.svelte';
	import ExpenseEditForm from '$lib/components/expense/ExpenseEditForm.svelte';
	import ExpenseDetailsDisplay from '$lib/components/expense/ExpenseDetailsDisplay.svelte';
	import ExpenseDocumentsCard from '$lib/components/expense/ExpenseDocumentsCard.svelte';
	import OCRReviewDialog from '$lib/components/OCRReviewDialog.svelte';
	import Button from '$lib/ui/Button.svelte';
	import ConfirmDialog from '$lib/ui/ConfirmDialog.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import { toastSuccess, toastError } from '$lib/data/toast-state.svelte';
	import AuditLogPanel from '$lib/components/AuditLogPanel.svelte';

	let expense = $state<Expense | null>(null);
	let contacts = $state<Contact[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state<string | null>(null);
	let editing = $state(false);
	let documents = $state<ExpenseDocument[]>([]);
	let ocrResult = $state<OCRResult | null>(null);
	let ocrProcessing = $state(false);
	let showDeleteConfirm = $state(false);
	let useItems = $state(false);

	let expenseId = $derived(Number(page.params.id));

	let form = $state({
		vendor_id: null as number | null,
		expense_number: '',
		category: '',
		description: '',
		issue_date: '',
		amount: 0,
		currency_code: 'CZK',
		vat_rate_percent: 0,
		is_tax_deductible: true,
		business_percent: 100,
		payment_method: 'bank_transfer',
		notes: ''
	});

	let items = $state<FormItem[]>([
		{ description: '', quantity: 1, unit: 'ks', unit_price: 0, vat_rate_percent: 21 }
	]);

	let vatAmount = $derived((form.amount * form.vat_rate_percent) / (100 + form.vat_rate_percent));

	onMount(() => {
		loadExpense();
		loadDocuments();
	});

	async function loadDocuments() {
		try {
			documents = await documentsApi.listByExpense(expenseId);
		} catch {
			// non-critical
		}
	}

	async function loadExpense() {
		loading = true;
		error = null;
		try {
			expense = await expensesApi.getById(expenseId);
			populateForm();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst náklad';
		} finally {
			loading = false;
		}
	}

	function populateForm() {
		if (!expense) return;
		form = {
			vendor_id: expense.vendor_id ?? null,
			expense_number: expense.expense_number,
			category: expense.category,
			description: expense.description,
			issue_date: expense.issue_date,
			amount: fromHalere(expense.amount),
			currency_code: expense.currency_code,
			vat_rate_percent: expense.vat_rate_percent,
			is_tax_deductible: expense.is_tax_deductible,
			business_percent: expense.business_percent,
			payment_method: expense.payment_method,
			notes: expense.notes
		};

		// Populate items if they exist
		if (expense.items && expense.items.length > 0) {
			useItems = true;
			items = expense.items.map((item) => ({
				description: item.description,
				quantity: fromHalere(item.quantity),
				unit: item.unit,
				unit_price: fromHalere(item.unit_price),
				vat_rate_percent: item.vat_rate_percent
			}));
		} else {
			useItems = false;
			items = [{ description: '', quantity: 1, unit: 'ks', unit_price: 0, vat_rate_percent: 21 }];
		}
	}

	async function startEditing() {
		editing = true;
		try {
			const res = await contactsApi.list({ limit: 1000 });
			contacts = res.data;
		} catch {
			// non-critical
		}
	}

	function cancelEditing() {
		editing = false;
		populateForm();
	}

	async function handleSave() {
		if (!form.description) {
			toastError('Popis je povinný');
			return;
		}
		if (!useItems && form.amount <= 0) {
			toastError('Částka musí být větší než 0');
			return;
		}

		saving = true;

		try {
			const payload: Record<string, unknown> = {
				vendor_id: form.vendor_id || undefined,
				expense_number: form.expense_number,
				category: form.category,
				description: form.description,
				issue_date: form.issue_date,
				currency_code: form.currency_code,
				exchange_rate: 100,
				is_tax_deductible: form.is_tax_deductible,
				business_percent: form.business_percent,
				payment_method: form.payment_method,
				notes: form.notes
			};

			if (useItems) {
				payload.items = items.map((item, i) => ({
					description: item.description,
					quantity: toHalere(item.quantity),
					unit: item.unit,
					unit_price: toHalere(item.unit_price),
					vat_rate_percent: item.vat_rate_percent,
					sort_order: i + 1
				}));
				payload.amount = 0;
				payload.vat_rate_percent = 0;
				payload.vat_amount = 0;
			} else {
				payload.amount = toHalere(form.amount);
				payload.vat_rate_percent = form.vat_rate_percent;
				payload.vat_amount = toHalere(vatAmount);
				payload.items = [];
			}

			await expensesApi.update(expenseId, payload);
			toastSuccess('Náklad uložen');
			editing = false;
			await loadExpense();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se uložit náklad');
		} finally {
			saving = false;
		}
	}

	function handleDelete() {
		showDeleteConfirm = true;
	}

	async function confirmDelete() {
		showDeleteConfirm = false;
		try {
			await expensesApi.delete(expenseId);
			toastSuccess('Náklad smazán');
			goto('/expenses');
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se smazat náklad');
		}
	}

	async function handleOcr(docId: number) {
		ocrProcessing = true;
		try {
			ocrResult = await ocrApi.processDocument(docId);
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'OCR zpracování selhalo');
		} finally {
			ocrProcessing = false;
		}
	}

	function handleOcrConfirm(data: OCRResult) {
		ocrResult = null;
		if (!expense) return;
		// Auto-fill form fields from OCR data and switch to edit mode.
		// Amounts in `form.amount` are CZK (display); OCR returns halere.
		editing = true;
		form.description = data.description || form.description;
		form.amount = data.total_amount ? fromHalere(data.total_amount) : form.amount;
		form.vat_rate_percent = data.vat_rate_percent || form.vat_rate_percent;
		form.currency_code = data.currency_code || form.currency_code;
		if (data.issue_date) form.issue_date = data.issue_date;

		if (data.items && data.items.length > 0) {
			useItems = true;
			items = data.items.map((item) => ({
				description: item.description,
				quantity: fromHalere(item.quantity),
				unit: 'ks',
				unit_price: fromHalere(item.unit_price),
				vat_rate_percent: item.vat_rate_percent
			}));
		}
	}

	async function handleDocumentDelete(id: number) {
		await documentsApi.delete(id);
		await loadDocuments();
	}
</script>

<svelte:head>
	<title>{expense ? `Náklad - ${expense.description}` : 'Náklad'} - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-5xl">
	<a href="/expenses" class="text-sm text-secondary hover:text-primary">&larr; Zpět na náklady</a>

	<ErrorAlert {error} class="mt-4" />

	{#if loading}
		<LoadingSpinner class="mt-8" />
	{:else if expense}
		<!-- Header -->
		<div class="mt-4 flex items-center justify-between">
			<div>
				<h1 class="text-xl font-semibold text-primary">{expense.description}</h1>
				{#if expense.expense_number}
					<p class="mt-1 text-sm text-tertiary">Doklad: {expense.expense_number}</p>
				{/if}
			</div>
			{#if !editing}
				<div class="flex gap-2">
					<Button variant="secondary" onclick={startEditing}>Upravit</Button>
					<Button variant="danger" onclick={handleDelete}>Smazat</Button>
				</div>
			{/if}
		</div>

		{#if editing}
			<ExpenseEditForm
				bind:form
				bind:items
				bind:useItems
				{contacts}
				{saving}
				{vatAmount}
				onsave={handleSave}
				oncancel={cancelEditing}
			/>
		{:else}
			<div class="mt-6 space-y-6">
				<ExpenseDetailsDisplay {expense} />

				<ExpenseDocumentsCard
					{expenseId}
					{documents}
					onuploaded={loadDocuments}
					ondelete={handleDocumentDelete}
					onocr={handleOcr}
				/>
			</div>

			{#if ocrResult}
				<OCRReviewDialog
					{ocrResult}
					onclose={() => {
						ocrResult = null;
					}}
					onconfirm={handleOcrConfirm}
				/>
			{/if}
		{/if}
	{/if}

	{#if expense}
		<AuditLogPanel entityType="expense" entityId={expense.id} />
	{/if}
</div>

<ConfirmDialog
	bind:open={showDeleteConfirm}
	title="Smazat náklad"
	message="Opravdu chcete smazat tento náklad?"
	confirmLabel="Smazat"
	onconfirm={confirmDelete}
	oncancel={() => (showDeleteConfirm = false)}
/>
