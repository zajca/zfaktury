<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { formatDate } from '$lib/utils/date';
	import { frequencyLabels } from '$lib/utils/invoice';
	import { recurringInvoicesApi, type RecurringInvoice } from '$lib/api/client';
	import Badge from '$lib/ui/Badge.svelte';
	import Button from '$lib/ui/Button.svelte';
	import ConfirmDialog from '$lib/ui/ConfirmDialog.svelte';
	import Card from '$lib/ui/Card.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import EmptyState from '$lib/ui/EmptyState.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';
	import HelpTip from '$lib/ui/HelpTip.svelte';
	import { toastSuccess, toastError } from '$lib/data/toast-state.svelte';

	let recurringInvoices = $state<RecurringInvoice[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let processing = $state(false);
	let showDeleteConfirm = $state(false);
	let deleteTargetId = $state<number | null>(null);

	async function loadRecurringInvoices() {
		loading = true;
		error = null;
		try {
			recurringInvoices = await recurringInvoicesApi.list();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodařilo se načíst opakující se faktury';
		} finally {
			loading = false;
		}
	}

	async function processDue() {
		processing = true;
		try {
			const data = await recurringInvoicesApi.processDue();
			if (data.generated_count > 0) {
				await loadRecurringInvoices();
			}
			toastSuccess(`Vygenerováno faktur: ${data.generated_count}`);
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se zpracovat splatné faktury');
		} finally {
			processing = false;
		}
	}

	function deleteRecurring(id: number) {
		deleteTargetId = id;
		showDeleteConfirm = true;
	}

	async function confirmDelete() {
		if (!deleteTargetId) return;
		showDeleteConfirm = false;
		try {
			await recurringInvoicesApi.delete(deleteTargetId);
			toastSuccess('Opakující se faktura smazána');
			await loadRecurringInvoices();
		} catch (e) {
			toastError(e instanceof Error ? e.message : 'Nepodařilo se smazat');
		} finally {
			deleteTargetId = null;
		}
	}

	onMount(() => {
		loadRecurringInvoices();
	});
</script>

<svelte:head>
	<title>Opakující se faktury - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-6xl">
	<PageHeader title="Opakující se faktury" description="Šablony pro automatické generování faktur">
		{#snippet actions()}
			<div class="flex gap-3">
				<Button
					variant="secondary"
					onclick={processDue}
					disabled={processing}
					title="Vygeneruje nové faktury ze všech aktivních šablon, jejichž datum dalšího vystavení už nastalo"
				>
					{processing ? 'Zpracovávám...' : 'Zpracovat splatné'}
				</Button>
				<Button variant="primary" href="/recurring/new">
					<svg
						class="h-4 w-4"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
					</svg>
					Nová opakující se faktura
				</Button>
			</div>
		{/snippet}
	</PageHeader>

	<p class="mt-2 text-sm text-tertiary">
		Šablony se automaticky převedou na nové faktury podle nastavené frekvence.
		<HelpTip topic="opakovane-faktury" />
	</p>

	<ErrorAlert {error} class="mt-4" />

	<Card padding={false} class="mt-4 overflow-hidden">
		{#if loading}
			<LoadingSpinner class="p-12" />
		{:else if recurringInvoices.length === 0}
			<EmptyState
				message="Zatím žádné opakující se faktury."
				actionHref="/recurring/new"
				actionLabel="Vytvořit šablonu"
			/>
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-border bg-elevated">
					<tr>
						<th class="th-default">Název</th>
						<th class="th-default">Zákazník</th>
						<th class="th-default hidden md:table-cell">Frekvence</th>
						<th class="th-default hidden md:table-cell">Další vystavení</th>
						<th class="th-default">Stav</th>
						<th class="th-default text-right">Akce</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border-subtle">
					{#each recurringInvoices as ri (ri.id)}
						<tr
							class="hover:bg-hover transition-colors cursor-pointer"
							role="link"
							tabindex="0"
							onclick={() => goto(`/recurring/${ri.id}`)}
							onkeydown={(e) => {
								if (e.key === 'Enter') goto(`/recurring/${ri.id}`);
							}}
						>
							<td class="px-4 py-2.5">
								<a href="/recurring/{ri.id}" class="text-accent-text hover:text-accent font-medium">
									{ri.name}
								</a>
							</td>
							<td class="px-4 py-2.5 text-secondary">
								{ri.customer?.name ?? '-'}
							</td>
							<td class="hidden px-4 py-2.5 text-secondary md:table-cell">
								{frequencyLabels[ri.frequency] ?? ri.frequency}
							</td>
							<td class="hidden px-4 py-2.5 text-secondary md:table-cell">
								{formatDate(ri.next_issue_date)}
							</td>
							<td class="px-4 py-2.5">
								<Badge variant={ri.is_active ? 'success' : 'muted'}>
									{ri.is_active ? 'Aktivní' : 'Neaktivní'}
								</Badge>
							</td>
							<td class="px-4 py-2.5 text-right">
								<button
									onclick={(e) => {
										e.stopPropagation();
										deleteRecurring(ri.id);
									}}
									class="text-sm text-danger hover:text-danger/80"
								>
									Smazat
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</Card>
</div>

<ConfirmDialog
	bind:open={showDeleteConfirm}
	title="Smazat opakující se fakturu"
	message="Opravdu chcete smazat tuto opakující se fakturu?"
	confirmLabel="Smazat"
	onconfirm={confirmDelete}
	oncancel={() => (showDeleteConfirm = false)}
/>
