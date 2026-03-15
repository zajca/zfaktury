<script lang="ts">
	import type { ExpenseDocument } from '$lib/api/client';
	import DocumentUpload from '$lib/components/DocumentUpload.svelte';
	import DocumentList from '$lib/components/DocumentList.svelte';
	import Card from '$lib/ui/Card.svelte';

	let {
		expenseId,
		documents,
		onuploaded,
		ondelete,
		onocr
	}: {
		expenseId: number;
		documents: ExpenseDocument[];
		onuploaded: () => void;
		ondelete: (id: number) => Promise<void>;
		onocr: (docId: number) => void;
	} = $props();
</script>

<Card>
	<h2 class="text-base font-semibold text-primary">Dokumenty</h2>
	<div class="mt-4">
		<DocumentUpload {expenseId} {onuploaded} />
		<div class="mt-4">
			<DocumentList {documents} {ondelete} {onocr} />
		</div>
	</div>
</Card>
