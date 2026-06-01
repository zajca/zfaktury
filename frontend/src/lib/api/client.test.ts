import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ApiError, contactsApi, invoicesApi, expensesApi, settingsApi } from './client';
import { currentCompany } from '$lib/stores/currentCompany.svelte';
import { TEST_COMPANY } from '../../test-setup';

// Mock global fetch
const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

const COMPANY_PREFIX = `/api/v1/companies/${TEST_COMPANY.id}`;

function jsonResponse(data: unknown, status = 200, headers: Record<string, string> = {}) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: {
			'Content-Type': 'application/json',
			'X-Company-Id': String(TEST_COMPANY.id),
			...headers
		}
	});
}

function errorResponse(status: number, body?: unknown) {
	return new Response(body ? JSON.stringify(body) : null, {
		status,
		statusText: 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

beforeEach(() => {
	mockFetch.mockReset();
});

// --- ApiError ---

describe('ApiError', () => {
	it('creates error with status and message', () => {
		const err = new ApiError(404, 'Not Found', { error: 'not found' });
		expect(err.status).toBe(404);
		expect(err.statusText).toBe('Not Found');
		expect(err.body).toEqual({ error: 'not found' });
		expect(err.name).toBe('ApiError');
		expect(err.message).toBe('not found'); // extracts from body.error
	});

	it('falls back to generic message when body has no error field', () => {
		const err = new ApiError(500, 'Internal Server Error');
		expect(err.message).toBe('API Error 500: Internal Server Error');
	});

	it('falls back to generic message when body.error is not a string', () => {
		const err = new ApiError(400, 'Bad Request', { error: 123 });
		expect(err.message).toBe('API Error 400: Bad Request');
	});

	it('extracts validation error message from body', () => {
		const err = new ApiError(422, 'Unprocessable Entity', { error: 'due_date is required' });
		expect(err.message).toBe('due_date is required');
	});
});

// --- Contacts API ---

describe('contactsApi', () => {
	it('list sends GET with query params under the per-company prefix', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ data: [], total: 0, limit: 20, offset: 0 }));

		const result = await contactsApi.list({ limit: 10, offset: 5, search: 'test' });

		expect(mockFetch).toHaveBeenCalledOnce();
		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toContain(`${COMPANY_PREFIX}/contacts`);
		expect(url).toContain('limit=10');
		expect(url).toContain('offset=5');
		expect(url).toContain('search=test');
		expect(opts.method).toBe('GET');
		expect(result.data).toEqual([]);
	});

	it('list without params sends no query string', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ data: [], total: 0, limit: 20, offset: 0 }));

		await contactsApi.list();

		const [url] = mockFetch.mock.calls[0];
		expect(url).toBe(`${COMPANY_PREFIX}/contacts`);
	});

	it('getById sends GET with id', async () => {
		const contact = { id: 1, name: 'Test' };
		mockFetch.mockResolvedValueOnce(jsonResponse(contact));

		const result = await contactsApi.getById(1);

		const [url] = mockFetch.mock.calls[0];
		expect(url).toBe(`${COMPANY_PREFIX}/contacts/1`);
		expect(result).toEqual(contact);
	});

	it('create returns WriteResult with data, submittedFor, respondedFor', async () => {
		const data = { type: 'company' as const, name: 'New Corp' };
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 1, ...data }));

		const result = await contactsApi.create(data);

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe(`${COMPANY_PREFIX}/contacts`);
		expect(opts.method).toBe('POST');
		expect(JSON.parse(opts.body)).toEqual(data);
		expect(result.submittedFor).toBe(TEST_COMPANY.id);
		expect(result.respondedFor).toBe(TEST_COMPANY.id);
		expect(result.data).toMatchObject(data);
	});

	it('create surfaces X-Company-Id mismatch via respondedFor', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 9 }, 200, { 'X-Company-Id': '7' }));

		const result = await contactsApi.create({ name: 'X' });
		expect(result.submittedFor).toBe(TEST_COMPANY.id);
		expect(result.respondedFor).toBe(7);
	});

	it('update sends PUT with id and body, returns WriteResult', async () => {
		const data = { name: 'Updated' };
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 1, name: 'Updated' }));

		const result = await contactsApi.update(1, data);

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe(`${COMPANY_PREFIX}/contacts/1`);
		expect(opts.method).toBe('PUT');
		expect(result.data).toEqual({ id: 1, name: 'Updated' });
	});

	it('delete sends DELETE, returns WriteResult<void>', async () => {
		mockFetch.mockResolvedValueOnce(
			new Response(null, {
				status: 204,
				statusText: 'No Content',
				headers: { 'X-Company-Id': String(TEST_COMPANY.id) }
			})
		);

		const result = await contactsApi.delete(1);

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe(`${COMPANY_PREFIX}/contacts/1`);
		expect(opts.method).toBe('DELETE');
		expect(result.data).toBeUndefined();
		expect(result.submittedFor).toBe(TEST_COMPANY.id);
	});

	it('lookupAres sends GET with ICO under the per-company prefix', async () => {
		const ares = { ico: '12345678', name: 'ARES Corp' };
		mockFetch.mockResolvedValueOnce(jsonResponse(ares));

		const result = await contactsApi.lookupAres('12345678');

		const [url] = mockFetch.mock.calls[0];
		expect(url).toBe(`${COMPANY_PREFIX}/contacts/ares/12345678`);
		expect(result).toEqual(ares);
	});

	it('throws ApiError on non-ok response', async () => {
		mockFetch.mockResolvedValueOnce(errorResponse(422, { error: 'validation failed' }));

		await expect(contactsApi.create({})).rejects.toThrow(ApiError);
		try {
			mockFetch.mockResolvedValueOnce(errorResponse(422, { error: 'validation failed' }));
			await contactsApi.create({});
		} catch (e) {
			expect(e).toBeInstanceOf(ApiError);
			expect((e as ApiError).status).toBe(422);
		}
	});
});

// --- Invoices API ---

describe('invoicesApi', () => {
	it('list sends GET with query params under the per-company prefix', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ data: [], total: 0, limit: 20, offset: 0 }));

		await invoicesApi.list({ limit: 10, status: 'draft' });

		const [url] = mockFetch.mock.calls[0];
		expect(url).toContain(`${COMPANY_PREFIX}/invoices`);
		expect(url).toContain('limit=10');
		expect(url).toContain('status=draft');
	});

	it('getById sends GET', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 1, invoice_number: 'FV001' }));

		const result = await invoicesApi.getById(1);

		expect(mockFetch.mock.calls[0][0]).toBe(`${COMPANY_PREFIX}/invoices/1`);
		expect(result.invoice_number).toBe('FV001');
	});

	it('send sends POST to /send', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 1, status: 'sent' }));

		const result = await invoicesApi.send(1);

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe(`${COMPANY_PREFIX}/invoices/1/send`);
		expect(opts.method).toBe('POST');
		expect(result.data.status).toBe('sent');
	});

	it('markPaid sends POST to /mark-paid with amount', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 1, status: 'paid' }));

		await invoicesApi.markPaid(1, 10000, '2026-03-10');

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe(`${COMPANY_PREFIX}/invoices/1/mark-paid`);
		expect(opts.method).toBe('POST');
		const body = JSON.parse(opts.body);
		expect(body.amount).toBe(10000);
		expect(body.paid_at).toBe('2026-03-10');
	});

	it('duplicate sends POST to /duplicate', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 2, status: 'draft' }));

		const result = await invoicesApi.duplicate(1);

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe(`${COMPANY_PREFIX}/invoices/1/duplicate`);
		expect(opts.method).toBe('POST');
		expect(result.data.id).toBe(2);
	});
});

// --- Expenses API ---

describe('expensesApi', () => {
	it('list sends GET', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ data: [], total: 0, limit: 20, offset: 0 }));

		await expensesApi.list({ search: 'office' });

		const [url] = mockFetch.mock.calls[0];
		expect(url).toContain(`${COMPANY_PREFIX}/expenses`);
		expect(url).toContain('search=office');
	});

	it('create sends POST', async () => {
		const data = { description: 'Office supplies', amount: 50000 };
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 1, ...data }));

		const result = await expensesApi.create(data);

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe(`${COMPANY_PREFIX}/expenses`);
		expect(opts.method).toBe('POST');
		expect(JSON.parse(opts.body)).toEqual(data);
		expect(result.data.id).toBe(1);
	});

	it('delete sends DELETE', async () => {
		mockFetch.mockResolvedValueOnce(
			new Response(null, {
				status: 204,
				statusText: 'No Content',
				headers: { 'X-Company-Id': String(TEST_COMPANY.id) }
			})
		);

		await expensesApi.delete(5);

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe(`${COMPANY_PREFIX}/expenses/5`);
		expect(opts.method).toBe('DELETE');
	});
});

// --- Settings API ---

describe('settingsApi', () => {
	it('getAll sends GET', async () => {
		const settings = { company_name: 'Test s.r.o.', ico: '12345678' };
		mockFetch.mockResolvedValueOnce(jsonResponse(settings));

		const result = await settingsApi.getAll();

		expect(mockFetch.mock.calls[0][0]).toBe(`${COMPANY_PREFIX}/settings`);
		expect(result).toEqual(settings);
	});

	it('update sends PUT with settings', async () => {
		const settings = { company_name: 'Updated', ico: '87654321' };
		mockFetch.mockResolvedValueOnce(jsonResponse(settings));

		const result = await settingsApi.update(settings);

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe(`${COMPANY_PREFIX}/settings`);
		expect(opts.method).toBe('PUT');
		expect(JSON.parse(opts.body)).toEqual(settings);
		expect(result.data).toEqual(settings);
	});
});

// --- Error handling ---

describe('API error handling', () => {
	it('throws ApiError with parsed body on 4xx', async () => {
		mockFetch.mockResolvedValueOnce(errorResponse(400, { error: 'bad request' }));

		try {
			await contactsApi.getById(999);
			expect.unreachable('should have thrown');
		} catch (e) {
			expect(e).toBeInstanceOf(ApiError);
			const apiErr = e as ApiError;
			expect(apiErr.status).toBe(400);
			expect(apiErr.body).toEqual({ error: 'bad request' });
		}
	});

	it('throws ApiError on 500', async () => {
		mockFetch.mockResolvedValueOnce(errorResponse(500, { error: 'internal error' }));

		await expect(contactsApi.list()).rejects.toThrow(ApiError);
	});

	it('handles 204 No Content without parsing body', async () => {
		mockFetch.mockResolvedValueOnce(
			new Response(null, {
				status: 204,
				statusText: 'No Content',
				headers: { 'X-Company-Id': String(TEST_COMPANY.id) }
			})
		);

		const result = await contactsApi.delete(1);
		expect(result.data).toBeUndefined();
	});
});

// --- NoCompanyError ---

describe('No active company guard', () => {
	it('reads throw NoCompanyError when no company selected', async () => {
		currentCompany.reset();
		await expect(contactsApi.list()).rejects.toThrow('no active company');
	});

	it('writes throw NoCompanyError when no company selected', async () => {
		currentCompany.reset();
		await expect(contactsApi.create({ name: 'X' })).rejects.toThrow('no active company');
	});
});
