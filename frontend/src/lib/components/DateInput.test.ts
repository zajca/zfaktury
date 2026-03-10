import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import DateInput from './DateInput.svelte';

beforeEach(() => {
	vi.useFakeTimers();
	vi.setSystemTime(new Date('2026-03-10T12:00:00Z'));
});

afterEach(() => {
	cleanup();
	vi.useRealTimers();
});

describe('DateInput', () => {
	it('renders date input with initial value', () => {
		render(DateInput, { props: { value: '2026-03-15' } });

		const dateInput = document.querySelector('input[type="date"]') as HTMLInputElement;
		expect(dateInput).toBeInTheDocument();
		expect(dateInput.value).toBe('2026-03-15');
	});

	it('renders Dnes button', () => {
		render(DateInput, { props: { value: '2026-03-15' } });

		const btn = screen.getByRole('button', { name: /dnes/i });
		expect(btn).toBeInTheDocument();
	});

	it('Dnes button sets today date and calls onchange', async () => {
		const onchange = vi.fn();
		render(DateInput, { props: { value: '2026-01-01', onchange } });

		const btn = screen.getByRole('button', { name: /dnes/i });
		await fireEvent.click(btn);

		expect(onchange).toHaveBeenCalledWith('2026-03-10');
	});

	it('T key shortcut sets today date', async () => {
		const onchange = vi.fn();
		render(DateInput, { props: { value: '2026-01-01', onchange } });

		const dateInput = document.querySelector('input[type="date"]') as HTMLInputElement;
		await fireEvent.keyDown(dateInput, { key: 't' });

		expect(onchange).toHaveBeenCalledWith('2026-03-10');
	});

	it('renders preset buttons when presets are provided', () => {
		const presets = [
			{ label: '+14 dni', days: 14 },
			{ label: '+30 dni', days: 30 }
		];
		render(DateInput, { props: { value: '2026-03-10', presets } });

		expect(screen.getByRole('button', { name: '+14 dni' })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: '+30 dni' })).toBeInTheDocument();
	});

	it('does not render preset buttons when no presets', () => {
		render(DateInput, { props: { value: '2026-03-10' } });

		const buttons = screen.getAllByRole('button');
		expect(buttons).toHaveLength(1);
	});

	it('preset click sets correct date and calls onchange', async () => {
		const onchange = vi.fn();
		const presets = [{ label: '+14 dni', days: 14 }];
		render(DateInput, { props: { value: '2026-03-10', presets, onchange } });

		const presetBtn = screen.getByRole('button', { name: '+14 dni' });
		await fireEvent.click(presetBtn);

		expect(onchange).toHaveBeenCalledWith('2026-03-24');
	});

	it('preset uses relativeToValue when provided', async () => {
		const onchange = vi.fn();
		const presets = [{ label: '+14 dni', days: 14 }];
		render(DateInput, {
			props: { value: '2026-03-10', presets, onchange, relativeToValue: '2026-04-01' }
		});

		const presetBtn = screen.getByRole('button', { name: '+14 dni' });
		await fireEvent.click(presetBtn);

		expect(onchange).toHaveBeenCalledWith('2026-04-15');
	});

	it('displays formatted Czech date', () => {
		render(DateInput, { props: { value: '2026-03-10' } });

		const formatted = document.querySelector('.text-xs.text-gray-400');
		expect(formatted).toBeInTheDocument();
		expect(formatted?.textContent).toContain('2026');
	});

	it('propagates required attribute', () => {
		const { container } = render(DateInput, { props: { value: '2026-03-10', required: true } });

		const dateInput = container.querySelector('input[type="date"]') as HTMLInputElement;
		expect(dateInput).toBeInTheDocument();
		expect(dateInput.hasAttribute('required')).toBe(true);
	});

	it('calls onchange on manual input', async () => {
		const onchange = vi.fn();
		render(DateInput, { props: { value: '2026-03-10', onchange } });

		const dateInput = document.querySelector('input[type="date"]') as HTMLInputElement;
		// Simulate native input: set value then dispatch input event
		dateInput.value = '2026-05-01';
		await fireEvent.input(dateInput);

		expect(onchange).toHaveBeenCalledWith('2026-05-01');
	});
});
