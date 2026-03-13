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
	import { formatCZK, toHalere, fromHalere } from '$lib/utils/money';
	import { formatDate } from '$lib/utils/date';
	import { paymentMethodLabels } from '$lib/utils/invoice';
	import CategoryPicker from '$lib/components/CategoryPicker.svelte';
	import DateInput from '$lib/components/DateInput.svelte';
	import InvoiceItemsEditor, { type FormItem } from '$lib/components/InvoiceItemsEditor.svelte';
	import DocumentUpload from '$lib/components/DocumentUpload.svelte';
	import DocumentList from '$lib/components/DocumentList.svelte';
	import OCRReviewDialog from '$lib/components/OCRReviewDialog.svelte';
	import Button from '$lib/ui/Button.svelte';
	import ConfirmDialog from '$lib/ui/ConfirmDialog.svelte';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';
	import Textarea from '$lib/ui/Textarea.svelte';
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
		// Auto-fill form fields from OCR data and switch to edit mode
		editing = true;
		form.description = data.description || form.description;
		form.amount = data.total_amount || form.amount;
		form.vat_rate_percent = data.vat_rate_percent || form.vat_rate_percent;
		form.currency_code = data.currency_code || form.currency_code;
		if (data.issue_date) form.issue_date = data.issue_date;
	}

	let hasItems = $derived(expense?.items && expense.items.length > 0);
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
			<!-- Edit mode -->
			<form
				onsubmit={(e) => {
					e.preventDefault();
					handleSave();
				}}
				class="mt-6 space-y-6"
			>
				<Card>
					<h2 class="text-base font-semibold text-primary">Základní údaje</h2>
					<div class="mt-4 space-y-4">
						<div>
							<label for="edit-desc" class="block text-sm font-medium text-secondary">Popis *</label
							>
							<input
								id="edit-desc"
								type="text"
								bind:value={form.description}
								required
								class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
							<div>
								<label for="edit-cat" class="block text-sm font-medium text-secondary"
									>Kategorie</label
								>
								<CategoryPicker
									id="edit-cat"
									value={form.category}
									onchange={(v) => {
										form.category = v;
									}}
								/>
							</div>
							<div>
								<label for="edit-num" class="block text-sm font-medium text-secondary"
									>Číslo dokladu <HelpTip topic="cislo-dokladu" /></label
								>
								<input
									id="edit-num"
									type="text"
									bind:value={form.expense_number}
									class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
								/>
							</div>
						</div>
						<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
							<div>
								<label for="edit-date" class="block text-sm font-medium text-secondary">Datum</label
								>
								<DateInput id="edit-date" bind:value={form.issue_date} required />
							</div>
							<div>
								<label for="edit-vendor" class="block text-sm font-medium text-secondary"
									>Dodavatel</label
								>
								<select
									id="edit-vendor"
									bind:value={form.vendor_id}
									class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
								>
									<option value={null}>-- Bez dodavatele --</option>
									{#each contacts as contact (contact.id)}
										<option value={contact.id}>{contact.name}</option>
									{/each}
								</select>
							</div>
						</div>
					</div>
				</Card>

				<!-- Toggle: flat amount vs items -->
				<Card>
					<div class="flex items-center gap-3">
						<input
							id="edit-use-items"
							type="checkbox"
							bind:checked={useItems}
							class="h-4 w-4 rounded border-border accent-accent"
						/>
						<label for="edit-use-items" class="text-sm font-medium text-secondary"
							>Zadat jednotlivé položky</label
						>
					</div>
				</Card>

				{#if useItems}
					<InvoiceItemsEditor bind:items idPrefix="edit-exp-" />
				{:else}
					<Card>
						<h2 class="text-base font-semibold text-primary">Částka a DPH</h2>
						<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
							<div>
								<label for="edit-amount" class="block text-sm font-medium text-secondary"
									>Částka s DPH (CZK)</label
								>
								<input
									id="edit-amount"
									type="number"
									step="0.01"
									min="0"
									bind:value={form.amount}
									class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary font-mono tabular-nums focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
								/>
							</div>
							<div>
								<label for="edit-vat" class="block text-sm font-medium text-secondary"
									>Sazba DPH <HelpTip topic="sazba-dph" /></label
								>
								<select
									id="edit-vat"
									bind:value={form.vat_rate_percent}
									class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
								>
									<option value={21}>21%</option>
									<option value={12}>12%</option>
									<option value={0}>0%</option>
								</select>
							</div>
							<div>
								<span class="block text-sm font-medium text-secondary">DPH</span>
								<div
									class="mt-1 bg-elevated border-border text-secondary rounded-lg px-3 py-2 text-sm font-mono tabular-nums"
								>
									{formatCZK(toHalere(vatAmount))}
								</div>
							</div>
						</div>
					</Card>
				{/if}

				<Card>
					<h2 class="text-base font-semibold text-primary">Daňové nastavení</h2>
					<div class="mt-4 space-y-4">
						<div class="flex items-center gap-3">
							<input
								id="edit-deductible"
								type="checkbox"
								bind:checked={form.is_tax_deductible}
								class="h-4 w-4 rounded border-border accent-accent"
							/>
							<label for="edit-deductible" class="text-sm font-medium text-secondary"
								>Daňově uznatelný náklad <HelpTip topic="danove-uznatelny" /></label
							>
						</div>
						<div>
							<label for="edit-biz" class="block text-sm font-medium text-secondary"
								>Podíl pro podnikání (%) <HelpTip topic="podil-podnikani" /></label
							>
							<input
								id="edit-biz"
								type="number"
								min="0"
								max="100"
								bind:value={form.business_percent}
								class="mt-1 w-32 rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary font-mono tabular-nums focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div>
							<label for="edit-pm" class="block text-sm font-medium text-secondary"
								>Způsob platby</label
							>
							<select
								id="edit-pm"
								bind:value={form.payment_method}
								class="mt-1 w-full max-w-xs rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							>
								<option value="bank_transfer">Bankovní převod</option>
								<option value="cash">Hotovost</option>
								<option value="card">Karta</option>
							</select>
						</div>
					</div>
				</Card>

				<Card>
					<h2 class="text-base font-semibold text-primary">Poznámky</h2>
					<div class="mt-4">
						<Textarea bind:value={form.notes} rows={3} />
					</div>
				</Card>

				<FormActions {saving} saveLabel="Uložit změny" oncancel={cancelEditing} />
			</form>
		{:else}
			<!-- View mode -->
			<div class="mt-6 space-y-6">
				<Card>
					<h2 class="text-base font-semibold text-primary">Základní údaje</h2>
					<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div>
							<dt class="text-sm font-medium text-tertiary">Kategorie</dt>
							<dd class="mt-1 text-sm text-primary">{expense.category || '-'}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-tertiary">Datum</dt>
							<dd class="mt-1 text-sm text-primary">{formatDate(expense.issue_date)}</dd>
						</div>
						{#if expense.expense_number}
							<div>
								<dt class="text-sm font-medium text-tertiary">Číslo dokladu</dt>
								<dd class="mt-1 text-sm text-primary">{expense.expense_number}</dd>
							</div>
						{/if}
						<div>
							<dt class="text-sm font-medium text-tertiary">Způsob platby</dt>
							<dd class="mt-1 text-sm text-primary">
								{paymentMethodLabels[expense.payment_method] ?? expense.payment_method}
							</dd>
						</div>
					</dl>
				</Card>

				{#if hasItems}
					<!-- Items table -->
					<Card>
						<h2 class="text-base font-semibold text-primary">Položky</h2>
						<div class="mt-4 overflow-x-auto">
							<table class="w-full text-sm">
								<thead>
									<tr class="border-b border-border text-left text-xs text-tertiary">
										<th class="pb-2 pr-4">Popis</th>
										<th class="pb-2 pr-4 text-right">Množství</th>
										<th class="pb-2 pr-4">Jedn.</th>
										<th class="pb-2 pr-4 text-right">Cena/ks</th>
										<th class="pb-2 pr-4 text-right">DPH %</th>
										<th class="pb-2 pr-4 text-right">DPH</th>
										<th class="pb-2 text-right">Celkem</th>
									</tr>
								</thead>
								<tbody>
									{#each expense.items! as item (item.id)}
										<tr class="border-b border-border/50">
											<td class="py-2 pr-4 text-primary">{item.description}</td>
											<td class="py-2 pr-4 text-right font-mono tabular-nums text-primary"
												>{fromHalere(item.quantity)}</td
											>
											<td class="py-2 pr-4 text-secondary">{item.unit}</td>
											<td class="py-2 pr-4 text-right font-mono tabular-nums text-primary"
												>{formatCZK(item.unit_price)}</td
											>
											<td class="py-2 pr-4 text-right text-secondary">{item.vat_rate_percent}%</td>
											<td class="py-2 pr-4 text-right font-mono tabular-nums text-secondary"
												>{formatCZK(item.vat_amount)}</td
											>
											<td class="py-2 text-right font-mono tabular-nums text-primary"
												>{formatCZK(item.total_amount)}</td
											>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
						<!-- Totals -->
						<div class="mt-4 border-t border-border pt-3">
							<div class="flex flex-col items-end gap-1 text-sm">
								<div class="flex gap-8">
									<span class="text-tertiary">Základ:</span>
									<span class="font-medium text-primary font-mono tabular-nums"
										>{formatCZK(expense.amount - expense.vat_amount)}</span
									>
								</div>
								<div class="flex gap-8">
									<span class="text-tertiary">DPH:</span>
									<span class="font-medium text-primary font-mono tabular-nums"
										>{formatCZK(expense.vat_amount)}</span
									>
								</div>
								<div class="flex gap-8 border-t border-border pt-1 text-base">
									<span class="font-semibold text-primary">Celkem:</span>
									<span class="font-semibold text-primary font-mono tabular-nums"
										>{formatCZK(expense.amount)}</span
									>
								</div>
							</div>
						</div>
					</Card>
				{:else}
					<!-- Flat amount display for legacy expenses -->
					<Card>
						<h2 class="text-base font-semibold text-primary">Částka</h2>
						<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
							<div>
								<dt class="text-sm font-medium text-tertiary">Částka s DPH</dt>
								<dd class="mt-1 text-lg font-semibold text-primary font-mono tabular-nums">
									{formatCZK(expense.amount)}
								</dd>
							</div>
							<div>
								<dt class="text-sm font-medium text-tertiary">DPH ({expense.vat_rate_percent}%)</dt>
								<dd class="mt-1 text-sm text-primary font-mono tabular-nums">
									{formatCZK(expense.vat_amount)}
								</dd>
							</div>
							<div>
								<dt class="text-sm font-medium text-tertiary">Základ</dt>
								<dd class="mt-1 text-sm text-primary font-mono tabular-nums">
									{formatCZK(expense.amount - expense.vat_amount)}
								</dd>
							</div>
						</dl>
					</Card>
				{/if}

				<Card>
					<h2 class="text-base font-semibold text-primary">Daňové údaje</h2>
					<dl class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div>
							<dt class="text-sm font-medium text-tertiary">Daňově uznatelný</dt>
							<dd class="mt-1 text-sm text-primary">{expense.is_tax_deductible ? 'Ano' : 'Ne'}</dd>
						</div>
						<div>
							<dt class="text-sm font-medium text-tertiary">Podíl pro podnikání</dt>
							<dd class="mt-1 text-sm text-primary">{expense.business_percent}%</dd>
						</div>
					</dl>
				</Card>

				<!-- Documents -->
				<Card>
					<h2 class="text-base font-semibold text-primary">Dokumenty</h2>
					<div class="mt-4">
						<DocumentUpload
							{expenseId}
							onuploaded={() => {
								loadDocuments();
							}}
						/>
						<div class="mt-4">
							<DocumentList
								{documents}
								ondelete={async (id) => {
									await documentsApi.delete(id);
									await loadDocuments();
								}}
								onocr={handleOcr}
							/>
						</div>
					</div>
				</Card>

				{#if expense.notes}
					<Card>
						<h2 class="text-base font-semibold text-primary">Poznámky</h2>
						<p class="mt-2 text-sm text-primary whitespace-pre-wrap">{expense.notes}</p>
					</Card>
				{/if}

				<div class="text-xs text-muted">
					Vytvořeno: {formatDate(expense.created_at)} | Upraveno: {formatDate(expense.updated_at)}
				</div>
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
