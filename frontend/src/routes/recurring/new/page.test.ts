import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

import Page from './+page.svelte';

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const sampleContacts = {
	data: [
		{ id: 1, name: 'Test Corp', ico: '12345678' },
		{ id: 2, name: 'Acme', ico: '' }
	],
	total: 2,
	limit: 1000,
	offset: 0
};

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('New recurring invoice page', () => {
	it('renders form heading', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleContacts));

		render(Page);

		expect(screen.getByText('Nova opakujici se faktura')).toBeInTheDocument();
	});

	it('loads contacts on mount', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleContacts));

		render(Page);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalled();
		});

		const contactCall = mockFetch.mock.calls.find(
			(call) => (call[0] as string).includes('/api/v1/contacts')
		);
		expect(contactCall).toBeTruthy();
	});

	it('renders default line item with description and quantity fields', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleContacts));

		render(Page);

		await waitFor(() => {
			expect(screen.getByLabelText('Popis')).toBeInTheDocument();
		});

		expect(screen.getByLabelText('Mnozstvi')).toBeInTheDocument();
	});

	it('adds a line item when Pridat polozku is clicked', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleContacts));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Pridat polozku')).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByText('Pridat polozku'));

		const descriptionInputs = screen.getAllByLabelText(/Popis/);
		expect(descriptionInputs.length).toBe(2);
	});

	it('validation: name required shows error', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleContacts));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Ulozit')).toBeInTheDocument();
		});

		// Remove required attributes to bypass HTML5 validation, test custom validation
		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		await fireEvent.click(screen.getByText('Ulozit'));

		await waitFor(() => {
			expect(screen.getByText('Zadejte nazev')).toBeInTheDocument();
		});
	});

	it('validation: customer required shows error', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleContacts));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Ulozit')).toBeInTheDocument();
		});

		// Set name and desc but leave customer at 0
		const nameInput = document.querySelector('#name') as HTMLInputElement;
		await fireEvent.input(nameInput, { target: { value: 'Test template' } });

		const descInput = document.querySelector('#desc-0') as HTMLInputElement;
		await fireEvent.input(descInput, { target: { value: 'Service' } });

		await fireEvent.click(screen.getByText('Ulozit'));

		await waitFor(() => {
			expect(screen.getByText('Vyberte zakaznika')).toBeInTheDocument();
		});
	});

	it('submit calls POST with correct payload', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleContacts));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Test Corp (12345678)')).toBeInTheDocument();
		});

		// Fill name
		const nameInput = document.querySelector('#name') as HTMLInputElement;
		await fireEvent.input(nameInput, { target: { value: 'Test template' } });

		// Select customer
		const customerSelect = document.querySelector('#customer') as HTMLSelectElement;
		await fireEvent.change(customerSelect, { target: { value: '1' } });

		// Fill line item description
		const descInput = document.querySelector('#desc-0') as HTMLInputElement;
		await fireEvent.input(descInput, { target: { value: 'Some service' } });

		// POST response
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 1, name: 'Test template' }));

		await fireEvent.click(screen.getByText('Ulozit'));

		await waitFor(() => {
			const postCall = mockFetch.mock.calls.find(
				(call) =>
					(call[0] as string).includes('/api/v1/recurring-invoices') &&
					call[1]?.method === 'POST'
			);
			expect(postCall).toBeTruthy();
		});
	});
});
