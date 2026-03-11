import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import Page from './+page.svelte';

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

beforeEach(async () => {
	vi.useFakeTimers();
	vi.setSystemTime(new Date('2026-03-10T12:00:00Z'));
	mockFetch.mockReset();
	const { goto } = await import('$app/navigation');
	(goto as ReturnType<typeof vi.fn>).mockReset();
	mockFetch.mockResolvedValue(jsonResponse({}));
});

afterEach(() => {
	cleanup();
	vi.useRealTimers();
});

describe('Control Statement Create', () => {
	it('renders form with heading', () => {
		render(Page);
		expect(screen.getByText('Nové kontrolní hlášení')).toBeInTheDocument();
	});

	it('renders year input with current year default', () => {
		render(Page);
		const yearInput = document.querySelector('#year') as HTMLInputElement;
		expect(yearInput).toBeInTheDocument();
		expect(yearInput.value).toBe('2026');
	});

	it('renders month select with current month default', () => {
		render(Page);
		const monthSelect = document.querySelector('#month') as HTMLSelectElement;
		expect(monthSelect).toBeInTheDocument();
		expect(monthSelect.value).toBe('3');
	});

	it('renders all 12 months', () => {
		render(Page);
		expect(screen.getByText('Leden')).toBeInTheDocument();
		expect(screen.getByText('Únor')).toBeInTheDocument();
		expect(screen.getByText('Březen')).toBeInTheDocument();
		expect(screen.getByText('Duben')).toBeInTheDocument();
		expect(screen.getByText('Květen')).toBeInTheDocument();
		expect(screen.getByText('Červen')).toBeInTheDocument();
		expect(screen.getByText('Červenec')).toBeInTheDocument();
		expect(screen.getByText('Srpen')).toBeInTheDocument();
		expect(screen.getByText('Září')).toBeInTheDocument();
		expect(screen.getByText('Říjen')).toBeInTheDocument();
		expect(screen.getByText('Listopad')).toBeInTheDocument();
		expect(screen.getByText('Prosinec')).toBeInTheDocument();
	});

	it('renders filing type select with options', () => {
		render(Page);
		expect(screen.getByText('Řádné')).toBeInTheDocument();
		expect(screen.getByText('Následné')).toBeInTheDocument();
		expect(screen.getByText('Opravné')).toBeInTheDocument();
	});

	it('renders submit and cancel buttons', () => {
		render(Page);
		expect(screen.getByText('Vytvořit hlášení')).toBeInTheDocument();
		expect(screen.getByText('Zrušit')).toBeInTheDocument();
	});

	it('renders back link to VAT page', () => {
		render(Page);
		const backLink = screen.getByText(/Zpět na DPH/);
		expect(backLink).toBeInTheDocument();
		expect(backLink.closest('a')?.getAttribute('href')).toBe('/vat');
	});

	it('submits with correct data and navigates to detail', async () => {
		const createdStatement = {
			id: 42,
			period: { year: 2026, month: 3, quarter: 1 },
			filing_type: 'regular',
			lines: null,
			has_xml: false,
			status: 'draft',
			filed_at: null,
			created_at: '2026-03-10T00:00:00Z',
			updated_at: '2026-03-10T00:00:00Z'
		};
		mockFetch.mockResolvedValueOnce(jsonResponse(createdStatement));

		render(Page);

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			const createCall = mockFetch.mock.calls.find(
				(call: any[]) =>
					typeof call[0] === 'string' &&
					call[0].includes('/api/v1/vat-control-statements') &&
					call[1]?.method === 'POST'
			);
			expect(createCall).toBeDefined();
			if (createCall) {
				const body = JSON.parse(createCall[1].body);
				expect(body.year).toBe(2026);
				expect(body.month).toBe(3);
				expect(body.filing_type).toBe('regular');
			}
		});

		const { goto } = await import('$app/navigation');
		expect(goto).toHaveBeenCalledWith('/vat/control/42');
	});

	it('shows error on API failure', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ error: 'Duplicate period' }, 409));

		render(Page);

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			expect(screen.getByText('Duplicate period')).toBeInTheDocument();
		});
	});

	it('shows validation error for invalid year', async () => {
		render(Page);

		const yearInput = document.querySelector('#year') as HTMLInputElement;
		yearInput.value = '1999';
		await fireEvent.input(yearInput);

		// Remove required attrs to bypass native validation
		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			expect(screen.getByText('Zadejte platný rok')).toBeInTheDocument();
		});
	});

	it('disables submit button while saving', async () => {
		mockFetch.mockReturnValueOnce(new Promise(() => {})); // never resolves

		render(Page);

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			expect(screen.getByText('Vytvářím...')).toBeInTheDocument();
		});
	});
});
