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

const sampleSettings = {
	company_name: 'Test OSVC',
	ico: '12345678',
	dic: 'CZ12345678',
	vat_registered: 'true',
	street: 'Hlavni 1',
	city: 'Praha',
	zip: '10000',
	email: 'test@example.com',
	phone: '+420123456789',
	bank_account: '123456789',
	bank_code: '0100',
	iban: 'CZ1234567890',
	swift: 'KOMBCZPP'
};

beforeEach(() => {
	vi.useFakeTimers();
	mockFetch.mockReset();
	mockFetch.mockResolvedValue(jsonResponse(sampleSettings));
});

afterEach(() => {
	cleanup();
	vi.useRealTimers();
});

describe('Settings Page', () => {
	it('loads settings on mount', async () => {
		render(Page);
		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				expect.stringContaining('/api/v1/settings'),
				expect.any(Object)
			);
		});
	});

	it('renders form with fields after loading', async () => {
		render(Page);
		await waitFor(() => {
			expect(document.querySelector('#company_name')).toBeInTheDocument();
		});
		expect(document.querySelector('#ico')).toBeInTheDocument();
		expect(document.querySelector('#dic')).toBeInTheDocument();
		expect(document.querySelector('#vat_registered')).toBeInTheDocument();
		expect(document.querySelector('#street')).toBeInTheDocument();
		expect(document.querySelector('#city')).toBeInTheDocument();
		expect(document.querySelector('#zip')).toBeInTheDocument();
		expect(document.querySelector('#email')).toBeInTheDocument();
		expect(document.querySelector('#phone')).toBeInTheDocument();
		expect(document.querySelector('#bank_account')).toBeInTheDocument();
		expect(document.querySelector('#bank_code')).toBeInTheDocument();
		expect(document.querySelector('#iban')).toBeInTheDocument();
		expect(document.querySelector('#swift')).toBeInTheDocument();
	});

	it('shows settings values in inputs', async () => {
		render(Page);
		await waitFor(() => {
			const nameInput = document.querySelector('#company_name') as HTMLInputElement;
			expect(nameInput).toBeInTheDocument();
			expect(nameInput.value).toBe('Test OSVC');
		});
		const icoInput = document.querySelector('#ico') as HTMLInputElement;
		expect(icoInput.value).toBe('12345678');
		const dicInput = document.querySelector('#dic') as HTMLInputElement;
		expect(dicInput.value).toBe('CZ12345678');
		const vatCheckbox = document.querySelector('#vat_registered') as HTMLInputElement;
		expect(vatCheckbox.checked).toBe(true);
		const streetInput = document.querySelector('#street') as HTMLInputElement;
		expect(streetInput.value).toBe('Hlavni 1');
		const emailInput = document.querySelector('#email') as HTMLInputElement;
		expect(emailInput.value).toBe('test@example.com');
	});

	it('save calls settingsApi.update (PUT /api/v1/settings)', async () => {
		render(Page);
		await waitFor(() => {
			expect(document.querySelector('#company_name')).toBeInTheDocument();
		});

		// Reset to track only the save call
		mockFetch.mockResolvedValue(jsonResponse(sampleSettings));

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			const putCall = mockFetch.mock.calls.find(
				(call: any[]) =>
					typeof call[0] === 'string' &&
					call[0].includes('/api/v1/settings') &&
					call[1]?.method === 'PUT'
			);
			expect(putCall).toBeDefined();
		});
	});

	it('success message appears after save', async () => {
		render(Page);
		await waitFor(() => {
			expect(document.querySelector('#company_name')).toBeInTheDocument();
		});

		mockFetch.mockResolvedValue(jsonResponse(sampleSettings));

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			expect(screen.getByText('Nastavení bylo uloženo.')).toBeInTheDocument();
		});
	});

	it('success message auto-dismisses after 3s', async () => {
		render(Page);
		await waitFor(() => {
			expect(document.querySelector('#company_name')).toBeInTheDocument();
		});

		mockFetch.mockResolvedValue(jsonResponse(sampleSettings));

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			expect(screen.getByText('Nastavení bylo uloženo.')).toBeInTheDocument();
		});

		// Advance timers by 3 seconds
		await vi.advanceTimersByTimeAsync(3000);

		await waitFor(() => {
			expect(screen.queryByText('Nastavení bylo uloženo.')).not.toBeInTheDocument();
		});
	});

	it('error state on load failure', async () => {
		mockFetch.mockRejectedValue(new Error('Network error'));

		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Network error')).toBeInTheDocument();
		});
	});

	it('error state on save failure', async () => {
		render(Page);
		await waitFor(() => {
			expect(document.querySelector('#company_name')).toBeInTheDocument();
		});

		mockFetch.mockResolvedValue(jsonResponse({ error: 'Save failed' }, 500));

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			const errorDiv = document.querySelector('.text-red-700');
			expect(errorDiv).toBeInTheDocument();
		});
	});

	it('shows links to sequences and categories pages', async () => {
		render(Page);

		const sequencesLink = screen.getByText('Číselné řady faktur');
		expect(sequencesLink.closest('a')?.getAttribute('href')).toBe('/settings/sequences');

		const categoriesLink = screen.getByText('Kategorie nákladů');
		expect(categoriesLink.closest('a')?.getAttribute('href')).toBe('/settings/categories');
	});
});
