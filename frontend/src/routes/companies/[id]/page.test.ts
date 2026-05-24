import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: {
		params: { id: '2' } as { id: string },
		url: { pathname: '/companies/2', searchParams: new URLSearchParams() }
	}
}));

import Page from './+page.svelte';

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const sampleCompany = {
	id: 2,
	name: 'Firma B',
	legal_name: 'Firma B s.r.o.',
	ico: '22222222',
	dic: 'CZ22222222',
	vat_registered: true,
	street: 'Krátká',
	house_number: '5',
	city: 'Brno',
	zip: '60200',
	email: 'info@firmab.cz',
	phone: '+420555000111',
	first_name: 'Petr',
	last_name: 'Novák',
	bank_account: '7777777777',
	bank_code: '0300',
	iban: 'CZ7700000000007777777777',
	swift: 'CEKOCZPP',
	created_at: '2026-01-01T00:00:00Z',
	updated_at: '2026-01-01T00:00:00Z'
};

beforeEach(async () => {
	mockFetch.mockReset();
	const { goto } = await import('$app/navigation');
	vi.mocked(goto).mockReset();
	const { page } = await import('$app/state');
	(page as any).params = { id: '2' };
});

afterEach(() => {
	cleanup();
});

describe('Companies edit page', () => {
	it('GETs /api/v1/companies/:id on mount and renders the edit heading', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleCompany));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Upravit firmu')).toBeInTheDocument();
		});

		expect(mockFetch).toHaveBeenCalledWith(
			'/api/v1/companies/2',
			expect.objectContaining({ method: 'GET' })
		);
	});

	it('pre-fills the form with the loaded company data', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleCompany));

		render(Page);

		await waitFor(() => {
			expect(screen.getByLabelText('Název *')).toBeInTheDocument();
		});

		const nameInput = screen.getByLabelText('Název *') as HTMLInputElement;
		expect(nameInput.value).toBe('Firma B');

		const icoInput = screen.getByRole('textbox', { name: /IČO/ }) as HTMLInputElement;
		expect(icoInput.value).toBe('22222222');

		const dicInput = screen.getByRole('textbox', { name: /DIČ/ }) as HTMLInputElement;
		expect(dicInput.value).toBe('CZ22222222');

		const cityInput = screen.getByLabelText('Město') as HTMLInputElement;
		expect(cityInput.value).toBe('Brno');

		const vatCheckbox = document.getElementById('company_vat_registered') as HTMLInputElement;
		expect(vatCheckbox).not.toBeNull();
		expect(vatCheckbox.checked).toBe(true);

		const bankAccountInput = screen.getByLabelText('Číslo účtu') as HTMLInputElement;
		expect(bankAccountInput.value).toBe('7777777777');
	});

	it('does NOT render the ARES button in edit mode', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleCompany));

		render(Page);

		await waitFor(() => {
			expect(screen.getByLabelText('Název *')).toBeInTheDocument();
		});

		expect(screen.queryByText('ARES')).not.toBeInTheDocument();
	});

	it('PUTs the updated company and routes back to /companies on save', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleCompany));

		render(Page);

		await waitFor(() => {
			expect(screen.getByLabelText('Název *')).toBeInTheDocument();
		});

		const nameInput = screen.getByLabelText('Název *') as HTMLInputElement;
		await fireEvent.input(nameInput, { target: { value: 'Firma B Updated' } });

		// Update returns 204; then the page refetches the list for the switcher.
		mockFetch
			.mockResolvedValueOnce(new Response(null, { status: 204, statusText: 'No Content' }))
			.mockResolvedValueOnce(jsonResponse([sampleCompany]));

		const submitBtn = screen.getByText('Uložit změny');
		await fireEvent.click(submitBtn);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/companies/2',
				expect.objectContaining({ method: 'PUT' })
			);
		});

		const { goto } = await import('$app/navigation');
		await waitFor(() => {
			expect(goto).toHaveBeenCalledWith('/companies');
		});
	});

	it('shows the role="alert" container on load failure', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ error: 'company not found' }, 404));

		render(Page);

		await waitFor(() => {
			expect(screen.getByRole('alert')).toBeInTheDocument();
		});
	});
});
