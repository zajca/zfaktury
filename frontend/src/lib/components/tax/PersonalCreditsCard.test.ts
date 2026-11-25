import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import type { TaxConstants } from '$lib/api/client';
import PersonalCreditsCard from './PersonalCreditsCard.svelte';
import PersonalCreditsCardTestWrapper from './PersonalCreditsCardTestWrapper.svelte';

afterEach(() => {
	cleanup();
});

function makeTaxConstants(studentCredit: number): TaxConstants {
	return {
		rule_set_id: 'cz-dpfo-2025.v1',
		rule_set_status: 'final',
		rule_set_hash: 'abc123',
		year: 2025,
		basic_credit: 3084000,
		spouse_credit: 2484000,
		spouse_income_limit: 6800000,
		student_credit: studentCredit,
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
}

type CardProps = {
	isStudent: boolean;
	studentMonths: number;
	disabilityLevel: number;
	disabilityMonths: number;
	ztpp: boolean;
	ztppMonths: number;
	summary: null;
	taxConstants: TaxConstants | null;
	saving: boolean;
	onSave: () => void;
};

function renderCard(onSave = vi.fn(), overrides: Partial<CardProps> = {}) {
	render(PersonalCreditsCard, {
		isStudent: false,
		studentMonths: 12,
		disabilityLevel: 0,
		disabilityMonths: 12,
		ztpp: false,
		ztppMonths: 12,
		summary: null,
		taxConstants: null,
		saving: false,
		onSave,
		...overrides
	});
	return onSave;
}

describe('PersonalCreditsCard', () => {
	it('offers only the three disability degrees', () => {
		renderCard();

		const options = Array.from(screen.getByRole('combobox').querySelectorAll('option')).map(
			(o) => o.textContent
		);
		expect(options).toEqual(['Žádná', '1. a 2. stupeň', '3. stupeň']);
	});

	it('hides the student checkbox and months when the year student credit is 0', () => {
		renderCard(vi.fn(), { taxConstants: makeTaxConstants(0) });

		expect(screen.queryByLabelText('Student')).toBeNull();
	});

	it('shows the abolition note in place of the student controls when the credit is 0', () => {
		renderCard(vi.fn(), { taxConstants: makeTaxConstants(0) });

		expect(screen.getByText('Sleva na studenta se od roku 2024 neuplatňuje.')).toBeTruthy();
		expect(screen.queryByLabelText('Student')).toBeNull();
	});

	it('shows the student controls when the tax constants are not loaded', () => {
		renderCard(vi.fn(), { taxConstants: null });

		expect(screen.getByLabelText('Student')).toBeTruthy();
	});

	it('hides the student months select even when the stored flag is set, for a zero-credit year', () => {
		renderCard(vi.fn(), { isStudent: true, taxConstants: makeTaxConstants(0) });

		expect(screen.queryByLabelText('Student')).toBeNull();
		expect(screen.queryAllByRole('combobox').length).toBe(1); // only the disability select
	});

	it('shows the student checkbox and months select when the year student credit is above 0', async () => {
		renderCard(vi.fn(), { taxConstants: makeTaxConstants(435600) });

		const checkbox = screen.getByLabelText('Student') as HTMLInputElement;
		expect(checkbox.checked).toBe(false);
		await fireEvent.click(checkbox);
		expect(checkbox.checked).toBe(true);
		// Two selects once the student months appear: student + disability.
		expect(screen.getAllByRole('combobox').length).toBe(2);
	});

	it('renders the ZTP/P checkbox and toggles it', async () => {
		renderCard();

		const checkbox = screen.getByLabelText('Držitel průkazu ZTP/P') as HTMLInputElement;
		expect(checkbox.checked).toBe(false);

		await fireEvent.click(checkbox);
		expect(checkbox.checked).toBe(true);
	});

	it('shows the disability months select only when a level is set, labelled for it', () => {
		renderCard(vi.fn(), { disabilityLevel: 0 });
		expect(screen.queryByLabelText('Měsíce (invalidita)')).toBeNull();

		cleanup();
		renderCard(vi.fn(), { disabilityLevel: 1 });
		const select = screen.getByLabelText('Měsíce (invalidita)') as HTMLSelectElement;
		expect(select.tagName).toBe('SELECT');
		expect(Array.from(select.querySelectorAll('option')).map((o) => o.value)).toEqual(
			Array.from({ length: 12 }, (_, i) => String(i + 1))
		);
	});

	it('shows the ZTP/P months select only when ZTP/P is set, labelled for it', async () => {
		renderCard(vi.fn(), { ztpp: false });
		expect(screen.queryByLabelText('Měsíce (ZTP/P)')).toBeNull();

		cleanup();
		renderCard(vi.fn(), { ztpp: true });
		expect((screen.getByLabelText('Měsíce (ZTP/P)') as HTMLSelectElement).tagName).toBe('SELECT');
	});

	it('defaults the months to 12 when the credit is switched on with stored months 0', async () => {
		render(PersonalCreditsCardTestWrapper, {
			initialLevel: 0,
			initialDisabilityMonths: 0,
			initialZtpp: false,
			initialZtppMonths: 0
		});

		const levelSelect = screen.getByRole('combobox') as HTMLSelectElement;
		await fireEvent.change(levelSelect, { target: { value: '1' } });
		expect(screen.getByTestId('disabilityMonths').textContent).toBe('12');
		expect((screen.getByLabelText('Měsíce (invalidita)') as HTMLSelectElement).value).toBe('12');

		await fireEvent.click(screen.getByLabelText('Držitel průkazu ZTP/P'));
		expect(screen.getByTestId('ztppMonths').textContent).toBe('12');
		expect((screen.getByLabelText('Měsíce (ZTP/P)') as HTMLSelectElement).value).toBe('12');
	});

	it('updates the bound months and submits via onSave', async () => {
		render(PersonalCreditsCardTestWrapper);

		const disabilityLabel = screen.getByText('Měsíce (invalidita)');
		const disabilitySelect = disabilityLabel.parentElement?.querySelector(
			'select'
		) as HTMLSelectElement;
		await fireEvent.change(disabilitySelect, { target: { value: '6' } });
		expect(screen.getByTestId('disabilityMonths').textContent).toBe('6');

		const ztppLabel = screen.getByText('Měsíce (ZTP/P)');
		const ztppSelect = ztppLabel.parentElement?.querySelector('select') as HTMLSelectElement;
		await fireEvent.change(ztppSelect, { target: { value: '9' } });
		expect(screen.getByTestId('ztppMonths').textContent).toBe('9');

		await fireEvent.click(screen.getByRole('button', { name: 'Uložit' }));
		expect(screen.getByTestId('saved').textContent).toBe('true');
	});
});
