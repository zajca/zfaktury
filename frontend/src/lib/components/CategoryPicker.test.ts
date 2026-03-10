import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import CategoryPicker from './CategoryPicker.svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const sampleCategories = [
	{ key: 'office', label_cs: 'Kancelar' },
	{ key: 'travel', label_cs: 'Cestovne' },
	{ key: 'services', label_cs: 'Sluzby' }
];

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('CategoryPicker', () => {
	it('shows loading state initially', () => {
		mockFetch.mockReturnValue(new Promise(() => {}));

		render(CategoryPicker, { props: { value: '', onchange: vi.fn() } });

		const select = screen.getByRole('combobox') as HTMLSelectElement;
		expect(select.disabled).toBe(true);
		expect(select).toHaveTextContent('Načítám...');
	});

	it('shows categories after loading', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleCategories));

		render(CategoryPicker, { props: { value: '', onchange: vi.fn() } });

		await waitFor(() => {
			expect(screen.getByText('Kancelar')).toBeInTheDocument();
		});

		expect(screen.getByText('Cestovne')).toBeInTheDocument();
		expect(screen.getByText('Sluzby')).toBeInTheDocument();
	});

	it('calls onchange when category is selected', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleCategories));
		const onchange = vi.fn();

		render(CategoryPicker, { props: { value: '', onchange } });

		await waitFor(() => {
			expect(screen.getByText('Kancelar')).toBeInTheDocument();
		});

		const select = screen.getByRole('combobox');
		await fireEvent.change(select, { target: { value: 'travel' } });

		expect(onchange).toHaveBeenCalledWith('travel');
	});

	it('switches to custom input mode', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleCategories));
		const onchange = vi.fn();

		render(CategoryPicker, { props: { value: '', onchange } });

		await waitFor(() => {
			expect(screen.getByText('Kancelar')).toBeInTheDocument();
		});

		const select = screen.getByRole('combobox');
		await fireEvent.change(select, { target: { value: '__custom__' } });

		expect(onchange).toHaveBeenCalledWith('');

		await waitFor(() => {
			expect(screen.getByPlaceholderText('Vlastní kategorie...')).toBeInTheDocument();
		});
	});

	it('calls onchange with custom value', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleCategories));
		const onchange = vi.fn();

		render(CategoryPicker, { props: { value: '', onchange } });

		await waitFor(() => {
			expect(screen.getByText('Kancelar')).toBeInTheDocument();
		});

		const select = screen.getByRole('combobox');
		await fireEvent.change(select, { target: { value: '__custom__' } });

		await waitFor(() => {
			expect(screen.getByPlaceholderText('Vlastní kategorie...')).toBeInTheDocument();
		});

		const textInput = screen.getByPlaceholderText('Vlastní kategorie...') as HTMLInputElement;
		textInput.value = 'Custom cat';
		await fireEvent.input(textInput);

		expect(onchange).toHaveBeenCalledWith('Custom cat');
	});

	it('shows error state on API failure', async () => {
		mockFetch.mockRejectedValueOnce(new Error('Network error'));

		render(CategoryPicker, { props: { value: '', onchange: vi.fn() } });

		await waitFor(() => {
			expect(screen.getByText('Network error')).toBeInTheDocument();
		});
	});

	it('enters custom mode if initial value does not match any category', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleCategories));

		render(CategoryPicker, { props: { value: 'unknown-cat', onchange: vi.fn() } });

		await waitFor(() => {
			expect(screen.getByPlaceholderText('Vlastní kategorie...')).toBeInTheDocument();
		});
	});
});
