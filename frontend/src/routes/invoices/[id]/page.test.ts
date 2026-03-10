import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);
vi.stubGlobal('confirm', vi.fn(() => true));

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: {
		params: { id: '1' },
		url: { pathname: '/invoices/1', searchParams: new URLSearchParams() }
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

const sampleInvoice = {
	id: 1,
	sequence_id: 1,
	invoice_number: 'FV2026001',
	type: 'regular',
	status: 'draft',
	issue_date: '2026-03-01',
	due_date: '2026-03-15',
	delivery_date: '2026-03-01',
	variable_symbol: '2026001',
	constant_symbol: '',
	customer_id: 1,
	customer: {
		id: 1,
		name: 'Test Corp',
		ico: '12345678',
		dic: 'CZ12345678',
		type: 'company'
	},
	currency_code: 'CZK',
	exchange_rate: 100,
	payment_method: 'bank_transfer',
	bank_account: '123456789',
	bank_code: '0100',
	iban: 'CZ1234567890',
	swift: 'KOMBCZPP',
	subtotal_amount: 100000,
	vat_amount: 21000,
	total_amount: 121000,
	paid_amount: 0,
	notes: 'Test note',
	internal_notes: '',
	items: [
		{
			id: 1,
			invoice_id: 1,
			description: 'Web dev',
			quantity: 100,
			unit: 'hod',
			unit_price: 100000,
			vat_rate_percent: 21,
			vat_amount: 21000,
			total_amount: 121000,
			sort_order: 0
		}
	],
	created_at: '2026-03-01T00:00:00Z',
	updated_at: '2026-03-01T00:00:00Z'
};

beforeEach(async () => {
	mockFetch.mockReset();
	const { goto } = await import('$app/navigation');
	(goto as ReturnType<typeof vi.fn>).mockReset();
	const { page } = await import('$app/state');
	(page as any).params = { id: '1' };
	(page as any).url = { pathname: '/invoices/1', searchParams: new URLSearchParams() };
});

afterEach(() => {
	cleanup();
});

describe('Invoice detail page', () => {
	it('loads invoice on mount', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleInvoice));

		render(Page);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/invoices/1',
				expect.objectContaining({ method: 'GET' })
			);
		});
	});

	it('displays invoice number and status in view mode', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleInvoice));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Faktura FV2026001')).toBeInTheDocument();
		});

		expect(screen.getByText('Koncept')).toBeInTheDocument();
	});

	it('shows customer name', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleInvoice));

		render(Page);

		await waitFor(() => {
			const matches = screen.getAllByText('Test Corp');
			expect(matches.length).toBeGreaterThanOrEqual(1);
		});
	});

	it('shows Upravit and Odeslat buttons for draft status', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleInvoice));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Upravit')).toBeInTheDocument();
		});

		expect(screen.getByText('Odeslat')).toBeInTheDocument();
	});

	it('shows Uhrazena button for sent status', async () => {
		const sentInvoice = { ...sampleInvoice, status: 'sent' };
		mockFetch.mockResolvedValueOnce(jsonResponse(sentInvoice));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Uhrazená')).toBeInTheDocument();
		});

		// Upravit and Odeslat should NOT be present for sent status
		expect(screen.queryByText('Upravit')).not.toBeInTheDocument();
		expect(screen.queryByText('Odeslat')).not.toBeInTheDocument();
	});

	it('edit toggle switches to form mode', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleInvoice));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Upravit')).toBeInTheDocument();
		});

		// Mock contacts list for startEditing
		mockFetch.mockResolvedValueOnce(
			jsonResponse({ data: [], total: 0, limit: 1000, offset: 0 })
		);

		await fireEvent.click(screen.getByText('Upravit'));

		await waitFor(() => {
			expect(screen.getByText('Uložit změny')).toBeInTheDocument();
		});

		expect(screen.getByText('Zrušit')).toBeInTheDocument();
	});

	it('delete with confirmation calls invoicesApi.delete', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleInvoice));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Smazat')).toBeInTheDocument();
		});

		// Mock the DELETE call
		mockFetch.mockResolvedValueOnce(new Response(null, { status: 204, statusText: 'No Content' }));

		await fireEvent.click(screen.getByText('Smazat'));

		expect(confirm).toHaveBeenCalledWith('Opravdu chcete smazat tuto fakturu?');

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/invoices/1',
				expect.objectContaining({ method: 'DELETE' })
			);
		});

		const { goto } = await import('$app/navigation');
		expect(goto).toHaveBeenCalledWith('/invoices');
	});

	it('duplicate calls invoicesApi.duplicate', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleInvoice));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Duplikovat')).toBeInTheDocument();
		});

		const duplicatedInvoice = { ...sampleInvoice, id: 2, invoice_number: 'FV2026002' };
		mockFetch.mockResolvedValueOnce(jsonResponse(duplicatedInvoice));

		await fireEvent.click(screen.getByText('Duplikovat'));

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/invoices/1/duplicate',
				expect.objectContaining({ method: 'POST' })
			);
		});

		const { goto } = await import('$app/navigation');
		expect(goto).toHaveBeenCalledWith('/invoices/2');
	});

	it('shows error on load failure', async () => {
		mockFetch.mockResolvedValueOnce(
			jsonResponse({ error: 'Invoice not found' }, 404)
		);

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Invoice not found')).toBeInTheDocument();
		});
	});

	it('QR code section visible when invoice has iban', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleInvoice));

		render(Page);

		await waitFor(() => {
			expect(screen.getByAltText('QR kód pro platbu')).toBeInTheDocument();
		});
	});

	it('PDF download link present', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleInvoice));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Stáhnout PDF')).toBeInTheDocument();
		});

		const pdfLink = screen.getByText('Stáhnout PDF').closest('a');
		expect(pdfLink).toHaveAttribute('href', '/api/v1/invoices/1/pdf');
	});
});
