import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);
vi.stubGlobal('alert', vi.fn());

import Page from './+page.svelte';

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const sampleResponse = {
	data: [
		{
			id: 1,
			name: 'Hosting',
			description: 'Monthly hosting',
			category: 'services',
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
			vendor_id: 1,
			vendor: { id: 1, name: 'Provider' },
			created_at: '2026-01-01',
			updated_at: '2026-03-01'
		}
	],
	total: 1,
	limit: 25,
	offset: 0
};

const emptyResponse = {
	data: [],
	total: 0,
	limit: 25,
	offset: 0
};

beforeEach(() => {
	mockFetch.mockReset();
	vi.mocked(alert).mockReset();
});

afterEach(() => {
	cleanup();
});

describe('Recurring expenses list page', () => {
	it('loads list on mount', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleResponse));

		render(Page);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalled();
		});

		const url = mockFetch.mock.calls[0][0] as string;
		expect(url).toContain('/api/v1/recurring-expenses');
	});

	it('renders rows with name, frequency, and amount', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleResponse));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Hosting')).toBeInTheDocument();
		});

		expect(screen.getByText('Monthly hosting')).toBeInTheDocument();
		expect(screen.getByText('Měsíčně')).toBeInTheDocument();
	});

	it('generate due button calls API and shows alert', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleResponse));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Vygenerovat splatné')).toBeInTheDocument();
		});

		// generate response, then reload
		mockFetch.mockResolvedValueOnce(jsonResponse({ generated: 3 }));
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleResponse));

		await fireEvent.click(screen.getByText('Vygenerovat splatné'));

		await waitFor(() => {
			const postCall = mockFetch.mock.calls.find(
				(call) =>
					(call[0] as string).includes('/api/v1/recurring-expenses/generate') &&
					call[1]?.method === 'POST'
			);
			expect(postCall).toBeTruthy();
		});

		await waitFor(() => {
			expect(alert).toHaveBeenCalledWith('Vygenerováno 3 nákladů.');
		});
	});

	it('shows empty state when no recurring expenses', async () => {
		mockFetch.mockResolvedValue(jsonResponse(emptyResponse));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Zatím žádné opakované náklady.')).toBeInTheDocument();
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
