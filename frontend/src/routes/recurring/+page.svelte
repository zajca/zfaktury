<script lang="ts">
	import { formatDate } from '$lib/utils/date';

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

<div>
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-gray-900">Opakujici se faktury</h1>
			<p class="mt-1 text-sm text-gray-500">Sablony pro automaticke generovani faktur</p>
		</div>
		<div class="flex gap-3">
			<button
				onclick={processDue}
				disabled={processing}
				class="inline-flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2.5 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 disabled:opacity-50 transition-colors"
			>
				{processing ? 'Zpracovavam...' : 'Zpracovat splatne'}
			</button>
			<a
				href="/recurring/new"
				class="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-blue-700 transition-colors"
			>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
				</svg>
				Nova opakujici se faktura
			</a>
		</div>
	</div>

	{#if error}
		<div
			role="alert"
			class="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700"
		>
			{error}
		</div>
	{/if}

	<div class="mt-4 overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
		{#if loading}
			<div class="flex items-center justify-center p-12">
				<div role="status">
					<div
						class="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-600"
					></div>
					<span class="sr-only">Nacitani...</span>
				</div>
			</div>
		{:else if recurringInvoices.length === 0}
			<div class="p-12 text-center text-gray-400">Zatim zadne opakujici se faktury.</div>
		{:else}
			<table class="w-full text-left text-sm">
				<thead class="border-b border-gray-200 bg-gray-50">
					<tr>
						<th class="px-4 py-3 font-medium text-gray-600">Nazev</th>
						<th class="px-4 py-3 font-medium text-gray-600">Zakaznik</th>
						<th class="hidden px-4 py-3 font-medium text-gray-600 md:table-cell">Frekvence</th>
						<th class="hidden px-4 py-3 font-medium text-gray-600 md:table-cell">Dalsi vystaveni</th
						>
						<th class="px-4 py-3 font-medium text-gray-600">Stav</th>
						<th class="px-4 py-3 text-right font-medium text-gray-600">Akce</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-100">
					{#each recurringInvoices as ri}
						<tr class="hover:bg-gray-50 transition-colors">
							<td class="px-4 py-3">
								<a href="/recurring/{ri.id}" class="font-medium text-blue-600 hover:text-blue-800">
									{ri.name}
								</a>
							</td>
							<td class="px-4 py-3 text-gray-700">
								{ri.customer?.name ?? '-'}
							</td>
							<td class="hidden px-4 py-3 text-gray-600 md:table-cell">
								{frequencyLabels[ri.frequency] ?? ri.frequency}
							</td>
							<td class="hidden px-4 py-3 text-gray-600 md:table-cell">
								{formatDate(ri.next_issue_date)}
							</td>
							<td class="px-4 py-3">
								<span
									class="inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium {ri.is_active
										? 'bg-green-100 text-green-700'
										: 'bg-gray-100 text-gray-500'}"
								>
									{ri.is_active ? 'Aktivni' : 'Neaktivni'}
								</span>
							</td>
							<td class="px-4 py-3 text-right">
								<button
									onclick={() => deleteRecurring(ri.id)}
									class="text-sm text-red-600 hover:text-red-800"
								>
									Smazat
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>
</div>
