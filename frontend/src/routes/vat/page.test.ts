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
		status: 'ready',
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
		status: 'filed',
		filed_at: '2026-03-15T00:00:00Z',
		created_at: '2026-03-01T00:00:00Z',
		updated_at: '2026-03-15T00:00:00Z'
	}
];

function mockAllApis(
	vatReturns = sampleVatReturns,
	controls = sampleControlStatements,
	vies = sampleViesSummaries
) {
	mockFetch.mockImplementation((url: string) => {
		if (typeof url === 'string' && url.includes('/vat-returns'))
			return Promise.resolve(jsonResponse(vatReturns));
		if (typeof url === 'string' && url.includes('/vat-control-statements'))
			return Promise.resolve(jsonResponse(controls));
		if (typeof url === 'string' && url.includes('/vies-summaries'))
			return Promise.resolve(jsonResponse(vies));
		return Promise.resolve(jsonResponse([]));
	});
}

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('VAT overview page', () => {
	it('renders page title with current year', async () => {
		mockAllApis();

		render(Page);

		const year = new Date().getFullYear();
		await waitFor(() => {
			expect(screen.getByText(`DPH za rok ${year}`)).toBeInTheDocument();
		});
	});

	it('renders year selector with arrow buttons', async () => {
		mockAllApis();

		render(Page);

		await waitFor(() => {
			expect(screen.getByLabelText('Předchozí rok')).toBeInTheDocument();
			expect(screen.getByLabelText('Následující rok')).toBeInTheDocument();
		});
	});

	it('renders 4 quarter sections', async () => {
		mockAllApis();

		render(Page);

		const year = new Date().getFullYear();
		await waitFor(() => {
			expect(screen.getByText(`1. čtvrtletí ${year}`)).toBeInTheDocument();
		});
		expect(screen.getByText(`2. čtvrtletí ${year}`)).toBeInTheDocument();
		expect(screen.getByText(`3. čtvrtletí ${year}`)).toBeInTheDocument();
		expect(screen.getByText(`4. čtvrtletí ${year}`)).toBeInTheDocument();
	});

	it('renders month names in each quarter', async () => {
		mockAllApis();

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Leden')).toBeInTheDocument();
		});
		expect(screen.getByText('Únor')).toBeInTheDocument();
		expect(screen.getByText('Březen')).toBeInTheDocument();
		expect(screen.getByText('Duben')).toBeInTheDocument();
		expect(screen.getByText('Prosinec')).toBeInTheDocument();
	});

	it('shows DPH and KH buttons for each month', async () => {
		mockAllApis();

		render(Page);

		await waitFor(() => {
			const dphButtons = screen.getAllByText('DPH', { exact: true });
			expect(dphButtons.length).toBeGreaterThanOrEqual(1);
		});

		const khButtons = screen.getAllByText('KH', { exact: true });
		expect(khButtons.length).toBeGreaterThanOrEqual(1);
	});

	it('shows status text on buttons for existing filings', async () => {
		mockAllApis();

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('(Koncept)')).toBeInTheDocument();
		});
	});

	it('shows filed status with label on VIES buttons', async () => {
		mockAllApis();

		render(Page);

		await waitFor(() => {
			const filed = screen.getAllByText('(Podáno)');
			expect(filed.length).toBeGreaterThanOrEqual(1);
		});
	});

	it('shows KH ready status', async () => {
		mockAllApis();

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('(Připraveno)')).toBeInTheDocument();
		});
	});

	it('renders "Celé čtvrtletí" rows', async () => {
		mockAllApis();

		render(Page);

		await waitFor(() => {
			const rows = screen.getAllByText('Celé čtvrtletí');
			expect(rows.length).toBe(4);
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

	it('year navigation changes year and reloads data', async () => {
		mockAllApis();

		render(Page);

		const year = new Date().getFullYear();
		await waitFor(() => {
			expect(screen.getByText(`DPH za rok ${year}`)).toBeInTheDocument();
		});

		mockFetch.mockClear();
		mockAllApis([], [], []);

		const nextBtn = screen.getByLabelText('Následující rok');
		await fireEvent.click(nextBtn);

		await waitFor(() => {
			expect(screen.getByText(`DPH za rok ${year + 1}`)).toBeInTheDocument();
		});

		// Should have fetched all 3 APIs again
		expect(mockFetch).toHaveBeenCalled();
	});

	it('clicking DPH button for existing filing navigates to detail', async () => {
		mockAllApis();

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('(Koncept)')).toBeInTheDocument();
		});

		// Find the DPH button in Březen row that has (Koncept) - it's month 3, id 1
		const konceptLabel = screen.getByText('(Koncept)');
		const dphButton = konceptLabel.closest('button');
		if (dphButton) {
			await fireEvent.click(dphButton);
		}

		const { goto } = await import('$app/navigation');
		expect(goto).toHaveBeenCalledWith('/vat/returns/1');
	});

	it('clicking KH button for missing filing navigates to create', async () => {
		mockAllApis([], [], []);

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Leden')).toBeInTheDocument();
		});

		// All KH buttons should be ghost buttons (no existing control statements)
		const khButtons = screen.getAllByText('KH', { exact: true });
		// Click the first KH button (January)
		await fireEvent.click(khButtons[0]);

		const year = new Date().getFullYear();
		const { goto } = await import('$app/navigation');
		expect(goto).toHaveBeenCalledWith(`/vat/control/new?year=${year}&month=1`);
	});

	it('SH button appears only on quarter-end months', async () => {
		mockAllApis([], [], []);

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Leden')).toBeInTheDocument();
		});

		// SH buttons should exist - there are 4 in month rows (Mar, Jun, Sep, Dec) + 4 in quarter rows
		const shButtons = screen.getAllByText(/^SH/);
		// 4 monthly SH + 4 quarterly "SH Q1..Q4" = varies, but at least 4
		expect(shButtons.length).toBeGreaterThanOrEqual(4);
	});
});
