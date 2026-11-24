import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup, within } from '@testing-library/svelte';
import { toasts, clearAllToasts } from '$lib/data/toast-state.svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);
vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: {
		params: { id: '1' },
		url: { pathname: '/tax/social/1', searchParams: new URLSearchParams() }
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
		secondary_months: [],
		secondary_reason: '',
		employment_assessment_base: null,
		reported_employment_base: null,
		main_tax_base_share: 0,
		secondary_tax_base_share: 0,
		secondary_assessment_base: 0,
		main_partial_base: 0,
		secondary_partial_base: 0,
		secondary_decisive_amount: 0,
		sum_base: 9777000,
		insured_base: 9777000,
		total_revenue: 10000000,
		total_expenses: 0,
		tax_base: 10000000,
		assessment_base: 5500000,
		min_assessment_base: 9777000,
		final_assessment_base: 9777000,
		insurance_rate: 292,
		total_insurance: 2854900,
		prepayments: 0,
		difference: 2854900,
		new_monthly_prepay: 572000,
		new_monthly_base: 1958700,
		filing_method: 'standard',
		advance_year: 2026,
		advance_start_month: 6,
		advance_periods: [
			{ from_month: 6, monthly_base: 1958700, monthly_prepay: 572000 },
			{ from_month: 7, monthly_base: 1713900, monthly_prepay: 500500 }
		],
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
	generateStatus?: number;
	generateBody?: unknown;
}

function mockApi(options: MockOptions = {}) {
	const overview = options.overview ?? makeOverview();

	mockFetch.mockImplementation((url: string) => {
		if (url.includes('/tax-constants')) {
			return Promise.resolve(jsonResponse({ error: 'not found' }, 404));
		}
		if (url.includes('/generate-xml') && options.generateStatus) {
			return Promise.resolve(
				jsonResponse(options.generateBody ?? { error: 'error' }, options.generateStatus)
			);
		}
		if (url.includes('/secondary-activity') || url.includes('/employment-base')) {
			if (options.updateStatus && options.updateStatus !== 200) {
				return Promise.resolve(
					jsonResponse(options.updateBody ?? { error: 'error' }, options.updateStatus)
				);
			}
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

describe('social insurance overview detail page', () => {
	it('shows the stored months of activity', async () => {
		mockApi();

		await renderPage();

		expect(
			screen.getByText('červenec, srpen, září, říjen, listopad, prosinec (6 měs.)')
		).toBeInTheDocument();
	});

	it('renders the advance schedule with its start month and warnings', async () => {
		mockApi({ overview: makeOverview({ warnings: ['filing_method_missing'] }) });

		await renderPage();

		expect(screen.getByText(/Nová záloha od:/)).toHaveTextContent('červen 2026');
		expect(screen.getByText('od červenec 2026:')).toBeInTheDocument();
		expect(screen.getByRole('link', { name: 'Nastavit způsob podání' })).toBeInTheDocument();
	});

	it('saves the months through the activity-months endpoint', async () => {
		mockApi({ updateOverview: makeOverview({ activity_months: [1, 7, 8, 9, 10, 11, 12] }) });

		await renderPage();
		await openEditor();
		await fireEvent.click(screen.getByLabelText('leden'));
		await fireEvent.click(screen.getByRole('button', { name: 'Uložit' }));

		await waitFor(() => {
			expect(
				mockFetch.mock.calls.some(([url]) => url === '/api/v1/social-insurance/1/activity-months')
			).toBe(true);
		});

		const call = mockFetch.mock.calls.find(
			([url]) => url === '/api/v1/social-insurance/1/activity-months'
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

	it('shows the server message when the XML cannot be generated', async () => {
		mockApi({
			generateStatus: 400,
			generateBody: { error: 'přehled je potřeba nejdřív přepočítat' }
		});

		await renderPage();
		await fireEvent.click(screen.getByRole('button', { name: 'Generovat XML' }));

		await waitFor(() => {
			expect(toasts.some((t) => t.message === 'přehled je potřeba nejdřív přepočítat')).toBe(true);
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

	it('labels the computed base as 55 percent', async () => {
		mockApi();

		await renderPage();

		expect(screen.getByText('Vypočtený vyměřovací základ (55 %)')).toBeInTheDocument();
		expect(screen.queryByText('Vyměřovací základ (50%)')).not.toBeInTheDocument();
	});

	it('shows the secondary activity with its reason and decisive amount', async () => {
		mockApi({
			overview: makeOverview({
				secondary_months: [7, 8, 9, 10, 11, 12],
				secondary_reason: 'student',
				secondary_assessment_base: 6600000,
				secondary_decisive_amount: 5586400
			})
		});

		await renderPage();

		const card = within(screen.getByRole('group', { name: 'Vedlejší činnost' }));
		expect(
			card.getByText(
				/červenec, srpen, září, říjen, listopad, prosinec \(6 měs\.\) – Nezaopatřené dítě/
			)
		).toBeInTheDocument();
		expect(screen.getByText('Rozhodná částka pro vedlejší činnost')).toBeInTheDocument();
		expect(screen.getByText('Vypočtený vyměřovací základ – vedlejší činnost')).toBeInTheDocument();
		expect(screen.queryByText(/rozhodné částky nedosáhl/)).not.toBeInTheDocument();
	});

	it('shows row 27 as the minimum base of a mixed year', async () => {
		mockApi({
			overview: makeOverview({
				activity_months: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12],
				secondary_months: [7, 8, 9, 10, 11, 12],
				secondary_reason: 'employment',
				secondary_assessment_base: 6600000,
				min_assessment_base: 12850200,
				final_assessment_base: 16377000
			})
		});

		await renderPage();

		const minimum = screen.getByText('Minimální vyměřovací základ');
		expect(minimum.nextElementSibling).toHaveTextContent(/163\s?770/);
	});

	it('says when the secondary income did not reach the decisive amount', async () => {
		mockApi({
			overview: makeOverview({
				secondary_months: [7, 8, 9, 10, 11, 12],
				secondary_reason: 'employment',
				secondary_assessment_base: 0,
				secondary_decisive_amount: 5586400
			})
		});

		await renderPage();

		expect(
			screen.getByText(
				'Příjem z vedlejší činnosti rozhodné částky nedosáhl, pojistné z ní se neplatí'
			)
		).toBeInTheDocument();
	});

	it('shows the warning that social advances are not required', async () => {
		mockApi({ overview: makeOverview({ warnings: ['social_advance_not_required'] }) });

		await renderPage();

		expect(screen.getByRole('alert')).toHaveTextContent(
			'Zálohy na důchodové pojištění v roce 2026'
		);
	});

	it('saves the secondary activity through its endpoint', async () => {
		mockApi({
			updateOverview: makeOverview({ secondary_months: [7], secondary_reason: 'employment' })
		});

		await renderPage();
		const card = within(screen.getByRole('group', { name: 'Vedlejší činnost' }));
		await fireEvent.click(card.getByRole('button', { name: 'Upravit' }));
		await fireEvent.click(card.getByLabelText('červenec'));
		await fireEvent.change(card.getByLabelText('Důvod'), { target: { value: 'employment' } });
		await fireEvent.click(card.getByRole('button', { name: 'Uložit vedlejší činnost' }));

		await waitFor(() => {
			expect(
				mockFetch.mock.calls.some(
					([url]) => url === '/api/v1/social-insurance/1/secondary-activity'
				)
			).toBe(true);
		});
		const call = mockFetch.mock.calls.find(
			([url]) => url === '/api/v1/social-insurance/1/secondary-activity'
		);
		expect(call?.[1].method).toBe('PUT');
		expect(JSON.parse(call?.[1].body as string)).toEqual({ months: [7], reason: 'employment' });
	});

	it('only offers the months of activity as secondary months', async () => {
		mockApi();

		await renderPage();
		const card = within(screen.getByRole('group', { name: 'Vedlejší činnost' }));
		await fireEvent.click(card.getByRole('button', { name: 'Upravit' }));

		expect(card.getByLabelText('leden')).toBeDisabled();
		expect(card.getByLabelText('červenec')).toBeEnabled();
	});

	it('shows the server message when the secondary activity is rejected', async () => {
		mockApi({
			updateStatus: 400,
			updateBody: { error: 'vyberte důvod vedlejší činnosti' }
		});

		await renderPage();
		const card = within(screen.getByRole('group', { name: 'Vedlejší činnost' }));
		await fireEvent.click(card.getByRole('button', { name: 'Upravit' }));
		await fireEvent.click(card.getByRole('button', { name: 'Uložit vedlejší činnost' }));

		await waitFor(() => {
			expect(toasts.some((t) => t.message === 'vyberte důvod vedlejší činnosti')).toBe(true);
		});
	});

	it('saves the employment base in halere', async () => {
		mockApi({ updateOverview: makeOverview({ employment_assessment_base: 50000000 }) });

		await renderPage();
		const card = within(screen.getByRole('group', { name: 'Zaměstnání' }));
		await fireEvent.input(card.getByLabelText(/Vyměřovací základ ze zaměstnání za rok 2025/), {
			target: { value: '500000' }
		});
		await fireEvent.click(
			card.getByRole('button', { name: 'Uložit vyměřovací základ ze zaměstnání' })
		);

		await waitFor(() => {
			expect(
				mockFetch.mock.calls.some(([url]) => url === '/api/v1/social-insurance/1/employment-base')
			).toBe(true);
		});
		const call = mockFetch.mock.calls.find(
			([url]) => url === '/api/v1/social-insurance/1/employment-base'
		);
		expect(call?.[1].method).toBe('PUT');
		expect(JSON.parse(call?.[1].body as string)).toEqual({ employment_assessment_base: 50000000 });
	});

	it('removes the employment base by sending null', async () => {
		mockApi({
			overview: makeOverview({ employment_assessment_base: 50000000 }),
			updateOverview: makeOverview()
		});

		await renderPage();
		const card = within(screen.getByRole('group', { name: 'Zaměstnání' }));
		await fireEvent.click(
			card.getByRole('button', { name: 'Odebrat vyměřovací základ ze zaměstnání' })
		);

		await waitFor(() => {
			expect(
				mockFetch.mock.calls.some(([url]) => url === '/api/v1/social-insurance/1/employment-base')
			).toBe(true);
		});
		const call = mockFetch.mock.calls.find(
			([url]) => url === '/api/v1/social-insurance/1/employment-base'
		);
		expect(JSON.parse(call?.[1].body as string)).toEqual({ employment_assessment_base: null });
	});

	it('does not send a fractional employment base', async () => {
		mockApi();

		await renderPage();
		const card = within(screen.getByRole('group', { name: 'Zaměstnání' }));
		await fireEvent.input(card.getByLabelText(/Vyměřovací základ ze zaměstnání za rok 2025/), {
			target: { value: '1234.5' }
		});
		await fireEvent.click(
			card.getByRole('button', { name: 'Uložit vyměřovací základ ze zaměstnání' })
		);

		await waitFor(() => {
			expect(toasts.some((t) => t.message.includes('celé nezáporné číslo'))).toBe(true);
		});
		expect(
			mockFetch.mock.calls.some(([url]) => url === '/api/v1/social-insurance/1/employment-base')
		).toBe(false);
	});

	it('shows the employment base and the insured base when an employment base is entered', async () => {
		mockApi({
			overview: makeOverview({
				employment_assessment_base: 50000000,
				reported_employment_base: 50000000,
				insured_base: 173473600
			})
		});

		await renderPage();

		expect(screen.getByText('Vyměřovací základ pro pojistné (ř. 31)')).toBeInTheDocument();
		const input = screen.getByLabelText(/Vyměřovací základ ze zaměstnání za rok 2025/);
		expect(input).toHaveValue(500000);
	});

	it('hides the secondary and employment editors for a filed overview', async () => {
		mockApi({ overview: makeOverview({ status: 'filed', filed_at: '2026-02-01T00:00:00Z' }) });

		await renderPage();

		const secondary = within(screen.getByRole('group', { name: 'Vedlejší činnost' }));
		expect(secondary.queryByRole('button', { name: 'Upravit' })).not.toBeInTheDocument();
		expect(
			screen.queryByLabelText(/Vyměřovací základ ze zaměstnání za rok/)
		).not.toBeInTheDocument();
	});

	it('hides the edit control for a filed overview', async () => {
		mockApi({ overview: makeOverview({ status: 'filed', filed_at: '2026-02-01T00:00:00Z' }) });

		await renderPage();

		expect(screen.queryByRole('button', { name: 'Upravit měsíce' })).not.toBeInTheDocument();
	});
});
