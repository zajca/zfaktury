import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: {
		params: { id: '1' },
		url: { pathname: '/vat/vies/1', searchParams: new URLSearchParams() }
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

const sampleSummary = {
	id: 1,
	period: { year: 2026, month: 0, quarter: 1 },
	filing_type: 'regular',
	lines: [
		{
			id: 1,
			partner_dic: 'DE123456789',
			country_code: 'DE',
			total_amount: 5000000,
			service_code: '0'
		},
		{
			id: 2,
			partner_dic: 'AT987654321',
			country_code: 'AT',
			total_amount: 2500000,
			service_code: '3'
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
	(page as any).url = { pathname: '/vat/vies/1', searchParams: new URLSearchParams() };
});

afterEach(() => {
	cleanup();
});

describe('VIES Summary Detail', () => {
	it('loads summary on mount', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleSummary));

		render(Page);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/vies-summaries/1',
				expect.objectContaining({ method: 'GET' })
			);
		});
	});

	it('displays period in heading', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleSummary));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText(/Souhrnné hlášení 2026 Q1/)).toBeInTheDocument();
		});
	});

	it('shows status badge', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleSummary));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Připraveno')).toBeInTheDocument();
		});
	});

	it('shows filing type label', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleSummary));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Řádné')).toBeInTheDocument();
		});
	});

	it('renders lines table with correct columns', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleSummary));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Kód země')).toBeInTheDocument();
		});
		expect(screen.getByText('DIC partnera')).toBeInTheDocument();
		expect(screen.getByText('Celková částka (CZK)')).toBeInTheDocument();
		expect(screen.getByText('Kód plnění')).toBeInTheDocument();
	});

	it('shows line data', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleSummary));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('DE123456789')).toBeInTheDocument();
		});
		expect(screen.getByText('DE')).toBeInTheDocument();
		expect(screen.getByText('AT987654321')).toBeInTheDocument();
		expect(screen.getByText('AT')).toBeInTheDocument();
	});

	it('renders action buttons', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleSummary));

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
		const filedSummary = { ...sampleSummary, status: 'filed', filed_at: '2026-03-15T00:00:00Z' };
		mockFetch.mockResolvedValueOnce(jsonResponse(filedSummary));

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
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleSummary));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Přepočítat')).toBeInTheDocument();
		});

		mockFetch.mockResolvedValueOnce(jsonResponse(sampleSummary));

		await fireEvent.click(screen.getByText('Přepočítat'));

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/vies-summaries/1/recalculate',
				expect.objectContaining({ method: 'POST' })
			);
		});
	});

	it('calls delete API and navigates to /vat', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleSummary));

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
				'/api/v1/vies-summaries/1',
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
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleSummary));

		render(Page);

		const backLink = screen.getByText(/Zpět na DPH/);
		expect(backLink).toBeInTheDocument();
		expect(backLink.closest('a')?.getAttribute('href')).toBe('/vat');
	});

	it('shows empty state when no lines', async () => {
		const emptySum = { ...sampleSummary, lines: [] };
		mockFetch.mockResolvedValueOnce(jsonResponse(emptySum));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Žádné řádky v souhrnném hlášení')).toBeInTheDocument();
		});
	});

	it('shows filed date when summary is filed', async () => {
		const filedSummary = { ...sampleSummary, status: 'filed', filed_at: '2026-03-15T00:00:00Z' };
		mockFetch.mockResolvedValueOnce(jsonResponse(filedSummary));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText(/Podáno:/)).toBeInTheDocument();
		});
	});
});
