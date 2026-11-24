import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);
vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: { params: {}, url: { pathname: '/tax/health/new', searchParams: new URLSearchParams() } }
}));

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const createdOverview = {
	id: 5,
	year: 2024,
	filing_type: 'regular',
	activity_months: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12],
	total_revenue: 0,
	total_expenses: 0,
	tax_base: 0,
	assessment_base: 0,
	min_assessment_base: 0,
	final_assessment_base: 0,
	insurance_rate: 135,
	total_insurance: 0,
	prepayments: 0,
	difference: 0,
	new_monthly_prepay: 0,
	has_xml: false,
	status: 'draft',
	filed_at: null,
	created_at: '2026-01-01T00:00:00Z',
	updated_at: '2026-01-01T00:00:00Z'
};

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('new health insurance overview page', () => {
	it('creates the overview with all twelve months', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(createdOverview));

		const { default: Page } = await import('./+page.svelte');
		render(Page);

		await fireEvent.click(screen.getByRole('button', { name: 'Vytvořit přehled' }));

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledTimes(1);
		});

		const body = JSON.parse(mockFetch.mock.calls[0][1].body as string);
		expect(body.activity_months).toEqual([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);

		const { goto } = await import('$app/navigation');
		expect(goto).toHaveBeenCalledWith('/tax/health/5');
	});

	it('does not submit without any month of activity', async () => {
		const { default: Page } = await import('./+page.svelte');
		render(Page);

		for (const month of [
			'leden',
			'únor',
			'březen',
			'duben',
			'květen',
			'červen',
			'červenec',
			'srpen',
			'září',
			'říjen',
			'listopad',
			'prosinec'
		]) {
			await fireEvent.click(screen.getByLabelText(month));
		}

		expect(screen.getByRole('alert')).toHaveTextContent('Vyberte alespoň jeden měsíc činnosti.');

		await fireEvent.click(screen.getByRole('button', { name: 'Vytvořit přehled' }));

		expect(mockFetch).not.toHaveBeenCalled();
	});
});
