import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

vi.mock('$app/navigation', () => ({ goto: vi.fn(), beforeNavigate: vi.fn() }));

import Page from './+page.svelte';

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const summary = {
	year: 2025,
	spouse: null,
	children: [],
	personal: {
		year: 2025,
		is_student: false,
		student_months: 0,
		disability_level: 2,
		disability_months: 12,
		ztpp: false,
		ztpp_months: 12,
		credit_student: 0,
		credit_disability: 504000,
		created_at: '2026-04-01T00:00:00Z',
		updated_at: '2026-04-01T00:00:00Z'
	},
	total_credits: 504000,
	total_child_benefit: 0
};

const taxConstants = {
	rule_set_id: 'cz-dpfo-2025.v1',
	rule_set_status: 'final',
	rule_set_hash: 'abc123',
	year: 2025,
	basic_credit: 3084000,
	spouse_credit: 2484000,
	spouse_income_limit: 6800000,
	student_credit: 0,
	disability_credit_1: 252000,
	disability_credit_3: 504000,
	disability_ztpp: 1614000,
	child_benefit_1: 1520400,
	child_benefit_2: 2232000,
	child_benefit_3_plus: 2784000,
	progressive_threshold: 167605200,
	flat_rate_caps: { '60': 120000000 },
	deduction_cap_mortgage: 15000000,
	deduction_cap_pension: 2400000,
	deduction_cap_life_insurance: 2400000,
	deduction_cap_savings_combined: 4800000,
	deduction_cap_union: 300000,
	time_test_years: 3,
	security_exemption_limit: 10000000
};

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('Tax credits page - personal credits', () => {
	it('saves the ZTP/P flag together with the disability degree', async () => {
		mockFetch.mockImplementation((url: string, init?: RequestInit) => {
			if (url.includes('/personal') && init?.method === 'PUT') {
				return Promise.resolve(jsonResponse({}));
			}
			if (url.includes('/tax-constants/')) {
				return Promise.resolve(jsonResponse(taxConstants));
			}
			if (url.includes('/tax-credits/')) {
				return Promise.resolve(jsonResponse(summary));
			}
			return Promise.resolve(jsonResponse([]));
		});

		render(Page);

		const checkbox = (await screen.findByLabelText('Držitel průkazu ZTP/P')) as HTMLInputElement;
		expect(checkbox.checked).toBe(false);
		await fireEvent.click(checkbox);

		const heading = screen.getByRole('heading', { name: /Osobní slevy/ });
		const saveButton = heading.parentElement?.querySelector('button:not([aria-label])');
		expect(saveButton).toBeTruthy();
		await fireEvent.click(saveButton as HTMLElement);

		await waitFor(() => {
			const put = mockFetch.mock.calls.find(
				([url, init]) => String(url).includes('/personal') && init?.method === 'PUT'
			);
			expect(put).toBeTruthy();
			expect(JSON.parse(put![1].body as string)).toEqual({
				is_student: false,
				student_months: 0,
				disability_level: 2,
				disability_months: 12,
				ztpp: true,
				ztpp_months: 12
			});
		});
	});

	it('clears a stale student flag when saving in a year with zero student credit', async () => {
		const summaryStudent = {
			...summary,
			personal: { ...summary.personal, is_student: true, student_months: 6 }
		};
		mockFetch.mockImplementation((url: string, init?: RequestInit) => {
			if (url.includes('/personal') && init?.method === 'PUT') {
				return Promise.resolve(jsonResponse({}));
			}
			if (url.includes('/tax-constants/')) {
				return Promise.resolve(jsonResponse(taxConstants)); // student_credit: 0
			}
			if (url.includes('/tax-credits/')) {
				return Promise.resolve(jsonResponse(summaryStudent));
			}
			return Promise.resolve(jsonResponse([]));
		});

		render(Page);

		// The stored flag is hidden (student_credit 0) and reset on load.
		await screen.findByLabelText('Držitel průkazu ZTP/P');
		expect(screen.queryByLabelText('Student')).toBeNull();

		// Change the disability degree and save the personal card.
		const heading = screen.getByRole('heading', { name: /Osobní slevy/ });
		const card = heading.parentElement?.parentElement as HTMLElement;
		const levelSelect = card.querySelector('select') as HTMLSelectElement;
		await fireEvent.change(levelSelect, { target: { value: '1' } });

		const saveButton = heading.parentElement?.querySelector('button:not([aria-label])');
		expect(saveButton).toBeTruthy();
		await fireEvent.click(saveButton as HTMLElement);

		await waitFor(() => {
			const put = mockFetch.mock.calls.find(
				([url, init]) => String(url).includes('/personal') && init?.method === 'PUT'
			);
			expect(put).toBeTruthy();
			expect(JSON.parse(put![1].body as string)).toEqual({
				is_student: false,
				student_months: 0,
				disability_level: 1,
				disability_months: 12,
				ztpp: false,
				ztpp_months: 0
			});
		});
	});
});

describe('Tax credits page - spouse credits', () => {
	it('saves the child-under-3 months together with the spouse credit', async () => {
		const summaryWithSpouse = {
			...summary,
			spouse: {
				id: 1,
				year: 2025,
				spouse_name: 'Jana Novakova',
				spouse_birth_number: '8555012345',
				spouse_income: 5000000,
				spouse_ztp: false,
				months_claimed: 12,
				child_under3_months: 6,
				credit_amount: 1242000,
				created_at: '2026-04-01T00:00:00Z',
				updated_at: '2026-04-01T00:00:00Z'
			}
		};
		mockFetch.mockImplementation((url: string, init?: RequestInit) => {
			if (url.includes('/spouse') && init?.method === 'PUT') {
				return Promise.resolve(jsonResponse({}));
			}
			if (url.includes('/tax-constants/')) {
				return Promise.resolve(jsonResponse(taxConstants));
			}
			if (url.includes('/tax-credits/')) {
				return Promise.resolve(jsonResponse(summaryWithSpouse));
			}
			return Promise.resolve(jsonResponse([]));
		});

		render(Page);

		// The spouse form is shown because summary.spouse exists; its Uložit button
		// is the second one on the page (after the personal credits card).
		const saveButtons = (await screen.findAllByRole('button', { name: 'Uložit' })) as HTMLElement[];
		expect(saveButtons.length).toBeGreaterThanOrEqual(2);
		await fireEvent.click(saveButtons[1]);

		await waitFor(() => {
			const put = mockFetch.mock.calls.find(
				([url, init]) => String(url).includes('/spouse') && init?.method === 'PUT'
			);
			expect(put).toBeTruthy();
			expect(JSON.parse(put![1].body as string)).toEqual({
				spouse_name: 'Jana Novakova',
				spouse_birth_number: '8555012345',
				spouse_income: 5000000,
				spouse_ztp: false,
				months_claimed: 12,
				child_under3_months: 6
			});
		});
	});

	it('defaults the child-under-3 months of a new spouse record to 0', async () => {
		mockFetch.mockImplementation((url: string, init?: RequestInit) => {
			if (url.includes('/spouse') && init?.method === 'PUT') {
				return Promise.resolve(jsonResponse({}));
			}
			if (url.includes('/tax-constants/')) {
				return Promise.resolve(jsonResponse(taxConstants));
			}
			if (url.includes('/tax-credits/')) {
				return Promise.resolve(jsonResponse(summary)); // spouse: null
			}
			return Promise.resolve(jsonResponse([]));
		});

		render(Page);

		// Open the spouse form via its Přidat button.
		const spouseHeading = await screen.findByRole('heading', { name: /Manžel\/ka/ });
		const headerButtons = Array.from(
			spouseHeading.parentElement?.querySelectorAll('button') ?? []
		) as HTMLElement[];
		const addButton = headerButtons.find((b) => b.textContent === 'Přidat');
		expect(addButton).toBeTruthy();
		await fireEvent.click(addButton as HTMLElement);

		// The second Uložit button belongs to the spouse card.
		const saveButtons = (await screen.findAllByRole('button', { name: 'Uložit' })) as HTMLElement[];
		await fireEvent.click(saveButtons[1]);

		await waitFor(() => {
			const put = mockFetch.mock.calls.find(
				([url, init]) => String(url).includes('/spouse') && init?.method === 'PUT'
			);
			expect(put).toBeTruthy();
			const body = JSON.parse(put![1].body as string);
			expect(body.child_under3_months).toBe(0);
			expect(body.months_claimed).toBe(12);
		});
	});
});
