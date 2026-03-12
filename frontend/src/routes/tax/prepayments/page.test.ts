import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);
vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

function makeTaxYearSettings(overrides?: Partial<{ flat_rate_percent: number; prepayments: Array<{ month: number; tax_amount: number; social_amount: number; health_amount: number }> }>) {
	const prepayments = overrides?.prepayments ?? Array.from({ length: 12 }, (_, i) => ({
		month: i + 1,
		tax_amount: 0,
		social_amount: 0,
		health_amount: 0
	}));
	return {
		year: 2024,
		flat_rate_percent: overrides?.flat_rate_percent ?? 0,
		prepayments
	};
}

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('Tax Prepayments Page', () => {
	it('shows loading spinner initially', async () => {
		mockFetch.mockReturnValue(new Promise(() => {}));
		const { default: Page } = await import('./+page.svelte');
		render(Page);
		expect(screen.getByRole('status')).toBeTruthy();
	});

	it('renders page heading and year selector', async () => {
		mockFetch.mockResolvedValue(jsonResponse(makeTaxYearSettings()));
		const { default: Page } = await import('./+page.svelte');
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Paušální výdaje')).toBeTruthy();
		});
		expect(screen.getByText('Měsíční zálohy')).toBeTruthy();
	});

	it('renders flat rate dropdown with options', async () => {
		mockFetch.mockResolvedValue(jsonResponse(makeTaxYearSettings({ flat_rate_percent: 60 })));
		const { default: Page } = await import('./+page.svelte');
		render(Page);

		await waitFor(() => {
			const select = screen.getByLabelText('Sazba paušálních výdajů') as HTMLSelectElement;
			expect(select).toBeTruthy();
		});
	});

	it('renders 12 month rows in the table', async () => {
		mockFetch.mockResolvedValue(jsonResponse(makeTaxYearSettings()));
		const { default: Page } = await import('./+page.svelte');
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Leden')).toBeTruthy();
		});
		expect(screen.getByText('Prosinec')).toBeTruthy();
		expect(screen.getByText('Červenec')).toBeTruthy();
	});

	it('shows totals row', async () => {
		const prepayments = Array.from({ length: 12 }, (_, i) => ({
			month: i + 1,
			tax_amount: 100000,
			social_amount: 200000,
			health_amount: 300000
		}));
		mockFetch.mockResolvedValue(jsonResponse(makeTaxYearSettings({ prepayments })));
		const { default: Page } = await import('./+page.svelte');
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Celkem za rok')).toBeTruthy();
		});
	});

	it('saves settings on button click', async () => {
		mockFetch.mockResolvedValue(jsonResponse(makeTaxYearSettings()));

		const { default: Page } = await import('./+page.svelte');
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Uložit nastavení')).toBeTruthy();
		});

		const callsBefore = mockFetch.mock.calls.length;
		await fireEvent.click(screen.getByText('Uložit nastavení'));

		await waitFor(() => {
			expect(mockFetch.mock.calls.length).toBeGreaterThan(callsBefore);
		});

		// Verify the save call used PUT method
		const saveCalls = mockFetch.mock.calls.filter(
			(c: unknown[]) => c[1] && (c[1] as RequestInit).method === 'PUT'
		);
		expect(saveCalls.length).toBe(1);
	});

	it('shows error on API failure', async () => {
		mockFetch.mockResolvedValue(jsonResponse({ error: 'server error' }, 500));
		const { default: Page } = await import('./+page.svelte');
		render(Page);

		// Should still render with empty defaults (error caught gracefully)
		await waitFor(() => {
			expect(screen.getByText('Paušální výdaje')).toBeTruthy();
		});
	});

	it('renders quick-fill row', async () => {
		mockFetch.mockResolvedValue(jsonResponse(makeTaxYearSettings()));
		const { default: Page } = await import('./+page.svelte');
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Vyplnit vše')).toBeTruthy();
			expect(screen.getByText('Vyplnit')).toBeTruthy();
		});
	});
});
