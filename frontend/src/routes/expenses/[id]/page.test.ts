import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: {
		params: { id: '1' },
		url: { pathname: '/expenses/1', searchParams: new URLSearchParams() }
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

const sampleExpense = {
	id: 1,
	vendor_id: 1,
	vendor: { id: 1, name: 'Vendor Corp' },
	expense_number: 'N2026001',
	category: 'office',
	description: 'Office supplies',
	issue_date: '2026-03-01',
	amount: 121000,
	currency_code: 'CZK',
	exchange_rate: 100,
	vat_rate_percent: 21,
	vat_amount: 21000,
	is_tax_deductible: true,
	business_percent: 100,
	payment_method: 'bank_transfer',
	notes: 'Test expense',
	created_at: '2026-03-01T00:00:00Z',
	updated_at: '2026-03-01T00:00:00Z'
};

const sampleCategories = [
	{ key: 'office', label_cs: 'Kancelář' },
	{ key: 'travel', label_cs: 'Cestovné' }
];

beforeEach(async () => {
	mockFetch.mockReset();
	const { goto } = await import('$app/navigation');
	(goto as ReturnType<typeof vi.fn>).mockReset();
	const { page } = await import('$app/state');
	(page as any).params = { id: '1' };
	(page as any).url = { pathname: '/expenses/1', searchParams: new URLSearchParams() };
});

afterEach(() => {
	cleanup();
});

describe('Expense detail page', () => {
	it('loads expense on mount', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleExpense));

		render(Page);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/expenses/1',
				expect.objectContaining({ method: 'GET' })
			);
		});
	});

	it('displays expense description as heading', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleExpense));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Office supplies')).toBeInTheDocument();
		});
	});

	it('shows expense number', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleExpense));

		render(Page);

		await waitFor(() => {
			const matches = screen.getAllByText(/N2026001/);
			expect(matches.length).toBeGreaterThanOrEqual(1);
		});
	});

	it('shows Upravit and Smazat buttons in view mode', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleExpense));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Upravit')).toBeInTheDocument();
		});

		expect(screen.getByText('Smazat')).toBeInTheDocument();
	});

	it('edit toggle switches to form', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleExpense));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Upravit')).toBeInTheDocument();
		});

		// startEditing loads contacts and CategoryPicker loads categories
		mockFetch.mockResolvedValueOnce(jsonResponse({ data: [], total: 0, limit: 1000, offset: 0 }));
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleCategories));

		await fireEvent.click(screen.getByText('Upravit'));

		await waitFor(() => {
			expect(screen.getByText('Uložit změny')).toBeInTheDocument();
		});

		expect(screen.getByText('Zrušit')).toBeInTheDocument();
	});

	it('delete with confirmation', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleExpense));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Smazat')).toBeInTheDocument();
		});

		mockFetch.mockResolvedValueOnce(new Response(null, { status: 204, statusText: 'No Content' }));

		await fireEvent.click(screen.getByText('Smazat'));

		await waitFor(() => {
			expect(screen.getByRole('alertdialog')).toBeInTheDocument();
		});
		const dialog = screen.getByRole('alertdialog');
		const confirmBtn = dialog.querySelectorAll('button')[1] as HTMLElement;
		await fireEvent.click(confirmBtn);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/expenses/1',
				expect.objectContaining({ method: 'DELETE' })
			);
		});

		const { goto } = await import('$app/navigation');
		expect(goto).toHaveBeenCalledWith('/expenses');
	});

	it('error on load failure', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ error: 'Expense not found' }, 404));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Expense not found')).toBeInTheDocument();
		});
	});
});
