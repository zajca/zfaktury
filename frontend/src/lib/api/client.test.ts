import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ApiError, contactsApi, invoicesApi, expensesApi, settingsApi } from './client';

// Mock global fetch
const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
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
		expect(err.message).toBe('API Error 404: Not Found');
	});
});

// --- Contacts API ---

describe('contactsApi', () => {
	it('list sends GET with query params', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ data: [], total: 0, limit: 20, offset: 0 }));

		const result = await contactsApi.list({ limit: 10, offset: 5, search: 'test' });

		expect(mockFetch).toHaveBeenCalledOnce();
		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toContain('/api/v1/contacts');
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
		expect(url).toBe('/api/v1/contacts');
	});

	it('getById sends GET with id', async () => {
		const contact = { id: 1, name: 'Test' };
		mockFetch.mockResolvedValueOnce(jsonResponse(contact));

		const result = await contactsApi.getById(1);

		const [url] = mockFetch.mock.calls[0];
		expect(url).toBe('/api/v1/contacts/1');
		expect(result).toEqual(contact);
	});

	it('create sends POST with body', async () => {
		const data = { type: 'company' as const, name: 'New Corp' };
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 1, ...data }));

		await contactsApi.create(data);

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe('/api/v1/contacts');
		expect(opts.method).toBe('POST');
		expect(JSON.parse(opts.body)).toEqual(data);
	});

	it('update sends PUT with id and body', async () => {
		const data = { name: 'Updated' };
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 1, name: 'Updated' }));

		await contactsApi.update(1, data);

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe('/api/v1/contacts/1');
		expect(opts.method).toBe('PUT');
	});

	it('delete sends DELETE', async () => {
		mockFetch.mockResolvedValueOnce(new Response(null, { status: 204, statusText: 'No Content' }));

		await contactsApi.delete(1);

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe('/api/v1/contacts/1');
		expect(opts.method).toBe('DELETE');
	});

	it('lookupAres sends GET with ICO', async () => {
		const ares = { ico: '12345678', name: 'ARES Corp' };
		mockFetch.mockResolvedValueOnce(jsonResponse(ares));

		const result = await contactsApi.lookupAres('12345678');

		const [url] = mockFetch.mock.calls[0];
		expect(url).toBe('/api/v1/contacts/ares/12345678');
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
	it('list sends GET with query params', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ data: [], total: 0, limit: 20, offset: 0 }));

		await invoicesApi.list({ limit: 10, status: 'draft' });

		const [url] = mockFetch.mock.calls[0];
		expect(url).toContain('/api/v1/invoices');
		expect(url).toContain('limit=10');
		expect(url).toContain('status=draft');
	});

	it('getById sends GET', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 1, invoice_number: 'FV001' }));

		const result = await invoicesApi.getById(1);

		expect(mockFetch.mock.calls[0][0]).toBe('/api/v1/invoices/1');
		expect(result.invoice_number).toBe('FV001');
	});

	it('send sends POST to /send', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 1, status: 'sent' }));

		await invoicesApi.send(1);

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe('/api/v1/invoices/1/send');
		expect(opts.method).toBe('POST');
	});

	it('markPaid sends POST to /mark-paid with amount', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 1, status: 'paid' }));

		await invoicesApi.markPaid(1, 10000, '2026-03-10');

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe('/api/v1/invoices/1/mark-paid');
		expect(opts.method).toBe('POST');
		const body = JSON.parse(opts.body);
		expect(body.amount).toBe(10000);
		expect(body.paid_at).toBe('2026-03-10');
	});

	it('duplicate sends POST to /duplicate', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 2, status: 'draft' }));

		await invoicesApi.duplicate(1);

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe('/api/v1/invoices/1/duplicate');
		expect(opts.method).toBe('POST');
	});
});

// --- Expenses API ---

describe('expensesApi', () => {
	it('list sends GET', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ data: [], total: 0, limit: 20, offset: 0 }));

		await expensesApi.list({ search: 'office' });

		const [url] = mockFetch.mock.calls[0];
		expect(url).toContain('/api/v1/expenses');
		expect(url).toContain('search=office');
	});

	it('create sends POST', async () => {
		const data = { description: 'Office supplies', amount: 50000 };
		mockFetch.mockResolvedValueOnce(jsonResponse({ id: 1, ...data }));

		await expensesApi.create(data);

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe('/api/v1/expenses');
		expect(opts.method).toBe('POST');
		expect(JSON.parse(opts.body)).toEqual(data);
	});

	it('delete sends DELETE', async () => {
		mockFetch.mockResolvedValueOnce(new Response(null, { status: 204, statusText: 'No Content' }));

		await expensesApi.delete(5);

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe('/api/v1/expenses/5');
		expect(opts.method).toBe('DELETE');
	});
});

// --- Settings API ---

describe('settingsApi', () => {
	it('getAll sends GET', async () => {
		const settings = { company_name: 'Test s.r.o.', ico: '12345678' };
		mockFetch.mockResolvedValueOnce(jsonResponse(settings));

		const result = await settingsApi.getAll();

		expect(mockFetch.mock.calls[0][0]).toBe('/api/v1/settings');
		expect(result).toEqual(settings);
	});

	it('update sends PUT with settings', async () => {
		const settings = { company_name: 'Updated', ico: '87654321' };
		mockFetch.mockResolvedValueOnce(jsonResponse(settings));

		await settingsApi.update(settings);

		const [url, opts] = mockFetch.mock.calls[0];
		expect(url).toBe('/api/v1/settings');
		expect(opts.method).toBe('PUT');
		expect(JSON.parse(opts.body)).toEqual(settings);
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
		mockFetch.mockResolvedValueOnce(new Response(null, { status: 204, statusText: 'No Content' }));

		const result = await contactsApi.delete(1);
		expect(result).toBeUndefined();
	});
});
