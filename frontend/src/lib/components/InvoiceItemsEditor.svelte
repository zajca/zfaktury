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

<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
	<div class="flex items-center justify-between">
		<h2 class="text-lg font-semibold text-gray-900">Položky</h2>
		<button
			type="button"
			onclick={addItem}
			class="inline-flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
			</svg>
			Přidat položku
		</button>
	</div>

	<div class="mt-4 space-y-4">
		{#each items as item, index}
			<div class="rounded-lg border border-gray-200 bg-gray-50 p-4">
				<div class="flex items-start gap-4">
					<div class="flex-1 space-y-3">
						<div>
							<label for="{idPrefix}desc-{index}" class="block text-sm font-medium text-gray-700"
								>Popis</label
							>
							<input
								id="{idPrefix}desc-{index}"
								type="text"
								bind:value={item.description}
								required
								class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white"
							/>
						</div>
						<div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
							<div>
								<label for="{idPrefix}qty-{index}" class="block text-sm font-medium text-gray-700"
									>Množství</label
								>
								<input
									id="{idPrefix}qty-{index}"
									type="number"
									step="0.01"
									min="0"
									bind:value={item.quantity}
									class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white"
								/>
							</div>
							<div>
								<label for="{idPrefix}unit-{index}" class="block text-sm font-medium text-gray-700"
									>Jednotka</label
								>
								<select
									id="{idPrefix}unit-{index}"
									bind:value={item.unit}
									class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white"
								>
									<option value="ks">ks</option>
									<option value="hod">hod</option>
									<option value="m2">m2</option>
									<option value="den">den</option>
									<option value="mesic">měsíc</option>
								</select>
							</div>
							<div>
								<label for="{idPrefix}price-{index}" class="block text-sm font-medium text-gray-700"
									>Cena/ks (CZK)</label
								>
								<input
									id="{idPrefix}price-{index}"
									type="number"
									step="0.01"
									min="0"
									bind:value={item.unit_price}
									class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white"
								/>
							</div>
							<div>
								<label for="{idPrefix}vat-{index}" class="block text-sm font-medium text-gray-700"
									>DPH %</label
								>
								<select
									id="{idPrefix}vat-{index}"
									bind:value={item.vat_rate_percent}
									class="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none bg-white"
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
							class="mt-6 rounded p-1 text-gray-400 hover:text-red-500 transition-colors"
							aria-label="Odebrat položku"
						>
							<svg
								class="h-5 w-5"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="2"
							>
								<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
							</svg>
						</button>
					{/if}
				</div>
				<!-- Item subtotal -->
				<div class="mt-2 text-right text-sm text-gray-500">
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
	<div class="mt-6 border-t border-gray-200 pt-4">
		<div class="flex flex-col items-end gap-1 text-sm">
			<div class="flex gap-8">
				<span class="text-gray-600">Základ:</span>
				<span class="font-medium text-gray-900">{formatCZK(toHalere(subtotal))}</span>
			</div>
			<div class="flex gap-8">
				<span class="text-gray-600">DPH:</span>
				<span class="font-medium text-gray-900">{formatCZK(toHalere(vatTotal))}</span>
			</div>
			<div class="flex gap-8 border-t border-gray-200 pt-1 text-base">
				<span class="font-semibold text-gray-900">Celkem:</span>
				<span class="font-bold text-gray-900">{formatCZK(toHalere(grandTotal))}</span>
			</div>
		</div>
	</div>
</div>
