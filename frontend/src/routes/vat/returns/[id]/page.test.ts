import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);
vi.stubGlobal('confirm', vi.fn(() => true));

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: {
		params: { id: '1' } as { id: string },
		url: { pathname: '/vat/returns/1', searchParams: new URLSearchParams() }
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

const sampleVatReturn = {
	id: 1,
	period: { year: 2026, month: 3, quarter: 1 },
	filing_type: 'regular',
	output_vat_base_21: 100000,
	output_vat_amount_21: 21000,
	output_vat_base_12: 50000,
	output_vat_amount_12: 6000,
	output_vat_base_0: 10000,
	reverse_charge_base_21: 0,
	reverse_charge_amount_21: 0,
	reverse_charge_base_12: 0,
	reverse_charge_amount_12: 0,
	input_vat_base_21: 80000,
	input_vat_amount_21: 16800,
	input_vat_base_12: 20000,
	input_vat_amount_12: 2400,
	total_output_vat: 27000,
	total_input_vat: 19200,
	net_vat: 7800,
	has_xml: false,
	status: 'draft',
	filed_at: null,
	created_at: '2026-03-01T00:00:00Z',
	updated_at: '2026-03-01T00:00:00Z'
};

const filedVatReturn = {
	...sampleVatReturn,
	status: 'filed',
	filed_at: '2026-03-10T00:00:00Z',
	has_xml: true
};

beforeEach(async () => {
	mockFetch.mockReset();
	const { page } = await import('$app/state');
	(page as any).params = { id: '1' };
});

afterEach(() => {
	cleanup();
});

describe('VAT return detail page', () => {
	it('loads VAT return on mount', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleVatReturn));

		render(Page);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalled();
		});

		const url = mockFetch.mock.calls[0][0] as string;
		expect(url).toContain('/api/v1/vat-returns/1');
	});

	it('renders heading with period', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleVatReturn));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText(/DPH Priznani - 3\/2026/)).toBeInTheDocument();
		});
	});

	it('renders status badge', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleVatReturn));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Koncept')).toBeInTheDocument();
		});
	});

	it('renders output VAT section', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleVatReturn));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Vystupni DPH')).toBeInTheDocument();
		});
	});

	it('renders input VAT section', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleVatReturn));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Vstupni DPH')).toBeInTheDocument();
		});
	});

	it('renders result section', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleVatReturn));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Vysledek')).toBeInTheDocument();
		});

		expect(screen.getByText(/Vlastni danova povinnost/)).toBeInTheDocument();
	});

	it('shows loading spinner while fetching', () => {
		mockFetch.mockReturnValue(new Promise(() => {}));

		render(Page);

		expect(screen.getByRole('status')).toBeInTheDocument();
	});

	it('shows error on fetch failure', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse('Not found', 404));

		render(Page);

		await waitFor(() => {
			expect(screen.getByRole('alert')).toBeInTheDocument();
		});
	});

	it('has back link to VAT dashboard', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleVatReturn));

		render(Page);

		const backLink = screen.getByText(/Zpet na DPH/);
		expect(backLink.closest('a')).toHaveAttribute('href', '/vat');
	});

	it('shows action buttons for draft status', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleVatReturn));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Prepocitat')).toBeInTheDocument();
		});

		expect(screen.getByText('Generovat XML')).toBeInTheDocument();
		expect(screen.getByText('Oznacit za podane')).toBeInTheDocument();
		expect(screen.getByText('Smazat')).toBeInTheDocument();
	});

	it('disables recalculate and generate for filed status', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(filedVatReturn));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Prepocitat')).toBeInTheDocument();
		});

		const recalcBtn = screen.getByText('Prepocitat');
		expect(recalcBtn).toBeDisabled();

		const generateBtn = screen.getByText('Generovat XML');
		expect(generateBtn).toBeDisabled();
	});

	it('hides mark-filed and delete buttons for filed status', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(filedVatReturn));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Prepocitat')).toBeInTheDocument();
		});

		expect(screen.queryByText('Oznacit za podane')).not.toBeInTheDocument();
		expect(screen.queryByText('Smazat')).not.toBeInTheDocument();
	});

	it('shows download XML button when has_xml is true', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(filedVatReturn));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Stahnout XML')).toBeInTheDocument();
		});
	});

	it('hides download XML button when has_xml is false', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleVatReturn));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Prepocitat')).toBeInTheDocument();
		});

		expect(screen.queryByText('Stahnout XML')).not.toBeInTheDocument();
	});

	it('recalculate button calls API', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleVatReturn));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Prepocitat')).toBeInTheDocument();
		});

		const recalculated = { ...sampleVatReturn, net_vat: 9000 };
		mockFetch.mockResolvedValueOnce(jsonResponse(recalculated));

		const btn = screen.getByText('Prepocitat');
		await fireEvent.click(btn);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledTimes(2);
		});

		const secondUrl = mockFetch.mock.calls[1][0] as string;
		expect(secondUrl).toContain('/api/v1/vat-returns/1/recalculate');
	});

	it('delete calls API and navigates away', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleVatReturn));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Smazat')).toBeInTheDocument();
		});

		mockFetch.mockResolvedValueOnce(new Response(null, { status: 200, statusText: 'OK' }));

		const btn = screen.getByText('Smazat');
		await fireEvent.click(btn);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledTimes(2);
		});

		const { goto } = await import('$app/navigation');
		expect(goto).toHaveBeenCalledWith('/vat');
	});
});
