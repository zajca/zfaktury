<script lang="ts">
	import type { SecurityTransaction } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import Input from '$lib/ui/Input.svelte';
	import Select from '$lib/ui/Select.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';

	interface Props {
		transactions: SecurityTransaction[];
		saving: boolean;
		showTransactionForm: boolean;
		editingTransactionId: number | null;
		txAssetType: string;
		txAssetName: string;
		txIsin: string;
		txType: string;
		txDate: string;
		txQuantity: number;
		txUnitPrice: number;
		txTotalAmount: number;
		txFees: number;
		txCurrency: string;
		txExchangeRate: number;
		onShowForm: () => void;
		onRecalculateFifo: () => void;
		onEdit: (tx: SecurityTransaction) => void;
		onSave: () => void;
		onDelete: (id: number) => void;
		onCancel: () => void;
		onAssetTypeChange: (value: string) => void;
		onAssetNameChange: (value: string) => void;
		onIsinChange: (value: string) => void;
		onTypeChange: (value: string) => void;
		onDateChange: (value: string) => void;
		onQuantityChange: (value: number) => void;
		onUnitPriceChange: (value: number) => void;
		onTotalAmountChange: (value: number) => void;
		onFeesChange: (value: number) => void;
		onCurrencyChange: (value: string) => void;
		onExchangeRateChange: (value: number) => void;
	}

	let {
		transactions,
		saving,
		showTransactionForm,
		editingTransactionId,
		txAssetType,
		txAssetName,
		txIsin,
		txType,
		txDate,
		txQuantity,
		txUnitPrice,
		txTotalAmount,
		txFees,
		txCurrency,
		txExchangeRate,
		onShowForm,
		onRecalculateFifo,
		onEdit,
		onSave,
		onDelete,
		onCancel,
		onAssetTypeChange,
		onAssetNameChange,
		onIsinChange,
		onTypeChange,
		onDateChange,
		onQuantityChange,
		onUnitPriceChange,
		onTotalAmountChange,
		onFeesChange,
		onCurrencyChange,
		onExchangeRateChange
	}: Props = $props();

	const assetTypeLabels: Record<string, string> = {
		stock: 'Akcie',
		etf: 'ETF',
		bond: 'Dluhopis',
		fund: 'Fond',
		crypto: 'Kryptoměna',
		other: 'Jiný'
	};

	function formatAmount(amountInHalere: number): string {
		return formatCZK(amountInHalere);
	}
</script>

<Card>
	<div class="flex items-center justify-between">
		<h2 class="text-base font-semibold text-primary">
			Obchody s CP a kryptem (§10) <HelpTip topic="obchody-cp-s10" />
		</h2>
		<div class="flex gap-2">
			<Button variant="secondary" size="sm" onclick={onRecalculateFifo} disabled={saving}
				>Přepočítat FIFO</Button
			>
			<HelpTip topic="fifo-prepocet" />
			<Button variant="primary" size="sm" onclick={onShowForm}>Přidat ručně</Button>
		</div>
	</div>

	{#if transactions.length > 0}
		<div class="mt-4 overflow-x-auto">
			<table class="w-full text-sm">
				<thead>
					<tr class="border-b border-border text-left text-xs text-tertiary">
						<th class="pb-2 pr-3">Datum</th>
						<th class="pb-2 pr-3">Typ</th>
						<th class="pb-2 pr-3">Název</th>
						<th class="pb-2 pr-3">ISIN</th>
						<th class="pb-2 pr-3 text-right">Počet</th>
						<th class="pb-2 pr-3 text-right">Cena</th>
						<th class="pb-2 pr-3 text-right">Poplatky</th>
						<th class="pb-2 pr-3 text-right">Nabývací cena</th>
						<th class="pb-2 pr-3 text-right">Zisk/ztráta</th>
						<th class="pb-2 pr-3">Časový test <HelpTip topic="casovy-test" /></th>
						<th class="pb-2 text-right">Akce</th>
					</tr>
				</thead>
				<tbody>
					{#each transactions as tx (tx.id)}
						<tr class="border-b border-border-subtle">
							<td class="py-2 pr-3 text-tertiary">{tx.transaction_date}</td>
							<td class="py-2 pr-3">
								<span
									class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium {tx.transaction_type ===
									'buy'
										? 'bg-accent-muted text-accent-text'
										: 'bg-warning-bg text-warning'}"
								>
									{tx.transaction_type === 'buy' ? 'Nákup' : 'Prodej'}
								</span>
							</td>
							<td class="py-2 pr-3 text-primary">
								<span class="text-xs uppercase text-tertiary"
									>{assetTypeLabels[tx.asset_type] ?? tx.asset_type}</span
								>
								{tx.asset_name}
							</td>
							<td class="py-2 pr-3 font-mono text-xs text-tertiary">{tx.isin || '-'}</td>
							<td class="py-2 pr-3 text-right tabular-nums">{tx.quantity}</td>
							<td class="py-2 pr-3 text-right tabular-nums">{formatAmount(tx.total_amount)}</td>
							<td class="py-2 pr-3 text-right tabular-nums text-tertiary"
								>{formatAmount(tx.fees)}</td
							>
							<td class="py-2 pr-3 text-right tabular-nums">{formatAmount(tx.cost_basis)}</td>
							<td
								class="py-2 pr-3 text-right tabular-nums {tx.computed_gain >= 0
									? 'text-success'
									: 'text-danger'}"
							>
								{#if tx.transaction_type === 'sell'}
									{formatAmount(tx.computed_gain)}
								{:else}
									<span class="text-tertiary">-</span>
								{/if}
							</td>
							<td class="py-2 pr-3">
								{#if tx.time_test_exempt}
									<span
										class="inline-flex rounded-full bg-success-bg px-2 py-0.5 text-xs font-medium text-success"
										>Osv.</span
									>
								{:else if tx.transaction_type === 'sell'}
									<span
										class="inline-flex rounded-full bg-danger-bg px-2 py-0.5 text-xs font-medium text-danger"
										>Ne</span
									>
								{:else}
									<span class="text-tertiary">-</span>
								{/if}
							</td>
							<td class="py-2 text-right">
								<div class="flex justify-end gap-1">
									<Button variant="ghost" size="sm" onclick={() => onEdit(tx)}>Upravit</Button>
									<Button
										variant="danger"
										size="sm"
										onclick={() => onDelete(tx.id)}
										disabled={saving}>Smazat</Button
									>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<p class="mt-4 text-sm text-tertiary">
			Žádné obchody s cennými papíry. Přidejte ručně nebo nahrajte dokument k extrakci.
		</p>
	{/if}

	{#if showTransactionForm}
		<div class="mt-4 rounded-lg border border-border-subtle bg-elevated p-4">
			<h3 class="text-sm font-medium text-primary">
				{editingTransactionId ? 'Upravit obchod' : 'Přidat obchod'}
			</h3>
			<div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-4">
				<div>
					<span class="text-xs text-tertiary">Typ aktiva</span>
					<Select
						value={txAssetType}
						onchange={(e: Event) => {
							onAssetTypeChange((e.currentTarget as HTMLSelectElement).value);
						}}
					>
						{#each Object.entries(assetTypeLabels) as [key, label]}
							<option value={key}>{label}</option>
						{/each}
					</Select>
				</div>
				<div>
					<span class="text-xs text-tertiary">Název</span>
					<Input
						value={txAssetName}
						oninput={(e: Event) => {
							onAssetNameChange((e.currentTarget as HTMLInputElement).value);
						}}
						placeholder="Apple Inc."
					/>
				</div>
				<div>
					<span class="text-xs text-tertiary">ISIN</span>
					<Input
						value={txIsin}
						oninput={(e: Event) => {
							onIsinChange((e.currentTarget as HTMLInputElement).value);
						}}
						placeholder="US0378331005"
					/>
				</div>
				<div>
					<span class="text-xs text-tertiary">Typ obchodu</span>
					<Select
						value={txType}
						onchange={(e: Event) => {
							onTypeChange((e.currentTarget as HTMLSelectElement).value);
						}}
					>
						<option value="buy">Nákup</option>
						<option value="sell">Prodej</option>
					</Select>
				</div>
				<div>
					<span class="text-xs text-tertiary">Datum</span>
					<Input
						type="date"
						value={txDate}
						oninput={(e: Event) => {
							onDateChange((e.currentTarget as HTMLInputElement).value);
						}}
					/>
				</div>
				<div>
					<span class="text-xs text-tertiary">Počet</span>
					<Input
						type="number"
						value={txQuantity}
						oninput={(e: Event) => {
							onQuantityChange(Number((e.currentTarget as HTMLInputElement).value));
						}}
						step="0.0001"
					/>
				</div>
				<div>
					<span class="text-xs text-tertiary">Cena za kus</span>
					<Input
						type="number"
						value={txUnitPrice}
						oninput={(e: Event) => {
							onUnitPriceChange(Number((e.currentTarget as HTMLInputElement).value));
						}}
						step="0.01"
					/>
				</div>
				<div>
					<span class="text-xs text-tertiary">Celková částka</span>
					<Input
						type="number"
						value={txTotalAmount}
						oninput={(e: Event) => {
							onTotalAmountChange(Number((e.currentTarget as HTMLInputElement).value));
						}}
						step="0.01"
					/>
				</div>
				<div>
					<span class="text-xs text-tertiary">Poplatky</span>
					<Input
						type="number"
						value={txFees}
						oninput={(e: Event) => {
							onFeesChange(Number((e.currentTarget as HTMLInputElement).value));
						}}
						step="0.01"
					/>
				</div>
				<div>
					<span class="text-xs text-tertiary">Měna</span>
					<Input
						value={txCurrency}
						oninput={(e: Event) => {
							onCurrencyChange((e.currentTarget as HTMLInputElement).value);
						}}
						placeholder="CZK"
						maxlength={3}
					/>
				</div>
				<div>
					<span class="text-xs text-tertiary">Kurz ČNB <HelpTip topic="kurz-cnb" /></span>
					<Input
						type="number"
						value={txExchangeRate}
						oninput={(e: Event) => {
							onExchangeRateChange(Number((e.currentTarget as HTMLInputElement).value));
						}}
						step="0.001"
					/>
				</div>
			</div>
			<div class="mt-3 flex gap-2">
				<Button variant="primary" size="sm" onclick={onSave} disabled={saving}>Uložit</Button>
				<Button variant="ghost" size="sm" onclick={onCancel}>Zrušit</Button>
			</div>
		</div>
	{/if}
</Card>
