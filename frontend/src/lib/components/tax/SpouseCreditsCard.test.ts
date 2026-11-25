import { describe, it, expect, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import SpouseCreditsCard from './SpouseCreditsCard.svelte';
import SpouseCreditsCardTestWrapper from './SpouseCreditsCardTestWrapper.svelte';

afterEach(() => {
	cleanup();
});

describe('SpouseCreditsCard', () => {
	it('renders the child-under-3 months input with a 0-12 range and a hint', () => {
		render(SpouseCreditsCard, {
			spouseName: 'Jana Novakova',
			spouseBirthNumber: '8555012345',
			spouseIncome: 50000,
			spouseZtp: false,
			spouseMonths: 12,
			spouseChildUnder3Months: 6,
			showSpouseForm: true,
			summary: null,
			taxConstants: null,
			saving: false,
			onSave: () => {},
			onDelete: () => {}
		});

		const select = screen.getByLabelText(
			'Měsíce s dítětem do 3 let ve společné domácnosti'
		) as HTMLSelectElement;
		expect(select.tagName).toBe('SELECT');
		const options = Array.from(select.querySelectorAll('option')).map((o) => o.value);
		expect(options).toEqual(Array.from({ length: 13 }, (_, i) => String(i)));
		expect(select.value).toBe('6');
		// The hint is associated with the select via aria-describedby.
		const hintId = select.getAttribute('aria-describedby');
		expect(hintId).toBeTruthy();
		expect(document.getElementById(hintId ?? '')?.textContent).toContain(
			'Počítají se jen měsíce, na jejichž počátku bylo dítě mladší 3 let.'
		);
	});

	it('updates the bound child-under-3 months and submits via onSave', async () => {
		render(SpouseCreditsCardTestWrapper);

		const select = screen.getByLabelText(
			'Měsíce s dítětem do 3 let ve společné domácnosti'
		) as HTMLSelectElement;
		await fireEvent.change(select, { target: { value: '6' } });
		expect(screen.getByTestId('childUnder3Months').textContent).toBe('6');

		await fireEvent.click(screen.getByRole('button', { name: 'Uložit' }));
		expect(screen.getByTestId('saved').textContent).toBe('true');
	});
});
