import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import { toasts, clearAllToasts } from '$lib/data/toast-state.svelte';
import Page from './+page.svelte';

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
	{
		id: 1,
		key: 'office',
		label_cs: 'Kancelář',
		label_en: 'Office',
		color: '#3B82F6',
		sort_order: 0,
		is_default: true,
		created_at: '2026-01-01'
	},
	{
		id: 2,
		key: 'travel',
		label_cs: 'Cestovné',
		label_en: 'Travel',
		color: '#10B981',
		sort_order: 1,
		is_default: false,
		created_at: '2026-01-01'
	}
];

beforeEach(() => {
	mockFetch.mockReset();
	mockFetch.mockResolvedValue(jsonResponse(sampleCategories));
	clearAllToasts();
});

afterEach(() => {
	cleanup();
});

describe('Categories Settings Page', () => {
	it('loads categories on mount', async () => {
		render(Page);
		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				expect.stringContaining('/api/v1/expense-categories'),
				expect.any(Object)
			);
		});
	});

	it('renders category rows with key and label_cs', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('office')).toBeInTheDocument();
		});
		expect(screen.getByText('Kancelář')).toBeInTheDocument();
		expect(screen.getByText('travel')).toBeInTheDocument();
		expect(screen.getByText('Cestovné')).toBeInTheDocument();
	});

	it('add button shows create form', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('office')).toBeInTheDocument();
		});

		const addBtn = screen.getByText('Přidat kategorii');
		await fireEvent.click(addBtn);

		expect(screen.getByText('Nová kategorie')).toBeInTheDocument();
		expect(document.querySelector('#cat-key')).toBeInTheDocument();
		expect(document.querySelector('#cat-label-cs')).toBeInTheDocument();
		expect(document.querySelector('#cat-label-en')).toBeInTheDocument();
	});

	it('cancel hides form', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('office')).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByText('Přidat kategorii'));
		expect(screen.getByText('Nová kategorie')).toBeInTheDocument();

		await fireEvent.click(screen.getByText('Zrušit'));
		expect(screen.queryByText('Nová kategorie')).not.toBeInTheDocument();
	});

	it('validation error when key missing', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('office')).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByText('Přidat kategorii'));

		// Leave key empty, submit form
		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			expect(toasts.some((t) => t.message === 'Klíč, český a anglický název jsou povinné')).toBe(
				true
			);
		});
	});

	it('create calls POST endpoint', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('office')).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByText('Přidat kategorii'));

		const keyInput = document.querySelector('#cat-key') as HTMLInputElement;
		const labelCsInput = document.querySelector('#cat-label-cs') as HTMLInputElement;
		const labelEnInput = document.querySelector('#cat-label-en') as HTMLInputElement;

		await fireEvent.input(keyInput, { target: { value: 'food' } });
		await fireEvent.input(labelCsInput, { target: { value: 'Jídlo' } });
		await fireEvent.input(labelEnInput, { target: { value: 'Food' } });

		mockFetch.mockResolvedValue(
			jsonResponse({
				id: 3,
				key: 'food',
				label_cs: 'Jídlo',
				label_en: 'Food',
				color: '#6B7280',
				sort_order: 0,
				is_default: false
			})
		);

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			const postCall = mockFetch.mock.calls.find(
				(call: any[]) =>
					typeof call[0] === 'string' &&
					call[0].includes('/api/v1/expense-categories') &&
					call[1]?.method === 'POST'
			);
			expect(postCall).toBeDefined();
		});
	});

	it('edit button populates form', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('office')).toBeInTheDocument();
		});

		const editBtns = screen.getAllByText('Upravit');
		await fireEvent.click(editBtns[0]);

		expect(screen.getByText('Upravit kategorii')).toBeInTheDocument();
		const keyInput = document.querySelector('#cat-key') as HTMLInputElement;
		expect(keyInput.value).toBe('office');
		const labelCsInput = document.querySelector('#cat-label-cs') as HTMLInputElement;
		expect(labelCsInput.value).toBe('Kancelář');
	});

	it('delete with confirmation', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('travel')).toBeInTheDocument();
		});

		// The non-default category has a delete button
		const deleteBtns = screen.getAllByText('Smazat');
		expect(deleteBtns.length).toBeGreaterThanOrEqual(1);

		// After delete (204), loadCategories will be called again
		mockFetch
			.mockResolvedValueOnce(new Response(null, { status: 204 }))
			.mockResolvedValueOnce(jsonResponse([sampleCategories[0]]));

		await fireEvent.click(deleteBtns[0]);

		await waitFor(() => {
			expect(screen.getByRole('alertdialog')).toBeInTheDocument();
		});
		const dialog = screen.getByRole('alertdialog');
		const confirmBtn = dialog.querySelectorAll('button')[1] as HTMLElement;
		await fireEvent.click(confirmBtn);

		await waitFor(() => {
			const deleteCall = mockFetch.mock.calls.find(
				(call: any[]) =>
					typeof call[0] === 'string' &&
					call[0].includes('/api/v1/expense-categories/') &&
					call[1]?.method === 'DELETE'
			);
			expect(deleteCall).toBeDefined();
		});
	});

	it('default category cannot be deleted (shows "výchozí" text)', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('office')).toBeInTheDocument();
		});

		// Default category shows "výchozí" instead of delete button
		expect(screen.getByText('výchozí')).toBeInTheDocument();

		// Only one "Smazat" button (for the non-default category)
		const deleteBtns = screen.getAllByText('Smazat');
		expect(deleteBtns.length).toBe(1);
	});

	it('error state on load failure', async () => {
		mockFetch.mockRejectedValue(new Error('Network error'));

		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Network error')).toBeInTheDocument();
		});
	});
});
