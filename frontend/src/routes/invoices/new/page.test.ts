import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import Page from './+page.svelte';

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const sampleContacts = {
	data: [
		{ id: 1, name: 'Test Corp', ico: '12345678', type: 'company' },
		{ id: 2, name: 'Jan Novak', ico: '', type: 'individual' }
	],
	total: 2,
	limit: 1000,
	offset: 0
};

beforeEach(() => {
	vi.useFakeTimers();
	vi.setSystemTime(new Date('2026-03-10T12:00:00Z'));
	mockFetch.mockReset();
	// Default: contacts load (contactsApi.list calls fetch with /api/v1/contacts)
	mockFetch.mockResolvedValue(jsonResponse(sampleContacts));
});

afterEach(() => {
	cleanup();
	vi.useRealTimers();
});

describe('Invoice Create', () => {
	it('renders form with heading', async () => {
		render(Page);
		expect(screen.getByText('Nová faktura')).toBeInTheDocument();
	});

	it('loads contacts on mount', async () => {
		render(Page);
		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				expect.stringContaining('/api/v1/contacts'),
				expect.any(Object)
			);
		});
	});

	it('renders customer select with loaded contacts', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Test Corp (12345678)')).toBeInTheDocument();
		});
		// Contact without ICO should render without parentheses
		expect(screen.getByText('Jan Novak')).toBeInTheDocument();
	});

	it('renders default customer placeholder option', async () => {
		render(Page);
		expect(screen.getByText('-- Vyberte --')).toBeInTheDocument();
	});

	it('renders default line item fields', () => {
		render(Page);
		expect(document.querySelector('#desc-0')).toBeInTheDocument();
		expect(document.querySelector('#qty-0')).toBeInTheDocument();
		expect(document.querySelector('#price-0')).toBeInTheDocument();
		expect(document.querySelector('#unit-0')).toBeInTheDocument();
		expect(document.querySelector('#vat-0')).toBeInTheDocument();
	});

	it('adds line item on button click', async () => {
		render(Page);
		const addBtn = screen.getByText('Přidat položku');
		await fireEvent.click(addBtn);

		expect(document.querySelector('#desc-1')).toBeInTheDocument();
		expect(document.querySelector('#qty-1')).toBeInTheDocument();
		expect(document.querySelector('#price-1')).toBeInTheDocument();
	});

	it('does not show remove button with single item', () => {
		render(Page);
		const removeBtn = screen.queryByLabelText('Odebrat položku');
		expect(removeBtn).not.toBeInTheDocument();
	});

	it('shows remove buttons when multiple items exist', async () => {
		render(Page);
		const addBtn = screen.getByText('Přidat položku');
		await fireEvent.click(addBtn);

		const removeBtns = screen.getAllByLabelText('Odebrat položku');
		expect(removeBtns.length).toBe(2);
	});

	it('removes a line item when remove button is clicked', async () => {
		render(Page);
		const addBtn = screen.getByText('Přidat položku');
		await fireEvent.click(addBtn);

		expect(document.querySelector('#desc-1')).toBeInTheDocument();

		const removeBtns = screen.getAllByLabelText('Odebrat položku');
		await fireEvent.click(removeBtns[1]);

		expect(document.querySelector('#desc-1')).not.toBeInTheDocument();
	});

	it('shows error when no customer selected on submit', async () => {
		render(Page);

		// Wait for contacts to load first
		await waitFor(() => {
			expect(screen.getByText('Test Corp (12345678)')).toBeInTheDocument();
		});

		// Remove HTML5 required attributes to bypass native validation
		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		// Click submit with no customer selected (customer_id defaults to 0)
		await fireEvent.click(screen.getByText('Uložit jako koncept'));

		// Advance fake timers to let Svelte flush updates
		await vi.advanceTimersByTimeAsync(10);

		const errorDiv = document.querySelector('.text-red-700');
		expect(errorDiv).toBeInTheDocument();
		expect(errorDiv?.textContent).toContain('Vyberte zákazníka');
	});

	it('renders date input fields', () => {
		render(Page);
		// DateInput renders <input type="date"> elements
		const dateInputs = document.querySelectorAll('input[type="date"]');
		// issue_date, due_date, delivery_date
		expect(dateInputs.length).toBeGreaterThanOrEqual(3);
	});

	it('renders payment method select', () => {
		render(Page);
		expect(screen.getByText('Bankovní převod')).toBeInTheDocument();
		expect(screen.getByText('Hotovost')).toBeInTheDocument();
		expect(screen.getByText('Karta')).toBeInTheDocument();
	});

	it('renders notes textareas', () => {
		render(Page);
		expect(screen.getByLabelText('Poznámka na faktuře')).toBeInTheDocument();
		expect(screen.getByLabelText('Interní poznámka')).toBeInTheDocument();
	});

	it('renders submit and cancel buttons', () => {
		render(Page);
		expect(screen.getByText('Uložit jako koncept')).toBeInTheDocument();
		expect(screen.getByText('Zrušit')).toBeInTheDocument();
	});

	it('renders back link to invoices', () => {
		render(Page);
		const backLink = screen.getByText(/Zpět na faktury/);
		expect(backLink).toBeInTheDocument();
		expect(backLink.closest('a')?.getAttribute('href')).toBe('/invoices');
	});

	it('submits invoice with correct payload', async () => {
		let callCount = 0;
		mockFetch.mockImplementation((url: string) => {
			callCount++;
			if (typeof url === 'string' && url.includes('/contacts')) {
				return Promise.resolve(jsonResponse(sampleContacts));
			}
			if (typeof url === 'string' && url.includes('/invoices')) {
				return Promise.resolve(jsonResponse({ id: 1 }));
			}
			return Promise.resolve(jsonResponse({}));
		});

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Test Corp (12345678)')).toBeInTheDocument();
		});

		// Select customer
		const customerSelect = document.querySelector('#customer') as HTMLSelectElement;
		await fireEvent.change(customerSelect, { target: { value: '1' } });

		// Fill item description
		const descInput = document.querySelector('#desc-0') as HTMLInputElement;
		descInput.value = 'Web development';
		await fireEvent.input(descInput);

		// Fill item price
		const priceInput = document.querySelector('#price-0') as HTMLInputElement;
		priceInput.value = '1000';
		await fireEvent.input(priceInput);

		// Submit
		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			const invoiceCall = mockFetch.mock.calls.find(
				(call: any[]) =>
					typeof call[0] === 'string' &&
					call[0].includes('/invoices') &&
					call[1]?.method === 'POST'
			);
			expect(invoiceCall).toBeDefined();
			if (invoiceCall) {
				const body = JSON.parse(invoiceCall[1].body);
				expect(body.customer_id).toBe(1);
				expect(body.type).toBe('regular');
				expect(body.status).toBe('draft');
				expect(typeof body.total_amount).toBe('number');
			}
		});
	});

	it('navigates to /invoices after successful submit', async () => {
		const { goto } = await import('$app/navigation');
		mockFetch.mockImplementation((url: string) => {
			if (typeof url === 'string' && url.includes('/contacts')) {
				return Promise.resolve(jsonResponse(sampleContacts));
			}
			if (typeof url === 'string' && url.includes('/invoices')) {
				return Promise.resolve(jsonResponse({ id: 1 }));
			}
			return Promise.resolve(jsonResponse({}));
		});

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Test Corp (12345678)')).toBeInTheDocument();
		});

		const customerSelect = document.querySelector('#customer') as HTMLSelectElement;
		await fireEvent.change(customerSelect, { target: { value: '1' } });

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			expect(goto).toHaveBeenCalledWith('/invoices');
		});
	});

	it('shows error on API failure', async () => {
		mockFetch.mockImplementation((url: string) => {
			if (typeof url === 'string' && url.includes('/contacts')) {
				return Promise.resolve(jsonResponse(sampleContacts));
			}
			if (typeof url === 'string' && url.includes('/invoices')) {
				return Promise.resolve(jsonResponse({ error: 'Server error' }, 500));
			}
			return Promise.resolve(jsonResponse({}));
		});

		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Test Corp (12345678)')).toBeInTheDocument();
		});

		const customerSelect = document.querySelector('#customer') as HTMLSelectElement;
		await fireEvent.change(customerSelect, { target: { value: '1' } });

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			const errorDiv = document.querySelector('.text-red-700');
			expect(errorDiv).toBeInTheDocument();
		});
	});

	it('disables submit button while saving', async () => {
		// Make the invoice creation hang so we can check disabled state
		mockFetch.mockImplementation((url: string) => {
			if (typeof url === 'string' && url.includes('/contacts')) {
				return Promise.resolve(jsonResponse(sampleContacts));
			}
			if (typeof url === 'string' && url.includes('/invoices')) {
				return new Promise(() => {}); // never resolves
			}
			return Promise.resolve(jsonResponse({}));
		});

		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Test Corp (12345678)')).toBeInTheDocument();
		});

		const customerSelect = document.querySelector('#customer') as HTMLSelectElement;
		await fireEvent.change(customerSelect, { target: { value: '1' } });

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			expect(screen.getByText('Ukládám...')).toBeInTheDocument();
		});
	});
});
