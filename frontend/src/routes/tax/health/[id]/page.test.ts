import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import { toasts, clearAllToasts } from '$lib/data/toast-state.svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);
vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: {
		params: { id: '1' },
		url: { pathname: '/tax/health/1', searchParams: new URLSearchParams() }
	}
}));

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

function makeOverview(overrides: Record<string, unknown> = {}) {
	return {
		id: 1,
		year: 2025,
		filing_type: 'regular',
		activity_months: [7, 8, 9, 10, 11, 12],
		total_revenue: 10000000,
		total_expenses: 0,
		tax_base: 10000000,
		assessment_base: 5500000,
		min_assessment_base: 13190100,
		final_assessment_base: 13190100,
		insurance_rate: 135,
		total_insurance: 1780700,
		prepayments: 0,
		difference: 1780700,
		new_monthly_prepay: 330600,
		no_minimum_months: [],
		no_minimum_reason: '',
		filing_method: 'standard',
		advance_year: 2026,
		advance_start_month: 5,
		warnings: [],
		has_xml: false,
		status: 'draft',
		filed_at: null,
		created_at: '2026-01-01T00:00:00Z',
		updated_at: '2026-01-01T00:00:00Z',
		...overrides
	};
}

interface MockOptions {
	overview?: Record<string, unknown>;
	updateOverview?: Record<string, unknown>;
	updateStatus?: number;
	updateBody?: unknown;
}

function mockApi(options: MockOptions = {}) {
	const overview = options.overview ?? makeOverview();

	mockFetch.mockImplementation((url: string) => {
		if (url.includes('/tax-constants')) {
			return Promise.resolve(jsonResponse({ error: 'not found' }, 404));
		}
		if (url.includes('/no-minimum')) {
			return Promise.resolve(jsonResponse(options.updateOverview ?? overview));
		}
		if (url.includes('/activity-months')) {
			if (options.updateStatus && options.updateStatus !== 200) {
				return Promise.resolve(
					jsonResponse(options.updateBody ?? { error: 'error' }, options.updateStatus)
				);
			}
			return Promise.resolve(jsonResponse(options.updateOverview ?? overview));
		}
		return Promise.resolve(jsonResponse(overview));
	});
}

beforeEach(() => {
	mockFetch.mockReset();
	clearAllToasts();
});

afterEach(() => {
	cleanup();
});

async function renderPage() {
	const { default: Page } = await import('./+page.svelte');
	render(Page);
	await screen.findByText('Měsíce činnosti');
}

async function openEditor() {
	await fireEvent.click(screen.getByRole('button', { name: 'Upravit měsíce' }));
}

describe('health insurance overview detail page', () => {
	it('shows the stored months of activity', async () => {
		mockApi();

		await renderPage();

		expect(
			screen.getByText('červenec, srpen, září, říjen, listopad, prosinec (6 měs.)')
		).toBeInTheDocument();
	});

	it('shows the advance start month', async () => {
		mockApi();

		await renderPage();

		expect(screen.getByText(/Nová záloha od:/)).toHaveTextContent('květen 2026');
	});

	it('shows the months without minimum with their reason', async () => {
		mockApi({
			overview: makeOverview({ no_minimum_months: [7, 8], no_minimum_reason: 'ztp' })
		});

		await renderPage();

		expect(screen.getByText('Měsíce bez minima')).toBeInTheDocument();
		expect(
			screen.getByText(/červenec, srpen \(2 měs\.\) – Držitel průkazu ZTP nebo ZTP\/P/)
		).toBeInTheDocument();
	});

	it('says the minimum applies when no months are exempt', async () => {
		mockApi();

		await renderPage();

		expect(screen.getByText('Minimum platí ve všech měsících činnosti')).toBeInTheDocument();
	});

	it('shows the warning that advances are not required', async () => {
		mockApi({ overview: makeOverview({ warnings: ['health_advance_not_required'] }) });

		await renderPage();

		expect(screen.getByRole('alert')).toHaveTextContent('v roce 2026 nejsou povinné');
	});

	it('saves the months without minimum through the no-minimum endpoint', async () => {
		mockApi({
			updateOverview: makeOverview({ no_minimum_months: [7], no_minimum_reason: 'employee' })
		});

		await renderPage();
		await fireEvent.click(screen.getByRole('button', { name: 'Upravit' }));
		await fireEvent.click(screen.getByLabelText('červenec'));
		await fireEvent.change(screen.getByLabelText('Důvod'), { target: { value: 'employee' } });
		await fireEvent.click(screen.getByRole('button', { name: 'Uložit' }));

		await waitFor(() => {
			expect(
				mockFetch.mock.calls.some(([url]) => url === '/api/v1/health-insurance/1/no-minimum')
			).toBe(true);
		});
		const call = mockFetch.mock.calls.find(
			([url]) => url === '/api/v1/health-insurance/1/no-minimum'
		);
		expect(call?.[1].method).toBe('PUT');
		expect(JSON.parse(call?.[1].body as string)).toEqual({ months: [7], reason: 'employee' });
	});

	it('saves the months through the activity-months endpoint', async () => {
		mockApi({ updateOverview: makeOverview({ activity_months: [1, 7, 8, 9, 10, 11, 12] }) });

		await renderPage();
		await openEditor();
		await fireEvent.click(screen.getByLabelText('leden'));
		await fireEvent.click(screen.getByRole('button', { name: 'Uložit' }));

		await waitFor(() => {
			expect(
				mockFetch.mock.calls.some(([url]) => url === '/api/v1/health-insurance/1/activity-months')
			).toBe(true);
		});

		const call = mockFetch.mock.calls.find(
			([url]) => url === '/api/v1/health-insurance/1/activity-months'
		);
		expect(call?.[1].method).toBe('PUT');
		expect(JSON.parse(call?.[1].body as string)).toEqual({
			activity_months: [1, 7, 8, 9, 10, 11, 12]
		});
	});

	it('explains a filed overview in Czech when the months cannot change', async () => {
		mockApi({ updateStatus: 409, updateBody: { error: 'filing already filed, cannot modify' } });

		await renderPage();
		await openEditor();
		await fireEvent.click(screen.getByRole('button', { name: 'Uložit' }));

		await waitFor(() => {
			expect(toasts.some((t) => t.message === 'Přehled je už podaný, měsíce nelze změnit.')).toBe(
				true
			);
		});
	});

	it('shows the server message for a 400 from the months edit', async () => {
		mockApi({
			updateStatus: 400,
			updateBody: { error: 'Pro rok 2019 zatím nejsou daňová pravidla.' }
		});

		await renderPage();
		await openEditor();
		await fireEvent.click(screen.getByRole('button', { name: 'Uložit' }));

		await waitFor(() => {
			expect(toasts.some((t) => t.message === 'Pro rok 2019 zatím nejsou daňová pravidla.')).toBe(
				true
			);
		});
	});

	it('falls back to the Czech hint when the server message is missing', async () => {
		mockApi({ updateStatus: 400, updateBody: {} });

		await renderPage();
		await openEditor();
		await fireEvent.click(screen.getByRole('button', { name: 'Uložit' }));

		await waitFor(() => {
			expect(toasts.some((t) => t.message === 'Vyberte alespoň jeden měsíc činnosti.')).toBe(true);
		});
	});

	it('shows the generic Czech text for an unexpected error', async () => {
		mockApi({ updateStatus: 500, updateBody: { error: 'internal' } });

		await renderPage();
		await openEditor();
		await fireEvent.click(screen.getByRole('button', { name: 'Uložit' }));

		await waitFor(() => {
			expect(toasts.some((t) => t.message === 'Nepodařilo se uložit měsíce činnosti')).toBe(true);
		});
	});

	it('hides the edit control for a filed overview', async () => {
		mockApi({ overview: makeOverview({ status: 'filed', filed_at: '2026-02-01T00:00:00Z' }) });

		await renderPage();

		expect(screen.queryByRole('button', { name: 'Upravit měsíce' })).not.toBeInTheDocument();
	});
});
