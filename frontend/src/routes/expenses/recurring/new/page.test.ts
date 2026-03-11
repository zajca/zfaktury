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
		{ id: 1, name: 'Provider Corp', ico: '12345678' },
		{ id: 2, name: 'Vendor Ltd', ico: '' }
	],
	total: 2,
	limit: 1000,
	offset: 0
};

const sampleCategories = [
	{ key: 'services', label_cs: 'Sluzby' },
	{ key: 'office', label_cs: 'Kancelar' }
];

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('New recurring expense page', () => {
	function setupDefaultMocks() {
		mockFetch.mockImplementation((url: string) => {
			if (url.includes('/api/v1/contacts')) {
				return Promise.resolve(jsonResponse(sampleContacts));
			}
			if (url.includes('/api/v1/expense-categories')) {
				return Promise.resolve(jsonResponse(sampleCategories));
			}
			return Promise.resolve(jsonResponse({}));
		});
	}

	it('renders form heading', async () => {
		setupDefaultMocks();

		render(Page);

		expect(screen.getByText('Nový opakovaný náklad')).toBeInTheDocument();
	});

	it('loads contacts on mount', async () => {
		setupDefaultMocks();

		render(Page);

		await waitFor(() => {
			const contactCall = mockFetch.mock.calls.find((call) =>
				(call[0] as string).includes('/api/v1/contacts')
			);
			expect(contactCall).toBeTruthy();
		});
	});

	it('validation: name required shows error', async () => {
		setupDefaultMocks();

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Uložit')).toBeInTheDocument();
		});

		// Remove HTML5 required to test custom validation in handleSubmit
		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		await fireEvent.click(screen.getByText('Uložit'));

		await waitFor(() => {
			expect(screen.getByText('Název je povinný')).toBeInTheDocument();
		});
	});

	it('validation: description required shows error', async () => {
		setupDefaultMocks();

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Uložit')).toBeInTheDocument();
		});

		// Remove HTML5 required to test custom validation
		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		// Fill name but leave description empty
		const nameInput = document.querySelector('#name') as HTMLInputElement;
		await fireEvent.input(nameInput, { target: { value: 'Test expense' } });

		await fireEvent.click(screen.getByText('Uložit'));

		await waitFor(() => {
			expect(screen.getByText('Popis je povinný')).toBeInTheDocument();
		});
	});

	it('validation: amount must be greater than 0', async () => {
		setupDefaultMocks();

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Uložit')).toBeInTheDocument();
		});

		const nameInput = screen.getByLabelText('Název *');
		await fireEvent.input(nameInput, { target: { value: 'Test expense' } });

		const descInput = screen.getByLabelText('Popis nákladu *');
		await fireEvent.input(descInput, { target: { value: 'Some description' } });

		// Amount defaults to 0
		await fireEvent.click(screen.getByText('Uložit'));

		await waitFor(() => {
			expect(screen.getByText('Částka musí být větší než 0')).toBeInTheDocument();
		});
	});

	it('submit calls create with halere amounts', async () => {
		setupDefaultMocks();

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Uložit')).toBeInTheDocument();
		});

		const nameInput = screen.getByLabelText('Název *');
		await fireEvent.input(nameInput, { target: { value: 'Test expense' } });

		const descInput = screen.getByLabelText('Popis nákladu *');
		await fireEvent.input(descInput, { target: { value: 'Monthly hosting' } });

		const amountInput = screen.getByLabelText(/Částka s DPH/);
		await fireEvent.input(amountInput, { target: { value: '500' } });

		// create response
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 1 }));

		await fireEvent.click(screen.getByText('Uložit'));

		await waitFor(() => {
			const postCall = mockFetch.mock.calls.find(
				(call) =>
					(call[0] as string).includes('/api/v1/recurring-expenses') &&
					call[1]?.method === 'POST' &&
					!(call[0] as string).includes('generate')
			);
			expect(postCall).toBeTruthy();
			if (postCall) {
				const body = JSON.parse(postCall[1].body as string);
				expect(body.amount).toBe(50000); // 500 CZK = 50000 halere
			}
		});
	});

	it('shows error state on create failure', async () => {
		setupDefaultMocks();

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Uložit')).toBeInTheDocument();
		});

		const nameInput = screen.getByLabelText('Název *');
		await fireEvent.input(nameInput, { target: { value: 'Test expense' } });

		const descInput = screen.getByLabelText('Popis nákladu *');
		await fireEvent.input(descInput, { target: { value: 'Monthly hosting' } });

		const amountInput = screen.getByLabelText(/Částka s DPH/);
		await fireEvent.input(amountInput, { target: { value: '500' } });

		// Make the create call fail
		mockFetch.mockRejectedValueOnce(new Error('Server error'));

		await fireEvent.click(screen.getByText('Uložit'));

		await waitFor(() => {
			expect(screen.getByText('Server error')).toBeInTheDocument();
		});
	});
});
