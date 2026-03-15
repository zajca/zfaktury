<script lang="ts">
	import type { Contact } from '$lib/api/client';
	import type { FormItem } from '$lib/components/InvoiceItemsEditor.svelte';
	import { formatCZK, toHalere } from '$lib/utils/money';
	import CategoryPicker from '$lib/components/CategoryPicker.svelte';
	import DateInput from '$lib/components/DateInput.svelte';
	import InvoiceItemsEditor from '$lib/components/InvoiceItemsEditor.svelte';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';
	import Textarea from '$lib/ui/Textarea.svelte';

	let {
		form = $bindable(),
		items = $bindable(),
		useItems = $bindable(),
		contacts,
		saving,
		vatAmount,
		onsave,
		oncancel
	}: {
		form: {
			vendor_id: number | null;
			expense_number: string;
			category: string;
			description: string;
			issue_date: string;
			amount: number;
			currency_code: string;
			vat_rate_percent: number;
			is_tax_deductible: boolean;
			business_percent: number;
			payment_method: string;
			notes: string;
		};
		items: FormItem[];
		useItems: boolean;
		contacts: Contact[];
		saving: boolean;
		vatAmount: number;
		onsave: () => void;
		oncancel: () => void;
	} = $props();
</script>

<form
	onsubmit={(e) => {
		e.preventDefault();
		onsave();
	}}
	class="mt-6 space-y-6"
>
	<Card>
		<h2 class="text-base font-semibold text-primary">Základní údaje</h2>
		<div class="mt-4 space-y-4">
			<div>
				<label for="edit-desc" class="block text-sm font-medium text-secondary">Popis *</label>
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
					<label for="edit-cat" class="block text-sm font-medium text-secondary">Kategorie</label>
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
					<label for="edit-date" class="block text-sm font-medium text-secondary">Datum</label>
					<DateInput id="edit-date" bind:value={form.issue_date} required />
				</div>
				<div>
					<label for="edit-vendor" class="block text-sm font-medium text-secondary">Dodavatel</label
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
				<label for="edit-pm" class="block text-sm font-medium text-secondary">Způsob platby</label>
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

	<FormActions {saving} saveLabel="Uložit změny" {oncancel} />
</form>
