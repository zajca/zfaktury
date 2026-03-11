import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import Page from './+page.svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);
vi.stubGlobal(
	'confirm',
	vi.fn(() => true)
);
vi.stubGlobal('alert', vi.fn());

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const sampleRecurring = [
	{
		id: 1,
		name: 'Mesicni hosting',
		customer_id: 1,
		customer: { id: 1, name: 'Test Corp' },
		frequency: 'monthly',
		next_issue_date: '2026-04-01',
		is_active: true,
		items: [],
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
	},
	{
		id: 2,
		name: 'Rocni licence',
		customer_id: 2,
		customer: { id: 2, name: 'Acme' },
		frequency: 'yearly',
		next_issue_date: '2027-01-01',
		is_active: false,
		items: [],
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
	}
];

beforeEach(() => {
	mockFetch.mockReset();
	vi.mocked(confirm).mockReturnValue(true);
	vi.mocked(alert).mockReset();
});

afterEach(() => {
	cleanup();
});

describe('Recurring invoices list page', () => {
	it('loads recurring invoices on mount', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleRecurring));

		render(Page);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalled();
		});

		const url = mockFetch.mock.calls[0][0] as string;
		expect(url).toBe('/api/v1/recurring-invoices');
	});

	it('renders rows with name, customer, frequency label, and status', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleRecurring));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Mesicni hosting')).toBeInTheDocument();
		});

		expect(screen.getByText('Test Corp')).toBeInTheDocument();
		expect(screen.getByText('Mesicni')).toBeInTheDocument();
		expect(screen.getByText('Aktivni')).toBeInTheDocument();

		expect(screen.getByText('Rocni licence')).toBeInTheDocument();
		expect(screen.getByText('Acme')).toBeInTheDocument();
		expect(screen.getByText('Rocni')).toBeInTheDocument();
		expect(screen.getByText('Neaktivni')).toBeInTheDocument();
	});

	it('process due button calls POST and shows alert', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleRecurring));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Zpracovat splatne')).toBeInTheDocument();
		});

		// Process due returns count, then reload returns list
		mockFetch.mockResolvedValueOnce(jsonResponse({ generated_count: 2 }));
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleRecurring));

		await fireEvent.click(screen.getByText('Zpracovat splatne'));

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/recurring-invoices/process-due',
				expect.objectContaining({ method: 'POST' })
			);
		});

		await waitFor(() => {
			expect(alert).toHaveBeenCalledWith('Vygenerovano faktur: 2');
		});
	});

	it('delete with confirmation calls DELETE', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleRecurring));

		render(Page);

		await waitFor(() => {
			expect(screen.getAllByText('Smazat').length).toBeGreaterThan(0);
		});

		// Delete response, then reload
		mockFetch.mockResolvedValueOnce(jsonResponse(null, 200));
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleRecurring));

		await fireEvent.click(screen.getAllByText('Smazat')[0]);

		expect(confirm).toHaveBeenCalledWith('Opravdu chcete smazat tuto opakujici se fakturu?');

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/recurring-invoices/1',
				expect.objectContaining({ method: 'DELETE' })
			);
		});
	});

	it('shows empty state when no recurring invoices', async () => {
		mockFetch.mockResolvedValue(jsonResponse([]));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Zatim zadne opakujici se faktury.')).toBeInTheDocument();
		});
	});

	it('shows error state on API failure', async () => {
		mockFetch.mockRejectedValue(new Error('Network error'));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Network error')).toBeInTheDocument();
		});
	});
});
