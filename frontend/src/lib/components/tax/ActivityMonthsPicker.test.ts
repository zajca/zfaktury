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

	it('disables the months that are not allowed', () => {
		render(ActivityMonthsPicker, {
			props: { value: [], onchange: vi.fn(), idPrefix: 't', allowedMonths: [7, 8] }
		});

		expect(screen.getByLabelText('červenec')).toBeEnabled();
		expect(screen.getByLabelText('leden')).toBeDisabled();
	});

	it('selects only the allowed months for the whole year', async () => {
		const onchange = vi.fn();
		render(ActivityMonthsPicker, {
			props: { value: [], onchange, idPrefix: 't', allowedMonths: [7, 8] }
		});

		await fireEvent.click(screen.getByRole('button', { name: 'Celý rok' }));

		expect(onchange).toHaveBeenCalledWith([7, 8]);
	});

	it('does not warn about an empty selection when it is not required', () => {
		render(ActivityMonthsPicker, {
			props: { value: [], onchange: vi.fn(), idPrefix: 't', required: false }
		});

		expect(screen.queryByRole('alert')).toBeNull();
	});

	it('shows a custom legend and hint', () => {
		render(ActivityMonthsPicker, {
			props: { value: [], onchange: vi.fn(), idPrefix: 't', legend: 'Vlastní', hint: 'Nápověda' }
		});

		expect(screen.getByText('Vlastní')).toBeInTheDocument();
		expect(screen.getByText('Nápověda')).toBeInTheDocument();
	});

	it('warns when no month is selected', () => {
		setup([]);

		expect(screen.getByRole('alert')).toHaveTextContent('Vyberte alespoň jeden měsíc činnosti.');
	});
});
