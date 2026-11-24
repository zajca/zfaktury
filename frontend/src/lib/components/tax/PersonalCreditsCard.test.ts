import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import PersonalCreditsCard from './PersonalCreditsCard.svelte';

afterEach(() => {
	cleanup();
});

function renderCard(onSave = vi.fn()) {
	render(PersonalCreditsCard, {
		isStudent: false,
		studentMonths: 12,
		disabilityLevel: 0,
		ztpp: false,
		summary: null,
		taxConstants: null,
		saving: false,
		onSave
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
});
