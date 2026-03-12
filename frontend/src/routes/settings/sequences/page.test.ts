import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
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

const sampleSequences = [
	{
		id: 1,
		prefix: 'FV',
		year: 2026,
		next_number: 5,
		format_pattern: '{prefix}{year}{number:04d}',
		preview: 'FV20260005'
	},
	{
		id: 2,
		prefix: 'ZF',
		year: 2026,
		next_number: 1,
		format_pattern: '{prefix}{year}{number:04d}',
		preview: 'ZF20260001'
	}
];

beforeEach(() => {
	mockFetch.mockReset();
	mockFetch.mockResolvedValue(jsonResponse(sampleSequences));
});

afterEach(() => {
	cleanup();
});

describe('Sequences Settings Page', () => {
	it('loads sequences on mount', async () => {
		render(Page);
		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				expect.stringContaining('/api/v1/invoice-sequences'),
				expect.any(Object)
			);
		});
	});

	it('renders sequence rows with prefix, year, next_number, preview', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('FV')).toBeInTheDocument();
		});
		// Both sequences have year 2026
		expect(screen.getAllByText('2026').length).toBe(2);
		expect(screen.getByText('5')).toBeInTheDocument();
		expect(screen.getByText('FV20260005')).toBeInTheDocument();
		expect(screen.getByText('ZF')).toBeInTheDocument();
		expect(screen.getByText('ZF20260001')).toBeInTheDocument();
	});

	it('empty state message', async () => {
		mockFetch.mockResolvedValue(jsonResponse([]));

		render(Page);
		await waitFor(() => {
			expect(screen.getByText(/Zatím žádné číselné řady/)).toBeInTheDocument();
		});
	});

	it('"Nová řada" button toggles create form', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('FV')).toBeInTheDocument();
		});

		const newBtn = screen.getByText('Nová řada');
		await fireEvent.click(newBtn);

		expect(screen.getByText('Nová číselná řada')).toBeInTheDocument();
		expect(document.querySelector('#create-prefix')).toBeInTheDocument();
		expect(document.querySelector('#create-year')).toBeInTheDocument();
		expect(document.querySelector('#create-next')).toBeInTheDocument();

		// Toggle off
		await fireEvent.click(newBtn);
		expect(screen.queryByText('Nová číselná řada')).not.toBeInTheDocument();
	});

	it('create form shows format preview', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('FV')).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByText('Nová řada'));

		// Default preview: FV{currentYear}0001
		await waitFor(() => {
			const previewText = screen.getByText(/Náhled:/);
			expect(previewText).toBeInTheDocument();
			// The preview contains the formatted string in a span
			const previewSpan = previewText.querySelector('.font-mono');
			expect(previewSpan).toBeInTheDocument();
		});
	});

	it('create calls POST endpoint', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('FV')).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByText('Nová řada'));

		mockFetch.mockResolvedValue(
			jsonResponse({
				id: 3,
				prefix: 'DN',
				year: 2026,
				next_number: 1,
				format_pattern: '{prefix}{year}{number:04d}',
				preview: 'DN20260001'
			})
		);

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			const postCall = mockFetch.mock.calls.find(
				(call: any[]) =>
					typeof call[0] === 'string' &&
					call[0].includes('/api/v1/invoice-sequences') &&
					call[1]?.method === 'POST'
			);
			expect(postCall).toBeDefined();
		});
	});

	it('edit button shows inline input', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('FV')).toBeInTheDocument();
		});

		const editBtns = screen.getAllByText('Upravit');
		await fireEvent.click(editBtns[0]);

		// Should show an inline number input for next_number
		await waitFor(() => {
			const numberInput = document.querySelector('input[type="number"]') as HTMLInputElement;
			expect(numberInput).toBeInTheDocument();
		});

		// Should show save/cancel buttons in edit mode
		expect(screen.getByText('Uložit')).toBeInTheDocument();
	});

	it('delete with confirmation', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('FV')).toBeInTheDocument();
		});

		const deleteBtns = screen.getAllByText('Smazat');
		expect(deleteBtns.length).toBeGreaterThanOrEqual(1);

		// After delete (204), loadSequences will be called again - mock both responses
		mockFetch
			.mockResolvedValueOnce(new Response(null, { status: 204 }))
			.mockResolvedValueOnce(jsonResponse([sampleSequences[1]]));

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
					call[0].includes('/api/v1/invoice-sequences/') &&
					call[1]?.method === 'DELETE'
			);
			expect(deleteCall).toBeDefined();
		});
	});

	it('error state on load failure', async () => {
		mockFetch.mockRejectedValue(new Error('Network error'));

		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Network error')).toBeInTheDocument();
		});
	});
});
