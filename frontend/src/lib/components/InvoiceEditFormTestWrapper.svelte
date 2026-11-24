<script lang="ts">
	import InvoiceEditForm from './invoice/InvoiceEditForm.svelte';

	let { dueDateOffset: initialOffset = 14 }: { dueDateOffset?: number } = $props();

	let form = $state({
		customer_id: 1,
		issue_date: '2026-03-10',
		due_date: '2026-03-24',
		delivery_date: '2026-03-10',
		variable_symbol: '',
		constant_symbol: '',
		currency_code: 'CZK',
		payment_method: 'bank_transfer',
		notes: '',
		internal_notes: ''
	});
	let items = $state([]);
	let dueDateOffset = $state(initialOffset);
</script>

<InvoiceEditForm
	bind:form
	bind:items
	bind:dueDateOffset
	contacts={[{ id: 1, name: 'Test Corp', ico: '12345678', type: 'company' } as never]}
	saving={false}
	onsave={() => {}}
	oncancel={() => {}}
/>
