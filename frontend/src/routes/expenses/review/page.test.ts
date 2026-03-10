import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
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
			expense_number: 'EXP-001',
			category: 'office',
			description: 'Office supplies',
			issue_date: '2026-03-01',
			amount: 121000,
			currency_code: 'CZK',
			vat_amount: 21000,
			is_tax_deductible: true,
			tax_reviewed_at: null,
			vendor: { id: 1, name: 'Supplier Inc' }
		},
		{
			id: 2,
			expense_number: 'EXP-002',
			category: 'travel',
			description: 'Train tickets',
			issue_date: '2026-03-05',
			amount: 50000,
			currency_code: 'CZK',
			vat_amount: 8678,
			is_tax_deductible: true,
			tax_reviewed_at: '2026-03-08T10:00:00Z',
			vendor: null
		},
		{
			id: 3,
			expense_number: '',
			category: '',
			description: 'Lunch meeting',
			issue_date: '2026-03-07',
			amount: 35000,
			currency_code: 'CZK',
			vat_amount: 6074,
			is_tax_deductible: false,
			tax_reviewed_at: null,
			vendor: null
		}
	],
	total: 3,
	limit: 50,
	offset: 0
};

const emptyExpenses = {
	data: [],
	total: 0,
	limit: 50,
	offset: 0
};

beforeEach(() => {
	vi.useFakeTimers();
	vi.setSystemTime(new Date('2026-03-10T12:00:00Z'));
	mockFetch.mockReset();
	// Default: return sample expenses for the initial onMount load
	mockFetch.mockResolvedValue(jsonResponse(sampleExpenses));
});

afterEach(() => {
	cleanup();
	vi.useRealTimers();
});

describe('Expense Tax Review', () => {
	it('renders page heading', async () => {
		render(Page);
		expect(screen.getByText('Daňová kontrola nákladů')).toBeInTheDocument();
		expect(
			screen.getByText('Označte výdaje jako daňově zkontrolované')
		).toBeInTheDocument();
	});

	it('renders back link to expenses', () => {
		render(Page);
		const backLink = screen.getByText('Zpět na náklady');
		expect(backLink).toBeInTheDocument();
		expect(backLink.closest('a')?.getAttribute('href')).toBe('/expenses');
	});

	it('loads expenses on mount', async () => {
		render(Page);
		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				expect.stringContaining('/api/v1/expenses')
			);
		});
	});

	it('renders expense rows after loading', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Office supplies')).toBeInTheDocument();
		});
		expect(screen.getByText('Train tickets')).toBeInTheDocument();
		expect(screen.getByText('Lunch meeting')).toBeInTheDocument();
	});

	it('renders expense numbers or dash for empty', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('EXP-001')).toBeInTheDocument();
		});
		expect(screen.getByText('EXP-002')).toBeInTheDocument();
		// Expense 3 has no number, should show '-'
		const dashes = screen.getAllByText('-');
		expect(dashes.length).toBeGreaterThanOrEqual(1);
	});

	it('shows reviewed status for reviewed expenses', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Office supplies')).toBeInTheDocument();
		});
		// 'Zkontrolováno' appears in filter dropdown + 1 table badge (expense 2)
		const reviewed = screen.getAllByText('Zkontrolováno');
		expect(reviewed.length).toBe(2); // 1 filter option + 1 table badge
		// 'Nekontrolováno' appears in filter dropdown + 2 table badges (expenses 1 and 3)
		const unreviewed = screen.getAllByText('Nekontrolováno');
		expect(unreviewed.length).toBe(3); // 1 filter option + 2 table badges
	});

	it('renders vendor name or dash', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Supplier Inc')).toBeInTheDocument();
		});
	});

	it('shows empty state when no expenses match', async () => {
		mockFetch.mockResolvedValue(jsonResponse(emptyExpenses));

		render(Page);
		await waitFor(() => {
			expect(
				screen.getByText('Žádné výdaje neodpovídají filtrům.')
			).toBeInTheDocument();
		});
	});

	it('renders filter controls', () => {
		render(Page);
		expect(screen.getByLabelText('Datum od')).toBeInTheDocument();
		expect(screen.getByLabelText('Datum do')).toBeInTheDocument();
		expect(screen.getByLabelText('Stav kontroly')).toBeInTheDocument();
		expect(screen.getByText('Filtrovat')).toBeInTheDocument();
	});

	it('renders filter options for tax review status', () => {
		render(Page);
		expect(screen.getByText('Vše')).toBeInTheDocument();
		expect(screen.getByText('Nekontrolováno')).toBeInTheDocument();
		// "Zkontrolováno" appears both as filter option and in table rows
		// Just check the select has the option
		const select = document.querySelector('#tax-reviewed') as HTMLSelectElement;
		expect(select).toBeInTheDocument();
		expect(select.options.length).toBe(3);
	});

	it('filter button reloads expenses', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Office supplies')).toBeInTheDocument();
		});

		mockFetch.mockClear();
		mockFetch.mockResolvedValue(jsonResponse(sampleExpenses));

		const filterBtn = screen.getByText('Filtrovat');
		await fireEvent.click(filterBtn);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				expect.stringContaining('/api/v1/expenses')
			);
		});
	});

	it('select all checkbox toggles all rows', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Office supplies')).toBeInTheDocument();
		});

		const selectAllCheckbox = screen.getByTitle('Vybrat vše');
		await fireEvent.change(selectAllCheckbox);

		// Bulk action bar should appear
		await waitFor(() => {
			expect(screen.getByText(/Vybráno: 3 výdajů/)).toBeInTheDocument();
		});
	});

	it('individual checkbox toggles single selection', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Office supplies')).toBeInTheDocument();
		});

		// Get all row checkboxes (excluding "select all" in header)
		const checkboxes = document.querySelectorAll(
			'tbody input[type="checkbox"]'
		) as NodeListOf<HTMLInputElement>;
		expect(checkboxes.length).toBe(3);

		await fireEvent.change(checkboxes[0]);

		await waitFor(() => {
			expect(screen.getByText(/Vybráno: 1 výdajů/)).toBeInTheDocument();
		});
	});

	it('deselecting all hides bulk action bar', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Office supplies')).toBeInTheDocument();
		});

		// Select one
		const checkboxes = document.querySelectorAll(
			'tbody input[type="checkbox"]'
		) as NodeListOf<HTMLInputElement>;
		await fireEvent.change(checkboxes[0]);

		await waitFor(() => {
			expect(screen.getByText(/Vybráno: 1 výdajů/)).toBeInTheDocument();
		});

		// Deselect it
		await fireEvent.change(checkboxes[0]);

		await waitFor(() => {
			expect(screen.queryByText(/Vybráno:/)).not.toBeInTheDocument();
		});
	});

	it('bulk mark reviewed calls correct endpoint', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Office supplies')).toBeInTheDocument();
		});

		// Select all
		const selectAllCheckbox = screen.getByTitle('Vybrat vše');
		await fireEvent.change(selectAllCheckbox);

		await waitFor(() => {
			expect(screen.getByText(/Vybráno: 3 výdajů/)).toBeInTheDocument();
		});

		mockFetch.mockClear();
		mockFetch.mockResolvedValue(jsonResponse(sampleExpenses));

		const markBtn = screen.getByText('Označit jako zkontrolováno');
		await fireEvent.click(markBtn);

		await waitFor(() => {
			const reviewCall = mockFetch.mock.calls.find(
				(call: any[]) =>
					typeof call[0] === 'string' && call[0].includes('/api/v1/expenses/review')
			);
			expect(reviewCall).toBeDefined();
			if (reviewCall) {
				expect(reviewCall[1].method).toBe('POST');
				const body = JSON.parse(reviewCall[1].body);
				expect(body.ids).toEqual(expect.arrayContaining([1, 2, 3]));
				expect(body.ids.length).toBe(3);
			}
		});
	});

	it('bulk unmark reviewed calls correct endpoint', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Office supplies')).toBeInTheDocument();
		});

		// Select all
		const selectAllCheckbox = screen.getByTitle('Vybrat vše');
		await fireEvent.change(selectAllCheckbox);

		await waitFor(() => {
			expect(screen.getByText(/Vybráno: 3 výdajů/)).toBeInTheDocument();
		});

		mockFetch.mockClear();
		mockFetch.mockResolvedValue(jsonResponse(sampleExpenses));

		const unmarkBtn = screen.getByText('Odznačit');
		await fireEvent.click(unmarkBtn);

		await waitFor(() => {
			const unreviewCall = mockFetch.mock.calls.find(
				(call: any[]) =>
					typeof call[0] === 'string' &&
					call[0].includes('/api/v1/expenses/unreview')
			);
			expect(unreviewCall).toBeDefined();
			if (unreviewCall) {
				expect(unreviewCall[1].method).toBe('POST');
				const body = JSON.parse(unreviewCall[1].body);
				expect(body.ids).toEqual(expect.arrayContaining([1, 2, 3]));
			}
		});
	});

	it('shows success message after bulk mark reviewed', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Office supplies')).toBeInTheDocument();
		});

		// Select one expense
		const checkboxes = document.querySelectorAll(
			'tbody input[type="checkbox"]'
		) as NodeListOf<HTMLInputElement>;
		await fireEvent.change(checkboxes[0]);

		mockFetch.mockResolvedValue(jsonResponse(sampleExpenses));

		const markBtn = screen.getByText('Označit jako zkontrolováno');
		await fireEvent.click(markBtn);

		await waitFor(() => {
			expect(
				screen.getByText('1 výdajů označeno jako zkontrolováno')
			).toBeInTheDocument();
		});
	});

	it('shows success message after bulk unmark', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Office supplies')).toBeInTheDocument();
		});

		const checkboxes = document.querySelectorAll(
			'tbody input[type="checkbox"]'
		) as NodeListOf<HTMLInputElement>;
		await fireEvent.change(checkboxes[0]);
		await fireEvent.change(checkboxes[1]);

		mockFetch.mockResolvedValue(jsonResponse(sampleExpenses));

		const unmarkBtn = screen.getByText('Odznačit');
		await fireEvent.click(unmarkBtn);

		await waitFor(() => {
			expect(screen.getByText('2 výdajů odznačeno')).toBeInTheDocument();
		});
	});

	it('shows error on bulk action API failure', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Office supplies')).toBeInTheDocument();
		});

		const checkboxes = document.querySelectorAll(
			'tbody input[type="checkbox"]'
		) as NodeListOf<HTMLInputElement>;
		await fireEvent.change(checkboxes[0]);

		mockFetch.mockResolvedValue(jsonResponse({ error: 'Internal error' }, 500));

		const markBtn = screen.getByText('Označit jako zkontrolováno');
		await fireEvent.click(markBtn);

		await waitFor(() => {
			const errorDiv = document.querySelector('.text-red-700');
			expect(errorDiv).toBeInTheDocument();
		});
	});

	it('renders summary section with totals', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Office supplies')).toBeInTheDocument();
		});

		expect(screen.getByText('Celková částka')).toBeInTheDocument();
		expect(screen.getByText('Celkové DPH')).toBeInTheDocument();
	});

	it('shows selected amount summary when items are selected', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Office supplies')).toBeInTheDocument();
		});

		const checkboxes = document.querySelectorAll(
			'tbody input[type="checkbox"]'
		) as NodeListOf<HTMLInputElement>;
		await fireEvent.change(checkboxes[0]);

		await waitFor(() => {
			expect(screen.getByText('Vybráno - částka')).toBeInTheDocument();
			expect(screen.getByText('Vybráno - DPH')).toBeInTheDocument();
		});
	});

	it('shows error on initial load failure', async () => {
		mockFetch.mockResolvedValue(jsonResponse({ error: 'Server error' }, 500));

		render(Page);
		await waitFor(() => {
			const errorDiv = document.querySelector('.text-red-700');
			expect(errorDiv).toBeInTheDocument();
		});
	});

	it('clears selection when expenses are reloaded', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Office supplies')).toBeInTheDocument();
		});

		// Select an item
		const checkboxes = document.querySelectorAll(
			'tbody input[type="checkbox"]'
		) as NodeListOf<HTMLInputElement>;
		await fireEvent.change(checkboxes[0]);

		await waitFor(() => {
			expect(screen.getByText(/Vybráno: 1 výdajů/)).toBeInTheDocument();
		});

		// Apply filter to reload
		mockFetch.mockResolvedValue(jsonResponse(sampleExpenses));
		const filterBtn = screen.getByText('Filtrovat');
		await fireEvent.click(filterBtn);

		await waitFor(() => {
			expect(screen.queryByText(/Vybráno:/)).not.toBeInTheDocument();
		});
	});
});
