<script lang="ts">
	import type { Contact } from '$lib/api/client';
	import type { FormItem } from '$lib/components/InvoiceItemsEditor.svelte';
	import DateInput from '$lib/components/DateInput.svelte';
	import InvoiceItemsEditor from '$lib/components/InvoiceItemsEditor.svelte';
	import Card from '$lib/ui/Card.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import FormActions from '$lib/ui/FormActions.svelte';
	import Textarea from '$lib/ui/Textarea.svelte';

	let {
		form = $bindable(),
		items = $bindable(),
		contacts,
		saving,
		dueDateOffset = $bindable(),
		onsave,
		oncancel
	}: {
		form: {
			customer_id: number;
			issue_date: string;
			due_date: string;
			delivery_date: string;
			variable_symbol: string;
			constant_symbol: string;
			currency_code: string;
			payment_method: string;
			notes: string;
			internal_notes: string;
		};
		items: FormItem[];
		contacts: Contact[];
		saving: boolean;
		dueDateOffset: number;
		onsave: () => void;
		oncancel: () => void;
	} = $props();

	import { addDays } from '$lib/utils/date';

	function handleIssueDateChange(newValue: string) {
		form.issue_date = newValue;
		if (newValue) form.due_date = addDays(newValue, dueDateOffset);
	}

	function handleDueDateChange(newValue: string) {
		form.due_date = newValue;
		if (form.issue_date && newValue) {
			const diff = (new Date(newValue).getTime() - new Date(form.issue_date).getTime()) / 86400000;
			dueDateOffset = Math.round(diff);
		}
	}
</script>

<form
	onsubmit={(e) => {
		e.preventDefault();
		onsave();
	}}
	class="mt-6 space-y-6"
>
	<!-- Customer -->
	<Card>
		<h2 class="text-base font-semibold text-primary">Zákazník</h2>
		<div class="mt-4">
			<select
				bind:value={form.customer_id}
				class="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
			>
				<option value={0}>-- Vyberte --</option>
				{#each contacts as contact (contact.id)}
					<option value={contact.id}>{contact.name} {contact.ico ? `(${contact.ico})` : ''}</option>
				{/each}
			</select>
		</div>
	</Card>

	<!-- Dates -->
	<Card>
		<h2 class="text-base font-semibold text-primary">Údaje faktury</h2>
		<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
			<div>
				<label for="edit-issue" class="block text-sm font-medium text-secondary"
					>Datum vystavení</label
				>
				<DateInput
					id="edit-issue"
					bind:value={form.issue_date}
					required
					onchange={handleIssueDateChange}
				/>
			</div>
			<div>
				<label for="edit-due" class="block text-sm font-medium text-secondary"
					>Datum splatnosti <HelpTip topic="datum-splatnosti" /></label
				>
				<DateInput
					id="edit-due"
					bind:value={form.due_date}
					required
					onchange={handleDueDateChange}
					presets={[
						{ label: '+7 dní', days: 7 },
						{ label: '+14 dní', days: 14 },
						{ label: '+30 dní', days: 30 },
						{ label: '+60 dní', days: 60 }
					]}
					relativeToValue={form.issue_date}
				/>
			</div>
			<div>
				<label for="edit-delivery" class="block text-sm font-medium text-secondary"
					>DUZP <HelpTip topic="duzp" /></label
				>
				<DateInput id="edit-delivery" bind:value={form.delivery_date} />
			</div>
		</div>
		<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
			<div>
				<label for="edit-vs" class="block text-sm font-medium text-secondary"
					>Variabilní symbol <HelpTip topic="variabilni-symbol" /></label
				>
				<input
					id="edit-vs"
					type="text"
					bind:value={form.variable_symbol}
					class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
				/>
			</div>
			<div>
				<label for="edit-payment" class="block text-sm font-medium text-secondary"
					>Způsob platby <HelpTip topic="zpusob-platby" /></label
				>
				<select
					id="edit-payment"
					bind:value={form.payment_method}
					class="mt-1 w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-primary focus:border-accent focus:ring-1 focus:ring-accent/50 focus:outline-none"
				>
					<option value="bank_transfer">Bankovní převod</option>
					<option value="cash">Hotovost</option>
					<option value="card">Karta</option>
				</select>
			</div>
		</div>
	</Card>

	<!-- Items -->
	<InvoiceItemsEditor bind:items idPrefix="edit-" />

	<!-- Notes -->
	<Card>
		<h2 class="text-base font-semibold text-primary">Poznámky</h2>
		<div class="mt-4 space-y-4">
			<div>
				<label for="edit-notes" class="block text-sm font-medium text-secondary"
					>Poznámka na faktuře <HelpTip topic="poznamka-faktura" /></label
				>
				<Textarea id="edit-notes" bind:value={form.notes} rows={2} class="mt-1" />
			</div>
			<div>
				<label for="edit-internal" class="block text-sm font-medium text-secondary"
					>Interní poznámka <HelpTip topic="poznamka-interni" /></label
				>
				<Textarea id="edit-internal" bind:value={form.internal_notes} rows={2} class="mt-1" />
			</div>
		</div>
	</Card>

	<!-- Actions -->
	<FormActions {saving} saveLabel="Uložit změny" {oncancel} />
</form>
