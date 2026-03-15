<script lang="ts">
	import type { CapitalIncomeEntry } from '$lib/api/client';
	import { formatCZK } from '$lib/utils/money';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import Input from '$lib/ui/Input.svelte';
	import Select from '$lib/ui/Select.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';

	interface Props {
		capitalIncome: CapitalIncomeEntry[];
		saving: boolean;
		showCapitalForm: boolean;
		editingCapitalId: number | null;
		capitalCategory: string;
		capitalDescription: string;
		capitalDate: string;
		capitalGrossAmount: number;
		capitalWithheldCz: number;
		capitalWithheldForeign: number;
		capitalCountry: string;
		capitalNeedsDeclaring: boolean;
		onShowForm: () => void;
		onEdit: (entry: CapitalIncomeEntry) => void;
		onSave: () => void;
		onDelete: (id: number) => void;
		onCancel: () => void;
		onCategoryChange: (value: string) => void;
		onDescriptionChange: (value: string) => void;
		onDateChange: (value: string) => void;
		onGrossAmountChange: (value: number) => void;
		onWithheldCzChange: (value: number) => void;
		onWithheldForeignChange: (value: number) => void;
		onCountryChange: (value: string) => void;
		onNeedsDeclaringChange: (value: boolean) => void;
	}

	let {
		capitalIncome,
		saving,
		showCapitalForm,
		editingCapitalId,
		capitalCategory,
		capitalDescription,
		capitalDate,
		capitalGrossAmount,
		capitalWithheldCz,
		capitalWithheldForeign,
		capitalCountry,
		capitalNeedsDeclaring,
		onShowForm,
		onEdit,
		onSave,
		onDelete,
		onCancel,
		onCategoryChange,
		onDescriptionChange,
		onDateChange,
		onGrossAmountChange,
		onWithheldCzChange,
		onWithheldForeignChange,
		onCountryChange,
		onNeedsDeclaringChange
	}: Props = $props();

	const capitalCategoryLabels: Record<string, string> = {
		dividend_cz: 'Dividenda (CZ)',
		dividend_foreign: 'Dividenda (zahraniční)',
		interest: 'Úrok',
		coupon: 'Kupón',
		fund_distribution: 'Výplata z fondu',
		other: 'Ostatní'
	};

	function formatAmount(amountInHalere: number): string {
		return formatCZK(amountInHalere);
	}
</script>

<Card>
	<div class="flex items-center justify-between">
		<h2 class="text-base font-semibold text-primary">
			Kapitálové příjmy (§8) <HelpTip topic="kapitalove-prijmy-s8" />
		</h2>
		<Button variant="primary" size="sm" onclick={onShowForm}>Přidat ručně</Button>
	</div>

	{#if capitalIncome.length > 0}
		<div class="mt-4 overflow-x-auto">
			<table class="w-full text-sm">
				<thead>
					<tr class="border-b border-border text-left text-xs text-tertiary">
						<th class="pb-2 pr-4">Datum</th>
						<th class="pb-2 pr-4">Kategorie</th>
						<th class="pb-2 pr-4">Popis</th>
						<th class="pb-2 pr-4 text-right">Hrubá částka</th>
						<th class="pb-2 pr-4 text-right">Sražená daň <HelpTip topic="srazena-dan" /></th>
						<th class="pb-2 pr-4">K přiznání</th>
						<th class="pb-2 text-right">Akce</th>
					</tr>
				</thead>
				<tbody>
					{#each capitalIncome as entry (entry.id)}
						<tr class="border-b border-border-subtle">
							<td class="py-2 pr-4 text-tertiary">{entry.income_date}</td>
							<td class="py-2 pr-4">
								<span class="text-xs font-medium uppercase text-accent"
									>{capitalCategoryLabels[entry.category] ?? entry.category}</span
								>
							</td>
							<td class="py-2 pr-4 text-primary">{entry.description}</td>
							<td class="py-2 pr-4 text-right font-medium text-primary"
								>{formatAmount(entry.gross_amount)}</td
							>
							<td class="py-2 pr-4 text-right text-tertiary">
								{formatAmount(entry.withheld_tax_cz + entry.withheld_tax_foreign)}
							</td>
							<td class="py-2 pr-4">
								{#if entry.needs_declaring}
									<span
										class="inline-flex rounded-full bg-warning-bg px-2 py-0.5 text-xs font-medium text-warning"
										>Ano</span
									>
								{:else}
									<span
										class="inline-flex rounded-full bg-success-bg px-2 py-0.5 text-xs font-medium text-success"
										>Ne</span
									>
								{/if}
							</td>
							<td class="py-2 text-right">
								<div class="flex justify-end gap-1">
									<Button variant="ghost" size="sm" onclick={() => onEdit(entry)}>Upravit</Button>
									<Button
										variant="danger"
										size="sm"
										onclick={() => onDelete(entry.id)}
										disabled={saving}>Smazat</Button
									>
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
							{formatAmount(
								capitalIncome.reduce(
									(sum, e) => sum + e.withheld_tax_cz + e.withheld_tax_foreign,
									0
								)
							)}
						</td>
						<td colspan="2"></td>
					</tr>
				</tfoot>
			</table>
		</div>
	{:else}
		<p class="mt-4 text-sm text-tertiary">
			Žádné kapitálové příjmy. Přidejte ručně nebo nahrajte dokument k extrakci.
		</p>
	{/if}

	{#if showCapitalForm}
		<div class="mt-4 rounded-lg border border-border-subtle bg-elevated p-4">
			<h3 class="text-sm font-medium text-primary">
				{editingCapitalId ? 'Upravit záznam' : 'Přidat záznam'}
			</h3>
			<div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-3">
				<div>
					<span class="text-xs text-tertiary">Kategorie</span>
					<Select
						value={capitalCategory}
						onchange={(e: Event) => {
							onCategoryChange((e.currentTarget as HTMLSelectElement).value);
						}}
					>
						{#each Object.entries(capitalCategoryLabels) as [key, label]}
							<option value={key}>{label}</option>
						{/each}
					</Select>
				</div>
				<div>
					<span class="text-xs text-tertiary">Popis</span>
					<Input
						value={capitalDescription}
						oninput={(e: Event) => {
							onDescriptionChange((e.currentTarget as HTMLInputElement).value);
						}}
						placeholder="Název akcie / fondu"
					/>
				</div>
				<div>
					<span class="text-xs text-tertiary">Datum</span>
					<Input
						type="date"
						value={capitalDate}
						oninput={(e: Event) => {
							onDateChange((e.currentTarget as HTMLInputElement).value);
						}}
					/>
				</div>
				<div>
					<span class="text-xs text-tertiary">Hrubá částka (CZK)</span>
					<Input
						type="number"
						value={capitalGrossAmount}
						oninput={(e: Event) => {
							onGrossAmountChange(Number((e.currentTarget as HTMLInputElement).value));
						}}
						step="0.01"
					/>
				</div>
				<div>
					<span class="text-xs text-tertiary">Sražená daň ČR (CZK)</span>
					<Input
						type="number"
						value={capitalWithheldCz}
						oninput={(e: Event) => {
							onWithheldCzChange(Number((e.currentTarget as HTMLInputElement).value));
						}}
						step="0.01"
					/>
				</div>
				<div>
					<span class="text-xs text-tertiary">Sražená daň v zahraničí (CZK)</span>
					<Input
						type="number"
						value={capitalWithheldForeign}
						oninput={(e: Event) => {
							onWithheldForeignChange(Number((e.currentTarget as HTMLInputElement).value));
						}}
						step="0.01"
					/>
				</div>
				<div>
					<span class="text-xs text-tertiary">Země</span>
					<Input
						value={capitalCountry}
						oninput={(e: Event) => {
							onCountryChange((e.currentTarget as HTMLInputElement).value);
						}}
						placeholder="CZ"
						maxlength={2}
					/>
				</div>
				<label class="flex items-center gap-2 text-sm text-primary">
					<input
						type="checkbox"
						checked={capitalNeedsDeclaring}
						onchange={(e: Event) => {
							onNeedsDeclaringChange((e.currentTarget as HTMLInputElement).checked);
						}}
						class="rounded border-border"
					/>
					Nutno přiznat v DP <HelpTip topic="nutno-priznat-dp" />
				</label>
			</div>
			<div class="mt-3 flex gap-2">
				<Button variant="primary" size="sm" onclick={onSave} disabled={saving}>Uložit</Button>
				<Button variant="ghost" size="sm" onclick={onCancel}>Zrušit</Button>
			</div>
		</div>
	{/if}
</Card>
