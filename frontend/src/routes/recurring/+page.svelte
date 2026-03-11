<script lang="ts">
	import { formatDate } from '$lib/utils/date';
	import Badge from '$lib/ui/Badge.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';
	import LoadingSpinner from '$lib/ui/LoadingSpinner.svelte';
	import ErrorAlert from '$lib/ui/ErrorAlert.svelte';
	import EmptyState from '$lib/ui/EmptyState.svelte';
	import PageHeader from '$lib/ui/PageHeader.svelte';

	interface RecurringInvoiceItem {
		id: number;
		recurring_invoice_id: number;
		description: string;
		quantity: number;
		unit: string;
		unit_price: number;
		vat_rate_percent: number;
		sort_order: number;
	}

	interface RecurringInvoice {
		id: number;
		name: string;
		customer_id: number;
		customer?: { id: number; name: string };
		frequency: string;
		next_issue_date: string;
		end_date?: string;
		currency_code: string;
		exchange_rate: number;
		payment_method: string;
		bank_account: string;
		bank_code: string;
		iban: string;
		swift: string;
		constant_symbol: string;
		notes: string;
		is_active: boolean;
		items: RecurringInvoiceItem[];
		created_at: string;
		updated_at: string;
	}

	const API_BASE = '/api/v1/recurring-invoices';

	let recurringInvoices = $state<RecurringInvoice[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let processing = $state(false);

	const frequencyLabels: Record<string, string> = {
		weekly: 'Tydenni',
		monthly: 'Mesicni',
		quarterly: 'Ctvrtletni',
		yearly: 'Rocni'
	};

	async function loadRecurringInvoices() {
		loading = true;
		error = null;
		try {
			const res = await fetch(API_BASE);
			if (!res.ok) throw new Error('Nepodarilo se nacist opakujici se faktury');
			recurringInvoices = await res.json();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se nacist opakujici se faktury';
		} finally {
			loading = false;
		}
	}

	async function processDue() {
		processing = true;
		error = null;
		try {
			const res = await fetch(`${API_BASE}/process-due`, { method: 'POST' });
			if (!res.ok) throw new Error('Nepodarilo se zpracovat splatne faktury');
			const data = await res.json();
			if (data.generated_count > 0) {
				await loadRecurringInvoices();
			}
			alert(`Vygenerovano faktur: ${data.generated_count}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se zpracovat splatne faktury';
		} finally {
			processing = false;
		}
	}

	async function deleteRecurring(id: number) {
		if (!confirm('Opravdu chcete smazat tuto opakujici se fakturu?')) return;
		try {
			const res = await fetch(`${API_BASE}/${id}`, { method: 'DELETE' });
			if (!res.ok) throw new Error('Nepodarilo se smazat');
			await loadRecurringInvoices();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Nepodarilo se smazat';
		}
	}

	$effect(() => {
		loadRecurringInvoices();
	});
</script>

<svelte:head>
	<title>Opakujici se faktury - ZFaktury</title>
</svelte:head>

<div class="mx-auto max-w-6xl">
	<PageHeader title="Opakujici se faktury" description="Sablony pro automaticke generovani faktur">
		{#snippet actions()}
			<div class="flex gap-3">
				<Button variant="secondary" onclick={processDue} disabled={processing}>
					{processing ? 'Zpracovavam...' : 'Zpracovat splatne'}
				</Button>
				<Button variant="primary" href="/recurring/new">
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
					</svg>
					Nova opakujici se faktura
				</Button>
			</div>
		{/snippet}
	</PageHeader>

	<ErrorAlert {error} class="mt-4" />

	<Card padding={false} class="mt-4 overflow-hidden">
		{#if loading}
			<LoadingSpinner class="p-12" />
		{:else if recurringInvoices.length === 0}
			<EmptyState message="Zatim zadne opakujici se faktury." />
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-border bg-elevated">
					<tr>
						<th class="th-default">Nazev</th>
						<th class="th-default">Zakaznik</th>
						<th class="th-default hidden md:table-cell">Frekvence</th>
						<th class="th-default hidden md:table-cell">Dalsi vystaveni</th>
						<th class="th-default">Stav</th>
						<th class="th-default text-right">Akce</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border-subtle">
					{#each recurringInvoices as ri (ri.id)}
						<tr class="hover:bg-hover transition-colors cursor-pointer">
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
									{ri.is_active ? 'Aktivni' : 'Neaktivni'}
								</Badge>
							</td>
							<td class="px-4 py-2.5 text-right">
								<button
									onclick={() => deleteRecurring(ri.id)}
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
