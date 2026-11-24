import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import ActivityMonthsPicker from './ActivityMonthsPicker.svelte';

afterEach(() => {
	cleanup();
});

function setup(value: number[]) {
	const onchange = vi.fn();
	render(ActivityMonthsPicker, {
		props: { value, onchange, idPrefix: 'test-months' }
	});
	return { onchange };
}

describe('ActivityMonthsPicker', () => {
	it('renders twelve month checkboxes with Czech names', () => {
		setup([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);

		expect(screen.getAllByRole('checkbox')).toHaveLength(12);
		expect(screen.getByLabelText('leden')).toBeInTheDocument();
		expect(screen.getByLabelText('prosinec')).toBeInTheDocument();
	});

	it('checks only the selected months', () => {
		setup([7, 8, 9, 10, 11, 12]);

		expect(screen.getByLabelText('červenec')).toBeChecked();
		expect(screen.getByLabelText('leden')).not.toBeChecked();
	});

	it('adds a month to the selection in ascending order', async () => {
		const { onchange } = setup([7, 8, 9, 10, 11, 12]);

		await fireEvent.click(screen.getByLabelText('leden'));

		expect(onchange).toHaveBeenCalledWith([1, 7, 8, 9, 10, 11, 12]);
	});

	it('selects the whole year', async () => {
		const { onchange } = setup([7]);

		await fireEvent.click(screen.getByRole('button', { name: 'Celý rok' }));

		expect(onchange).toHaveBeenCalledWith([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
	});

	it('warns when no month is selected', () => {
		setup([]);

		expect(screen.getByRole('alert')).toHaveTextContent('Vyberte alespoň jeden měsíc činnosti.');
	});
});
