import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, cleanup } from '@testing-library/svelte';
import Page from './+page.svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const sampleExpenses = {
	data: [
		{
			id: 1,
			description: 'Office supplies',
			category: 'office',
			issue_date: '2026-03-01',
			amount: 50000
		},
		{
			id: 2,
			description: 'Travel',
			category: 'travel',
			issue_date: '2026-02-15',
			amount: 30000
		}
	],
	total: 2,
	limit: 25,
	offset: 0
};

const emptyExpenses = {
	data: [],
	total: 0,
	limit: 25,
	offset: 0
};

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('Expenses list page', () => {
	it('loads expenses on mount', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleExpenses));

		render(Page);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalled();
		});

		const url = mockFetch.mock.calls[0][0] as string;
		expect(url).toContain('/api/v1/expenses');
	});

	it('renders expense rows with description, category, and amount', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleExpenses));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Office supplies')).toBeInTheDocument();
		});

		expect(screen.getByText('Travel')).toBeInTheDocument();
		expect(screen.getByText('office')).toBeInTheDocument();
		expect(screen.getByText('travel')).toBeInTheDocument();
	});

	it('shows empty state message when no expenses', async () => {
		mockFetch.mockResolvedValue(jsonResponse(emptyExpenses));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Zatím žádné náklady.')).toBeInTheDocument();
		});
	});

	it('shows error state on API failure', async () => {
		mockFetch.mockRejectedValue(new Error('Network error'));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Network error')).toBeInTheDocument();
		});
	});

	it('has a search input', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleExpenses));

		render(Page);

		const searchInput = screen.getByPlaceholderText('Hledat podle popisu, dodavatele...');
		expect(searchInput).toBeInTheDocument();
	});
});
