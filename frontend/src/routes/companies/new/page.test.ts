import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: {
		params: {},
		url: { pathname: '/companies/new', searchParams: new URLSearchParams() }
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

const createdCompany = {
	id: 7,
	name: 'New OSVC',
	legal_name: 'New OSVC s.r.o.',
	ico: '99999999',
	dic: '',
	vat_registered: false,
	created_at: '',
	updated_at: ''
};

const aresResult = {
	ico: '12345678',
	dic: 'CZ12345678',
	name: 'ARES Corp s.r.o.',
	street: 'Dlouha 10',
	city: 'Brno',
	zip: '60200',
	country: 'CZ'
};

beforeEach(async () => {
	mockFetch.mockReset();
	const { goto } = await import('$app/navigation');
	vi.mocked(goto).mockReset();
	const { page } = await import('$app/state');
	(page as any).url = { pathname: '/companies/new', searchParams: new URLSearchParams() };
});

afterEach(() => {
	cleanup();
});

describe('Companies new page', () => {
	it('renders the "Nová firma" heading and an empty form', async () => {
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Nová firma')).toBeInTheDocument();
		});

		const nameInput = screen.getByLabelText('Název *') as HTMLInputElement;
		expect(nameInput.value).toBe('');

		const icoInput = screen.getByRole('textbox', { name: /IČO/ }) as HTMLInputElement;
		expect(icoInput.value).toBe('');
	});

	it('renders the ARES lookup button', async () => {
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('ARES')).toBeInTheDocument();
		});
	});

	it('ARES lookup fills name, DIČ, and address fields', async () => {
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Nová firma')).toBeInTheDocument();
		});

		const icoInput = screen.getByRole('textbox', { name: /IČO/ }) as HTMLInputElement;
		await fireEvent.input(icoInput, { target: { value: '12345678' } });

		// Test-setup seeds company id=1 active, so the per-company ARES URL resolves.
		mockFetch.mockResolvedValueOnce(jsonResponse(aresResult));

		const aresBtn = screen.getByText('ARES');
		await fireEvent.click(aresBtn);

		await waitFor(() => {
			const nameInput = screen.getByLabelText('Název *') as HTMLInputElement;
			expect(nameInput.value).toBe('ARES Corp s.r.o.');
		});

		const dicInput = screen.getByRole('textbox', { name: /DIČ/ }) as HTMLInputElement;
		expect(dicInput.value).toBe('CZ12345678');

		const cityInput = screen.getByLabelText('Město') as HTMLInputElement;
		expect(cityInput.value).toBe('Brno');

		const zipInput = screen.getByLabelText('PSČ') as HTMLInputElement;
		expect(zipInput.value).toBe('60200');

		// The ARES request goes through the per-company URL.
		const url = mockFetch.mock.calls[0][0] as string;
		expect(url).toContain('/contacts/ares/12345678');
	});

	it('pre-fills the form via ARES when ?ico=... query param is present', async () => {
		const { page } = await import('$app/state');
		(page as any).url = {
			pathname: '/companies/new',
			searchParams: new URLSearchParams('ico=12345678')
		};

		mockFetch.mockResolvedValueOnce(jsonResponse(aresResult));

		render(Page);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalled();
		});

		const url = mockFetch.mock.calls[0][0] as string;
		expect(url).toContain('/contacts/ares/12345678');

		await waitFor(() => {
			const nameInput = screen.getByLabelText('Název *') as HTMLInputElement;
			expect(nameInput.value).toBe('ARES Corp s.r.o.');
		});
	});

	it('submits POST /api/v1/companies and routes home on success', async () => {
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Nová firma')).toBeInTheDocument();
		});

		// Remove HTML5 required so fireEvent.click submit isn't blocked by
		// native validation when we omit some fields.
		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		const nameInput = screen.getByLabelText('Název *') as HTMLInputElement;
		await fireEvent.input(nameInput, { target: { value: 'New OSVC' } });

		const icoInput = screen.getByRole('textbox', { name: /IČO/ }) as HTMLInputElement;
		await fireEvent.input(icoInput, { target: { value: '99999999' } });

		// POST creates the company, then the page re-reads the list to refresh
		// the switcher and activates the new company.
		mockFetch
			.mockResolvedValueOnce(jsonResponse(createdCompany))
			.mockResolvedValueOnce(jsonResponse([createdCompany]));

		const submitBtn = screen.getByText('Vytvořit firmu');
		await fireEvent.click(submitBtn);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/companies',
				expect.objectContaining({ method: 'POST' })
			);
		});

		// The post-create flow re-reads the list (for the switcher) and routes home.
		const { goto } = await import('$app/navigation');
		await waitFor(() => {
			expect(goto).toHaveBeenCalledWith('/');
		});
	});

	it('surfaces a server error via the role="alert" container', async () => {
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Nová firma')).toBeInTheDocument();
		});

		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		const nameInput = screen.getByLabelText('Název *') as HTMLInputElement;
		await fireEvent.input(nameInput, { target: { value: 'X' } });

		mockFetch.mockResolvedValueOnce(jsonResponse({ error: 'duplicate ico' }, 409));

		const submitBtn = screen.getByText('Vytvořit firmu');
		await fireEvent.click(submitBtn);

		await waitFor(() => {
			expect(screen.getByRole('alert')).toBeInTheDocument();
		});
	});
});
