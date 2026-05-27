import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import Page from './+page.svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: {
			'Content-Type': 'application/json',
			'X-Company-Id': '1'
		}
	});
}

const sampleCompany = {
	id: 1,
	name: 'Test OSVC',
	legal_name: 'Test OSVC s.r.o.',
	first_name: 'Jan',
	last_name: 'Novak',
	ico: '12345678',
	dic: 'CZ12345678',
	vat_registered: true,
	street: 'Hlavni',
	house_number: '1',
	city: 'Praha',
	zip: '10000',
	email: 'test@example.com',
	phone: '+420123456789',
	bank_account: '123456789',
	bank_code: '0100',
	iban: 'CZ1234567890',
	swift: 'KOMBCZPP'
};

const sampleSettings = {
	c_ufo: '464',
	c_pracufo: '3305',
	c_okec: '582900',
	financni_urad_code: '0451',
	cssz_code: 'prehledosvc',
	health_insurance_code: '111'
};

function mockResponses() {
	mockFetch.mockImplementation((url: string, init?: RequestInit) => {
		const method = init?.method ?? 'GET';
		if (typeof url === 'string' && url.includes('/invoice-sequences')) {
			// Some shared bootstrap may probe this; not relevant here.
			return Promise.resolve(jsonResponse([]));
		}
		if (method === 'GET' && url.endsWith('/api/v1/companies/1')) {
			return Promise.resolve(jsonResponse(sampleCompany));
		}
		if (method === 'GET' && url.endsWith('/api/v1/companies/1/settings')) {
			return Promise.resolve(jsonResponse(sampleSettings));
		}
		if (method === 'PUT') {
			return Promise.resolve(jsonResponse(method === 'PUT' ? sampleCompany : sampleSettings));
		}
		if (url.endsWith('/api/v1/companies')) {
			return Promise.resolve(jsonResponse([sampleCompany]));
		}
		return Promise.resolve(jsonResponse({}));
	});
}

beforeEach(async () => {
	mockFetch.mockReset();
	mockResponses();
	const { currentCompany } = await import('$lib/stores/currentCompany.svelte');
	currentCompany.setCompanies([sampleCompany as never]);
	currentCompany.select(1);
});

afterEach(async () => {
	cleanup();
	const { currentCompany } = await import('$lib/stores/currentCompany.svelte');
	currentCompany.reset();
});

describe('Settings Firma Page', () => {
	it('loads the active company and tax-code settings on mount', async () => {
		render(Page);
		await waitFor(() => {
			const companyCall = mockFetch.mock.calls.find(
				(call: unknown[]) =>
					typeof call[0] === 'string' && call[0].endsWith('/api/v1/companies/1')
			);
			const settingsCall = mockFetch.mock.calls.find(
				(call: unknown[]) =>
					typeof call[0] === 'string' && call[0].endsWith('/api/v1/companies/1/settings')
			);
			expect(companyCall).toBeDefined();
			expect(settingsCall).toBeDefined();
		});
	});

	it('renders all sections of the form after loading', async () => {
		render(Page);
		await waitFor(() => {
			expect(document.querySelector('#company_ico')).toBeInTheDocument();
		});
		// CompanyEditForm fields (prefixed company_*).
		expect(document.querySelector('#company_name')).toBeInTheDocument();
		expect(document.querySelector('#company_legal_name')).toBeInTheDocument();
		expect(document.querySelector('#company_dic')).toBeInTheDocument();
		expect(document.querySelector('#company_vat_registered')).toBeInTheDocument();
		expect(document.querySelector('#company_street')).toBeInTheDocument();
		expect(document.querySelector('#company_city')).toBeInTheDocument();
		expect(document.querySelector('#company_iban')).toBeInTheDocument();
		// Tax-office code fields (settings-keyed, unprefixed).
		expect(document.querySelector('#c_ufo')).toBeInTheDocument();
		expect(document.querySelector('#c_pracufo')).toBeInTheDocument();
		expect(document.querySelector('#c_okec')).toBeInTheDocument();
		expect(document.querySelector('#financni_urad_code')).toBeInTheDocument();
		expect(document.querySelector('#cssz_code')).toBeInTheDocument();
		expect(document.querySelector('#health_insurance_code')).toBeInTheDocument();
	});

	it('shows loaded company values + tax codes in inputs', async () => {
		render(Page);
		await waitFor(() => {
			const ico = document.querySelector('#company_ico') as HTMLInputElement;
			expect(ico).toBeInTheDocument();
			expect(ico.value).toBe('12345678');
		});
		const name = document.querySelector('#company_name') as HTMLInputElement;
		expect(name.value).toBe('Test OSVC');
		const dic = document.querySelector('#company_dic') as HTMLInputElement;
		expect(dic.value).toBe('CZ12345678');
		const ufo = document.querySelector('#c_ufo') as HTMLInputElement;
		expect(ufo.value).toBe('464');
	});

	it('save issues both a company PUT and a settings PUT', async () => {
		render(Page);
		await waitFor(() => {
			expect(document.querySelector('#company_ico')).toBeInTheDocument();
		});

		const form = document.querySelector('form')!;
		await fireEvent.submit(form);

		await waitFor(() => {
			const companyPut = mockFetch.mock.calls.find(
				(call: unknown[]) => {
					const u = call[0] as string;
					const init = call[1] as RequestInit | undefined;
					return (
						typeof u === 'string' &&
						u.endsWith('/api/v1/companies/1') &&
						init?.method === 'PUT'
					);
				}
			);
			const settingsPut = mockFetch.mock.calls.find(
				(call: unknown[]) => {
					const u = call[0] as string;
					const init = call[1] as RequestInit | undefined;
					return (
						typeof u === 'string' &&
						u.endsWith('/api/v1/companies/1/settings') &&
						init?.method === 'PUT'
					);
				}
			);
			expect(companyPut).toBeDefined();
			expect(settingsPut).toBeDefined();
		});
	});

	it('shows an error when company load fails', async () => {
		mockFetch.mockReset();
		mockFetch.mockRejectedValue(new Error('Network error'));

		render(Page);
		await waitFor(() => {
			expect(screen.getByText('Network error')).toBeInTheDocument();
		});
	});
});
