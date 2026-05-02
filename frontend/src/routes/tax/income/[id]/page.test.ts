import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: {
		params: { id: '1' } as { id: string },
		url: { pathname: '/tax/income/1', searchParams: new URLSearchParams() }
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

const baseReturn = {
	id: 1,
	year: 2025,
	filing_type: 'regular',
	total_revenue: 50000000,
	actual_expenses: 0,
	flat_rate_percent: 60,
	flat_rate_amount: 30000000,
	used_expenses: 30000000,
	tax_base: 20000000,
	total_deductions: 0,
	tax_base_rounded: 20000000,
	tax_at_15: 3000000,
	tax_at_23: 0,
	total_tax: 3000000,
	credit_basic: 3084000,
	credit_spouse: 0,
	credit_disability: 0,
	credit_student: 0,
	total_credits: 3084000,
	tax_after_credits: 0,
	child_benefit: 0,
	tax_after_benefit: 0,
	prepayments: 0,
	tax_due: 0,
	capital_income_gross: 0,
	capital_income_tax: 0,
	capital_income_net: 0,
	other_income_gross: 0,
	other_income_expenses: 0,
	other_income_exempt: 0,
	other_income_net: 0,
	has_xml: false,
	status: 'draft',
	filed_at: null,
	created_at: '2026-04-01T00:00:00Z',
	updated_at: '2026-04-01T00:00:00Z'
};

const taxConstants = {
	year: 2025,
	basic_credit: 3084000,
	spouse_credit: 2484000,
	spouse_income_limit: 6800000,
	student_credit: 0,
	disability_credit_1: 0,
	disability_credit_3: 0,
	disability_ztpp: 0,
	child_benefit_1: 0,
	child_benefit_2: 0,
	child_benefit_3_plus: 0,
	max_child_bonus: 0,
	progressive_threshold: 167605200,
	flat_rate_caps: { '60': 120000000 },
	deduction_cap_mortgage: 0,
	deduction_cap_pension: 0,
	deduction_cap_life_insurance: 0,
	deduction_cap_union: 0,
	time_test_years: 3,
	security_exemption_limit: 10000000
};

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('Income tax return detail - §6 panel', () => {
	it('hides §6 panel when section6_gross_income is 0', async () => {
		mockFetch.mockImplementation((url: string) => {
			if (url.includes('/income-tax-returns/')) {
				return Promise.resolve(jsonResponse(baseReturn));
			}
			if (url.includes('/tax-constants/')) {
				return Promise.resolve(jsonResponse(taxConstants));
			}
			return Promise.resolve(jsonResponse({}));
		});

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Daň z příjmů - 2025')).toBeInTheDocument();
		});

		expect(screen.queryByText('§6 Závislá činnost')).not.toBeInTheDocument();
	});

	it('shows §6 panel when section6_gross_income > 0', async () => {
		const withSection6 = {
			...baseReturn,
			section6_gross_income: 240000,
			section6_foreign_tax: 0,
			section6_tax_base: 240000,
			section6_advance_withheld: 36000,
			section6_withholding_credited: 0,
			section6_monthly_bonus_paid: 0
		};

		mockFetch.mockImplementation((url: string) => {
			if (url.includes('/income-tax-returns/')) {
				return Promise.resolve(jsonResponse(withSection6));
			}
			if (url.includes('/tax-constants/')) {
				return Promise.resolve(jsonResponse(taxConstants));
			}
			return Promise.resolve(jsonResponse({}));
		});

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('§6 Závislá činnost')).toBeInTheDocument();
		});
		expect(screen.getByText('Upravit certifikáty')).toBeInTheDocument();
		// ř.31 row label.
		expect(screen.getByText(/ř\. 31 Úhrn příjmů §6/)).toBeInTheDocument();
		// ř.84 row label.
		expect(screen.getByText(/ř\. 84 Sražené zálohy/)).toBeInTheDocument();
	});

	it('renders warning banner for known progressive_rate_review code', async () => {
		const withWarning = {
			...baseReturn,
			warnings: ['progressive_rate_review']
		};

		mockFetch.mockImplementation((url: string) => {
			if (url.includes('/income-tax-returns/')) {
				return Promise.resolve(jsonResponse(withWarning));
			}
			if (url.includes('/tax-constants/')) {
				return Promise.resolve(jsonResponse(taxConstants));
			}
			return Promise.resolve(jsonResponse({}));
		});

		render(Page);

		await waitFor(() => {
			expect(screen.getByTestId('warnings-section')).toBeInTheDocument();
		});
		const banners = screen.getAllByTestId('warning-banner');
		expect(banners.length).toBe(1);
		expect(banners[0]).toHaveAttribute('role', 'alert');
		expect(screen.getByText('Zkontrolujte progresivní sazbu daně 23 %')).toBeInTheDocument();
	});

	it('hides warnings section when warnings array is empty', async () => {
		const noWarnings = { ...baseReturn, warnings: [] };

		mockFetch.mockImplementation((url: string) => {
			if (url.includes('/income-tax-returns/')) {
				return Promise.resolve(jsonResponse(noWarnings));
			}
			if (url.includes('/tax-constants/')) {
				return Promise.resolve(jsonResponse(taxConstants));
			}
			return Promise.resolve(jsonResponse({}));
		});

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Daň z příjmů - 2025')).toBeInTheDocument();
		});
		expect(screen.queryByTestId('warnings-section')).not.toBeInTheDocument();
	});

	it('renders raw code for unknown warning code without crashing', async () => {
		const withUnknown = {
			...baseReturn,
			warnings: ['some_future_unknown_code']
		};

		mockFetch.mockImplementation((url: string) => {
			if (url.includes('/income-tax-returns/')) {
				return Promise.resolve(jsonResponse(withUnknown));
			}
			if (url.includes('/tax-constants/')) {
				return Promise.resolve(jsonResponse(taxConstants));
			}
			return Promise.resolve(jsonResponse({}));
		});

		render(Page);

		await waitFor(() => {
			expect(screen.getByTestId('warnings-section')).toBeInTheDocument();
		});
		const banner = screen.getByTestId('warning-banner');
		expect(banner).toHaveAttribute('role', 'alert');
		expect(banner).toHaveTextContent('some_future_unknown_code');
	});

	it('shows ř.89 row only when monthly_bonus_paid > 0', async () => {
		const withBonus = {
			...baseReturn,
			section6_gross_income: 240000,
			section6_foreign_tax: 0,
			section6_tax_base: 240000,
			section6_advance_withheld: 36000,
			section6_withholding_credited: 0,
			section6_monthly_bonus_paid: 15300
		};

		mockFetch.mockImplementation((url: string) => {
			if (url.includes('/income-tax-returns/')) {
				return Promise.resolve(jsonResponse(withBonus));
			}
			if (url.includes('/tax-constants/')) {
				return Promise.resolve(jsonResponse(taxConstants));
			}
			return Promise.resolve(jsonResponse({}));
		});

		render(Page);

		await waitFor(() => {
			expect(screen.getByText(/ř\. 89 Vyplacené měsíční bonusy/)).toBeInTheDocument();
		});
	});
});
