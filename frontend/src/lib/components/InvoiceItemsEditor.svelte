<script lang="ts" module>
	export interface FormItem {
		description: string;
		quantity: number;
		unit: string;
		unit_price: number; // in crowns (user input)
		vat_rate_percent: number;
	}

	export function calcSubtotal(items: FormItem[]): number {
		return items.reduce((sum, item) => sum + item.quantity * item.unit_price, 0);
	}

	export function calcVatTotal(items: FormItem[]): number {
		return items.reduce((sum, item) => {
			const itemSubtotal = item.quantity * item.unit_price;
			return sum + itemSubtotal * (item.vat_rate_percent / 100);
		}, 0);
	}

	export function calcGrandTotal(items: FormItem[]): number {
		return calcSubtotal(items) + calcVatTotal(items);
	}
</script>

<script lang="ts">
	import { formatCZK, toHalere } from '$lib/utils/money';

	interface Props {
		items: FormItem[];
		idPrefix?: string;
	}

	let { items = $bindable(), idPrefix = '' }: Props = $props();

	let subtotal = $derived(calcSubtotal(items));
	let vatTotal = $derived(calcVatTotal(items));
	let grandTotal = $derived(calcGrandTotal(items));

	function addItem() {
		items = [
			...items,
			{ description: '', quantity: 1, unit: 'ks', unit_price: 0, vat_rate_percent: 21 }
		];
	}

	function removeItem(index: number) {
		if (items.length <= 1) return;
		items = items.filter((_, i) => i !== index);
	}
</script>

<div class="rounded-lg border border-border bg-surface p-5">
	<div class="flex items-center justify-between">
		<h2 class="text-base font-semibold text-primary">Položky</h2>
		<button
			type="button"
			onclick={addItem}
			class="inline-flex items-center gap-1.5 rounded-md border border-border-strong px-2.5 py-1.5 text-xs font-medium text-secondary hover:bg-hover hover:text-primary transition-colors"
		>
			<svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
			</svg>
			Přidat položku
		</button>
	</div>

	<div class="mt-4 space-y-3">
		{#each items as item, index (index)}
			<div class="rounded-lg border border-border bg-elevated p-4">
				<div class="flex items-start gap-4">
					<div class="flex-1 space-y-3">
						<div>
							<label for="{idPrefix}desc-{index}" class="block text-sm font-medium text-secondary">Popis</label>
							<input
								id="{idPrefix}desc-{index}"
								type="text"
								bind:value={item.description}
								required
								class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary placeholder:text-muted focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
							/>
						</div>
						<div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
							<div>
								<label for="{idPrefix}qty-{index}" class="block text-sm font-medium text-secondary">Množství</label>
								<input
									id="{idPrefix}qty-{index}"
									type="number"
									step="0.01"
									min="0"
									bind:value={item.quantity}
									class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary font-mono tabular-nums focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
								/>
							</div>
							<div>
								<label for="{idPrefix}unit-{index}" class="block text-sm font-medium text-secondary">Jednotka</label>
								<select
									id="{idPrefix}unit-{index}"
									bind:value={item.unit}
									class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
								>
									<option value="ks">ks</option>
									<option value="hod">hod</option>
									<option value="m2">m2</option>
									<option value="den">den</option>
									<option value="mesic">měsíc</option>
								</select>
							</div>
							<div>
								<label for="{idPrefix}price-{index}" class="block text-sm font-medium text-secondary">Cena/ks (CZK)</label>
								<input
									id="{idPrefix}price-{index}"
									type="number"
									step="0.01"
									min="0"
									bind:value={item.unit_price}
									class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary font-mono tabular-nums focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
								/>
							</div>
							<div>
								<label for="{idPrefix}vat-{index}" class="block text-sm font-medium text-secondary">DPH %</label>
								<select
									id="{idPrefix}vat-{index}"
									bind:value={item.vat_rate_percent}
									class="mt-1 w-full rounded-lg border border-border bg-elevated px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
								>
									<option value={21}>21%</option>
									<option value={12}>12%</option>
									<option value={0}>0%</option>
								</select>
							</div>
						</div>
					</div>
					{#if items.length > 1}
						<button
							type="button"
							onclick={() => removeItem(index)}
							class="mt-6 rounded p-1 text-muted hover:text-danger transition-colors"
							aria-label="Odebrat položku"
						>
							<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
								<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
							</svg>
						</button>
					{/if}
				</div>
				<!-- Item subtotal -->
				<div class="mt-2 text-right text-xs text-tertiary font-mono tabular-nums">
					Základ: {formatCZK(toHalere(item.quantity * item.unit_price))} | DPH: {formatCZK(
						toHalere((item.quantity * item.unit_price * item.vat_rate_percent) / 100)
					)} | Celkem: {formatCZK(
						toHalere(item.quantity * item.unit_price * (1 + item.vat_rate_percent / 100))
					)}
				</div>
			</div>
		{/each}
	</div>

	<!-- Totals -->
	<div class="mt-5 border-t border-border pt-3">
		<div class="flex flex-col items-end gap-1 text-sm">
			<div class="flex gap-8">
				<span class="text-tertiary">Základ:</span>
				<span class="font-medium text-primary font-mono tabular-nums">{formatCZK(toHalere(subtotal))}</span>
			</div>
			<div class="flex gap-8">
				<span class="text-tertiary">DPH:</span>
				<span class="font-medium text-primary font-mono tabular-nums">{formatCZK(toHalere(vatTotal))}</span>
			</div>
			<div class="flex gap-8 border-t border-border pt-1 text-base">
				<span class="font-semibold text-primary">Celkem:</span>
				<span class="font-semibold text-primary font-mono tabular-nums">{formatCZK(toHalere(grandTotal))}</span>
			</div>
		</div>
	</div>
</div>
