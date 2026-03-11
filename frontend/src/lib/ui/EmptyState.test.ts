import { describe, it, expect, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';
import EmptyState from './EmptyState.svelte';

afterEach(() => {
	cleanup();
});

describe('EmptyState', () => {
	it('renders default message', () => {
		render(EmptyState, { props: { message: 'No items yet.' } });
		expect(screen.getByText('No items yet.')).toBeInTheDocument();
	});

	it('renders filtered message when isFiltered is true', () => {
		render(EmptyState, {
			props: {
				message: 'No items yet.',
				filteredMessage: 'No items match filter.',
				isFiltered: true
			}
		});
		expect(screen.getByText('No items match filter.')).toBeInTheDocument();
		expect(screen.queryByText('No items yet.')).not.toBeInTheDocument();
	});

	it('renders default message when isFiltered is false', () => {
		render(EmptyState, {
			props: {
				message: 'No items yet.',
				filteredMessage: 'No items match filter.',
				isFiltered: false
			}
		});
		expect(screen.getByText('No items yet.')).toBeInTheDocument();
	});

	it('renders default message when filteredMessage is not provided even if isFiltered', () => {
		render(EmptyState, {
			props: {
				message: 'No items yet.',
				isFiltered: true
			}
		});
		expect(screen.getByText('No items yet.')).toBeInTheDocument();
	});

	it('accepts custom class', () => {
		const { container } = render(EmptyState, { props: { message: 'Empty', class: 'custom-empty' } });
		const wrapper = container.firstElementChild;
		expect(wrapper?.className).toContain('custom-empty');
	});

	it('has centered text styling', () => {
		const { container } = render(EmptyState, { props: { message: 'Empty' } });
		const wrapper = container.firstElementChild;
		expect(wrapper?.className).toContain('text-center');
		expect(wrapper?.className).toContain('text-muted');
	});
});
