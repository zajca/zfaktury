import { describe, it, expect, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';
import ErrorAlert from './ErrorAlert.svelte';

afterEach(() => {
	cleanup();
});

describe('ErrorAlert', () => {
	it('renders error message with alert role', () => {
		render(ErrorAlert, { props: { error: 'Something went wrong' } });
		const alert = screen.getByRole('alert');
		expect(alert).toBeInTheDocument();
		expect(alert.textContent?.trim()).toBe('Something went wrong');
	});

	it('does not render when error is null', () => {
		render(ErrorAlert, { props: { error: null } });
		expect(screen.queryByRole('alert')).not.toBeInTheDocument();
	});

	it('has danger styling classes', () => {
		render(ErrorAlert, { props: { error: 'Error' } });
		const alert = screen.getByRole('alert');
		expect(alert.className).toContain('text-danger');
		expect(alert.className).toContain('bg-danger-bg');
	});

	it('accepts custom class', () => {
		render(ErrorAlert, { props: { error: 'Error', class: 'mt-4' } });
		const alert = screen.getByRole('alert');
		expect(alert.className).toContain('mt-4');
	});
});
