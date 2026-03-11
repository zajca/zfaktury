import { describe, it, expect, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';
import LoadingSpinner from './LoadingSpinner.svelte';

afterEach(() => {
	cleanup();
});

describe('LoadingSpinner', () => {
	it('renders spinner with status role', () => {
		render(LoadingSpinner);
		expect(screen.getByRole('status')).toBeInTheDocument();
	});

	it('has sr-only loading text', () => {
		render(LoadingSpinner);
		expect(screen.getByText('Nacitani...')).toBeInTheDocument();
	});

	it('renders spinning animation element', () => {
		render(LoadingSpinner);
		const spinner = screen.getByRole('status').querySelector('div');
		expect(spinner?.className).toContain('animate-spin');
	});

	it('accepts custom class', () => {
		const { container } = render(LoadingSpinner, { props: { class: 'mt-8 p-12' } });
		const wrapper = container.firstElementChild;
		expect(wrapper?.className).toContain('mt-8 p-12');
	});

	it('has default flex centering classes', () => {
		const { container } = render(LoadingSpinner, { props: {} });
		const wrapper = container.firstElementChild;
		expect(wrapper?.className).toContain('flex');
		expect(wrapper?.className).toContain('items-center');
		expect(wrapper?.className).toContain('justify-center');
	});
});
