import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import PersonalCreditsCard from './PersonalCreditsCard.svelte';
import PersonalCreditsCardTestWrapper from './PersonalCreditsCardTestWrapper.svelte';

afterEach(() => {
	cleanup();
});

type CardProps = {
	isStudent: boolean;
	studentMonths: number;
	disabilityLevel: number;
	disabilityMonths: number;
	ztpp: boolean;
	ztppMonths: number;
	summary: null;
	taxConstants: null;
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

	it('renders the ZTP/P checkbox and toggles it', async () => {
		renderCard();

		const checkbox = screen.getByLabelText('Držitel průkazu ZTP/P') as HTMLInputElement;
		expect(checkbox.checked).toBe(false);

		await fireEvent.click(checkbox);
		expect(checkbox.checked).toBe(true);
	});

	it('shows the disability months select only when a level is set', () => {
		renderCard(vi.fn(), { disabilityLevel: 0 });
		expect(screen.queryByText('Měsíce (invalidita)')).toBeNull();

		cleanup();
		renderCard(vi.fn(), { disabilityLevel: 1 });
		const label = screen.getByText('Měsíce (invalidita)');
		const select = label.parentElement?.querySelector('select') as HTMLSelectElement;
		expect(select).toBeTruthy();
		expect(Array.from(select.querySelectorAll('option')).map((o) => o.value)).toEqual(
			Array.from({ length: 12 }, (_, i) => String(i + 1))
		);
	});

	it('shows the ZTP/P months select only when ZTP/P is set', async () => {
		renderCard(vi.fn(), { ztpp: false });
		expect(screen.queryByText('Měsíce (ZTP/P)')).toBeNull();

		cleanup();
		renderCard(vi.fn(), { ztpp: true });
		expect(screen.getByText('Měsíce (ZTP/P)')).toBeTruthy();
	});

	it('updates the bound months and submits via onSave', async () => {
		render(PersonalCreditsCardTestWrapper);

		const disabilityLabel = screen.getByText('Měsíce (invalidita)');
		const disabilitySelect = disabilityLabel.parentElement?.querySelector('select') as HTMLSelectElement;
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
