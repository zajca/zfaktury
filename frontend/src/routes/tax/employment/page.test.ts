import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: {
		params: {} as Record<string, string>,
		url: { pathname: '/tax/employment', searchParams: new URLSearchParams() }
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

function emptyResponse(status = 204) {
	return new Response(null, { status, statusText: status === 204 ? 'No Content' : 'OK' });
}

const advanceCert = {
	id: 10,
	year: 2025,
	document_id: 1,
	certificate_type: 'advance' as const,
	employer_name: 'Acme s.r.o.',
	employer_ico: '12345678',
	employer_address: 'Hlavni 1, Praha',
	contract_type: 'dpc' as const,
	period_from: '2025-01-01',
	period_to: '2025-12-31',
	gross_income_czk: 240000,
	income_without_advance_czk: 0,
	foreign_tax_paid_czk: 0,
	advance_tax_withheld_czk: 36000,
	annual_settlement_refund_czk: 0,
	monthly_bonus_paid_czk: 0,
	withheld_final_tax_czk: 0,
	include_withholding_in_dap: false,
	notes: '',
	confidence: 0.92,
	status: 'draft' as const,
	created_at: '2026-04-01T00:00:00Z',
	updated_at: '2026-04-01T00:00:00Z'
};

const uploadedDocument = {
	id: 1,
	year: 2025,
	kind: 'advance' as const,
	filename: 'potvrzeni.pdf',
	content_type: 'application/pdf',
	size: 12345,
	extraction_status: 'pending' as const,
	created_at: '2026-04-01T00:00:00Z'
};

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('Employment income page', () => {
	it('renders empty state when no certificates', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse([]));

		render(Page);

		await waitFor(() => {
			expect(screen.getByTestId('empty-state')).toBeInTheDocument();
		});
		expect(screen.getByText(/Zatím žádná Potvrzení/)).toBeInTheDocument();
	});

	it('renders OCR config advisory below upload tiles', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse([]));

		render(Page);

		await waitFor(() => {
			expect(screen.getByTestId('ocr-config-advisory')).toBeInTheDocument();
		});
		const advisory = screen.getByTestId('ocr-config-advisory');
		expect(advisory).toHaveAttribute('role', 'status');
		expect(advisory).toHaveTextContent(/OCR vyžaduje aktivní AI poskytovatel/);
		expect(advisory).toHaveTextContent(/Zadat ručně/);
	});

	it('renders clickable "Zadat ručně" buttons', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse([]));

		render(Page);

		await waitFor(() => {
			expect(screen.getByTestId('manual-buttons-wrapper')).toBeInTheDocument();
		});
		const advanceBtn = screen.getByRole('button', { name: 'Zálohové' });
		const withholdingBtn = screen.getByRole('button', { name: 'Srážkové' });
		expect(advanceBtn).toBeInTheDocument();
		expect(withholdingBtn).toBeInTheDocument();
		expect(advanceBtn).not.toBeDisabled();
		expect(withholdingBtn).not.toBeDisabled();

		// Click should open the manual editor (advance section appears).
		await fireEvent.click(advanceBtn);
		await waitFor(() => {
			expect(screen.getByTestId('advance-section')).toBeInTheDocument();
		});
	});

	it('lists certificates fetched on mount', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse([advanceCert]));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Acme s.r.o.')).toBeInTheDocument();
		});
		expect(screen.getByText('DPČ')).toBeInTheDocument();
		expect(screen.getByText('Zálohové (vzor 33)')).toBeInTheDocument();
	});

	it('opens manual editor for advance certificate', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse([]));

		render(Page);

		await waitFor(() => {
			expect(screen.getByTestId('manual-buttons-wrapper')).toBeInTheDocument();
		});

		const manualBtn = screen.getByRole('button', { name: 'Zálohové' });
		await fireEvent.click(manualBtn);

		await waitFor(() => {
			expect(screen.getByTestId('advance-section')).toBeInTheDocument();
		});
		expect(screen.getByTestId('employer-name-input')).toBeInTheDocument();
	});

	it('switches form fields when toggling certificate type', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse([]));

		render(Page);

		await waitFor(() => {
			expect(screen.getByRole('button', { name: 'Zálohové' })).toBeInTheDocument();
		});
		await fireEvent.click(screen.getByRole('button', { name: 'Zálohové' }));

		await waitFor(() => {
			expect(screen.getByTestId('advance-section')).toBeInTheDocument();
		});

		const typeSelect = screen.getByTestId('certificate-type-select') as HTMLSelectElement;
		await fireEvent.change(typeSelect, { target: { value: 'withholding' } });

		await waitFor(() => {
			expect(screen.getByTestId('withholding-section')).toBeInTheDocument();
		});
		expect(screen.queryByTestId('advance-section')).not.toBeInTheDocument();
		expect(screen.getByTestId('include-withholding-checkbox')).toBeInTheDocument();
	});

	it('blocks save with invalid IČO and shows alert', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse([]));

		render(Page);

		await waitFor(() => {
			expect(screen.getByRole('button', { name: 'Zálohové' })).toBeInTheDocument();
		});
		await fireEvent.click(screen.getByRole('button', { name: 'Zálohové' }));

		// Fill required fields with invalid IČO.
		const nameInput = (await screen.findByTestId('employer-name-input')) as HTMLInputElement;
		const icoInput = screen.getByTestId('employer-ico-input') as HTMLInputElement;

		await fireEvent.input(nameInput, { target: { value: 'Acme s.r.o.' } });
		await fireEvent.input(icoInput, { target: { value: '123' } });

		// Bypass HTML5 native validation so the custom message can fire on click.
		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		const saveDraft = screen.getByRole('button', { name: 'Uložit jako koncept' });
		await fireEvent.click(saveDraft);

		await waitFor(() => {
			expect(screen.getByText(/IČO musí být 8 číslic/)).toBeInTheDocument();
		});

		// Should NOT have called the create endpoint.
		const createCalls = mockFetch.mock.calls.filter(
			([url, init]) =>
				typeof url === 'string' &&
				url.endsWith('/tax/employment/certificates') &&
				(init as RequestInit | undefined)?.method === 'POST'
		);
		expect(createCalls.length).toBe(0);
	});

	it('blocks save when period_from > period_to', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse([]));

		render(Page);
		await waitFor(() => {
			expect(screen.getByRole('button', { name: 'Zálohové' })).toBeInTheDocument();
		});
		await fireEvent.click(screen.getByRole('button', { name: 'Zálohové' }));

		const nameInput = (await screen.findByTestId('employer-name-input')) as HTMLInputElement;
		const icoInput = screen.getByTestId('employer-ico-input') as HTMLInputElement;
		const fromInput = screen.getByTestId('period-from-input') as HTMLInputElement;
		const toInput = screen.getByTestId('period-to-input') as HTMLInputElement;

		await fireEvent.input(nameInput, { target: { value: 'Acme s.r.o.' } });
		await fireEvent.input(icoInput, { target: { value: '12345678' } });
		await fireEvent.input(fromInput, { target: { value: '2025-12-31' } });
		await fireEvent.input(toInput, { target: { value: '2025-01-01' } });

		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		await fireEvent.click(screen.getByRole('button', { name: 'Uložit jako koncept' }));

		await waitFor(() => {
			expect(screen.getByText(/Datum "od" nesmí být pozdější/)).toBeInTheDocument();
		});
	});

	it('shows OCR confidence badge in editor when draft has confidence', async () => {
		// Render the editor directly via a draft that already includes confidence.
		// Full upload flow exercises the file picker which depends on browser-only
		// File constructor semantics; here we test that the badge renders correctly
		// once the editor receives a draft with confidence set.
		mockFetch.mockResolvedValueOnce(jsonResponse([advanceCert]));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Acme s.r.o.')).toBeInTheDocument();
		});

		// Click Upravit on the row to open the editor with the cert (which has confidence: 0.92).
		const editButtons = screen.getAllByRole('button', { name: 'Upravit' });
		await fireEvent.click(editButtons[0]);

		await waitFor(() => {
			expect(screen.getByTestId('ocr-confidence-badge')).toBeInTheDocument();
		});
		expect(screen.getByTestId('ocr-confidence-badge')).toHaveTextContent(/92/);
	});
});
