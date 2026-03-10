import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);
vi.stubGlobal('confirm', vi.fn(() => true));

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/state', () => ({
	page: {
		params: { id: '1' } as { id: string },
		url: { pathname: '/contacts/1', searchParams: new URLSearchParams() }
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

const sampleContact = {
	id: 1,
	type: 'company',
	name: 'Test Corp',
	ico: '12345678',
	dic: 'CZ12345678',
	street: 'Hlavni 1',
	city: 'Praha',
	zip: '10000',
	country: 'CZ',
	email: 'test@example.com',
	phone: '+420123456789',
	web: '',
	bank_account: '123456789',
	bank_code: '0100',
	iban: '',
	swift: '',
	payment_terms_days: 14,
	tags: '',
	notes: '',
	is_favorite: false,
	vat_unreliable_at: null,
	created_at: '2026-03-01T00:00:00Z',
	updated_at: '2026-03-01T00:00:00Z'
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

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('Contact detail page - new mode', () => {
	beforeEach(async () => {
		const { page } = await import('$app/state');
		(page as any).params = { id: 'new' };
		(page as any).url = { pathname: '/contacts/new', searchParams: new URLSearchParams() };
	});

	it('renders heading "Novy kontakt"', async () => {
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Novy kontakt')).toBeInTheDocument();
		});
	});

	it('no delete button', async () => {
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Novy kontakt')).toBeInTheDocument();
		});

		expect(screen.queryByText('Smazat')).not.toBeInTheDocument();
	});

	it('renders empty form fields', async () => {
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Novy kontakt')).toBeInTheDocument();
		});

		const nameInput = screen.getByLabelText('Nazev') as HTMLInputElement;
		expect(nameInput.value).toBe('');

		const icoInput = screen.getByLabelText('ICO') as HTMLInputElement;
		expect(icoInput.value).toBe('');
	});

	it('submit for new contact calls POST', async () => {
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Novy kontakt')).toBeInTheDocument();
		});

		const nameInput = screen.getByLabelText('Nazev') as HTMLInputElement;
		await fireEvent.input(nameInput, { target: { value: 'New Company' } });

		mockFetch.mockResolvedValueOnce(
			jsonResponse({ ...sampleContact, id: 2, name: 'New Company' })
		);

		const submitBtn = screen.getByText('Ulozit');
		await fireEvent.click(submitBtn);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/contacts',
				expect.objectContaining({ method: 'POST' })
			);
		});

		const { goto } = await import('$app/navigation');
		expect(goto).toHaveBeenCalledWith('/contacts');
	});
});

describe('Contact detail page - edit mode', () => {
	beforeEach(async () => {
		const { page } = await import('$app/state');
		(page as any).params = { id: '1' };
		(page as any).url = { pathname: '/contacts/1', searchParams: new URLSearchParams() };
	});

	it('loads contact and shows heading "Upravit kontakt"', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleContact));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Upravit kontakt')).toBeInTheDocument();
		});
	});

	it('form prefilled with contact data', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleContact));

		render(Page);

		await waitFor(() => {
			expect(screen.getByLabelText('Nazev')).toBeInTheDocument();
		});

		const nameInput = screen.getByLabelText('Nazev') as HTMLInputElement;
		expect(nameInput.value).toBe('Test Corp');

		const icoInput = screen.getByLabelText('ICO') as HTMLInputElement;
		expect(icoInput.value).toBe('12345678');

		const dicInput = screen.getByLabelText('DIC') as HTMLInputElement;
		expect(dicInput.value).toBe('CZ12345678');

		const streetInput = screen.getByLabelText('Ulice') as HTMLInputElement;
		expect(streetInput.value).toBe('Hlavni 1');

		const emailInput = screen.getByLabelText('Email') as HTMLInputElement;
		expect(emailInput.value).toBe('test@example.com');
	});

	it('delete button visible', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleContact));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Smazat')).toBeInTheDocument();
		});
	});

	it('ARES lookup fills fields', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleContact));

		render(Page);

		await waitFor(() => {
			expect(screen.getByLabelText('Nazev')).toBeInTheDocument();
		});

		// Mock the ARES lookup fetch
		mockFetch.mockResolvedValueOnce(jsonResponse(aresResult));

		const aresBtn = screen.getByText('ARES');
		await fireEvent.click(aresBtn);

		await waitFor(() => {
			const nameInput = screen.getByLabelText('Nazev') as HTMLInputElement;
			expect(nameInput.value).toBe('ARES Corp s.r.o.');
		});

		const streetInput = screen.getByLabelText('Ulice') as HTMLInputElement;
		expect(streetInput.value).toBe('Dlouha 10');

		const cityInput = screen.getByLabelText('Mesto') as HTMLInputElement;
		expect(cityInput.value).toBe('Brno');

		const zipInput = screen.getByLabelText('PSC') as HTMLInputElement;
		expect(zipInput.value).toBe('60200');
	});

	it('submit for edit calls PUT', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleContact));

		render(Page);

		await waitFor(() => {
			expect(screen.getByLabelText('Nazev')).toBeInTheDocument();
		});

		mockFetch.mockResolvedValueOnce(jsonResponse(sampleContact));

		const submitBtn = screen.getByText('Ulozit');
		await fireEvent.click(submitBtn);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/contacts/1',
				expect.objectContaining({ method: 'PUT' })
			);
		});

		const { goto } = await import('$app/navigation');
		expect(goto).toHaveBeenCalledWith('/contacts');
	});

	it('error on load failure', async () => {
		mockFetch.mockResolvedValueOnce(
			jsonResponse({ error: 'Contact not found' }, 404)
		);

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Contact not found')).toBeInTheDocument();
		});
	});

	it('delete with confirmation calls DELETE', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(sampleContact));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Smazat')).toBeInTheDocument();
		});

		mockFetch.mockResolvedValueOnce(new Response(null, { status: 204, statusText: 'No Content' }));

		await fireEvent.click(screen.getByText('Smazat'));

		expect(confirm).toHaveBeenCalledWith('Opravdu chcete smazat tento kontakt?');

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				'/api/v1/contacts/1',
				expect.objectContaining({ method: 'DELETE' })
			);
		});

		const { goto } = await import('$app/navigation');
		expect(goto).toHaveBeenCalledWith('/contacts');
	});
});
