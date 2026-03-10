import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

import Page from './+page.svelte';

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const sampleVatReturns = [
	{
		id: 1,
		period: { year: 2026, month: 3, quarter: 1 },
		filing_type: 'regular',
		output_vat_base_21: 100000,
		output_vat_amount_21: 21000,
		output_vat_base_12: 0,
		output_vat_amount_12: 0,
		output_vat_base_0: 0,
		reverse_charge_base_21: 0,
		reverse_charge_amount_21: 0,
		reverse_charge_base_12: 0,
		reverse_charge_amount_12: 0,
		input_vat_base_21: 50000,
		input_vat_amount_21: 10500,
		input_vat_base_12: 0,
		input_vat_amount_12: 0,
		total_output_vat: 21000,
		total_input_vat: 10500,
		net_vat: 10500,
		has_xml: false,
		status: 'draft',
		filed_at: null,
		created_at: '2026-03-01T00:00:00Z',
		updated_at: '2026-03-01T00:00:00Z'
	},
	{
		id: 2,
		period: { year: 2026, month: 0, quarter: 1 },
		filing_type: 'regular',
		output_vat_base_21: 200000,
		output_vat_amount_21: 42000,
		output_vat_base_12: 0,
		output_vat_amount_12: 0,
		output_vat_base_0: 0,
		reverse_charge_base_21: 0,
		reverse_charge_amount_21: 0,
		reverse_charge_base_12: 0,
		reverse_charge_amount_12: 0,
		input_vat_base_21: 100000,
		input_vat_amount_21: 21000,
		input_vat_base_12: 0,
		input_vat_amount_12: 0,
		total_output_vat: 42000,
		total_input_vat: 21000,
		net_vat: 21000,
		has_xml: true,
		status: 'filed',
		filed_at: '2026-03-10T00:00:00Z',
		created_at: '2026-03-01T00:00:00Z',
		updated_at: '2026-03-10T00:00:00Z'
	}
];

const sampleControlStatements = [
	{
		id: 10,
		period: { year: 2026, month: 3, quarter: 0 },
		filing_type: 'regular',
		lines: null,
		has_xml: false,
		status: 'draft',
		filed_at: null,
		created_at: '2026-03-01T00:00:00Z',
		updated_at: '2026-03-01T00:00:00Z'
	}
];

const sampleViesSummaries = [
	{
		id: 20,
		period: { year: 2026, month: 0, quarter: 1 },
		filing_type: 'regular',
		lines: null,
		has_xml: false,
		status: 'ready',
		filed_at: null,
		created_at: '2026-03-01T00:00:00Z',
		updated_at: '2026-03-01T00:00:00Z'
	}
];

// Mock all 3 API calls. The page calls them via Promise.all so order is:
// vat-returns, vat-control-statements, vies-summaries
function mockAllApis(vatReturns = sampleVatReturns, cs = sampleControlStatements, vies = sampleViesSummaries) {
	mockFetch.mockImplementation((url: string) => {
		if (typeof url === 'string' && url.includes('/vat-control-statements')) return Promise.resolve(jsonResponse(cs));
		if (typeof url === 'string' && url.includes('/vies-summaries')) return Promise.resolve(jsonResponse(vies));
		// Default to vat-returns for any URL containing vat-returns or as fallback
		return Promise.resolve(jsonResponse(vatReturns));
	});
}

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('VAT dashboard page', () => {
	it('loads data on mount', async () => {
		mockAllApis();

		render(Page);

		await waitFor(() => {
			// Verify data was loaded by checking rendered content
			expect(screen.getByText('3/2026')).toBeInTheDocument();
		});
	});

	it('renders page title and heading', async () => {
		mockAllApis();

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('DPH')).toBeInTheDocument();
		});
	});

	it('renders VAT return rows with period and status', async () => {
		mockAllApis();

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('3/2026')).toBeInTheDocument();
		});

		expect(screen.getByText('Q1/2026')).toBeInTheDocument();
		expect(screen.getByText('Koncept')).toBeInTheDocument();
		expect(screen.getByText('Podano')).toBeInTheDocument();
	});

	it('shows empty state when no VAT returns', async () => {
		mockAllApis([], [], []);

		render(Page);

		await waitFor(() => {
			expect(screen.getByText(/Zadna DPH priznani/)).toBeInTheDocument();
		});
	});

	it('shows loading spinner while fetching', () => {
		mockFetch.mockReturnValue(new Promise(() => {}));

		render(Page);

		expect(screen.getByRole('status')).toBeInTheDocument();
	});

	it('shows error on fetch failure', async () => {
		mockFetch.mockRejectedValue(new Error('Network error'));

		render(Page);

		await waitFor(() => {
			expect(screen.getByRole('alert')).toBeInTheDocument();
		});
	});

	it('renders tab buttons', async () => {
		mockAllApis();

		render(Page);

		expect(screen.getByText('DPH Priznani')).toBeInTheDocument();
		expect(screen.getByText('Kontrolni hlaseni')).toBeInTheDocument();
		expect(screen.getByText('Souhrnne hlaseni')).toBeInTheDocument();
	});

	it('switches to control statement tab and shows list', async () => {
		mockAllApis();

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('3/2026')).toBeInTheDocument();
		});

		const controlTab = screen.getByText('Kontrolni hlaseni');
		await fireEvent.click(controlTab);

		expect(screen.getByText('3/2026')).toBeInTheDocument();
	});

	it('switches to VIES tab and shows list', async () => {
		mockAllApis();

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('3/2026')).toBeInTheDocument();
		});

		const viesTab = screen.getByText('Souhrnne hlaseni');
		await fireEvent.click(viesTab);

		expect(screen.getByText('Q1/2026')).toBeInTheDocument();
	});

	it('has link to create new filing', async () => {
		mockAllApis();

		render(Page);

		await waitFor(() => {
			const link = screen.getByText('Nove priznani');
			expect(link.closest('a')).toHaveAttribute('href', '/vat/returns/new');
		});
	});

	it('clickable rows navigate to detail on Enter', async () => {
		mockAllApis();

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('3/2026')).toBeInTheDocument();
		});

		const tableRows = document.querySelectorAll('tr[role="link"]');
		expect(tableRows.length).toBeGreaterThan(0);

		await fireEvent.keyDown(tableRows[0], { key: 'Enter' });

		const { goto } = await import('$app/navigation');
		expect(goto).toHaveBeenCalledWith('/vat/returns/1');
	});
});
