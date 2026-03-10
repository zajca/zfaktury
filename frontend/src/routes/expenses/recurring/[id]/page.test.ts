import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);
vi.stubGlobal('confirm', vi.fn(() => true));

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: {
		params: { id: '1' },
		url: { pathname: '/expenses/recurring/1', searchParams: new URLSearchParams() }
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

const sampleItem = {
	id: 1,
	name: 'Hosting',
	vendor_id: 1,
	vendor: { id: 1, name: 'Provider' },
	category: 'services',
	description: 'Monthly hosting',
	amount: 50000,
	currency_code: 'CZK',
	exchange_rate: 100,
	vat_rate_percent: 21,
	vat_amount: 8678,
	is_tax_deductible: true,
	business_percent: 100,
	payment_method: 'bank_transfer',
	notes: '',
	frequency: 'monthly',
	next_issue_date: '2026-04-01',
	end_date: null,
	is_active: true,
	created_at: '2026-01-01',
	updated_at: '2026-03-01'
};

const sampleContacts = {
	data: [
		{ id: 1, name: 'Provider', ico: '12345678' }
	],
	total: 1,
	limit: 1000,
	offset: 0
};

const sampleCategories = [
	{ key: 'services', label_cs: 'Sluzby' }
];

beforeEach(async () => {
	mockFetch.mockReset();
	const { goto } = await import('$app/navigation');
	vi.mocked(goto).mockReset();
	vi.mocked(confirm).mockReturnValue(true);
	const { page } = await import('$app/state');
	(page as any).params = { id: '1' };
	(page as any).url = { pathname: '/expenses/recurring/1', searchParams: new URLSearchParams() };
});

afterEach(() => {
	cleanup();
});

describe('Recurring expense detail page', () => {
	it('loads item on mount', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleItem));

		render(Page);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/recurring-expenses/1',
				expect.objectContaining({ method: 'GET' })
			);
		});
	});

	it('shows item name and status', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleItem));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Hosting')).toBeInTheDocument();
		});

		expect(screen.getByText('Aktivní')).toBeInTheDocument();
		const matches = screen.getAllByText('Měsíčně');
		expect(matches.length).toBeGreaterThanOrEqual(1);
	});

	it('activate/deactivate toggle button calls API', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleItem));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Deaktivovat')).toBeInTheDocument();
		});

		// deactivate response, then reload
		mockFetch.mockResolvedValueOnce(jsonResponse(null, 200));
		mockFetch.mockResolvedValueOnce(jsonResponse({ ...sampleItem, is_active: false }));

		await fireEvent.click(screen.getByText('Deaktivovat'));

		await waitFor(() => {
			const deactivateCall = mockFetch.mock.calls.find(
				(call) =>
					(call[0] as string) === '/api/v1/recurring-expenses/1/deactivate' &&
					call[1]?.method === 'POST'
			);
			expect(deactivateCall).toBeTruthy();
		});
	});

	it('edit button switches to form', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleItem));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Upravit')).toBeInTheDocument();
		});

		// startEditing loads contacts, CategoryPicker loads categories
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleContacts));
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleCategories));

		await fireEvent.click(screen.getByText('Upravit'));

		await waitFor(() => {
			expect(screen.getByText('Uložit změny')).toBeInTheDocument();
		});

		expect(screen.getByText('Zrušit')).toBeInTheDocument();
	});

	it('delete with confirmation calls API and navigates', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleItem));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Smazat')).toBeInTheDocument();
		});

		mockFetch.mockResolvedValueOnce(new Response(null, { status: 204, statusText: 'No Content' }));

		await fireEvent.click(screen.getByText('Smazat'));

		expect(confirm).toHaveBeenCalledWith('Opravdu chcete smazat tento opakovaný náklad?');

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/recurring-expenses/1',
				expect.objectContaining({ method: 'DELETE' })
			);
		});

		await waitFor(async () => {
			const { goto } = await import('$app/navigation');
			expect(goto).toHaveBeenCalledWith('/expenses/recurring');
		});
	});

	it('error state on load failure', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ error: 'Not found' }, 404));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Not found')).toBeInTheDocument();
		});
	});
});
