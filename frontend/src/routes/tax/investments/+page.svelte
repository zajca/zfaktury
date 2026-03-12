<script lang="ts">
	import { onMount } from 'svelte';
	import {
		investmentsApi,
		type InvestmentDocument,
		type CapitalIncomeEntry,
		type SecurityTransaction,
		type InvestmentYearSummary
	} from '$lib/api/client';
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

	const platformLabels: Record<string, string> = {
		portu: 'Portu',
		zonky: 'Zonky',
		trading212: 'Trading 212',
		revolut: 'Revolut',
		other: 'Jiny'
	};

	const capitalCategoryLabels: Record<string, string> = {
		dividend_cz: 'Dividenda (CZ)',
		dividend_foreign: 'Dividenda (zahranicni)',
		interest: 'Urok',
		coupon: 'Kupon',
		fund_distribution: 'Vyplata z fondu',
		other: 'Ostatni'
	};

	const assetTypeLabels: Record<string, string> = {
		stock: 'Akcie',
		etf: 'ETF',
		bond: 'Dluhopis',
		fund: 'Fond',
		crypto: 'Kryptomena',
		other: 'Jiny'
	};

	const statusLabels: Record<string, { text: string; class: string }> = {
		pending: { text: 'Ceka na zpracovani', class: 'bg-warning-bg text-warning' },
		extracted: { text: 'Extrahovano', class: 'bg-success-bg text-success' },
		failed: { text: 'Chyba', class: 'bg-danger-bg text-danger' }
	};

	function formatAmount(amountInHalere: number): string {
		return formatCZK(amountInHalere);
	}

	function formatFileSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}

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
			error = e instanceof Error ? e.message : 'Nepodarilo se nacist data';
		} finally {
			loading = false;
		}
	}

	// --- Document actions ---

	async function uploadDocument() {
		const input = document.createElement('input');
		input.type = 'file';
		input.accept = '.pdf,.csv,.xlsx,.xls';
		input.onchange = async () => {
			const file = input.files?.[0];
			if (!file) return;
			uploading = true;
			error = null;
			try {
				await investmentsApi.uploadDocument(selectedYear, uploadPlatform, file);
				await loadData();
			} catch (e) {
				error = e instanceof Error ? e.message : 'Chyba pri nahravani';
			} finally {
				uploading = false;
			}
		};
		input.click();
	}

	async function extractDocument(id: number) {
		saving = true;
		error = null;
		try {
			await investmentsApi.extractDocument(id);
			await loadData();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba pri extrakci';
		} finally {
			saving = false;
		}
	}

	async function deleteDocument(id: number) {
		saving = true;
		error = null;
		try {
			await investmentsApi.deleteDocument(id);
			await loadData();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba pri mazani';
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
		error = null;
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
			error = e instanceof Error ? e.message : 'Chyba pri ukladani';
		} finally {
			saving = false;
		}
	}

	async function deleteCapitalEntry(id: number) {
		saving = true;
		error = null;
		try {
			await investmentsApi.deleteCapitalIncome(id);
			await loadData();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba pri mazani';
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
		error = null;
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
			error = e instanceof Error ? e.message : 'Chyba pri ukladani';
		} finally {
			saving = false;
		}
	}

	async function deleteTransaction(id: number) {
		saving = true;
		error = null;
		try {
			await investmentsApi.deleteSecurityTransaction(id);
			await loadData();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba pri mazani';
		} finally {
			saving = false;
		}
	}

	async function recalculateFifo() {
		saving = true;
		error = null;
		try {
			await investmentsApi.recalculateFifo(selectedYear);
			await loadData();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Chyba pri prepoctu FIFO';
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
	<title>Investicni prijmy {selectedYear} - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-6xl">
	<h1 class="text-xl font-semibold text-primary">Investicni prijmy za rok {selectedYear}</h1>

	<!-- Year selector -->
	<div class="mt-4 flex items-center gap-3">
		<Button variant="ghost" size="sm" onclick={() => { selectedYear--; }} title="Predchozi rok" aria-label="Predchozi rok">
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
			</svg>
		</Button>
		<span class="min-w-[4rem] text-center text-xl font-semibold text-primary tabular-nums">{selectedYear}</span>
		<Button variant="ghost" size="sm" onclick={() => { selectedYear++; }} title="Nasledujici rok" aria-label="Nasledujici rok">
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
				class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'documents' ? 'border-b-2 border-accent text-accent' : 'text-tertiary hover:text-primary'}"
				onclick={() => (activeTab = 'documents')}
			>
				Dokumenty ({documents.length})
			</button>
			<button
				class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'capital' ? 'border-b-2 border-accent text-accent' : 'text-tertiary hover:text-primary'}"
				onclick={() => (activeTab = 'capital')}
			>
				Dividendy a uroky ({capitalIncome.length})
			</button>
			<button
				class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'securities' ? 'border-b-2 border-accent text-accent' : 'text-tertiary hover:text-primary'}"
				onclick={() => (activeTab = 'securities')}
			>
				Obchody s CP a kryptem ({transactions.length})
			</button>
		</div>

		<div class="mt-6 space-y-6">
			<!-- Tab: Documents -->
			{#if activeTab === 'documents'}
				<Card>
					<div class="flex items-center justify-between">
						<h2 class="text-base font-semibold text-primary">Nahrane dokumenty</h2>
						<div class="flex items-center gap-2">
							<Select value={uploadPlatform} onchange={(e: Event) => { uploadPlatform = (e.currentTarget as HTMLSelectElement).value; }}>
								{#each Object.entries(platformLabels) as [key, label]}
									<option value={key}>{label}</option>
								{/each}
							</Select>
							<Button variant="primary" size="sm" onclick={uploadDocument} disabled={uploading}>
								{uploading ? 'Nahrava se...' : 'Nahrat dokument'}
							</Button>
						</div>
					</div>

					{#if documents.length > 0}
						<div class="mt-4 overflow-x-auto">
							<table class="w-full text-sm">
								<thead>
									<tr class="border-b border-border text-left text-xs text-tertiary">
										<th class="pb-2 pr-4">Nazev souboru</th>
										<th class="pb-2 pr-4">Platforma</th>
										<th class="pb-2 pr-4">Stav</th>
										<th class="pb-2 pr-4">Velikost</th>
										<th class="pb-2 text-right">Akce</th>
									</tr>
								</thead>
								<tbody>
									{#each documents as doc (doc.id)}
										<tr class="border-b border-border-subtle">
											<td class="py-2 pr-4">
												<a href={investmentsApi.downloadDocumentUrl(doc.id)} class="text-accent hover:underline" target="_blank">{doc.filename}</a>
											</td>
											<td class="py-2 pr-4 text-tertiary">{platformLabels[doc.platform] ?? doc.platform}</td>
											<td class="py-2 pr-4">
												{#if statusLabels[doc.extraction_status]}
													{@const status = statusLabels[doc.extraction_status]}
													<span class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium {status.class}">
														{status.text}
													</span>
												{:else}
													<span class="inline-flex rounded-full bg-surface px-2 py-0.5 text-xs font-medium text-tertiary">
														{doc.extraction_status}
													</span>
												{/if}
												{#if doc.extraction_error}
													<span class="ml-1 text-xs text-danger" title={doc.extraction_error}>!</span>
												{/if}
											</td>
											<td class="py-2 pr-4 text-tertiary">{formatFileSize(doc.size)}</td>
											<td class="py-2 text-right">
												<div class="flex justify-end gap-1">
													{#if doc.extraction_status !== 'extracted'}
														<Button variant="secondary" size="sm" onclick={() => extractDocument(doc.id)} disabled={saving}>Extrahovat</Button>
													{/if}
													<Button variant="danger" size="sm" onclick={() => deleteDocument(doc.id)} disabled={saving}>Smazat</Button>
												</div>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					{:else}
						<p class="mt-4 text-sm text-tertiary">Zadne nahrane dokumenty. Nahrajte vypisyz investicnich platforem pro automatickou extrakci dat.</p>
					{/if}
				</Card>

			<!-- Tab: Capital Income -->
			{:else if activeTab === 'capital'}
				<Card>
					<div class="flex items-center justify-between">
						<h2 class="text-base font-semibold text-primary">Kapitalove prijmy (§8) <HelpTip topic="kapitalove-prijmy-s8" /></h2>
						<Button variant="primary" size="sm" onclick={() => { resetCapitalForm(); showCapitalForm = true; }}>Pridat rucne</Button>
					</div>

					{#if capitalIncome.length > 0}
						<div class="mt-4 overflow-x-auto">
							<table class="w-full text-sm">
								<thead>
									<tr class="border-b border-border text-left text-xs text-tertiary">
										<th class="pb-2 pr-4">Datum</th>
										<th class="pb-2 pr-4">Kategorie</th>
										<th class="pb-2 pr-4">Popis</th>
										<th class="pb-2 pr-4 text-right">Hruba castka</th>
										<th class="pb-2 pr-4 text-right">Srazena dan <HelpTip topic="srazena-dan" /></th>
										<th class="pb-2 pr-4">K priznani</th>
										<th class="pb-2 text-right">Akce</th>
									</tr>
								</thead>
								<tbody>
									{#each capitalIncome as entry (entry.id)}
										<tr class="border-b border-border-subtle">
											<td class="py-2 pr-4 text-tertiary">{entry.income_date}</td>
											<td class="py-2 pr-4">
												<span class="text-xs font-medium uppercase text-accent">{capitalCategoryLabels[entry.category] ?? entry.category}</span>
											</td>
											<td class="py-2 pr-4 text-primary">{entry.description}</td>
											<td class="py-2 pr-4 text-right font-medium text-primary">{formatAmount(entry.gross_amount)}</td>
											<td class="py-2 pr-4 text-right text-tertiary">
												{formatAmount(entry.withheld_tax_cz + entry.withheld_tax_foreign)}
											</td>
											<td class="py-2 pr-4">
												{#if entry.needs_declaring}
													<span class="inline-flex rounded-full bg-warning-bg px-2 py-0.5 text-xs font-medium text-warning">Ano</span>
												{:else}
													<span class="inline-flex rounded-full bg-success-bg px-2 py-0.5 text-xs font-medium text-success">Ne</span>
												{/if}
											</td>
											<td class="py-2 text-right">
												<div class="flex justify-end gap-1">
													<Button variant="ghost" size="sm" onclick={() => editCapitalEntry(entry)}>Upravit</Button>
													<Button variant="danger" size="sm" onclick={() => deleteCapitalEntry(entry.id)} disabled={saving}>Smazat</Button>
												</div>
											</td>
										</tr>
									{/each}
								</tbody>
								<tfoot>
									<tr class="border-t border-border font-medium">
										<td colspan="3" class="py-2 pr-4 text-tertiary">Celkem</td>
										<td class="py-2 pr-4 text-right text-primary">
											{formatAmount(capitalIncome.reduce((sum, e) => sum + e.gross_amount, 0))}
										</td>
										<td class="py-2 pr-4 text-right text-tertiary">
											{formatAmount(capitalIncome.reduce((sum, e) => sum + e.withheld_tax_cz + e.withheld_tax_foreign, 0))}
										</td>
										<td colspan="2"></td>
									</tr>
								</tfoot>
							</table>
						</div>
					{:else}
						<p class="mt-4 text-sm text-tertiary">Zadne kapitalove prijmy. Pridejte rucne nebo nahrajte dokument k extrakci.</p>
					{/if}

					{#if showCapitalForm}
						<div class="mt-4 rounded-lg border border-border-subtle bg-elevated p-4">
							<h3 class="text-sm font-medium text-primary">{editingCapitalId ? 'Upravit zaznam' : 'Pridat zaznam'}</h3>
							<div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-3">
								<div>
									<span class="text-xs text-tertiary">Kategorie</span>
									<Select value={capitalCategory} onchange={(e: Event) => { capitalCategory = (e.currentTarget as HTMLSelectElement).value; }}>
										{#each Object.entries(capitalCategoryLabels) as [key, label]}
											<option value={key}>{label}</option>
										{/each}
									</Select>
								</div>
								<div>
									<span class="text-xs text-tertiary">Popis</span>
									<Input value={capitalDescription} oninput={(e: Event) => { capitalDescription = (e.currentTarget as HTMLInputElement).value; }} placeholder="Nazev akcie / fondu" />
								</div>
								<div>
									<span class="text-xs text-tertiary">Datum</span>
									<Input type="date" value={capitalDate} oninput={(e: Event) => { capitalDate = (e.currentTarget as HTMLInputElement).value; }} />
								</div>
								<div>
									<span class="text-xs text-tertiary">Hruba castka (CZK)</span>
									<Input type="number" value={capitalGrossAmount} oninput={(e: Event) => { capitalGrossAmount = Number((e.currentTarget as HTMLInputElement).value); }} step="0.01" />
								</div>
								<div>
									<span class="text-xs text-tertiary">Srazena dan CR (CZK)</span>
									<Input type="number" value={capitalWithheldCz} oninput={(e: Event) => { capitalWithheldCz = Number((e.currentTarget as HTMLInputElement).value); }} step="0.01" />
								</div>
								<div>
									<span class="text-xs text-tertiary">Srazena dan zahranici (CZK)</span>
									<Input type="number" value={capitalWithheldForeign} oninput={(e: Event) => { capitalWithheldForeign = Number((e.currentTarget as HTMLInputElement).value); }} step="0.01" />
								</div>
								<div>
									<span class="text-xs text-tertiary">Zeme</span>
									<Input value={capitalCountry} oninput={(e: Event) => { capitalCountry = (e.currentTarget as HTMLInputElement).value; }} placeholder="CZ" maxlength={2} />
								</div>
								<label class="flex items-center gap-2 text-sm text-primary">
									<input type="checkbox" bind:checked={capitalNeedsDeclaring} class="rounded border-border" />
									Nutno priznat v DP <HelpTip topic="nutno-priznat-dp" />
								</label>
							</div>
							<div class="mt-3 flex gap-2">
								<Button variant="primary" size="sm" onclick={saveCapitalEntry} disabled={saving}>Ulozit</Button>
								<Button variant="ghost" size="sm" onclick={resetCapitalForm}>Zrusit</Button>
							</div>
						</div>
					{/if}
				</Card>

			<!-- Tab: Security Transactions -->
			{:else if activeTab === 'securities'}
				<Card>
					<div class="flex items-center justify-between">
						<h2 class="text-base font-semibold text-primary">Obchody s CP a kryptem (§10) <HelpTip topic="obchody-cp-s10" /></h2>
						<div class="flex gap-2">
							<Button variant="secondary" size="sm" onclick={recalculateFifo} disabled={saving}>Prepocitat FIFO</Button> <HelpTip topic="fifo-prepocet" />
							<Button variant="primary" size="sm" onclick={() => { resetTransactionForm(); showTransactionForm = true; }}>Pridat rucne</Button>
						</div>
					</div>

					{#if transactions.length > 0}
						<div class="mt-4 overflow-x-auto">
							<table class="w-full text-sm">
								<thead>
									<tr class="border-b border-border text-left text-xs text-tertiary">
										<th class="pb-2 pr-3">Datum</th>
										<th class="pb-2 pr-3">Typ</th>
										<th class="pb-2 pr-3">Nazev</th>
										<th class="pb-2 pr-3">ISIN</th>
										<th class="pb-2 pr-3 text-right">Pocet</th>
										<th class="pb-2 pr-3 text-right">Cena</th>
										<th class="pb-2 pr-3 text-right">Poplatky</th>
										<th class="pb-2 pr-3 text-right">Nabyvaci cena</th>
										<th class="pb-2 pr-3 text-right">Zisk/ztrata</th>
										<th class="pb-2 pr-3">Casovy test <HelpTip topic="casovy-test" /></th>
										<th class="pb-2 text-right">Akce</th>
									</tr>
								</thead>
								<tbody>
									{#each transactions as tx (tx.id)}
										<tr class="border-b border-border-subtle">
											<td class="py-2 pr-3 text-tertiary">{tx.transaction_date}</td>
											<td class="py-2 pr-3">
												<span class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium {tx.transaction_type === 'buy' ? 'bg-accent-muted text-accent-text' : 'bg-warning-bg text-warning'}">
													{tx.transaction_type === 'buy' ? 'Nakup' : 'Prodej'}
												</span>
											</td>
											<td class="py-2 pr-3 text-primary">
												<span class="text-xs uppercase text-tertiary">{assetTypeLabels[tx.asset_type] ?? tx.asset_type}</span>
												{tx.asset_name}
											</td>
											<td class="py-2 pr-3 font-mono text-xs text-tertiary">{tx.isin || '-'}</td>
											<td class="py-2 pr-3 text-right tabular-nums">{tx.quantity}</td>
											<td class="py-2 pr-3 text-right tabular-nums">{formatAmount(tx.total_amount)}</td>
											<td class="py-2 pr-3 text-right tabular-nums text-tertiary">{formatAmount(tx.fees)}</td>
											<td class="py-2 pr-3 text-right tabular-nums">{formatAmount(tx.cost_basis)}</td>
											<td class="py-2 pr-3 text-right tabular-nums {tx.computed_gain >= 0 ? 'text-success' : 'text-danger'}">
												{#if tx.transaction_type === 'sell'}
													{formatAmount(tx.computed_gain)}
												{:else}
													<span class="text-tertiary">-</span>
												{/if}
											</td>
											<td class="py-2 pr-3">
												{#if tx.time_test_exempt}
													<span class="inline-flex rounded-full bg-success-bg px-2 py-0.5 text-xs font-medium text-success">Osv.</span>
												{:else if tx.transaction_type === 'sell'}
													<span class="inline-flex rounded-full bg-danger-bg px-2 py-0.5 text-xs font-medium text-danger">Ne</span>
												{:else}
													<span class="text-tertiary">-</span>
												{/if}
											</td>
											<td class="py-2 text-right">
												<div class="flex justify-end gap-1">
													<Button variant="ghost" size="sm" onclick={() => editTransaction(tx)}>Upravit</Button>
													<Button variant="danger" size="sm" onclick={() => deleteTransaction(tx.id)} disabled={saving}>Smazat</Button>
												</div>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					{:else}
						<p class="mt-4 text-sm text-tertiary">Zadne obchody s cennymi papiry. Pridejte rucne nebo nahrajte dokument k extrakci.</p>
					{/if}

					{#if showTransactionForm}
						<div class="mt-4 rounded-lg border border-border-subtle bg-elevated p-4">
							<h3 class="text-sm font-medium text-primary">{editingTransactionId ? 'Upravit obchod' : 'Pridat obchod'}</h3>
							<div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-4">
								<div>
									<span class="text-xs text-tertiary">Typ aktiva</span>
									<Select value={txAssetType} onchange={(e: Event) => { txAssetType = (e.currentTarget as HTMLSelectElement).value; }}>
										{#each Object.entries(assetTypeLabels) as [key, label]}
											<option value={key}>{label}</option>
										{/each}
									</Select>
								</div>
								<div>
									<span class="text-xs text-tertiary">Nazev</span>
									<Input value={txAssetName} oninput={(e: Event) => { txAssetName = (e.currentTarget as HTMLInputElement).value; }} placeholder="Apple Inc." />
								</div>
								<div>
									<span class="text-xs text-tertiary">ISIN</span>
									<Input value={txIsin} oninput={(e: Event) => { txIsin = (e.currentTarget as HTMLInputElement).value; }} placeholder="US0378331005" />
								</div>
								<div>
									<span class="text-xs text-tertiary">Typ obchodu</span>
									<Select value={txType} onchange={(e: Event) => { txType = (e.currentTarget as HTMLSelectElement).value; }}>
										<option value="buy">Nakup</option>
										<option value="sell">Prodej</option>
									</Select>
								</div>
								<div>
									<span class="text-xs text-tertiary">Datum</span>
									<Input type="date" value={txDate} oninput={(e: Event) => { txDate = (e.currentTarget as HTMLInputElement).value; }} />
								</div>
								<div>
									<span class="text-xs text-tertiary">Pocet</span>
									<Input type="number" value={txQuantity} oninput={(e: Event) => { txQuantity = Number((e.currentTarget as HTMLInputElement).value); }} step="0.0001" />
								</div>
								<div>
									<span class="text-xs text-tertiary">Cena za kus</span>
									<Input type="number" value={txUnitPrice} oninput={(e: Event) => { txUnitPrice = Number((e.currentTarget as HTMLInputElement).value); }} step="0.01" />
								</div>
								<div>
									<span class="text-xs text-tertiary">Celkova castka</span>
									<Input type="number" value={txTotalAmount} oninput={(e: Event) => { txTotalAmount = Number((e.currentTarget as HTMLInputElement).value); }} step="0.01" />
								</div>
								<div>
									<span class="text-xs text-tertiary">Poplatky</span>
									<Input type="number" value={txFees} oninput={(e: Event) => { txFees = Number((e.currentTarget as HTMLInputElement).value); }} step="0.01" />
								</div>
								<div>
									<span class="text-xs text-tertiary">Mena</span>
									<Input value={txCurrency} oninput={(e: Event) => { txCurrency = (e.currentTarget as HTMLInputElement).value; }} placeholder="CZK" maxlength={3} />
								</div>
								<div>
									<span class="text-xs text-tertiary">Kurz CNB <HelpTip topic="kurz-cnb" /></span>
									<Input type="number" value={txExchangeRate} oninput={(e: Event) => { txExchangeRate = Number((e.currentTarget as HTMLInputElement).value); }} step="0.001" />
								</div>
							</div>
							<div class="mt-3 flex gap-2">
								<Button variant="primary" size="sm" onclick={saveTransaction} disabled={saving}>Ulozit</Button>
								<Button variant="ghost" size="sm" onclick={resetTransactionForm}>Zrusit</Button>
							</div>
						</div>
					{/if}
				</Card>
			{/if}

			<!-- Summary -->
			{#if summary}
				<Card>
					<h2 class="text-base font-semibold text-primary">Souhrn investicnich prijmu {selectedYear}</h2>
					<div class="mt-4 grid grid-cols-1 gap-6 md:grid-cols-2">
						<!-- §8 Capital income -->
						<div>
							<h3 class="text-sm font-medium text-tertiary">Kapitalove prijmy (§8)</h3>
							<div class="mt-2 space-y-1 text-sm">
								<div class="flex justify-between">
									<span class="text-tertiary">Hrube prijmy</span>
									<strong class="text-primary">{formatAmount(summary.capital_income_gross)}</strong>
								</div>
								<div class="flex justify-between">
									<span class="text-tertiary">Srazena dan</span>
									<strong class="text-primary">{formatAmount(summary.capital_income_tax)}</strong>
								</div>
								<div class="flex justify-between border-t border-border-subtle pt-1">
									<span class="text-tertiary">Ciste prijmy</span>
									<strong class="text-primary">{formatAmount(summary.capital_income_net)}</strong>
								</div>
							</div>
						</div>

						<!-- §10 Other income -->
						<div>
							<h3 class="text-sm font-medium text-tertiary">Ostatni prijmy - CP (§10)</h3>
							<div class="mt-2 space-y-1 text-sm">
								<div class="flex justify-between">
									<span class="text-tertiary">Hrube prijmy</span>
									<strong class="text-primary">{formatAmount(summary.other_income_gross)}</strong>
								</div>
								<div class="flex justify-between">
									<span class="text-tertiary">Vydaje (FIFO)</span>
									<strong class="text-primary">{formatAmount(summary.other_income_expenses)}</strong>
								</div>
								<div class="flex justify-between">
									<span class="text-tertiary">Osvobozeno (casovy test)</span>
									<strong class="text-primary">{formatAmount(summary.other_income_exempt)}</strong>
								</div>
								<div class="flex justify-between border-t border-border-subtle pt-1">
									<span class="text-tertiary">Zdanitelny prijem</span>
									<strong class="text-primary">{formatAmount(summary.other_income_net)}</strong>
								</div>
							</div>
						</div>
					</div>
				</Card>
			{/if}
		</div>
	{/if}
</div>
