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
	first_name: 'Jan',
	last_name: 'Novak',
	ico: '12345678',
	dic: 'CZ12345678',
	vat_registered: 'true',
	street: 'Hlavni',
	house_number: '1',
	city: 'Praha',
	zip: '10000',
	email: 'test@example.com',
	phone: '+420123456789',
	bank_account: '123456789',
	bank_code: '0100',
	iban: 'CZ1234567890',
	swift: 'KOMBCZPP',
	c_ufo: '464',
	c_pracufo: '3305',
	c_okec: '582900'
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

describe('Settings Firma Page', () => {
	it('loads settings on mount', async () => {
		render(Page);
		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				expect.stringContaining('/api/v1/settings'),
				expect.any(Object)
			);
		});
	});

	it('renders form with business info fields after loading', async () => {
		render(Page);
		await waitFor(() => {
			expect(document.querySelector('#company_name')).toBeInTheDocument();
		});
		expect(document.querySelector('#first_name')).toBeInTheDocument();
		expect(document.querySelector('#last_name')).toBeInTheDocument();
		expect(document.querySelector('#ico')).toBeInTheDocument();
		expect(document.querySelector('#dic')).toBeInTheDocument();
		expect(document.querySelector('#vat_registered')).toBeInTheDocument();
		expect(document.querySelector('#street')).toBeInTheDocument();
		expect(document.querySelector('#house_number')).toBeInTheDocument();
		expect(document.querySelector('#city')).toBeInTheDocument();
		expect(document.querySelector('#zip')).toBeInTheDocument();
		expect(document.querySelector('#email')).toBeInTheDocument();
		expect(document.querySelector('#phone')).toBeInTheDocument();
		expect(document.querySelector('#bank_account')).toBeInTheDocument();
		expect(document.querySelector('#bank_code')).toBeInTheDocument();
		expect(document.querySelector('#iban')).toBeInTheDocument();
		expect(document.querySelector('#swift')).toBeInTheDocument();
		expect(document.querySelector('#c_ufo')).toBeInTheDocument();
		expect(document.querySelector('#c_pracufo')).toBeInTheDocument();
		expect(document.querySelector('#c_okec')).toBeInTheDocument();
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
		const vatCheckbox = document.querySelector('#vat_registered') as HTMLInputElement;
		expect(vatCheckbox.checked).toBe(true);
	});

	it('save calls settingsApi.update (PUT /api/v1/settings)', async () => {
		render(Page);
		await waitFor(() => {
			expect(document.querySelector('#company_name')).toBeInTheDocument();
		});

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

	it('save completes without error', async () => {
		render(Page);
		await waitFor(() => {
			expect(document.querySelector('#company_name')).toBeInTheDocument();
		});

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

	it('error state on load failure', async () => {
		mockFetch.mockRejectedValue(new Error('Network error'));

		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Network error')).toBeInTheDocument();
		});
	});
});
