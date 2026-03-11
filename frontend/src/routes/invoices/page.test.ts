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

const sampleInvoices = {
	data: [
		{
			id: 1,
			invoice_number: 'FV2026001',
			type: 'regular',
			status: 'draft',
			issue_date: '2026-03-01',
			due_date: '2026-03-15',
			total_amount: 121000,
			customer: { id: 1, name: 'Test Corp' }
		},
		{
			id: 2,
			invoice_number: 'FV2026002',
			type: 'regular',
			status: 'paid',
			issue_date: '2026-02-01',
			due_date: '2026-02-15',
			total_amount: 50000,
			customer: { id: 2, name: 'Acme s.r.o.' }
		}
	],
	total: 2,
	limit: 25,
	offset: 0
};

const mixedTypeInvoices = {
	data: [
		{
			id: 1,
			invoice_number: 'FV2026001',
			type: 'regular',
			status: 'paid',
			issue_date: '2026-03-01',
			due_date: '2026-03-15',
			total_amount: 121000,
			customer: { id: 1, name: 'Test Corp' }
		},
		{
			id: 2,
			invoice_number: 'ZF2026001',
			type: 'proforma',
			status: 'sent',
			issue_date: '2026-02-01',
			due_date: '2026-02-15',
			total_amount: 50000,
			customer: { id: 2, name: 'Acme s.r.o.' }
		},
		{
			id: 3,
			invoice_number: 'DP2026001',
			type: 'credit_note',
			status: 'draft',
			issue_date: '2026-02-10',
			due_date: '2026-02-24',
			total_amount: -30000,
			customer: { id: 1, name: 'Test Corp' }
		}
	],
	total: 3,
	limit: 25,
	offset: 0
};

const emptyInvoices = {
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

describe('Invoices list page', () => {
	it('loads invoices on mount', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleInvoices));

		render(Page);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalled();
		});

		const url = mockFetch.mock.calls[0][0] as string;
		expect(url).toContain('/api/v1/invoices');
	});

	it('renders invoice rows with number, customer, amount, and status', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleInvoices));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('FV2026001')).toBeInTheDocument();
		});

		expect(screen.getByText('FV2026002')).toBeInTheDocument();
		expect(screen.getByText('Test Corp')).toBeInTheDocument();
		expect(screen.getByText('Acme s.r.o.')).toBeInTheDocument();
		// 'Koncept' appears both in status filter dropdown and in the table row
		const konceptElements = screen.getAllByText('Koncept');
		expect(konceptElements.length).toBeGreaterThanOrEqual(2); // dropdown option + status badge
		expect(screen.getAllByText('Uhrazená').length).toBeGreaterThanOrEqual(1);
	});

	it('shows empty state message when no invoices', async () => {
		mockFetch.mockResolvedValue(jsonResponse(emptyInvoices));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Zatím žádné faktury.')).toBeInTheDocument();
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
		mockFetch.mockResolvedValue(jsonResponse(sampleInvoices));

		render(Page);

		const searchInput = screen.getByPlaceholderText('Hledat podle čísla, zákazníka...');
		expect(searchInput).toBeInTheDocument();
	});

	it('has a status filter dropdown', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleInvoices));

		render(Page);

		expect(screen.getByText('Všechny stavy')).toBeInTheDocument();
	});

	it('shows type badge for proforma and credit note invoices', async () => {
		mockFetch.mockResolvedValue(jsonResponse(mixedTypeInvoices));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('FV2026001')).toBeInTheDocument();
		});

		// "Zálohová faktura" appears in dropdown AND as badge, "Dobropis" also in both
		const proformaElements = screen.getAllByText('Zálohová faktura');
		expect(proformaElements.length).toBeGreaterThanOrEqual(2); // dropdown + badge
		const creditNoteElements = screen.getAllByText('Dobropis');
		expect(creditNoteElements.length).toBeGreaterThanOrEqual(2); // dropdown + badge

		// Verify badges are inside table cells (next to invoice numbers)
		const proformaRow = screen.getByText('ZF2026001').closest('td');
		expect(proformaRow?.textContent).toContain('Zálohová faktura');
		const creditNoteRow = screen.getByText('DP2026001').closest('td');
		expect(creditNoteRow?.textContent).toContain('Dobropis');
	});

	it('does not show type badge for regular invoices', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleInvoices));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('FV2026001')).toBeInTheDocument();
		});

		// Regular invoices should not have type badge in their table cell
		const regularRow = screen.getByText('FV2026001').closest('td');
		expect(regularRow?.textContent?.trim()).toBe('FV2026001');
	});

	it('hides pagination when single page', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleInvoices));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('FV2026001')).toBeInTheDocument();
		});

		expect(screen.queryByText('Předchozí')).not.toBeInTheDocument();
		expect(screen.queryByText('Další')).not.toBeInTheDocument();
	});
});
