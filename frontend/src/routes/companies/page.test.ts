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

const sampleCompanies = [
	{
		id: 1,
		name: 'Firma A',
		legal_name: 'Firma A s.r.o.',
		ico: '11111111',
		dic: 'CZ11111111',
		vat_registered: true,
		city: 'Praha',
		created_at: '',
		updated_at: ''
	},
	{
		id: 2,
		name: 'Firma B',
		legal_name: 'Firma B',
		ico: '22222222',
		dic: '',
		vat_registered: false,
		city: 'Brno',
		created_at: '',
		updated_at: ''
	}
];

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('Companies list page', () => {
	it('loads companies on mount via the global registry endpoint', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleCompanies));

		render(Page);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalled();
		});

		const url = mockFetch.mock.calls[0][0] as string;
		expect(url).toBe('/api/v1/companies');
	});

	it('renders company rows with name, ICO, and city', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleCompanies));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Firma A')).toBeInTheDocument();
		});

		expect(screen.getByText('Firma B')).toBeInTheDocument();
		expect(screen.getByText('11111111')).toBeInTheDocument();
		expect(screen.getByText('22222222')).toBeInTheDocument();
		expect(screen.getByText('Praha')).toBeInTheDocument();
		expect(screen.getByText('Brno')).toBeInTheDocument();
	});

	it('shows the empty-state action when no companies', async () => {
		mockFetch.mockResolvedValue(jsonResponse([]));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Zatím žádné firmy.')).toBeInTheDocument();
		});

		const cta = screen.getByText('Přidat první firmu').closest('a');
		expect(cta?.getAttribute('href')).toBe('/companies/new');
	});

	it('marks the active company with the "Aktivní" badge', async () => {
		// test-setup seeds company id=1 as the active one.
		mockFetch.mockResolvedValue(jsonResponse(sampleCompanies));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Aktivní')).toBeInTheDocument();
		});
	});

	it('shows error state on API failure', async () => {
		mockFetch.mockRejectedValue(new Error('boom'));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('boom')).toBeInTheDocument();
		});

		// Surfaces via the role="alert" container.
		expect(screen.getByRole('alert')).toBeInTheDocument();
	});

	it('opens a confirm dialog and DELETEs the selected company on confirm', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleCompanies));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Firma A')).toBeInTheDocument();
		});

		// Click the first delete button (for Firma A).
		const deleteButtons = screen.getAllByText('Smazat');
		await fireEvent.click(deleteButtons[0]);

		// Confirm dialog appears.
		await waitFor(() => {
			expect(screen.getByRole('alertdialog')).toBeInTheDocument();
		});

		expect(screen.getByText(/Opravdu chcete smazat firmu/)).toBeInTheDocument();

		// Mock both the DELETE response and the subsequent list refetch.
		mockFetch
			.mockResolvedValueOnce(new Response(null, { status: 204, statusText: 'No Content' }))
			.mockResolvedValueOnce(jsonResponse([sampleCompanies[1]]));

		const dialog = screen.getByRole('alertdialog');
		// Second button inside the dialog is the confirm button.
		const confirmBtn = dialog.querySelectorAll('button')[1] as HTMLElement;
		await fireEvent.click(confirmBtn);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/companies/1',
				expect.objectContaining({ method: 'DELETE' })
			);
		});
	});

	it('cancel button closes the confirm dialog without DELETE', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleCompanies));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Firma A')).toBeInTheDocument();
		});

		const deleteButtons = screen.getAllByText('Smazat');
		await fireEvent.click(deleteButtons[0]);

		await waitFor(() => {
			expect(screen.getByRole('alertdialog')).toBeInTheDocument();
		});

		// Reset the fetch mock so we can assert no further calls.
		mockFetch.mockReset();

		const dialog = screen.getByRole('alertdialog');
		const cancelBtn = dialog.querySelectorAll('button')[0] as HTMLElement;
		await fireEvent.click(cancelBtn);

		await waitFor(() => {
			expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
		});

		expect(mockFetch).not.toHaveBeenCalled();
	});
});
