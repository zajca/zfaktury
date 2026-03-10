// API client for ZFaktury backend

// --- Domain Types ---

export interface Contact {
	id: number;
	type: 'company' | 'individual';
	name: string;
	ico: string;
	dic: string;
	street: string;
	city: string;
	zip: string;
	country: string;
	email: string;
	phone: string;
	web: string;
	bank_account: string;
	bank_code: string;
	iban: string;
	swift: string;
	payment_terms_days: number;
	tags: string;
	notes: string;
	is_favorite: boolean;
	vat_unreliable: boolean;
	created_at: string;
	updated_at: string;
	deleted_at?: string;
}

export interface InvoiceItem {
	id: number;
	invoice_id: number;
	description: string;
	quantity: number;
	unit: string;
	unit_price: number;
	vat_rate_percent: number;
	vat_amount: number;
	total_amount: number;
	sort_order: number;
}

export interface Invoice {
	id: number;
	sequence_id: number;
	invoice_number: string;
	type: 'regular' | 'proforma' | 'credit_note';
	status: 'draft' | 'sent' | 'paid' | 'overdue' | 'cancelled';
	issue_date: string;
	due_date: string;
	delivery_date: string;
	variable_symbol: string;
	constant_symbol: string;
	customer_id: number;
	customer?: Contact;
	currency_code: string;
	exchange_rate: number;
	payment_method: string;
	bank_account: string;
	bank_code: string;
	iban: string;
	swift: string;
	subtotal_amount: number;
	vat_amount: number;
	total_amount: number;
	paid_amount: number;
	notes: string;
	internal_notes: string;
	sent_at?: string;
	paid_at?: string;
	items: InvoiceItem[];
	created_at: string;
	updated_at: string;
	deleted_at?: string;
}

export interface Expense {
	id: number;
	description: string;
	amount: number;
	vat_amount: number;
	total_amount: number;
	currency_code: string;
	category: string;
	vendor_id?: number;
	vendor?: Contact;
	expense_date: string;
	payment_method: string;
	document_number: string;
	notes: string;
	is_tax_deductible: boolean;
	created_at: string;
	updated_at: string;
	deleted_at?: string;
}

export interface PaginatedResponse<T> {
	data: T[];
	total: number;
	page: number;
	per_page: number;
}

export interface AresResult {
	ico: string;
	dic: string;
	name: string;
	street: string;
	city: string;
	zip: string;
	country: string;
}

// --- API Error ---

export class ApiError extends Error {
	constructor(
		public status: number,
		public statusText: string,
		public body?: unknown
	) {
		super(`API Error ${status}: ${statusText}`);
		this.name = 'ApiError';
	}
}

// --- Fetch Wrapper ---

async function request<T>(path: string, options?: RequestInit): Promise<T> {
	const url = `/api${path}`;
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...(options?.headers as Record<string, string>)
	};

	const response = await fetch(url, {
		...options,
		headers
	});

	if (!response.ok) {
		let body: unknown;
		try {
			body = await response.json();
		} catch {
			// response body is not JSON
		}
		throw new ApiError(response.status, response.statusText, body);
	}

	if (response.status === 204) {
		return undefined as T;
	}

	return response.json();
}

function get<T>(path: string): Promise<T> {
	return request<T>(path, { method: 'GET' });
}

function post<T>(path: string, body: unknown): Promise<T> {
	return request<T>(path, { method: 'POST', body: JSON.stringify(body) });
}

function put<T>(path: string, body: unknown): Promise<T> {
	return request<T>(path, { method: 'PUT', body: JSON.stringify(body) });
}

function del<T>(path: string): Promise<T> {
	return request<T>(path, { method: 'DELETE' });
}

// --- Contacts API ---

export const contactsApi = {
	list(params?: { page?: number; per_page?: number; search?: string }) {
		const query = new URLSearchParams();
		if (params?.page) query.set('page', String(params.page));
		if (params?.per_page) query.set('per_page', String(params.per_page));
		if (params?.search) query.set('search', params.search);
		const qs = query.toString();
		return get<PaginatedResponse<Contact>>(`/contacts${qs ? `?${qs}` : ''}`);
	},

	getById(id: number) {
		return get<Contact>(`/contacts/${id}`);
	},

	create(data: Partial<Contact>) {
		return post<Contact>('/contacts', data);
	},

	update(id: number, data: Partial<Contact>) {
		return put<Contact>(`/contacts/${id}`, data);
	},

	delete(id: number) {
		return del<void>(`/contacts/${id}`);
	},

	lookupAres(ico: string) {
		return get<AresResult>(`/contacts/ares/${ico}`);
	}
};

// --- Invoices API ---

export const invoicesApi = {
	list(params?: { page?: number; per_page?: number; search?: string; status?: string }) {
		const query = new URLSearchParams();
		if (params?.page) query.set('page', String(params.page));
		if (params?.per_page) query.set('per_page', String(params.per_page));
		if (params?.search) query.set('search', params.search);
		if (params?.status) query.set('status', params.status);
		const qs = query.toString();
		return get<PaginatedResponse<Invoice>>(`/invoices${qs ? `?${qs}` : ''}`);
	},

	getById(id: number) {
		return get<Invoice>(`/invoices/${id}`);
	},

	create(data: Partial<Invoice>) {
		return post<Invoice>('/invoices', data);
	},

	update(id: number, data: Partial<Invoice>) {
		return put<Invoice>(`/invoices/${id}`, data);
	},

	delete(id: number) {
		return del<void>(`/invoices/${id}`);
	},

	send(id: number) {
		return post<Invoice>(`/invoices/${id}/send`, {});
	},

	markPaid(id: number) {
		return post<Invoice>(`/invoices/${id}/mark-paid`, {});
	},

	duplicate(id: number) {
		return post<Invoice>(`/invoices/${id}/duplicate`, {});
	}
};

// --- Expenses API ---

export const expensesApi = {
	list(params?: { page?: number; per_page?: number; search?: string }) {
		const query = new URLSearchParams();
		if (params?.page) query.set('page', String(params.page));
		if (params?.per_page) query.set('per_page', String(params.per_page));
		if (params?.search) query.set('search', params.search);
		const qs = query.toString();
		return get<PaginatedResponse<Expense>>(`/expenses${qs ? `?${qs}` : ''}`);
	},

	getById(id: number) {
		return get<Expense>(`/expenses/${id}`);
	},

	create(data: Partial<Expense>) {
		return post<Expense>('/expenses', data);
	},

	update(id: number, data: Partial<Expense>) {
		return put<Expense>(`/expenses/${id}`, data);
	},

	delete(id: number) {
		return del<void>(`/expenses/${id}`);
	}
};
