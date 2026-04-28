<script lang="ts">
	import { onMount } from 'svelte';
	import {
		investmentsApi,
		type InvestmentDocument,
		type CapitalIncomeEntry,
		type SecurityTransaction,
		type InvestmentYearSummary
	} from '$lib/api/client';
	import { fromHalere, toHalere } from '$lib/utils/money';
	import { toastError } from '$lib/data/toast-state.svelte';
	import Button from '$lib/ui/Button.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import InvestmentDocumentsTab from '$lib/components/tax/InvestmentDocumentsTab.svelte';
	import InvestmentCapitalIncomeTab from '$lib/components/tax/InvestmentCapitalIncomeTab.svelte';
	import InvestmentSecurityTransactionsTab from '$lib/components/tax/InvestmentSecurityTransactionsTab.svelte';
	import InvestmentSummaryCard from '$lib/components/tax/InvestmentSummaryCard.svelte';

	let selectedYear = $state(new Date().getFullYear() - 1);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let saving = $state(false);

	// Active tab: 'documents' | 'capital' | 'securities'
	let activeTab = $state<'documents' | 'capital' | 'securities'>('documents');

	// Data
	let documents = $state<InvestmentDocument[]>([]);
	let capitalIncome = $state<CapitalIncomeEntry[]>([]);
	let transactions = $state<SecurityTransaction[]>([]);
	let summary = $state<InvestmentYearSummary | null>(null);

	// Upload form
	let uploadPlatform = $state('portu');
	let uploading = $state(false);

	// Capital income form
	let showCapitalForm = $state(false);
	let editingCapitalId = $state<number | null>(null);
	let capitalCategory = $state('dividend_cz');
	let capitalDescription = $state('');
	let capitalDate = $state('');
	let capitalGrossAmount = $state(0);
	let capitalWithheldCz = $state(0);
	let capitalWithheldForeign = $state(0);
	let capitalCountry = $state('CZ');
	let capitalNeedsDeclaring = $state(false);

	// Security transaction form
	let showTransactionForm = $state(false);
	let editingTransactionId = $state<number | null>(null);
	let txAssetType = $state('stock');
	let txAssetName = $state('');
	let txIsin = $state('');
	let txType = $state('buy');
	let txDate = $state('');
	let txQuantity = $state(0);
	let txUnitPrice = $state(0);
	let txTotalAmount = $state(0);
	let txFees = $state(0);
	let txCurrency = $state('CZK');
	let txExchangeRate = $state(1);

	async function loadData() {
		loading = true;
		error = null;
		try {
			const [docs, capital, txs, sum] = await Promise.all([
				investmentsApi.listDocuments(selectedYear),
				investmentsApi.listCapitalIncome(selectedYear),
				investmentsApi.listSecurityTransactions(selectedYear),
				investmentsApi.getYearSummary(selectedYear)
			]);
			documents = docs ?? [];
			capitalIncome = capital ?? [];
			transactions = txs ?? [];
			summary = sum;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst data';
		} finally {
			loading = false;
		}
	}

	// --- Document actions ---

	let fileInputEl = $state<HTMLInputElement | null>(null);
	let pendingKind: 'statement' | 'data' = 'statement';

	function uploadDocument(kind: 'statement' | 'data' = 'statement') {
		if (!fileInputEl) return;
		pendingKind = kind;
		fileInputEl.accept =
			kind === 'statement'
				? '.pdf,.png,.jpg,.jpeg,.webp,.heic'
				: '.pdf,.csv,.xlsx,.xls,.ods,.zip,.json,.txt';
		// Reset value so picking the same filename twice still fires change.
		fileInputEl.value = '';
		fileInputEl.click();
	}

	async function onFilePicked(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		uploading = true;
		try {
			await investmentsApi.uploadDocument(selectedYear, uploadPlatform, file, pendingKind);
			await loadData();
		} catch (err) {
			toastError(err instanceof Error ? err.message : 'Chyba při nahrávání');
		} finally {
			uploading = false;
			// Clear so the same file can be re-selected later if needed.
			input.value = '';
		}
	}

	let extractingId = $state<number | null>(null);

	async function extractDocument(id: number) {
		saving = true;
		extractingId = id;
		try {
			await investmentsApi.extractDocument(id);
			await loadData();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Chyba při extrakci');
		} finally {
			saving = false;
			extractingId = null;
		}
	}

	async function deleteDocument(id: number) {
		saving = true;
		try {
			await investmentsApi.deleteDocument(id);
			await loadData();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Chyba při mazání');
		} finally {
			saving = false;
		}
	}

	// --- Capital income actions ---

	function resetCapitalForm() {
		showCapitalForm = false;
		editingCapitalId = null;
		capitalCategory = 'dividend';
		capitalDescription = '';
		capitalDate = '';
		capitalGrossAmount = 0;
		capitalWithheldCz = 0;
		capitalWithheldForeign = 0;
		capitalCountry = 'CZ';
		capitalNeedsDeclaring = false;
	}

	function editCapitalEntry(entry: CapitalIncomeEntry) {
		editingCapitalId = entry.id;
		capitalCategory = entry.category;
		capitalDescription = entry.description;
		capitalDate = entry.income_date;
		capitalGrossAmount = fromHalere(entry.gross_amount);
		capitalWithheldCz = fromHalere(entry.withheld_tax_cz);
		capitalWithheldForeign = fromHalere(entry.withheld_tax_foreign);
		capitalCountry = entry.country_code;
		capitalNeedsDeclaring = entry.needs_declaring;
		showCapitalForm = true;
	}

	async function saveCapitalEntry() {
		saving = true;
		try {
			const data = {
				year: selectedYear,
				category: capitalCategory,
				description: capitalDescription,
				income_date: capitalDate,
				gross_amount: toHalere(capitalGrossAmount),
				withheld_tax_cz: toHalere(capitalWithheldCz),
				withheld_tax_foreign: toHalere(capitalWithheldForeign),
				country_code: capitalCountry,
				needs_declaring: capitalNeedsDeclaring
			};
			if (editingCapitalId) {
				await investmentsApi.updateCapitalIncome(editingCapitalId, data);
			} else {
				await investmentsApi.createCapitalIncome(data);
			}
			resetCapitalForm();
			await loadData();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Chyba při ukládání');
		} finally {
			saving = false;
		}
	}

	async function deleteCapitalEntry(id: number) {
		saving = true;
		try {
			await investmentsApi.deleteCapitalIncome(id);
			await loadData();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Chyba při mazání');
		} finally {
			saving = false;
		}
	}

	// --- Security transaction actions ---

	function resetTransactionForm() {
		showTransactionForm = false;
		editingTransactionId = null;
		txAssetType = 'stock';
		txAssetName = '';
		txIsin = '';
		txType = 'buy';
		txDate = '';
		txQuantity = 0;
		txUnitPrice = 0;
		txTotalAmount = 0;
		txFees = 0;
		txCurrency = 'CZK';
		txExchangeRate = 1;
	}

	function editTransaction(tx: SecurityTransaction) {
		editingTransactionId = tx.id;
		txAssetType = tx.asset_type;
		txAssetName = tx.asset_name;
		txIsin = tx.isin;
		txType = tx.transaction_type;
		txDate = tx.transaction_date;
		txQuantity = tx.quantity;
		txUnitPrice = fromHalere(tx.unit_price);
		txTotalAmount = fromHalere(tx.total_amount);
		txFees = fromHalere(tx.fees);
		txCurrency = tx.currency_code;
		txExchangeRate = tx.exchange_rate;
		showTransactionForm = true;
	}

	async function saveTransaction() {
		saving = true;
		try {
			const data = {
				year: selectedYear,
				asset_type: txAssetType,
				asset_name: txAssetName,
				isin: txIsin,
				transaction_type: txType,
				transaction_date: txDate,
				quantity: txQuantity,
				unit_price: toHalere(txUnitPrice),
				total_amount: toHalere(txTotalAmount),
				fees: toHalere(txFees),
				currency_code: txCurrency,
				exchange_rate: txExchangeRate
			};
			if (editingTransactionId) {
				await investmentsApi.updateSecurityTransaction(editingTransactionId, data);
			} else {
				await investmentsApi.createSecurityTransaction(data);
			}
			resetTransactionForm();
			await loadData();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Chyba při ukládání');
		} finally {
			saving = false;
		}
	}

	async function deleteTransaction(id: number) {
		saving = true;
		try {
			await investmentsApi.deleteSecurityTransaction(id);
			await loadData();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Chyba při mazání');
		} finally {
			saving = false;
		}
	}

	async function recalculateFifo() {
		saving = true;
		try {
			await investmentsApi.recalculateFifo(selectedYear);
			await loadData();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Chyba při přepočtu FIFO');
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
	<title>Investiční příjmy {selectedYear} - ZFaktury</title>
</svelte:head>

<!-- Single stable hidden file input shared by both upload buttons. -->
<input
	bind:this={fileInputEl}
	type="file"
	class="sr-only"
	tabindex="-1"
	aria-hidden="true"
	onchange={onFilePicked}
/>

<div class="mx-auto max-w-6xl">
	<h1 class="text-xl font-semibold text-primary">Investiční příjmy za rok {selectedYear}</h1>

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

	{#if loading}
		<LoadingSpinner class="mt-8 p-12" />
	{:else}
		<!-- Tabs -->
		<div class="mt-6 flex gap-1 border-b border-border">
			<button
				class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'documents'
					? 'border-b-2 border-accent text-accent'
					: 'text-tertiary hover:text-primary'}"
				onclick={() => (activeTab = 'documents')}
			>
				Dokumenty ({documents.length})
			</button>
			<button
				class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'capital'
					? 'border-b-2 border-accent text-accent'
					: 'text-tertiary hover:text-primary'}"
				onclick={() => (activeTab = 'capital')}
			>
				Dividendy a úroky ({capitalIncome.length})
			</button>
			<button
				class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'securities'
					? 'border-b-2 border-accent text-accent'
					: 'text-tertiary hover:text-primary'}"
				onclick={() => (activeTab = 'securities')}
			>
				Obchody s CP a kryptem ({transactions.length})
			</button>
		</div>

		<div class="mt-6 space-y-6">
			{#if activeTab === 'documents'}
				<InvestmentDocumentsTab
					{documents}
					{uploadPlatform}
					{uploading}
					{saving}
					{extractingId}
					onUploadPlatformChange={(v) => (uploadPlatform = v)}
					onUpload={uploadDocument}
					onExtract={extractDocument}
					onDelete={deleteDocument}
				/>
			{:else if activeTab === 'capital'}
				<InvestmentCapitalIncomeTab
					{capitalIncome}
					{saving}
					{showCapitalForm}
					{editingCapitalId}
					{capitalCategory}
					{capitalDescription}
					{capitalDate}
					{capitalGrossAmount}
					{capitalWithheldCz}
					{capitalWithheldForeign}
					{capitalCountry}
					{capitalNeedsDeclaring}
					onShowForm={() => {
						resetCapitalForm();
						showCapitalForm = true;
					}}
					onEdit={editCapitalEntry}
					onSave={saveCapitalEntry}
					onDelete={deleteCapitalEntry}
					onCancel={resetCapitalForm}
					onCategoryChange={(v) => (capitalCategory = v)}
					onDescriptionChange={(v) => (capitalDescription = v)}
					onDateChange={(v) => (capitalDate = v)}
					onGrossAmountChange={(v) => (capitalGrossAmount = v)}
					onWithheldCzChange={(v) => (capitalWithheldCz = v)}
					onWithheldForeignChange={(v) => (capitalWithheldForeign = v)}
					onCountryChange={(v) => (capitalCountry = v)}
					onNeedsDeclaringChange={(v) => (capitalNeedsDeclaring = v)}
				/>
			{:else if activeTab === 'securities'}
				<InvestmentSecurityTransactionsTab
					{transactions}
					{saving}
					{showTransactionForm}
					{editingTransactionId}
					{txAssetType}
					{txAssetName}
					{txIsin}
					{txType}
					{txDate}
					{txQuantity}
					{txUnitPrice}
					{txTotalAmount}
					{txFees}
					{txCurrency}
					{txExchangeRate}
					onShowForm={() => {
						resetTransactionForm();
						showTransactionForm = true;
					}}
					onRecalculateFifo={recalculateFifo}
					onEdit={editTransaction}
					onSave={saveTransaction}
					onDelete={deleteTransaction}
					onCancel={resetTransactionForm}
					onAssetTypeChange={(v) => (txAssetType = v)}
					onAssetNameChange={(v) => (txAssetName = v)}
					onIsinChange={(v) => (txIsin = v)}
					onTypeChange={(v) => (txType = v)}
					onDateChange={(v) => (txDate = v)}
					onQuantityChange={(v) => (txQuantity = v)}
					onUnitPriceChange={(v) => (txUnitPrice = v)}
					onTotalAmountChange={(v) => (txTotalAmount = v)}
					onFeesChange={(v) => (txFees = v)}
					onCurrencyChange={(v) => (txCurrency = v)}
					onExchangeRateChange={(v) => (txExchangeRate = v)}
				/>
			{/if}

			{#if summary}
				<InvestmentSummaryCard {summary} {selectedYear} />
			{/if}
		</div>
	{/if}
</div>
