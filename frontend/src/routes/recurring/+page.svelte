<script lang="ts">
	import { formatDate } from '$lib/utils/date';
	import Badge from '$lib/ui/Badge.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Card from '$lib/ui/Card.svelte';

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
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-xl font-semibold text-primary">Opakujici se faktury</h1>
			<p class="mt-1 text-sm text-tertiary">Sablony pro automaticke generovani faktur</p>
		</div>
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
	</div>

	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-danger/20 bg-danger-bg p-4 text-sm text-danger"
		>
			{error}
		</div>
	{/if}

	<Card padding={false} class="mt-4 overflow-hidden">
		{#if loading}
			<div class="flex items-center justify-center p-12">
				<div role="status">
					<div
						class="h-8 w-8 animate-spin rounded-full border-4 border-border border-t-accent"
					></div>
					<span class="sr-only">Nacitani...</span>
				</div>
			</div>
		{:else if recurringInvoices.length === 0}
			<div class="p-12 text-center text-muted">Zatim zadne opakujici se faktury.</div>
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-border bg-elevated">
					<tr>
						<th class="px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">Nazev</th>
						<th class="px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">Zakaznik</th>
						<th class="hidden px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted md:table-cell">Frekvence</th>
						<th class="hidden px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted md:table-cell">Dalsi vystaveni</th>
						<th class="px-4 py-2.5 text-xs font-medium uppercase tracking-wider text-muted">Stav</th>
						<th class="px-4 py-2.5 text-right text-xs font-medium uppercase tracking-wider text-muted">Akce</th>
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
