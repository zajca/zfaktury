import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import Pagination from './Pagination.svelte';

afterEach(() => {
	cleanup();
});

describe('Pagination', () => {
	it('does not render when totalPages is 1', () => {
		const { container } = render(Pagination, {
			props: { page: 1, totalPages: 1, total: 5, onPageChange: vi.fn() }
		});
		expect(container.innerHTML.trim()).toBe('<!---->');
	});

	it('renders when totalPages > 1', () => {
		render(Pagination, {
			props: { page: 1, totalPages: 3, total: 75, onPageChange: vi.fn() }
		});
		expect(screen.getByText('Celkem 75 položek')).toBeInTheDocument();
		expect(screen.getByText('1 / 3')).toBeInTheDocument();
	});

	it('renders custom label', () => {
		render(Pagination, {
			props: { page: 1, totalPages: 2, total: 30, label: 'faktur', onPageChange: vi.fn() }
		});
		expect(screen.getByText('Celkem 30 faktur')).toBeInTheDocument();
	});

	it('disables previous button on first page', () => {
		render(Pagination, {
			props: { page: 1, totalPages: 3, total: 75, onPageChange: vi.fn() }
		});
		const prevBtn = screen.getByText('Předchozí');
		expect(prevBtn).toBeDisabled();
	});

	it('disables next button on last page', () => {
		render(Pagination, {
			props: { page: 3, totalPages: 3, total: 75, onPageChange: vi.fn() }
		});
		const nextBtn = screen.getByText('Další');
		expect(nextBtn).toBeDisabled();
	});

	it('calls onPageChange with previous page', async () => {
		const onPageChange = vi.fn();
		render(Pagination, {
			props: { page: 2, totalPages: 3, total: 75, onPageChange }
		});
		await fireEvent.click(screen.getByText('Předchozí'));
		expect(onPageChange).toHaveBeenCalledWith(1);
	});

	it('calls onPageChange with next page', async () => {
		const onPageChange = vi.fn();
		render(Pagination, {
			props: { page: 2, totalPages: 3, total: 75, onPageChange }
		});
		await fireEvent.click(screen.getByText('Další'));
		expect(onPageChange).toHaveBeenCalledWith(3);
	});

	it('does not go below page 1', async () => {
		const onPageChange = vi.fn();
		render(Pagination, {
			props: { page: 1, totalPages: 3, total: 75, onPageChange }
		});
		// Previous button is disabled, but verify logic
		const prevBtn = screen.getByText('Předchozí');
		expect(prevBtn).toBeDisabled();
	});
});
