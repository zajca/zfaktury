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
		{ id: 1, name: 'Supplier Inc', ico: '87654321', type: 'company' },
		{ id: 2, name: 'Freelancer', ico: '', type: 'individual' }
	],
	total: 2,
	limit: 1000,
	offset: 0
};

const sampleCategories = [
	{ key: 'office', label_cs: 'Kancelar' },
	{ key: 'travel', label_cs: 'Cestovne' },
	{ key: 'services', label_cs: 'Sluzby' }
];

beforeEach(() => {
	vi.useFakeTimers();
	vi.setSystemTime(new Date('2026-03-10T12:00:00Z'));
	mockFetch.mockReset();
	// Default: respond to both contacts and categories fetches
	mockFetch.mockImplementation((url: string) => {
		if (typeof url === 'string' && url.includes('/expense-categories')) {
			return Promise.resolve(jsonResponse(sampleCategories));
		}
		if (typeof url === 'string' && url.includes('/contacts')) {
			return Promise.resolve(jsonResponse(sampleContacts));
		}
		return Promise.resolve(jsonResponse({}));
	});
});

afterEach(() => {
	cleanup();
	vi.useRealTimers();
});

describe('Expense Create', () => {
	it('renders form with heading', async () => {
		render(Page);
		expect(screen.getByText('Nový náklad')).toBeInTheDocument();
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

	it('renders vendor select with loaded contacts', async () => {
		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Supplier Inc (87654321)')).toBeInTheDocument();
		});
		expect(screen.getByText('Freelancer')).toBeInTheDocument();
	});

	it('renders vendor placeholder option', () => {
		render(Page);
		expect(screen.getByText('-- Bez dodavatele --')).toBeInTheDocument();
	});

	it('renders description input', () => {
		render(Page);
		const descInput = document.querySelector('#description') as HTMLInputElement;
		expect(descInput).toBeInTheDocument();
	});

	it('renders amount input', () => {
		render(Page);
		const amountInput = document.querySelector('#amount') as HTMLInputElement;
		expect(amountInput).toBeInTheDocument();
	});

	it('renders VAT rate select with options', () => {
		render(Page);
		expect(screen.getByText('21%')).toBeInTheDocument();
		expect(screen.getByText('12%')).toBeInTheDocument();
		expect(screen.getByText('0% (bez DPH)')).toBeInTheDocument();
	});

	it('renders tax deductible checkbox (checked by default)', () => {
		render(Page);
		const checkbox = document.querySelector('#tax_deductible') as HTMLInputElement;
		expect(checkbox).toBeInTheDocument();
		expect(checkbox.checked).toBe(true);
	});

	it('renders business percent input (default 100)', () => {
		render(Page);
		const bpInput = document.querySelector('#business_percent') as HTMLInputElement;
		expect(bpInput).toBeInTheDocument();
		expect(bpInput.value).toBe('100');
	});

	it('renders payment method select', () => {
		render(Page);
		expect(screen.getByText('Bankovní převod')).toBeInTheDocument();
		expect(screen.getByText('Hotovost')).toBeInTheDocument();
		expect(screen.getByText('Karta')).toBeInTheDocument();
	});

	it('renders date input field', () => {
		render(Page);
		const dateInputs = document.querySelectorAll('input[type="date"]');
		expect(dateInputs.length).toBeGreaterThanOrEqual(1);
	});

	it('renders submit and cancel buttons', () => {
		render(Page);
		expect(screen.getByText('Uložit náklad')).toBeInTheDocument();
		expect(screen.getByText('Zrušit')).toBeInTheDocument();
	});

	it('renders back link to expenses', () => {
		render(Page);
		const backLink = screen.getByText(/Zpět na náklady/);
		expect(backLink).toBeInTheDocument();
		expect(backLink.closest('a')?.getAttribute('href')).toBe('/expenses');
	});

	it('shows validation error for empty description', async () => {
		render(Page);
		// Description is empty by default, amount is 0 by default
		// Set amount > 0 to isolate description validation
		const amountInput = document.querySelector('#amount') as HTMLInputElement;
		amountInput.value = '100';
		await fireEvent.input(amountInput);

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			expect(screen.getByText('Popis je povinný')).toBeInTheDocument();
		});
	});

	it('shows validation error for amount <= 0', async () => {
		render(Page);
		// Fill description but leave amount at 0
		const descInput = document.querySelector('#description') as HTMLInputElement;
		descInput.value = 'Test expense';
		await fireEvent.input(descInput);

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			expect(screen.getByText('Částka musí být větší než 0')).toBeInTheDocument();
		});
	});

	it('submits expense with correct halere amounts', async () => {
		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Supplier Inc (87654321)')).toBeInTheDocument();
		});

		// Fill description
		const descInput = document.querySelector('#description') as HTMLInputElement;
		descInput.value = 'Office supplies';
		await fireEvent.input(descInput);

		// Fill amount (1210 CZK with 21% VAT)
		const amountInput = document.querySelector('#amount') as HTMLInputElement;
		amountInput.value = '1210';
		await fireEvent.input(amountInput);

		// Submit
		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			const expenseCall = mockFetch.mock.calls.find(
				(call: any[]) =>
					typeof call[0] === 'string' &&
					call[0].includes('/api/v1/expenses') &&
					!call[0].includes('/expense-categories') &&
					call[1]?.method === 'POST'
			);
			expect(expenseCall).toBeDefined();
			if (expenseCall) {
				const body = JSON.parse(expenseCall[1].body);
				expect(body.description).toBe('Office supplies');
				// amount should be in halere: 1210 * 100 = 121000
				expect(body.amount).toBe(121000);
				// VAT reverse calc: 1210 * 21 / (100 + 21) = 210
				// vat_amount in halere: 210 * 100 = 21000
				expect(body.vat_amount).toBe(21000);
				expect(body.vat_rate_percent).toBe(21);
				expect(body.currency_code).toBe('CZK');
			}
		});
	});

	it('navigates to /expenses after successful submit', async () => {
		const { goto } = await import('$app/navigation');
		render(Page);

		// Fill required fields
		const descInput = document.querySelector('#description') as HTMLInputElement;
		descInput.value = 'Office supplies';
		await fireEvent.input(descInput);

		const amountInput = document.querySelector('#amount') as HTMLInputElement;
		amountInput.value = '100';
		await fireEvent.input(amountInput);

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			expect(goto).toHaveBeenCalledWith('/expenses');
		});
	});

	it('shows error on API failure', async () => {
		mockFetch.mockImplementation((url: string) => {
			if (typeof url === 'string' && url.includes('/expense-categories')) {
				return Promise.resolve(jsonResponse(sampleCategories));
			}
			if (typeof url === 'string' && url.includes('/contacts')) {
				return Promise.resolve(jsonResponse(sampleContacts));
			}
			if (typeof url === 'string' && url.includes('/expenses')) {
				return Promise.resolve(jsonResponse({ error: 'Server error' }, 500));
			}
			return Promise.resolve(jsonResponse({}));
		});

		render(Page);

		// Fill required fields
		const descInput = document.querySelector('#description') as HTMLInputElement;
		descInput.value = 'Test expense';
		await fireEvent.input(descInput);

		const amountInput = document.querySelector('#amount') as HTMLInputElement;
		amountInput.value = '100';
		await fireEvent.input(amountInput);

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			const errorDiv = document.querySelector('.text-red-700');
			expect(errorDiv).toBeInTheDocument();
		});
	});

	it('disables submit button while saving', async () => {
		mockFetch.mockImplementation((url: string) => {
			if (typeof url === 'string' && url.includes('/expense-categories')) {
				return Promise.resolve(jsonResponse(sampleCategories));
			}
			if (typeof url === 'string' && url.includes('/contacts')) {
				return Promise.resolve(jsonResponse(sampleContacts));
			}
			if (typeof url === 'string' && url.includes('/expenses')) {
				return new Promise(() => {}); // never resolves
			}
			return Promise.resolve(jsonResponse({}));
		});

		render(Page);

		const descInput = document.querySelector('#description') as HTMLInputElement;
		descInput.value = 'Test expense';
		await fireEvent.input(descInput);

		const amountInput = document.querySelector('#amount') as HTMLInputElement;
		amountInput.value = '100';
		await fireEvent.input(amountInput);

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			expect(screen.getByText('Ukládám...')).toBeInTheDocument();
		});
	});
});
