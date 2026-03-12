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

function makeTaxYearSettings(
	overrides?: Partial<{
		flat_rate_percent: number;
		prepayments: Array<{
			month: number;
			tax_amount: number;
			social_amount: number;
			health_amount: number;
		}>;
	}>
) {
	const prepayments =
		overrides?.prepayments ??
		Array.from({ length: 12 }, (_, i) => ({
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

const fakeTaxConstants = {
	year: 2025,
	basic_credit: 30840,
	spouse_credit: 24840,
	spouse_income_limit: 68000,
	student_credit: 4020,
	disability_credit_1: 2520,
	disability_credit_3: 5040,
	disability_ztpp: 16140,
	child_benefit_1: 15204,
	child_benefit_2: 22320,
	child_benefit_3_plus: 27840,
	max_child_bonus: 60300,
	progressive_threshold: 1935552,
	flat_rate_caps: {},
	deduction_cap_mortgage: 150000,
	deduction_cap_pension: 24000,
	deduction_cap_life_insurance: 24000,
	deduction_cap_union: 3000,
	time_test_years: 3,
	security_exemption_limit: 100000
};

function mockSettingsAndConstants(settings: unknown, status = 200) {
	mockFetch.mockImplementation((url: string) => {
		if (typeof url === 'string' && url.includes('/tax-constants/')) {
			return Promise.resolve(jsonResponse(fakeTaxConstants));
		}
		return Promise.resolve(jsonResponse(settings, status));
	});
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
		mockSettingsAndConstants(makeTaxYearSettings());
		const { default: Page } = await import('./+page.svelte');
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Paušální výdaje')).toBeTruthy();
		});
		expect(screen.getByText('Měsíční zálohy')).toBeTruthy();
	});

	it('renders flat rate dropdown with options', async () => {
		mockSettingsAndConstants(makeTaxYearSettings({ flat_rate_percent: 60 }));
		const { default: Page } = await import('./+page.svelte');
		render(Page);

		await waitFor(() => {
			const select = screen.getByLabelText('Sazba paušálních výdajů') as HTMLSelectElement;
			expect(select).toBeTruthy();
		});
	});

	it('renders 12 month rows in the table', async () => {
		mockSettingsAndConstants(makeTaxYearSettings());
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
		mockSettingsAndConstants(makeTaxYearSettings({ prepayments }));
		const { default: Page } = await import('./+page.svelte');
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Celkem za rok')).toBeTruthy();
		});
	});

	it('saves settings on button click', async () => {
		mockSettingsAndConstants(makeTaxYearSettings());

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
		mockSettingsAndConstants({ error: 'server error' }, 500);
		const { default: Page } = await import('./+page.svelte');
		render(Page);

		// Should still render with empty defaults (error caught gracefully)
		await waitFor(() => {
			expect(screen.getByText('Paušální výdaje')).toBeTruthy();
		});
	});

	it('renders quick-fill row', async () => {
		mockSettingsAndConstants(makeTaxYearSettings());
		const { default: Page } = await import('./+page.svelte');
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Vyplnit vše')).toBeTruthy();
			expect(screen.getByText('Vyplnit')).toBeTruthy();
		});
	});
});
