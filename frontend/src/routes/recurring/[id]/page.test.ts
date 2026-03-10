import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: {
		params: { id: '1' },
		url: { pathname: '/recurring/1', searchParams: new URLSearchParams() }
	}
}));

import Page from './+page.svelte';

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const sampleRecurringInvoice = {
	id: 1,
	name: 'Mesicni hosting',
	customer_id: 1,
	customer: { id: 1, name: 'Test Corp', ico: '12345678' },
	frequency: 'monthly',
	next_issue_date: '2026-04-01',
	is_active: true,
	items: [
		{
			id: 1,
			recurring_invoice_id: 1,
			description: 'Hosting',
			quantity: 100,
			unit: 'ks',
			unit_price: 50000,
			vat_rate_percent: 21,
			sort_order: 0
		}
	],
	currency_code: 'CZK',
	exchange_rate: 100,
	payment_method: 'bank_transfer',
	bank_account: '',
	bank_code: '',
	iban: '',
	swift: '',
	constant_symbol: '',
	notes: '',
	created_at: '2026-01-01',
	updated_at: '2026-03-01'
};

const sampleContacts = {
	data: [
		{ id: 1, name: 'Test Corp', ico: '12345678' }
	],
	total: 1,
	limit: 1000,
	offset: 0
};

beforeEach(async () => {
	mockFetch.mockReset();
	const { goto } = await import('$app/navigation');
	(goto as ReturnType<typeof vi.fn>).mockReset();
	const { page } = await import('$app/state');
	(page as any).params = { id: '1' };
	(page as any).url = { pathname: '/recurring/1', searchParams: new URLSearchParams() };
});

afterEach(() => {
	cleanup();
});

describe('Recurring invoice detail page', () => {
	it('loads template by ID', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleRecurringInvoice));
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleContacts));

		render(Page);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalled();
		});

		const riCall = mockFetch.mock.calls.find(
			(call) => (call[0] as string) === '/api/v1/recurring-invoices/1'
		);
		expect(riCall).toBeTruthy();
	});

	it('shows template name and details in view mode', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleRecurringInvoice));
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleContacts));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Mesicni hosting')).toBeInTheDocument();
		});

		expect(screen.getByText('Test Corp')).toBeInTheDocument();
		expect(screen.getByText('Mesicni')).toBeInTheDocument();
		expect(screen.getByText('Aktivni')).toBeInTheDocument();
	});

	it('generate button calls POST /generate', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleRecurringInvoice));
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleContacts));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Vygenerovat fakturu')).toBeInTheDocument();
		});

		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 42 }));

		await fireEvent.click(screen.getByText('Vygenerovat fakturu'));

		await waitFor(() => {
			const generateCall = mockFetch.mock.calls.find(
				(call) =>
					(call[0] as string) === '/api/v1/recurring-invoices/1/generate' &&
					call[1]?.method === 'POST'
			);
			expect(generateCall).toBeTruthy();
		});

		await waitFor(async () => {
			const { goto } = await import('$app/navigation');
			expect(goto).toHaveBeenCalledWith('/invoices/42');
		});
	});

	it('edit button switches to edit mode', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleRecurringInvoice));
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleContacts));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Upravit')).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByText('Upravit'));

		await waitFor(() => {
			expect(screen.getByText(/Upravit: Mesicni hosting/)).toBeInTheDocument();
		});

		expect(screen.getByText('Ulozit zmeny')).toBeInTheDocument();
	});

	it('error state on load failure', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ error: 'Not found' }, 404));
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleContacts));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Nepodarilo se nacist opakujici se fakturu')).toBeInTheDocument();
		});
	});
});
