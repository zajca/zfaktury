import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);
vi.stubGlobal(
	'confirm',
	vi.fn(() => true)
);

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: {
		params: { id: '1' },
		url: { pathname: '/vat/control/1', searchParams: new URLSearchParams() }
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

const sampleStatement = {
	id: 1,
	period: { year: 2026, month: 3, quarter: 1 },
	filing_type: 'regular',
	lines: [
		{
			id: 1,
			section: 'A4',
			partner_dic: 'CZ12345678',
			document_number: 'FV2026001',
			dppd: '2026-03-15',
			base: 1000000,
			vat: 210000,
			vat_rate_percent: 21,
			invoice_id: 1,
			expense_id: null
		},
		{
			id: 2,
			section: 'A5',
			partner_dic: '',
			document_number: '',
			dppd: '',
			base: 500000,
			vat: 105000,
			vat_rate_percent: 21,
			invoice_id: null,
			expense_id: null
		},
		{
			id: 3,
			section: 'B2',
			partner_dic: 'CZ87654321',
			document_number: 'N2026001',
			dppd: '2026-03-10',
			base: 200000,
			vat: 42000,
			vat_rate_percent: 21,
			invoice_id: null,
			expense_id: 1
		}
	],
	has_xml: true,
	status: 'ready',
	filed_at: null,
	created_at: '2026-03-01T00:00:00Z',
	updated_at: '2026-03-01T00:00:00Z'
};

beforeEach(async () => {
	mockFetch.mockReset();
	const { goto } = await import('$app/navigation');
	(goto as ReturnType<typeof vi.fn>).mockReset();
	const { page } = await import('$app/state');
	(page as any).params = { id: '1' };
	(page as any).url = { pathname: '/vat/control/1', searchParams: new URLSearchParams() };
});

afterEach(() => {
	cleanup();
});

describe('Control Statement Detail', () => {
	it('loads statement on mount', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleStatement));

		render(Page);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/vat-control-statements/1',
				expect.objectContaining({ method: 'GET' })
			);
		});
	});

	it('displays period in heading', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleStatement));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText(/Kontrolní hlášení 2026\/03/)).toBeInTheDocument();
		});
	});

	it('shows status badge', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleStatement));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Připraveno')).toBeInTheDocument();
		});
	});

	it('shows filing type label', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleStatement));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Řádné')).toBeInTheDocument();
		});
	});

	it('renders section tabs', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleStatement));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('A4 - Výstup nad 10 000')).toBeInTheDocument();
		});
		expect(screen.getByText('A5 - Výstup do 10 000')).toBeInTheDocument();
		expect(screen.getByText('B2 - Vstup nad 10 000')).toBeInTheDocument();
		expect(screen.getByText('B3 - Vstup do 10 000')).toBeInTheDocument();
	});

	it('shows A4 lines by default with partner info columns', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleStatement));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('CZ12345678')).toBeInTheDocument();
		});
		expect(screen.getByText('FV2026001')).toBeInTheDocument();
	});

	it('switches to A5 tab showing aggregated lines', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleStatement));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('A4 - Výstup nad 10 000')).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByText('A5 - Výstup do 10 000'));

		await waitFor(() => {
			// A5 should not show partner DIC column header
			const headers = document.querySelectorAll('th');
			const headerTexts = Array.from(headers).map((h) => h.textContent);
			expect(headerTexts).not.toContain('DIC partnera');
		});
	});

	it('switches to B2 tab showing input lines', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleStatement));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('A4 - Výstup nad 10 000')).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByText('B2 - Vstup nad 10 000'));

		await waitFor(() => {
			expect(screen.getByText('CZ87654321')).toBeInTheDocument();
		});
	});

	it('shows empty state for B3 tab', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleStatement));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('A4 - Výstup nad 10 000')).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByText('B3 - Vstup do 10 000'));

		await waitFor(() => {
			expect(screen.getByText('Žádné řádky v sekci B3')).toBeInTheDocument();
		});
	});

	it('renders action buttons', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleStatement));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Přepočítat')).toBeInTheDocument();
		});
		expect(screen.getByText('Generovat XML')).toBeInTheDocument();
		expect(screen.getByText('Stáhnout XML')).toBeInTheDocument();
		expect(screen.getByText('Označit za podané')).toBeInTheDocument();
		expect(screen.getByText('Smazat')).toBeInTheDocument();
	});

	it('disables buttons when status is filed', async () => {
		const filedStatement = {
			...sampleStatement,
			status: 'filed',
			filed_at: '2026-03-15T00:00:00Z'
		};
		mockFetch.mockResolvedValueOnce(jsonResponse(filedStatement));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Podáno')).toBeInTheDocument();
		});

		const recalcBtn = screen.getByText('Přepočítat');
		expect(recalcBtn).toBeDisabled();

		const genXmlBtn = screen.getByText('Generovat XML');
		expect(genXmlBtn).toBeDisabled();

		const markFiledBtn = screen.getByText('Označit za podané');
		expect(markFiledBtn).toBeDisabled();

		const deleteBtn = screen.getByText('Smazat');
		expect(deleteBtn).toBeDisabled();
	});

	it('calls recalculate API on button click', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleStatement));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Přepočítat')).toBeInTheDocument();
		});

		mockFetch.mockResolvedValueOnce(jsonResponse(sampleStatement));

		await fireEvent.click(screen.getByText('Přepočítat'));

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/vat-control-statements/1/recalculate',
				expect.objectContaining({ method: 'POST' })
			);
		});
	});

	it('calls delete API and navigates to /vat', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleStatement));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Smazat')).toBeInTheDocument();
		});

		mockFetch.mockResolvedValueOnce(new Response(null, { status: 204, statusText: 'No Content' }));

		await fireEvent.click(screen.getByText('Smazat'));

		expect(confirm).toHaveBeenCalledWith('Opravdu chcete smazat toto kontrolní hlášení?');

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/vat-control-statements/1',
				expect.objectContaining({ method: 'DELETE' })
			);
		});

		const { goto } = await import('$app/navigation');
		expect(goto).toHaveBeenCalledWith('/vat');
	});

	it('shows error on load failure', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ error: 'Not found' }, 404));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Not found')).toBeInTheDocument();
		});
	});

	it('shows loading spinner initially', () => {
		mockFetch.mockReturnValueOnce(new Promise(() => {}));

		render(Page);

		expect(screen.getByRole('status')).toBeInTheDocument();
	});

	it('shows back link to VAT page', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleStatement));

		render(Page);

		const backLink = screen.getByText(/Zpět na DPH/);
		expect(backLink).toBeInTheDocument();
		expect(backLink.closest('a')?.getAttribute('href')).toBe('/vat');
	});

	it('shows filed date when statement is filed', async () => {
		const filedStatement = {
			...sampleStatement,
			status: 'filed',
			filed_at: '2026-03-15T00:00:00Z'
		};
		mockFetch.mockResolvedValueOnce(jsonResponse(filedStatement));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText(/Podáno:/)).toBeInTheDocument();
		});
	});
});
