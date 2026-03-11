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

const createdReturn = {
	id: 5,
	period: { year: 2026, month: 3, quarter: 0 },
	filing_type: 'regular',
	output_vat_base_21: 0,
	output_vat_amount_21: 0,
	output_vat_base_12: 0,
	output_vat_amount_12: 0,
	output_vat_base_0: 0,
	reverse_charge_base_21: 0,
	reverse_charge_amount_21: 0,
	reverse_charge_base_12: 0,
	reverse_charge_amount_12: 0,
	input_vat_base_21: 0,
	input_vat_amount_21: 0,
	input_vat_base_12: 0,
	input_vat_amount_12: 0,
	total_output_vat: 0,
	total_input_vat: 0,
	net_vat: 0,
	has_xml: false,
	status: 'draft',
	filed_at: null,
	created_at: '2026-03-10T00:00:00Z',
	updated_at: '2026-03-10T00:00:00Z'
};

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('New VAT return page', () => {
	it('renders page heading', () => {
		render(Page);

		expect(screen.getByText('Nové DPH přiznání')).toBeInTheDocument();
	});

	it('renders form fields for year, period type toggle, and filing type', () => {
		render(Page);

		expect(screen.getByLabelText('Rok')).toBeInTheDocument();
		// Month/quarter are behind a toggle; month is shown by default
		expect(screen.getByText('Měsíční')).toBeInTheDocument();
		expect(screen.getByText('Čtvrtletní')).toBeInTheDocument();
		expect(screen.getByLabelText('Měsíc')).toBeInTheDocument();
		expect(screen.getByLabelText('Typ podání')).toBeInTheDocument();
	});

	it('has back link to VAT dashboard', () => {
		render(Page);

		const backLink = screen.getByText(/Zpět na DPH/);
		expect(backLink.closest('a')).toHaveAttribute('href', '/vat');
	});

	it('submits form and navigates to detail on success', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(createdReturn));

		render(Page);

		// Remove required attrs to bypass HTML5 validation
		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		const submitButton = screen.getByText('Vytvořit přiznání');
		await fireEvent.click(submitButton);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalled();
		});

		const { goto } = await import('$app/navigation');
		expect(goto).toHaveBeenCalledWith('/vat/returns/5');
	});

	it('shows error on submit failure', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse('Chyba serveru', 500));

		render(Page);

		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		const submitButton = screen.getByText('Vytvořit přiznání');
		await fireEvent.click(submitButton);

		await waitFor(() => {
			expect(screen.getByRole('alert')).toBeInTheDocument();
		});
	});

	it('has cancel link back to VAT dashboard', () => {
		render(Page);

		const cancelLink = screen.getByText('Zrušit');
		expect(cancelLink.closest('a')).toHaveAttribute('href', '/vat');
	});

	it('renders filing type options', () => {
		render(Page);

		const select = screen.getByLabelText('Typ podání') as HTMLSelectElement;
		const options = Array.from(select.options);
		const values = options.map((o) => o.value);

		expect(values).toContain('regular');
		expect(values).toContain('corrective');
		expect(values).toContain('supplementary');
	});
});
